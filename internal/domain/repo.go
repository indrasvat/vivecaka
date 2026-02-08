package domain

import "time"

// RepoLocation tracks where a repository is cloned locally.
type RepoLocation struct {
	Repo     RepoRef   `json:"repo"`
	Path     string    `json:"path"`
	LastSeen time.Time `json:"last_seen"`
	Source   string    `json:"source"` // "detected", "cloned", "manual"
}

// RepoConfig represents a saved repo configuration.
type RepoConfig struct {
	Repo     RepoRef `json:"repo"`
	Favorite bool    `json:"favorite"`
}

// ListOpts controls PR list filtering and pagination.
type ListOpts struct {
	State    PRState     `json:"state,omitempty"`
	Author   string      `json:"author,omitempty"`
	Labels   []string    `json:"labels,omitempty"`
	CI       CIStatus    `json:"ci,omitempty"`
	Review   ReviewState `json:"review,omitempty"`
	Draft    DraftFilter `json:"draft,omitempty"`
	Search   string      `json:"search,omitempty"`
	Sort     string      `json:"sort,omitempty"`
	SortDesc bool        `json:"sort_desc,omitempty"`
	Page     int         `json:"page,omitempty"`
	PerPage  int         `json:"per_page,omitempty"`
}

// DraftFilter controls how draft PRs are included in results.
type DraftFilter string

const (
	DraftInclude DraftFilter = "include"
	DraftExclude DraftFilter = "exclude"
	DraftOnly    DraftFilter = "only"
)

// MergeOpts controls PR merge behavior (post-MVP).
type MergeOpts struct {
	Method        string `json:"method"` // "merge", "squash", "rebase"
	DeleteBranch  bool   `json:"delete_branch"`
	CommitMessage string `json:"commit_message,omitempty"`
}

// ToastLevel indicates the severity of a toast notification.
type ToastLevel string

const (
	ToastInfo    ToastLevel = "info"
	ToastSuccess ToastLevel = "success"
	ToastWarning ToastLevel = "warning"
	ToastError   ToastLevel = "error"
)
