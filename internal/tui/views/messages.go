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

// CheckoutDoneMsg is sent after a PR checkout completes.
type CheckoutDoneMsg struct {
	Branch string
	Err    error
}

// ReviewSubmittedMsg is sent after a review submission completes.
type ReviewSubmittedMsg struct {
	Err error
}
