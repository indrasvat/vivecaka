# vivecaka (विवेचक) - Product Requirements Document

> **A stunningly beautiful, blazing-fast TUI for browsing and reviewing GitHub Pull Requests.**
>
> *vivecaka* (Sanskrit: विवेचक) means "one who examines/analyzes" - a discerning reviewer.

**Version:** 1.0.0-MVP
**Date:** February 2026
**License:** MIT
**Go Version:** 1.25+ (targeting 1.26 w/ Green Tea GC)
**BubbleTea:** v1.3.x stable (v2-ready architecture)

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Problem Statement & Market Analysis](#2-problem-statement--market-analysis)
3. [Architecture Overview](#3-architecture-overview)
4. [Plugin System Design](#4-plugin-system-design)
5. [Core Features (MVP)](#5-core-features-mvp)
6. [Standout Features](#6-standout-features)
7. [UI Design & ASCII Mocks](#7-ui-design--ascii-mocks)
8. [User Flows](#8-user-flows)
9. [Technical Specifications](#9-technical-specifications)
10. [Configuration](#10-configuration)
11. [Task Breakdown](#11-task-breakdown)
12. [Testing Strategy](#12-testing-strategy)
13. [Makefile & CI](#13-makefile--ci)
14. [Terminal Compatibility](#14-terminal-compatibility)
15. [Visual QA with iterm2-driver](#15-visual-qa-with-iterm2-driver)

---

## 1. Executive Summary

**vivecaka** is a plugin-based, keyboard-driven TUI for GitHub PR management. It fills the gap between `gh pr list` (too bare) and the browser (too heavy) by providing a rich, context-aware, auto-refreshing PR dashboard right in your terminal.

### Why Another Tool?

| Tool | Limitation vivecaka Solves |
|------|---------------------------|
| `gh pr list/view` | No TUI, no inline review, no diff viewer, verbose |
| `gh-dash` | No PR creation, static view, poor vim keybindings, no inline comments |
| `lazygit` | No PR interaction beyond opening browser, no review |
| `gitui` | Zero GitHub integration |
| Octo.nvim | Neovim-only, not standalone |

### Core Differentiators

1. **Plugin architecture** - Everything is a plugin. Swap `gh` CLI for direct API, add GitLab, extend views.
2. **Context-aware** - Auto-detects repo, remembers last view per repo, smart filters.
3. **Inline review** - Add/view/resolve review comments on diff lines without leaving the terminal.
4. **Dual diff viewer** - Built-in syntax-highlighted diff + delegate to external tools (delta, difftastic).
5. **Auto-refresh** - Background polling with visual indicators for new activity.
6. **Multi-repo favorites** - Pin repos for quick switching, unified PR inbox.
7. **Beautiful by default** - Charmbracelet theme support, adaptive colors, modern Unicode rendering.

---

## 2. Problem Statement & Market Analysis

### Developer Pain Points (from research across GitHub Issues, HN, Reddit)

1. **Context switching** - Reviewing PRs in browser breaks terminal flow
2. **No inline comments from terminal** - `gh` CLI lacks inline comment support entirely (issues #12396, #5788)
3. **Static views** - `gh-dash` shows a snapshot; you must manually refresh
4. **Poor keyboard navigation** - Tools use Ctrl+D/U instead of vim's hjkl (gh-dash #214)
5. **No PR push** - Can't push changes back to a PR after checkout (gh #2189, #3370)
6. **Incomplete review data** - `gh pr view --json reviews` misses line comments (gh #3993)
7. **No multi-repo view** - Each tool operates on one repo at a time

### Competitive Landscape (Feb 2026)

| Tool | Language | Stars | Last Active | Strengths | Weaknesses |
|------|----------|-------|-------------|-----------|------------|
| gh-dash | Go/BubbleTea | 7k+ | Active | Good dashboard, custom actions | No review, static, poor vim keys |
| lazygit | Go | 55k+ | Active | Excellent git TUI, huge community | No PR review, browser-only PRs |
| gitui | Rust | 19k+ | Active | Fast, good git UI | Zero forge integration |
| stax | Rust | New | Active | Stacked PRs, tree view | Stacked-only, niche |
| PRR | Rust | 1k+ | Moderate | Editor-based review | Not a TUI, limited |
| gh-pr-review | Go | New | Active | Inline review via gh | CLI-only, not a TUI |

### Opportunity

No tool provides: **beautiful TUI + inline review + diff viewer + auto-refresh + multi-repo + plugin architecture** in a single package. vivecaka fills this gap.

---

## 3. Architecture Overview

### Clean Architecture Layers

```
┌─────────────────────────────────────────────────┐
│                   cmd/vivecaka                   │  Entry point, DI wiring
├─────────────────────────────────────────────────┤
│                  internal/tui                    │  BubbleTea UI layer
│  ┌──────────┐ ┌──────────┐ ┌──────────────────┐ │
│  │  Views   │ │Components│ │   Theme/KeyMap    │ │
│  └──────────┘ └──────────┘ └──────────────────┘ │
├─────────────────────────────────────────────────┤
│               internal/usecase                   │  Application logic
│  ┌──────────┐ ┌──────────┐ ┌──────────────────┐ │
│  │ ListPRs  │ │ ViewPR   │ │  ReviewPR, etc.  │ │
│  └──────────┘ └──────────┘ └──────────────────┘ │
├─────────────────────────────────────────────────┤
│               internal/domain                    │  Entities & interfaces
│  ┌──────────┐ ┌──────────┐ ┌──────────────────┐ │
│  │    PR     │ │  Review  │ │  Repository I/F  │ │
│  └──────────┘ └──────────┘ └──────────────────┘ │
├─────────────────────────────────────────────────┤
│               internal/adapter                   │  Interface implementations
│  ┌──────────────┐  ┌──────────────────────────┐ │
│  │  ghcli (MVP) │  │  github-api (future)     │ │
│  └──────────────┘  └──────────────────────────┘ │
├─────────────────────────────────────────────────┤
│               internal/plugin                    │  Plugin registry & hooks
│  ┌──────────┐ ┌──────────┐ ┌──────────────────┐ │
│  │ Registry │ │  Hooks   │ │  Plugin I/F      │ │
│  └──────────┘ └──────────┘ └──────────────────┘ │
├─────────────────────────────────────────────────┤
│               internal/config                    │  XDG config management
│  ┌──────────────────────────────────────────┐   │
│  │ TOML config + favorites + keybindings    │   │
│  └──────────────────────────────────────────┘   │
└─────────────────────────────────────────────────┘
```

### Dependency Rule

```
tui → usecase → domain ← adapter
                domain ← plugin
config is injected everywhere via functional options
```

- **domain** knows nothing about outer layers
- **usecase** depends only on domain interfaces
- **adapter** implements domain interfaces
- **tui** consumes use cases, never touches adapters directly
- **plugin** extends domain interfaces and hooks into lifecycle

### Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Plugin mechanism | Interface-based (compile-time) | Idiomatic Go, zero overhead, type-safe |
| DI approach | Manual + functional options | Simple, no framework, easy to test |
| State management | BubbleTea Elm architecture | Proven, testable, no race conditions |
| Config format | TOML | Human-readable, well-supported in Go |
| Data fetching | Commands (tea.Cmd) | Non-blocking, BubbleTea native |
| Error handling | Sentinel errors + wrapping | Standard Go patterns |

---

## 4. Plugin System Design

### Plugin Interface

> **Architecture note:** Plugin interfaces live in `internal/plugin/`, NOT in `internal/domain/`.
> They intentionally reference `tea.Cmd`, `tea.Model`, and `key.Binding` because plugins bridge
> the domain and TUI layers. Domain interfaces (PRReader, PRReviewer, PRWriter) remain pure and
> framework-agnostic in `internal/domain/interfaces.go`. The dependency rule is preserved:
> `domain ← plugin ← tui`, where `plugin` may import both `domain` and `bubbletea`.

```go
// Plugin is the base interface all plugins must implement.
type Plugin interface {
    // Info returns metadata about the plugin.
    Info() PluginInfo

    // Init is called once when the plugin is registered.
    // Returns a tea.Cmd for any async initialization.
    Init(app AppContext) tea.Cmd
}

type PluginInfo struct {
    Name        string   // unique identifier: "ghcli", "github-api"
    Version     string   // semver
    Description string   // human-readable
    Provides    []string // capabilities: "pr-reader", "pr-reviewer"
}
```

### Capability Interfaces (ISP)

```go
// PRReader provides read-only access to pull requests.
type PRReader interface {
    ListPRs(ctx context.Context, repo RepoRef, opts ListOpts) ([]PR, error)
    GetPR(ctx context.Context, repo RepoRef, number int) (*PRDetail, error)
    GetDiff(ctx context.Context, repo RepoRef, number int) (*Diff, error)
    GetChecks(ctx context.Context, repo RepoRef, number int) ([]Check, error)
    GetComments(ctx context.Context, repo RepoRef, number int) ([]CommentThread, error)
}

// PRReviewer provides review capabilities.
type PRReviewer interface {
    SubmitReview(ctx context.Context, repo RepoRef, number int, review Review) error
    AddComment(ctx context.Context, repo RepoRef, number int, input InlineCommentInput) error
    ResolveThread(ctx context.Context, repo RepoRef, threadID string) error
}

// PRWriter provides write capabilities.
// NOTE: Only Checkout is exposed in MVP. Merge and UpdateLabels are defined
// for plugin extensibility and future features (post-MVP).
type PRWriter interface {
    Checkout(ctx context.Context, repo RepoRef, number int) (branch string, err error)
    Merge(ctx context.Context, repo RepoRef, number int, opts MergeOpts) error
    UpdateLabels(ctx context.Context, repo RepoRef, number int, labels []string) error
}

// ViewPlugin provides custom UI views.
type ViewPlugin interface {
    Plugin
    Views() []ViewRegistration
}

// KeyPlugin provides custom key bindings.
type KeyPlugin interface {
    Plugin
    KeyBindings() []KeyRegistration
}
```

### Plugin Registry

```go
type Registry struct {
    plugins   map[string]Plugin
    readers   []PRReader
    reviewers []PRReviewer
    writers   []PRWriter
    views     []ViewRegistration
    keys      []KeyRegistration
    hooks     *HookManager
}

// Register adds a plugin and auto-discovers its capabilities.
func (r *Registry) Register(p Plugin) error {
    info := p.Info()
    if _, exists := r.plugins[info.Name]; exists {
        return fmt.Errorf("plugin %q already registered", info.Name)
    }
    r.plugins[info.Name] = p

    // Auto-discover capabilities via type assertion
    if reader, ok := p.(PRReader); ok {
        r.readers = append(r.readers, reader)
    }
    if reviewer, ok := p.(PRReviewer); ok {
        r.reviewers = append(r.reviewers, reviewer)
    }
    // ... etc
    return nil
}
```

### Hook System

```go
type HookPoint string

const (
    HookBeforeFetch   HookPoint = "before_fetch"
    HookAfterFetch    HookPoint = "after_fetch"
    HookBeforeRender  HookPoint = "before_render"
    HookOnPRSelect    HookPoint = "on_pr_select"
    HookOnViewChange  HookPoint = "on_view_change"
)

type HookHandler func(ctx context.Context, data any) error

type HookManager struct {
    mu    sync.RWMutex
    hooks map[HookPoint][]HookHandler
}

func (hm *HookManager) On(point HookPoint, handler HookHandler) {
    hm.mu.Lock()
    defer hm.mu.Unlock()
    hm.hooks[point] = append(hm.hooks[point], handler)
}

func (hm *HookManager) Emit(ctx context.Context, point HookPoint, data any) error {
    hm.mu.RLock()
    handlers := hm.hooks[point]
    hm.mu.RUnlock()
    for _, handler := range handlers {
        if err := handler(ctx, data); err != nil {
            return err
        }
    }
    return nil
}
```

### MVP Plugins

1. **ghcli** - PR data via `gh` CLI (implements PRReader, PRReviewer, PRWriter)
2. **diff-builtin** - Built-in diff viewer (ViewPlugin)
3. **diff-external** - External diff tool delegation (ViewPlugin)

---

## 5. Core Features (MVP)

### F1: PR List View
- List open PRs for current repo (auto-detected from CWD)
- Columns: number, title, author, status (draft/ready), CI status, updated-at, review state
- Sortable by any column
- Filterable: by author, label, status, CI state, review state, search text
- Paginated with infinite scroll
- Visual indicators: draft (dimmed), approved (green check), changes requested (orange), CI failing (red)
- Keyboard: j/k navigate, Enter opens detail, / searches, f filters, s sorts

### F2: PR Detail View
- Full PR metadata: title, body (markdown rendered), author, branch, labels, assignees, reviewers
- Review summary: approved by, changes requested by, pending reviewers
- CI checks with status and links
- File changed list with +/- line counts
- Comment threads (threaded, collapsible)
- Keyboard: Tab switches panes, q goes back, c checkout, d opens diff, r review

### F3: Diff Viewer (Built-in)
- Unified and side-by-side modes (toggle with `t`)
- Syntax highlighting via Glamour/chroma
- File tree navigation (left pane) + diff content (right pane)
- Hunk navigation: `[` / `]` to jump between hunks
- File navigation: `{` / `}` to jump between files
- Line numbers with +/- indicators
- Collapse/expand files with `za`
- Keyboard: hjkl scroll, gg/G top/bottom, / search, n/N next/prev search match

### F4: Diff Viewer (External)
- Open current file's diff in configured external tool
- Supports: delta, difftastic, VS Code, any tool
- Configurable via `config.toml`: `diff.external_tool = "delta"`
- Keyboard: `e` opens external viewer for current file

### F5: PR Checkout
- One-key checkout: `c` from PR list or detail
- Shows confirmation with branch name
- Handles existing local branch (force/skip prompt)
- Returns to PR list after checkout
- Status bar shows current branch

### F6: PR Review Submission
- `r` opens review form (via `huh` library)
- Options: Approve, Request Changes, Comment
- Body editor (multi-line textarea)
- Submit confirmation
- Success/error notification in status bar

### F7: Inline Comments
- In diff view, press `c` on any line to add comment
- Comment editor with markdown support
- View existing comment threads inline (highlighted lines)
- Reply to threads with `r`
- Resolve threads with `x`

### F8: CI Status
- Dedicated pane in PR detail showing all checks
- Color-coded: green (pass), red (fail), yellow (pending), gray (skipped)
- Show check name, status, duration
- Deep-link to check details (opens in browser with `o`)

### F9: Multi-Repo Favorites
- `config.toml` section for pinned repos
- Repo switcher: `Ctrl+R` opens fuzzy finder
- Last-used repo remembered on restart
- Repo indicator in header bar

### F10: Auto-Refresh
- Background polling every 30s (configurable)
- Visual indicator: dot/badge on PRs with new activity
- Toast notification for new PRs / review requests
- Pause auto-refresh with `p` (resume with `p` again)
- Configurable interval in `config.toml`

### F11: Theme Support
- Ships with Charmbracelet-compatible dark themes
- Switchable via `config.toml`: `theme = "catppuccin-mocha"`
- Adaptive colors (auto-detect terminal background)
- Consistent color semantics: success=green, error=red, warning=yellow, info=blue

### F12: Help System
- Context-aware help: `?` shows keys for current view
- Full help overlay: `?` shows all keybindings (context-aware per view)
- Inline hints in status bar for common actions
- Discoverable: first-launch tutorial overlay

---

## 6. Standout Features

These differentiate vivecaka from every other tool:

### S1: Smart Context Awareness
- Auto-detects repo from CWD git remote
- If on a branch with an associated PR, highlights that PR
- Remembers last filter/sort per repo
- "My PRs" and "Needs My Review" as first-class quick filters

### S2: Keyboard-First with Zero Hand Gymnastics
- Full vim-style navigation (hjkl, gg/G, Ctrl+d/u for half-page)
- Modal interface hints (like vim's `-- INSERT --`)
- Minimal Ctrl usage: only Ctrl+R (repo switch) and Ctrl+D/U (page scroll) - all primary actions are single-key
- Customizable keybindings in config

### S3: Rich Inline Diff Review
- View + add + resolve comments directly on diff lines
- Threaded conversation view per-hunk
- Markdown rendering in comment bodies
- Suggestion blocks (code suggestions that can be applied) — **post-MVP**: requires GitHub GraphQL API for suggestion application

### S4: Unified PR Inbox
- See PRs across all favorite repos in one view
- "Assigned to me", "Review requested", "My PRs" tabs
- Unread indicators per-PR
- Priority sorting: review-requested > CI-failing > stale

### S5: Progressive Disclosure
- Simple by default, powerful when needed
- `?` for context help at every level
- Advanced filters behind `f` key
- Batch operations behind visual selection mode (`v`)

### S6: Instant Startup
- Target: <100ms to first render
- Cache last-fetched PR list for instant display
- Background refresh to update stale data
- Optimistic UI: show cached data immediately, update when fresh data arrives

---

## 7. UI Design & ASCII Mocks

### 7.0 Startup Banner

Shown briefly on launch (200ms) before transitioning to the PR list. Also displayed via `vivecaka --version`. Rendered using the theme's primary/accent colors.

```
              ██                                             ▄▄
              ▀▀                                             ██
 ██▄  ▄██   ████     ██▄  ▄██   ▄████▄    ▄█████▄   ▄█████▄  ██ ▄██▀    ▄█████▄
  ██  ██      ██      ██  ██   ██▄▄▄▄██  ██▀    ▀   ▀ ▄▄▄██  ██▄██      ▀ ▄▄▄██
  ▀█▄▄█▀      ██      ▀█▄▄█▀   ██▀▀▀▀▀▀  ██        ▄██▀▀▀██  ██▀██▄    ▄██▀▀▀██
   ████    ▄▄▄██▄▄▄    ████    ▀██▄▄▄▄█  ▀██▄▄▄▄█  ██▄▄▄███  ██  ▀█▄   ██▄▄▄███
    ▀▀     ▀▀▀▀▀▀▀▀     ▀▀       ▀▀▀▀▀     ▀▀▀▀▀    ▀▀▀▀ ▀▀  ▀▀   ▀▀▀   ▀▀▀▀ ▀▀

                          विवेचक
                 ─── the discerning reviewer ───

               v1.0.0 · github.com/indrasvat/vivecaka
```

Generated with `toilet -f mono12 vivecaka`. Uses Unicode block elements (▄▀█) for solid, high-contrast rendering in any modern terminal.

**Design notes:**
- Uses Unicode block elements (▄▀█) for solid, high-contrast rendering in all modern terminals
- The Sanskrit text (विवेचक) uses Devanagari script, widely supported in monospace fonts
- Colors: Banner in `theme.primary` (mauve), tagline in `theme.subtext`, version in `theme.muted`
- Shown in `--version` flag output (always), and as startup splash (configurable, default on)
- Startup splash auto-dismisses after 200ms or on any keypress

### 7.1 Main Layout Structure

```
┌─[ vivecaka ]──────────────────────────────────────────────────────────┐
│ ◉ indrasvat/vivecaka  │  ⚡ 12 open  │  ▸ My PRs  │  ⟳ 30s       │
├───────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  #  │ Title                          │ Author   │ CI │ Review │ Age  │
│ ────┼────────────────────────────────┼──────────┼────┼────────┼───── │
│ ▸142│ Add plugin architecture        │ indrasvat│ ✓  │ ✓ 2/2  │  2h  │
│  141│ Fix diff viewer alignment      │ alice    │ ✗  │ ● 0/1  │  5h  │
│  140│ Update CI pipeline             │ bob      │ ◐  │ —      │  1d  │
│  139│ [DRAFT] New theme engine       │ indrasvat│ —  │ —      │  2d  │
│  138│ Refactor config loader         │ carol    │ ✓  │ ! 1/2  │  3d  │
│  137│ Add inline comment support     │ dave     │ ✓  │ ✓ 3/3  │  3d  │
│  ...│                                │          │    │        │      │
│                                                                       │
├───────────────────────────────────────────────────────────────────────┤
│ j/k navigate  Enter detail  c checkout  / search  f filter  ? help  │
└───────────────────────────────────────────────────────────────────────┘
```

**Legend:**
- CI: `✓` pass, `✗` fail, `◐` pending, `—` none
- Review: `✓ 2/2` approved, `! 1/2` changes requested, `● 0/1` pending, `—` none
- `▸` = selected row, `[DRAFT]` prefix for drafts (dimmed styling)
- `◉` = current repo indicator, `⟳ 30s` = auto-refresh countdown

### 7.2 PR Detail View

```
┌─[ PR #142 ]───────────────────────────────────────────────────────────┐
│ Add plugin architecture                              indrasvat → main │
│ ─────────────────────────────────────────────────────────────────────  │
│                                                                       │
│ ┌─ Info ──────────────────────────────┐ ┌─ Checks ─────────────────┐ │
│ │ Branch: feat/plugin-arch → main     │ │ ✓ lint        12s        │ │
│ │ Labels: enhancement, architecture   │ │ ✓ test        45s        │ │
│ │ Assignees: indrasvat                │ │ ✓ build       23s        │ │
│ │ Reviewers: alice ✓, bob ●          │ │                           │ │
│ │ Created: 2h ago  Updated: 30m ago  │ │ All checks passing       │ │
│ └─────────────────────────────────────┘ └───────────────────────────┘ │
│                                                                       │
│ ┌─ Description ───────────────────────────────────────────────────┐   │
│ │ This PR introduces the plugin system for vivecaka.              │   │
│ │                                                                 │   │
│ │ ## Changes                                                      │   │
│ │ - Plugin registry with capability auto-discovery                │   │
│ │ - Hook system for lifecycle events                              │   │
│ │ - GH CLI adapter as first plugin                                │   │
│ │                                                                 │   │
│ │ ## Testing                                                      │   │
│ │ - Unit tests for registry, hooks                                │   │
│ │ - Integration test with mock plugin                             │   │
│ └─────────────────────────────────────────────────────────────────┘   │
│                                                                       │
│ ┌─ Files Changed (8) ─────────────────────────────────────────────┐   │
│ │ ▸ internal/plugin/registry.go       +142  -0                    │   │
│ │   internal/plugin/hooks.go          +87   -0                    │   │
│ │   internal/plugin/interfaces.go     +65   -0                    │   │
│ │   internal/adapter/ghcli/plugin.go  +203  -0                    │   │
│ │   internal/domain/pr.go             +12   -3                    │   │
│ │   ... 3 more files                                              │   │
│ └─────────────────────────────────────────────────────────────────┘   │
│                                                                       │
├───────────────────────────────────────────────────────────────────────┤
│ d diff  c checkout  r review  o open in browser  Tab switch  q back  │
└───────────────────────────────────────────────────────────────────────┘
```

### 7.3 Diff View (Unified)

```
┌─[ Diff: PR #142 ]────────────────────────────────────────────────────┐
│ ┌─ Files ──────────┐ ┌─ internal/plugin/registry.go ──────────────┐ │
│ │ ▸registry.go     │ │                                            │ │
│ │  hooks.go        │ │  @@ -0,0 +1,142 @@                        │ │
│ │  interfaces.go   │ │                                            │ │
│ │  plugin.go       │ │   1 + package plugin                       │ │
│ │  pr.go           │ │   2 +                                      │ │
│ │  config.go       │ │   3 + import (                             │ │
│ │  main.go         │ │   4 +     "context"                        │ │
│ │  go.mod          │ │   5 +     "fmt"                            │ │
│ │                  │ │   6 +     "sync"                           │ │
│ │                  │ │   7 + )                                    │ │
│ │                  │ │   8 +                                      │ │
│ │                  │ │   9 + // Registry manages plugin lifecycle │ │
│ │                  │ │  10 + type Registry struct {               │ │
│ │                  │ │  11 +     mu      sync.RWMutex             │ │
│ │                  │ │  12 +     plugins map[string]Plugin        │ │
│ │                  │ │  13 + }                                    │ │
│ │                  │ │  14 +                                      │ │
│ │                  │ │  15 + // Register adds a plugin.           │ │
│ │  ── 8 files ──   │ │  16 + func (r *Registry) Register(       │ │
│ │  +523 -12        │ │       ▼ 126 more lines                    │ │
│ └──────────────────┘ └────────────────────────────────────────────┘ │
│                                                                       │
│ ┌─ Comments (2) ──────────────────────────────────────────────────┐   │
│ │ alice (line 12): Should plugins be stored in insertion order?   │   │
│ │   └─ indrasvat: Good point, switching to ordered map.          │   │
│ └─────────────────────────────────────────────────────────────────┘   │
│                                                                       │
├───────────────────────────────────────────────────────────────────────┤
│ [/] hunks  {/} files  c comment  t toggle unified/split  e external │
└───────────────────────────────────────────────────────────────────────┘
```

### 7.4 Repo Switcher (Ctrl+R)

```
┌─[ Switch Repo ]──────────────────────────────────┐
│                                                    │
│  Search: ind▌                                      │
│                                                    │
│  ★ indrasvat/vivecaka          (current)           │
│  ★ indrasvat/dotfiles          12 open PRs         │
│    indrasvat/cli-tools          3 open PRs         │
│                                                    │
│  ★ = favorite    Enter to switch    Esc to cancel  │
└──────────────────────────────────────────────────┘
```

### 7.5 Review Form

```
┌─[ Submit Review: PR #142 ]────────────────────────┐
│                                                     │
│  Action:                                            │
│  ( ) Comment                                        │
│  (●) Approve                                        │
│  ( ) Request Changes                                │
│                                                     │
│  Body (optional):                                   │
│  ┌─────────────────────────────────────────────┐    │
│  │ LGTM! Plugin architecture looks clean.      │    │
│  │ One minor suggestion on the hook system -   │    │
│  │ consider adding priority ordering.          │    │
│  │                                             │    │
│  └─────────────────────────────────────────────┘    │
│                                                     │
│  [ Submit ]  [ Cancel ]                             │
│                                                     │
└─────────────────────────────────────────────────────┘
```

### 7.6 Filter Panel

```
┌─[ Filters ]──────────────────────────────────────┐
│                                                    │
│  Status:    [●] Open  [ ] Closed  [ ] Merged       │
│  Author:    _______________  (fuzzy match)         │
│  Label:     [●] enhancement  [ ] bug  [ ] docs     │
│  CI:        [●] All  [ ] Passing  [ ] Failing      │
│  Review:    [●] All  [ ] Approved  [ ] Pending      │
│  Draft:     [●] Include  [ ] Exclude  [ ] Only      │
│                                                    │
│  Quick:  [m] My PRs  [n] Needs My Review            │
│                                                    │
│  [ Apply ]  [ Reset ]  [ Cancel ]                   │
│                                                    │
└──────────────────────────────────────────────────┘
```

### 7.7 Help Overlay

```
┌─[ Help: PR List ]────────────────────────────────────────────────────┐
│                                                                       │
│  Navigation                    Actions                                │
│  ──────────                    ───────                                │
│  j/k      Move up/down        Enter    Open PR detail                │
│  g/G      Go to top/bottom    c        Checkout PR branch            │
│  Ctrl+d   Half page down      o        Open in browser               │
│  Ctrl+u   Half page up        r        Submit review                 │
│  /        Search PRs          y        Copy PR URL                   │
│  Esc      Clear search                                               │
│                                                                       │
│  Filtering                     Global                                 │
│  ─────────                     ──────                                 │
│  f        Open filter panel   Ctrl+R   Switch repo                   │
│  m        My PRs only         p        Toggle auto-refresh           │
│  n        Needs my review     T        Cycle theme                   │
│  s        Sort by column      ?        Toggle this help              │
│  R        Force refresh       q        Quit                          │
│                                                                       │
│                          Press ? or Esc to close                      │
└───────────────────────────────────────────────────────────────────────┘
```

---

## 8. User Flows

### Flow 1: Browse and Checkout a PR

```
Start vivecaka
    │
    ▼
┌─────────────┐     j/k        ┌──────────────┐
│  PR List    │ ──────────────▸ │  Navigate    │
│  (auto-load)│                 │  to PR #142  │
└─────────────┘                 └──────┬───────┘
                                       │ Enter
                                       ▼
                                ┌──────────────┐
                                │  PR Detail   │
                                │  View        │
                                └──────┬───────┘
                                       │ c
                                       ▼
                                ┌──────────────┐
                                │  Checkout    │
                                │  Confirm     │
                                └──────┬───────┘
                                       │ Enter
                                       ▼
                                ┌──────────────┐
                                │  ✓ Checked   │
                                │  out to      │
                                │  feat/plugin │
                                └──────────────┘
```

### Flow 2: Review a PR with Inline Comments

```
PR Detail View
    │ d
    ▼
┌──────────────┐
│  Diff View   │ ── navigate to file/line ──┐
│  (built-in)  │                             │
└──────────────┘                             │ c (on line)
                                             ▼
                                      ┌──────────────┐
                                      │  Comment     │
                                      │  Editor      │
                                      └──────┬───────┘
                                             │ Ctrl+S
                                             ▼
                                      ┌──────────────┐
                                      │  Comment     │
                                      │  Submitted   │
                                      └──────────────┘
    ... review more files ...
    │ q (back to detail)
    ▼
PR Detail View
    │ r
    ▼
┌──────────────┐
│  Review Form │
│  Approve/    │
│  Req Changes │
└──────┬───────┘
       │ Submit
       ▼
┌──────────────┐
│  ✓ Review    │
│  Submitted   │
└──────────────┘
```

### Flow 3: Multi-Repo Quick Switch

```
Any View
    │ Ctrl+R
    ▼
┌──────────────┐
│  Repo Picker │ ── type to fuzzy search ──┐
│  (favorites) │                            │
└──────────────┘                            │ Enter
                                            ▼
                                     ┌──────────────┐
                                     │  PR List for │
                                     │  new repo    │
                                     │  (fresh load)│
                                     └──────────────┘
```

### Flow 4: Filter and Sort PRs

```
PR List View
    │ f
    ▼
┌──────────────┐
│  Filter      │ ── toggle options ──┐
│  Panel       │                      │
└──────────────┘                      │ Apply
                                      ▼
                               ┌──────────────┐
                               │  PR List     │
                               │  (filtered)  │
                               │  Badge: 3    │
                               │  filters     │
                               └──────────────┘
```

---

## 9. Technical Specifications

### 9.1 Project Structure

```
vivecaka/
├── cmd/
│   └── vivecaka/
│       └── main.go                     # Entry point, DI wiring
├── internal/
│   ├── domain/                         # Core entities & interfaces
│   │   ├── pr.go                       # PR, PRDetail, Diff, Check entities
│   │   ├── review.go                   # Review, Comment, Thread entities
│   │   ├── repo.go                     # RepoRef, RepoConfig
│   │   ├── errors.go                   # Sentinel errors
│   │   └── interfaces.go              # PRReader, PRReviewer, PRWriter
│   ├── usecase/                        # Application logic
│   │   ├── list_prs.go                 # ListPRs use case
│   │   ├── get_pr.go                   # GetPRDetail use case
│   │   ├── get_diff.go                 # GetDiff use case
│   │   ├── review_pr.go               # SubmitReview use case
│   │   ├── checkout_pr.go             # Checkout use case
│   │   └── comment.go                 # Add/resolve comment use cases
│   ├── adapter/
│   │   └── ghcli/                      # GH CLI adapter (MVP)
│   │       ├── plugin.go              # Plugin implementation
│   │       ├── reader.go              # PRReader implementation
│   │       ├── reviewer.go            # PRReviewer implementation
│   │       ├── writer.go              # PRWriter implementation
│   │       └── parser.go             # JSON output parsing
│   ├── plugin/                         # Plugin infrastructure
│   │   ├── interfaces.go             # Plugin, ViewPlugin, KeyPlugin
│   │   ├── registry.go               # Plugin registry
│   │   └── hooks.go                  # Hook manager
│   ├── tui/                            # BubbleTea UI
│   │   ├── app.go                     # Root model, message routing
│   │   ├── keymap.go                  # Global keymap definitions
│   │   ├── theme.go                   # Theme definitions & loading
│   │   ├── styles.go                  # LipGloss style helpers
│   │   ├── views/                     # View models
│   │   │   ├── prlist.go             # PR list view
│   │   │   ├── prdetail.go           # PR detail view
│   │   │   ├── diff.go               # Built-in diff viewer
│   │   │   ├── review.go             # Review form
│   │   │   ├── filter.go             # Filter panel
│   │   │   ├── reposwitcher.go       # Repo switcher overlay
│   │   │   └── help.go               # Help overlay
│   │   └── components/               # Reusable UI components
│   │       ├── table.go              # Styled table component
│   │       ├── statusbar.go          # Bottom status bar
│   │       ├── header.go             # Top header bar
│   │       ├── toast.go              # Notification toasts
│   │       ├── spinner.go            # Loading spinner
│   │       ├── badge.go              # Status badges (CI, review)
│   │       └── confirm.go            # Confirmation dialog
│   └── config/                        # Configuration
│       ├── config.go                  # Config struct & loading
│       ├── defaults.go               # Default values
│       └── xdg.go                    # XDG path helpers
├── docs/
│   ├── PRD.md                         # This document
│   └── PROGRESS.md                    # Agent progress tracking
├── CLAUDE.md                          # Agent conventions & guardrails
├── Makefile                           # Build, test, lint targets
├── lefthook.yml                       # Git hooks config
├── .goreleaser.yaml                   # Release automation
├── .golangci.yml                      # Linter config
├── go.mod
├── go.sum
├── LICENSE                            # MIT
└── README.md
```

### 9.2 Domain Entities

```go
// RepoRef identifies a GitHub repository.
type RepoRef struct {
    Owner string
    Name  string
}

func (r RepoRef) String() string { return r.Owner + "/" + r.Name }

// PR is the list-level representation of a pull request.
type PR struct {
    Number    int
    Title     string
    Author    string
    State     PRState      // Open, Closed, Merged
    Draft     bool
    Branch    BranchInfo
    Labels    []string
    CI        CIStatus     // Pass, Fail, Pending, None
    Review    ReviewStatus // Approved, ChangesRequested, Pending, None
    UpdatedAt time.Time
    CreatedAt time.Time
    URL       string

    // Activity tracking (for S4 inbox: unread indicators + priority sort)
    LastViewedAt  *time.Time // nil = never viewed; persisted in XDG data dir
    LastActivityAt time.Time // most recent comment/review/push; from GitHub
}

type PRState string
const (
    PRStateOpen   PRState = "open"
    PRStateClosed PRState = "closed"
    PRStateMerged PRState = "merged"
)

type CIStatus string
const (
    CIPass    CIStatus = "pass"
    CIFail    CIStatus = "fail"
    CIPending CIStatus = "pending"
    CISkipped CIStatus = "skipped" // gray in UI (neutral/skipped checks)
    CINone    CIStatus = "none"
)

type ReviewStatus struct {
    State    ReviewState
    Approved int
    Total    int
}

type ReviewState string
const (
    ReviewApproved         ReviewState = "approved"
    ReviewChangesRequested ReviewState = "changes_requested"
    ReviewPending          ReviewState = "pending"
    ReviewNone             ReviewState = "none"
)

// PRDetail is the full representation for the detail view.
type PRDetail struct {
    PR
    Body      string          // Markdown body
    Assignees []string
    Reviewers []ReviewerInfo
    Checks    []Check
    Files     []FileChange
    Comments  []CommentThread
}

type BranchInfo struct {
    Head string
    Base string
}

type ReviewerInfo struct {
    Login string
    State ReviewState
}

type Check struct {
    Name     string
    Status   CIStatus
    Duration time.Duration
    URL      string
}

type FileChange struct {
    Path      string
    Additions int
    Deletions int
    Status    string // added, modified, removed, renamed
}

// Diff represents the diff content for a PR.
type Diff struct {
    Files []FileDiff
}

type FileDiff struct {
    Path    string
    Hunks   []Hunk
    OldPath string // for renames
}

type Hunk struct {
    Header  string
    Lines   []DiffLine
}

type DiffLine struct {
    Type    DiffLineType // Add, Delete, Context
    Content string
    OldNum  int
    NewNum  int
}

type DiffLineType string
const (
    DiffAdd     DiffLineType = "add"
    DiffDelete  DiffLineType = "delete"
    DiffContext DiffLineType = "context"
)

// CommentThread represents a review comment thread.
type CommentThread struct {
    ID        string
    Path      string
    Line      int
    Resolved  bool
    Comments  []Comment
}

type Comment struct {
    ID        string
    Author    string
    Body      string
    CreatedAt time.Time
}

// InlineCommentInput is used when creating a new inline comment on a diff line.
// Separate from Comment (which represents an existing comment in a thread).
type InlineCommentInput struct {
    Path      string // File path relative to repo root
    Line      int    // Line number in the diff
    Side      string // "LEFT" or "RIGHT" (for side-by-side)
    Body      string // Comment body (markdown)
    CommitID  string // SHA of the commit to comment on
    InReplyTo string // Thread ID if replying (empty for new thread)
}

// Review represents a review submission.
type Review struct {
    Action ReviewAction
    Body   string
}

type ReviewAction string
const (
    ReviewActionApprove        ReviewAction = "approve"
    ReviewActionRequestChanges ReviewAction = "request_changes"
    ReviewActionComment        ReviewAction = "comment"
)

// ListOpts controls PR list filtering and pagination.
type ListOpts struct {
    State    PRState      // Filter by state (empty = all)
    Author   string       // Filter by author (empty = all)
    Labels   []string     // Filter by labels (empty = all)
    CI       CIStatus     // Filter by CI status (empty = all)
    Review   ReviewState  // Filter by review state (empty = all)
    Draft    DraftFilter  // Filter by draft status (default: Include)
    Search   string       // Full-text search across title/body
    Sort     string       // Sort field: "updated", "created", "number", "title", "author"
    SortDesc bool         // Sort descending (default true)
    Page     int          // Page number (1-based)
    PerPage  int          // Items per page
}

// DraftFilter controls how draft PRs are included in results.
type DraftFilter string
const (
    DraftInclude DraftFilter = "include" // Show all PRs (default)
    DraftExclude DraftFilter = "exclude" // Hide drafts
    DraftOnly    DraftFilter = "only"    // Show only drafts
)

// MergeOpts controls PR merge behavior.
// NOTE: Merge is defined in PRWriter for plugin extensibility but is
// NOT exposed in the MVP UI. The MVP only uses Checkout from PRWriter.
type MergeOpts struct {
    Method        string // "merge", "squash", "rebase"
    DeleteBranch  bool   // Delete head branch after merge
    CommitMessage string // Custom merge commit message (empty = default)
}

// ViewRegistration describes a custom view provided by a plugin.
type ViewRegistration struct {
    Name     string    // Unique view name
    Title    string    // Display title
    Position string    // "tab", "overlay", "pane"
    Model    tea.Model // BubbleTea model for the view
}

// KeyRegistration describes a custom key binding provided by a plugin.
type KeyRegistration struct {
    Key    key.Binding    // The key binding (from bubbles/key)
    View   string         // Which view this applies to ("" = global)
    Action func() tea.Cmd // Action to execute
}

// AppContext provides plugins access to application state during Init().
// NOTE: AppContext lives in internal/plugin/ (not domain), so it may reference
// TUI types. Plugin interfaces intentionally bridge domain and TUI layers.
type AppContext interface {
    ConfigValue(key string) any // Access config by key (avoids config package dep)
    ThemeName() string          // Current theme name
    CurrentRepo() RepoRef       // Active repository
    SendMessage(tea.Msg)        // Send message to TUI event loop
}

// ToastLevel indicates the severity of a toast notification.
type ToastLevel string
const (
    ToastInfo    ToastLevel = "info"
    ToastSuccess ToastLevel = "success"
    ToastWarning ToastLevel = "warning"
    ToastError   ToastLevel = "error"
)
```

### 9.3 GH CLI Adapter

```go
// The ghcli adapter uses github.com/cli/go-gh/v2 to interact with
// the authenticated gh CLI. It implements PRReader, PRReviewer, PRWriter.

// Key dependency: github.com/cli/go-gh/v2 (v2.13.0+)
//
// Authentication: Automatically uses gh's stored credentials.
// If not authenticated, returns a clear error prompting `gh auth login`.

// ListPRs executes: gh pr list --json <fields> --repo owner/name
// GetPR executes: gh pr view <number> --json <fields> --repo owner/name
// GetDiff executes: gh pr diff <number> --repo owner/name
// GetChecks executes: gh pr checks <number> --json <fields> --repo owner/name
// SubmitReview: gh pr review <number> --approve|--request-changes|--comment --body
// Checkout: gh pr checkout <number>
// AddComment: gh api repos/{owner}/{repo}/pulls/{number}/comments (REST)
//   NOTE: Inline comments require commit_id, path, position/line, side fields.
//   The adapter must resolve these from the InlineCommentInput + current diff state.
// GetComments: gh api repos/{owner}/{repo}/pulls/{number}/comments (REST, paginated)
// ResolveThread: gh api /graphql (minimizeComment mutation - verify availability)
//
// IMPLEMENTATION NOTE: Some operations (resolve thread, inline comments) may require
// the GitHub REST or GraphQL API via `gh api`. Verify exact API support during T4.x.
```

### 9.4 Key Dependencies

| Dependency | Version | Purpose |
|-----------|---------|---------|
| `github.com/charmbracelet/bubbletea` | v1.3.x | TUI framework |
| `github.com/charmbracelet/lipgloss` | v1.1.x | Styling |
| `github.com/charmbracelet/bubbles` | v0.21.x | UI components (list, viewport, textinput, help, spinner, key) |
| `github.com/charmbracelet/huh` | v0.8.x | Form building (review form, filters) |
| `github.com/charmbracelet/glamour` | v0.8.x | Markdown rendering (PR body, comments) |
| `github.com/cli/go-gh/v2` | v2.13.x | GH CLI integration |
| `github.com/pelletier/go-toml/v2` | v2.2.x | Config parsing |
| `github.com/alecthomas/chroma/v2` | v2.14.x | Syntax highlighting in diffs |
| `github.com/stretchr/testify` | v1.10.x | Test assertions |

### 9.5 Message Types (BubbleTea)

```go
// Data messages (from commands)
type PRsLoadedMsg struct { PRs []domain.PR; Err error }
type PRDetailLoadedMsg struct { Detail *domain.PRDetail; Err error }
type DiffLoadedMsg struct { Diff *domain.Diff; Err error }
type ChecksLoadedMsg struct { Checks []domain.Check; Err error }
type ReviewSubmittedMsg struct { Err error }
type CheckoutCompleteMsg struct { Branch string; Err error }
type CommentPostedMsg struct { Err error }
type AutoRefreshMsg struct {}

// Navigation messages
type NavigateToDetailMsg struct { Number int }
type NavigateToDiffMsg struct { Number int }
type NavigateBackMsg struct {}
type SwitchRepoMsg struct { Repo domain.RepoRef }

// UI state messages
type FilterChangedMsg struct { Filter ListOpts }
type ThemeChangedMsg struct { Theme string }
type ErrorMsg struct { Err error; Transient bool }
type ToastMsg struct { Text string; Level ToastLevel }
```

---

## 10. Configuration

### 10.1 Config File Location

```
~/.config/vivecaka/config.toml
```

Falls back to XDG_CONFIG_HOME if set. Created with defaults on first run.

### 10.2 Default Config

```toml
# vivecaka configuration

[general]
# Theme name (charmbracelet themes)
theme = "default-dark"
# Auto-refresh interval in seconds (0 to disable)
refresh_interval = 30
# Default sort: "updated", "created", "number"
default_sort = "updated"
# Default filter: "open", "closed", "merged", "all" (maps to PRState; "all" = empty/no filter)
default_filter = "open"
# Number of PRs per page
page_size = 50
# Show startup banner on launch (true/false)
show_banner = true
# Cache TTL in minutes for PR data (0 = no cache)
cache_ttl = 5
# Number of days without activity before a PR is considered "stale" (for priority sort)
stale_days = 7

[diff]
# Built-in diff mode: "unified" or "split"
mode = "unified"
# External diff tool (blank = disabled)
external_tool = ""
# Show line numbers
line_numbers = true
# Context lines around hunks
context_lines = 3
# Markdown rendering style for PR body/comments: "dark", "light", "notty"
markdown_style = "dark"

[repos]
# Favorite repos for quick switching
favorites = [
    # "owner/repo",
]

[keybindings]
# Override default keybindings
# Format: "action" = "key"
# Example: navigate_down = "j"
# See `vivecaka --keys` for all available actions

[notifications]
# Show toast for new PRs
new_prs = true
# Show toast for review requests
review_requests = true
# Show toast for CI status changes
ci_changes = true
```

---

## 11. Task Breakdown

Each task has: description, acceptance criteria, test requirements, verification steps, and estimated complexity (S/M/L).

### Phase 0: Project Bootstrap (Foundation)

#### T0.1: Initialize Go Module and Project Structure
**Description:** Create `go.mod`, directory structure, and empty packages per the architecture.
**Acceptance Criteria:**
- `go mod init github.com/indrasvat/vivecaka` succeeds
- All directories from section 9.1 exist
- Each package has at least one `.go` file with correct package declaration
- `go build ./...` succeeds (no compilation errors)
**Test:** `go build ./...` && `go vet ./...`
**Verification:** Run `tree` and compare against section 9.1
**Complexity:** S

#### T0.2: Create Comprehensive Makefile
**Description:** Create self-documenting Makefile with all standard targets.
**Acceptance Criteria:**
- `make help` prints formatted target list with descriptions
- Targets: `build`, `run`, `test`, `lint`, `fmt`, `vet`, `clean`, `deps`, `coverage`, `ci`, `install`, `dev`, `hooks-install`, `hooks-uninstall`, `snapshot`
- `make ci` runs: `fmt`, `vet`, `lint`, `test`, `build` in sequence
- `make build` produces `bin/vivecaka` binary
- `make dev` runs with `go run`
- Version info injected via ldflags
**Test:** Each make target runs without error
**Verification:** `make help` shows all targets; `make ci` passes
**Complexity:** S

#### T0.3: Create CLAUDE.md
**Description:** Create the agent conventions document.
**Acceptance Criteria:**
- Located at project root
- Covers: commit conventions, make usage, testing requirements, progress tracking
- References all relevant make targets
- Includes instructions for maintaining docs/PROGRESS.md
**Test:** File exists and is well-structured markdown
**Verification:** Read and confirm completeness
**Complexity:** S

#### T0.4: Create docs/PROGRESS.md Template
**Description:** Create the live progress tracking document.
**Acceptance Criteria:**
- Template with sections: Current Phase, Completed Tasks, In Progress, Blocked, Notes
- Updated by agents after each task completion
**Test:** File exists with correct template
**Verification:** Read and confirm structure
**Complexity:** S

#### T0.5: Setup Lefthook
**Description:** Configure lefthook for pre-push hooks.
**Acceptance Criteria:**
- `lefthook.yml` at project root
- Pre-push hook runs `make ci`
- Pre-commit hook runs `make fmt` and `make vet`
- `make hooks-install` works
**Test:** `lefthook run pre-push` succeeds
**Verification:** Make an intentional lint error, verify hook catches it
**Complexity:** S

#### T0.6: Setup golangci-lint Config
**Description:** Create `.golangci.yml` with appropriate linters.
**Acceptance Criteria:**
- Linters enabled: govet, errcheck, staticcheck, gosimple, unused, ineffassign, gocritic, gofmt, misspell
- Reasonable timeout (3min)
- Exclusions for test files where appropriate
**Test:** `make lint` runs successfully
**Verification:** Introduce lint violation, verify it's caught
**Complexity:** S

#### T0.7: Setup GoReleaser Config
**Description:** Create `.goreleaser.yaml` for release automation.
**Acceptance Criteria:**
- Builds for linux/darwin (amd64/arm64) and windows (amd64)
- Version injection via ldflags
- Changelog generation from conventional commits
- `make snapshot` produces local build via goreleaser
**Test:** `goreleaser check` passes; `make snapshot` produces binaries
**Verification:** Inspect generated binaries for correct version info
**Complexity:** S

#### T0.8: Create LICENSE and Update README
**Description:** Add MIT license and expand README with project description.
**Acceptance Criteria:**
- MIT LICENSE file
- README with: description, features, installation, usage, configuration, contributing
- ASCII art logo or clean header
**Test:** Files exist
**Verification:** Read and confirm content
**Complexity:** S

---

### Phase 1: Domain Layer

#### T1.1: Define Domain Entities
**Description:** Implement all structs from section 9.2 in `internal/domain/`.
**Acceptance Criteria:**
- All types from section 9.2 are defined
- Types are well-documented with godoc comments
- JSON tags on all fields (for serialization)
- Stringer interfaces where useful (RepoRef, PRState, etc.)
**Test:** `go build ./internal/domain/...`; unit test for Stringer methods
**Verification:** `go doc ./internal/domain/` shows clean docs
**Complexity:** S

#### T1.2: Define Domain Interfaces
**Description:** Implement PRReader, PRReviewer, PRWriter interfaces.
**Acceptance Criteria:**
- Interfaces defined per section 4 Plugin System
- Small, focused interfaces (ISP)
- Context as first parameter on all methods
- Error return on all methods
**Test:** `go vet ./internal/domain/...`
**Verification:** Review interfaces for ISP compliance
**Complexity:** S

#### T1.3: Define Domain Errors
**Description:** Create sentinel errors and custom error types.
**Acceptance Criteria:**
- Sentinel errors: `ErrNotFound`, `ErrUnauthorized`, `ErrNotAuthenticated`, `ErrRateLimited`
- Custom type: `ValidationError` with field/message
- All errors implement `error` interface
- Errors support `errors.Is()` and `errors.As()`
**Test:** Unit tests verifying `errors.Is()` / `errors.As()` behavior
**Verification:** Tests pass
**Complexity:** S

---

### Phase 2: Plugin Infrastructure

#### T2.1: Implement Plugin Interfaces
**Description:** Create base plugin interface and capability interfaces in `internal/plugin/`.
**Acceptance Criteria:**
- Plugin, PluginInfo interfaces defined
- ViewPlugin, KeyPlugin interfaces defined
- AppContext interface for plugin initialization
**Test:** `go build ./internal/plugin/...`
**Verification:** Interfaces match section 4 design
**Complexity:** S

#### T2.2: Implement Plugin Registry
**Description:** Create thread-safe plugin registry with auto-discovery.
**Acceptance Criteria:**
- Register/Unregister plugins by name
- Auto-discover capabilities via type assertion
- Thread-safe (sync.RWMutex)
- Duplicate name detection returns error
- GetReaders(), GetReviewers(), GetWriters() accessors
**Test:** Table-driven tests: register, duplicate, unregister, get by capability
**Verification:** Tests pass with race detector (`-race`)
**Complexity:** M

#### T2.3: Implement Hook System
**Description:** Create event hook manager.
**Acceptance Criteria:**
- HookPoint constants defined
- On() registers handlers
- Emit() calls all handlers in order
- Error in handler stops chain and returns error
- Thread-safe
**Test:** Unit tests: register handlers, emit, error propagation, ordering
**Verification:** Tests pass with race detector
**Complexity:** M

---

### Phase 3: Configuration

#### T3.1: Implement Config Struct and Loading
**Description:** Create config types and TOML loading in `internal/config/`.
**Acceptance Criteria:**
- Config struct matching section 10.2
- Load from XDG config path
- Create default config on first run
- Merge user config over defaults
- Validate config values (e.g., refresh_interval >= 0)
**Test:** Unit tests: load defaults, load custom, merge, validate invalid
**Verification:** Tests pass; run binary and verify `~/.config/vivecaka/config.toml` created
**Complexity:** M

#### T3.2: Implement XDG Path Helpers
**Description:** XDG-compliant path resolution.
**Acceptance Criteria:**
- ConfigDir() returns `~/.config/vivecaka` (or XDG_CONFIG_HOME)
- DataDir() returns `~/.local/share/vivecaka` (or XDG_DATA_HOME)
- CacheDir() returns `~/.cache/vivecaka` (or XDG_CACHE_HOME)
- Directories created automatically if missing
**Test:** Unit tests with mocked env vars
**Verification:** Tests pass
**Complexity:** S

---

### Phase 4: GH CLI Adapter

#### T4.1: Implement GH CLI Plugin Wrapper
**Description:** Create the ghcli plugin struct implementing Plugin + capabilities.
**Acceptance Criteria:**
- Implements Plugin interface (Info, Init)
- Detects if `gh` CLI is installed and authenticated
- Returns clear error if not authenticated (prompting `gh auth login`)
- Uses `go-gh` v2 library
**Test:** Unit test with mocked `gh.Exec`; integration test with real `gh`
**Verification:** Binary starts and detects gh auth status
**Complexity:** M

#### T4.2: Implement PRReader (GH CLI)
**Description:** Implement ListPRs, GetPR, GetDiff, GetChecks via `gh` CLI.
**Acceptance Criteria:**
- ListPRs: `gh pr list --json` with all needed fields, pagination
- GetPR: `gh pr view --json` with full detail fields
- GetDiff: `gh pr diff` raw output, parsed into domain.Diff
- GetChecks: `gh pr checks --json`
- All methods respect context cancellation
- Proper error wrapping with context
**Test:** Unit tests with captured `gh` output fixtures (JSON files in testdata/)
**Verification:** Tests pass; manual test against real repo
**Complexity:** L

#### T4.3: Implement PRReviewer (GH CLI)
**Description:** Implement SubmitReview, AddComment, ResolveThread via `gh` CLI.
**Acceptance Criteria:**
- SubmitReview: `gh pr review` with approve/request-changes/comment
- AddComment: `gh api` for inline review comments
- ResolveThread: `gh api` for resolving comment threads
- All methods respect context cancellation
**Test:** Unit tests with mocked `gh` execution
**Verification:** Tests pass; manual test submitting a review
**Complexity:** M

#### T4.4: Implement PRWriter (GH CLI)
**Description:** Implement Checkout via `gh` CLI.
**Acceptance Criteria:**
- Checkout: `gh pr checkout <number>`
- Handles existing branch gracefully
- Returns branch name on success
**Test:** Unit tests with mocked `gh` execution
**Verification:** Tests pass; manual test checking out a PR
**Complexity:** S

#### T4.5: Implement Diff Parser
**Description:** Parse unified diff output into domain.Diff structs.
**Acceptance Criteria:**
- Parses standard unified diff format
- Handles: additions, deletions, context lines, renames, binary files
- Preserves line numbers (old and new)
- Handles large diffs (streaming/lazy if needed)
**Test:** Table-driven tests with various diff fixtures (added files, deleted, renamed, binary, large)
**Verification:** Tests pass with diverse diff samples
**Complexity:** M

---

### Phase 5: Use Cases

#### T5.1: Implement ListPRs Use Case
**Description:** Orchestrate PR listing with filtering and sorting.
**Acceptance Criteria:**
- Accepts ListOpts (filter, sort, page)
- Delegates to PRReader from registry
- Applies client-side filtering for fields not supported by backend
- Returns sorted, filtered PR list
**Test:** Unit tests with mock PRReader
**Verification:** Tests pass
**Complexity:** S

#### T5.2: Implement GetPRDetail Use Case
**Description:** Fetch full PR detail with all associated data.
**Acceptance Criteria:**
- Fetches PR detail, checks, and comments in parallel (errgroup, NOT tea.Batch - use cases are UI-agnostic)
- Aggregates into single PRDetail response
- Handles partial failures gracefully (e.g., checks fail but PR loads)
**Test:** Unit tests with mock, including partial failure scenarios
**Verification:** Tests pass
**Complexity:** M

#### T5.3: Implement ReviewPR Use Case
**Description:** Submit review through registered reviewer.
**Acceptance Criteria:**
- Validates review action and body
- Delegates to PRReviewer from registry
- Returns success/error
**Test:** Unit tests with mock PRReviewer
**Verification:** Tests pass
**Complexity:** S

#### T5.4: Implement CheckoutPR Use Case
**Description:** Checkout PR branch via registered writer.
**Acceptance Criteria:**
- Delegates to PRWriter from registry
- Returns branch name on success
**Test:** Unit tests with mock PRWriter
**Verification:** Tests pass
**Complexity:** S

#### T5.5: Implement Comment Use Cases
**Description:** Add comment and resolve thread use cases.
**Acceptance Criteria:**
- AddComment validates required fields (path, line, body)
- ResolveThread delegates to PRReviewer
**Test:** Unit tests with mock
**Verification:** Tests pass
**Complexity:** S

---

### Phase 6: TUI Foundation

#### T6.1: Implement Theme System
**Description:** Create theme definitions and loading.
**Acceptance Criteria:**
- Theme struct with all semantic colors (primary, secondary, success, error, warning, info, muted, border, etc.)
- Built-in themes: `default-dark`, `catppuccin-mocha`, `catppuccin-frappe`, `tokyo-night`, `dracula`
- Theme loaded from config
- AdaptiveColor support for terminal background detection
- Styles helper functions using current theme
**Test:** Unit tests: theme loading, style generation
**Verification:** Visual check with iterm2-driver showing each theme
**Complexity:** M

#### T6.2: Implement Keymap System
**Description:** Create global and view-specific keymaps.
**Acceptance Criteria:**
- Global keymap: quit, help, repo-switch, theme-cycle, refresh-toggle
- Per-view keymaps: PR list, PR detail, diff, review, filter
- Uses `bubbles/key` package
- Implements `help.KeyMap` interface for help display
- Keybindings overridable via config
**Test:** Unit tests: key matching, help text generation
**Verification:** Help overlay shows correct bindings
**Complexity:** M

#### T6.3: Implement Root App Model
**Description:** Create the main BubbleTea model with view routing.
**Acceptance Criteria:**
- Root model manages current view state (enum)
- Routes messages to active view
- Handles global keys (quit, help, repo-switch)
- Layout: header + content + status bar
- Proper Init() fetching initial PR list
- Window resize handling
**Test:** Unit test: message routing, view switching
**Verification:** Binary starts and shows loading state; iterm2-driver screenshot
**Complexity:** L

#### T6.4: Implement Header Component
**Description:** Top bar showing repo, PR count, active filter, refresh status.
**Acceptance Criteria:**
- Shows: repo name (◉ icon), open PR count, active filter name, refresh countdown
- Responsive: truncates gracefully on narrow terminals
- Uses theme colors
**Test:** Unit test: render with various widths
**Verification:** iterm2-driver screenshot at different terminal widths
**Complexity:** S

#### T6.5: Implement Status Bar Component
**Description:** Bottom bar showing context-aware key hints and notifications.
**Acceptance Criteria:**
- Shows key hints for current view (condensed format)
- Shows transient toast notifications (auto-dismiss)
- Shows errors with red styling
- Shows success messages with green styling
**Test:** Unit test: render with various states
**Verification:** iterm2-driver screenshot
**Complexity:** S

#### T6.6: Implement Toast Notification Component
**Description:** Non-blocking notification overlay.
**Acceptance Criteria:**
- Appears at top-right of screen
- Auto-dismisses after configurable duration (default 3s)
- Supports levels: info, success, warning, error
- Stacks multiple toasts
- Does not block interaction
**Test:** Unit test: toast creation, dismissal timing
**Verification:** iterm2-driver screenshot showing toast appearing/dismissing
**Complexity:** M

---

### Phase 7: PR List View

#### T7.1: Implement PR List View Model
**Description:** Full PR list view with table, filtering, sorting, and selection.
**Acceptance Criteria:**
- Table with columns: #, Title, Author, CI, Review, Age
- Column widths adapt to terminal width
- Selected row highlighted
- j/k navigates, Enter opens detail
- Draft PRs visually dimmed
- Empty state message when no PRs match
- Loading spinner during fetch
**Test:** Unit test: navigation, selection, message handling
**Verification:** iterm2-driver: launch, navigate, verify alignment
**Complexity:** L

#### T7.2: Implement PR List Filtering
**Description:** Filter panel for the PR list.
**Acceptance Criteria:**
- `f` opens filter overlay
- Filter options per section 7.6 mock
- Quick filters: `m` (my PRs), `n` (needs my review)
- Active filter badge in header
- `/` for text search across title/author
- Esc clears search
**Test:** Unit test: filter application, search, clear
**Verification:** iterm2-driver: open filter, apply, verify list updates
**Complexity:** M

#### T7.3: Implement PR List Sorting
**Description:** Column-based sorting.
**Acceptance Criteria:**
- `s` opens sort selector (or direct column click)
- Sort options: updated, created, number, title, author
- Sort direction toggle (asc/desc)
- Visual indicator on sorted column (▲/▼)
**Test:** Unit test: sort application, direction toggle
**Verification:** iterm2-driver: sort by different columns, verify order
**Complexity:** S

#### T7.4: Implement Current Branch Highlight (S1)
**Description:** Auto-detect if the user's current git branch corresponds to an open PR and highlight it.
**Acceptance Criteria:**
- Detect current branch via `git rev-parse --abbrev-ref HEAD`
- If a PR exists for the current branch, add a visual indicator (◉ or distinct color)
- Highlight persists across refresh cycles
- Works correctly after checkout changes the branch
**Test:** Unit test: branch detection, PR matching, highlight rendering
**Verification:** iterm2-driver: checkout a PR branch, verify it's highlighted in list
**Complexity:** S

#### T7.5: Implement Per-Repo Filter Memory (S1)
**Description:** Remember last-used filter and sort settings per repo.
**Acceptance Criteria:**
- Store filter/sort state per RepoRef in XDG data directory
- Restore filter/sort when switching back to a repo
- Clear stored filters with a "Reset" action
- Data persists across app restarts
**Test:** Unit test: save/load filter state, per-repo isolation
**Verification:** Set filter on repo A, switch to repo B, switch back, verify filter restored
**Complexity:** S

#### T7.6: Implement Clipboard Integration
**Description:** Copy PR URL to clipboard with `y` key.
**Acceptance Criteria:**
- `y` on selected PR copies its URL to system clipboard
- Uses Go clipboard library (e.g., `golang.design/x/clipboard` or `atotto/clipboard`)
- Toast notification: "Copied PR URL"
- Graceful degradation if clipboard is unavailable (e.g., headless SSH)
**Test:** Unit test: message handling for copy action
**Verification:** Press `y`, paste in another app, verify URL matches
**Complexity:** S

#### T7.7: Implement Auto-Refresh
**Description:** Background polling for PR list updates.
**Acceptance Criteria:**
- Ticks every `refresh_interval` seconds
- Fetches fresh PR list in background
- Merges updates without losing scroll position
- Visual indicator: new/updated PRs get dot badge
- `p` toggles pause/resume
- Countdown timer in header
**Test:** Unit test: tick handling, merge logic, pause/resume
**Verification:** iterm2-driver: observe refresh indicator, verify list updates
**Complexity:** M

---

### Phase 8: PR Detail View

#### T8.1: Implement PR Detail View Model
**Description:** Full PR detail view with metadata, description, files, checks.
**Acceptance Criteria:**
- Layout per section 7.2 mock
- Info pane: branch, labels, assignees, reviewers with status
- Description pane: markdown rendered via Glamour
- Checks pane: CI status with color coding
- Files pane: file list with +/- counts
- Tab to cycle between panes
- q to go back to list
**Test:** Unit test: pane navigation, render with mock data
**Verification:** iterm2-driver: navigate to detail, Tab between panes
**Complexity:** L

#### T8.2: Implement Checks Display
**Description:** CI checks component within detail view.
**Acceptance Criteria:**
- Shows each check: name, status icon, duration
- Color coding: green ✓, red ✗, yellow ◐, gray —
- Summary line: "3/3 passing" or "1 failing"
- `o` opens check URL in browser
**Test:** Unit test: render with various check states
**Verification:** iterm2-driver: view PR with mixed check statuses
**Complexity:** S

#### T8.3: Implement Markdown Rendering
**Description:** Render PR body and comments as rich markdown.
**Acceptance Criteria:**
- Uses Glamour with theme-matched style
- Renders: headers, bold, italic, code blocks, lists, links, tables
- Code blocks with syntax highlighting
- Links rendered as clickable (OSC 8 hyperlinks)
- Graceful fallback for unsupported terminals
**Test:** Unit test: render sample markdown
**Verification:** iterm2-driver: view PR with complex markdown body
**Complexity:** M

---

### Phase 9: Diff Viewer

#### T9.1: Implement Built-in Diff View Model
**Description:** Two-pane diff viewer (file tree + diff content).
**Acceptance Criteria:**
- Left pane: file tree with selection
- Right pane: diff content for selected file
- Unified mode (default)
- Line numbers with +/- indicators
- Color: green for additions, red for deletions
- hjkl scrolls diff, {/} navigates files, [/] navigates hunks
- Syntax highlighting via chroma
**Test:** Unit test: navigation, file switching, hunk jumping
**Verification:** iterm2-driver: open diff, navigate files/hunks, verify colors
**Complexity:** L

#### T9.2: Implement Side-by-Side Diff Mode
**Description:** Split diff view as toggle from unified.
**Acceptance Criteria:**
- `t` toggles between unified and split
- Split shows old on left, new on right
- Synchronized scrolling between panes
- Line numbers on both sides
**Test:** Unit test: toggle, synchronized scroll
**Verification:** iterm2-driver: toggle mode, verify alignment
**Complexity:** M

#### T9.3: Implement External Diff Delegation
**Description:** Open diff in external tool.
**Acceptance Criteria:**
- `e` opens current file's diff in configured external tool
- Supports: delta, difftastic, VS Code
- Falls back to system diff if no tool configured
- Returns to TUI after external tool exits
**Test:** Unit test: command construction for various tools
**Verification:** Manual test with delta/difftastic installed
**Complexity:** M

#### T9.4: Implement Inline Comments in Diff
**Description:** View and add comments on diff lines.
**Acceptance Criteria:**
- Lines with existing comments highlighted (subtle background)
- `c` on a line opens comment editor
- Existing threads shown as collapsible sections between diff lines
- `r` replies to thread under cursor
- `x` resolves thread under cursor
- Comment editor: multi-line textarea, Ctrl+S to submit, Esc to cancel
**Test:** Unit test: comment overlay, submit, resolve
**Verification:** iterm2-driver: add comment on line, view thread, resolve
**Complexity:** L

---

### Phase 10: Review & Forms

#### T10.1: Implement Review Form
**Description:** Review submission form using huh library.
**Acceptance Criteria:**
- Layout per section 7.5 mock
- Radio select: Comment, Approve, Request Changes
- Multi-line body textarea
- Submit and Cancel buttons
- Keyboard: Tab between fields, Enter submit, Esc cancel
- Success toast on submission
- Error display on failure
**Test:** Unit test: form navigation, submission
**Verification:** iterm2-driver: open form, fill, submit
**Complexity:** M

#### T10.2: Implement Confirmation Dialog
**Description:** Reusable confirmation component.
**Acceptance Criteria:**
- Shows action description and confirm/cancel buttons
- Keyboard: Enter confirm, Esc cancel, y/n shortcuts
- Used by: checkout, resolve thread, discard comment draft
**Test:** Unit test: confirm and cancel paths
**Verification:** iterm2-driver: trigger checkout, verify dialog
**Complexity:** S

---

### Phase 11: Repo Switching

#### T11.1: Implement Repo Switcher Overlay
**Description:** Fuzzy repo picker overlay.
**Acceptance Criteria:**
- Ctrl+R opens overlay (per section 7.4 mock)
- Shows favorites (starred) and recent repos
- Fuzzy search as you type
- Current repo highlighted
- Enter switches, Esc cancels
- Triggers full PR list reload on switch
**Test:** Unit test: search filtering, selection
**Verification:** iterm2-driver: open switcher, search, switch repo
**Complexity:** M

#### T11.2: Implement Unified PR Inbox (S4)
**Description:** Aggregate PRs from all favorite repos into a single view with tabs.
**Acceptance Criteria:**
- New view mode: "Inbox" accessible via `I` from PR list
- Tabs: "All", "Assigned to me", "Review requested", "My PRs"
- PRs sorted by priority: review-requested > CI-failing > stale > updated
- Unread indicators: dot badge on PRs with activity since last viewed
- Repo name shown as prefix on each PR row (e.g., `owner/repo #142`)
- Fetches PRs from all configured favorite repos in parallel
- Caches per-repo results with independent refresh
- Tab switching does not re-fetch (client-side filter)
**Test:** Unit tests: multi-repo aggregation, tab filtering, priority sort, unread tracking
**Verification:** iterm2-driver: configure 2+ favorites, open inbox, switch tabs, verify content
**Complexity:** L

#### T11.3: Implement Repo Auto-Detection
**Description:** Detect repo from CWD git remote.
**Acceptance Criteria:**
- Parse `.git/config` or use `git remote get-url origin`
- Extract owner/name from GitHub remote URL (SSH and HTTPS)
- Handle: no git repo, no remote, non-GitHub remote
- Falls back to first favorite or prompts
**Test:** Unit tests: parse various remote URL formats
**Verification:** Run from different repos, verify detection
**Complexity:** S

---

### Phase 12: Help System

#### T12.1: Implement Help Overlay
**Description:** Context-aware help display.
**Acceptance Criteria:**
- `?` toggles help overlay for current view
- Shows keybindings grouped by category
- Layout per section 7.7 mock
- Esc or `?` closes
- Adapts content based on active view
**Test:** Unit test: render for each view
**Verification:** iterm2-driver: open help in each view, verify content matches
**Complexity:** M

#### T12.2: Implement First-Launch Tutorial
**Description:** One-time tutorial overlay shown on first launch.
**Acceptance Criteria:**
- Detects first launch via flag in XDG data directory
- Shows a brief walkthrough: navigation, opening PRs, help key
- Step-through with Enter/Space, skip with Esc
- Never shown again after dismissal (or completion)
- Can be re-triggered via `vivecaka --tutorial`
**Test:** Unit test: first-launch detection, step navigation, dismissal
**Verification:** Delete data dir, launch app, verify tutorial appears; relaunch, verify it doesn't
**Complexity:** M

#### T12.3: Implement Status Bar Hints
**Description:** Condensed key hints in status bar.
**Acceptance Criteria:**
- Shows most common keys for current view
- Format: `j/k navigate  Enter detail  c checkout  ? help`
- Updates on view change
- Truncates gracefully on narrow terminals
**Test:** Unit test: render for each view at various widths
**Verification:** iterm2-driver: verify hints match active view
**Complexity:** S

---

### Phase 13: Integration & Polish

#### T13.1: End-to-End Integration Test
**Description:** Full flow test using teatest or similar.
**Acceptance Criteria:**
- Test: start app → load PRs → navigate → open detail → go back → quit
- Test: start app → search → filter → clear
- Test: error handling (gh not installed, not authenticated, network error)
- Tests run headlessly via teatest
**Test:** Integration tests in `internal/tui/integration_test.go`
**Verification:** `make test` passes with integration tests
**Complexity:** L

#### T13.2: Visual QA with iterm2-driver
**Description:** Automated visual testing across all views.
**Acceptance Criteria:**
- Screenshot every major view state
- Compare against expected layout (no visual regressions)
- Check: alignment, color correctness, border integrity, truncation
- Run after each phase completion
**Test:** iterm2-driver script that launches app and screenshots each view
**Verification:** Screenshots reviewed and approved
**Complexity:** M

#### T13.3: Error Handling & Edge Cases
**Description:** Ensure all error paths are covered.
**Acceptance Criteria:**
- `gh` not installed: clear error message with install instructions
- `gh` not authenticated: clear error with `gh auth login` prompt
- Network errors: retry with exponential backoff, user-visible error
- Empty repo (no PRs): friendly empty state
- Very long PR titles: truncated with ellipsis
- Very large diffs: paginated/lazy-loaded
- Terminal too small: graceful degradation with minimum size warning
**Test:** Unit tests for each error scenario
**Verification:** Trigger each error manually and verify UX
**Complexity:** M

#### T13.4: Performance Optimization
**Description:** Ensure startup and interaction performance targets.
**Acceptance Criteria:**
- First render: <100ms (cached data)
- PR list load: <500ms (network permitting)
- Keyboard response: <16ms (60fps)
- No visible flicker during refresh
- Memory usage: <50MB with 500 PRs cached
**Test:** Benchmark tests for critical paths
**Verification:** Profile with `go tool pprof`; measure startup time
**Complexity:** M

#### T13.5: Implement PR Data Cache (S6)
**Description:** Implement local caching of PR data for instant startup.
**Acceptance Criteria:**
- Cache PR list per-repo in XDG cache directory (`~/.cache/vivecaka/`)
- JSON serialization of cached PRs with timestamp
- On startup: load cached data immediately, then fetch fresh in background
- Cache invalidated after `cache_ttl` minutes
- Manual cache clear: `vivecaka --clear-cache`
- Graceful fallback: if cache is corrupt, ignore and fetch fresh
**Test:** Unit test: cache write/read, TTL expiry, corruption recovery
**Verification:** Time first launch (cold) vs second launch (cached); verify <100ms cached render
**Complexity:** M

#### T13.6: Implement Visual Selection Mode (S5)
**Description:** Multi-select mode for batch operations on PRs.
**Acceptance Criteria:**
- `v` enters visual selection mode (highlighted indicator in status bar)
- `j`/`k` extends selection, `Space` toggles individual items
- Selected PRs shown with distinct background color
- Batch actions on selected: `c` checkout first, `o` open all in browser, `y` copy all URLs
- `Esc` exits selection mode and clears selection
- `V` selects/deselects all visible PRs
**Test:** Unit test: enter/exit mode, extend selection, batch actions
**Verification:** iterm2-driver: enter `v`, select multiple PRs, batch open, verify
**Complexity:** M

#### T13.7: README and Documentation
**Description:** Final README with installation, usage, screenshots.
**Acceptance Criteria:**
- Installation: `go install`, pre-built binaries
- Quick start guide
- Configuration reference
- Keybinding reference
- Screenshots (captured via iterm2-driver)
- Contributing guide (minimal)
**Test:** README renders correctly on GitHub
**Verification:** Fresh checkout follows README to successful usage
**Complexity:** S

---

## 12. Testing Strategy

### Layers

| Layer | Test Type | Tool | Coverage Target |
|-------|-----------|------|-----------------|
| Domain entities | Unit | `go test` | 100% |
| Domain interfaces | Compile-time | `go build` | N/A |
| Use cases | Unit (mocked adapters) | `go test` + testify | 90%+ |
| Adapters | Unit (fixture data) + Integration | `go test` + testdata/ | 80%+ |
| Plugin system | Unit | `go test` + testify | 90%+ |
| Config | Unit | `go test` | 90%+ |
| TUI models | Unit (Update/View) | `go test` | 70%+ |
| TUI integration | Integration | teatest | Key flows |
| Visual | Screenshot | iterm2-driver | All views |

### Test File Naming

```
internal/domain/pr_test.go         # Unit tests beside source
internal/adapter/ghcli/testdata/   # JSON fixtures for gh output
internal/tui/views/prlist_test.go  # View model tests
```

### Mock Strategy

- Interfaces defined in `domain/` - mocks in test files or `internal/mock/`
- Use testify/mock for complex mock behavior
- Use simple struct implementations for straightforward mocks
- Fixture data in `testdata/` directories (JSON files for gh CLI output)

### CI Pipeline (make ci)

```
make fmt      → format check
make vet      → static analysis
make lint     → golangci-lint
make test     → go test -race -cover ./...
make build    → compile binary
```

---

## 13. Makefile & CI

### Target Summary

```makefile
# Build & Run
build          Build binary to bin/vivecaka
install        Install to $GOPATH/bin
run            Run with go run
dev            Run with auto-reload (via air or reflex)

# Quality
fmt            Format code with gofmt
vet            Run go vet
lint           Run golangci-lint
test           Run tests with race detector
coverage       Generate and open coverage report
ci             Run all quality checks (fmt, vet, lint, test, build)

# Dependencies
deps           Download and tidy dependencies
tools          Install development tools (golangci-lint, lefthook, goreleaser)

# Git Hooks
hooks-install  Install lefthook git hooks
hooks-uninstall Remove lefthook git hooks

# Release
snapshot       Build snapshot release with goreleaser (local only)
release        Run goreleaser (CI only)

# Maintenance
clean          Remove build artifacts
help           Show this help message
```

---

## 14. Terminal Compatibility

### Universal Features (Safe to Use)

| Feature | iTerm2 | Ghostty | Warp | Kitty |
|---------|--------|---------|------|-------|
| TrueColor (24-bit) | Yes | Yes | Yes | Yes |
| AltScreen | Yes | Yes | Buggy* | Yes |
| Mouse events | Yes | Yes | Yes | Yes |
| Unicode box drawing | Yes | Yes | Yes | Yes |
| Bold/Italic/Underline | Yes | Yes | Yes | Yes |
| Mode 2026 (sync output) | Yes | Yes | Unknown | Yes |
| OSC 8 hyperlinks | Yes | Yes | Partial | Yes |

### Known Issues & Mitigations

**Warp Terminal:**
- Full-screen TUIs can have scroll issues (Warp treats them as blocks)
- Warp intercepts some keybindings (Cmd+P, Cmd+F)
- **Mitigation:** Document known Warp limitations; test core flows in Warp

**Ghostty:**
- Strict font metrics may affect icon spacing
- **Mitigation:** Use standard Unicode symbols, not Nerd Font glyphs for essential UI

**Emoji Width:**
- Different terminals render emojis at different widths
- **Mitigation:** Use text symbols (✓, ✗, ◐) not emoji for status indicators

### Design Rules

1. Use standard Unicode symbols for UI elements (not emoji, not Nerd Font)
2. Always account for borders in layout calculations
3. Use proportional sizing (not hardcoded widths)
4. Truncate aggressively - never let text auto-wrap inside bordered panels
5. Test with minimum 80x24 terminal; show warning below that
6. Strip ANSI codes before width calculations

---

## 15. Visual QA with iterm2-driver

### Strategy

Every visual change must be verified via iterm2-driver automation. This ensures:
- No alignment issues or color bleeds
- Keyboard navigation works correctly
- Modals/overlays display properly
- Theme consistency across views

### iterm2-driver Test Script Outline

```
# After each phase, run:
1. Launch vivecaka in iTerm2 via iterm2-driver
2. Wait for initial load
3. Screenshot: PR list view (default state)
4. Navigate with j/k, screenshot: selection highlight
5. Press /, type search text, screenshot: search active
6. Press f, screenshot: filter panel
7. Press Enter, screenshot: PR detail view
8. Press Tab (cycle panes), screenshot: pane focus
9. Press d, screenshot: diff view
10. Press t, screenshot: toggle split/unified
11. Press ?, screenshot: help overlay
12. Press Ctrl+R, screenshot: repo switcher
13. Press q repeatedly to exit
14. Compare all screenshots against expected layouts
```

### Verification Checklist (Per View)

- [ ] All borders aligned and closed (no gaps)
- [ ] Colors match theme definition
- [ ] Text truncated properly (no overflow/wrap)
- [ ] Selected/focused elements visually distinct
- [ ] Status bar shows correct hints
- [ ] Header shows correct repo/state
- [ ] Loading states render cleanly
- [ ] Error states render with clear messaging
- [ ] Empty states render with helpful text
- [ ] Minimum terminal size (80x24) works

---

## Appendix A: Keybinding Reference (MVP)

### Global

| Key | Action |
|-----|--------|
| `q` | Quit / Go back |
| `?` | Toggle help overlay |
| `Ctrl+R` | Open repo switcher |
| `p` | Toggle auto-refresh |
| `T` | Cycle theme |
| `Ctrl+C` | Force quit |

### PR List View

| Key | Action |
|-----|--------|
| `j` / `k` / `↓` / `↑` | Navigate up/down |
| `g` / `G` | Go to top / bottom |
| `Ctrl+D` / `Ctrl+U` | Half page down / up |
| `Enter` | Open PR detail |
| `c` | Checkout PR |
| `o` | Open PR in browser |
| `y` | Copy PR URL |
| `/` | Search |
| `Esc` | Clear search |
| `f` | Open filter panel |
| `m` | Quick filter: My PRs |
| `n` | Quick filter: Needs my review |
| `s` | Sort selector |
| `R` | Force refresh |
| `I` | Toggle unified PR inbox |
| `v` | Enter visual selection mode |
| `V` | Select / deselect all visible |

### PR Detail View

| Key | Action |
|-----|--------|
| `Tab` | Cycle panes |
| `d` | Open diff view |
| `c` | Checkout |
| `r` | Submit review |
| `o` | Open in browser |
| `q` | Back to list |

### Diff View

| Key | Action |
|-----|--------|
| `h` / `l` / `j` / `k` | Scroll |
| `{` / `}` | Previous / next file |
| `[` / `]` | Previous / next hunk |
| `g` / `G` | Top / bottom of file |
| `t` | Toggle unified / split |
| `e` | Open in external diff tool |
| `c` | Add comment on line |
| `r` | Reply to thread |
| `x` | Resolve thread |
| `/` | Search in diff |
| `n` / `N` | Next / previous search match |
| `Esc` | Clear search |
| `za` | Toggle file collapse (two-key sequence via pending-key state machine) |
| `q` | Back to detail |

### Review Form

| Key | Action |
|-----|--------|
| `Tab` | Next field |
| `Shift+Tab` | Previous field |
| `Enter` | Submit |
| `Esc` | Cancel |

### Filter Panel

| Key | Action |
|-----|--------|
| `Tab` | Next field |
| `Space` | Toggle checkbox |
| `Enter` | Apply |
| `Esc` | Cancel |
| `r` | Reset all |

### Error State

| Key | Action |
|-----|--------|
| `q` | Quit |
| `R` | Retry last action |
| `?` | Toggle help |

### Repo Switcher

| Key | Action |
|-----|--------|
| (text) | Fuzzy search |
| `j` / `k` / `↓` / `↑` | Navigate results |
| `Enter` | Switch to selected repo |
| `Esc` | Cancel |

---

## Appendix B: Phase Dependency Graph

```
Phase 0 (Bootstrap)
    │
    ├── Phase 1 (Domain)
    │       │
    │       ├── Phase 2 (Plugin Infra)
    │       │       │
    │       │       └── Phase 4 (GH CLI Adapter)
    │       │               │
    │       │               └── Phase 5 (Use Cases)
    │       │                       │
    │       │                       └── Phase 7 (PR List)
    │       │                       │       │
    │       │                       │       └── Phase 8 (PR Detail)
    │       │                       │               │
    │       │                       │               └── Phase 9 (Diff)
    │       │                       │                       │
    │       │                       │                       └── Phase 10 (Review)
    │       │                       │
    │       │                       └── Phase 11 (Repo Switch) ── also depends on Phase 6
    │       │
    │       └── Phase 3 (Config)
    │               │
    │               └── Phase 6 (TUI Foundation)
    │                       │
    │                       └── Phases 7-12 (All Views)
    │
    └── Phase 12 (Help) ── depends on all views being done
            │
            └── Phase 13 (Integration & Polish)
```

### Parallel Opportunities

- **Phase 1 + Phase 0.5-0.8** can overlap (domain while finishing bootstrap)
- **Phase 2 + Phase 3** can run in parallel
- **Phase 6** can start once Phase 3 is done (config needed for themes)
- **Phase 7 + Phase 11** can run in parallel (both need use cases + TUI foundation)
- **Phase 9 + Phase 10** can run in parallel

---

## Appendix C: Glossary

| Term | Definition |
|------|-----------|
| **PR** | Pull Request |
| **CWD** | Current Working Directory |
| **XDG** | XDG Base Directory Specification |
| **ISP** | Interface Segregation Principle |
| **OCP** | Open/Closed Principle |
| **DIP** | Dependency Inversion Principle |
| **TUI** | Terminal User Interface |
| **DX** | Developer Experience |
| **Hunk** | A contiguous block of changed lines in a diff |
| **Thread** | A conversation chain on a specific line in a PR |
| **Fixture** | Static test data used in unit tests |
| **teatest** | Charmbracelet's BubbleTea testing library |
