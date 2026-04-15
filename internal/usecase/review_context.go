package usecase

import (
	"context"

	"github.com/indrasvat/vivecaka/internal/cache"
	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/indrasvat/vivecaka/internal/reviewprogress"
)

// GetReviewContext loads the current diff and derives incremental review state.
type GetReviewContext struct {
	reader domain.PRReader
}

// NewGetReviewContext creates a new GetReviewContext use case.
func NewGetReviewContext(reader domain.PRReader) *GetReviewContext {
	return &GetReviewContext{reader: reader}
}

// Execute computes the review context for a PR.
// Diff failures are tolerated; the context falls back to file metadata digests.
func (uc *GetReviewContext) Execute(
	ctx context.Context,
	repo domain.RepoRef,
	number int,
	detail *domain.PRDetail,
	state cache.PRReviewState,
) (*reviewprogress.Context, *domain.Diff, error) {
	diff, err := uc.reader.GetDiff(ctx, repo, number)
	digests := reviewprogress.DigestsFromDiff(diff)
	degraded := false
	if len(digests) == 0 {
		degraded = true
		digests = make(map[string]string, len(detail.Files))
		for _, file := range detail.Files {
			digests[file.Path] = reviewprogress.FallbackDigest(file)
		}
	}

	context := reviewprogress.Build(detail, digests, state, degraded || err != nil)
	return context, diff, nil
}
