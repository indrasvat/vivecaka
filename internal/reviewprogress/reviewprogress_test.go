package reviewprogress

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/indrasvat/vivecaka/internal/cache"
	"github.com/indrasvat/vivecaka/internal/domain"
)

func testDetail() *domain.PRDetail {
	return &domain.PRDetail{
		PR: domain.PR{
			Number: 42,
			Branch: domain.BranchInfo{
				Head:    "feat/incremental-review",
				Base:    "main",
				HeadSHA: "head-2",
				BaseSHA: "base-1",
			},
		},
		Files: []domain.FileChange{
			{Path: "README.md", Additions: 10, Deletions: 2, Status: "modified"},
			{Path: "internal/tui/app.go", Additions: 3, Deletions: 1, Status: "modified"},
			{Path: "docs/PRD.md", Additions: 1, Deletions: 0, Status: "added"},
		},
	}
}

func TestScopeCycle(t *testing.T) {
	assert.Equal(t, ScopeSinceReview, ScopeSinceVisit.Cycle())
	assert.Equal(t, ScopeUnviewed, ScopeSinceReview.Cycle())
	assert.Equal(t, ScopeAll, ScopeUnviewed.Cycle())
	assert.Equal(t, ScopeSinceVisit, ScopeAll.Cycle())
}

func TestBuild_ContextFlagsAndActionable(t *testing.T) {
	detail := testDetail()
	digests := map[string]string{
		"README.md":           "digest-a",
		"internal/tui/app.go": "digest-b",
		"docs/PRD.md":         "digest-c",
	}
	state := cache.PRReviewState{
		ActiveScope:       string(ScopeSinceReview),
		LastReviewHeadSHA: "head-1",
		LastReviewFiles: map[string]string{
			"README.md":           "digest-a",
			"internal/tui/app.go": "digest-old",
		},
		ViewedFiles: map[string]cache.FileReviewState{
			"README.md": {PatchDigest: "digest-a"},
		},
	}

	ctx := Build(detail, digests, state, false)
	require.NotNil(t, ctx)
	assert.Equal(t, 3, ctx.TotalFiles)
	assert.Equal(t, 1, ctx.ViewedFiles)
	assert.Equal(t, 2, ctx.SinceReviewFiles)
	assert.Equal(t, 2, ctx.ActionableFiles)
	assert.Equal(t, "internal/tui/app.go", ctx.NextActionablePath)
}

func TestBuild_UnviewedWithoutBaselines(t *testing.T) {
	ctx := Build(testDetail(), map[string]string{
		"README.md":           "digest-a",
		"internal/tui/app.go": "digest-b",
		"docs/PRD.md":         "digest-c",
	}, cache.PRReviewState{}, false)

	require.NotNil(t, ctx)
	assert.Equal(t, ScopeSinceReview, ctx.Scope)
	assert.Equal(t, 3, ctx.ActionableFiles)
}

func TestBuild_ViewedDigestCarriesAcrossHeadChanges(t *testing.T) {
	detail := testDetail()
	ctx := Build(detail, map[string]string{
		"README.md":           "same-digest",
		"internal/tui/app.go": "new-digest",
		"docs/PRD.md":         "another-digest",
	}, cache.PRReviewState{
		ActiveScope: string(ScopeUnviewed),
		ViewedFiles: map[string]cache.FileReviewState{
			"README.md": {PatchDigest: "same-digest", ViewedHeadSHA: "head-1"},
		},
	}, false)

	file, ok := ctx.FindFile("README.md")
	require.True(t, ok)
	assert.True(t, file.Viewed)
}

func TestNextActionableAfter(t *testing.T) {
	ctx := &Context{
		Files: []File{
			{Path: "a.go", Actionable: false},
			{Path: "b.go", Actionable: true},
			{Path: "c.go", Actionable: true},
		},
		ActionableFiles: 2,
	}

	assert.Equal(t, "b.go", ctx.NextActionableAfter(""))
	assert.Equal(t, "c.go", ctx.NextActionableAfter("b.go"))
	assert.Equal(t, "b.go", ctx.NextActionableAfter("c.go"))
}

func TestDigestsFromDiffAreStableAndContentSensitive(t *testing.T) {
	diff := &domain.Diff{Files: []domain.FileDiff{
		{
			Path:    "renamed.go",
			OldPath: "old.go",
			Hunks: []domain.Hunk{{
				Header: "@@ -1 +1,2 @@",
				Lines: []domain.DiffLine{
					{Type: domain.DiffContext, Content: "package old", OldNum: 1, NewNum: 1},
					{Type: domain.DiffAdd, Content: "func added() {}", NewNum: 2},
				},
			}},
		},
	}}

	first := DigestsFromDiff(diff)
	second := DigestsFromDiff(diff)
	require.Equal(t, first, second, "same parsed diff must produce a stable patch digest")
	require.Len(t, first["renamed.go"], 40)

	diff.Files[0].Hunks[0].Lines[1].Content = "func changed() {}"
	changed := DigestsFromDiff(diff)
	assert.NotEqual(t, first["renamed.go"], changed["renamed.go"], "digest should change when patch content changes")
	assert.Nil(t, DigestsFromDiff(nil))
}

func TestFallbackDigestAndBuildCoverBaselineTransitions(t *testing.T) {
	detail := &domain.PRDetail{
		PR: domain.PR{Branch: domain.BranchInfo{HeadSHA: "head-2", BaseSHA: "base-1"}},
		Files: []domain.FileChange{
			{Path: "changed.go", Status: "modified", Additions: 3, Deletions: 1},
			{Path: "viewed.go", Status: "modified", Additions: 1, Deletions: 0},
			{Path: "new.go", Status: "added", Additions: 9, Deletions: 0},
		},
	}
	changedDigest := "changed-now"
	viewedDigest := FallbackDigest(detail.Files[1])
	state := cache.PRReviewState{
		ActiveScope:       string(ScopeSinceVisit),
		LastVisitAt:       time.Now().Add(-time.Hour),
		LastReviewAt:      time.Now().Add(-2 * time.Hour),
		LastVisitHeadSHA:  "head-1",
		LastReviewHeadSHA: "head-0",
		LastVisitFiles:    map[string]string{"changed.go": "old", "viewed.go": viewedDigest},
		LastReviewFiles:   map[string]string{"changed.go": "old", "viewed.go": viewedDigest},
		ViewedFiles:       map[string]cache.FileReviewState{"viewed.go": {PatchDigest: viewedDigest}},
	}

	ctx := Build(detail, map[string]string{"changed.go": changedDigest}, state, true)
	require.NotNil(t, ctx)
	assert.True(t, ctx.DegradedDigestSource)
	assert.True(t, ctx.HasVisitBaseline)
	assert.True(t, ctx.HasReviewBaseline)
	assert.Equal(t, 1, ctx.ViewedFiles)
	assert.Equal(t, 2, ctx.SinceVisitFiles)
	assert.Equal(t, 2, ctx.ActionableFiles, "changed and new are actionable under since-visit")
	assert.Equal(t, "changed.go", ctx.NextActionablePath)
	assert.Equal(t, changedDigest, ctx.CurrentDigests["changed.go"])
	assert.NotEmpty(t, ctx.CurrentDigests["new.go"], "missing diff digest falls back to metadata digest")

	viewed, ok := ctx.FindFile("viewed.go")
	require.True(t, ok)
	assert.True(t, viewed.Viewed)
	assert.False(t, viewed.Actionable)

	assert.NotEqual(t, FallbackDigest(detail.Files[0]), FallbackDigest(domain.FileChange{
		Path: "changed.go", Status: "modified", Additions: 4, Deletions: 1,
	}))
	assert.Nil(t, Build(nil, nil, cache.PRReviewState{}, false))
}

func TestBuildActionableFallbacksWhenBaselinesAreMissing(t *testing.T) {
	detail := &domain.PRDetail{
		PR: domain.PR{Branch: domain.BranchInfo{HeadSHA: "head"}},
		Files: []domain.FileChange{
			{Path: "seen.go", Additions: 1},
			{Path: "unseen.go", Additions: 1},
		},
	}
	seenDigest := FallbackDigest(detail.Files[0])
	state := cache.PRReviewState{
		ActiveScope: string(ScopeSinceReview),
		ViewedFiles: map[string]cache.FileReviewState{
			"seen.go": {PatchDigest: seenDigest},
		},
	}

	ctx := Build(detail, nil, state, false)
	require.NotNil(t, ctx)
	assert.False(t, ctx.HasReviewBaseline)
	assert.Equal(t, 1, ctx.ActionableFiles, "without a review baseline, unviewed files are actionable")
	assert.Equal(t, "unseen.go", ctx.NextActionableAfter("seen.go"))

	unviewedState := state
	unviewedState.ActiveScope = string(ScopeUnviewed)
	ctx = Build(detail, nil, unviewedState, false)
	assert.Equal(t, 1, ctx.ActionableFiles)

	allState := state
	allState.ActiveScope = string(ScopeAll)
	ctx = Build(detail, nil, allState, false)
	assert.Equal(t, 2, ctx.ActionableFiles)
}

func TestSummary(t *testing.T) {
	t.Run("nil context returns zero summary", func(t *testing.T) {
		var ctx *Context
		s := ctx.Summary()
		assert.Equal(t, 0, s.Percent)
		assert.False(t, s.Complete)
	})

	t.Run("partial progress", func(t *testing.T) {
		ctx := &Context{
			Scope:           ScopeSinceReview,
			ViewedFiles:     1,
			TotalFiles:      3,
			ActionableFiles: 2,
		}
		s := ctx.Summary()
		assert.Equal(t, 1, s.ViewedFiles)
		assert.Equal(t, 3, s.TotalFiles)
		assert.Equal(t, 33, s.Percent)
		assert.Equal(t, 2, s.Remaining)
		assert.Equal(t, 2, s.ActionableLeft)
		assert.Equal(t, "Since Review", s.ScopeLabel)
		assert.False(t, s.Complete)
	})

	t.Run("complete review", func(t *testing.T) {
		ctx := &Context{
			Scope:           ScopeAll,
			ViewedFiles:     4,
			TotalFiles:      4,
			ActionableFiles: 0,
		}
		s := ctx.Summary()
		assert.Equal(t, 100, s.Percent)
		assert.Equal(t, 0, s.Remaining)
		assert.True(t, s.Complete)
	})
}

func TestProgressSummaryString(t *testing.T) {
	t.Run("no files", func(t *testing.T) {
		s := ProgressSummary{}
		assert.Equal(t, "no files", s.String())
	})

	t.Run("partial progress", func(t *testing.T) {
		s := ProgressSummary{
			ViewedFiles:    1,
			TotalFiles:     3,
			ActionableLeft: 2,
			ScopeLabel:     "Since Review",
		}
		assert.Equal(t, "1/3 reviewed · 2 actionable (Since Review)", s.String())
	})

	t.Run("complete", func(t *testing.T) {
		s := ProgressSummary{
			ViewedFiles: 4,
			TotalFiles:  4,
			ScopeLabel:  "All",
			Complete:    true,
		}
		assert.Equal(t, "4/4 reviewed (All)", s.String())
	})
}

func TestSnapshotFromContext(t *testing.T) {
	now := time.Now()
	ctx := &Context{
		HeadSHA: "head-2",
		CurrentDigests: map[string]string{
			"a.go": "digest-a",
		},
	}

	head, files := SnapshotFromContext(ctx, now)
	assert.Equal(t, "head-2", head)
	assert.Equal(t, map[string]string{"a.go": "digest-a"}, files)
}
