# Task 030: Smart Checkout — Context-Aware PR Checkout with Managed Clones

## Status: DONE

## Problem

vivecaka allows launching from anywhere and browsing any GitHub repo via the repo switcher.
However, `gh pr checkout` requires a local git clone. This creates three failure modes:

1. **No git repo at all**: User launches from `/tmp/demo`, switches to `steipete/CodexBar`, presses `c` (checkout).
   Fails with: `fatal: not a git repository (or any of the parent directories): .git`

2. **Wrong repo**: User is in `~/code/indrasvat/vivecaka` (repo A), switches to `steipete/CodexBar` (repo B),
   presses `c`. The `gh pr checkout --repo steipete/CodexBar` runs inside repo A's working tree — either fails
   or silently creates a foreign branch in the wrong repo.

3. **Right repo, wrong branch**: User is in the correct repo but on an important branch and doesn't want to
   disrupt it. Current checkout silently switches the branch.

All three cases currently produce confusing errors or dangerous silent behavior.

## Solution: Three-Mechanism Smart Checkout (Zero Config)

### Mechanism 1: Known-Repos Registry (Auto-Learned)

Every time vivecaka launches in a directory that IS a git repo, it records the mapping:

```
steipete/CodexBar → /Users/indra/code/steipete/CodexBar
```

Stored at `~/.local/share/vivecaka/known-repos.json` (XDG data dir, `config.DataDir()`).
No manual configuration needed — vivecaka learns repo locations organically over time.

### Mechanism 2: Managed Clone Cache

When no known path exists for a repo, vivecaka offers to clone into its managed cache:

```
~/.cache/vivecaka/clones/steipete/CodexBar/
```

XDG cache dir (`config.CacheDir()`) + `clones/` + `owner/name/`. First checkout is slow (full clone),
subsequent checkouts of PRs in the same repo are fast (fetch + branch switch). After cloning, the
path is registered in known-repos, so future checkouts skip the dialog entirely.

### Mechanism 3: Git Worktrees (Optional, Right-Repo Case)

When the user IS in the correct repo's clone, offer the choice to create a worktree instead of
switching the current branch. This keeps the current branch intact while checking out the PR
in a parallel working tree.

## PRD Reference

Read `docs/PRD.md` sections:
- Section 7.1 (PR List View) — checkout action
- Section 7.7 (Repo Switching) — multi-repo context
- Section 7.5 (Review Forms) — confirmation dialogs
- F9 (Multi-Repo Favorites) — cross-repo UX

## Checkout Decision Cascade

When user presses `c`:

```
1. Detect CWD repo identity (already done at startup via detectRepoCmd)
2. Compare a.repo (browsing target) against CWD repo:

   a.repo == CWD repo?
   ├── YES → Show normal checkout dialog (existing behavior)
   │         WITH additional worktree option (Mechanism 3)
   │
   └── NO → Look up a.repo in known-repos registry
            ├── FOUND (path exists on disk + is correct repo) →
            │   Show "Checkout at known path" confirmation dialog
            │   Execute checkout with CWD override to that path
            │
            ├── FOUND (but path no longer exists/valid) →
            │   Remove stale entry, fall through to NOT FOUND
            │
            └── NOT FOUND →
                Show "Checkout Options" dialog:
                  ▸ Clone to vivecaka cache
                    Clone to custom path...
                    Open on GitHub
                    [Esc] Cancel
```

## ASCII Mocks — All Dialog States

### Mock A: Normal Checkout (CWD = correct repo, existing behavior preserved)

```
╭────────────────────────────────────────────────╮
│                                                │
│  Checkout Branch                               │
│                                                │
│  Check out branch "feat/oauth" for PR #255?    │
│                                                │
│  Enter/y Yes   Esc/n No                        │
│                                                │
╰────────────────────────────────────────────────╯
```

Theme: title=Primary+Bold, message=Fg, confirm keys=Success+Bold, cancel keys=Error+Bold.
Border=Primary (rounded). This is the EXISTING confirm dialog, unchanged.

### Mock B: Checkout with Worktree Option (CWD = correct repo)

```
╭────────────────────────────────────────────────╮
│                                                │
│  Checkout PR #255                              │
│  feat/oauth                                    │
│                                                │
│  ▸ Switch branch                               │
│    Replaces current branch (main)              │
│                                                │
│    New worktree                                │
│    .worktrees/pr-255-feat-oauth                │
│    Keeps current branch intact                 │
│                                                │
│  ↑/↓ select   Enter confirm   Esc cancel       │
│                                                │
╰────────────────────────────────────────────────╯
```

Theme: title=Primary+Bold, branch name=Info, `▸` cursor=Primary, selected option label=Fg+Bold,
unselected option label=Fg, description=Subtext, worktree path=Muted+Italic, key hints at bottom.
Border=Primary (rounded).

### Mock C: Checkout at Known Path (CWD != browsed repo, known-repos has entry)

```
╭────────────────────────────────────────────────╮
│                                                │
│  Checkout PR #255                              │
│  steipete/CodexBar → feat/oauth                │
│                                                │
│  Will check out in:                            │
│  ~/code/steipete/CodexBar                      │
│                                                │
│  Enter confirm   Esc cancel                    │
│                                                │
╰────────────────────────────────────────────────╯
```

Theme: title=Primary+Bold, repo=Info, arrow `→`=Muted, branch=Info,
path=Warning+Bold (stands out — user MUST see where checkout will happen),
key hints=Success/Error+Bold. Border=Primary (rounded).

### Mock D: Checkout Options (no known path, no local clone)

```
╭────────────────────────────────────────────────╮
│                                                │
│  ⚠ No local clone found                        │
│                                                │
│  steipete/CodexBar is not cloned locally.      │
│                                                │
│  ▸ Clone to vivecaka cache                     │
│    ~/.cache/vivecaka/clones/steipete/CodexBar  │
│                                                │
│    Clone to custom path...                     │
│    Enter a directory path                      │
│                                                │
│    Open on GitHub                              │
│    View PR #255 in browser                     │
│                                                │
│  ↑/↓ select   Enter confirm   Esc cancel       │
│                                                │
╰────────────────────────────────────────────────╯
```

Theme: `⚠`=Warning+Bold, title=Warning+Bold (not Primary — this is a warning state),
repo name=Info, `▸` cursor=Primary, selected option=Fg+Bold, description=Subtext,
path=Muted+Italic. Border=Warning (rounded, not Primary — signals caution).

### Mock E: Clone in Progress (after selecting clone option)

```
╭────────────────────────────────────────────────╮
│                                                │
│  Cloning Repository                            │
│                                                │
│  ⠹ Cloning steipete/CodexBar...               │
│    ~/.cache/vivecaka/clones/steipete/CodexBar  │
│                                                │
╰────────────────────────────────────────────────╯
```

Theme: title=Primary+Bold, spinner=Primary+Bold, clone message=Fg,
path=Muted+Italic. Border=Primary. Keys disabled during clone.

### Mock F: Clone Done + Checkout in Progress

```
╭────────────────────────────────────────────────╮
│                                                │
│  Checking Out                                  │
│                                                │
│  ⠹ Checking out "feat/oauth" for PR #255...   │
│    in ~/.cache/vivecaka/clones/steipete/       │
│    CodexBar                                    │
│                                                │
╰────────────────────────────────────────────────╯
```

Same theme as loading states in existing ConfirmModel.

### Mock G: Success with Copyable Path (checkout was NOT in CWD)

```
╭────────────────────────────────────────────────╮
│                                                │
│  ✓ Checkout Complete                           │
│                                                │
│  Branch: feat/oauth                            │
│  Path:   ~/.cache/vivecaka/clones/steipete/    │
│          CodexBar                              │
│                                                │
│  cd ~/.cache/vivecaka/clones/steipete/CodexBar │
│                                                │
│  y copy cd command   any key dismiss           │
│                                                │
╰────────────────────────────────────────────────╯
```

Theme: `✓`=Success+Bold, title=Success+Bold, "Branch:" label=Subtext, branch value=Info,
"Path:" label=Subtext, path value=Warning+Bold (prominent),
cd command=Fg on BgDim (monospace block, visually distinct),
`y` key=Info+Bold, "any key"=Muted. Border=Success (rounded).

### Mock H: Success (checkout WAS in CWD — existing behavior, enhanced)

```
╭────────────────────────────────────────────────╮
│                                                │
│  ✓ Checkout Complete                           │
│                                                │
│  Checked out branch: feat/oauth                │
│                                                │
│  Press any key to continue                     │
│                                                │
╰────────────────────────────────────────────────╯
```

This is the EXISTING success dialog, unchanged.

### Mock I: Clone to Custom Path (text input)

```
╭────────────────────────────────────────────────╮
│                                                │
│  Clone steipete/CodexBar                       │
│                                                │
│  Enter path:                                   │
│  ❯ ~/code/steipete/CodexBar▎                   │
│                                                │
│  Enter confirm   Esc cancel                    │
│                                                │
╰────────────────────────────────────────────────╯
```

Theme: title=Primary+Bold, "Enter path:" label=Subtext, `❯` prompt=Primary,
input text=Fg, cursor=Primary. Border=Primary (rounded).
Default value pre-filled with `~/code/{owner}/{repo}` as a sensible suggestion.

## Architecture & Layering

**IMPORTANT**: This feature follows vivecaka's plugin-based architecture. New capabilities
are exposed as domain interfaces, implemented in adapters, discoverable via the plugin registry,
and wired through functional options — the same pattern as `PRReader`, `PRReviewer`, `PRWriter`.

### New Domain Interface: `RepoManager`

`CloneRepo`, `CheckoutAt`, and `CreateWorktree` are **repo-level git operations**, NOT PR operations.
They do NOT belong on `PRWriter` (which is for PR writes: checkout-by-number, merge, labels).
Instead, create a new domain interface:

```go
// internal/domain/interfaces.go

// RepoManager provides local git repository management capabilities.
// Implemented by adapters that can perform git/clone operations.
type RepoManager interface {
    // CheckoutAt checks out a PR branch in the specified working directory.
    // If workDir is "", uses the process CWD (same as PRWriter.Checkout).
    CheckoutAt(ctx context.Context, repo RepoRef, number int, workDir string) (branch string, err error)
    // CloneRepo clones a repository to the specified local path.
    CloneRepo(ctx context.Context, repo RepoRef, targetPath string) error
    // CreateWorktree creates a git worktree for a branch at the given path.
    CreateWorktree(ctx context.Context, repoPath, branch, worktreePath string) error
}
```

`PRWriter` stays UNCHANGED. `PRWriter.Checkout()` remains the simple CWD-only checkout
used by the existing confirm dialog path. `RepoManager` is the new capability for smart checkout.

### New Domain Type

```go
// internal/domain/pr.go (or new file domain/repo.go)

// RepoLocation tracks where a repo is cloned locally.
type RepoLocation struct {
    Repo     RepoRef   `json:"repo"`
    Path     string    `json:"path"`
    LastSeen time.Time `json:"last_seen"`
    Source   string    `json:"source"` // "detected", "cloned", "manual"
}
```

### Plugin Registry Extension

Add `RepoManager` capability discovery to the plugin registry, following the existing pattern:

```go
// internal/plugin/registry.go

type Registry struct {
    mu           sync.RWMutex
    plugins      map[string]Plugin
    readers      []domain.PRReader
    reviewers    []domain.PRReviewer
    writers      []domain.PRWriter
    repoManagers []domain.RepoManager  // NEW
    views        []ViewRegistration
    keys         []KeyRegistration
    hooks        *HookManager
}

// In Register():
if rm, ok := p.(domain.RepoManager); ok {
    r.repoManagers = append(r.repoManagers, rm)
}

// GetRepoManagers returns all registered RepoManager implementations.
func (r *Registry) GetRepoManagers() []domain.RepoManager {
    r.mu.RLock()
    defer r.mu.RUnlock()
    return r.repoManagers
}
```

### Adapter Implementation

`ghcli.Adapter` implements `domain.RepoManager` (it already implements `PRReader`,
`PRReviewer`, `PRWriter`). Methods go in a new file `internal/adapter/ghcli/repomanager.go`
to keep writer.go focused on PR writes:

```go
// internal/adapter/ghcli/repomanager.go

func (a *Adapter) CheckoutAt(ctx context.Context, repo domain.RepoRef, number int, workDir string) (string, error)
func (a *Adapter) CloneRepo(ctx context.Context, repo domain.RepoRef, targetPath string) error
func (a *Adapter) CreateWorktree(ctx context.Context, repoPath, branch, worktreePath string) error
```

`CheckoutAt` uses `cmd.Dir = workDir` to set the working directory before executing.
When `workDir == ""`, it behaves identically to the existing `Checkout()` method.
`CloneRepo` runs `gh repo clone owner/name targetPath`. Ensures parent dir exists via `os.MkdirAll`.
`CreateWorktree` runs `git worktree add <path> <branch>` with `cmd.Dir = repoPath`.

The existing `PRWriter.Checkout()` in `writer.go` is NOT changed — backward compatibility preserved.

### App Wiring (Functional Options)

```go
// internal/tui/app.go

// New functional option — same pattern as WithReader/WithWriter/WithReviewer
func WithRepoManager(rm domain.RepoManager) Option {
    return func(a *App) { a.repoManager = rm }
}
```

```go
// cmd/vivecaka/main.go

adapter := ghcli.New()
app := tui.New(cfg,
    tui.WithVersion(version),
    tui.WithReader(adapter),
    tui.WithReviewer(adapter),
    tui.WithWriter(adapter),
    tui.WithRepoManager(adapter),   // NEW — same adapter, new capability
)
```

### New Infrastructure Package: `internal/repolocator/`

Manages the known-repos registry. Does NOT import BubbleTea types.
Depends only on `domain` and `config` (for XDG paths).
This is infrastructure like `internal/cache/`, NOT a plugin or adapter.

```go
// internal/repolocator/locator.go

type Locator struct {
    dataPath string // ~/.local/share/vivecaka/known-repos.json
}

func New() *Locator
func (l *Locator) Lookup(repo domain.RepoRef) (string, bool)     // Find known path
func (l *Locator) Register(repo domain.RepoRef, path, source string) error  // Save mapping
func (l *Locator) Remove(repo domain.RepoRef) error              // Remove stale entry
func (l *Locator) Validate(repo domain.RepoRef) (string, bool)   // Lookup + verify path still valid
func (l *Locator) All() ([]domain.RepoLocation, error)           // List all known repos
func (l *Locator) CacheClonePath(repo domain.RepoRef) string     // Deterministic cache path
```

Storage: JSON file at `config.DataDir() + "/known-repos.json"`. Atomic writes (tmp + rename),
same pattern as `cache.Save()`.

### New Use Case: `internal/usecase/smart_checkout.go`

Orchestrates the decision cascade. Depends on `domain.RepoManager` (NOT `PRWriter`) and
`repolocator.Locator`. Does NOT import BubbleTea types — returns data, not commands.

```go
type SmartCheckout struct {
    repoMgr domain.RepoManager  // NOT PRWriter
    locator *repolocator.Locator
}

// CheckoutContext describes the current checkout situation.
type CheckoutContext struct {
    BrowsingRepo domain.RepoRef
    CWDRepo      domain.RepoRef // zero value if CWD is not a git repo
    CWDPath      string         // os.Getwd()
}

// CheckoutPlan describes what the app should do for checkout.
type CheckoutPlan struct {
    Strategy       CheckoutStrategy
    TargetPath     string // where checkout will happen
    KnownRepo      bool   // true if path came from known-repos
    CacheClonePath string // the managed cache path (for dialog display)
}

type CheckoutStrategy int
const (
    StrategyLocal      CheckoutStrategy = iota // CWD is correct repo
    StrategyKnownPath                          // Known-repos has a valid path
    StrategyNeedsClone                         // No local clone found
)

func (uc *SmartCheckout) Plan(ctx CheckoutContext) CheckoutPlan
func (uc *SmartCheckout) ExecuteClone(ctx context.Context, repo domain.RepoRef, targetPath string) error
func (uc *SmartCheckout) ExecuteCheckout(ctx context.Context, repo domain.RepoRef, number int, workDir string) (string, error)
func (uc *SmartCheckout) ExecuteWorktree(ctx context.Context, repo domain.RepoRef, number int, branch, basePath string) (string, error)
```

### Dependency Graph

```
tui.App
  → usecase.SmartCheckout
      → domain.RepoManager (interface)   ← ghcli.Adapter (implementation)
      → repolocator.Locator (infra)      → config.DataDir() (XDG paths)
  → usecase.CheckoutPR (UNCHANGED)
      → domain.PRWriter (interface)       ← ghcli.Adapter (implementation)

plugin.Registry
  → domain.RepoManager (auto-discovered via type assertion, same as PRReader/PRWriter/PRReviewer)
```

This follows the existing layering rule: `tui → usecase → domain ← adapter`.
`repolocator` is infrastructure (like `cache`), not a domain or adapter concern.
The plugin registry discovers `RepoManager` the same way it discovers `PRReader`.

### TUI Changes

**New view model**: `internal/tui/views/checkoutdialog.go` — multi-state checkout dialog.
Replaces the simple `ConfirmModel.Show()` call for checkout (confirm dialog still used for other actions).

States:
1. `checkoutWorktreeChoice` — Mock B (CWD = correct repo, choose switch vs worktree)
2. `checkoutKnownConfirm` — Mock C (known path confirmation)
3. `checkoutOptions` — Mock D (clone/path/browser options)
4. `checkoutCustomPath` — Mock I (text input for custom path)
5. `checkoutCloning` — Mock E (clone in progress)
6. `checkoutInProgress` — Mock F (checkout running)
7. `checkoutSuccess` — Mock G or H (result with optional copy)
8. `checkoutError` — existing error pattern (red border, ✗ icon)

**App changes** (`internal/tui/app.go`):
- Store `cwdRepo domain.RepoRef` and `cwdPath string` alongside `repo` (browsing target)
- On `CheckoutPRMsg`: call `smartCheckout.Plan()` to get strategy, then show the appropriate dialog
- New `handleSmartCheckout*` handlers for each dialog result message
- After successful clone: register path in known-repos via `repolocator`

**New tea.Cmd functions** (`internal/tui/commands.go`):
- `cloneRepoCmd(rm domain.RepoManager, repo domain.RepoRef, targetPath string) tea.Cmd`
- `checkoutAtCmd(rm domain.RepoManager, repo domain.RepoRef, number int, workDir string) tea.Cmd`
- `createWorktreeCmd(rm domain.RepoManager, repoPath, branch, worktreePath string) tea.Cmd`
- `copyToClipboardCmd(text string) tea.Cmd` (reuse existing clipboard infra from `platform.go`)

## Files to Create

| File | Purpose |
|------|---------|
| `internal/repolocator/locator.go` | Known-repos registry (Lookup, Register, Remove, Validate) |
| `internal/repolocator/locator_test.go` | Unit tests with temp dir isolation |
| `internal/usecase/smart_checkout.go` | Checkout planning via `domain.RepoManager` (decision cascade) |
| `internal/usecase/smart_checkout_test.go` | Unit tests with mock RepoManager + locator |
| `internal/tui/views/checkoutdialog.go` | Multi-state checkout dialog model |
| `internal/tui/views/checkoutdialog_test.go` | Unit tests for all dialog states/transitions |
| `internal/adapter/ghcli/repomanager.go` | `RepoManager` impl: CheckoutAt, CloneRepo, CreateWorktree |
| `internal/adapter/ghcli/repomanager_test.go` | Tests for RepoManager adapter methods |

## Files to Modify

| File | Changes |
|------|---------|
| `internal/domain/interfaces.go` | Add new `RepoManager` interface (`PRWriter` UNCHANGED) |
| `internal/domain/pr.go` | Add `RepoLocation` type |
| `internal/plugin/registry.go` | Add `repoManagers` slice, auto-discover `RepoManager`, add `GetRepoManagers()` |
| `internal/plugin/registry_test.go` | Test `RepoManager` discovery in registry |
| `internal/tui/app.go` | Add `WithRepoManager()` option, store `cwdRepo`/`cwdPath`, integrate SmartCheckout + new dialog |
| `internal/tui/commands.go` | Add `cloneRepoCmd`, `checkoutAtCmd`, `createWorktreeCmd` |
| `internal/tui/integration_test.go` | Integration tests for smart checkout flows |
| `cmd/vivecaka/main.go` | Add `tui.WithRepoManager(adapter)` to wiring |

## Execution Steps

### Step 1: Domain Interface & Types

1. Read `CLAUDE.md`
2. Add `RepoLocation` to `internal/domain/pr.go`
3. Add NEW `RepoManager` interface to `internal/domain/interfaces.go` — do NOT modify `PRWriter`
4. Update any mock implementations that need to satisfy the test harness

### Step 1b: Plugin Registry Extension

1. Add `repoManagers []domain.RepoManager` to `plugin.Registry` struct
2. Add type assertion in `Register()`: `if rm, ok := p.(domain.RepoManager); ok { ... }`
3. Add `GetRepoManagers() []domain.RepoManager` accessor
4. Add test in `registry_test.go` for RepoManager discovery

### Step 2: Repo Locator Package

1. Create `internal/repolocator/locator.go`:
   - JSON storage at `config.DataDir() + "/known-repos.json"`
   - `Lookup()`: find path for a repo, return `("", false)` if not found
   - `Register()`: add/update a repo→path mapping with timestamp and source tag
   - `Remove()`: delete a stale entry
   - `Validate()`: Lookup + verify the path still exists + has correct git remote
   - `All()`: list all entries (for future `vivecaka cache list` command)
   - `CacheClonePath()`: return deterministic `config.CacheDir() + "/clones/" + owner + "/" + name`
   - Atomic writes via temp file + rename (same pattern as `cache.Save()`)
2. Create `internal/repolocator/locator_test.go`:
   - Use `t.TempDir()` for isolation
   - Test Lookup miss, Register + Lookup hit, Remove, Validate with real git dir
   - Test atomic write (concurrent-safe)
   - Test `CacheClonePath` returns expected structure

### Step 3: Adapter — RepoManager Implementation

1. Create `internal/adapter/ghcli/repomanager.go` (NEW file — keep writer.go for PRWriter):
   - `CheckoutAt()`: like existing `Checkout()` but sets `cmd.Dir = workDir` on the `exec.Cmd`
     and on the post-checkout `git branch --show-current` command. If `workDir == ""`,
     falls back to CWD (identical to current behavior).
   - `CloneRepo()`: runs `gh repo clone owner/name targetPath`. Use `ghExec` with appropriate args.
     Ensure parent directory exists (`os.MkdirAll`).
   - `CreateWorktree()`: runs `git worktree add <worktreePath> <branch>` with `cmd.Dir = repoPath`.
2. `PRWriter.Checkout()` in `writer.go` remains UNCHANGED — no delegation, no touching.
   `RepoManager.CheckoutAt()` is a separate, parallel implementation on the same `*Adapter`.
3. Tests in `internal/adapter/ghcli/repomanager_test.go` (mock exec or skip with integration tag).
4. Verify `ghcli.Adapter` now satisfies both `domain.PRWriter` AND `domain.RepoManager` at compile time.

### Step 4: Smart Checkout Use Case

1. Create `internal/usecase/smart_checkout.go`:
   - Constructor: `NewSmartCheckout(repoMgr domain.RepoManager, locator *repolocator.Locator)`
   - `Plan()`: implements the decision cascade (see above). Pure logic, no I/O.
   - `ExecuteClone()`: calls `repoMgr.CloneRepo()`, then registers in locator.
   - `ExecuteCheckout()`: calls `repoMgr.CheckoutAt()` with the given workDir.
   - `ExecuteWorktree()`: calls `repoMgr.CreateWorktree()`.
   - Note: depends on `domain.RepoManager`, NOT `domain.PRWriter`. Different capability.
2. Tests in `internal/usecase/smart_checkout_test.go`:
   - Test each strategy path with mock RepoManager + mock locator
   - Test Plan() returns correct strategy for all 5 scenarios in the cascade table

### Step 5: Checkout Dialog View Model

1. Create `internal/tui/views/checkoutdialog.go`:
   - Multi-state model (see States section above)
   - Each state has its own `View()` method matching the ASCII mocks
   - Key handling per state:
     - Choice states (B, D): j/k navigate, Enter select, Esc cancel
     - Confirm state (C): Enter confirm, Esc cancel
     - Text input state (I): typing, Enter confirm, Esc cancel
     - Loading states (E, F): no keys, spinner ticking
     - Success state (G): `y` copies cd command, any other key dismisses
     - Error state: any key dismisses
   - Emit typed messages: `CheckoutStrategyChosenMsg`, `ClonePathChosenMsg`,
     `CheckoutDialogCloseMsg`, `CopyCdCommandMsg`
   - Follow all CLAUDE.md visual gotchas:
     - Full-width padding lines (`strings.Repeat(" ", width)`, never `""`)
     - ANSI resets between styled elements
     - Border width subtracted from content calculations
     - No `lipgloss.Height(n)` — use MaxHeight + manual padding
     - Truncate paths aggressively, never auto-wrap
   - `SetSize(w, h)` for responsive layout
   - `SetStyles(core.Styles)` for theme changes

2. Tests in `internal/tui/views/checkoutdialog_test.go`:
   - Test each state transition
   - Test key handling in each state
   - Test j/k navigation in choice states
   - Test Enter/Esc in all states
   - Test `y` copy in success state
   - Test View() output contains expected content for each state

### Step 6: App Integration

1. In `internal/tui/app.go`:
   - Add `WithRepoManager(rm domain.RepoManager) Option` — follows existing `WithReader/WithWriter/WithReviewer` pattern
   - Add fields: `repoManager domain.RepoManager`, `cwdRepo domain.RepoRef`, `cwdPath string`,
     `smartCheckout *usecase.SmartCheckout`, `repoLocator *repolocator.Locator`,
     `checkoutDialog views.CheckoutDialog`
   - In `Init()`: construct `SmartCheckout` use case from `a.repoManager` + `a.repoLocator`
     (same pattern as `a.checkoutPR = usecase.NewCheckoutPR(a.writer)`)
   - On startup: save CWD path, and after `RepoDetectedMsg` arrives, register it
     in known-repos (this is the auto-learning — every launch from a git repo = one learned path)
   - On `CheckoutPRMsg`:
     - Call `smartCheckout.Plan(CheckoutContext{...})` to determine strategy
     - `StrategyLocal` with worktree option: show checkout dialog in worktree-choice state (Mock B)
     - `StrategyLocal` simple: show existing confirm dialog (Mock A, backward compat)
     - `StrategyKnownPath`: show checkout dialog in known-confirm state (Mock C)
     - `StrategyNeedsClone`: show checkout dialog in options state (Mock D)
   - Handle `CheckoutStrategyChosenMsg`, `ClonePathChosenMsg`, etc.
   - After successful clone: call `repoLocator.Register()` so future checkouts skip the dialog
   - After successful remote checkout: show success with cd command (Mock G)
   - If `a.repoManager == nil`: fall through to existing `PRWriter.Checkout()` path (graceful degradation)

2. In `cmd/vivecaka/main.go`:
   - Add `tui.WithRepoManager(adapter)` alongside existing `WithWriter(adapter)`
   - Same `ghcli.Adapter` instance — it now satisfies both interfaces

3. In `internal/tui/commands.go`:
   - Add `cloneRepoCmd()`, `checkoutAtCmd()`, `createWorktreeCmd()`
   - These call `domain.RepoManager` methods (not `PRWriter`)

4. In `internal/tui/integration_test.go`:
   - Test smart checkout flow: CheckoutPRMsg → correct dialog state based on strategy
   - Test clone + checkout sequence
   - Test known-path checkout sequence
   - Test graceful degradation when `repoManager == nil`

### Step 7: Auto-Register on Startup

In `app.go` `handleRepoDetected()`, after successfully detecting the CWD repo:

```go
if a.repoLocator != nil && msg.Err == nil {
    cwd, _ := os.Getwd()
    _ = a.repoLocator.Register(msg.Repo, cwd, "detected")
}
```

This is the core of the "auto-learning" mechanism. Every launch from a git repo directory
records that repo's location. Over time, known-repos accumulates all repos the user works with.

## Critical Gotchas (from CLAUDE.md)

- **Full-width padding**: every line must be `strings.Repeat(" ", width)`, never `""` — breaks `ansi.Cut()` overlays
- **ANSI resets**: add `\033[0m` between styled elements to prevent style bleeding
- **Border width**: subtract border + padding from content width calculations
- **No `lipgloss.Height(n)`**: use `MaxHeight` + manual padding
- **No `switch msg := msg.(type)`**: use `typedMsg` + reassign to avoid shadowing
- **Async action UX (CRITICAL)**: keep dialog visible through clone→checkout→result. Do NOT close dialog and show toast.
- **`gh pr checkout` outputs to stderr**: adapter uses `git branch --show-current` for reliable branch name
- **`gh` output quirks**: verify stdout vs stderr for `gh repo clone` output before relying on it
- **Truncate paths aggressively**: never let text auto-wrap in bordered panels
- **Screen clearing**: use `tea.ClearScreen` on view transitions if needed
- **Terminal background**: empty cells use terminal's default background (see CLAUDE.md for termenv pattern)
- **Glamour**: use `WithStandardStyle("dracula")`, NOT `WithAutoStyle()` (5-second delay)
- **Domain purity**: `repolocator` must NOT import BubbleTea types
- **Use case purity**: `smart_checkout.go` must NOT import BubbleTea types, use errgroup for concurrency

## Verification

### Functional Tests

```bash
# Unit tests
go test -race -v ./internal/repolocator/...
go test -race -v ./internal/usecase/... -run TestSmartCheckout
go test -race -v ./internal/tui/views/... -run TestCheckoutDialog
go test -race -v ./internal/adapter/ghcli/... -run TestCheckoutAt
go test -race -v ./internal/adapter/ghcli/... -run TestCloneRepo

# Integration tests
go test -race -v ./internal/tui/... -run TestSmartCheckout

# Full CI
make ci
```

### Visual QA (iterm2-driver) — COMPREHENSIVE SCENARIO SUITE

**CRITICAL**: Each scenario must be tested with iterm2-driver automation. Each produces screenshots
that are visually compared against the ASCII mocks above. Do NOT mark this task done without
running ALL scenarios and capturing ALL screenshots.

Use multi-agent orchestration (parallel tmux panes) where scenarios are independent.

---

#### Scenario 0: Existing Checkout Still Works (Regression Guard)

**Purpose**: Verify the EXISTING checkout flow is not broken by this change.

**Setup**:
```bash
cd ~/code/github.com/indrasvat-vivecaka   # or any local repo with open PRs
bin/vivecaka
```

**Steps**:
1. Wait for PR list to load
2. Navigate to a PR with j/k
3. Press `c` — **screenshot S0-A**: existing confirm dialog (Mock A)
4. Press `Esc` — **screenshot S0-B**: dialog dismissed, back to PR list
5. Press `c` again, then `Enter` — **screenshot S0-C**: loading spinner
6. Wait for checkout to complete — **screenshot S0-D**: success result (Mock H)
7. Press any key — **screenshot S0-E**: back to PR list, branch name updated in header

**Verify**: All 5 screenshots match existing behavior. No regressions.

**Cleanup**:
```bash
cd ~/code/github.com/indrasvat-vivecaka && git checkout main
```

---

#### Scenario 1: Launch From Non-Git Directory, Browse Remote Repo, Attempt Checkout

**Purpose**: Test the "no local clone" path (Mock D → E → F → G).

**Setup**:
```bash
rm -rf /tmp/vivecaka-test-s1
mkdir -p /tmp/vivecaka-test-s1
```

**Steps**:
1. Launch `cd /tmp/vivecaka-test-s1 && /path/to/bin/vivecaka`
2. Expect warning about non-git-repo at startup — **screenshot S1-A**: warning visible
3. Press `Ctrl+R` to open repo switcher
4. Type `steipete/CodexBar` (or any public repo with open PRs)
5. Press Enter on the ghost add entry — **screenshot S1-B**: repo switches, PR list loads
6. Navigate to a PR, press `c` — **screenshot S1-C**: "No local clone found" dialog (Mock D)
   - Verify three options: "Clone to vivecaka cache", "Clone to custom path...", "Open on GitHub"
   - Verify cache path shown: `~/.cache/vivecaka/clones/steipete/CodexBar`
7. Press `Enter` on "Clone to vivecaka cache" — **screenshot S1-D**: clone spinner (Mock E)
8. Wait for clone to complete — **screenshot S1-E**: checkout spinner (Mock F)
9. Wait for checkout to complete — **screenshot S1-F**: success with copyable path (Mock G)
   - Verify "cd ~/.cache/vivecaka/clones/steipete/CodexBar" is shown
   - Verify `y` key hint for copy
10. Press `y` — verify cd command copied to clipboard
11. Press any key — **screenshot S1-G**: dialog dismissed

**Verify**:
- Clone actually exists: `ls ~/.cache/vivecaka/clones/steipete/CodexBar/.git`
- Known-repos updated: `cat ~/.local/share/vivecaka/known-repos.json` contains steipete/CodexBar
- Branch checked out: `cd ~/.cache/vivecaka/clones/steipete/CodexBar && git branch --show-current`

**Cleanup**:
```bash
rm -rf /tmp/vivecaka-test-s1
rm -rf ~/.cache/vivecaka/clones/steipete/CodexBar
# Remove entry from known-repos.json (or leave for Scenario 3)
```

---

#### Scenario 2: Launch From Repo A, Browse Repo B, Attempt Checkout (Mismatch)

**Purpose**: Test the "wrong repo" detection and redirect to options dialog.

**Setup**:
```bash
cd ~/code/github.com/indrasvat-vivecaka   # repo A (CWD)
# Ensure steipete/CodexBar is NOT in known-repos (clean from Scenario 1 cleanup)
```

**Steps**:
1. Launch `bin/vivecaka` from repo A
2. Wait for PR list to load — **screenshot S2-A**: repo A's PRs shown
3. Press `Ctrl+R`, switch to a different public repo (e.g., `steipete/CodexBar`)
4. Wait for PRs to load — **screenshot S2-B**: repo B's PRs shown
5. Navigate to a PR, press `c` — **screenshot S2-C**: "No local clone found" dialog (Mock D)
   - NOT the normal checkout confirm! Must detect mismatch.
   - Verify message mentions `steipete/CodexBar` is not cloned locally
6. Press `Esc` to cancel — **screenshot S2-D**: dialog dismissed, back to PR list

**Verify**: Checkout was NOT attempted in repo A's directory. `git log --oneline -1` in repo A
shows no unexpected branch changes.

**Cleanup**: None needed (cancelled before any side effects).

---

#### Scenario 3: Second Checkout of Same Remote Repo (Known-Repos Hit)

**Purpose**: Test that after Scenario 1's clone, subsequent checkouts skip the dialog.

**Setup**:
```bash
# Re-run Scenario 1 first (or ensure known-repos has steipete/CodexBar entry)
# Verify: cat ~/.local/share/vivecaka/known-repos.json
rm -rf /tmp/vivecaka-test-s3
mkdir -p /tmp/vivecaka-test-s3
```

**Steps**:
1. Launch `cd /tmp/vivecaka-test-s3 && /path/to/bin/vivecaka`
2. Press `Ctrl+R`, switch to `steipete/CodexBar`
3. Navigate to a DIFFERENT PR than Scenario 1, press `c`
4. **screenshot S3-A**: "Checkout at known path" dialog (Mock C)
   - Verify it shows the cached clone path, NOT the "no clone found" dialog
   - This proves known-repos is working
5. Press Enter to confirm — **screenshot S3-B**: checkout spinner
6. Wait for checkout — **screenshot S3-C**: success with path (Mock G)

**Verify**:
- No re-clone happened (should be fast — just fetch + branch switch)
- Correct branch checked out: `cd ~/.cache/vivecaka/clones/steipete/CodexBar && git branch --show-current`

**Cleanup**:
```bash
rm -rf /tmp/vivecaka-test-s3
rm -rf ~/.cache/vivecaka/clones/steipete/CodexBar
# Clean known-repos entry
```

---

#### Scenario 4: Stale Known-Repos Entry (Path Deleted)

**Purpose**: Test graceful handling when a known path no longer exists.

**Setup**:
```bash
# Run Scenario 1 to get a known-repos entry
# Then delete the clone:
rm -rf ~/.cache/vivecaka/clones/steipete/CodexBar
# known-repos.json still has the entry — it's now stale
rm -rf /tmp/vivecaka-test-s4 && mkdir -p /tmp/vivecaka-test-s4
```

**Steps**:
1. Launch `cd /tmp/vivecaka-test-s4 && /path/to/bin/vivecaka`
2. Switch to `steipete/CodexBar`, navigate to a PR, press `c`
3. **screenshot S4-A**: should show "No local clone found" dialog (Mock D), NOT the "known path" dialog
   - The stale entry was detected and removed

**Verify**: `cat ~/.local/share/vivecaka/known-repos.json` no longer contains steipete/CodexBar.

**Cleanup**:
```bash
rm -rf /tmp/vivecaka-test-s4
```

---

#### Scenario 5: "Open on GitHub" Option

**Purpose**: Test the browser fallback option in the checkout dialog.

**Setup**: Same as Scenario 1 setup.

**Steps**:
1. Launch from non-git dir, switch to a remote repo, press `c` on a PR
2. **screenshot S5-A**: options dialog (Mock D)
3. Navigate to "Open on GitHub" option with j/k — **screenshot S5-B**: cursor on "Open on GitHub"
4. Press Enter — verify browser opens the PR page

**Verify**: Browser opened with correct URL (e.g., `https://github.com/steipete/CodexBar/pull/255`).

**Cleanup**: Same as Scenario 1 cleanup.

---

#### Scenario 6: Clone to Custom Path

**Purpose**: Test the "Clone to custom path..." option with text input.

**Setup**: Same as Scenario 1 setup.

**Steps**:
1. Launch from non-git dir, switch to a remote repo, press `c` on a PR
2. Navigate to "Clone to custom path..." option, press Enter
3. **screenshot S6-A**: text input dialog (Mock I) with pre-filled default path
4. Clear and type a custom path: `/tmp/vivecaka-test-s6/my-clone`
5. Press Enter — **screenshot S6-B**: clone spinner (Mock E)
6. Wait for clone + checkout — **screenshot S6-C**: success with the custom path shown (Mock G)

**Verify**:
- Clone exists at custom path: `ls /tmp/vivecaka-test-s6/my-clone/.git`
- Known-repos updated with custom path
- Branch checked out correctly

**Cleanup**:
```bash
rm -rf /tmp/vivecaka-test-s6
# Clean known-repos entry
```

---

#### Scenario 7: Worktree Option (CWD = Correct Repo)

**Purpose**: Test the worktree choice dialog when in the correct repo.

**Setup**:
```bash
cd ~/code/github.com/indrasvat-vivecaka
git checkout main   # ensure we're on a known branch
```

**Steps**:
1. Launch `bin/vivecaka`
2. Wait for PR list to load (should show vivecaka's own PRs, or switch to a repo with PRs)
3. Navigate to a PR, press `c`
4. **screenshot S7-A**: worktree choice dialog (Mock B)
   - "Switch branch" (replaces current branch)
   - "New worktree" (keeps current branch intact)
5. Navigate to "New worktree" with j/k — **screenshot S7-B**: cursor on worktree option
   - Verify worktree path shown: `.worktrees/pr-N-branch-name`
6. Press Enter — **screenshot S7-C**: checkout/worktree spinner
7. Wait for completion — **screenshot S7-D**: success with worktree path

**Verify**:
- Current branch unchanged: `git branch --show-current` → still `main`
- Worktree created: `git worktree list` shows the new worktree
- PR branch in worktree: `cd .worktrees/pr-N-* && git branch --show-current`

**Cleanup**:
```bash
git worktree remove .worktrees/pr-N-*   # remove the test worktree
```

---

#### Scenario 8: Worktree "Switch Branch" Option (Backward Compat)

**Purpose**: Verify "Switch branch" in worktree dialog behaves identically to old checkout.

**Setup**: Same as Scenario 7.

**Steps**:
1. Same as Scenario 7 steps 1-4
2. With cursor on "Switch branch" (default), press Enter — **screenshot S8-A**: loading spinner
3. Wait for checkout — **screenshot S8-B**: success (Mock H, same as existing)

**Verify**: Branch actually switched: `git branch --show-current` → the PR's branch name.

**Cleanup**:
```bash
git checkout main
```

---

#### Scenario 9: Auto-Learning Verification

**Purpose**: Verify that launching from a git directory auto-registers in known-repos.

**Setup**:
```bash
# Clear known-repos
rm -f ~/.local/share/vivecaka/known-repos.json
```

**Steps**:
1. Launch `bin/vivecaka` from `~/code/github.com/indrasvat-vivecaka`
2. Wait for startup — **screenshot S9-A**: app loaded
3. Press `q` to quit
4. Inspect known-repos: `cat ~/.local/share/vivecaka/known-repos.json`

**Verify**: File contains `indrasvat/vivecaka` mapped to the CWD path with `source: "detected"`.

**Cleanup**: None (this is the desired persistent state).

---

#### Scenario 10: Esc Cancels at Every Dialog Stage

**Purpose**: Verify Esc cleanly cancels at every point in the flow.

**Steps** (run as sub-scenarios):
1. Options dialog (Mock D) → press Esc → back to PR list. **screenshot S10-A**
2. Known path confirm (Mock C) → press Esc → back to PR list. **screenshot S10-B**
3. Worktree choice (Mock B) → press Esc → back to PR list. **screenshot S10-C**
4. Custom path input (Mock I) → press Esc → back to options dialog. **screenshot S10-D**

**Verify**: No side effects (no clone started, no branch changed, no worktree created).

---

### Test Execution Strategy

**Parallel execution via tmux** (multi-agent orchestration):

- **Agent 1**: Scenarios 0, 7, 8 (all require being in the correct local repo)
- **Agent 2**: Scenarios 1, 3, 4 (sequential — S3 depends on S1's clone, S4 depends on S3's cleanup)
- **Agent 3**: Scenarios 2, 5, 6 (independent, all start from "wrong repo" or "no repo")
- **Agent 4**: Scenarios 9, 10 (quick verification scenarios)

Each agent runs iterm2-driver scripts and captures screenshots to `screenshots/smart-checkout/`.

### Screenshot Naming Convention

```
screenshots/smart-checkout/
  S0-A-existing-confirm.png
  S0-B-cancel-return.png
  S0-C-loading-spinner.png
  S0-D-success-result.png
  S0-E-dismiss-pr-list.png
  S1-A-no-git-warning.png
  S1-B-repo-switched.png
  S1-C-no-clone-dialog.png
  S1-D-clone-spinner.png
  S1-E-checkout-spinner.png
  S1-F-success-with-path.png
  S1-G-dismissed.png
  ... (etc for all scenarios)
```

## Commit Strategy

This is a large feature. Split into atomic commits:

```
1. feat(domain): add RepoManager interface and RepoLocation type
2. feat(plugin): register RepoManager capability in plugin registry
3. feat(repolocator): add known-repos registry with auto-learning
4. feat(adapter): implement RepoManager (CheckoutAt, CloneRepo, CreateWorktree)
5. feat(usecase): add SmartCheckout use case with checkout planning
6. feat(tui): add multi-state checkout dialog
7. feat(tui): integrate smart checkout with RepoManager wiring
8. test: add smart checkout integration tests and visual QA
```

Each commit must pass `make ci` independently.

## Design Review Fixes (from Codex review)

The following issues were identified during design review and MUST be addressed during implementation.

### DR-1: `Plan()` Needs I/O for Path Validation (Critical)

The task says `Plan()` is pure/no-I/O, but it must validate known-repos paths (check path exists,
check git remote matches). This is I/O.

**Fix**: Split into two steps:
```go
// Pure planning — determines strategy from in-memory data
func (uc *SmartCheckout) Plan(ctx CheckoutContext, knownPath string, knownPathValid bool) CheckoutPlan

// I/O validation — called by TUI before Plan()
func (l *Locator) Validate(repo domain.RepoRef) (path string, valid bool)
```

The TUI calls `locator.Validate()` first (async, via tea.Cmd), then feeds the result into
the pure `Plan()`. This keeps the use case testable with no mocks.

### DR-2: nil `repoManager` Fallback Must Be Safe (Critical)

When `repoManager == nil`, falling back to `PRWriter.Checkout()` is ONLY safe when
`a.repo == a.cwdRepo`. If the repos don't match, the fallback reintroduces the exact
wrong-repo bug this task is fixing.

**Fix**: In the fallback path:
```go
if a.repoManager == nil {
    if a.repo == a.cwdRepo {
        // Safe: CWD matches browsing target, use old path
        return a.handleCheckoutConfirm(msg)
    }
    // Unsafe: show error dialog explaining RepoManager is required
    a.confirmDialog.ShowResult("Checkout Unavailable",
        "Cannot check out PRs from a different repo without RepoManager capability.", false)
    return a, nil
}
```

### DR-3: Worktree Flow Needs Fetch Step (High)

`git worktree add <path> <branch>` requires the branch to exist locally or as a remote
tracking branch. For PR branches from forks, the branch may not exist at all.

**Fix**: The worktree sequence must be:
1. `gh pr checkout <number> --detach` (fetches the PR ref and detaches HEAD — does NOT switch branch)
   OR `git fetch origin pull/<number>/head:<local-branch-name>`
2. `git worktree add <path> <local-branch-name>`
3. On failure: `git worktree remove <path>` + `git branch -D <local-branch-name>` (cleanup)

Document this in the adapter's `CreateWorktree` implementation notes.

### DR-4: Clone Timeout and Cancellation (High)

Clone operations can take minutes for large repos. Current checkout uses `context.Background()`
without timeout. Clone needs:
- A generous timeout (e.g., 5 minutes): `context.WithTimeout(ctx, 5*time.Minute)`
- The dialog must show elapsed time or at minimum a "this may take a while" message
- Esc during clone should cancel the context and clean up the partial clone directory
- On cancellation: `os.RemoveAll(targetPath)` to avoid leaving a corrupted partial clone

### DR-5: Known-Repos File Locking (High)

Atomic rename prevents corruption but doesn't prevent read-modify-write races across concurrent
vivecaka instances. Two instances launching simultaneously could both read, both modify, and the
second write overwrites the first's changes.

**Fix**: Use `flock` (file locking) around read+modify+write in `repolocator`:
```go
func (l *Locator) withLock(fn func() error) error {
    lockPath := l.dataPath + ".lock"
    f, _ := os.Create(lockPath)
    defer f.Close()
    syscall.Flock(int(f.Fd()), syscall.LOCK_EX)
    defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
    return fn()
}
```

This matches the pattern used by package managers and other XDG-aware tools.

### DR-6: Plugin Registry — Scope as Future-Ready (Medium)

The plugin registry (`internal/plugin/registry.go`) exists but is NOT used in `main.go` runtime
wiring. Adding `RepoManager` discovery is architecturally correct but will be dead code until
the registry is wired into app startup.

**Fix**: Still add the registry support (it's cheap and keeps the pattern consistent), but
document clearly that it's for future plugin authors, not for current MVP wiring. The current
wiring path is `main.go → WithRepoManager(adapter)` direct injection.

### DR-7: Naming Consistency (Medium)

1. **Dialog type**: Use `CheckoutDialogModel` (not `CheckoutDialog`) to match `ConfirmModel`,
   `PRListModel`, `RepoSwitcherModel` convention.
2. **RepoLocation placement**: Put in `internal/domain/repo.go` (not `pr.go`) — repo-scoped
   types already live there (`RepoConfig`, `ListOpts`).
3. **Field name**: Use `checkoutDialog CheckoutDialogModel` in app.go.

### DR-8: Keyboard Navigation (Medium)

ASCII mocks show `↑/↓ select` but task text says `j/k`. The correct answer: use the app's
`KeyMap` bindings (`keys.Up`/`keys.Down`) so both work, and user overrides apply. The hint
text in the dialog should show `j/k` (matching the rest of the app's hint style), not `↑/↓`.

Fix all ASCII mocks: change `↑/↓ select` to `j/k select`.

### DR-9: Modal Routing Integration (Medium)

App currently hardwires `ViewConfirm` to `confirmDialog` in key routing, rendering, and status
hints. The new `checkoutDialog` needs its own `ViewState` or reuse of `ViewConfirm`.

**Fix**: Add `ViewSmartCheckout` to `core.ViewState`. This avoids collision with the existing
confirm dialog (which is still used for non-checkout confirmations). Route to `checkoutDialog`
when `a.view == core.ViewSmartCheckout`.

### DR-10: Edge Case Handling (Medium)

Add explicit handling for these cases in the adapter/use case:

| Case | Handling |
|------|---------|
| Clone target path already exists (corrupted) | `os.RemoveAll(targetPath)` before cloning |
| Clone target path already exists (valid clone) | Skip clone, just fetch + checkout |
| `gh` auth failure / private repo | Surface `gh` error message in dialog |
| ENOSPC (disk full during clone) | Clean up partial clone, surface error |
| Custom path input: `~` prefix | Expand via `os.UserHomeDir()` + string replace |
| Custom path input: relative path | Resolve via `filepath.Abs()` |
| Custom path input: empty/invalid | Validate before starting clone, show inline error |
| Worktree path collision | Check `git worktree list` before creating |

### DR-11: Additional Test Cases (from review)

Add these to the functional test suite:

```bash
# Edge case tests for repolocator
go test -race -v ./internal/repolocator/... -run TestConcurrentRegister
go test -race -v ./internal/repolocator/... -run TestPathExpansion
go test -race -v ./internal/repolocator/... -run TestCorruptedJSON

# Edge case tests for adapter
go test -race -v ./internal/adapter/ghcli/... -run TestCloneRepoAuthFailure
go test -race -v ./internal/adapter/ghcli/... -run TestCloneRepoExistingPath
go test -race -v ./internal/adapter/ghcli/... -run TestCheckoutAtTimeout

# Integration tests for dialog state machine
go test -race -v ./internal/tui/... -run TestSmartCheckoutFallbackSafety
go test -race -v ./internal/tui/... -run TestSmartCheckoutCancelDuringClone
```

### DR-12: Existing Integration Tests Will Need Updates

The current checkout integration tests (`internal/tui/integration_test.go:438-468`) test the
`CheckoutPRMsg → ViewConfirm → ConfirmResultMsg → CheckoutDoneMsg` flow directly. When smart
checkout intercepts `CheckoutPRMsg`, these tests will route differently.

**Fix**: Update existing tests to explicitly set `a.cwdRepo = a.repo` (simulating the "CWD matches
browsing target" case) so they exercise the StrategyLocal path and produce the same dialog behavior.
Add separate tests for the smart checkout paths.

## Session Protocol

1. Read `CLAUDE.md`
2. Read this task file — including ALL Design Review Fixes (DR-1 through DR-12)
3. Read relevant PRD sections
4. Execute steps 1-6 sequentially (each step = 1 commit)
5. Step 7: run ALL 11 visual QA scenarios via iterm2-driver
6. Verify — `make ci` passes + ALL screenshots captured + visual match confirmed
7. Update `docs/PROGRESS.md` — mark task 030 DONE with notes
8. Update this file — set `Status: DONE`
9. `git add` changed files + `git commit`
