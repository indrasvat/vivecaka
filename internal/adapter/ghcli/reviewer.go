package ghcli

import (
	"context"
	"fmt"

	"github.com/indrasvat/vivecaka/internal/domain"
)

// SubmitReview submits a review via gh pr review.
func (a *Adapter) SubmitReview(ctx context.Context, repo domain.RepoRef, number int, review domain.Review) error {
	args := []string{"pr", "review", fmt.Sprintf("%d", number)}
	args = append(args, repoArgs(repo)...)

	switch review.Action {
	case domain.ReviewActionApprove:
		args = append(args, "--approve")
	case domain.ReviewActionRequestChanges:
		args = append(args, "--request-changes")
	case domain.ReviewActionComment:
		args = append(args, "--comment")
	default:
		return fmt.Errorf("unknown review action: %q", review.Action)
	}

	if review.Body != "" {
		args = append(args, "--body", review.Body)
	}

	if _, err := ghExec(ctx, args...); err != nil {
		return fmt.Errorf("submitting review for PR #%d: %w", number, err)
	}
	return nil
}

// AddComment adds an inline review comment via the GitHub REST API.
func (a *Adapter) AddComment(ctx context.Context, repo domain.RepoRef, number int, input domain.InlineCommentInput) error {
	endpoint := fmt.Sprintf("repos/%s/pulls/%d/comments", repo, number)

	args := []string{"api", endpoint, "--method", "POST",
		"--raw-field", fmt.Sprintf("body=%s", input.Body),
		"--raw-field", fmt.Sprintf("path=%s", input.Path),
		"--field", fmt.Sprintf("line=%d", input.Line),
		"--raw-field", fmt.Sprintf("side=%s", input.Side),
		"--raw-field", fmt.Sprintf("commit_id=%s", input.CommitID),
	}
	if input.InReplyTo != "" {
		args = append(args, "--raw-field", fmt.Sprintf("in_reply_to=%s", input.InReplyTo))
	}

	if _, err := ghExec(ctx, args...); err != nil {
		return fmt.Errorf("adding comment to PR #%d: %w", number, err)
	}
	return nil
}

// ResolveThread resolves a review comment thread via the GraphQL API.
func (a *Adapter) ResolveThread(ctx context.Context, _ domain.RepoRef, threadID string) error {
	query := `mutation($id: ID!) { resolveReviewThread(input: {threadId: $id}) { thread { isResolved } } }`
	args := []string{"api", "graphql",
		"-f", fmt.Sprintf("query=%s", query),
		"-f", fmt.Sprintf("id=%s", threadID),
	}

	if _, err := ghExec(ctx, args...); err != nil {
		return fmt.Errorf("resolving thread %s: %w", threadID, err)
	}
	return nil
}
