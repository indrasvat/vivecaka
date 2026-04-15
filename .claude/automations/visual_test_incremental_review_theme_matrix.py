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
Theme-matrix verification for incremental review mode in vivecaka.

Tests:
    1. Launch vivecaka once per supported theme using isolated XDG config homes.
    2. Open the same real PR and switch to the Files tab.
    3. Capture the incremental-review UI for each theme to verify color separation and no background bleed.

Verification Strategy:
    - Write a minimal config file per theme under a temporary XDG config root.
    - Run the real binary with that config so screenshots reflect the true app styling path.
    - Treat missing review-context markers or unreadable contrast as failures.

Screenshots:
    - theme_default-dark.png
    - theme_catppuccin-mocha.png
    - theme_catppuccin-frappe.png
    - theme_tokyo-night.png
    - theme_dracula.png

Key Bindings:
    - Enter: Open the selected PR.
    - 3: Switch to the Files tab.

Usage:
    uv run .claude/automations/visual_test_incremental_review_theme_matrix.py
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
    print_header,
    send_text,
    wait_for_text,
)


APPDIR = Path("/Users/robin.sharma/code/github.com/indrasvat/vivecaka")
BINARY = APPDIR / "bin" / "vivecaka"
TARGET_REPO_DIR = Path(
    os.environ.get("VIVECAKA_VISUAL_TARGET_DIR", "/Users/robin.sharma/code/github.com/indrasvat/dootsabha")
)
SCREENSHOT_DIR = APPDIR / "screenshots" / "incremental-review-theme-matrix"
WINDOW_PREFIX = "vivecaka-incremental-themes-"
PR_TITLE = os.environ.get("VIVECAKA_VISUAL_PR_TITLE", "Update default model to claude-opus-4-6")
PR_NUMBER = os.environ.get("VIVECAKA_VISUAL_PR_NUMBER", "11")
GH_CONFIG_DIR = Path(os.environ.get("GH_CONFIG_DIR", str(Path.home() / ".config" / "gh")))
THEMES = [
    "default-dark",
    "catppuccin-mocha",
    "catppuccin-frappe",
    "tokyo-night",
    "dracula",
]


def write_theme_config(root: Path, theme: str) -> Path:
    config_dir = root / "vivecaka"
    config_dir.mkdir(parents=True, exist_ok=True)
    config_path = config_dir / "config.toml"
    config_path.write_text(
        "\n".join(
            [
                "[general]",
                f'theme = "{theme}"',
                "refresh_interval = 0",
                "page_size = 50",
                "",
                "[diff]",
                'mode = "unified"',
                'markdown_style = "dark"',
                "",
            ]
        )
    )
    return config_path


async def capture_theme(window, session, theme: str):
    with tempfile.TemporaryDirectory(prefix=f"vivecaka-theme-{theme}-") as tmp_dir:
        config_root = Path(tmp_dir)
        write_theme_config(config_root / ".config", theme)
        command = f"GH_CONFIG_DIR={GH_CONFIG_DIR} XDG_CONFIG_HOME={config_root / '.config'} {BINARY}"

        print_header(f"THEME {theme}")
        await send_text(session, f"cd {TARGET_REPO_DIR}\n", 0.4)
        await send_text(session, command + "\n", 1.5)
        await wait_for_text(session, [f"#{PR_NUMBER}", PR_TITLE, "Loading pull requests"], 25.0)
        await send_text(session, "\r", 1.4)
        await wait_for_text(session, ["Description", "Checks", "Files", "Comments"], 20.0, require_all=True)
        await wait_for_text(session, ["scope:", "no prior review baseline", "Δ "], 20.0)
        await send_text(session, "3", 1.0)
        await wait_for_text(session, ["Review", "scope:"], 20.0, require_all=True)
        shot = await capture_screenshot(window, SCREENSHOT_DIR, f"theme_{theme}")
        await send_text(session, "q", 0.8)
        return shot


async def main(connection):
    created_sessions = []
    await cleanup_stale_windows(connection, WINDOW_PREFIX)

    try:
        window, session = await create_window(connection, WINDOW_PREFIX + "main")
        created_sessions.append(session)

        shots = []
        for theme in THEMES:
            shots.append(await capture_theme(window, session, theme))

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
