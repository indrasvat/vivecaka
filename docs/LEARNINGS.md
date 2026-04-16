# vivecaka — Learnings

> Capture implementation discoveries that are easy to forget and expensive to relearn. Update this file whenever a bug fix, visual regression, or integration quirk reveals a non-obvious rule.

## BubbleTea / Terminal Rendering

1. **Bubble Tea v1 does not restore terminal background for you**. `vivecaka` is on `github.com/charmbracelet/bubbletea v1.3.10`, so the `luma` Bubble Tea v2 pattern using `tea.View.BackgroundColor` is not directly available here.
2. **`termenv.Output.Reset()` is not sufficient after `SetBackgroundColor()`**. `output.Reset()` only sends SGR reset (`\033[0m`), which clears text attributes but does **not** undo terminal default background changes from OSC 11.
3. **If you set terminal background with OSC 11, you must restore it with OSC 111**. For `vivecaka`, the correct shutdown sequence is: set default background at app start, then emit `\x1b]111\x07` on exit to restore the shell's native background.
4. **Keep OSC background changes scoped to TUI runtime only**. Background mutation belongs in the app launch path, not generic CLI paths like `--help` or `--version`.
5. **Use `termenv.RGBColor` for deterministic OSC payloads**. `output.Color("#...")` can depend on terminal profile detection and may collapse on non-terminal writers in tests. `termenv.RGBColor("#1E1E2E")` preserves the exact color in the emitted OSC 11 sequence.
6. **The symptom can survive process exit and look like a terminal bug**. If the whole shell stays purple after quitting vivecaka, assume an OSC 11 cleanup bug before blaming Lip Gloss or alt-screen teardown.

## Visual Verification

1. **Terminal background bugs must be verified in a real terminal emulator**. Unit tests can validate emitted escape sequences, but only iTerm2/Ghostty screenshots prove whether the shell background is actually restored after exit.
2. **The screenshot sequence must include both pre-launch and post-exit states in the same window**. For this class of regression, the minimum useful visual evidence is:
   - fresh terminal
   - vivecaka banner
   - one or more active TUI screens
   - the same terminal after exit
3. **Wait for TUI states to settle before capturing**. PR list/detail/comment screens can briefly render loading states even after the right tab labels appear. Require the loading marker to disappear before taking the screenshot.
4. **Do not depend on custom shell prompts in automation**. Prompt frameworks like Starship and Powerlevel10k will ignore `PS1`. Use explicit sentinel text (`printf 'marker\n'`) and wait for that instead.
5. **For background-bleed checks, compare empty terminal area, not just prompt chrome**. Prompt themes can hide subtle differences; the large unused terminal background is the real signal.

## Regression Guardrails

1. **Add a direct test for the reset sequence**. This bug is easy to reintroduce with a “cleanup” refactor, so keep a test that asserts the app emits OSC 111 on shutdown.
2. **Document version-specific behavior explicitly**. When another project like `luma` solves the same problem differently, first check framework/runtime versions before copying the implementation.
