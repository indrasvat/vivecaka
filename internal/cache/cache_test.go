package cache

import (
	"os"
	"testing"
	"time"

	"github.com/indrasvat/vivecaka/internal/domain"
)

func TestSaveAndLoad(t *testing.T) {
	// Use temp dir as XDG cache.
	tmp := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmp)

	repo := domain.RepoRef{Owner: "test", Name: "repo"}
	now := time.Now().Truncate(time.Second)

	prs := []domain.PR{
		{Number: 1, Title: "First PR", Author: "alice", UpdatedAt: now},
		{Number: 2, Title: "Second PR", Author: "bob", UpdatedAt: now},
	}

	// Save.
	if err := Save(repo, prs); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify file exists.
	path := CachePath(repo)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("cache file not found: %v", err)
	}

	// Load.
	loaded, updated, err := Load(repo)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if updated.IsZero() {
		t.Fatal("expected non-zero updated time")
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 PRs, got %d", len(loaded))
	}
	if loaded[0].Title != "First PR" {
		t.Errorf("expected 'First PR', got %q", loaded[0].Title)
	}
	if loaded[1].Author != "bob" {
		t.Errorf("expected 'bob', got %q", loaded[1].Author)
	}
}

func TestLoadMissing(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmp)

	repo := domain.RepoRef{Owner: "nonexistent", Name: "repo"}
	prs, updated, err := Load(repo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prs != nil {
		t.Errorf("expected nil PRs, got %d", len(prs))
	}
	if !updated.IsZero() {
		t.Errorf("expected zero time")
	}
}

func TestIsStale(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmp)

	repo := domain.RepoRef{Owner: "test", Name: "stale"}

	// No cache → stale.
	if !IsStale(repo, time.Hour) {
		t.Error("expected stale for missing cache")
	}

	// Save cache.
	if err := Save(repo, []domain.PR{{Number: 1}}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Fresh with large TTL.
	if IsStale(repo, time.Hour) {
		t.Error("expected fresh with 1h TTL")
	}

	// Stale with tiny TTL.
	if !IsStale(repo, time.Nanosecond) {
		t.Error("expected stale with 1ns TTL")
	}
}

func TestSaveOverwrite(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmp)

	repo := domain.RepoRef{Owner: "test", Name: "overwrite"}

	// Save initial.
	if err := Save(repo, []domain.PR{{Number: 1, Title: "Old"}}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Overwrite.
	if err := Save(repo, []domain.PR{{Number: 2, Title: "New"}}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, _, err := Load(repo)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded) != 1 || loaded[0].Title != "New" {
		t.Errorf("expected overwritten PR, got %v", loaded)
	}
}
