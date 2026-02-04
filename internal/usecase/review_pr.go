package usecase

import (
	"context"
	"fmt"

	"github.com/indrasvat/vivecaka/internal/domain"
)

// ReviewPR submits a review for a pull request.
type ReviewPR struct {
	reviewer domain.PRReviewer
}

// NewReviewPR creates a new ReviewPR use case.
func NewReviewPR(reviewer domain.PRReviewer) *ReviewPR {
	return &ReviewPR{reviewer: reviewer}
}

// Execute validates and submits a review.
func (uc *ReviewPR) Execute(ctx context.Context, repo domain.RepoRef, number int, review domain.Review) error {
	switch review.Action {
	case domain.ReviewActionApprove, domain.ReviewActionRequestChanges, domain.ReviewActionComment:
		// Valid actions.
	default:
		return &domain.ValidationError{
			Field:   "action",
			Message: fmt.Sprintf("invalid review action: %q", review.Action),
		}
	}

	if review.Action == domain.ReviewActionRequestChanges && review.Body == "" {
		return &domain.ValidationError{
			Field:   "body",
			Message: "body is required when requesting changes",
		}
	}

	return uc.reviewer.SubmitReview(ctx, repo, number, review)
}
