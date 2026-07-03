package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/indrasvat/vivecaka/internal/domain"
)

func TestSaveAndLoadInbox(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	now := time.Now().Truncate(time.Second)
	items := []domain.InboxItem{{
		PR:      domain.PR{Number: 42, Title: "cached", Author: "alice", UpdatedAt: now},
		Repo:    domain.RepoRef{Owner: "owner", Name: "repo"},
		Sources: []domain.InboxSource{domain.InboxSourceFavorite, domain.InboxSourceOwned},
		Score:   123,
		Reason:  "ci",
		Cached:  true,
	}}
	result := domain.InboxResult{Items: items, CachedAt: now}

	require.NoError(t, SaveInbox("indrasvat", result))
	loaded, updated, err := LoadInbox("indrasvat")

	require.NoError(t, err)
	assert.False(t, updated.IsZero())
	require.Len(t, loaded.Items, 1)
	assert.Equal(t, "cached", loaded.Items[0].Title)
	assert.Equal(t, []domain.InboxSource{domain.InboxSourceFavorite, domain.InboxSourceOwned}, loaded.Items[0].Sources)
	assert.True(t, loaded.Items[0].Cached)
}

func TestLoadInboxMissing(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	loaded, updated, err := LoadInbox("missing")

	require.NoError(t, err)
	assert.Empty(t, loaded.Items)
	assert.True(t, updated.IsZero())
}
