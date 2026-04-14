package domain

import "time"

// CommentThread represents a review comment thread on a specific line.
type CommentThread struct {
	ID        string    `json:"id"`                    // Root review comment database ID.
	ThreadID  string    `json:"thread_id,omitempty"`   // GraphQL review-thread ID.
	ReplyToID string    `json:"reply_to_id,omitempty"` // Root review comment database ID used for replies.
	Path      string    `json:"path"`
	Line      int       `json:"line"`
	Resolved  bool      `json:"resolved"`
	Comments  []Comment `json:"comments"`
}

// Comment represents a single comment within a thread.
type Comment struct {
	ID        string    `json:"id"`
	Author    string    `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	URL       string    `json:"url,omitempty"`
}

// DiscussionKind identifies the source/type of a PR discussion item.
type DiscussionKind string

const (
	DiscussionInlineThread DiscussionKind = "inline_thread"
	DiscussionReview       DiscussionKind = "review"
	DiscussionComment      DiscussionKind = "comment"
)

// DiscussionItem is a single item shown in the PR detail Comments tab.
type DiscussionItem struct {
	ID          string         `json:"id"`
	Kind        DiscussionKind `json:"kind"`
	Path        string         `json:"path,omitempty"`
	Line        int            `json:"line,omitempty"`
	Resolved    bool           `json:"resolved,omitempty"`
	ReviewState ReviewState    `json:"review_state,omitempty"`
	StateLabel  string         `json:"state_label,omitempty"`
	ThreadID    string         `json:"thread_id,omitempty"`
	ReplyToID   string         `json:"reply_to_id,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	URL         string         `json:"url,omitempty"`
	Comments    []Comment      `json:"comments"`
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
