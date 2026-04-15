from __future__ import annotations

import asyncio
import subprocess
import time
from datetime import datetime
from pathlib import Path

import iterm2
import Quartz


def print_header(title: str) -> None:
    print(f"\n{'=' * 72}")
    print(title)
    print("=" * 72)


async def cleanup_stale_windows(connection, prefix: str) -> None:
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


async def create_window(connection, name: str, x_pos: int = 160, width: int = 1180, height: int = 860):
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


async def capture_screenshot(window, out_dir: Path, stem: str) -> Path:
    out_dir.mkdir(parents=True, exist_ok=True)
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    path = out_dir / f"{stem}_{timestamp}.png"
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


async def wait_for_text(session, needles: list[str], timeout: float = 20.0, require_all: bool = False) -> str:
    start = time.monotonic()
    while time.monotonic() - start < timeout:
        text = await screen_text(session)
        if require_all:
            if all(needle in text for needle in needles):
                return text
        else:
            for needle in needles:
                if needle in text:
                    return text
        await asyncio.sleep(0.3)
    text = await screen_text(session)
    print("SCREEN DUMP ON FAILURE:\n")
    print(text)
    raise RuntimeError(f"Timed out waiting for markers: {needles}")


async def wait_for_absent(session, needles: list[str], timeout: float = 20.0) -> str:
    start = time.monotonic()
    while time.monotonic() - start < timeout:
        text = await screen_text(session)
        if all(needle not in text for needle in needles):
            return text
        await asyncio.sleep(0.3)
    text = await screen_text(session)
    print("SCREEN DUMP ON FAILURE:\n")
    print(text)
    raise RuntimeError(f"Timed out waiting for markers to disappear: {needles}")


async def send_text(session, text: str, delay: float = 0.25) -> None:
    await session.async_send_text(text)
    await asyncio.sleep(delay)


async def launch_vivecaka(session, repo_dir: Path, command: str, ready_markers: list[str], timeout: float = 25.0) -> None:
    await send_text(session, f"cd {repo_dir}\n", 0.4)
    await send_text(session, command + "\n", 1.5)
    start = time.monotonic()
    while time.monotonic() - start < timeout:
        text = await screen_text(session)
        for marker in ready_markers:
            if marker in text:
                await asyncio.sleep(1.5)
                return
        if "Welcome to vivecaka!" in text:
            await send_text(session, "\x1b", 0.6)
        await asyncio.sleep(0.3)
    text = await screen_text(session)
    print("SCREEN DUMP ON FAILURE:\n")
    print(text)
    raise RuntimeError(f"Timed out waiting for markers: {ready_markers}")
