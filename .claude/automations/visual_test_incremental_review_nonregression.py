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
Non-regression verification for core PR review surfaces after incremental review mode.

Tests:
    1. Launch vivecaka against a real repository and open a live PR.
    2. Capture the Description tab with the incremental-review bar present.
    3. Capture the Comments tab to ensure discussion rendering still works.
    4. Open the diff viewer and capture unified and split modes.

Verification Strategy:
    - Reuse the same live PR across all surfaces so counts and content stay comparable.
    - Verify each screen with concrete text markers before capture.
    - Treat missing comments, missing diff panes, or missing review headers as regressions.

Screenshots:
    - nonreg_description.png
    - nonreg_comments.png
    - nonreg_diff_unified.png
    - nonreg_diff_split.png

Key Bindings:
    - Enter: Open the selected PR.
    - 4: Switch to the Comments tab.
    - d: Open the diff viewer.
    - t: Toggle split mode.

Usage:
    uv run .claude/automations/visual_test_incremental_review_nonregression.py
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
SCREENSHOT_DIR = APPDIR / "screenshots" / "incremental-review-nonregression"
WINDOW_PREFIX = "vivecaka-incremental-nonreg-"
PR_TITLE = os.environ.get("VIVECAKA_VISUAL_PR_TITLE", "Update default model to claude-opus-4-6")
PR_NUMBER = os.environ.get("VIVECAKA_VISUAL_PR_NUMBER", "11")
COMMENT_MARKER = os.environ.get("VIVECAKA_VISUAL_COMMENT_MARKER", "Please verify the migration path")
async def main(connection):
    created_sessions = []
    await cleanup_stale_windows(connection, WINDOW_PREFIX)

    try:
        window, session = await create_window(connection, WINDOW_PREFIX + "main")
        created_sessions.append(session)

        print_header("LAUNCH")
        await launch_vivecaka(session, TARGET_REPO_DIR, str(BINARY), [f"#{PR_NUMBER}", PR_TITLE, "Loading pull requests"])

        print_header("DESCRIPTION")
        await send_text(session, "\r", 1.4)
        await wait_for_text(session, ["Description", "Checks", "Files", "Comments"], 20.0, require_all=True)
        await wait_for_text(session, ["scope:", "no prior review baseline", "Δ "], 20.0)
        description = await capture_screenshot(window, SCREENSHOT_DIR, "nonreg_description")

        print_header("COMMENTS")
        await send_text(session, "4", 1.0)
        await wait_for_text(session, [COMMENT_MARKER, "Comments"], 20.0)
        comments = await capture_screenshot(window, SCREENSHOT_DIR, "nonreg_comments")

        print_header("DIFF UNIFIED")
        await send_text(session, "d", 1.2)
        await wait_for_text(session, ["Tab pane", "c comment"], 20.0, require_all=True)
        await asyncio.sleep(1.0)
        diff_unified = await capture_screenshot(window, SCREENSHOT_DIR, "nonreg_diff_unified")

        print_header("DIFF SPLIT")
        await send_text(session, "t", 1.0)
        await wait_for_text(session, ["Tab pane", "c comment"], 20.0, require_all=True)
        await asyncio.sleep(1.0)
        diff_split = await capture_screenshot(window, SCREENSHOT_DIR, "nonreg_diff_split")

        print_header("SUMMARY")
        print(description)
        print(comments)
        print(diff_unified)
        print(diff_split)
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
