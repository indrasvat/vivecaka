package tui

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/indrasvat/vivecaka/internal/cache"
	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/indrasvat/vivecaka/internal/repolocator"
	"github.com/indrasvat/vivecaka/internal/tui/views"
	"github.com/indrasvat/vivecaka/internal/usecase"
)

type commandReader struct {
	prs        []domain.PR
	detail     *domain.PRDetail
	diff       *domain.Diff
	checks     []domain.Check
	comments   []domain.CommentThread
	discussion []domain.DiscussionItem
	count      int
	err        error
	diffErr    error
}

func (m *commandReader) ListPRs(_ context.Context, _ domain.RepoRef, _ domain.ListOpts) ([]domain.PR, error) {
	return m.prs, m.err
}
func (m *commandReader) GetPR(_ context.Context, _ domain.RepoRef, _ int) (*domain.PRDetail, error) {
	return m.detail, m.err
}
func (m *commandReader) GetDiff(_ context.Context, _ domain.RepoRef, _ int) (*domain.Diff, error) {
	return m.diff, m.diffErr
}
func (m *commandReader) GetChecks(_ context.Context, _ domain.RepoRef, _ int) ([]domain.Check, error) {
	return m.checks, m.err
}
func (m *commandReader) GetComments(_ context.Context, _ domain.RepoRef, _ int) ([]domain.CommentThread, error) {
	return m.comments, m.err
}
func (m *commandReader) GetDiscussion(_ context.Context, _ domain.RepoRef, _ int) ([]domain.DiscussionItem, error) {
	return m.discussion, m.err
}
func (m *commandReader) GetPRCount(_ context.Context, _ domain.RepoRef, _ domain.PRState) (int, error) {
	return m.count, m.err
}

type commandReviewer struct{ err error }

func (m *commandReviewer) SubmitReview(_ context.Context, _ domain.RepoRef, _ int, _ domain.Review) error {
	return m.err
}
func (m *commandReviewer) AddComment(_ context.Context, _ domain.RepoRef, _ int, _ domain.InlineCommentInput) error {
	return m.err
}
func (m *commandReviewer) ResolveThread(_ context.Context, _ domain.RepoRef, _ string) error {
	return m.err
}

type commandWriter struct {
	branch string
	err    error
}

func (m *commandWriter) Checkout(_ context.Context, _ domain.RepoRef, _ int) (string, error) {
	return m.branch, m.err
}
func (m *commandWriter) Merge(_ context.Context, _ domain.RepoRef, _ int, _ domain.MergeOpts) error {
	return m.err
}
func (m *commandWriter) UpdateLabels(_ context.Context, _ domain.RepoRef, _ int, _ []string) error {
	return m.err
}

type commandRepoManager struct {
	branch string
	err    error
}

func (m *commandRepoManager) CheckoutAt(_ context.Context, _ domain.RepoRef, _ int, _ string) (string, error) {
	return m.branch, m.err
}
func (m *commandRepoManager) CloneRepo(_ context.Context, _ domain.RepoRef, _ string) error {
	return m.err
}
func (m *commandRepoManager) CreateWorktree(_ context.Context, _ string, _ int, _ string, _ string) error {
	return m.err
}

func runCmd(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	require.NotNil(t, cmd)
	return cmd()
}

func installCommandFakeCLIs(t *testing.T) {
	t.Helper()
	bin := t.TempDir()
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	gh := `#!/bin/sh
case "$1 $2" in
  "api user")
    printf 'octocat\n'
    ;;
  "repo list")
    printf '[{"nameWithOwner":"owner/repo"},{"nameWithOwner":"bad"}]'
    ;;
  "repo view")
    printf '{"name":"repo"}'
    ;;
  *)
    exit 0
    ;;
esac
`
	require.NoError(t, os.WriteFile(filepath.Join(bin, "gh"), []byte(gh), 0o755))

	git := `#!/bin/sh
case "$1 $2 $3" in
  "remote get-url origin")
    printf 'git@github.com:owner/repo.git\n'
    ;;
  "rev-parse --abbrev-ref HEAD")
    printf 'feature/test\n'
    ;;
  *)
    exit 0
    ;;
esac
`
	require.NoError(t, os.WriteFile(filepath.Join(bin, "git"), []byte(git), 0o755))
}

func TestDetectionCommandsReturnTypedMessages(t *testing.T) {
	installCommandFakeCLIs(t)

	repoMsg := runCmd(t, detectRepoCmd()).(views.RepoDetectedMsg)
	assert.Equal(t, domain.RepoRef{Owner: "owner", Name: "repo"}, repoMsg.Repo)
	require.NoError(t, repoMsg.Err)

	userMsg := runCmd(t, detectUserCmd()).(views.UserDetectedMsg)
	assert.Equal(t, "octocat", userMsg.Username)
	require.NoError(t, userMsg.Err)

	branchMsg := runCmd(t, detectBranchCmd()).(views.BranchDetectedMsg)
	assert.Equal(t, "feature/test", branchMsg.Branch)
	require.NoError(t, branchMsg.Err)

	reposMsg := runCmd(t, discoverReposCmd()).(views.ReposDiscoveredMsg)
	assert.Equal(t, []domain.RepoRef{{Owner: "owner", Name: "repo"}}, reposMsg.Repos)
	require.NoError(t, reposMsg.Err)

	validated := runCmd(t, validateRepoCmd(domain.RepoRef{Owner: "owner", Name: "repo"})).(views.RepoValidatedMsg)
	assert.Equal(t, domain.RepoRef{Owner: "owner", Name: "repo"}, validated.Repo)
	require.NoError(t, validated.Err)
}

func TestDataLoadingCommandsPreserveSuccessAndErrorPayloads(t *testing.T) {
	repo := domain.RepoRef{Owner: "owner", Name: "repo"}
	prs := []domain.PR{{Number: 1, Title: "first"}, {Number: 2, Title: "second"}}
	diff := &domain.Diff{Files: []domain.FileDiff{{Path: "a.go"}}}
	detail := &domain.PRDetail{PR: domain.PR{Number: 1, Title: "detail"}}
	reader := &commandReader{prs: prs, diff: diff, detail: detail, count: 9}

	prMsg := runCmd(t, loadPRsCmd(usecase.NewListPRs(reader), repo, domain.ListOpts{})).(views.PRsLoadedMsg)
	assert.Equal(t, prs, prMsg.PRs)
	require.NoError(t, prMsg.Err)

	moreMsg := runCmd(t, loadMorePRsCmd(usecase.NewListPRs(reader), repo, domain.ListOpts{PerPage: 2}, 3)).(views.MorePRsLoadedMsg)
	assert.Equal(t, prs, moreMsg.PRs)
	assert.Equal(t, 3, moreMsg.Page)
	assert.True(t, moreMsg.HasMore)

	countMsg := runCmd(t, loadPRCountCmd(reader, repo, domain.PRStateOpen)).(views.PRCountLoadedMsg)
	assert.Equal(t, 9, countMsg.Total)
	require.NoError(t, countMsg.Err)

	detailMsg := runCmd(t, loadPRDetailCmd(usecase.NewGetPRDetail(reader), repo, 1)).(views.PRDetailLoadedMsg)
	require.NoError(t, detailMsg.Err)
	assert.Equal(t, "detail", detailMsg.Detail.Title)

	diffMsg := runCmd(t, loadDiffCmd(reader, repo, 1)).(views.DiffLoadedMsg)
	assert.Same(t, diff, diffMsg.Diff)
	assert.Equal(t, 1, diffMsg.Number)
	require.NoError(t, diffMsg.Err)

	reader.err = errors.New("list failed")
	errMsg := runCmd(t, loadPRsCmd(usecase.NewListPRs(reader), repo, domain.ListOpts{})).(views.PRsLoadedMsg)
	require.Error(t, errMsg.Err)

	reader.err = nil
	reader.diffErr = errors.New("diff failed")
	diffErrMsg := runCmd(t, loadDiffCmd(reader, repo, 2)).(views.DiffLoadedMsg)
	require.Error(t, diffErrMsg.Err)
}

func TestActionCommandsMapUsecaseResultsToMessages(t *testing.T) {
	repo := domain.RepoRef{Owner: "owner", Name: "repo"}

	reviewMsg := runCmd(t, submitReviewCmd(
		usecase.NewReviewPR(&commandReviewer{}),
		repo,
		42,
		domain.Review{Action: domain.ReviewActionApprove},
	)).(views.ReviewSubmittedMsg)
	require.NoError(t, reviewMsg.Err)

	checkoutMsg := runCmd(t, checkoutPRCmd(
		usecase.NewCheckoutPR(&commandWriter{branch: "feat/test"}),
		repo,
		42,
	)).(views.CheckoutDoneMsg)
	assert.Equal(t, "feat/test", checkoutMsg.Branch)
	require.NoError(t, checkoutMsg.Err)

	commentMsg := runCmd(t, addInlineCommentCmd(
		usecase.NewAddComment(&commandReviewer{}),
		repo,
		42,
		domain.InlineCommentInput{Path: "a.go", Line: 4, Body: "tighten", Side: "RIGHT"},
	)).(views.InlineCommentAddedMsg)
	require.NoError(t, commentMsg.Err)

	resolveMsg := runCmd(t, resolveThreadCmd(
		usecase.NewResolveThread(&commandReviewer{}),
		repo,
		"thread-1",
	)).(resolveThreadDoneMsg)
	assert.Equal(t, "thread-1", resolveMsg.ThreadID)
	require.NoError(t, resolveMsg.Err)

	errMsg := runCmd(t, submitReviewCmd(
		usecase.NewReviewPR(&commandReviewer{err: errors.New("denied")}),
		repo,
		42,
		domain.Review{Action: domain.ReviewActionComment},
	)).(views.ReviewSubmittedMsg)
	require.Error(t, errMsg.Err)
}

func TestCacheAndInboxCommandsUseDomainBoundaries(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	repo := domain.RepoRef{Owner: "owner", Name: "repo"}
	prs := []domain.PR{{Number: 1, Title: "cached"}}
	require.NoError(t, cache.Save(repo, prs))

	cached := runCmd(t, loadCachedPRsCmd(repo)).(cachedPRsLoadedMsg)
	assert.Equal(t, prs, cached.PRs)
	assert.False(t, cached.Updated.IsZero())

	assert.Nil(t, runCmd(t, saveCacheCmd(repo, []domain.PR{{Number: 2, Title: "saved"}})))
	cached = runCmd(t, loadCachedPRsCmd(repo)).(cachedPRsLoadedMsg)
	assert.Equal(t, 2, cached.PRs[0].Number)

	reader := &commandReader{prs: []domain.PR{{Number: 3, Title: "inbox"}}}
	msg := runCmd(t, loadInboxCmd(usecase.NewGetInboxPRs(reader), []domain.RepoRef{repo})).(views.InboxPRsLoadedMsg)
	require.Len(t, msg.PRs, 1)
	assert.Equal(t, repo, msg.PRs[0].Repo)

	reader.err = errors.New("partial failure")
	empty := runCmd(t, loadInboxCmd(usecase.NewGetInboxPRs(reader), []domain.RepoRef{repo})).(views.InboxPRsLoadedMsg)
	assert.Empty(t, empty.PRs)
}

func TestSmartCheckoutCommandsReturnDialogMessages(t *testing.T) {
	repo := domain.RepoRef{Owner: "owner", Name: "repo"}
	loc := repolocator.NewWithPath(filepath.Join(t.TempDir(), "known.json"))
	uc := usecase.NewSmartCheckout(&commandRepoManager{branch: "feat/checkout"}, loc)

	clone := runCmd(t, cloneRepoCmd(uc, repo, 42, "feat/checkout", filepath.Join(t.TempDir(), "repo"))).(views.CloneDoneMsg)
	require.NoError(t, clone.Err)
	assert.NotEmpty(t, clone.Path)

	checkout := runCmd(t, smartCheckoutCmd(uc, repo, 42, "/work/repo", false)).(views.SmartCheckoutDoneMsg)
	require.NoError(t, checkout.Err)
	assert.Equal(t, "feat/checkout", checkout.Branch)
	assert.Equal(t, "/work/repo", checkout.Path)

	worktree := runCmd(t, worktreeCmd(uc, repo, 42, "feat/checkout", "/work/repo")).(views.SmartCheckoutDoneMsg)
	require.NoError(t, worktree.Err)
	assert.Equal(t, "feat/checkout", worktree.Branch)
	assert.Contains(t, worktree.Path, "pr-42-feat-checkout")

	errUC := usecase.NewSmartCheckout(&commandRepoManager{err: errors.New("boom")}, loc)
	cloneErr := runCmd(t, cloneRepoCmd(errUC, repo, 42, "feat", "/tmp/repo")).(views.CloneDoneMsg)
	require.Error(t, cloneErr.Err)
	checkoutErr := runCmd(t, smartCheckoutCmd(errUC, repo, 42, "/work/repo", false)).(views.SmartCheckoutDoneMsg)
	require.Error(t, checkoutErr.Err)
	worktreeErr := runCmd(t, worktreeCmd(errUC, repo, 42, "feat", "/work/repo")).(views.SmartCheckoutDoneMsg)
	require.Error(t, worktreeErr.Err)
}
