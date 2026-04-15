package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/indrasvat/vivecaka/internal/config"
	"github.com/indrasvat/vivecaka/internal/domain"
)

// RepoState holds per-repo persistent state.
type RepoState struct {
	LastSort    string          `json:"last_sort"`
	LastSortAsc bool            `json:"last_sort_asc"`
	LastFilter  domain.ListOpts `json:"last_filter"`

	// LastViewedPRs maps PR number → last viewed timestamp.
	LastViewedPRs map[int]time.Time     `json:"last_viewed_prs"`
	PRReviews     map[int]PRReviewState `json:"pr_reviews,omitempty"`
}

// PRReviewState stores persisted incremental review state for a single PR.
type PRReviewState struct {
	LastVisitAt      time.Time         `json:"last_visit_at,omitempty"`
	LastVisitHeadSHA string            `json:"last_visit_head_sha,omitempty"`
	LastVisitFiles   map[string]string `json:"last_visit_files,omitempty"`

	LastReviewAt      time.Time         `json:"last_review_at,omitempty"`
	LastReviewHeadSHA string            `json:"last_review_head_sha,omitempty"`
	LastReviewFiles   map[string]string `json:"last_review_files,omitempty"`

	ActiveScope string                     `json:"active_scope,omitempty"`
	ViewedFiles map[string]FileReviewState `json:"viewed_files,omitempty"`
}

// FileReviewState records when and at what digest a file was reviewed.
type FileReviewState struct {
	ViewedAt      time.Time `json:"viewed_at,omitempty"`
	ViewedHeadSHA string    `json:"viewed_head_sha,omitempty"`
	PatchDigest   string    `json:"patch_digest,omitempty"`
}

// StatePath returns the state file path for a given repo.
func StatePath(repo domain.RepoRef) string {
	name := fmt.Sprintf("%s_%s.json", repo.Owner, repo.Name)
	return filepath.Join(config.DataDir(), "state", name)
}

// SaveRepoState writes repo state to disk.
func SaveRepoState(repo domain.RepoRef, state RepoState) error {
	path := StatePath(repo)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}

	out, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, out, 0o644); err != nil {
		return fmt.Errorf("write state: %w", err)
	}
	return os.Rename(tmp, path)
}

// LoadRepoState reads repo state from disk.
// Returns a zero-value RepoState if no state file exists.
func LoadRepoState(repo domain.RepoRef) (RepoState, error) {
	path := StatePath(repo)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return RepoState{}, nil
		}
		return RepoState{}, fmt.Errorf("read state: %w", err)
	}

	var state RepoState
	if err := json.Unmarshal(raw, &state); err != nil {
		return RepoState{}, fmt.Errorf("unmarshal state: %w", err)
	}
	return state, nil
}

// MarkPRViewed records the current time as the last-viewed time for a PR.
func (s *RepoState) MarkPRViewed(number int) {
	if s.LastViewedPRs == nil {
		s.LastViewedPRs = make(map[int]time.Time)
	}
	s.LastViewedPRs[number] = time.Now()
}

// ReviewState returns the review state for a PR, creating the map lazily if needed.
func (s *RepoState) ReviewState(number int) PRReviewState {
	if s.PRReviews == nil {
		return PRReviewState{}
	}
	return s.PRReviews[number]
}

// SetReviewState stores a review state for a PR.
func (s *RepoState) SetReviewState(number int, state PRReviewState) {
	if s.PRReviews == nil {
		s.PRReviews = make(map[int]PRReviewState)
	}
	s.PRReviews[number] = state
}

// IsUnread returns true if the PR has been updated since last viewed.
func (s *RepoState) IsUnread(number int, updatedAt time.Time) bool {
	if s.LastViewedPRs == nil {
		return false // no tracking → not marked as unread
	}
	viewed, ok := s.LastViewedPRs[number]
	if !ok {
		return false // never viewed → not marked (avoids all being unread on first use)
	}
	return updatedAt.After(viewed)
}
