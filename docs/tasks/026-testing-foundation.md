# Task 026: Testing Foundation (testify Migration, Adapter Fixtures)

## Status: TODO

## Problem

Multiple testing gaps:
1. **testify not used**: Listed as dependency but never imported. All tests use raw `if/t.Error` patterns.
2. **Adapter fixtures missing**: `testdata/` directory doesn't exist. No JSON fixture files. Adapter coverage is 25.3% (parser only).
3. **TUI tests shallow**: Most verify "doesn't panic" and "returns non-empty string", not actual content.

## PRD Reference

Read `CLAUDE.md` sections:
- Testing Requirements — testify, fixture data, coverage targets
- Read `docs/PRD.md` testing sections

## Files to Create/Modify

- `internal/adapter/ghcli/testdata/` (CREATE DIR) — JSON fixture files
- `internal/adapter/ghcli/testdata/pr_list.json` — sample PR list response
- `internal/adapter/ghcli/testdata/pr_detail.json` — sample PR detail response
- `internal/adapter/ghcli/testdata/pr_diff.txt` — sample diff output
- `internal/adapter/ghcli/testdata/pr_checks.json` — sample checks response
- `internal/adapter/ghcli/testdata/pr_comments.json` — sample comments response
- `internal/adapter/ghcli/reader_test.go` (CREATE or EXTEND) — fixture-based tests
- `internal/adapter/ghcli/reviewer_test.go` (CREATE) — mock-based tests
- Migrate ALL existing test files to use testify assertions

## Execution Steps

1. Read `CLAUDE.md`
2. Verify `testify` is in go.mod: `github.com/stretchr/testify`
3. **Create fixture data**:
   - Capture real `gh pr list --json ...` output for a repo with diverse PRs
   - Capture `gh pr view --json ...` output
   - Capture `gh pr diff` output
   - Capture `gh pr checks --json ...` output
   - Capture `gh api repos/.../pulls/.../comments` output
   - Save as JSON/text files in `testdata/`
4. **Create adapter tests**:
   - Test JSON parsing of each fixture into domain types
   - Mock exec command to return fixture data instead of calling real gh
   - Target 80%+ coverage on reader, reviewer, writer
5. **Migrate to testify** across all test files:
   - Replace `if got != want { t.Errorf(...) }` with `assert.Equal(t, want, got)`
   - Replace `if err != nil { t.Fatal(err) }` with `require.NoError(t, err)`
   - Replace `if x == nil { t.Fatal("nil") }` with `require.NotNil(t, x)`
   - Do this systematically across ALL test files
6. Run `make ci`
7. Check coverage: `go test -coverprofile=coverage.out ./internal/adapter/ghcli/...`

## Verification

### Functional
```bash
go test -race -v ./internal/adapter/ghcli/...
go test -race -v ./...
make ci
# Coverage check:
go test -coverprofile=coverage.out ./internal/adapter/ghcli/... && go tool cover -func=coverage.out | tail -1
```

Verify adapter coverage >= 80%.

### No Visual (testing-only task)
This task is code-only. No iterm2-driver verification needed.

## Commit

```
test: add adapter fixtures, migrate to testify, improve coverage

Created testdata/ with JSON fixtures for all adapter responses.
Added fixture-based tests for reader/reviewer/writer (80%+ coverage).
Migrated all test files from raw if/t.Error to testify assert/require.
```

## Session Protocol

1. Read `CLAUDE.md`
2. Read this task file
3. Execute
4. Verify (functional only — coverage numbers)
5. Update `docs/PROGRESS.md` — mark this task done, update coverage numbers
6. `git add` changed files + `git commit`
7. Move to next task or end session
