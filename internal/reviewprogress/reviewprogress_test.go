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
