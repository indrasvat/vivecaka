package domain

import "context"

// PRReader provides read-only access to pull requests.
type PRReader interface {
	ListPRs(ctx context.Context, repo RepoRef, opts ListOpts) ([]PR, error)
	GetPR(ctx context.Context, repo RepoRef, number int) (*PRDetail, error)
	GetDiff(ctx context.Context, repo RepoRef, number int) (*Diff, error)
	GetChecks(ctx context.Context, repo RepoRef, number int) ([]Check, error)
	GetComments(ctx context.Context, repo RepoRef, number int) ([]CommentThread, error)
	GetPRCount(ctx context.Context, repo RepoRef, state PRState) (int, error)
}

// PRReviewer provides review capabilities.
type PRReviewer interface {
	SubmitReview(ctx context.Context, repo RepoRef, number int, review Review) error
	AddComment(ctx context.Context, repo RepoRef, number int, input InlineCommentInput) error
	ResolveThread(ctx context.Context, repo RepoRef, threadID string) error
}

// PRWriter provides write capabilities.
// Only Checkout is exposed in MVP. Merge and UpdateLabels are for plugin extensibility.
type PRWriter interface {
	Checkout(ctx context.Context, repo RepoRef, number int) (branch string, err error)
	Merge(ctx context.Context, repo RepoRef, number int, opts MergeOpts) error
	UpdateLabels(ctx context.Context, repo RepoRef, number int, labels []string) error
}

// RepoManager provides local git repository management capabilities.
// Implemented by adapters that can perform git/clone operations.
// Separate from PRWriter — these are repo-level git ops, not PR ops.
type RepoManager interface {
	// CheckoutAt checks out a PR branch in the specified working directory.
	// If workDir is "", uses the process CWD (same as PRWriter.Checkout).
	CheckoutAt(ctx context.Context, repo RepoRef, number int, workDir string) (branch string, err error)
	// CloneRepo clones a repository to the specified local path.
	CloneRepo(ctx context.Context, repo RepoRef, targetPath string) error
	// CreateWorktree creates a git worktree for a PR branch at the given path.
	// It fetches the PR ref first, then creates the worktree.
	CreateWorktree(ctx context.Context, repoPath string, number int, branch, worktreePath string) error
}
