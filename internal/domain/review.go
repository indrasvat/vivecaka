package domain

import "time"

// CommentThread represents a review comment thread on a specific line.
type CommentThread struct {
	ID       string    `json:"id"`
	Path     string    `json:"path"`
	Line     int       `json:"line"`
	Resolved bool      `json:"resolved"`
	Comments []Comment `json:"comments"`
}

// Comment represents a single comment within a thread.
type Comment struct {
	ID        string    `json:"id"`
	Author    string    `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

// InlineCommentInput is used when creating a new inline comment on a diff line.
type InlineCommentInput struct {
	Path      string `json:"path"`                  // File path relative to repo root
	Line      int    `json:"line"`                  // Line number in the diff
	Side      string `json:"side"`                  // "LEFT" or "RIGHT"
	Body      string `json:"body"`                  // Comment body (markdown)
	CommitID  string `json:"commit_id"`             // SHA of the commit to comment on
	InReplyTo string `json:"in_reply_to,omitempty"` // Thread ID if replying
}

// Review represents a review submission.
type Review struct {
	Action ReviewAction `json:"action"`
	Body   string       `json:"body"`
}

// ReviewAction represents the type of review being submitted.
type ReviewAction string

const (
	ReviewActionApprove        ReviewAction = "approve"
	ReviewActionRequestChanges ReviewAction = "request_changes"
	ReviewActionComment        ReviewAction = "comment"
)

func (a ReviewAction) String() string { return string(a) }
