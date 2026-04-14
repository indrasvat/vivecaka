package usecase

import (
	"context"
	"sort"
	"time"

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
	var inlineComments []domain.CommentThread
	var discussion []domain.DiscussionItem

	g.Go(func() error {
		var err error
		detail, err = uc.reader.GetPR(ctx, repo, number)
		return err // This is required - no detail means failure.
	})

	g.Go(func() error {
		var err error
		inlineComments, err = uc.reader.GetComments(ctx, repo, number)
		if err != nil {
			inlineComments = nil // Tolerate failure.
		}
		return nil
	})

	g.Go(func() error {
		var err error
		discussion, err = uc.reader.GetDiscussion(ctx, repo, number)
		if err != nil {
			discussion = nil // Tolerate failure.
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Checks are already populated by GetPR from statusCheckRollup.
	detail.InlineComments = inlineComments
	detail.Discussion = mergeDiscussion(inlineComments, discussion)
	return detail, nil
}

func mergeDiscussion(inline []domain.CommentThread, discussion []domain.DiscussionItem) []domain.DiscussionItem {
	items := make([]domain.DiscussionItem, 0, len(inline)+len(discussion))
	items = append(items, discussion...)
	for _, thread := range inline {
		createdAt := time.Time{}
		if len(thread.Comments) > 0 {
			createdAt = thread.Comments[0].CreatedAt
		}
		items = append(items, domain.DiscussionItem{
			ID:        thread.ID,
			Kind:      domain.DiscussionInlineThread,
			Path:      thread.Path,
			Line:      thread.Line,
			Resolved:  thread.Resolved,
			ThreadID:  thread.ThreadID,
			ReplyToID: thread.ReplyToID,
			CreatedAt: createdAt,
			Comments:  thread.Comments,
		})
	}

	sort.SliceStable(items, func(i, j int) bool {
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
	return items
}
