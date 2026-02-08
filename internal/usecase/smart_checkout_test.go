package usecase

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/indrasvat/vivecaka/internal/repolocator"
)

// mockRepoManager is a test double for domain.RepoManager.
type mockRepoManager struct {
	checkoutAtCalls     []checkoutAtCall
	cloneRepoCalls      []cloneRepoCall
	createWorktreeCalls []createWorktreeCall
	checkoutAtErr       error
	cloneRepoErr        error
	createWorktreeErr   error
	checkoutBranch      string
}

type checkoutAtCall struct {
	Repo    domain.RepoRef
	Number  int
	WorkDir string
}

type cloneRepoCall struct {
	Repo       domain.RepoRef
	TargetPath string
}

type createWorktreeCall struct {
	RepoPath     string
	Number       int
	Branch       string
	WorktreePath string
}

func (m *mockRepoManager) CheckoutAt(_ context.Context, repo domain.RepoRef, number int, workDir string) (string, error) {
	m.checkoutAtCalls = append(m.checkoutAtCalls, checkoutAtCall{Repo: repo, Number: number, WorkDir: workDir})
	if m.checkoutAtErr != nil {
		return "", m.checkoutAtErr
	}
	branch := m.checkoutBranch
	if branch == "" {
		branch = "feat/test"
	}
	return branch, nil
}

func (m *mockRepoManager) CloneRepo(_ context.Context, repo domain.RepoRef, targetPath string) error {
	m.cloneRepoCalls = append(m.cloneRepoCalls, cloneRepoCall{Repo: repo, TargetPath: targetPath})
	return m.cloneRepoErr
}

func (m *mockRepoManager) CreateWorktree(_ context.Context, repoPath string, number int, branch, worktreePath string) error {
	m.createWorktreeCalls = append(m.createWorktreeCalls, createWorktreeCall{
		RepoPath: repoPath, Number: number, Branch: branch, WorktreePath: worktreePath,
	})
	return m.createWorktreeErr
}

func TestPlanStrategyLocal(t *testing.T) {
	loc := repolocator.NewWithPath(filepath.Join(t.TempDir(), "known.json"))
	rm := &mockRepoManager{}
	uc := NewSmartCheckout(rm, loc)

	repo := domain.RepoRef{Owner: "indrasvat", Name: "vivecaka"}
	ctx := CheckoutContext{
		BrowsingRepo: repo,
		CWDRepo:      repo,
		CWDPath:      "/Users/test/code/vivecaka",
	}

	plan := uc.Plan(ctx, "", false)
	assert.Equal(t, StrategyLocal, plan.Strategy)
	assert.Equal(t, "/Users/test/code/vivecaka", plan.TargetPath)
}

func TestPlanStrategyLocalCaseInsensitive(t *testing.T) {
	loc := repolocator.NewWithPath(filepath.Join(t.TempDir(), "known.json"))
	rm := &mockRepoManager{}
	uc := NewSmartCheckout(rm, loc)

	ctx := CheckoutContext{
		BrowsingRepo: domain.RepoRef{Owner: "Indrasvat", Name: "Vivecaka"},
		CWDRepo:      domain.RepoRef{Owner: "indrasvat", Name: "vivecaka"},
		CWDPath:      "/path",
	}

	plan := uc.Plan(ctx, "", false)
	assert.Equal(t, StrategyLocal, plan.Strategy)
}

func TestPlanStrategyKnownPath(t *testing.T) {
	loc := repolocator.NewWithPath(filepath.Join(t.TempDir(), "known.json"))
	rm := &mockRepoManager{}
	uc := NewSmartCheckout(rm, loc)

	browsingRepo := domain.RepoRef{Owner: "steipete", Name: "CodexBar"}
	ctx := CheckoutContext{
		BrowsingRepo: browsingRepo,
		CWDRepo:      domain.RepoRef{Owner: "indrasvat", Name: "vivecaka"},
		CWDPath:      "/Users/test/code/vivecaka",
	}

	plan := uc.Plan(ctx, "/Users/test/code/steipete/CodexBar", true)
	assert.Equal(t, StrategyKnownPath, plan.Strategy)
	assert.Equal(t, "/Users/test/code/steipete/CodexBar", plan.TargetPath)
	assert.True(t, plan.KnownRepo)
}

func TestPlanStrategyNeedsClone(t *testing.T) {
	loc := repolocator.NewWithPath(filepath.Join(t.TempDir(), "known.json"))
	rm := &mockRepoManager{}
	uc := NewSmartCheckout(rm, loc)

	browsingRepo := domain.RepoRef{Owner: "steipete", Name: "CodexBar"}
	ctx := CheckoutContext{
		BrowsingRepo: browsingRepo,
		CWDRepo:      domain.RepoRef{Owner: "indrasvat", Name: "vivecaka"},
		CWDPath:      "/Users/test/code/vivecaka",
	}

	plan := uc.Plan(ctx, "", false)
	assert.Equal(t, StrategyNeedsClone, plan.Strategy)
	assert.NotEmpty(t, plan.CacheClonePath)
}

func TestPlanNoCWDRepo(t *testing.T) {
	loc := repolocator.NewWithPath(filepath.Join(t.TempDir(), "known.json"))
	rm := &mockRepoManager{}
	uc := NewSmartCheckout(rm, loc)

	ctx := CheckoutContext{
		BrowsingRepo: domain.RepoRef{Owner: "steipete", Name: "CodexBar"},
		CWDPath:      "/tmp/demo",
	}

	plan := uc.Plan(ctx, "", false)
	assert.Equal(t, StrategyNeedsClone, plan.Strategy)
}

func TestExecuteClone(t *testing.T) {
	loc := repolocator.NewWithPath(filepath.Join(t.TempDir(), "known.json"))
	rm := &mockRepoManager{}
	uc := NewSmartCheckout(rm, loc)

	repo := domain.RepoRef{Owner: "steipete", Name: "CodexBar"}
	err := uc.ExecuteClone(context.Background(), repo, "/tmp/clone")
	require.NoError(t, err)
	assert.Len(t, rm.cloneRepoCalls, 1)
	assert.Equal(t, "/tmp/clone", rm.cloneRepoCalls[0].TargetPath)

	// Should be registered in known-repos.
	path, found := loc.Lookup(repo)
	assert.True(t, found)
	assert.Equal(t, "/tmp/clone", path)
}

func TestExecuteCheckout(t *testing.T) {
	loc := repolocator.NewWithPath(filepath.Join(t.TempDir(), "known.json"))
	rm := &mockRepoManager{checkoutBranch: "feat/oauth"}
	uc := NewSmartCheckout(rm, loc)

	repo := domain.RepoRef{Owner: "steipete", Name: "CodexBar"}
	branch, err := uc.ExecuteCheckout(context.Background(), repo, 255, "/some/path")
	require.NoError(t, err)
	assert.Equal(t, "feat/oauth", branch)
	assert.Equal(t, "/some/path", rm.checkoutAtCalls[0].WorkDir)
}

func TestExecuteWorktree(t *testing.T) {
	loc := repolocator.NewWithPath(filepath.Join(t.TempDir(), "known.json"))
	rm := &mockRepoManager{}
	uc := NewSmartCheckout(rm, loc)

	repo := domain.RepoRef{Owner: "indrasvat", Name: "vivecaka"}
	wtPath, err := uc.ExecuteWorktree(context.Background(), repo, 255, "feat/oauth", "/base/path")
	require.NoError(t, err)
	assert.Contains(t, wtPath, ".worktrees")
	assert.Contains(t, wtPath, "pr-255-feat-oauth")
	assert.Len(t, rm.createWorktreeCalls, 1)
}

func TestWorktreePathSanitizesBranch(t *testing.T) {
	loc := repolocator.NewWithPath(filepath.Join(t.TempDir(), "known.json"))
	rm := &mockRepoManager{}
	uc := NewSmartCheckout(rm, loc)

	path := uc.WorktreePath(42, "feature/add:thing", "/base")
	assert.Equal(t, "/base/.worktrees/pr-42-feature-add-thing", path)
}

func TestSanitizeBranchName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"feat/oauth", "feat-oauth"},
		{"simple", "simple"},
		{"path/with/slashes", "path-with-slashes"},
		{"has spaces", "has-spaces"},
		{"has:colons", "has-colons"},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.expected, sanitizeBranchName(tc.input))
	}
}
