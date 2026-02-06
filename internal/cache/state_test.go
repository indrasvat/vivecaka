package cache

import (
	"testing"
	"time"

	"github.com/indrasvat/vivecaka/internal/domain"
)

func TestSaveAndLoadRepoState(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)

	repo := domain.RepoRef{Owner: "test", Name: "state"}

	state := RepoState{
		LastSort:    "author",
		LastSortAsc: true,
		LastFilter: domain.ListOpts{
			State: domain.PRStateOpen,
			CI:    domain.CIPass,
		},
		LastViewedPRs: map[int]time.Time{
			42: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	if err := SaveRepoState(repo, state); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := LoadRepoState(repo)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.LastSort != "author" {
		t.Errorf("expected sort 'author', got %q", loaded.LastSort)
	}
	if !loaded.LastSortAsc {
		t.Error("expected sort asc to be true")
	}
	if loaded.LastFilter.CI != domain.CIPass {
		t.Errorf("expected CI filter, got %v", loaded.LastFilter.CI)
	}
	if _, ok := loaded.LastViewedPRs[42]; !ok {
		t.Error("expected PR 42 in last viewed")
	}
}

func TestLoadRepoStateMissing(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)

	repo := domain.RepoRef{Owner: "missing", Name: "repo"}
	state, err := LoadRepoState(repo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.LastSort != "" {
		t.Errorf("expected empty sort, got %q", state.LastSort)
	}
}

func TestMarkPRViewed(t *testing.T) {
	state := RepoState{}
	state.MarkPRViewed(42)

	if state.LastViewedPRs == nil {
		t.Fatal("expected non-nil map")
	}
	if _, ok := state.LastViewedPRs[42]; !ok {
		t.Error("expected PR 42 in map")
	}
}

func TestIsUnread(t *testing.T) {
	viewed := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	state := RepoState{
		LastViewedPRs: map[int]time.Time{42: viewed},
	}

	// Updated after viewed → unread.
	if !state.IsUnread(42, viewed.Add(time.Hour)) {
		t.Error("expected unread")
	}

	// Updated before viewed → not unread.
	if state.IsUnread(42, viewed.Add(-time.Hour)) {
		t.Error("expected not unread")
	}

	// Never viewed → not unread (avoids flood on first use).
	if state.IsUnread(99, time.Now()) {
		t.Error("expected not unread for unknown PR")
	}
}
