# Task 024: Config Enhancements (Keybinding Overrides, Adaptive Colors, Notifications)

## Status: DONE

## Problem

Multiple config features exist as struct fields but are never wired:
1. **Keybinding overrides**: `[keybindings]` config section defined but nothing reads or applies overrides
2. **Adaptive colors**: No `lipgloss.AdaptiveColor` usage — static colors only, breaking on light terminals
3. **Notification config**: `[notifications]` section from PRD completely absent from config struct

## PRD Reference

Read `docs/PRD.md` sections:
- Config section — keybindings, notifications, theme
- Look for keybinding override format, notification preferences
- F11 (Themes) — adaptive colors spec

## Files to Modify

- `internal/config/config.go` — add `[notifications]` section, verify keybindings section
- `internal/tui/core/keymap.go` — add `ApplyOverrides(overrides map[string]string)` method
- `internal/tui/core/theme.go` — convert static colors to `lipgloss.AdaptiveColor` for light/dark detection
- `internal/tui/app.go` — apply keybinding overrides from config on init
- `internal/config/config_test.go` — test override parsing
- `internal/tui/core/keymap_test.go` — test override application

## Execution Steps

1. Read `CLAUDE.md`
2. Read relevant PRD sections
3. **Keybinding overrides**:
   - Config format: `[keybindings] quit = "ctrl+q"` or similar
   - Parse override strings into `key.Binding` values
   - Apply overrides in `New()` after creating default keymap
4. **Adaptive colors**:
   - In theme definitions, use `lipgloss.AdaptiveColor{Light: "#...", Dark: "#..."}`
   - At minimum for: background, foreground, border, accent colors
   - Test on both dark and light terminal backgrounds
5. **Notification config**:
   - Add `NotificationsConfig` struct: `Enabled bool`, `Sound bool`, `DesktopNotify bool`
   - Add `[notifications]` section to config TOML
   - Wire into toast/auto-refresh notification logic
6. Add tests
7. Run `make ci`

## Verification

### Functional
```bash
go test -race -v ./internal/config/
go test -race -v ./internal/tui/core/
make ci
```

### Visual (iterm2-driver)
Create and run an iterm2-driver script that:
1. Set up config with custom keybinding override (e.g., quit = "ctrl+q")
2. Launch `bin/vivecaka` — verify custom key works
3. Test adaptive colors if possible (may need light terminal profile)
4. Screenshots showing proper rendering

## Commit

```
feat(config): wire keybinding overrides, adaptive colors, notifications

Keybinding overrides from config.toml now applied on startup.
Theme colors use lipgloss.AdaptiveColor for light/dark detection.
Added [notifications] config section for future notification features.
```

## Session Protocol

1. Read `CLAUDE.md`
2. Read this task file
3. Read relevant PRD section
4. Execute
5. Verify (functional + visual)
6. Update `docs/PROGRESS.md` — mark this task done
7. `git add` changed files + `git commit`
8. Move to next task or end session
