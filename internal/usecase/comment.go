package usecase

import (
	"context"

	"github.com/indrasvat/vivecaka/internal/domain"
)

// AddComment adds an inline comment to a PR diff.
type AddComment struct {
	reviewer domain.PRReviewer
}

// NewAddComment creates a new AddComment use case.
func NewAddComment(reviewer domain.PRReviewer) *AddComment {
	return &AddComment{reviewer: reviewer}
}

// Execute validates and adds an inline comment.
func (uc *AddComment) Execute(ctx context.Context, repo domain.RepoRef, number int, input domain.InlineCommentInput) error {
	if input.Path == "" {
		return &domain.ValidationError{Field: "path", Message: "path is required"}
	}
	if input.Line <= 0 {
		return &domain.ValidationError{Field: "line", Message: "line must be positive"}
	}
	if input.Body == "" {
		return &domain.ValidationError{Field: "body", Message: "body is required"}
	}
	return uc.reviewer.AddComment(ctx, repo, number, input)
}

// ResolveThread resolves a review comment thread.
type ResolveThread struct {
	reviewer domain.PRReviewer
}

// NewResolveThread creates a new ResolveThread use case.
func NewResolveThread(reviewer domain.PRReviewer) *ResolveThread {
	return &ResolveThread{reviewer: reviewer}
}

// Execute resolves a thread by ID.
func (uc *ResolveThread) Execute(ctx context.Context, repo domain.RepoRef, threadID string) error {
	if threadID == "" {
		return &domain.ValidationError{Field: "thread_id", Message: "thread ID is required"}
	}
	return uc.reviewer.ResolveThread(ctx, repo, threadID)
}
