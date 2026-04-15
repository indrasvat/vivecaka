#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.14"
# dependencies = [
#   "iterm2",
#   "pyobjc",
#   "pyobjc-framework-Quartz",
# ]
# ///

"""
Baseline visual verification for incremental-review planning in vivecaka.

Tests:
    1. Launch vivecaka against a repository with an open PR.
    2. Open a known PR detail view and capture the default Description tab.
    3. Switch to the Files tab and capture the current pre-feature file list layout.
    4. Open the diff viewer and capture unified and side-by-side states.

Verification Strategy:
    - Create an isolated iTerm2 window and drive the TUI only with keyboard input.
    - Wait for concrete screen markers before taking screenshots.
    - Capture baseline screenshots for the surfaces that incremental review mode will modify.
    - Fail loudly with a screen dump if expected markers do not appear.

Screenshots:
    - pr_detail_description.png: baseline detail view before incremental-review chrome exists.
    - pr_detail_files.png: baseline files tab layout and counts.
    - pr_diff_unified.png: current diff viewer with file tree and unified mode.
    - pr_diff_split.png: current diff viewer in side-by-side mode.

Screenshot Inspection Checklist:
    - Tab row is intact with Description, Checks, Files, Comments.
    - Files tab shows file rows without layout bleed or truncation artifacts.
    - Diff view preserves two-pane layout with file tree on the left.
    - Split mode remains aligned with no border break or status-bar overlap.

Key Bindings:
    - Enter: Open the selected PR.
    - 3: Switch to the Files tab.
    - d: Open the diff viewer.
    - t: Toggle unified / side-by-side mode.

Usage:
    uv run .claude/automations/visual_test_incremental_review_baseline.py
"""

import asyncio
import os
import subprocess
import sys
import time
from datetime import datetime
from pathlib import Path

import iterm2

try:
    import Quartz
except ImportError as exc:  # pragma: no cover - runtime requirement
    raise SystemExit(f"Quartz is required for screenshots: {exc}") from exc


APPDIR = Path("/Users/robin.sharma/code/github.com/indrasvat/vivecaka")
TARGET_REPO_DIR = Path(
    os.environ.get("VIVECAKA_VISUAL_TARGET_DIR", "/Users/robin.sharma/code/github.com/indrasvat/dootsabha")
)
SCREENSHOT_DIR = APPDIR / "screenshots" / "incremental-review-baseline"
WINDOW_PREFIX = "vivecaka-incremental-baseline-"
PR_TITLE = os.environ.get("VIVECAKA_VISUAL_PR_TITLE", "Update default model to claude-opus-4-6")
PR_NUMBER = os.environ.get("VIVECAKA_VISUAL_PR_NUMBER", "11")
FILE_MARKER = os.environ.get("VIVECAKA_VISUAL_FILE_MARKER", "README.md")
TIMEOUT_SECONDS = 20.0


def print_header(title: str) -> None:
    print(f"\n{'=' * 72}")
    print(title)
    print("=" * 72)


async def cleanup_stale_windows(connection, prefix: str = WINDOW_PREFIX) -> None:
    app = await iterm2.async_get_app(connection)
    for window in app.terminal_windows:
        for tab in window.tabs:
            for session in tab.sessions:
                if session.name and session.name.startswith(prefix):
                    try:
                        await session.async_send_text("\x03")
                        await asyncio.sleep(0.1)
                        await session.async_send_text("exit\n")
                        await asyncio.sleep(0.1)
                        await session.async_close()
                    except Exception:
                        pass


async def create_window(connection, name: str, x_pos: int = 180, width: int = 1180, height: int = 860):
    window = await iterm2.Window.async_create(connection)
    await asyncio.sleep(0.5)

    app = await iterm2.async_get_app(connection)
    if window.current_tab is None:
        for candidate in app.terminal_windows:
            if candidate.window_id == window.window_id:
                window = candidate
                break

    for _ in range(20):
        if window.current_tab and window.current_tab.current_session:
            break
        await asyncio.sleep(0.2)

    if not window.current_tab or not window.current_tab.current_session:
        raise RuntimeError(f"Window {name!r} did not become ready")

    session = window.current_tab.current_session
    await session.async_set_name(name)
    frame = await window.async_get_frame()
    await window.async_set_frame(
        iterm2.Frame(iterm2.Point(x_pos, frame.origin.y), iterm2.Size(width, height))
    )
    await asyncio.sleep(0.3)
    return window, session


def find_quartz_window_id(target_x, target_w, target_h, tolerance: int = 30):
    window_list = Quartz.CGWindowListCopyWindowInfo(
        Quartz.kCGWindowListOptionOnScreenOnly | Quartz.kCGWindowListExcludeDesktopElements,
        Quartz.kCGNullWindowID,
    )
    best_id, best_score = None, float("inf")
    for window in window_list:
        if "iTerm" not in window.get("kCGWindowOwnerName", ""):
            continue
        bounds = window.get("kCGWindowBounds", {})
        score = (
            abs(float(bounds.get("X", 0)) - target_x) * 2
            + abs(float(bounds.get("Width", 0)) - target_w)
            + abs(float(bounds.get("Height", 0)) - target_h)
        )
        if score < best_score:
            best_id, best_score = window.get("kCGWindowNumber"), score
    return best_id if best_score < tolerance else None


async def capture_screenshot(window, stem: str) -> Path:
    SCREENSHOT_DIR.mkdir(parents=True, exist_ok=True)
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    path = SCREENSHOT_DIR / f"{stem}_{timestamp}.png"
    frame = await window.async_get_frame()
    quartz_id = find_quartz_window_id(frame.origin.x, frame.size.width, frame.size.height)
    if quartz_id is None:
        raise RuntimeError("Could not correlate iTerm2 window for screenshot capture")
    subprocess.run(["screencapture", "-x", "-l", str(quartz_id), str(path)], check=True)
    print(f"SCREENSHOT: {path}")
    return path


async def screen_text(session) -> str:
    screen = await session.async_get_screen_contents()
    return "\n".join(screen.line(i).string for i in range(screen.number_of_lines))


async def wait_for_text(session, needles: list[str], timeout: float = TIMEOUT_SECONDS) -> str:
    start = time.monotonic()
    while time.monotonic() - start < timeout:
        text = await screen_text(session)
        for needle in needles:
            if needle in text:
                print(f"Found screen marker: {needle}")
                return text
        await asyncio.sleep(0.3)
    text = await screen_text(session)
    print("SCREEN DUMP ON FAILURE:\n")
    print(text)
    raise RuntimeError(f"Timed out waiting for any of: {needles}")


async def send_text(session, text: str, delay: float = 0.25) -> None:
    await session.async_send_text(text)
    await asyncio.sleep(delay)


async def main(connection):
    created_sessions = []
    await cleanup_stale_windows(connection)

    try:
        print_header("CREATE WINDOW")
        window, session = await create_window(connection, WINDOW_PREFIX + "main")
        created_sessions.append(session)

        print_header("LAUNCH VIVECAKA")
        await send_text(session, f"cd {TARGET_REPO_DIR}\n", 0.4)
        await send_text(session, f"{APPDIR / 'bin' / 'vivecaka'}\n", 1.5)
        await wait_for_text(session, [f"#{PR_NUMBER}", PR_TITLE, "Loading pull requests"], 25)
        await asyncio.sleep(2.0)

        print_header("OPEN PR DETAIL")
        await send_text(session, "\r", 1.5)
        await wait_for_text(session, ["Description", "Checks", "Files", "Comments"], 20)
        await wait_for_text(session, ["scope:", "no prior review baseline", "Δ "], 20)
        detail_shot = await capture_screenshot(window, "pr_detail_description")

        print_header("CAPTURE FILES TAB")
        await send_text(session, "\t\t", 1.6)
        await wait_for_text(session, [FILE_MARKER, "Files"], 20)
        await asyncio.sleep(1.0)
        files_shot = await capture_screenshot(window, "pr_detail_files")

        print_header("CAPTURE DIFF UNIFIED")
        await send_text(session, "d", 1.2)
        await wait_for_text(session, ["Unified", FILE_MARKER], 20)
        await asyncio.sleep(2.0)
        diff_unified = await capture_screenshot(window, "pr_diff_unified")

        print_header("CAPTURE DIFF SPLIT")
        await send_text(session, "t", 1.0)
        await wait_for_text(session, ["Split", FILE_MARKER], 20)
        await asyncio.sleep(1.5)
        diff_split = await capture_screenshot(window, "pr_diff_split")

        print_header("SUMMARY")
        print(f"Detail screenshot: {detail_shot}")
        print(f"Files screenshot: {files_shot}")
        print(f"Unified diff screenshot: {diff_unified}")
        print(f"Split diff screenshot: {diff_split}")
        return 0
    finally:
        for session in created_sessions:
            try:
                await session.async_send_text("\x03")
                await asyncio.sleep(0.1)
                await session.async_send_text("exit\n")
                await asyncio.sleep(0.1)
                await session.async_close()
            except Exception:
                pass


if __name__ == "__main__":
    sys.exit(iterm2.run_until_complete(main, retry=True))
