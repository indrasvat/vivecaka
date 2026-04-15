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
Live dogfood verification for incremental review mode on the implementation PR itself.

Tests:
    1. Open the implementation PR in vivecaka.
    2. Submit a real review from the TUI to establish the local review baseline.
    3. Create a real follow-up commit on the same branch and push it to GitHub.
    4. Reopen the PR in vivecaka and verify Since Review narrows the follow-up work.
    5. Capture files and diff screenshots that prove only the follow-up review work is surfaced.

Verification Strategy:
    - Use the actual branch and PR under development, not fixtures.
    - Drive vivecaka in iTerm2 for the review and re-review steps.
    - Use git and push from Python subprocesses for the follow-up commit so the scenario is fully reproducible.
    - Fail fast if the review form cannot be submitted or if follow-up deltas are not visible afterward.

Screenshots:
    - dogfood_after_review.png
    - dogfood_since_review_files.png
    - dogfood_since_review_diff.png

Key Bindings:
    - Enter: Open the selected PR.
    - r: Open the review form.
    - 3: Switch to the Files tab.
    - d: Open the diff viewer.

Usage:
    VIVECAKA_DOGFOOD_PR_NUMBER=<pr> uv run .claude/automations/visual_test_incremental_review_dogfood.py
"""

from __future__ import annotations

import asyncio
import os
import subprocess
import sys
from datetime import datetime
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
TARGET_REPO_DIR = Path(os.environ.get("VIVECAKA_DOGFOOD_TARGET_DIR", str(APPDIR)))
SCREENSHOT_DIR = APPDIR / "screenshots" / "incremental-review-dogfood"
WINDOW_PREFIX = "vivecaka-incremental-dogfood-"
PR_NUMBER = os.environ["VIVECAKA_DOGFOOD_PR_NUMBER"]
PR_TITLE = os.environ.get("VIVECAKA_DOGFOOD_PR_TITLE", "incremental review mode")
FOLLOWUP_FILE = Path(os.environ.get("VIVECAKA_DOGFOOD_FOLLOWUP_FILE", "docs/tasks/031-incremental-review-mode.md"))
FOLLOWUP_COMMIT = os.environ.get(
    "VIVECAKA_DOGFOOD_FOLLOWUP_COMMIT",
    "test(dogfood): add follow-up marker for incremental review",
)


def run_git(*args: str) -> None:
    subprocess.run(["git", *args], cwd=TARGET_REPO_DIR, check=True)


def create_followup_commit() -> None:
    marker = f"\n<!-- dogfood-followup {datetime.now().isoformat()} -->\n"
    target = TARGET_REPO_DIR / FOLLOWUP_FILE
    target.write_text(target.read_text() + marker)
    run_git("add", str(FOLLOWUP_FILE))
    run_git("commit", "-m", FOLLOWUP_COMMIT)
    run_git("push")


async def submit_review_baseline(session) -> None:
    await send_text(session, "r", 1.0)
    await wait_for_text(session, ["Review Action", "Submit Review?"], 20.0, require_all=True)
    await send_text(session, "\t\t", 0.8)
    await send_text(session, "\r", 1.8)
    await wait_for_text(session, ["Review submitted", "Description"], 25.0)


async def open_files_tab(session) -> None:
    await send_text(session, "3", 1.0)
    await wait_for_text(session, ["Review", "scope:"], 20.0, require_all=True)


async def main(connection):
    created_sessions = []
    await cleanup_stale_windows(connection, WINDOW_PREFIX)

    try:
        window, session = await create_window(connection, WINDOW_PREFIX + "main")
        created_sessions.append(session)

        print_header("OPEN IMPLEMENTATION PR")
        await launch_vivecaka(session, TARGET_REPO_DIR, str(BINARY), [f"#{PR_NUMBER}", PR_TITLE, "Loading pull requests"])
        await send_text(session, "\r", 1.4)
        await wait_for_text(session, ["Description", "Checks", "Files", "Comments"], 20.0, require_all=True)
        await wait_for_text(session, ["scope:", "no prior review baseline", "Δ "], 20.0)

        print_header("SUBMIT REVIEW BASELINE")
        await submit_review_baseline(session)
        after_review = await capture_screenshot(window, SCREENSHOT_DIR, "dogfood_after_review")

        print_header("CREATE FOLLOW-UP COMMIT")
        await send_text(session, "q", 0.8)
        create_followup_commit()
        await send_text(session, f"{BINARY}\n", 1.5)
        await wait_for_text(session, [f"#{PR_NUMBER}", PR_TITLE, "Loading pull requests"], 25.0)
        await send_text(session, "\r", 1.4)
        await wait_for_text(session, ["Description", "Checks", "Files", "Comments"], 20.0, require_all=True)
        await wait_for_text(session, ["Δ ", "scope: Since Review"], 25.0)
        await open_files_tab(session)
        since_review_files = await capture_screenshot(window, SCREENSHOT_DIR, "dogfood_since_review_files")

        print_header("VERIFY DIFF")
        await send_text(session, "d", 1.2)
        await wait_for_text(session, ["scope:", "progress:"], 20.0, require_all=True)
        since_review_diff = await capture_screenshot(window, SCREENSHOT_DIR, "dogfood_since_review_diff")

        print_header("SUMMARY")
        print(after_review)
        print(since_review_files)
        print(since_review_diff)
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
