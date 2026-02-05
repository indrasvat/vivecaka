# Task 009: Syntax Highlighting in Diff (Chroma)

## Status: TODO

## Depends On: 008 (Glamour may bring Chroma as transitive dep)

## Problem

Diff content is rendered as plain text with only add/delete coloring (green/red). The PRD specifies syntax highlighting via Chroma, which should colorize code within diff lines based on file extension.

## PRD Reference

Read `docs/PRD.md` sections:
- Section 7.3 (Diff Viewer) — syntax highlighting via Chroma
- Look for code coloring specs, theme integration

## Files to Modify

- `go.mod` — add `github.com/alecthomas/chroma/v2` dependency (may already be transitive via Glamour)
- `internal/tui/views/diffview.go` — apply syntax highlighting to diff line content
- `internal/tui/views/diffview_test.go` — test that highlighted output differs from plain text

## Execution Steps

1. Read `CLAUDE.md`
2. Read relevant PRD sections
3. Check if Chroma is already a transitive dependency from Glamour
4. Create a helper `highlightDiffLine(line string, filename string) string`:
   - Detect language from filename extension using `chroma/lexers`
   - Use a terminal-friendly formatter (`chroma/formatters/terminal256`)
   - Pick a style that complements the active theme (Catppuccin Mocha → use monokai or similar dark style)
   - Only highlight the code portion, preserve +/- prefix and line number styling
   - Cache lexer per file (don't re-detect per line)
5. Apply highlighting in the line rendering loop
6. Handle edge cases: binary files, unknown extensions, very long lines
7. Ensure add/delete background colors still show through syntax colors
8. Add tests
9. Run `make ci`

## Verification

### Functional
```bash
go test -race -v ./internal/tui/views/ -run TestDiffView
make ci
```

### Visual (iterm2-driver)
Create and run an iterm2-driver script that:
1. Launches `bin/vivecaka` in a repo with PRs modifying Go/JS/Python files
2. Opens a PR, navigates to diff view
3. Screenshots the diff — verify syntax colors are visible (keywords, strings, comments in different colors)
4. Compare against the non-highlighted state (before this task)
5. Switch between files of different types — verify language detection works

## Commit

```
feat(tui): add syntax highlighting to diff viewer via Chroma

Diff lines now have language-aware syntax coloring based on file
extension. Uses Chroma lexers with terminal256 formatter. Add/delete
background colors preserved alongside syntax colors.
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
