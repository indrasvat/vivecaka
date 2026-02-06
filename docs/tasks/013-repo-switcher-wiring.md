# Task 013: Repo Switcher Wiring

## Status: TODO

## Problem

Ctrl+R opens the repo switcher overlay, but the repo list is **never populated**. `SetRepos()` is never called. The switcher renders an empty list. Favorites from config are never read or passed to the switcher. Last-used repo is not persisted. The modal itself is visually plain compared to the rest of the app.

## Expanded Scope

This task goes beyond basic wiring to deliver a polished, full-featured repo switcher:

1. **Three-source repo population** — favorites (config), user's repos (`gh repo list`), manual entry
2. **In-TUI favorite management** — `s` key toggles star, writes back to `config.toml`
3. **Aesthetic modal redesign** — upgraded visuals consistent with app theme
4. **Scrollable list** — dynamic height up to 50% terminal, j/k scrolling with indicators
5. **Fast fuzzy search** — dual-purpose: filter existing repos + add new `owner/repo`

## PRD Reference

Read `docs/PRD.md` sections:
- Section 7.7 (Repo Switching) — favorites, fuzzy search, current repo highlight
- F9 (Multi-Repo Favorites) — pinned repos, last-used memory
- S1 (Smart Context Awareness) — repo persistence
- Config section — `[repos] favorites` field

## Visual Design

```
╭──── ⟡ Switch Repository ────────────────╮
│                                          │
│   ❯ acme/fro▎                            │
│                                          │
│   ╶── FAVORITES ──────────────────────╴  │
│   ▸ ★  acme / frontend          ● 12    │
│     ★  acme / backend              8    │
│     ★  someuser / cool-lib         3    │
│                                          │
│   ╶── YOUR REPOS ─────────────────────╴  │
│        indrasvat / vivecaka        0    │
│        indrasvat / dotfiles             │
│        indrasvat / blog            2    │
│                                     ▼   │
│                                          │
│   s star   ↵ switch   Esc close          │
╰──────────────────────────────────────────╯
```

### Visual Element Spec

| Element | Treatment | Theme Field |
|---------|-----------|-------------|
| Title icon `⟡` | Decorative symbol | `Secondary` |
| Title text "Switch Repository" | Bold | `Primary` |
| Search prompt `❯` | Bold | `Primary` |
| Search input text | Normal | `Fg` |
| Cursor `▎` | Blinking-style char | `Primary` |
| Section headers `╶── FAVORITES ──╴` | Ruled lines + label | line=`Muted`, label=`Subtext` |
| Owner text (e.g. `acme`) | Dimmed | `Subtext` |
| Separator `/` | Dimmed | `Muted` |
| Repo name (e.g. `frontend`) | Bold | `Fg` |
| Cursor indicator `▸` | On selected row | `Primary` |
| Selected row background | Full-width band | `BgDim` |
| Star `★` | Favorite indicator | `Warning` (gold) |
| Current dot `●` | Current repo marker | `Success` (green) |
| PR count | Right-aligned | `Muted` |
| Scroll indicators `▲`/`▼` | When list overflows | `Muted` |
| Ghost add entry `+ add owner/repo` | When search has no match and looks like owner/repo | `Muted` italic |
| Key hints | Bottom bar | keys=`Info`, labels=`Muted` |
| Border | Rounded, prominent | `Primary` (not default `Border`) |

### Layout Rules

- Box width: `max(40, min(60, termWidth - 4))`
- Inner width: `boxWidth - 4` (border + padding)
- Max visible list rows: `min(listLen, max(5, termHeight/2 - 8))` (cap at ~50% terminal height)
- Search field pinned at top, key hints pinned at bottom — only repo list scrolls between them
- Non-selected rows: no background (modal's implicit bg)
- Selected row: `BgDim` background band across full `innerWidth`
- Section headers scroll with the list content (not pinned)
- Each line must be full-width spaces (never empty `""`) — critical for `ansi.Cut()` overlays
- Account for borders in all width calculations
- Truncate repo names aggressively — never let text auto-wrap

## Three-Source Data Model

### Source 1: Favorites (instant)
- Read from `config.toml` `[repos] favorites = ["owner/repo", ...]` at startup
- Shown in `FAVORITES` section with `★` icon
- CWD-detected repo auto-prepended if not already in favorites
- Deduped against other sources

### Source 2: User's Repos (lazy, cached)
- Fetched via `gh repo list --json nameWithOwner -L 20` on **first** Ctrl+R open
- Cached for session duration (no repeated API calls)
- Shown in `YOUR REPOS` section, without `★`
- Deduped against favorites (don't show a repo in both sections)
- Fetch is async — modal opens instantly with favorites; discovered repos trickle in via `tea.Cmd`

### Source 3: Manual Entry (on-demand)
- When search query looks like `owner/repo` and matches nothing in the list
- Show ghost entry: `+ add owner/repo` at bottom in `Muted` italic
- Enter on ghost entry → async validate via `gh repo view owner/repo --json name` (with spinner)
- On success: switch to repo + auto-add to favorites + write config
- On failure: show inline error "Repository not found" in `Error` color

## Favorite Toggle (`s` key)

- Press `s` on any repo entry → toggle its favorite status
- If unfavorited → add to `[repos] favorites` in config, move entry to FAVORITES section, show `★`
- If favorited → remove from `[repos] favorites` in config, move entry to YOUR REPOS section (or remove if not in discovered list), hide `★`
- Config write-back: update the `favorites` array in `config.toml` immediately
- Requires a `config.UpdateFavorites([]string)` or `config.Save()` method

## Fuzzy Search

- Search field at top: `❯ query▎`
- Filters across **both** sections simultaneously
- Uses existing `fuzzyMatch()` — all query chars must appear in order in `owner/repo` string
- Matching is case-insensitive
- Results update on every keystroke (no debounce needed — local data, fast)
- Cursor resets to first match on each filter change
- If no match and query contains `/`, show ghost add entry
- Section headers hidden when their section has 0 visible entries after filtering

## Files to Modify

| File | Changes |
|------|---------|
| `internal/tui/views/reposwitcher.go` | Major rewrite: aesthetic View(), scrolling, sections, `s` toggle, ghost add entry, async repo discovery |
| `internal/tui/views/reposwitcher_test.go` | New: comprehensive tests for all behaviors |
| `internal/tui/app.go` | On init: parse favorites, set repos, set current. On switch: update current. Handle async discover/validate msgs |
| `internal/config/config.go` | Add `UpdateFavorites()` method to write favorites back to TOML |
| `internal/config/config_test.go` | Test `UpdateFavorites()` round-trip |
| `internal/adapter/ghcli/repodetect.go` | Add `ListUserRepos(ctx) ([]domain.RepoRef, error)` via `gh repo list` |
| `internal/adapter/ghcli/repodetect_test.go` | Test `ListUserRepos` with fixture data |

## Execution Steps

1. Read `CLAUDE.md` — all conventions and gotchas
2. Read relevant PRD sections (7.7, F9, S1, Config)
3. Read existing files: `reposwitcher.go`, `app.go`, `config.go`, `repodetect.go`
4. **Config write-back**: add `UpdateFavorites(favorites []string) error` to config package
5. **Adapter**: add `ListUserRepos(ctx) ([]domain.RepoRef, error)` to ghcli adapter
6. **Rewrite `reposwitcher.go`**:
   - New `RepoEntry` with `Section` field (favorites/discovered/ghost)
   - Sectioned list rendering with visual design spec above
   - Scrollable viewport with `▲`/`▼` indicators
   - `s` key handler → toggle favorite → emit `ToggleFavoriteMsg`
   - Ghost add entry logic when search looks like `owner/repo`
   - Full-width padding lines (never empty strings — see CLAUDE.md)
   - Explicit ANSI resets between styled elements
   - Width calculations accounting for borders
7. **Wire in `app.go`**:
   - On init: parse `cfg.Repos.Favorites` → `[]RepoEntry`, prepend CWD repo, call `SetRepos()`
   - On Ctrl+R first open: fire async `ListUserRepos` cmd
   - Handle `ReposDiscoveredMsg` → merge into switcher, dedup against favorites
   - Handle `ToggleFavoriteMsg` → update config via `UpdateFavorites()`
   - Handle ghost entry validation → `gh repo view` async → switch or show error
   - On repo switch: update `SetCurrentRepo()`, reload PRs
8. **Tests**: unit tests for switcher model (Update + View), config write-back, adapter
9. **`make ci`** — must pass clean

## Critical Gotchas (from CLAUDE.md)

- **Full-width padding**: every line must be `strings.Repeat(" ", width)`, never `""` — breaks `ansi.Cut()` overlays
- **ANSI resets**: add `\033[0m` between styled elements to prevent style bleeding
- **Border width**: subtract border + padding from content width calculations
- **No `lipgloss.Height(n)`**: use `MaxHeight` + manual padding instead
- **No `switch msg := msg.(type)`**: use `typedMsg` + reassign to avoid shadowing
- **Async pattern**: modal opens instantly with favorites; discovered repos arrive via message — don't block on API call
- **Async action UX**: for ghost entry validation, keep dialog visible through loading → result states (like checkout dialog)
- **`gh` output quirks**: verify stdout vs stderr for any `gh` subcommands used
- **Truncate aggressively**: never let text auto-wrap in bordered panels

## Verification

### Functional
```bash
go test -race -v ./internal/tui/views/... -run TestRepoSwitch
go test -race -v ./internal/config/... -run TestUpdateFavorites
go test -race -v ./internal/adapter/ghcli/... -run TestListUserRepos
make ci
```

### Visual (iterm2-driver) — Comprehensive Screenshot Suite

Create and run an iterm2-driver script that captures **all** of the following states:

#### 1. Modal appearance & layout
- Set up config with 3-4 favorite repos in `~/.config/vivecaka/config.toml`
- Launch `bin/vivecaka`, wait for PR list to load
- Press Ctrl+R → **screenshot**: modal appears with FAVORITES section populated, current repo has `●`, stars are gold, border is Primary color, section headers visible

#### 2. Discovered repos loading
- **screenshot**: after discovered repos load, YOUR REPOS section appears below favorites
- Verify deduplication: repos in favorites don't appear again in YOUR REPOS

#### 3. Cursor navigation
- Press j/k to move cursor through list → **screenshot** at each section boundary
- Verify `▸` cursor indicator moves, selected row gets BgDim background
- Verify cursor wraps or stops at list boundaries

#### 4. Scrolling (if enough repos)
- If list exceeds modal height → **screenshot** showing `▼` scroll indicator at bottom
- Scroll down → **screenshot** showing `▲` indicator at top
- Verify section headers scroll with content

#### 5. Fuzzy search filtering
- Type partial repo name (e.g., "front") → **screenshot**: only matching repos shown, non-matching hidden
- Verify section headers hidden when section has 0 matches
- Clear search → **screenshot**: full list restored

#### 6. Favorite toggle
- Move cursor to a YOUR REPOS entry, press `s` → **screenshot**: repo moves to FAVORITES with `★`
- Move cursor to a FAVORITES entry, press `s` → **screenshot**: repo loses `★`, moves to YOUR REPOS
- Verify config.toml updated on disk after toggle

#### 7. Ghost add entry
- Type an `owner/repo` string that doesn't match anything → **screenshot**: `+ add owner/repo` ghost entry visible in Muted italic
- (Optional) Press Enter on ghost entry → **screenshot**: validation spinner → result

#### 8. Repo switching
- Select a different repo, press Enter → **screenshot**: modal closes, PR list reloads for new repo
- Verify header shows new repo name
- Press Ctrl+R again → **screenshot**: previous repo still in list, new repo now has `●` current marker

#### 9. Owner/repo text styling
- **screenshot** close-up: verify owner text is dimmed (Subtext), `/` is Muted, repo name is Fg bold

#### 10. Empty state
- Remove all favorites from config, restart → **screenshot**: modal shows only CWD repo + YOUR REPOS from discovery
- If no repos at all → verify graceful empty state message

## Commit

```
feat(tui): wire repo switcher with favorites, discovery, and aesthetic redesign

Three-source repo population: favorites from config.toml, user repos
via `gh repo list` (lazy, cached), and manual owner/repo entry.
In-TUI favorite toggle with `s` key writes back to config. Redesigned
modal with sectioned list, fuzzy search, scroll support, and polished
visuals using theme colors throughout.
```

## Session Protocol

1. Read `CLAUDE.md`
2. Read this task file
3. Read relevant PRD sections
4. Execute (steps 4-8 above)
5. Verify — `make ci` + iterm2-driver visual QA (ALL 10 screenshot scenarios)
6. Update `docs/PROGRESS.md` — mark this task DONE with notes
7. Update this file — set `Status: DONE`
8. `git add` changed files + `git commit`
9. Move to next task
