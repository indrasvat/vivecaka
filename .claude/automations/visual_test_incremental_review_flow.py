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
Incremental review flow verification for vivecaka.

Tests:
    1. Launch vivecaka against a real repository with an open PR.
    2. Open the PR, switch to the Files tab, and capture the initial incremental-review state.
    3. Mark the focused file as viewed and verify progress updates immediately.
    4. Quit and relaunch vivecaka, then verify the viewed progress persists for the same PR.
    5. Cycle scope to All and open the diff viewer to confirm the review header follows the restored state.

Verification Strategy:
    - Drive the real binary in iTerm2 using only keyboard input.
    - Wait for review-context markers before each screenshot.
    - Treat persistence as broken unless the reopened session shows the updated viewed count.
    - Fail loudly with a full screen dump on missing markers.

Screenshots:
    - flow_files_initial.png: Files tab before marking anything viewed.
    - flow_files_marked.png: Files tab after toggling viewed on the focused file.
    - flow_files_reopened.png: Files tab after relaunch, proving persistence.
    - flow_files_all_scope.png: Files tab after cycling to All.
    - flow_diff_reopened.png: Diff view after reopening with review header visible.

Key Bindings:
    - Enter: Open the selected PR.
    - 3: Switch to the Files tab.
    - V: Toggle viewed for the selected file.
    - i: Cycle incremental review scope.
    - d: Open the diff viewer.
    - q: Quit vivecaka.

Usage:
    uv run .claude/automations/visual_test_incremental_review_flow.py
"""

from __future__ import annotations

import asyncio
import os
import sys
import tempfile
from pathlib import Path

import iterm2

from vivecaka_iterm_helpers import (
    capture_screenshot,
    cleanup_stale_windows,
    create_window,
    launch_vivecaka,
    print_header,
    send_text,
    wait_for_text,
)


APPDIR = Path("/Users/robin.sharma/code/github.com/indrasvat/vivecaka")
BINARY = APPDIR / "bin" / "vivecaka"
TARGET_REPO_DIR = Path(
    os.environ.get("VIVECAKA_VISUAL_TARGET_DIR", "/Users/robin.sharma/code/github.com/indrasvat/dootsabha")
)
SCREENSHOT_DIR = APPDIR / "screenshots" / "incremental-review-flow"
WINDOW_PREFIX = "vivecaka-incremental-flow-"
PR_TITLE = os.environ.get("VIVECAKA_VISUAL_PR_TITLE", "Update default model to claude-opus-4-6")
PR_NUMBER = os.environ.get("VIVECAKA_VISUAL_PR_NUMBER", "11")
FILES_TOTAL = os.environ.get("VIVECAKA_VISUAL_FILES_TOTAL", "27")
READY_MARKERS = [f"#{PR_NUMBER}", PR_TITLE, "Loading pull requests"]


async def open_pr_files_tab(session) -> None:
    await send_text(session, "\r", 1.4)
    await wait_for_text(session, ["Description", "Checks", "Files", "Comments"], 20.0, require_all=True)
    await wait_for_text(session, ["scope:", "no prior review baseline", "Δ "], 20.0)
    await send_text(session, "3", 1.0)
    await wait_for_text(session, [f"0/{FILES_TOTAL} viewed", "1/"], 20.0)


async def main(connection):
    created_sessions = []
    await cleanup_stale_windows(connection, WINDOW_PREFIX)

    try:
        print_header("CREATE WINDOW")
        window, session = await create_window(connection, WINDOW_PREFIX + "main")
        created_sessions.append(session)

        with tempfile.TemporaryDirectory(prefix="vivecaka-flow-state-") as tmp_dir:
            root = Path(tmp_dir)
            command = (
                f"XDG_DATA_HOME={root / '.local' / 'share'} "
                f"XDG_CACHE_HOME={root / '.cache'} "
                f"{BINARY}"
            )

            print_header("FIRST LAUNCH")
            await launch_vivecaka(session, TARGET_REPO_DIR, command, READY_MARKERS)
            await open_pr_files_tab(session)
            await wait_for_text(session, [f"0/{FILES_TOTAL} viewed"], 20.0)
            initial = await capture_screenshot(window, SCREENSHOT_DIR, "flow_files_initial")

            print_header("MARK FILE VIEWED")
            await send_text(session, "V", 1.1)
            await wait_for_text(session, [f"1/{FILES_TOTAL} viewed"], 20.0)
            marked = await capture_screenshot(window, SCREENSHOT_DIR, "flow_files_marked")

            print_header("RELAUNCH AND VERIFY PERSISTENCE")
            await send_text(session, "q", 0.8)
            await send_text(session, command + "\n", 1.5)
            await wait_for_text(session, READY_MARKERS, 25.0)
            await asyncio.sleep(1.0)
            await open_pr_files_tab(session)
            await wait_for_text(session, [f"1/{FILES_TOTAL} viewed"], 20.0)
            reopened = await capture_screenshot(window, SCREENSHOT_DIR, "flow_files_reopened")

            print_header("CYCLE TO ALL SCOPE")
            await send_text(session, "i", 0.8)
            await wait_for_text(session, ["scope: Unviewed"], 20.0)
            await send_text(session, "i", 0.8)
            await wait_for_text(session, ["scope: All"], 20.0)
            all_scope = await capture_screenshot(window, SCREENSHOT_DIR, "flow_files_all_scope")

            print_header("OPEN DIFF")
            await send_text(session, "d", 1.2)
            await wait_for_text(session, ["scope:", "progress:"], 20.0, require_all=True)
            diff = await capture_screenshot(window, SCREENSHOT_DIR, "flow_diff_reopened")

        print_header("SUMMARY")
        print(f"Initial files: {initial}")
        print(f"Marked viewed: {marked}")
        print(f"Reopened files: {reopened}")
        print(f"All scope: {all_scope}")
        print(f"Diff: {diff}")
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
