package usecase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/indrasvat/vivecaka/internal/domain"
)

type fakeInboxInsightReader struct {
	query   domain.InboxInsightQuery
	insight *domain.InboxInsight
	err     error
}

func (f *fakeInboxInsightReader) GetInboxInsight(_ context.Context, q domain.InboxInsightQuery) (*domain.InboxInsight, error) {
	f.query = q
	return f.insight, f.err
}

func TestGetInboxInsightPassesBaselineQuery(t *testing.T) {
	repo := domain.RepoRef{Owner: "owner", Name: "repo"}
	reader := &fakeInboxInsightReader{insight: &domain.InboxInsight{Repo: repo, Number: 7, FileDelta: 2}}
	uc := NewGetInboxInsight(reader)

	got, err := uc.Execute(context.Background(), domain.InboxInsightQuery{
		Repo:              repo,
		Number:            7,
		LastReviewHeadSHA: "base",
		LastReviewFiles:   map[string]string{"a.go": "digest"},
	})

	require.NoError(t, err)
	assert.Equal(t, 2, got.FileDelta)
	assert.Equal(t, "base", reader.query.LastReviewHeadSHA)
	assert.Equal(t, map[string]string{"a.go": "digest"}, reader.query.LastReviewFiles)
}

func TestGetInboxInsightNilReaderReturnsEmptyInsight(t *testing.T) {
	got, err := NewGetInboxInsight(nil).Execute(context.Background(), domain.InboxInsightQuery{Number: 3})

	require.NoError(t, err)
	assert.Equal(t, 3, got.Number)
}
