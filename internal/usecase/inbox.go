package usecase

import (
	"context"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/indrasvat/vivecaka/internal/domain"
)

// InboxPR wraps a PR with its source repo for multi-repo inbox.
type InboxPR struct {
	domain.PR
	Repo domain.RepoRef
}

// GetInboxPRs fetches PRs from multiple repos in parallel.
type GetInboxPRs struct {
	reader domain.PRReader
}

// NewGetInboxPRs creates a new GetInboxPRs use case.
func NewGetInboxPRs(reader domain.PRReader) *GetInboxPRs {
	return &GetInboxPRs{reader: reader}
}

// Execute fetches open PRs from all given repos concurrently.
// Partial failures are tolerated: results from successful repos are returned.
func (uc *GetInboxPRs) Execute(ctx context.Context, repos []domain.RepoRef) ([]InboxPR, error) {
	if len(repos) == 0 {
		return nil, nil
	}

	var mu sync.Mutex
	var result []InboxPR

	g, ctx := errgroup.WithContext(ctx)

	for _, repo := range repos {
		g.Go(func() error {
			opts := domain.ListOpts{
				State:   domain.PRStateOpen,
				PerPage: 30,
			}
			prs, err := uc.reader.ListPRs(ctx, repo, opts)
			if err != nil {
				return nil //nolint:nilerr // tolerate individual repo failures
			}
			mu.Lock()
			for _, pr := range prs {
				result = append(result, InboxPR{PR: pr, Repo: repo})
			}
			mu.Unlock()
			return nil
		})
	}

	// errgroup never returns error since we swallow per-repo failures.
	_ = g.Wait()

	return result, nil
}
