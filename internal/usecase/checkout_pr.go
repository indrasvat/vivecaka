package usecase

import (
	"context"

	"github.com/indrasvat/vivecaka/internal/domain"
)

// CheckoutPR checks out a PR branch locally.
type CheckoutPR struct {
	writer domain.PRWriter
}

// NewCheckoutPR creates a new CheckoutPR use case.
func NewCheckoutPR(writer domain.PRWriter) *CheckoutPR {
	return &CheckoutPR{writer: writer}
}

// Execute checks out the PR and returns the local branch name.
func (uc *CheckoutPR) Execute(ctx context.Context, repo domain.RepoRef, number int) (string, error) {
	return uc.writer.Checkout(ctx, repo, number)
}
