package domain

import "time"

// RepoRef identifies a GitHub repository.
type RepoRef struct {
	Owner string `json:"owner"`
	Name  string `json:"name"`
}

func (r RepoRef) String() string { return r.Owner + "/" + r.Name }

// PR is the list-level representation of a pull request.
type PR struct {
	Number         int          `json:"number"`
	Title          string       `json:"title"`
	Author         string       `json:"author"`
	State          PRState      `json:"state"`
	Draft          bool         `json:"draft"`
	Branch         BranchInfo   `json:"branch"`
	Labels         []string     `json:"labels"`
	CI             CIStatus     `json:"ci"`
	Review         ReviewStatus `json:"review"`
	UpdatedAt      time.Time    `json:"updated_at"`
	CreatedAt      time.Time    `json:"created_at"`
	URL            string       `json:"url"`
	LastViewedAt   *time.Time   `json:"last_viewed_at,omitempty"`
	LastActivityAt time.Time    `json:"last_activity_at"`
}

// PRState represents the state of a pull request.
type PRState string

const (
	PRStateOpen   PRState = "open"
	PRStateClosed PRState = "closed"
	PRStateMerged PRState = "merged"
)

func (s PRState) String() string { return string(s) }

// CIStatus represents the aggregate CI check status.
type CIStatus string

const (
	CIPass    CIStatus = "pass"
	CIFail    CIStatus = "fail"
	CIPending CIStatus = "pending"
	CISkipped CIStatus = "skipped"
	CINone    CIStatus = "none"
)

func (s CIStatus) String() string { return string(s) }

// ReviewStatus represents the aggregate review state.
type ReviewStatus struct {
	State    ReviewState `json:"state"`
	Approved int         `json:"approved"`
	Total    int         `json:"total"`
}

// ReviewState represents the review verdict.
type ReviewState string

const (
	ReviewApproved         ReviewState = "approved"
	ReviewChangesRequested ReviewState = "changes_requested"
	ReviewPending          ReviewState = "pending"
	ReviewNone             ReviewState = "none"
)

func (s ReviewState) String() string { return string(s) }

// PRDetail is the full representation for the detail view.
type PRDetail struct {
	PR
	Body      string          `json:"body"`
	Assignees []string        `json:"assignees"`
	Reviewers []ReviewerInfo  `json:"reviewers"`
	Checks    []Check         `json:"checks"`
	Files     []FileChange    `json:"files"`
	Comments  []CommentThread `json:"comments"`
}

// BranchInfo represents head and base branches.
type BranchInfo struct {
	Head string `json:"head"`
	Base string `json:"base"`
}

// ReviewerInfo represents a reviewer and their verdict.
type ReviewerInfo struct {
	Login string      `json:"login"`
	State ReviewState `json:"state"`
}

// Check represents a CI check result.
type Check struct {
	Name     string        `json:"name"`
	Status   CIStatus      `json:"status"`
	Duration time.Duration `json:"duration"`
	URL      string        `json:"url"`
}

// FileChange represents a file changed in a PR.
type FileChange struct {
	Path      string `json:"path"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Status    string `json:"status"` // added, modified, removed, renamed
}
