package tui

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/indrasvat/vivecaka/internal/cache"
	"github.com/indrasvat/vivecaka/internal/config"
	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/indrasvat/vivecaka/internal/repolocator"
	"github.com/indrasvat/vivecaka/internal/reviewprogress"
	"github.com/indrasvat/vivecaka/internal/tui/core"
	"github.com/indrasvat/vivecaka/internal/tui/views"
	"github.com/indrasvat/vivecaka/internal/usecase"
)

func edgeApp(t *testing.T) *App {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	cfg := config.Default()
	cfg.General.RefreshInterval = 0
	cfg.General.PageSize = 2
	cfg.Repos.Favorites = []string{"owner/repo"}

	reader := &commandReader{
		prs:    []domain.PR{{Number: 1, Title: "first"}},
		detail: sampleDetail(),
		diff:   &domain.Diff{Files: []domain.FileDiff{{Path: "auth/middleware.go"}}},
		count:  1,
	}
	app := New(
		cfg,
		WithVersion("edge"),
		WithReader(reader),
		WithReviewer(&commandReviewer{}),
		WithWriter(&commandWriter{branch: "feat/checkout"}),
		WithRepoManager(&commandRepoManager{branch: "feat/checkout"}),
		WithRepo(domain.RepoRef{Owner: "owner", Name: "repo"}),
	)
	app.ready = true
	app.width = 120
	app.height = 40
	app.banner.Hide()
	app.repoLocator = repolocator.NewWithPath(filepath.Join(t.TempDir(), "known.json"))
	app.smartCheckout = usecase.NewSmartCheckout(app.repoManager, app.repoLocator)
	return app
}

func TestAppOptionsWireUsecasesAndExplicitRepo(t *testing.T) {
	app := edgeApp(t)

	assert.Equal(t, "edge", app.version)
	assert.Equal(t, domain.RepoRef{Owner: "owner", Name: "repo"}, app.repo)
	assert.True(t, app.repoExplicit)
	assert.NotNil(t, app.listPRs)
	assert.NotNil(t, app.getPRDetail)
	assert.NotNil(t, app.getReviewContext)
	assert.NotNil(t, app.reviewPR)
	assert.NotNil(t, app.checkoutPR)
	assert.NotNil(t, app.addComment)
	assert.NotNil(t, app.resolveThread)
	assert.NotNil(t, app.getInboxPRs)
	assert.NotNil(t, app.smartCheckout)
}

func TestAppRepoFavoriteInboxAndValidationHandlers(t *testing.T) {
	app := edgeApp(t)

	updated, cmd := app.handleReposDiscovered(views.ReposDiscoveredMsg{
		Repos: []domain.RepoRef{{Owner: "other", Name: "repo"}},
	})
	app = updated.(*App)
	assert.Nil(t, cmd)
	assert.False(t, app.repoSwitcher.NeedsDiscovery())

	updated, cmd = app.handleReposDiscovered(views.ReposDiscoveredMsg{Err: errors.New("gh failed")})
	app = updated.(*App)
	assert.NotNil(t, cmd)

	app.ensureCWDRepoInFavorites()
	assert.Contains(t, app.collectFavoriteStrings(), "owner/repo")
	updated, cmd = app.handleToggleFavorite(views.ToggleFavoriteMsg{
		Repo:     domain.RepoRef{Owner: "owner", Name: "repo"},
		Favorite: true,
	})
	app = updated.(*App)
	assert.NotNil(t, cmd)
	assert.Contains(t, app.cfg.Repos.Favorites, "owner/repo")

	updated, cmd = app.openInbox()
	app = updated.(*App)
	assert.Equal(t, core.ViewInbox, app.view)
	assert.NotNil(t, cmd)

	updated, cmd = app.handleOpenInboxPR(views.OpenInboxPRMsg{
		Repo:   domain.RepoRef{Owner: "other", Name: "repo"},
		Number: 9,
	})
	app = updated.(*App)
	assert.Equal(t, core.ViewPRDetail, app.view)
	assert.Equal(t, domain.RepoRef{Owner: "other", Name: "repo"}, app.repo)
	assert.NotNil(t, cmd)

	updated, cmd = app.handleRepoValidated(views.RepoValidatedMsg{
		Repo: domain.RepoRef{Owner: "valid", Name: "repo"},
	})
	app = updated.(*App)
	assert.Equal(t, domain.RepoRef{Owner: "valid", Name: "repo"}, app.repo)
	assert.Equal(t, core.ViewPRList, app.view)
	assert.NotNil(t, cmd)

	updated, cmd = app.handleRepoValidated(views.RepoValidatedMsg{
		Repo: domain.RepoRef{Owner: "missing", Name: "repo"},
		Err:  errors.New("not found"),
	})
	require.Same(t, app, updated)
	assert.NotNil(t, cmd)
}

func TestAppActionHandlersUseConfiguredUsecases(t *testing.T) {
	app := edgeApp(t)
	app.view = core.ViewDiff
	app.prDetail.SetDetail(sampleDetail())

	cmd := app.handleAddInlineComment(views.AddInlineCommentMsg{
		Number: 1,
		Input:  domain.InlineCommentInput{Path: "a.go", Line: 2, Body: "note"},
	})
	assert.NotNil(t, cmd)

	cmd = app.handleSubmitReview(views.SubmitReviewMsg{
		Number: 1,
		Review: domain.Review{Action: domain.ReviewActionApprove},
	})
	assert.NotNil(t, cmd)

	cmd = app.handleResolveThread(views.ResolveThreadMsg{ThreadID: "thread-1"})
	assert.NotNil(t, cmd)

	cmd = app.handleResolveThreadDone(resolveThreadDoneMsg{ThreadID: "thread-1"})
	assert.NotNil(t, cmd)
	cmd = app.handleResolveThreadDone(resolveThreadDoneMsg{ThreadID: "thread-1", Err: errors.New("denied")})
	assert.NotNil(t, cmd)

	cmd = app.handleInlineCommentAdded(views.InlineCommentAddedMsg{})
	assert.NotNil(t, cmd)
	cmd = app.handleInlineCommentAdded(views.InlineCommentAddedMsg{Err: errors.New("bad line")})
	assert.NotNil(t, cmd)

	app.view = core.ViewReview
	cmd = app.handleReviewSubmitted(views.ReviewSubmittedMsg{})
	assert.Equal(t, core.ViewPRDetail, app.view)
	assert.NotNil(t, cmd)
	cmd = app.handleReviewSubmitted(views.ReviewSubmittedMsg{Err: errors.New("no permission")})
	assert.NotNil(t, cmd)
}

func TestAppSmartCheckoutStrategiesAndBrowserBranch(t *testing.T) {
	app := edgeApp(t)
	app.view = core.ViewPRDetail
	app.cwdRepo = app.repo
	app.cwdPath = t.TempDir()

	updated, cmd := app.handleSmartCheckout(views.CheckoutPRMsg{Number: 42, Branch: "feat/auth"})
	app = updated.(*App)
	assert.Equal(t, core.ViewSmartCheckout, app.view)
	assert.Nil(t, cmd)

	for _, strategy := range []string{"switch", "worktree", "known-path", "clone-cache", "clone-custom"} {
		updated, cmd = app.handleCheckoutStrategyChosen(views.CheckoutStrategyChosenMsg{
			Strategy: strategy,
			Repo:     app.repo,
			PRNumber: 42,
			Branch:   "feat/auth",
			Path:     filepath.Join(t.TempDir(), strategy),
		})
		app = updated.(*App)
		assert.NotNil(t, cmd, strategy)
	}

	oldOpen := openBrowser
	defer func() { openBrowser = oldOpen }()
	openBrowser = func(string) error { return errors.New("no browser") }
	updated, cmd = app.handleCheckoutStrategyChosen(views.CheckoutStrategyChosenMsg{
		Strategy: "browser",
		Repo:     app.repo,
		PRNumber: 42,
	})
	app = updated.(*App)
	assert.Equal(t, core.ViewPRDetail, app.view)
	assert.NotNil(t, cmd)

	app.checkoutDialog.ShowOptions(app.repo, 42, "feat/auth", usecase.CheckoutPlan{})
	updated, cmd = app.handleCloneDone(views.CloneDoneMsg{Path: t.TempDir()})
	app = updated.(*App)
	assert.NotNil(t, cmd)
	updated, cmd = app.handleCloneDone(views.CloneDoneMsg{Err: errors.New("clone failed")})
	app = updated.(*App)
	assert.Nil(t, cmd)

	model := app.handleSmartCheckoutDone(views.SmartCheckoutDoneMsg{Branch: "feat/auth", Path: app.cwdPath})
	app = model.(*App)
	assert.True(t, app.checkoutDialog.Active())

	model = app.handleSmartCheckoutDone(views.SmartCheckoutDoneMsg{Err: errors.New("checkout failed")})
	app = model.(*App)
	assert.True(t, app.checkoutDialog.Active())
}

func TestAppReviewProgressStateTransitions(t *testing.T) {
	app := edgeApp(t)
	detail := sampleDetail()
	detail.Branch.HeadSHA = "head-1"
	app.prDetail.SetDetail(detail)
	app.currentReviewPR = detail.Number
	app.currentReviewContext = reviewprogress.Build(detail, map[string]string{
		detail.Files[0].Path: "digest-a",
		detail.Files[1].Path: "digest-b",
	}, cache.PRReviewState{ActiveScope: string(reviewprogress.ScopeSinceReview)}, false)
	app.prDetail.SetReviewContext(app.currentReviewContext)
	app.diffView.SetReviewContext(app.currentReviewContext)

	assert.Equal(t, detail.Files[0].Path, app.nextReviewTargetPath(""))
	model := app.handleToggleViewedFile(views.ToggleViewedFileMsg{Path: detail.Files[0].Path})
	app = model.(*App)
	state := app.repoState.ReviewState(detail.Number)
	assert.Contains(t, state.ViewedFiles, detail.Files[0].Path)

	model = app.handleToggleViewedFile(views.ToggleViewedFileMsg{Path: detail.Files[0].Path})
	app = model.(*App)
	state = app.repoState.ReviewState(detail.Number)
	assert.NotContains(t, state.ViewedFiles, detail.Files[0].Path)

	model = app.handleCycleReviewScope()
	app = model.(*App)
	assert.NotEmpty(t, app.repoState.ReviewState(detail.Number).ActiveScope)

	app.view = core.ViewPRDetail
	model = app.handleJumpNextReviewTarget(views.JumpNextReviewTargetMsg{CurrentPath: ""})
	app = model.(*App)
	assert.Equal(t, core.ViewPRDetail, app.view)

	app.finalizeCurrentPRVisit()
	state = app.repoState.ReviewState(detail.Number)
	assert.Equal(t, "head-1", state.LastVisitHeadSHA)

	app.markCurrentPRReviewed()
	state = app.repoState.ReviewState(detail.Number)
	assert.Equal(t, "head-1", state.LastReviewHeadSHA)
	assert.NotEmpty(t, state.ViewedFiles)
}

func TestAppViewDispatchersAndRepoStateHelpers(t *testing.T) {
	app := edgeApp(t)
	app.prDetail.SetDetail(sampleDetail())
	app.diffView.SetDiff(&domain.Diff{Files: []domain.FileDiff{{Path: "a.go"}}})

	for _, view := range []core.ViewState{
		core.ViewPRList, core.ViewPRDetail, core.ViewDiff, core.ViewReview,
		core.ViewRepoSwitch, core.ViewHelp, core.ViewInbox, core.ViewFilter,
		core.ViewConfirm, core.ViewSmartCheckout,
	} {
		app.view = view
		_ = app.dispatchKeyToView(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		_ = app.updateActiveView(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	}

	app.repo = domain.RepoRef{}
	app.loadRepoState()
	app.saveRepoState()

	repo := domain.RepoRef{Owner: "state", Name: "repo"}
	app.repo = repo
	filter := domain.ListOpts{State: domain.PRStateClosed, Author: "alice"}
	require.NoError(t, cache.SaveRepoState(repo, cache.RepoState{LastFilter: filter}))
	app.loadRepoState()
	assert.Equal(t, filter, app.filterOpts)

	app.cwdRepo = repo
	app.cwdPath = t.TempDir()
	assert.Equal(t, app.cwdPath, app.findRepoDir())

	known := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(known, ".git"), 0o755))
	bin := t.TempDir()
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	require.NoError(t, os.WriteFile(filepath.Join(bin, "git"), []byte("#!/bin/sh\nprintf 'git@github.com:state/repo.git\\n'\n"), 0o755))
	app.cwdRepo = domain.RepoRef{Owner: "other", Name: "repo"}
	require.NoError(t, app.repoLocator.Register(repo, known, "manual"))
	assert.Equal(t, known, app.findRepoDir())
}

func TestAppRefreshAndStartRepoLoadPaths(t *testing.T) {
	app := edgeApp(t)
	app.refreshInterval = 2
	app.repo = domain.RepoRef{Owner: "owner", Name: "repo"}
	app.view = core.ViewPRList
	app.prList.SetPRs([]domain.PR{{Number: 1}})

	cmd := app.startRefreshTimer()
	assert.NotNil(t, cmd)
	assert.Equal(t, 2, app.refreshCountdown)

	updated, cmd := app.handleRefreshTick()
	app = updated.(*App)
	assert.NotNil(t, cmd)
	assert.Equal(t, 1, app.refreshCountdown)

	app.refreshPaused = true
	updated, cmd = app.handleRefreshTick()
	app = updated.(*App)
	assert.NotNil(t, cmd)
	assert.Equal(t, 1, app.refreshCountdown)

	app.refreshPaused = false
	updated, cmd = app.handleRefreshTick()
	app = updated.(*App)
	assert.NotNil(t, cmd)
	assert.Equal(t, app.refreshInterval, app.refreshCountdown)
	assert.Equal(t, 1, app.prevPRCount)

	app.listPRs = nil
	cmds := app.startRepoLoad(domain.RepoRef{Owner: "other", Name: "repo"}, true)
	require.Len(t, cmds, 1)
	_, ok := cmds[0]().(viewReadyMsg)
	assert.True(t, ok)

	app.refreshInterval = 0
	assert.Nil(t, app.refreshTick())
	assert.Nil(t, app.startRefreshTimer())
	updated, cmd = app.handleRefreshTick()
	assert.Same(t, app, updated)
	assert.Nil(t, cmd)
}

func TestAppCopyOpenAndBatchURLHandlers(t *testing.T) {
	app := edgeApp(t)

	oldCopy := copyToClipboard
	oldOpen := openBrowser
	defer func() {
		copyToClipboard = oldCopy
		openBrowser = oldOpen
	}()

	var copied string
	copyToClipboard = func(s string) error {
		copied = s
		return nil
	}
	var opened []string
	openBrowser = func(s string) error {
		opened = append(opened, s)
		return nil
	}

	assert.NotNil(t, app.handleCopyURL(views.CopyURLMsg{URL: "https://example.test/1"}))
	assert.Equal(t, "https://example.test/1", copied)
	assert.Nil(t, app.handleOpenBrowser(views.OpenBrowserMsg{URL: "https://example.test/1"}))
	assert.Equal(t, []string{"https://example.test/1"}, opened)

	assert.NotNil(t, app.handleBatchCopyURLs(views.BatchCopyURLsMsg{URLs: []string{"u1", "u2"}}))
	assert.Equal(t, "u1\nu2", copied)
	assert.NotNil(t, app.handleBatchOpenBrowser(views.BatchOpenBrowserMsg{URLs: []string{"u3", "u4"}}))
	assert.Equal(t, []string{"https://example.test/1", "u3", "u4"}, opened)

	app.prevView = core.ViewPRDetail
	app.view = core.ViewSmartCheckout
	assert.NotNil(t, app.handleCopyCdCommand(views.CopyCdCommandMsg{Path: "/tmp/repo"}))
	assert.Equal(t, core.ViewPRDetail, app.view)
	assert.Equal(t, "cd /tmp/repo", copied)

	copyToClipboard = func(string) error { return errors.New("clipboard failed") }
	openBrowser = func(string) error { return errors.New("browser failed") }
	assert.NotNil(t, app.handleCopyURL(views.CopyURLMsg{URL: "bad"}))
	assert.NotNil(t, app.handleBatchCopyURLs(views.BatchCopyURLsMsg{URLs: []string{"bad"}}))
	assert.NotNil(t, app.handleCopyCdCommand(views.CopyCdCommandMsg{Path: "/bad"}))
	assert.NotNil(t, app.handleOpenBrowser(views.OpenBrowserMsg{URL: "bad"}))
	assert.NotNil(t, app.handleBatchOpenBrowser(views.BatchOpenBrowserMsg{URLs: []string{"bad"}}))
}

func TestAppConfirmAndCheckoutResultBranches(t *testing.T) {
	app := edgeApp(t)
	app.view = core.ViewPRDetail
	app.prevView = core.ViewPRList

	updated, cmd := app.handleConfirmResult(views.ConfirmResultMsg{Confirmed: false})
	app = updated.(*App)
	assert.Equal(t, core.ViewPRList, app.view)
	assert.Nil(t, cmd)

	app.view = core.ViewConfirm
	app.prevView = core.ViewPRDetail
	updated, cmd = app.handleConfirmResult(views.ConfirmResultMsg{
		Confirmed: true,
		Action:    views.CheckoutPRMsg{Number: 42, Branch: "feat/auth"},
	})
	app = updated.(*App)
	assert.Equal(t, core.ViewConfirm, app.view)
	assert.NotNil(t, cmd)

	updated, cmd = app.handleConfirmResult(views.ConfirmResultMsg{Confirmed: true, Action: "unknown"})
	app = updated.(*App)
	assert.Equal(t, core.ViewPRDetail, app.view)
	assert.Nil(t, cmd)

	app.view = core.ViewConfirm
	updated, cmd = app.handleCheckoutDone(views.CheckoutDoneMsg{Branch: "feat/auth"})
	app = updated.(*App)
	assert.Equal(t, core.ViewConfirm, app.view)
	assert.Nil(t, cmd)

	updated, cmd = app.handleCheckoutDone(views.CheckoutDoneMsg{Err: errors.New("conflict")})
	app = updated.(*App)
	assert.Equal(t, core.ViewConfirm, app.view)
	assert.Nil(t, cmd)

	app.view = core.ViewPRDetail
	updated, cmd = app.handleCheckoutDone(views.CheckoutDoneMsg{Branch: "feat/auth"})
	app = updated.(*App)
	assert.NotNil(t, cmd)
	updated, cmd = app.handleCheckoutDone(views.CheckoutDoneMsg{Err: errors.New("conflict")})
	require.Same(t, app, updated)
	assert.NotNil(t, cmd)
}

func TestAppReviewContextAndExternalDiffHandlers(t *testing.T) {
	app := edgeApp(t)
	detail := sampleDetail()
	detail.Branch = domain.BranchInfo{Base: "main", Head: "feat/auth", HeadSHA: "head"}
	app.prDetail.SetDetail(detail)
	app.currentReviewPR = detail.Number
	app.view = core.ViewDiff

	diff := &domain.Diff{Files: []domain.FileDiff{{Path: detail.Files[0].Path}}}
	ctx := reviewprogress.Build(detail, map[string]string{detail.Files[0].Path: "digest"}, cache.PRReviewState{}, false)
	model := app.handleReviewContextLoaded(views.ReviewContextLoadedMsg{
		Number:  detail.Number,
		Context: ctx,
		Diff:    diff,
	})
	app = model.(*App)
	assert.Same(t, ctx, app.currentReviewContext)
	assert.Same(t, diff, app.currentReviewDiff)

	model = app.handleReviewContextLoaded(views.ReviewContextLoadedMsg{
		Number:  detail.Number + 1,
		Context: ctx,
		Diff:    diff,
	})
	assert.Same(t, app, model)
	model = app.handleReviewContextLoaded(views.ReviewContextLoadedMsg{
		Number: detail.Number,
		Err:    errors.New("diff failed"),
	})
	assert.Same(t, app, model)

	app.currentReviewDiff = diff
	updated, cmd := app.handleOpenDiff(views.OpenDiffMsg{Number: detail.Number})
	app = updated.(*App)
	assert.Equal(t, core.ViewDiff, app.view)
	assert.Nil(t, cmd)

	app.cfg.Diff.ExternalTool = "bad;tool"
	cmd = app.handleOpenExternalDiff(views.OpenExternalDiffMsg{Number: detail.Number})
	assert.NotNil(t, cmd)

	app.cfg.Diff.ExternalTool = "missing-diff-tool"
	cmd = app.handleOpenExternalDiff(views.OpenExternalDiffMsg{Number: detail.Number})
	assert.NotNil(t, cmd)

	app.cfg.Diff.ExternalTool = ""
	bin := t.TempDir()
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	require.NoError(t, os.WriteFile(filepath.Join(bin, "git"), []byte("#!/bin/sh\nif [ \"$1 $2\" = 'config diff.external' ]; then printf 'mydiff\\n'; fi\n"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(bin, "mydiff"), []byte("#!/bin/sh\nexit 0\n"), 0o755))
	assert.Equal(t, "mydiff", detectDiffTool())

	app.cfg.Diff.ExternalTool = "mydiff"
	cmd = app.handleOpenExternalDiff(views.OpenExternalDiffMsg{Number: detail.Number})
	assert.NotNil(t, cmd)
}
