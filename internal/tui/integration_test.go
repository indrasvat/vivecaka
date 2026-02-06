package tui

import (
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

	// Repo switcher → back to PR list.
	updated, _ = app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	app = updated.(*App)
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

	// Checkout request shows confirm dialog.
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
	allViews := []core.ViewState{
		core.ViewPRList, core.ViewPRDetail, core.ViewDiff,
		core.ViewReview, core.ViewHelp, core.ViewRepoSwitch,
		core.ViewInbox, core.ViewFilter,
	}
	for _, v := range allViews {
		app := readyApp()
		app.banner.Hide()
		app.view = v

		_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
		require.NotNil(t, cmd, "quit from view %d should return cmd", v)
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
