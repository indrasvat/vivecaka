package usecase

import (
	"context"

	"golang.org/x/sync/errgroup"

	"github.com/indrasvat/vivecaka/internal/domain"
)

// GetPRDetail fetches full PR detail with associated data in parallel.
type GetPRDetail struct {
	reader domain.PRReader
}

// NewGetPRDetail creates a new GetPRDetail use case.
func NewGetPRDetail(reader domain.PRReader) *GetPRDetail {
	return &GetPRDetail{reader: reader}
}

// Execute fetches a PR and its comments in parallel using errgroup.
// Note: Checks are already included in GetPR via statusCheckRollup, no separate call needed.
// Partial failures are tolerated: comments may be empty if the fetch fails.
func (uc *GetPRDetail) Execute(ctx context.Context, repo domain.RepoRef, number int) (*domain.PRDetail, error) {
	g, ctx := errgroup.WithContext(ctx)

	var detail *domain.PRDetail
	var comments []domain.CommentThread

	g.Go(func() error {
		var err error
		detail, err = uc.reader.GetPR(ctx, repo, number)
		return err // This is required - no detail means failure.
	})

	g.Go(func() error {
		var err error
		comments, err = uc.reader.GetComments(ctx, repo, number)
		if err != nil {
			comments = nil // Tolerate failure.
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Checks are already populated by GetPR from statusCheckRollup.
	detail.Comments = comments
	return detail, nil
}
