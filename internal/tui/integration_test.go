package tui

import (
	"context"
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/indrasvat/vivecaka/internal/config"
	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/indrasvat/vivecaka/internal/tui/components"
	"github.com/indrasvat/vivecaka/internal/tui/core"
	"github.com/indrasvat/vivecaka/internal/tui/views"
)

// -- test setup ------------------------------------------------------------

func init() {
	// Stub out platform functions so tests never open a real browser or clipboard.
	openBrowser = func(string) error { return nil }
	copyToClipboard = func(string) error { return nil }
}

// -- helpers ---------------------------------------------------------------

func integrationApp() *App {
	cfg := config.Default()
	cfg.General.RefreshInterval = 0 // disable auto-refresh in tests
	return New(cfg, WithVersion("test-integration"))
}

func readyApp() *App {
	app := integrationApp()
	app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	return app
}

func samplePRs() []domain.PR {
	return []domain.PR{
		{
			Number: 1, Title: "Add authentication", Author: "alice",
			State: domain.PRStateOpen, Draft: false,
			Branch: domain.BranchInfo{Head: "feat/auth", Base: "main"},
			Labels: []string{"enhancement"},
			CI:     domain.CIPass,
			Review: domain.ReviewStatus{State: domain.ReviewApproved},
			URL:    "https://github.com/test/repo/pull/1",
		},
		{
			Number: 2, Title: "Fix login bug", Author: "bob",
			State: domain.PRStateOpen, Draft: true,
			Branch: domain.BranchInfo{Head: "fix/login", Base: "main"},
			CI:     domain.CIFail,
			Review: domain.ReviewStatus{State: domain.ReviewChangesRequested},
			URL:    "https://github.com/test/repo/pull/2",
		},
		{
			Number: 3, Title: "Update docs", Author: "charlie",
			State: domain.PRStateOpen, Draft: false,
			CI:  domain.CIPending,
			URL: "https://github.com/test/repo/pull/3",
		},
	}
}

func sampleDetail() *domain.PRDetail {
	return &domain.PRDetail{
		PR: domain.PR{
			Number: 1, Title: "Add authentication", Author: "alice",
			State:  domain.PRStateOpen,
			Branch: domain.BranchInfo{Head: "feat/auth", Base: "main"},
			Labels: []string{"enhancement"},
			CI:     domain.CIPass,
			URL:    "https://github.com/test/repo/pull/1",
		},
		Body:      "This PR adds **OAuth2** authentication.",
		Assignees: []string{"alice"},
		Reviewers: []domain.ReviewerInfo{
			{Login: "bob", State: domain.ReviewApproved},
		},
		Files: []domain.FileChange{
			{Path: "auth/middleware.go", Additions: 120, Deletions: 5},
			{Path: "cmd/server/main.go", Additions: 3, Deletions: 1},
		},
		Checks: []domain.Check{
			{Name: "ci/build", Status: domain.CIPass, Duration: 45 * time.Second},
			{Name: "ci/test", Status: domain.CIPending},
		},
	}
}

// -- full-flow integration tests -------------------------------------------

func TestIntegrationInitToRepoDetectToPRList(t *testing.T) {
	app := readyApp()

	// 1. Banner is shown on startup.
	assert.Equal(t, core.ViewBanner, app.view)
	assert.True(t, app.banner.Visible())

	// 2. Repo detected while banner is visible (no reader → goes to PRList).
	updated, _ := app.Update(views.RepoDetectedMsg{
		Repo: domain.RepoRef{Owner: "test", Name: "repo"},
	})
	app = updated.(*App)
	assert.Equal(t, "test", app.repo.Owner)
	assert.Equal(t, "repo", app.repo.Name)

	// 3. Without a reader, the app goes directly to PRList.
	// Simulate manual PR loading.
	updated, _ = app.Update(views.PRsLoadedMsg{PRs: samplePRs()})
	app = updated.(*App)
	assert.Equal(t, 3, app.prList.TotalPRs())
	assert.Equal(t, core.ViewPRList, app.view)

	// 4. PR list renders with actual PR content.
	view := app.View()
	assert.Contains(t, view, "Add authentication", "should show PR title")
	assert.Contains(t, view, "alice", "should show PR author")
}

func TestIntegrationBannerThenPRsThenDismiss(t *testing.T) {
	app := readyApp()
	assert.Equal(t, core.ViewBanner, app.view)

	// PRs loaded while banner visible (simulating async load).
	updated, _ := app.Update(views.PRsLoadedMsg{PRs: samplePRs()})
	app = updated.(*App)
	// View stays on banner since banner is visible.
	assert.Equal(t, core.ViewBanner, app.view, "should stay on banner while visible")
	assert.Equal(t, 3, app.prList.TotalPRs())

	// Dismiss banner → should jump to PR list since PRs are already loaded.
	updated, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	app = updated.(*App)
	assert.Equal(t, core.ViewPRList, app.view, "should jump to PR list")
	assert.NotNil(t, cmd, "should return ClearScreen cmd")
}

func TestIntegrationBannerDismissToLoading(t *testing.T) {
	app := readyApp()

	// Dismiss banner BEFORE PRs arrive → should go to loading view.
	updated, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	app = updated.(*App)
	assert.Equal(t, core.ViewLoading, app.view, "should show loading when PRs not yet loaded")

	// Now PRs arrive.
	updated, _ = app.Update(views.PRsLoadedMsg{PRs: samplePRs()})
	app = updated.(*App)
	assert.Equal(t, core.ViewPRList, app.view, "should transition to PR list")
}

func TestIntegrationOpenPRToDetail(t *testing.T) {
	app := readyApp()
	app.banner.Hide()
	app.view = core.ViewLoading

	// Load PRs.
	updated, _ := app.Update(views.PRsLoadedMsg{PRs: samplePRs()})
	app = updated.(*App)
	assert.Equal(t, core.ViewPRList, app.view)

	// Open PR #1.
	updated, _ = app.Update(views.OpenPRMsg{Number: 1})
	app = updated.(*App)
	assert.Equal(t, core.ViewPRDetail, app.view)

	// Detail loading view should mention the PR number.
	view := app.View()
	assert.Contains(t, view, "#1", "loading view should show PR number")

	// Detail arrives.
	updated, _ = app.Update(views.PRDetailLoadedMsg{Detail: sampleDetail()})
	app = updated.(*App)

	// Detail view renders with correct content.
	view = app.View()
	assert.Contains(t, view, "Add authentication", "should show PR title in detail")
	assert.Contains(t, view, "alice", "should show PR author in detail")
}

func TestIntegrationOpenDiffFromDetail(t *testing.T) {
	app := readyApp()
	app.banner.Hide()
	app.view = core.ViewPRDetail
	app.prDetail.SetDetail(sampleDetail())
	app.repo = domain.RepoRef{Owner: "test", Name: "repo"}

	// Open diff.
	updated, _ := app.Update(views.OpenDiffMsg{Number: 1})
	app = updated.(*App)
	assert.Equal(t, core.ViewDiff, app.view)

	// Diff loaded.
	diff := &domain.Diff{
		Files: []domain.FileDiff{
			{
				Path: "auth/middleware.go",
				Hunks: []domain.Hunk{
					{
						Header: "@@ -0,0 +1,5 @@",
						Lines: []domain.DiffLine{
							{Type: domain.DiffAdd, Content: "package auth", NewNum: 1},
							{Type: domain.DiffAdd, Content: "func Auth() {}", NewNum: 2},
						},
					},
				},
			},
		},
	}
	updated, _ = app.Update(views.DiffLoadedMsg{Diff: diff})
	app = updated.(*App)

	view := app.View()
	assert.Contains(t, view, "auth/middleware.go", "diff should show file path")
}

func TestIntegrationReviewFlow(t *testing.T) {
	app := readyApp()
	app.banner.Hide()
	app.view = core.ViewPRDetail
	app.prDetail.SetDetail(sampleDetail())

	// Start review.
	updated, cmd := app.Update(views.StartReviewMsg{Number: 1})
	app = updated.(*App)
	assert.Equal(t, core.ViewReview, app.view)
	assert.NotNil(t, cmd, "Init should return a cmd")

	// Review submitted.
	updated, cmd = app.Update(views.ReviewSubmittedMsg{})
	app = updated.(*App)
	assert.Equal(t, core.ViewPRDetail, app.view, "should return to detail after review")
	assert.NotNil(t, cmd, "should return toast cmd")
}

func TestIntegrationReviewError(t *testing.T) {
	app := readyApp()
	app.banner.Hide()
	app.view = core.ViewReview

	updated, cmd := app.Update(views.ReviewSubmittedMsg{Err: fmt.Errorf("permission denied")})
	app = updated.(*App)
	assert.NotNil(t, cmd, "should return error toast cmd")
	// View stays on review since there's an error, but actually the code goes to ViewPRDetail always... let me check.
	// Actually looking at handleReviewSubmitted, on error it shows toast but doesn't change view.
	// On success it sets view to ViewPRDetail.
	assert.Equal(t, core.ViewReview, app.view, "should stay on review on error")
}

func TestIntegrationRepoDetectedError(t *testing.T) {
	app := readyApp()

	updated, cmd := app.Update(views.RepoDetectedMsg{
		Err: fmt.Errorf("no git remote found"),
	})
	app = updated.(*App)
	assert.Equal(t, core.ViewPRList, app.view, "should go to PR list even on error")
	assert.NotNil(t, cmd, "should return error toast cmd")
}

func TestIntegrationPRsLoadedError(t *testing.T) {
	app := readyApp()
	app.banner.Hide()
	app.view = core.ViewLoading

	updated, cmd := app.Update(views.PRsLoadedMsg{
		Err: fmt.Errorf("API rate limit exceeded"),
	})
	app = updated.(*App)
	assert.Equal(t, core.ViewPRList, app.view, "should transition to PR list on error")
	assert.NotNil(t, cmd, "should return error toast cmd")
}

func TestIntegrationNavigationFlow(t *testing.T) {
	app := readyApp()
	app.banner.Hide()
	app.view = core.ViewPRList
	app.prList.SetPRs(samplePRs())

	// PR list → Help.
	updated, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	app = updated.(*App)
	assert.Equal(t, core.ViewHelp, app.view)
	assert.Equal(t, core.ViewPRList, app.prevView)

	// Help → back to PR list.
	updated, _ = app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	app = updated.(*App)
	assert.Equal(t, core.ViewPRList, app.view)

	// PR list → Repo Switcher.
	updated, _ = app.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	app = updated.(*App)
	assert.Equal(t, core.ViewRepoSwitch, app.view)
	assert.Equal(t, core.ViewPRList, app.prevView)

	// Repo switcher → back to PR list (intercepted, needs message cycle).
	updated, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	app = updated.(*App)
	if cmd != nil {
		updated, _ = app.Update(cmd())
		app = updated.(*App)
	}
	assert.Equal(t, core.ViewPRList, app.view)
}

func TestIntegrationDetailToDiffToDetailBack(t *testing.T) {
	app := readyApp()
	app.banner.Hide()
	app.view = core.ViewPRDetail
	app.prDetail.SetDetail(sampleDetail())

	// Detail → Diff.
	updated, _ := app.Update(views.OpenDiffMsg{Number: 1})
	app = updated.(*App)
	assert.Equal(t, core.ViewDiff, app.view)

	// Diff → back to Detail.
	updated, _ = app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	app = updated.(*App)
	assert.Equal(t, core.ViewPRDetail, app.view)

	// Detail → back to PR list.
	updated, _ = app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	app = updated.(*App)
	assert.Equal(t, core.ViewPRList, app.view)
}

func TestIntegrationThemeCycleUpdatesRendering(t *testing.T) {
	app := readyApp()
	app.banner.Hide()
	app.view = core.ViewPRList
	app.prList.SetPRs(samplePRs())

	origTheme := app.theme.Name

	// Cycle theme.
	updated, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T'}})
	app = updated.(*App)
	assert.NotEqual(t, origTheme, app.theme.Name)

	// View still renders correctly after theme change.
	view := app.View()
	assert.NotEmpty(t, view)
}

func TestIntegrationUserDetected(t *testing.T) {
	app := readyApp()

	updated, _ := app.Update(views.UserDetectedMsg{Username: "testuser"})
	app = updated.(*App)
	assert.Equal(t, "testuser", app.username)
}

func TestIntegrationUserDetectedError(t *testing.T) {
	app := readyApp()

	updated, cmd := app.Update(views.UserDetectedMsg{
		Err: fmt.Errorf("gh auth not configured"),
	})
	app = updated.(*App)
	assert.Empty(t, app.username, "username should remain empty on error")
	assert.NotNil(t, cmd, "should return warning toast cmd")
}

func TestIntegrationBranchDetected(t *testing.T) {
	app := readyApp()
	app.banner.Hide()
	app.view = core.ViewPRList

	updated, _ := app.Update(views.BranchDetectedMsg{Branch: "feat/auth"})
	app = updated.(*App)

	view := app.View()
	assert.Contains(t, view, "feat/auth", "header should show detected branch")
}

func TestIntegrationPRDetailError(t *testing.T) {
	app := readyApp()
	app.banner.Hide()
	app.view = core.ViewPRDetail

	updated, cmd := app.Update(views.PRDetailLoadedMsg{
		Err: fmt.Errorf("PR not found"),
	})
	app = updated.(*App)
	assert.Equal(t, core.ViewPRDetail, app.view, "should stay on detail view")
	assert.NotNil(t, cmd, "should return error toast cmd")
}

func TestIntegrationFilterApply(t *testing.T) {
	app := readyApp()
	app.banner.Hide()
	app.view = core.ViewFilter
	app.prevView = core.ViewPRList

	newOpts := domain.ListOpts{
		State: domain.PRStateOpen,
		Draft: domain.DraftExclude,
	}

	updated, _ := app.Update(views.ApplyFilterMsg{Opts: newOpts})
	app = updated.(*App)
	assert.Equal(t, core.ViewPRList, app.view, "should return to PR list after filter apply")
	assert.Equal(t, domain.DraftExclude, app.filterOpts.Draft)
}

func TestIntegrationSwitchRepo(t *testing.T) {
	app := readyApp()
	app.banner.Hide()
	app.view = core.ViewRepoSwitch
	app.repo = domain.RepoRef{Owner: "test", Name: "repo1"}

	updated, _ := app.Update(views.SwitchRepoMsg{
		Repo: domain.RepoRef{Owner: "other", Name: "repo2"},
	})
	app = updated.(*App)
	assert.Equal(t, core.ViewPRList, app.view, "should go to PR list after repo switch")
	assert.Equal(t, "other", app.repo.Owner)
	assert.Equal(t, "repo2", app.repo.Name)
}

func TestIntegrationBannerAutoDismiss(t *testing.T) {
	app := readyApp()
	assert.True(t, app.banner.Visible())

	// Simulate auto-dismiss.
	updated, cmd := app.Update(components.BannerDismissMsg{})
	app = updated.(*App)
	assert.False(t, app.banner.Visible())
	assert.NotEqual(t, core.ViewBanner, app.view)
	assert.NotNil(t, cmd, "should return ClearScreen+loadingTick")
}

func TestIntegrationCheckoutConfirmFlow(t *testing.T) {
	app := readyApp()
	app.banner.Hide()
	app.view = core.ViewPRDetail
	// DR-12: Set cwdRepo = repo so StrategyLocal fallback routes to legacy confirm dialog.
	app.repo = domain.RepoRef{Owner: "test", Name: "repo"}
	app.cwdRepo = app.repo

	// Checkout request shows confirm dialog (legacy path when no RepoManager).
	updated, _ := app.Update(views.CheckoutPRMsg{Number: 42, Branch: "feat/auth"})
	app = updated.(*App)
	assert.Equal(t, core.ViewConfirm, app.view)

	// Cancel.
	updated, _ = app.Update(views.ConfirmResultMsg{Confirmed: false})
	app = updated.(*App)
	assert.Equal(t, core.ViewPRDetail, app.view, "should return to previous view on cancel")
}

func TestIntegrationCheckoutDoneSuccess(t *testing.T) {
	app := readyApp()
	app.banner.Hide()
	app.view = core.ViewConfirm
	app.prevView = core.ViewPRDetail

	updated, _ := app.Update(views.CheckoutDoneMsg{Branch: "feat/auth"})
	app = updated.(*App)
	// Confirm dialog shows success result.
	assert.Equal(t, core.ViewConfirm, app.view, "dialog stays open showing result")
}

func TestIntegrationCheckoutDoneError(t *testing.T) {
	app := readyApp()
	app.banner.Hide()
	app.view = core.ViewConfirm

	updated, _ := app.Update(views.CheckoutDoneMsg{Err: fmt.Errorf("branch conflict")})
	app = updated.(*App)
	// Confirm dialog shows error result.
	assert.Equal(t, core.ViewConfirm, app.view, "dialog stays open showing error")
}

func TestIntegrationInlineCommentAdded(t *testing.T) {
	app := readyApp()
	app.banner.Hide()
	app.view = core.ViewDiff

	_, cmd := app.Update(views.InlineCommentAddedMsg{})
	assert.NotNil(t, cmd, "should return success toast cmd")
}

func TestIntegrationInlineCommentError(t *testing.T) {
	app := readyApp()
	app.banner.Hide()
	app.view = core.ViewDiff

	_, cmd := app.Update(views.InlineCommentAddedMsg{Err: fmt.Errorf("cannot comment")})
	assert.NotNil(t, cmd, "should return error toast cmd")
}

func TestIntegrationQuitFromAnyView(t *testing.T) {
	// Views with text input intercept all keys — 'q' types into search,
	// so only Ctrl+C can quit from those views.
	textInputViews := map[core.ViewState]bool{
		core.ViewRepoSwitch: true,
		core.ViewFilter:     true,
		core.ViewReview:     true,
	}

	allViews := []core.ViewState{
		core.ViewPRList, core.ViewPRDetail, core.ViewDiff,
		core.ViewReview, core.ViewHelp, core.ViewRepoSwitch,
		core.ViewInbox, core.ViewFilter,
	}
	for _, v := range allViews {
		app := readyApp()
		app.banner.Hide()
		app.view = v

		if textInputViews[v] {
			// 'q' should NOT quit — it's intercepted by the view.
			_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
			assert.Nil(t, cmd, "q from text-input view %d should NOT quit", v)

			// Ctrl+C should still quit.
			_, cmd = app.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
			require.NotNil(t, cmd, "ctrl+c from view %d should return quit cmd", v)
		} else {
			_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
			require.NotNil(t, cmd, "quit from view %d should return cmd", v)
		}
	}
}

// -- content verification tests --------------------------------------------

func TestIntegrationPRListRenderContent(t *testing.T) {
	app := readyApp()
	app.banner.Hide()
	app.view = core.ViewPRList
	app.prList.SetPRs(samplePRs())

	view := app.View()

	// Should contain PR titles.
	assert.Contains(t, view, "Add authentication")
	assert.Contains(t, view, "Fix login bug")
	assert.Contains(t, view, "Update docs")

	// Should contain authors.
	assert.Contains(t, view, "alice")
	assert.Contains(t, view, "bob")

	// Draft indicator.
	assert.Contains(t, view, "DRAFT", "draft PR should show DRAFT indicator")
}

func TestIntegrationDetailRenderContent(t *testing.T) {
	app := readyApp()
	app.banner.Hide()
	app.view = core.ViewPRDetail
	app.prDetail.SetDetail(sampleDetail())

	view := app.View()

	// Should contain PR metadata.
	assert.Contains(t, view, "Add authentication")
	assert.Contains(t, view, "alice")
	assert.Contains(t, view, "#1", "should show PR number")
}

func TestIntegrationHelpRenderContent(t *testing.T) {
	app := readyApp()
	app.banner.Hide()
	app.view = core.ViewHelp
	app.prevView = core.ViewPRList
	app.helpOverlay.SetContext(core.ViewPRList)

	view := app.View()

	// Help should show key bindings.
	assert.Contains(t, view, "q", "should show quit key")
	assert.Contains(t, view, "?", "should show help key")
}

func TestIntegrationLoadingRenderContent(t *testing.T) {
	app := readyApp()
	app.banner.Hide()
	app.view = core.ViewLoading

	view := app.View()

	assert.Contains(t, view, "Loading", "loading view should show Loading text")
}

func TestIntegrationRepoSwitcherRenderContent(t *testing.T) {
	app := readyApp()
	app.banner.Hide()
	app.view = core.ViewRepoSwitch

	view := app.View()

	assert.Contains(t, view, "Switch", "repo switcher should show title")
}

func TestIntegrationFullFlowEndToEnd(t *testing.T) {
	app := integrationApp()

	// 1. WindowSize → ready
	updated, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = updated.(*App)
	assert.True(t, app.ready)

	// 2. Repo + user detected
	updated, _ = app.Update(views.RepoDetectedMsg{
		Repo: domain.RepoRef{Owner: "test", Name: "repo"},
	})
	app = updated.(*App)
	updated, _ = app.Update(views.UserDetectedMsg{Username: "testuser"})
	app = updated.(*App)
	assert.Equal(t, "testuser", app.username)

	// 3. PRs loaded
	updated, _ = app.Update(views.PRsLoadedMsg{PRs: samplePRs()})
	app = updated.(*App)
	assert.Equal(t, 3, app.prList.TotalPRs())

	// 4. Dismiss banner → PR list
	updated, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	app = updated.(*App)
	assert.Equal(t, core.ViewPRList, app.view)

	// 5. Verify PR list content
	view := app.View()
	assert.Contains(t, view, "Add authentication")

	// 6. Open PR
	updated, _ = app.Update(views.OpenPRMsg{Number: 1})
	app = updated.(*App)
	assert.Equal(t, core.ViewPRDetail, app.view)

	// 7. Detail loaded
	updated, _ = app.Update(views.PRDetailLoadedMsg{Detail: sampleDetail()})
	app = updated.(*App)
	view = app.View()
	assert.Contains(t, view, "Add authentication")

	// 8. Open diff
	updated, _ = app.Update(views.OpenDiffMsg{Number: 1})
	app = updated.(*App)
	assert.Equal(t, core.ViewDiff, app.view)

	// 9. Navigate back to detail
	updated, _ = app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	app = updated.(*App)
	assert.Equal(t, core.ViewPRDetail, app.view)

	// 10. Navigate back to list
	updated, _ = app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	app = updated.(*App)
	assert.Equal(t, core.ViewPRList, app.view)
}

// -- mock RepoManager for smart checkout tests ----------------------------

type mockRepoManager struct {
	checkoutBranch string
	checkoutErr    error
	cloneErr       error
	worktreeErr    error
}

func (m *mockRepoManager) CheckoutAt(_ context.Context, _ domain.RepoRef, _ int, _ string) (string, error) {
	return m.checkoutBranch, m.checkoutErr
}

func (m *mockRepoManager) CloneRepo(_ context.Context, _ domain.RepoRef, _ string) error {
	return m.cloneErr
}

func (m *mockRepoManager) CreateWorktree(_ context.Context, _ string, _ int, _, _ string) error {
	return m.worktreeErr
}

func smartCheckoutApp(rm domain.RepoManager) *App {
	cfg := config.Default()
	cfg.General.RefreshInterval = 0
	app := New(cfg, WithVersion("test-smart"), WithRepoManager(rm))
	app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	return app
}

// -- smart checkout integration tests -------------------------------------

func TestIntegrationSmartCheckoutNoRepoManagerReposMatch(t *testing.T) {
	// No RepoManager + repos match → legacy confirm dialog (DR-2).
	app := readyApp()
	app.banner.Hide()
	app.view = core.ViewPRDetail
	app.repo = domain.RepoRef{Owner: "test", Name: "repo"}
	app.cwdRepo = app.repo

	updated, _ := app.Update(views.CheckoutPRMsg{Number: 42, Branch: "feat/auth"})
	app = updated.(*App)
	assert.Equal(t, core.ViewConfirm, app.view, "should fallback to legacy confirm")
}

func TestIntegrationSmartCheckoutNoRepoManagerReposMismatch(t *testing.T) {
	// No RepoManager + repos DON'T match → error dialog.
	app := readyApp()
	app.banner.Hide()
	app.view = core.ViewPRDetail
	app.repo = domain.RepoRef{Owner: "other", Name: "repo2"}
	app.cwdRepo = domain.RepoRef{Owner: "test", Name: "repo"}

	updated, _ := app.Update(views.CheckoutPRMsg{Number: 42, Branch: "feat/auth"})
	app = updated.(*App)
	assert.Equal(t, core.ViewSmartCheckout, app.view, "should show error dialog")
	view := app.View()
	assert.Contains(t, view, "Checkout Failed", "should show error title")
}

func TestIntegrationSmartCheckoutStrategyLocal(t *testing.T) {
	// With RepoManager + CWD matches → worktree choice dialog.
	rm := &mockRepoManager{checkoutBranch: "feat/auth"}
	app := smartCheckoutApp(rm)
	app.banner.Hide()
	app.view = core.ViewPRDetail
	app.repo = domain.RepoRef{Owner: "test", Name: "repo"}
	app.cwdRepo = app.repo
	app.cwdPath = "/Users/test/code/repo"

	updated, _ := app.Update(views.CheckoutPRMsg{Number: 42, Branch: "feat/auth"})
	app = updated.(*App)
	assert.Equal(t, core.ViewSmartCheckout, app.view, "should show smart checkout dialog")
	view := app.View()
	assert.Contains(t, view, "Checkout PR #42", "should show PR number in title")
	assert.Contains(t, view, "Switch branch", "should offer switch branch option")
	assert.Contains(t, view, "New worktree", "should offer worktree option")
}

func TestIntegrationSmartCheckoutStrategyNeedsClone(t *testing.T) {
	// With RepoManager + repos DON'T match + no known path → options dialog.
	rm := &mockRepoManager{}
	app := smartCheckoutApp(rm)
	app.banner.Hide()
	app.view = core.ViewPRDetail
	app.repo = domain.RepoRef{Owner: "other", Name: "repo2"}
	app.cwdRepo = domain.RepoRef{Owner: "test", Name: "repo"}
	app.cwdPath = "/Users/test/code/repo"

	updated, _ := app.Update(views.CheckoutPRMsg{Number: 10, Branch: "fix/bug"})
	app = updated.(*App)
	assert.Equal(t, core.ViewSmartCheckout, app.view, "should show smart checkout dialog")
	view := app.View()
	assert.Contains(t, view, "No local clone found", "should show no-clone message")
	assert.Contains(t, view, "Clone to vivecaka cache", "should offer clone option")
	assert.Contains(t, view, "Open on GitHub", "should offer browser option")
}

func TestIntegrationSmartCheckoutStrategyChosenSwitch(t *testing.T) {
	// Strategy "switch" → checking out spinner + cmd.
	rm := &mockRepoManager{checkoutBranch: "feat/auth"}
	app := smartCheckoutApp(rm)
	app.banner.Hide()
	app.view = core.ViewSmartCheckout
	app.cwdPath = "/Users/test/code/repo"
	app.repo = domain.RepoRef{Owner: "test", Name: "repo"}

	updated, cmd := app.Update(views.CheckoutStrategyChosenMsg{
		Strategy: "switch",
		Repo:     domain.RepoRef{Owner: "test", Name: "repo"},
		PRNumber: 42,
		Branch:   "feat/auth",
	})
	app = updated.(*App)
	assert.True(t, app.checkoutDialog.IsLoading(), "should show loading state")
	assert.NotNil(t, cmd, "should return batch cmd for spinner+checkout")
}

func TestIntegrationSmartCheckoutStrategyChosenBrowser(t *testing.T) {
	// Strategy "browser" → returns to previous view.
	rm := &mockRepoManager{}
	app := smartCheckoutApp(rm)
	app.banner.Hide()
	app.view = core.ViewSmartCheckout
	app.prevView = core.ViewPRDetail
	app.repo = domain.RepoRef{Owner: "test", Name: "repo"}

	updated, _ := app.Update(views.CheckoutStrategyChosenMsg{
		Strategy: "browser",
		Repo:     domain.RepoRef{Owner: "test", Name: "repo"},
		PRNumber: 42,
	})
	app = updated.(*App)
	assert.Equal(t, core.ViewPRDetail, app.view, "browser should return to prev view")
}

func TestIntegrationSmartCheckoutCloneDoneSuccess(t *testing.T) {
	// Clone done → transitions to checking out spinner.
	rm := &mockRepoManager{checkoutBranch: "feat/auth"}
	app := smartCheckoutApp(rm)
	app.banner.Hide()
	app.view = core.ViewSmartCheckout
	app.repo = domain.RepoRef{Owner: "test", Name: "repo"}
	app.checkoutDialog.ShowCloning("/cache/clone/path")

	updated, cmd := app.Update(views.CloneDoneMsg{Path: "/cache/clone/path"})
	app = updated.(*App)
	assert.True(t, app.checkoutDialog.IsLoading(), "should transition to checking out")
	assert.NotNil(t, cmd, "should return checkout cmd")
}

func TestIntegrationSmartCheckoutCloneDoneError(t *testing.T) {
	// Clone done with error → shows error dialog.
	rm := &mockRepoManager{}
	app := smartCheckoutApp(rm)
	app.banner.Hide()
	app.view = core.ViewSmartCheckout
	app.checkoutDialog.ShowCloning("/cache/clone/path")

	updated, _ := app.Update(views.CloneDoneMsg{Err: fmt.Errorf("clone failed: permission denied")})
	app = updated.(*App)
	view := app.View()
	assert.Contains(t, view, "Checkout Failed", "should show error")
	assert.Contains(t, view, "clone failed", "should show error message")
}

func TestIntegrationSmartCheckoutDoneSuccess(t *testing.T) {
	// Checkout done → success dialog (CWD checkout).
	rm := &mockRepoManager{checkoutBranch: "feat/auth"}
	app := smartCheckoutApp(rm)
	app.banner.Hide()
	app.view = core.ViewSmartCheckout
	app.cwdPath = "/Users/test/code/repo"
	app.checkoutDialog.ShowCheckingOut("/Users/test/code/repo")

	updated, _ := app.Update(views.SmartCheckoutDoneMsg{
		Branch: "feat/auth",
		Path:   "/Users/test/code/repo",
	})
	app = updated.(*App)
	view := app.View()
	assert.Contains(t, view, "Checkout Complete", "should show success")
	assert.Contains(t, view, "feat/auth", "should show branch name")
	assert.NotContains(t, view, "copy cd command", "CWD checkout should not show copy hint")
}

func TestIntegrationSmartCheckoutDoneSuccessRemote(t *testing.T) {
	// Checkout done → success dialog (remote path → shows copy hint).
	rm := &mockRepoManager{checkoutBranch: "feat/auth"}
	app := smartCheckoutApp(rm)
	app.banner.Hide()
	app.view = core.ViewSmartCheckout
	app.cwdPath = "/Users/test/code/repo"
	app.checkoutDialog.ShowCheckingOut("/cache/clone/path")

	updated, _ := app.Update(views.SmartCheckoutDoneMsg{
		Branch: "feat/auth",
		Path:   "/cache/clone/path",
	})
	app = updated.(*App)
	view := app.View()
	assert.Contains(t, view, "Checkout Complete", "should show success")
	assert.Contains(t, view, "copy cd command", "remote checkout should show copy hint")
	assert.Contains(t, view, "cd /cache/clone/path", "should show cd command")
}

func TestIntegrationSmartCheckoutDoneError(t *testing.T) {
	// Checkout done with error → error dialog.
	rm := &mockRepoManager{}
	app := smartCheckoutApp(rm)
	app.banner.Hide()
	app.view = core.ViewSmartCheckout
	app.checkoutDialog.ShowCheckingOut("/some/path")

	updated, _ := app.Update(views.SmartCheckoutDoneMsg{Err: fmt.Errorf("git: branch conflict")})
	app = updated.(*App)
	view := app.View()
	assert.Contains(t, view, "Checkout Failed", "should show error title")
	assert.Contains(t, view, "git: branch conflict", "should show error message")
}

func TestIntegrationSmartCheckoutDialogClose(t *testing.T) {
	// CheckoutDialogCloseMsg → returns to previous view.
	rm := &mockRepoManager{}
	app := smartCheckoutApp(rm)
	app.banner.Hide()
	app.view = core.ViewSmartCheckout
	app.prevView = core.ViewPRDetail

	updated, _ := app.Update(views.CheckoutDialogCloseMsg{})
	app = updated.(*App)
	assert.Equal(t, core.ViewPRDetail, app.view, "should return to previous view")
}

func TestIntegrationSmartCheckoutCopyCdCommand(t *testing.T) {
	// CopyCdCommandMsg → handled (clipboard may fail in CI, just verify message routing).
	rm := &mockRepoManager{}
	app := smartCheckoutApp(rm)
	app.banner.Hide()
	app.view = core.ViewSmartCheckout
	app.prevView = core.ViewPRDetail

	updated, cmd := app.Update(views.CopyCdCommandMsg{Path: "/cache/path"})
	app = updated.(*App)
	// Should return to previous view regardless of clipboard success.
	assert.Equal(t, core.ViewPRDetail, app.view, "should return to previous view after copy")
	assert.NotNil(t, cmd, "should return toast cmd")
}

func TestIntegrationSmartCheckoutFallbackSafety(t *testing.T) {
	// DR-11: Verify fallback safety — no RepoManager, repos match, still works.
	app := readyApp()
	app.banner.Hide()
	app.view = core.ViewPRDetail
	app.repo = domain.RepoRef{Owner: "test", Name: "repo"}
	app.cwdRepo = domain.RepoRef{Owner: "TEST", Name: "REPO"} // case-insensitive match

	updated, _ := app.Update(views.CheckoutPRMsg{Number: 1, Branch: "main"})
	app = updated.(*App)
	assert.Equal(t, core.ViewConfirm, app.view, "case-insensitive match should use legacy confirm")
}

func TestIntegrationRepoDetectedAutoLearns(t *testing.T) {
	// Verify auto-learn: RepoDetectedMsg registers in locator.
	rm := &mockRepoManager{}
	app := smartCheckoutApp(rm)
	app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	repo := domain.RepoRef{Owner: "steipete", Name: "CodexBar"}
	updated, _ := app.Update(views.RepoDetectedMsg{Repo: repo})
	app = updated.(*App)

	assert.Equal(t, repo.Owner, app.cwdRepo.Owner, "should set cwdRepo")
	assert.Equal(t, repo.Name, app.cwdRepo.Name, "should set cwdRepo name")
}
