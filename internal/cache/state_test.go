package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

	err := SaveRepoState(repo, state)
	require.NoError(t, err)

	loaded, err := LoadRepoState(repo)
	require.NoError(t, err)

	assert.Equal(t, "author", loaded.LastSort)
	assert.True(t, loaded.LastSortAsc)
	assert.Equal(t, domain.CIPass, loaded.LastFilter.CI)
	_, ok := loaded.LastViewedPRs[42]
	assert.True(t, ok, "expected PR 42 in last viewed")
}

func TestLoadRepoStateMissing(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)

	repo := domain.RepoRef{Owner: "missing", Name: "repo"}
	state, err := LoadRepoState(repo)
	require.NoError(t, err)
	assert.Empty(t, state.LastSort)
}

func TestMarkPRViewed(t *testing.T) {
	state := RepoState{}
	state.MarkPRViewed(42)

	assert.NotNil(t, state.LastViewedPRs, "expected non-nil map")
	_, ok := state.LastViewedPRs[42]
	assert.True(t, ok, "expected PR 42 in map")
}

func TestIsUnread(t *testing.T) {
	viewed := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	state := RepoState{
		LastViewedPRs: map[int]time.Time{42: viewed},
	}

	// Updated after viewed → unread.
	assert.True(t, state.IsUnread(42, viewed.Add(time.Hour)), "expected unread")

	// Updated before viewed → not unread.
	assert.False(t, state.IsUnread(42, viewed.Add(-time.Hour)), "expected not unread")

	// Never viewed → not unread (avoids flood on first use).
	assert.False(t, state.IsUnread(99, time.Now()), "expected not unread for unknown PR")
}
