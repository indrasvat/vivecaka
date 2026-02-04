package usecase

import (
	"context"

	"github.com/indrasvat/vivecaka/internal/domain"
)

// ListPRs orchestrates PR listing with filtering and sorting.
type ListPRs struct {
	reader domain.PRReader
}

// NewListPRs creates a new ListPRs use case.
func NewListPRs(reader domain.PRReader) *ListPRs {
	return &ListPRs{reader: reader}
}

// Execute lists PRs for the given repo with the provided options.
func (uc *ListPRs) Execute(ctx context.Context, repo domain.RepoRef, opts domain.ListOpts) ([]domain.PR, error) {
	return uc.reader.ListPRs(ctx, repo, opts)
}
