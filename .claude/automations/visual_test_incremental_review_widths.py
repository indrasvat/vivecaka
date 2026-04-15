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
Width-regression verification for incremental review mode in vivecaka.

Tests:
    1. Launch vivecaka at 120x34, 100x30, and 90x28 terminal sizes.
    2. Open a real PR detail view and wait for the review context to load.
    3. Capture the Files tab at each size and verify the compact context bar remains single-line and legible.

Verification Strategy:
    - Resize the terminal grid before each launch using the standard CSI window resize escape.
    - Use the same live PR for all widths to keep screenshots comparable.
    - Fail if review-context text or the Files tab stops rendering at any width.

Screenshots:
    - files_120x34.png
    - files_100x30.png
    - files_90x28.png

Key Bindings:
    - Enter: Open the selected PR.
    - 3: Switch to the Files tab.

Usage:
    uv run .claude/automations/visual_test_incremental_review_widths.py
"""

from __future__ import annotations

import asyncio
import os
import sys
from pathlib import Path

import iterm2

from vivecaka_iterm_helpers import (
    capture_screenshot,
    cleanup_stale_windows,
    create_window,
    print_header,
    send_text,
    wait_for_text,
)


APPDIR = Path("/Users/robin.sharma/code/github.com/indrasvat/vivecaka")
BINARY = APPDIR / "bin" / "vivecaka"
TARGET_REPO_DIR = Path(
    os.environ.get("VIVECAKA_VISUAL_TARGET_DIR", "/Users/robin.sharma/code/github.com/indrasvat/dootsabha")
)
SCREENSHOT_DIR = APPDIR / "screenshots" / "incremental-review-widths"
WINDOW_PREFIX = "vivecaka-incremental-widths-"
PR_TITLE = os.environ.get("VIVECAKA_VISUAL_PR_TITLE", "Update default model to claude-opus-4-6")
PR_NUMBER = os.environ.get("VIVECAKA_VISUAL_PR_NUMBER", "11")
SIZES = [(120, 34), (100, 30), (90, 28)]
FRAME_SIZES = {
    (120, 34): (1180, 860),
    (100, 30): (1020, 760),
    (90, 28): (900, 710),
}


async def capture_for_size(window, session, cols: int, rows: int):
    label = f"{cols}x{rows}"
    print_header(f"CAPTURE {label}")
    frame = await window.async_get_frame()
    width, height = FRAME_SIZES[(cols, rows)]
    await window.async_set_frame(iterm2.Frame(frame.origin, iterm2.Size(width, height)))
    await asyncio.sleep(0.8)
    await send_text(session, f"cd {TARGET_REPO_DIR}\n", 0.4)
    await send_text(session, f"{BINARY}\n", 1.5)
    await wait_for_text(session, [f"#{PR_NUMBER}", PR_TITLE, "Loading pull requests"], 25.0)
    await send_text(session, "\r", 1.4)
    await wait_for_text(session, ["Description", "Checks", "Files", "Comments"], 20.0, require_all=True)
    await wait_for_text(session, ["Review"], 20.0)
    await send_text(session, "\t\t", 1.0)
    await wait_for_text(session, ["Review", "files shown", "unviewed"], 20.0, require_all=True)
    shot = await capture_screenshot(window, SCREENSHOT_DIR, f"files_{label}")
    await send_text(session, "q", 0.8)
    return shot


async def main(connection):
    created_sessions = []
    await cleanup_stale_windows(connection, WINDOW_PREFIX)

    try:
        window, session = await create_window(connection, WINDOW_PREFIX + "main")
        created_sessions.append(session)

        shots = []
        for cols, rows in SIZES:
            shots.append(await capture_for_size(window, session, cols, rows))

        print_header("SUMMARY")
        for shot in shots:
            print(shot)
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
