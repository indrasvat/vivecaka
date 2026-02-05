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
