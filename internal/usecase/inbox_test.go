package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockInboxReader is a test double that returns different PRs per repo.
type mockInboxReader struct {
	prsByRepo map[string][]domain.PR
	failRepos map[string]bool
}

func (m *mockInboxReader) ListPRs(_ context.Context, repo domain.RepoRef, _ domain.ListOpts) ([]domain.PR, error) {
	key := repo.String()
	if m.failRepos[key] {
		return nil, errors.New("API error")
	}
	return m.prsByRepo[key], nil
}

func (m *mockInboxReader) GetPR(_ context.Context, _ domain.RepoRef, _ int) (*domain.PRDetail, error) {
	return nil, nil
}

func (m *mockInboxReader) GetDiff(_ context.Context, _ domain.RepoRef, _ int) (*domain.Diff, error) {
	return nil, nil
}

func (m *mockInboxReader) GetChecks(_ context.Context, _ domain.RepoRef, _ int) ([]domain.Check, error) {
	return nil, nil
}

func (m *mockInboxReader) GetComments(_ context.Context, _ domain.RepoRef, _ int) ([]domain.CommentThread, error) {
	return nil, nil
}

func (m *mockInboxReader) GetPRCount(_ context.Context, _ domain.RepoRef, _ domain.PRState) (int, error) {
	return 0, nil
}

func TestGetInboxPRs_MultiRepo(t *testing.T) {
	now := time.Now()
	repoA := domain.RepoRef{Owner: "org", Name: "alpha"}
	repoB := domain.RepoRef{Owner: "org", Name: "beta"}

	reader := &mockInboxReader{
		prsByRepo: map[string][]domain.PR{
			"org/alpha": {
				{Number: 1, Title: "PR1", Author: "alice", UpdatedAt: now},
				{Number: 2, Title: "PR2", Author: "bob", UpdatedAt: now},
			},
			"org/beta": {
				{Number: 10, Title: "PR10", Author: "carol", UpdatedAt: now},
			},
		},
	}

	uc := NewGetInboxPRs(reader)
	prs, err := uc.Execute(context.Background(), []domain.RepoRef{repoA, repoB})
	require.NoError(t, err)
	require.Len(t, prs, 3)

	// Verify repos are annotated.
	repoSet := make(map[string]int)
	for _, pr := range prs {
		repoSet[pr.Repo.String()]++
	}
	assert.Equal(t, 2, repoSet["org/alpha"])
	assert.Equal(t, 1, repoSet["org/beta"])
}

func TestGetInboxPRs_PartialFailure(t *testing.T) {
	now := time.Now()
	repoA := domain.RepoRef{Owner: "org", Name: "alpha"}
	repoB := domain.RepoRef{Owner: "org", Name: "broken"}

	reader := &mockInboxReader{
		prsByRepo: map[string][]domain.PR{
			"org/alpha": {
				{Number: 1, Title: "PR1", Author: "alice", UpdatedAt: now},
			},
		},
		failRepos: map[string]bool{
			"org/broken": true,
		},
	}

	uc := NewGetInboxPRs(reader)
	prs, err := uc.Execute(context.Background(), []domain.RepoRef{repoA, repoB})
	require.NoError(t, err)
	// Should still get PRs from the successful repo.
	require.Len(t, prs, 1)
	assert.Equal(t, "org/alpha", prs[0].Repo.String())
}

func TestGetInboxPRs_EmptyRepos(t *testing.T) {
	uc := NewGetInboxPRs(&mockInboxReader{})
	prs, err := uc.Execute(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, prs)
}
