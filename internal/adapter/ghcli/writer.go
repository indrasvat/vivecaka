package ghcli

import (
	"context"
	"fmt"
	"strings"

	"github.com/indrasvat/vivecaka/internal/domain"
)

// Checkout checks out a PR branch locally via gh pr checkout.
func (a *Adapter) Checkout(ctx context.Context, repo domain.RepoRef, number int) (string, error) {
	args := []string{"pr", "checkout", fmt.Sprintf("%d", number)}
	args = append(args, repoArgs(repo)...)

	out, err := ghExec(ctx, args...)
	if err != nil {
		return "", fmt.Errorf("checking out PR #%d: %w", number, err)
	}

	// gh pr checkout prints the branch name to stdout.
	branch := strings.TrimSpace(string(out))
	return branch, nil
}

// Merge merges a PR via gh pr merge (post-MVP, not exposed in UI).
func (a *Adapter) Merge(ctx context.Context, repo domain.RepoRef, number int, opts domain.MergeOpts) error {
	args := []string{"pr", "merge", fmt.Sprintf("%d", number)}
	args = append(args, repoArgs(repo)...)

	switch opts.Method {
	case "squash":
		args = append(args, "--squash")
	case "rebase":
		args = append(args, "--rebase")
	default:
		args = append(args, "--merge")
	}

	if opts.DeleteBranch {
		args = append(args, "--delete-branch")
	}
	if opts.CommitMessage != "" {
		args = append(args, "--body", opts.CommitMessage)
	}

	if _, err := ghExec(ctx, args...); err != nil {
		return fmt.Errorf("merging PR #%d: %w", number, err)
	}
	return nil
}

// UpdateLabels updates labels on a PR via gh pr edit (post-MVP).
func (a *Adapter) UpdateLabels(ctx context.Context, repo domain.RepoRef, number int, labels []string) error {
	args := []string{"pr", "edit", fmt.Sprintf("%d", number)}
	args = append(args, repoArgs(repo)...)

	for _, l := range labels {
		args = append(args, "--add-label", l)
	}

	if _, err := ghExec(ctx, args...); err != nil {
		return fmt.Errorf("updating labels on PR #%d: %w", number, err)
	}
	return nil
}
