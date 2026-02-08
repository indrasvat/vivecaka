package cache

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
	err := Save(repo, prs)
	require.NoError(t, err)

	// Verify file exists.
	path := CachePath(repo)
	_, err = os.Stat(path)
	require.NoError(t, err, "cache file should exist")

	// Load.
	loaded, updated, err := Load(repo)
	require.NoError(t, err)
	assert.False(t, updated.IsZero(), "expected non-zero updated time")
	require.Len(t, loaded, 2)
	assert.Equal(t, "First PR", loaded[0].Title)
	assert.Equal(t, "bob", loaded[1].Author)
}

func TestLoadMissing(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmp)

	repo := domain.RepoRef{Owner: "nonexistent", Name: "repo"}
	prs, updated, err := Load(repo)
	require.NoError(t, err)
	assert.Nil(t, prs, "expected nil PRs")
	assert.True(t, updated.IsZero(), "expected zero time")
}

func TestIsStale(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmp)

	repo := domain.RepoRef{Owner: "test", Name: "stale"}

	// No cache → stale.
	assert.True(t, IsStale(repo, time.Hour), "expected stale for missing cache")

	// Save cache.
	err := Save(repo, []domain.PR{{Number: 1}})
	require.NoError(t, err)

	// Fresh with large TTL.
	assert.False(t, IsStale(repo, time.Hour), "expected fresh with 1h TTL")

	// Stale with tiny TTL.
	assert.True(t, IsStale(repo, time.Nanosecond), "expected stale with 1ns TTL")
}

func TestSaveOverwrite(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmp)

	repo := domain.RepoRef{Owner: "test", Name: "overwrite"}

	// Save initial.
	err := Save(repo, []domain.PR{{Number: 1, Title: "Old"}})
	require.NoError(t, err)

	// Overwrite.
	err = Save(repo, []domain.PR{{Number: 2, Title: "New"}})
	require.NoError(t, err)

	loaded, _, err := Load(repo)
	require.NoError(t, err)
	require.Len(t, loaded, 1)
	assert.Equal(t, "New", loaded[0].Title)
}
