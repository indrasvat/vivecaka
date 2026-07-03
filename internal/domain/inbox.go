package domain

import (
	"context"
	"time"
)

// InboxSource identifies why a PR belongs in the attention inbox.
type InboxSource string

const (
	InboxSourceAttention InboxSource = "attention"
	InboxSourceAssigned  InboxSource = "assigned"
	InboxSourceFavorite  InboxSource = "favorite"
	InboxSourceOwned     InboxSource = "owned"
	InboxSourceHome      InboxSource = "home"
)

// InboxRankProfile controls how the inbox prioritizes rows.
type InboxRankProfile string

const (
	InboxRankBalanced   InboxRankProfile = "balanced"
	InboxRankFavorites  InboxRankProfile = "favorites"
	InboxRankStale      InboxRankProfile = "stale"
	InboxRankMergeReady InboxRankProfile = "merge-ready"
)

// InboxLocalState describes whether local repo operations are immediately available.
type InboxLocalState string

const (
	InboxLocalUnknown InboxLocalState = "unknown"
	InboxLocalReady   InboxLocalState = "ready"
	InboxLocalAPIOnly InboxLocalState = "api-only"
)

// InboxItem is a ranked, source-annotated PR row for the attention inbox.
type InboxItem struct {
	PR
	Repo       RepoRef         `json:"repo"`
	Sources    []InboxSource   `json:"sources"`
	Score      int             `json:"score"`
	Reason     string          `json:"reason"`
	LocalState InboxLocalState `json:"local_state"`
	Cached     bool            `json:"cached"`
	Enriched   bool            `json:"enriched"`
}

// InboxInsightQuery asks for the lazy, selected-row insight strip for one PR.
type InboxInsightQuery struct {
	Repo              RepoRef
	Number            int
	LastReviewHeadSHA string
	LastReviewFiles   map[string]string
	LocalState        InboxLocalState
}

// InboxInsight is the compact delta data shown below the selected Inbox row.
type InboxInsight struct {
	Repo              RepoRef         `json:"repo"`
	Number            int             `json:"number"`
	HeadSHA           string          `json:"head_sha,omitempty"`
	CommitDelta       int             `json:"commit_delta,omitempty"`
	FileDelta         int             `json:"file_delta,omitempty"`
	UnresolvedThreads int             `json:"unresolved_threads,omitempty"`
	HasReviewBaseline bool            `json:"has_review_baseline,omitempty"`
	LocalState        InboxLocalState `json:"local_state,omitempty"`
	FetchedAt         time.Time       `json:"fetched_at,omitempty"`
}

// InboxSourceStatus records fetch timing/error state for one source gate.
type InboxSourceStatus struct {
	Source  InboxSource   `json:"source"`
	Label   string        `json:"label"`
	Count   int           `json:"count"`
	Elapsed time.Duration `json:"elapsed"`
	Err     string        `json:"err,omitempty"`
}

// InboxRateLimit captures the GitHub budget visible before source fetches.
type InboxRateLimit struct {
	SearchLimit      int       `json:"search_limit"`
	SearchRemaining  int       `json:"search_remaining"`
	SearchResetAt    time.Time `json:"search_reset_at"`
	GraphQLLimit     int       `json:"graphql_limit"`
	GraphQLRemaining int       `json:"graphql_remaining"`
	GraphQLResetAt   time.Time `json:"graphql_reset_at"`
}

// InboxQuery defines the personal source pool for the attention inbox.
type InboxQuery struct {
	Username          string
	HomeRepo          RepoRef
	Favorites         []RepoRef
	OwnedOwner        string
	IncludeOwnedRepos bool
	RankProfile       InboxRankProfile
	StaleDays         int
	Limit             int
	SourceTimeout     time.Duration
	RateLowWatermark  int
}

// InboxResult is the full output of an inbox fetch.
type InboxResult struct {
	Items    []InboxItem         `json:"items"`
	Sources  []InboxSourceStatus `json:"sources"`
	Rate     InboxRateLimit      `json:"rate"`
	CachedAt time.Time           `json:"cached_at,omitempty"`
}

// InboxReader provides fast, cross-repo pull request candidates.
type InboxReader interface {
	GetInbox(context.Context, InboxQuery) (*InboxResult, error)
}

// InboxInsightReader provides lazy, per-row insight data for the selected PR.
type InboxInsightReader interface {
	GetInboxInsight(context.Context, InboxInsightQuery) (*InboxInsight, error)
}
