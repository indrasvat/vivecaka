package views

import "github.com/indrasvat/vivecaka/internal/domain"

// RepoDetectedMsg is sent when the current repo is identified from git remote.
type RepoDetectedMsg struct {
	Repo domain.RepoRef
	Err  error
}

// UserDetectedMsg is sent when the current GitHub user is identified.
type UserDetectedMsg struct {
	Username string
	Err      error
}

// PRListFilterMsg is sent when the PR list quick filter changes.
type PRListFilterMsg struct {
	Label string
}

// OpenFilterMsg is sent when the filter panel is requested.
type OpenFilterMsg struct{}

// ApplyFilterMsg is sent when filters should be applied.
type ApplyFilterMsg struct {
	Opts domain.ListOpts
}

// CloseFilterMsg is sent when the filter panel is dismissed.
type CloseFilterMsg struct{}

// CheckoutDoneMsg is sent after a PR checkout completes.
type CheckoutDoneMsg struct {
	Branch string
	Err    error
}

// ReviewSubmittedMsg is sent after a review submission completes.
type ReviewSubmittedMsg struct {
	Err error
}

// LoadMorePRsMsg is sent when the user scrolls near the bottom and more PRs should be loaded.
type LoadMorePRsMsg struct {
	Page int
}

// MorePRsLoadedMsg is sent when additional PRs have been loaded (pagination).
type MorePRsLoadedMsg struct {
	PRs     []domain.PR
	Page    int
	HasMore bool
	Err     error
}
