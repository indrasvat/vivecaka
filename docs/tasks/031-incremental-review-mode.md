# Task 031: Incremental Review Mode

## Status: TODO

## Summary

Add a first-class incremental review workflow so vivecaka can reopen a PR and immediately answer:

1. What changed since I last looked?
2. What changed since I last submitted a review?
3. Which files have I already reviewed at this revision?
4. Where should I jump next?

The feature must preserve vivecaka's existing identity:
- beautiful and theme-consistent
- fast on large PRs
- keyboard-first
- zero background bleed
- modular and open for extension

## Why This Feature Now

vivecaka already handles first-pass review well:
- rich diff viewer
- inline comments and thread actions
- PR detail tabs
- inbox and unread indicators
- smart checkout

The biggest remaining workflow gap is **repeat review**. Large PRs rarely land after one pass. Reviewers reopen the same PR after new commits, but today vivecaka only shows the full diff again. That forces humans to reconstruct context manually.

Incremental Review Mode makes vivecaka materially better, not merely broader.

## Product Goal

Reduce the time to perform a follow-up review on an already-seen PR by at least **50%** for common cases:
- author addresses comments and pushes 1-3 commits
- reviewer wants only new or changed files
- reviewer wants to resume exactly where they stopped

## Non-Goals (V1)

- Applying GitHub suggestion blocks
- Full stacked-PR awareness
- Commit-by-commit review UI
- Merge queue orchestration
- Multi-reviewer shared progress state
- Cross-device sync of viewed-file state

## UX Principles

1. **Compact by default**
   The new UI must fit inside existing PR detail and diff layouts without introducing another heavy pane or noisy dashboard.

2. **Progress must be glanceable**
   Review state should be visible in one line: viewed progress, changed-since-baseline count, and active scope.

3. **No theme regressions**
   All rendering must reuse existing semantic colors and styles. No new background blocks should leak into terminal cells. Full-width padding rules from `CLAUDE.md` still apply.

4. **Fast path first**
   Expensive diff comparisons must happen once per load, never on every render frame or keypress.

5. **Resume, don’t restart**
   When reopening a PR, the default cursor target should be the first actionable file for the chosen scope.

## User Stories

### Story A: Resume after a quick revisit

1. Open PR #42 on Monday.
2. Review half the files.
3. Quit vivecaka.
4. Reopen the PR later the same day.
5. See `7/14 viewed` and jump directly to the first unviewed file.

### Story B: Review only new changes after feedback

1. Review PR #42 and submit changes requested.
2. Author pushes two commits.
3. Reopen the PR.
4. See `Δ 3 files since last review`.
5. Toggle scope to `Since Review`.
6. Review only those files and submit approval.

### Story C: Large PR, narrow terminal

1. Open a PR with 80 changed files.
2. Incremental review controls stay on a single compact line in detail and diff.
3. No layout jitter, overlap, or clipped borders on 120x34 and 100x30 terminals.

## UX Surface Area

### 1. PR List

Keep PR list changes minimal. Add only a compact badge when a PR has delta against the last review baseline.

Example row suffix:

```text
Δ3
```

Meaning: three files changed since the last submitted review baseline.

### 2. PR Detail

Add a **Review Context Bar** below the tab row and above the tab content.

It must be a single compact line on normal widths and degrade gracefully on narrower terminals.

### 3. Files Tab

Show per-file review state and support scope filtering.

### 4. Diff View

Add the same review context in a tighter file-aware form:
- current scope
- current file state
- global progress

## ASCII Mocks

### Mock A: PR Detail With Review Context Bar

```text
┌─────────────────────────────────────────────────────────────────────────────┐
│ #11 Update default model to claude-opus-4-6      indrasvat → main    Open │
├─────────────────────────────────────────────────────────────────────────────┤
│  Description  Checks (4)  Files (12)  Comments (8)                         │
├─────────────────────────────────────────────────────────────────────────────┤
│  Review  7/12 viewed   Δ 3 since review   scope: Since Review [i]          │
│  Next target: README.md                                  u jump   V viewed │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  Markdown body / selected tab content                                       │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
 i cycle scope  u next target  V viewed  d diff  c checkout  r review  Esc back
```

### Mock B: Files Tab With Review Markers

```text
┌─────────────────────────────────────────────────────────────────────────────┐
│ #11 Update default model to claude-opus-4-6      indrasvat → main    Open │
├─────────────────────────────────────────────────────────────────────────────┤
│  Description  Checks (4)  [Files (12)]  Comments (8)                       │
├─────────────────────────────────────────────────────────────────────────────┤
│  Review  7/12 viewed   Δ 3 since review   scope: Unviewed [i]              │
├─────────────────────────────────────────────────────────────────────────────┤
│  ● README.md                               +16 -2   changed since review   │
│  ◌ docs/automation.md                       +9  -0   unviewed               │
│  ✓ internal/tui/views/prdetail.go           +2  -1   viewed at current rev │
│  ✓ internal/cache/state.go                  +1  -0   viewed at current rev │
│                                                                             │
│                                                  ↓ 8 more                   │
└─────────────────────────────────────────────────────────────────────────────┘
 j/k move  Enter diff  i cycle scope  u next target  V viewed  Esc back
```

Legend:
- `●` actionable delta file
- `◌` unviewed file with no new post-review delta
- `✓` viewed for the active revision

### Mock C: Diff View With Incremental Review Header

```text
┌─────────────────────────────────────────────────────────────────────────────┐
│ README.md                                                  +16 -2  modified │
├──────────────────────────────┬──────────────────────────────────────────────┤
│ ● README.md                  │ scope: Since Review   progress: 7/12 viewed │
│ ◌ docs/automation.md         │ file: changed since review     V mark viewed │
│ ✓ internal/tui/views/...     ├──────────────────────────────────────────────┤
│ ✓ internal/cache/state.go    │ @@ -10,6 +10,10 @@                           │
│                              │ + Incremental review mode ...                │
│                              │ + ...                                        │
│                              │                                              │
└──────────────────────────────┴──────────────────────────────────────────────┘
 Tab pane  i scope  u next target  V viewed  [/] hunk  {/} file  t split  Esc back
```

### Mock D: Narrow Terminal Degradation (100 cols)

```text
┌──────────────────────────────────────────────────────────────────────────────────┐
│ Review 7/12 viewed · Δ3 since review · Since Review [i]        u next  V viewed │
└──────────────────────────────────────────────────────────────────────────────────┘
```

Rules:
- Collapse labels before removing signal.
- Keep the active scope visible.
- Never wrap the context bar to two lines unless the terminal is extremely narrow.

## Key Bindings

Bindings must be consistent between PR detail and diff whenever possible.

| Key | View | Action |
|-----|------|--------|
| `i` | Detail, Files, Diff | Cycle incremental scope: `All` → `Since Visit` → `Since Review` → `Unviewed` |
| `u` | Detail, Files, Diff | Jump to next actionable file for current scope |
| `V` | Files, Diff | Mark current file viewed/unviewed at the current head revision |
| `g` / `G` | Files, Diff | Preserve existing top/bottom behavior |
| `Enter` | Files | Open diff at selected file |

Notes:
- Do **not** overload `v`; it already means visual selection in the PR list.
- Do **not** introduce control-key gymnastics for primary actions.
- `V` is acceptable because it is mnemonic for "Viewed" and not currently central in detail/diff.

## Scope Model

### Supported Scopes

```go
type ReviewScope string

const (
    ReviewScopeAll         ReviewScope = "all"
    ReviewScopeSinceVisit  ReviewScope = "since_visit"
    ReviewScopeSinceReview ReviewScope = "since_review"
    ReviewScopeUnviewed    ReviewScope = "unviewed"
)
```

### Scope Semantics

- `all`
  Show every changed file in the PR.

- `since_visit`
  Show files whose patch digest differs from the last visited snapshot or that have never been viewed.

- `since_review`
  Show files whose patch digest differs from the last submitted review baseline or that were added after that review.

- `unviewed`
  Show files not yet marked viewed for the active head revision.

## Data Model

### Adapter Additions

Incremental review needs immutable revision identifiers. Add head/base commit SHAs to PR detail and optionally lightweight list payloads.

```go
type BranchInfo struct {
    Head    string `json:"head"`
    Base    string `json:"base"`
    HeadSHA string `json:"head_sha"`
    BaseSHA string `json:"base_sha"`
}
```

### Persisted Review State

Extend per-repo state with per-PR review metadata.

```go
type RepoState struct {
    LastSort    string
    LastSortAsc bool
    LastFilter  domain.ListOpts

    LastViewedPRs map[int]time.Time
    PRReviews     map[int]PRReviewState
}

type PRReviewState struct {
    LastVisitAt       time.Time                   `json:"last_visit_at,omitempty"`
    LastVisitHeadSHA  string                      `json:"last_visit_head_sha,omitempty"`
    LastReviewAt      time.Time                   `json:"last_review_at,omitempty"`
    LastReviewHeadSHA string                      `json:"last_review_head_sha,omitempty"`
    ActiveScope       ReviewScope                 `json:"active_scope,omitempty"`
    Files             map[string]FileReviewState  `json:"files,omitempty"`
}

type FileReviewState struct {
    ViewedAt     time.Time `json:"viewed_at,omitempty"`
    ViewedHeadSHA string   `json:"viewed_head_sha,omitempty"`
    PatchDigest  string    `json:"patch_digest,omitempty"`
}
```

### Computed Review Context

Do not let TUI views compute diff state ad hoc. Add a dedicated use case output.

```go
type ReviewContext struct {
    Scope              ReviewScope
    HeadSHA            string
    LastVisitHeadSHA   string
    LastReviewHeadSHA  string
    ViewedFiles        int
    TotalFiles         int
    SinceVisitFiles    int
    SinceReviewFiles   int
    ActionableFiles    int
    Files              []ReviewedFile
}

type ReviewedFile struct {
    Path               string
    Additions          int
    Deletions          int
    Status             string
    PatchDigest        string
    ViewedAtHead       bool
    ChangedSinceVisit  bool
    ChangedSinceReview bool
    Actionable         bool
}
```

### Patch Digest

To detect whether a file meaningfully changed relative to a prior review baseline, compute a stable digest from parsed diff hunks.

Requirements:
- computed once when diff data loads
- stable across renders
- based on semantic diff content, not terminal formatting
- stored per file path

Suggested implementation:
- add `PatchDigest string` to `domain.DiffFile`
- compute digest in the diff parser or immediately after diff load inside a dedicated helper
- use `sha1` or `fnv64a`; security is irrelevant, stability and speed matter

## Architecture & Layering

### New Use Case

Add a new review-progress use case rather than embedding rules in TUI models.

```text
tui
  -> usecase.GetReviewContext
      -> domain.PRReader      (existing)
      -> cache/review state   (existing package, extended)
      -> reviewprogress       (new pure-Go helper package)
```

### New Helper Package

Create `internal/reviewprogress/` for pure computation:
- patch-digest comparison
- scope filtering
- next-target selection
- progress summary derivation

This keeps:
- `cache` focused on persistence
- `tui` focused on rendering and key handling
- `usecase` focused on orchestration

### Open-Closed Principle

The design should allow future extension without rewriting the diff view:
- new scopes can be added via `ReviewScope`
- stacked-PR awareness can later add another baseline source
- GitHub-specific persistence remains behind adapters and use cases

## TUI Integration Plan

### PR Detail

- Add `reviewContext *ReviewContext` to `PRDetailModel`
- Add compact `renderReviewContextBar(width int)` helper
- On tab switch, preserve context bar
- On `u`, if current tab is not `Files`, switch to `Files` and move cursor to next actionable file

### Files Tab

- Render per-file review marker at the far left
- Render a concise status phrase at the far right when width permits
- Filter visible file list by active scope
- Never mutate the underlying file list on every render; precompute filtered indexes when scope changes

### Diff View

- Reuse the existing file tree; add review markers without replacing existing git status icons
- Add a compact review header on the content pane
- `V` toggles viewed state for the current file at `HeadSHA`
- `u` jumps to the next actionable file respecting current scope
- Default file selection on open should honor `ActionableFiles`

### Default Open Behavior

When opening a diff from a PR with incremental state:

1. If the active scope has actionable files, open on the first actionable file.
2. Otherwise, open on the previously selected file if available.
3. Otherwise, fall back to the first changed file.

## Performance Guardrails

These are non-negotiable.

1. No O(files * hunks) recomputation during `View()`.
2. No hashing or scope derivation on every keypress.
3. Persist review state with atomic writes, but batch writes to state transitions:
   - open detail
   - mark viewed
   - submit review
   - scope change
4. No extra network round trip solely for incremental scope toggling.
5. Existing large-diff protections in `diffview.go` must remain in force.

## Styling Guardrails

1. Reuse existing semantic colors:
   - `Success` for viewed/current
   - `Warning` for actionable delta
   - `Muted/Subtext` for inactive metadata
2. No heavy background blocks spanning arbitrary widths.
3. Context bar must render with exact-width padding and reset sequences where needed.
4. Focus states in diff panes must remain visually stronger than review-state decoration.
5. The feature must look correct in all shipped themes without theme-specific branches.

## Submit Review Behavior

When a review is successfully submitted:

1. Save `LastReviewAt = now`
2. Save `LastReviewHeadSHA = current HeadSHA`
3. Mark all currently visible files in the submitted scope as viewed at this head
4. Keep `LastVisit*` synced to the same head after submit

Rationale:
- after a submitted review, "since review" becomes the natural baseline
- the next revisit should highlight only newly changed files

## Error and Edge Cases

### PR rebased or force-pushed

If `LastReviewHeadSHA` or `LastVisitHeadSHA` disappears from history, do not fail. Treat digest comparison as snapshot-based only.

### Renamed files

V1 behavior:
- path change counts as a new file for viewed-state purposes
- optional future enhancement: carry viewed state across pure renames if patch digest matches

### Huge PRs

If diff parsing already falls back to reduced rendering, incremental review still works off file-level digest metadata. Do not require full syntax highlighting or full-screen scan.

### Missing prior review

If no review baseline exists:
- `Since Review` behaves like `Unviewed`
- context bar should say `no prior review baseline`

## Testing Strategy

### Unit Tests

Add pure-Go coverage for:
- scope filtering
- patch digest stability
- viewed/unviewed transitions
- next actionable file selection
- submit-review baseline update
- renamed file fallback behavior

### TUI Tests

Add model tests for:
- PR detail context bar rendering
- files tab markers and scope filtering
- diff view incremental header
- `i`, `u`, and `V` key handling
- preservation of existing keys and focus behavior

### Integration Tests

Add message-passing tests for:
- open detail -> load review context
- open diff -> jump to first actionable file
- mark viewed -> persist -> reopen -> state restored
- submit review -> baseline updated -> delta counts cleared

### Real GitHub Scenario (Required)

Synthetic fixtures are not enough for this feature. Incremental review must be validated against a **real live PR** on GitHub.

Required dogfooding loop:

1. Create a feature-branch PR for the incremental review implementation itself.
2. Install or run the branch build of vivecaka against that PR.
3. Open the PR in vivecaka and add at least one real inline comment or review-body comment.
4. Address that comment with a follow-up commit on the same PR branch.
5. Reopen the PR in vivecaka.
6. Verify that:
   - `Since Review` shows only files changed after the earlier review baseline
   - unchanged files do not appear in the actionable scope
   - previously reviewed files remain marked viewed when their patch digest is unchanged
   - only follow-up review work is surfaced
7. Submit a second review and verify the baseline resets correctly.

This dogfooding path is part of the feature definition, not an optional smoke test.

## L4 Visual Testing (iTerm2 Driver)

This feature is not done without visual proof.

### Automation Scripts

Existing scripts remain part of the non-regression suite:
- `.claude/automations/visual_test_filter_panel.py`
- `.claude/automations/visual_test_pr_comments.py`

New scripts required for this feature:
- `.claude/automations/visual_test_incremental_review_baseline.py`
  Capture current pre-feature detail/files/diff surfaces and preserve screenshots for comparison.
- `.claude/automations/visual_test_incremental_review_flow.py`
  Validate first-open, mark-viewed, reopen, and since-review scopes.
- `.claude/automations/visual_test_incremental_review_dogfood.py`
  Drive a live GitHub PR review on the implementation branch itself: leave a real comment, make a follow-up commit, reopen, and verify only follow-up review work is surfaced.
- `.claude/automations/visual_test_incremental_review_theme_matrix.py`
  Verify Catppuccin, Tokyo Night, Dracula, Solarized, and default-dark.
- `.claude/automations/visual_test_incremental_review_widths.py`
  Verify 120x34, 100x30, and 90x28 window sizes.
- `.claude/automations/visual_test_incremental_review_nonregression.py`
  Re-run core PR list/detail/diff flows to ensure no layout regressions.

### Screenshot Matrix

The implementation must capture and inspect at least these screenshots:

1. PR detail default scope
2. PR detail narrow-width degradation
3. Files tab in `All`
4. Files tab in `Since Review`
5. Files tab in `Unviewed`
6. Diff view with actionable file selected
7. Diff view after `V` mark viewed
8. Diff view after `u` jump
9. Same three states across all supported themes
10. Dogfood PR after first review submission
11. Dogfood PR after follow-up commit with `Since Review` active
12. Dogfood diff view showing only follow-up files

### Visual Acceptance Checklist

- No border breaks
- No status bar overlap
- No wrapped tab labels in standard widths
- No background-color bleed into empty cells
- No clipping of right-aligned scope metadata
- File tree focus border still obvious
- Existing inline comment threads still render correctly
- Existing large-diff error card still renders correctly

## Definition of Done

The feature is done only when all of the following are true:

1. `make ci` passes.
2. New unit and integration tests land with meaningful coverage.
3. All new visual automation scripts run successfully via `uv run`.
4. Screenshots are captured and manually inspected.
5. Existing visual scripts still pass.
6. No regressions in:
   - filter panel
   - PR comments
   - diff two-pane layout
   - side-by-side diff
   - smart checkout dialogs
7. Performance on large PRs remains acceptable.
8. The live dogfood PR scenario passes end to end on GitHub.

## Execution Plan

### Phase 1: Baseline and data plumbing

- extend persisted repo state
- add head/base SHA support
- add reviewprogress pure-Go helpers
- capture baseline screenshots before UI changes

### Phase 2: Detail/files incremental context

- add context bar
- add scope cycling
- add files-tab markers and filtering
- persist viewed state

### Phase 3: Diff integration

- add diff header
- default jump to actionable file
- add `V` and `u` behavior

### Phase 4: Submit-review baseline and regression hardening

- update baseline on review submit
- run the live dogfood PR scenario on the implementation branch
- add visual/theme/width matrix scripts
- run full regression suite

## Verification Commands

```bash
make ci
uv run .claude/automations/visual_test_incremental_review_baseline.py
uv run .claude/automations/visual_test_incremental_review_flow.py
uv run .claude/automations/visual_test_incremental_review_dogfood.py
uv run .claude/automations/visual_test_incremental_review_theme_matrix.py
uv run .claude/automations/visual_test_incremental_review_widths.py
uv run .claude/automations/visual_test_incremental_review_nonregression.py
```

<!-- dogfood-followup 2026-04-14T21:55:33.622529 -->

<!-- dogfood-followup 2026-04-14T22:01:28.555405 -->

<!-- dogfood-followup 2026-04-14T22:03:02.267369 -->
