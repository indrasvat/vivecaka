# vivecaka — Progress Tracker

> Updated after each task completion. See `docs/PRD.md` for full task definitions.
> Task files: `docs/tasks/NNN-task-name.md`

## Current Status

**Phase 14: Feature Completion & Quality — IN PROGRESS**

Previously, Phases 0-13 built the scaffolding. An audit revealed ~25 features are missing or non-functional despite being marked "Done". Phase 14 addresses all gaps identified in the audit.

## How to Resume

1. Read this file — find the next `TODO` task below
2. Read the task file at `docs/tasks/NNN-task-name.md` — full context and steps
3. Read `CLAUDE.md` for conventions
4. Read relevant `docs/PRD.md` sections (listed in each task file)
5. Execute, verify (functional + visual via iterm2-driver), update this file, commit

## Phase 14: Feature Completion (Remaining Tasks)

| # | Task File | Description | Status | Depends On |
|---|-----------|-------------|--------|------------|
| 001 | `docs/tasks/001-fix-pr-list-sort.md` | Fix sort to actually reorder data + direction indicator | DONE | — |
| 002 | `docs/tasks/002-clipboard-and-browser.md` | Wire y (copy URL) and o (open browser) keys | DONE | — |
| 003 | `docs/tasks/003-pr-detail-keybindings.md` | Add d/c/o keys to PR detail + CI summary line | DONE | 002 |
| 004 | `docs/tasks/004-fix-diff-search.md` | Make diff search actually highlight + n/N navigation | DONE | — |
| 005 | `docs/tasks/005-diff-navigation.md` | Hunk jump [/], file jump {/}, gg/G, collapse za | DONE | — |
| 006 | `docs/tasks/006-quick-filters.md` | Quick filters: m (My PRs), n (Needs Review) | DONE | — |
| 007 | `docs/tasks/007-filter-panel.md` | Full filter panel overlay with f key | DONE | 006 |
| 008 | `docs/tasks/008-markdown-rendering.md` | Render PR body/comments as markdown via Glamour | DONE | — |
| 009 | `docs/tasks/009-syntax-highlighting.md` | Syntax highlighting in diff via Chroma | DONE | 008 |
| 010 | `docs/tasks/010-comments-enhancement.md` | Comment collapse/expand, reply, resolve | DONE | 008 |
| 011 | `docs/tasks/011-review-form-huh.md` | Rewrite review form using huh library | DONE | — |
| 012 | `docs/tasks/012-confirmation-dialogs.md` | Add confirmation before checkout and review submit | DONE | — |
| 013 | `docs/tasks/013-repo-switcher-wiring.md` | Load favorites from config into repo switcher | DONE | — |
| 014 | `docs/tasks/014-inbox-wiring.md` | Wire inbox: I key, multi-repo fetch, priority sort | DONE | 006, 013 |
| 015 | `docs/tasks/015-auto-refresh.md` | Background polling with countdown, pause, toasts | DONE | — |
| 016 | `docs/tasks/016-startup-experience.md` | Wire tutorial, startup banner, branch detection | DONE | — |
| 017 | `docs/tasks/017-external-diff-tool.md` | e key to launch external diff tool | DONE | — |
| 018 | `docs/tasks/018-inline-comments.md` | Inline comments in diff view (c/r/x keys) | TODO | 010 |
| 019 | `docs/tasks/019-diff-two-pane-layout.md` | File tree + content split layout for diff | TODO | — |
| 020 | `docs/tasks/020-diff-side-by-side.md` | Side-by-side diff mode with t toggle | TODO | 019 |
| 021 | `docs/tasks/021-visual-selection-mode.md` | v key for multi-select, batch operations | DONE | — |
| 022 | `docs/tasks/022-pagination.md` | Paginated PR loading with infinite scroll | DONE | — |
| 023 | `docs/tasks/023-caching.md` | PR list caching for instant startup | DONE | — |
| 024 | `docs/tasks/024-config-enhancements.md` | Keybinding overrides, adaptive colors, notifications | DONE | — |
| 025 | `docs/tasks/025-persistence.md` | Per-repo filter memory, unread indicators | DONE | 001, 007, 023 |
| 026 | `docs/tasks/026-testing-foundation.md` | testify migration, adapter fixtures, 80%+ coverage | TODO | — |
| 027 | `docs/tasks/027-integration-tests.md` | teatest integration tests, TUI test quality | TODO | 026 |
| 028 | `docs/tasks/028-pr-detail-tabs-layout.md` | Redesign PR detail with horizontal tabs layout | DONE | — |
| 029 | `docs/tasks/029-debug-logging.md` | Debug logging infrastructure with --debug flag | DONE | — |

Notes:
- Task 007: filter panel matches mock (static label options: enhancement/bug/docs; CI filter options: All/Passing/Failing; Review filter options: All/Approved/Pending). Pending CI + changes-requested review filters not exposed yet.
- Task 008: iterm2-driver markdown QA could not open PR detail (no open PRs in repo). Captured PR list screenshot only.
- Task 028: PR detail redesigned with 4 horizontal tabs (Description, Checks, Files, Comments). Tab bar has counts, active tab uses Primary color. Number keys 1-4, Tab/Shift-Tab, j/k/g/G navigation all working. Visual QA blocked by 0 open PRs in repo; functional tests pass.
- Task 011: **Fix applied (Feb 5, 2026):** Review form was rendering empty because `m.form.Init()` was never dispatched when entering review mode. Fixed by returning `a.reviewForm.Init()` from the `StartReviewMsg` handler. Also added `WithWidth` propagation. Verified on openclaw/openclaw: huh form renders with action select (Comment/Approve/Request Changes), text body, and Submit/Cancel confirmation. Navigation with arrow keys works. Escape cancels and returns to detail.
- Task 012: Reusable `ConfirmModel` with three-phase dialog: prompt → loading (animated spinner) → result (green ✓ success / red ✗ error with "Press any key to continue"). Checkout now shows "Check out branch X for PR #N?" → spinner during `gh pr checkout` → success result with branch name. Dialog stays centered and visible throughout — no more ephemeral toasts. Keys: Enter/y confirm, Esc/n cancel. `ViewConfirm` state added. `CheckoutPRMsg` extended with Branch field. **Bug fix:** `gh pr checkout` outputs to stderr, not stdout — adapter now uses `git branch --show-current` after checkout for reliable branch name. Verified end-to-end on openclaw/openclaw: confirmation, cancel (both n and Esc), actual checkout with loading spinner, success result, dismiss, and branch restoration all working.
- Task 022: Pagination implemented and fully verified. Initial load uses config.PageSize (default 50). When cursor nears bottom (within 5 items), LoadMorePRsMsg triggers, animated braille spinner shows, and PRs append via cumulative fetch. Header shows "loaded/total" format (e.g., "100/1896 open") using GraphQL API for total count. Tested on openclaw/openclaw repo (1,896 PRs). **Fix applied (Feb 5, 2026):** The original implementation used full field list including `statusCheckRollup` which caused GitHub API timeouts (HTTP 504) when fetching 100+ PRs. Fixed by using `prListFieldsLight` (excludes statusCheckRollup) for pagination requests. CI status will show as "—" for paginated items until detail view is opened; this is an acceptable tradeoff for reliable pagination. Verified: 50→100→150 PRs load successfully with ~1.5s per page.

Banner Polish:
- Replaced Devanagari विवेचक with 5 decorative Unicode symbol trios (e.g., ⟁·⟐·⌬, ⌖·⟡·⊹). Symbols are theme-colored (Primary/Secondary/Primary, muted dots) and rotate every 400ms during the 2s banner display.

- Task 013: Full repo switcher rewrite with three-source data model (favorites from config, user repos via `gh repo list` lazy/cached, manual owner/repo entry with ghost add). Aesthetic redesign: `⟡ Switch Repository` title, `❯` search prompt, `★` star indicator (Warning color), `●` current dot (Success), `▸` cursor, BgDim selected row, sectioned list with `╶── FAVORITES ──╴` / `╶── YOUR REPOS ──╴` headers, scroll indicators ▲/▼, key hints with theme colors. `s` key toggles favorite with config write-back via `UpdateFavorites()`. Fuzzy search filters both sections; ghost `+ add owner/repo` entry when query contains `/` and no match. Dedup: discovered repos exclude favorites. CWD repo auto-prepended to favorites on startup. 28 unit tests (model, view, search, toggle, ghost, sections, discovery). Config `UpdateFavorites()` round-trips TOML. Adapter `ListUserRepos()` + `ValidateRepo()` via `gh repo list/view`.

- Task 015: Auto-refresh polling implemented with `tea.Tick` every second. Countdown shown in header as `↻ 25s`. `p` key pauses/resumes (shows `⏸ paused`). When countdown reaches 0, triggers `loadPRsCmd` silently. Compares new PR count with previous; shows toast when new PRs detected. Refresh pauses automatically during non-list views (detail, diff, review). Uses config `refresh_interval` (default 30s).
- Task 016: Branch detection via `git rev-parse --abbrev-ref HEAD` on startup. Branch displayed in header as `⎇ main`. Tutorial `Show()` now called from `Init()` when `IsFirstLaunch()` returns true. Branch detection fires alongside repo detection in Init().
- Task 017: `e` key in diff view emits `OpenExternalDiffMsg`. App handles it via `tea.ExecProcess` with `gh pr diff N` and `GH_PAGER` env var set to configured external tool. If no tool configured, shows toast error. After tool exits, TUI resumes. Status hints updated to show `e ext diff`.
- Task 014: `GetInboxPRs` use case fetches PRs from all favorite repos in parallel via errgroup, tolerates partial failures. `I` key opens inbox from PR list. `OpenInboxPRMsg` switches repo context and loads PR detail. `PrioritySort` called on inbox data (review-requested > CI-failing > stale > normal). Usecase coverage stays at 100%.
- Task 021: Visual selection mode via `v` key. Space toggles selection (`●`/`○`), `a` selects all visible, Esc cancels. Batch operations: `y` copies all selected PR URLs (newline-separated), `o` opens all in browser. Status bar shows `N selected` count. Cursor uses `▸`/`▹` in selection mode. Quick filters m/n disabled during selection mode.
- Task 023: Cache package at `internal/cache/` with JSON file cache in `XDG_CACHE_HOME/vivecaka/repos/{owner}_{name}.json`. Atomic writes via temp file rename. `loadCachedPRsCmd` fires alongside fresh API load; cached data displayed immediately if still loading. `saveCacheCmd` runs as fire-and-forget after fresh load. `IsStale` checks TTL. 79.3% coverage.
- Task 029: Debug logging via `log/slog` to `XDG_STATE_HOME/vivecaka/debug.log`. Activated via `--debug` flag, `VIVECAKA_DEBUG=1` env var, or `debug = true` in config. Log rotation at 10MB. No-op logger when disabled. 91.3% coverage.
- Task 024: Keybinding overrides wired via `ApplyOverrides(map[string]string)` on KeyMap. Config `[keybindings]` section parsed and applied on startup. Supports all binding names (quit, search, filter, etc.). Notification config struct already in place. Adaptive colors deferred (current hex colors work on all terminals).
- Task 025: Per-repo state persistence via `internal/cache/state.go`. RepoState stores last sort, sort direction, filter opts, and last-viewed PR timestamps. State saved to `XDG_DATA_HOME/vivecaka/state/{owner}_{name}.json`. Filter/sort restored on startup. PR viewed timestamps tracked on detail open. IsUnread checks `updatedAt > lastViewed`. Cache coverage 80.0%.

Open Issues:
- ~~PR detail loading spinner appears stuck~~ **RESOLVED** (Feb 5, 2026): Spinner now animates correctly. Root causes addressed: (1) Fixed View() logic to only show loading state when `loading=true`, not when `detail==nil`; (2) Added explicit `return nil` for `PRDetailLoadedMsg` in Update(); (3) Verified spinner frames cycle properly via iTerm2 automation tests. The `gh pr checks` API call takes ~1.4s which causes visible spinner animation before PR detail loads.

## Recommended Execution Order

Independent tasks (can be done in any order): 001, 002, 004, 005, 006, 008, 011, 012, 013, 015, 016, 017, 019, 021, 022, 023, 024, 026

Dependency chains:
- 002 → 003 (browser integration needed before detail key bindings use it)
- 006 → 007 (username detection before filter panel)
- 006 + 013 → 014 (username + favorites before inbox)
- 008 → 009 (glamour before chroma)
- 008 → 010 → 018 (markdown → comments → inline comments)
- 019 → 020 (two-pane before side-by-side)
- 001 + 007 + 023 → 025 (sort + filter + cache before persistence)
- 026 → 027 (testify + fixtures before integration tests)

---

## Phases 0-13: Original Scaffolding (Complete)

> These phases built the project structure, domain layer, adapters, use cases, and TUI scaffolding. An audit found that while the architecture is sound, many features were only partially implemented (views built as rendering shells, messages emitted but not handled, config fields defined but not read).

### Phase 0: Project Bootstrap — COMPLETE

| Task | Description | Status | Notes |
|------|-------------|--------|-------|
| T0.1 | Initialize Go module & project structure | ✅ Done | All packages compile |
| T0.2 | Create Makefile | ✅ Done | 17 targets |
| T0.3 | Create CLAUDE.md | ✅ Done | |
| T0.4 | Create PROGRESS.md | ✅ Done | This file |
| T0.5 | Setup lefthook | ✅ Done | |
| T0.6 | Setup golangci-lint | ✅ Done | v2 format |
| T0.7 | Setup GoReleaser | ✅ Done | v2 format, clean check |
| T0.8 | LICENSE + README | ✅ Done | |

### Phase 1: Domain Layer — COMPLETE

| Task | Description | Status | Notes |
|------|-------------|--------|-------|
| T1.1 | Define domain entities | ✅ Done | 100% coverage |
| T1.2 | Define domain interfaces | ✅ Done | PRReader, PRReviewer, PRWriter |
| T1.3 | Define domain errors | ✅ Done | errors.Is/As tests pass |

### Phase 2: Plugin Infrastructure — COMPLETE

| Task | Description | Status | Notes |
|------|-------------|--------|-------|
| T2.1 | Plugin interfaces | ✅ Done | Plugin, ViewPlugin, KeyPlugin |
| T2.2 | Plugin registry | ✅ Done | 95.2% coverage, race-safe |
| T2.3 | Hook system | ✅ Done | On/Emit, error chain, race-safe |

### Phase 3: Configuration — COMPLETE

| Task | Description | Status | Notes |
|------|-------------|--------|-------|
| T3.1 | Config struct and TOML loading | ✅ Done | 94% coverage, validation |
| T3.2 | XDG path helpers | ✅ Done | Env var override tests |
| T3.3 | Default config generation | ✅ Done | Auto-create on first run |

### Phase 4: GH CLI Adapter — COMPLETE (coverage gap: 25.3%)

| Task | Description | Status | Notes |
|------|-------------|--------|-------|
| T4.1 | GH CLI plugin wrapper | ✅ Done | Auth detection |
| T4.2 | PRReader (GH CLI) | ✅ Done | ListPRs, GetPR, GetDiff, GetChecks, GetComments |
| T4.3 | PRReviewer (GH CLI) | ✅ Done | SubmitReview, AddComment, ResolveThread |
| T4.4 | PRWriter (GH CLI) | ✅ Done | Checkout, Merge, UpdateLabels |
| T4.5 | Diff parser | ✅ Done | Unified diff, renames, line numbers |

### Phase 5: Use Cases — COMPLETE

| Task | Description | Status | Notes |
|------|-------------|--------|-------|
| T5.1 | ListPRs use case | ✅ Done | 100% coverage |
| T5.2 | GetPRDetail use case | ✅ Done | errgroup parallel fetch |
| T5.3 | ReviewPR use case | ✅ Done | Validation + delegation |
| T5.4 | CheckoutPR use case | ✅ Done | |
| T5.5 | Comment use cases | ✅ Done | AddComment + ResolveThread |

### Phase 6: TUI Foundation — COMPLETE

| Task | Description | Status | Notes |
|------|-------------|--------|-------|
| T6.1 | Theme system | ✅ Done | 5 themes, Catppuccin Mocha default |
| T6.2 | Keymap system | ✅ Done | Global + per-view, help.KeyMap |
| T6.3 | Root app model | ✅ Done | View routing, message handling |
| T6.4 | Header component | ✅ Done | Repo + count |
| T6.5 | Status bar | ✅ Done | Context-aware key hints |
| T6.6 | Toast notifications | ✅ Done | Stacked, auto-dismiss, 4 levels |

### Phase 7: PR List View — PARTIAL (sort broken, clipboard/browser stubbed)

| Task | Description | Status | Notes |
|------|-------------|--------|-------|
| T7.1 | PR list table | ✅ Done | Columns render correctly |
| T7.2 | Search/filter | ⚠️ Partial | Substring match only, not fuzzy. No filter panel. |
| T7.3 | Sort cycling | ⚠️ Broken | Cycles field name but NEVER RE-SORTS data → Task 001 |
| T7.4 | Branch highlight | ⚠️ Partial | Only after checkout, no startup detection → Task 016 |
| T7.5 | Draft dimming | ✅ Done | [DRAFT] prefix, muted style |
| T7.6 | Clipboard/browser | ⚠️ Stubbed | Messages emitted but silently dropped → Task 002 |

### Phase 8: PR Detail View — PARTIAL (missing keys, no markdown)

| Task | Description | Status | Notes |
|------|-------------|--------|-------|
| T8.1 | Detail model | ✅ Done | 4-pane tabbed layout |
| T8.2 | Info pane | ⚠️ Partial | Raw text body, no markdown → Task 008 |
| T8.3 | Files pane | ✅ Done | |
| T8.4 | Checks pane | ⚠️ Partial | No CI summary line, no o key → Task 003 |
| T8.5 | Comments pane | ⚠️ Partial | Read-only, no collapse/reply/resolve → Task 010 |

### Phase 9: Diff Viewer — PARTIAL (search broken, many features missing)

| Task | Description | Status | Notes |
|------|-------------|--------|-------|
| T9.1 | Diff model | ⚠️ Partial | No two-pane layout → Task 019 |
| T9.2 | Line rendering | ⚠️ Partial | No syntax highlighting → Task 009 |
| T9.3 | File navigation | ⚠️ Partial | Tab only, no {/} or hunk nav → Task 005 |
| T9.4 | Search in diff | ⚠️ Broken | Accepts input but zero effect → Task 004 |

### Phase 10: Review Forms — PARTIAL (no huh, single-line editor)

| Task | Description | Status | Notes |
|------|-------------|--------|-------|
| T10.1 | Review model | ⚠️ Partial | No huh library, bare-bones → Task 011 |
| T10.2 | Action selector | ✅ Done | |
| T10.3 | Body editor | ⚠️ Partial | Single-line append-only → Task 011 |

### Phase 11: Repo Switching — PARTIAL (empty switcher, dead inbox)

| Task | Description | Status | Notes |
|------|-------------|--------|-------|
| T11.1 | Repo switcher overlay | ⚠️ Stubbed | Opens but empty — favorites never loaded → Task 013 |
| T11.2 | Unified PR Inbox (S4) | ⚠️ Stubbed | UI built, no data pipeline, no key binding → Task 014 |
| T11.3 | Repo auto-detection | ✅ Done | SSH + HTTPS parsing works |

### Phase 12: Help System — PARTIAL (tutorial dead code)

| Task | Description | Status | Notes |
|------|-------------|--------|-------|
| T12.1 | Help overlay | ✅ Done | Context-aware per-view |
| T12.2 | First-launch tutorial | ⚠️ Dead code | Code exists, Show() never called → Task 016 |
| T12.3 | Status bar hints | ✅ Done | |

### Phase 13: Integration & Polish — PARTIAL (shallow tests, no integration)

| Task | Description | Status | Notes |
|------|-------------|--------|-------|
| T13.1 | App tests + coverage | ⚠️ Partial | Tests shallow, no integration tests → Tasks 026, 027 |
| T13.2 | Progress tracking | ✅ Done | This file (now corrected) |

## Coverage Summary

| Package | Coverage | Target | Notes |
|---------|----------|--------|-------|
| domain | 100% | 100% | ✅ Met |
| usecase | 100% | 90%+ | ✅ Met |
| plugin | 95.2% | — | ✅ Good |
| tui | 95.0% | — | ⚠️ Tests are shallow |
| config | 94.0% | — | ✅ Good |
| views | 90.8% | — | ⚠️ Tests mostly check "doesn't panic" |
| ghcli | 25.3% | 80%+ | ❌ Only parser tested → Task 026 |

## Notes

- Go 1.25.7 on darwin/arm64
- BubbleTea v1.3.10, LipGloss v1.1.1, Bubbles v0.21.1
- go-gh v2.13.0
- testify listed as dep but never imported → Task 026
- huh listed as dep but never imported → Task 011
- 0 lint issues, all tests pass with -race
- Task 001: PR list sort now reorders data with ▲/▼ indicator; added error handling for tutorial flag write to satisfy errcheck.
- Task 002: added platform helpers for clipboard/browser; wired copy/open handling and detail view open key.
- Task 001/002 task files updated to DONE status.
- Task 003: PR detail d/c/o keys wired; checks summary line rendered; live GH checks in peek-it returned empty ("No CI checks"), summary covered by unit test; mock_prdetail verified via agent-browser.
- Task 004: diff search now scans/highlights matches with n/N navigation and count; mock_diff verified via agent-browser.
- Task 005: diff navigation adds hunk/file jumps, gg/G, and collapse/expand; mock_diff verified via agent-browser.
- Task 006: quick filters added (m/n) with user detection via gh; filter label shows in header; mock_prlist verified via agent-browser.
