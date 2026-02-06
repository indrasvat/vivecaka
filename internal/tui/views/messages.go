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

// PRCountLoadedMsg is sent when the total PR count is fetched.
type PRCountLoadedMsg struct {
	Total int
	Err   error
}

// BranchDetectedMsg is sent when the current git branch is detected.
type BranchDetectedMsg struct {
	Branch string
	Err    error
}

// ReposDiscoveredMsg is sent when user's repos are fetched via gh repo list.
type ReposDiscoveredMsg struct {
	Repos []domain.RepoRef
	Err   error
}

// ToggleFavoriteMsg is sent when user toggles favorite status on a repo.
type ToggleFavoriteMsg struct {
	Repo     domain.RepoRef
	Favorite bool // new state: true = add to favorites, false = remove
}

// ValidateRepoRequestMsg is sent when user wants to add a manually entered repo.
type ValidateRepoRequestMsg struct {
	Repo domain.RepoRef
}

// RepoValidatedMsg is sent after validating a manually entered repo.
type RepoValidatedMsg struct {
	Repo domain.RepoRef
	Err  error
}
