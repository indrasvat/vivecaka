package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/indrasvat/vivecaka/internal/domain"
)

type fakeAttentionInboxReader struct {
	result *domain.InboxResult
	err    error
	query  domain.InboxQuery
}

func (f *fakeAttentionInboxReader) GetInbox(_ context.Context, q domain.InboxQuery) (*domain.InboxResult, error) {
	f.query = q
	return f.result, f.err
}

func TestRankInboxItemsPrefersAttentionFavoriteFailuresAndStability(t *testing.T) {
	now := time.Now()
	items := []domain.InboxItem{
		{PR: domain.PR{Number: 1, Title: "owned", Author: "bot[bot]", CI: domain.CIPass, UpdatedAt: now}, Repo: domain.RepoRef{Owner: "me", Name: "owned"}, Sources: []domain.InboxSource{domain.InboxSourceOwned}},
		{PR: domain.PR{Number: 2, Title: "favorite fail", Author: "dep", CI: domain.CIFail, UpdatedAt: now.Add(-time.Hour)}, Repo: domain.RepoRef{Owner: "me", Name: "fav"}, Sources: []domain.InboxSource{domain.InboxSourceOwned, domain.InboxSourceFavorite}},
		{PR: domain.PR{Number: 3, Title: "attention", Author: "alice", CI: domain.CIPending, Review: domain.ReviewStatus{State: domain.ReviewPending}, UpdatedAt: now.Add(-2 * time.Hour)}, Repo: domain.RepoRef{Owner: "team", Name: "repo"}, Sources: []domain.InboxSource{domain.InboxSourceAttention}},
	}

	RankInboxItems(items, domain.InboxRankBalanced, "me", 7)

	assert.Equal(t, 3, items[0].Number)
	assert.Equal(t, "review", items[0].Reason)
	assert.Equal(t, 2, items[1].Number)
	assert.Equal(t, "ci", items[1].Reason)
	assert.Greater(t, items[0].Score, items[1].Score)
}

func TestRankInboxItemsFavoritesProfileBoostsFavoritesWithoutDroppingAttention(t *testing.T) {
	now := time.Now()
	items := []domain.InboxItem{
		{PR: domain.PR{Number: 1, Author: "alice", CI: domain.CIPass, Review: domain.ReviewStatus{State: domain.ReviewPending}, UpdatedAt: now}, Sources: []domain.InboxSource{domain.InboxSourceAttention}},
		{PR: domain.PR{Number: 2, Author: "dep", CI: domain.CIFail, UpdatedAt: now}, Sources: []domain.InboxSource{domain.InboxSourceFavorite}},
	}

	RankInboxItems(items, domain.InboxRankFavorites, "me", 7)

	assert.Equal(t, 1, items[0].Number, "direct attention remains protected")
	assert.Equal(t, 2, items[1].Number)
	assert.Greater(t, items[1].Score, 500, "favorite profile materially boosts favorites")
}

func TestGetAttentionInboxPassesQueryAndRanksResults(t *testing.T) {
	now := time.Now()
	reader := &fakeAttentionInboxReader{result: &domain.InboxResult{Items: []domain.InboxItem{
		{PR: domain.PR{Number: 1, Author: "bot[bot]", CI: domain.CIPass, UpdatedAt: now}, Repo: domain.RepoRef{Owner: "me", Name: "owned"}, Sources: []domain.InboxSource{domain.InboxSourceOwned}},
		{PR: domain.PR{Number: 2, Author: "alice", CI: domain.CIFail, UpdatedAt: now}, Repo: domain.RepoRef{Owner: "me", Name: "fav"}, Sources: []domain.InboxSource{domain.InboxSourceFavorite}},
	}}}
	uc := NewGetAttentionInbox(reader)

	got, err := uc.Execute(context.Background(), domain.InboxQuery{
		Username:          "me",
		HomeRepo:          domain.RepoRef{Owner: "me", Name: "home"},
		Favorites:         []domain.RepoRef{{Owner: "me", Name: "fav"}},
		OwnedOwner:        "me",
		IncludeOwnedRepos: true,
		RankProfile:       domain.InboxRankBalanced,
		StaleDays:         7,
		Limit:             10,
	})

	require.NoError(t, err)
	require.Len(t, got.Items, 2)
	assert.Equal(t, "me", reader.query.OwnedOwner)
	assert.Equal(t, 2, got.Items[0].Number)
	assert.Equal(t, "ci", got.Items[0].Reason)
}

func TestGetAttentionInboxReturnsEmptyResultOnReaderError(t *testing.T) {
	uc := NewGetAttentionInbox(&fakeAttentionInboxReader{err: errors.New("rate limited")})

	got, err := uc.Execute(context.Background(), domain.InboxQuery{})

	require.Error(t, err)
	assert.Empty(t, got.Items)
}
