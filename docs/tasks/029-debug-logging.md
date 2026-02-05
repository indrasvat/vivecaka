# Task 029: Debug Logging Infrastructure

## Status: TODO

## Problem

Currently there's no way to debug issues in vivecaka without adding temporary print statements that bleed into the TUI. Users and developers need a proper logging mechanism that:
- Doesn't interfere with the TUI rendering
- Can be enabled on demand
- Persists across sessions for post-mortem analysis

## Solution

Add structured debug logging using Go's `log/slog` package:
- Debug flag (`--debug` or `-d`) to enable logging
- Config option `debug = true` in TOML for persistent enablement
- Log to XDG state directory: `~/.local/state/vivecaka/debug.log`
- Rotate logs (keep last 5, max 10MB each)
- Never write to stdout/stderr during TUI operation

## PRD Reference

Not in original PRD. This is a developer experience improvement.

## Design

### Log Levels
- `DEBUG`: Verbose tracing (API calls, state changes, key events)
- `INFO`: Significant events (view transitions, PR loads)
- `WARN`: Recoverable issues (API timeouts, cache misses)
- `ERROR`: Failures that affect functionality

### Log Format
```
2026-02-05T08:43:10.123-08:00 DEBUG [tui/app] view_transition from=PRList to=PRDetail pr=142
2026-02-05T08:43:10.456-08:00 DEBUG [adapter/ghcli] api_call method=GetPR repo=owner/repo number=142 duration=1.2s
2026-02-05T08:43:10.789-08:00 INFO  [tui/views] pr_detail_loaded pr=142 files=8 checks=3 comments=1
```

### Activation
```bash
# Via flag
vivecaka --debug

# Via config (~/.config/vivecaka/config.toml)
[general]
debug = true

# Via environment variable
VIVECAKA_DEBUG=1 vivecaka
```

## Files to Create/Modify

- `internal/logging/logger.go` — New: slog wrapper with file rotation
- `internal/logging/logger_test.go` — New: tests
- `internal/config/config.go` — Add `Debug bool` field
- `cmd/vivecaka/main.go` — Add `--debug` flag, initialize logger
- `internal/tui/app.go` — Add logging calls for view transitions
- `internal/adapter/ghcli/*.go` — Add logging for API calls
- `internal/usecase/*.go` — Add logging for use case execution

## Implementation Steps

1. Create `internal/logging/logger.go`:
   - Initialize slog with file handler
   - XDG state path: `~/.local/state/vivecaka/debug.log`
   - Log rotation (lumberjack or custom)
   - No-op logger when debug disabled
   - Global `Log` variable for easy access

2. Update config:
   - Add `Debug bool` to config struct
   - Add `[general]` section with `debug` option

3. Update main.go:
   - Parse `--debug` / `-d` flag
   - Initialize logger before TUI starts
   - Ensure logger is flushed on exit

4. Add logging calls:
   - App: view transitions, message handling
   - Adapter: API request/response timing
   - Use cases: execution start/end, errors

5. Test:
   - Verify no output to stdout/stderr
   - Verify log file is created only when debug enabled
   - Verify log rotation works

## Key Constraints

- **CRITICAL**: Never write to stdout/stderr during TUI operation
- Log file must be in XDG state dir, not config or cache
- Debug mode must be explicitly enabled (not default)
- Logging overhead should be minimal when disabled (no-op)

## Verification

### Functional
```bash
# Test debug flag
vivecaka --debug &
sleep 2
kill %1
cat ~/.local/state/vivecaka/debug.log | head -20

# Test config option
echo -e '[general]\ndebug = true' >> ~/.config/vivecaka/config.toml
vivecaka &
sleep 2
kill %1
cat ~/.local/state/vivecaka/debug.log | head -20

# Verify no TUI bleed
vivecaka 2>&1 | grep -v "^$" | head  # Should show nothing
```

### Unit Tests
```bash
go test -race -v ./internal/logging/
```

## Commit

```
feat(logging): add debug logging infrastructure

- Add --debug flag and config option for debug mode
- Log to ~/.local/state/vivecaka/debug.log
- Use slog for structured logging with file rotation
- Add logging to app, adapter, and use cases
- Never write to stdout/stderr during TUI operation
```

## Session Protocol

1. Read `CLAUDE.md`
2. Read this task file
3. Execute
4. Verify (functional + unit tests)
5. Update `docs/PROGRESS.md` — mark this task done
6. `git add` changed files + `git commit`
7. Move to next task or end session
