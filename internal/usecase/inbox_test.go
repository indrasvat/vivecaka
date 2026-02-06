package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/indrasvat/vivecaka/internal/domain"
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prs) != 3 {
		t.Fatalf("expected 3 PRs, got %d", len(prs))
	}

	// Verify repos are annotated.
	repoSet := make(map[string]int)
	for _, pr := range prs {
		repoSet[pr.Repo.String()]++
	}
	if repoSet["org/alpha"] != 2 {
		t.Errorf("expected 2 PRs from org/alpha, got %d", repoSet["org/alpha"])
	}
	if repoSet["org/beta"] != 1 {
		t.Errorf("expected 1 PR from org/beta, got %d", repoSet["org/beta"])
	}
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should still get PRs from the successful repo.
	if len(prs) != 1 {
		t.Fatalf("expected 1 PR, got %d", len(prs))
	}
	if prs[0].Repo.String() != "org/alpha" {
		t.Errorf("expected repo org/alpha, got %s", prs[0].Repo.String())
	}
}

func TestGetInboxPRs_EmptyRepos(t *testing.T) {
	uc := NewGetInboxPRs(&mockInboxReader{})
	prs, err := uc.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prs) != 0 {
		t.Errorf("expected empty, got %d", len(prs))
	}
}
