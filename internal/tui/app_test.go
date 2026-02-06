package tui

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/indrasvat/vivecaka/internal/config"
	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/indrasvat/vivecaka/internal/tui/core"
	"github.com/indrasvat/vivecaka/internal/tui/views"
)

func newTestApp() *App {
	cfg := config.Default()
	return New(cfg, WithVersion("test"))
}

func TestNewAppDefaults(t *testing.T) {
	app := newTestApp()

	assert.Equal(t, "test", app.version)
	assert.Equal(t, core.ViewBanner, app.view)
}

func TestAppInitReturnsCmd(t *testing.T) {
	app := newTestApp()
	cmd := app.Init()
	assert.NotNil(t, cmd, "Init() should return a cmd")
}

func TestAppWindowSizeMsg(t *testing.T) {
	app := newTestApp()

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updated, _ := app.Update(msg)
	a := updated.(*App)

	assert.Equal(t, 120, a.width)
	assert.Equal(t, 40, a.height)
	assert.True(t, a.ready, "ready should be true after WindowSizeMsg")
}

func TestAppQuitKey(t *testing.T) {
	app := newTestApp()
	app.ready = true
	app.view = core.ViewPRList // Ensure we're past the banner
	app.banner.Hide()          // Ensure banner is not visible

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := app.Update(msg)

	assert.NotNil(t, cmd, "quit key should return a cmd")
}

func TestAppQuitDuringBanner(t *testing.T) {
	app := newTestApp()
	app.ready = true
	// Banner is visible by default (view=ViewBanner, banner.Visible()=true)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := app.Update(msg)

	assert.NotNil(t, cmd, "quit key during banner should return tea.Quit cmd")
	// The banner should still be visible (we quit, not dismiss)
	assert.True(t, app.banner.Visible(), "banner should still be visible (quit, not dismiss)")
}

func TestAppDismissBannerKey(t *testing.T) {
	app := newTestApp()
	app.ready = true
	// Banner is visible by default

	// Press a non-quit key to dismiss
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	updated, cmd := app.Update(msg)
	a := updated.(*App)

	assert.False(t, a.banner.Visible(), "banner should be dismissed after keypress")
	assert.Equal(t, core.ViewLoading, a.view, "view after banner dismiss")
	assert.NotNil(t, cmd, "should return tea.ClearScreen + loading tick cmd")
}

func TestAppLoadingTickAnimation(t *testing.T) {
	app := newTestApp()
	app.ready = true
	app.view = core.ViewLoading
	app.width = 80
	app.height = 24

	// Verify loading tick increments frame
	initialFrame := app.loadingFrame
	updated, cmd := app.Update(loadingTickMsg{})
	a := updated.(*App)

	assert.Equal(t, initialFrame+1, a.loadingFrame)
	assert.NotNil(t, cmd, "loading tick should return another tick cmd")
}

func TestAppLoadingTickStopsOnViewChange(t *testing.T) {
	app := newTestApp()
	app.ready = true
	app.view = core.ViewPRList // Not on loading screen

	_, cmd := app.Update(loadingTickMsg{})

	assert.Nil(t, cmd, "loading tick should not continue when not on loading view")
}

func TestAppHelpToggle(t *testing.T) {
	app := newTestApp()
	app.ready = true
	app.view = core.ViewPRList

	// Press ? to open help.
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	updated, _ := app.Update(msg)
	a := updated.(*App)

	assert.Equal(t, core.ViewHelp, a.view)
	assert.Equal(t, core.ViewPRList, a.prevView)

	// Press ? again to close help.
	updated, _ = a.Update(msg)
	a = updated.(*App)

	assert.Equal(t, core.ViewPRList, a.view, "view after closing help")
}

func TestAppBackNavigation(t *testing.T) {
	app := newTestApp()
	app.ready = true

	tests := []struct {
		from core.ViewState
		to   core.ViewState
	}{
		{core.ViewHelp, core.ViewPRList},
		{core.ViewPRDetail, core.ViewPRList},
		{core.ViewDiff, core.ViewPRDetail},
		{core.ViewReview, core.ViewPRDetail},
		{core.ViewRepoSwitch, core.ViewPRList},
		{core.ViewInbox, core.ViewPRList},
		{core.ViewFilter, core.ViewPRList},
	}

	for _, tt := range tests {
		app.view = tt.from
		app.prevView = core.ViewPRList // For help/repo-switch navigation.

		msg := tea.KeyMsg{Type: tea.KeyEscape}
		updated, _ := app.Update(msg)
		a := updated.(*App)

		assert.Equal(t, tt.to, a.view, "back from %d", tt.from)
	}
}

func TestAppViewReady(t *testing.T) {
	app := newTestApp()
	app.ready = true
	app.width = 80
	app.height = 24
	app.view = core.ViewPRList

	// Should render without panic.
	view := app.View()
	assert.NotEmpty(t, view, "View() should not be empty when ready")
}

func TestAppViewNotReady(t *testing.T) {
	app := newTestApp()
	view := app.View()
	assert.Empty(t, view, "View() should be empty when not ready")
}

func TestAppViewReadyMsg(t *testing.T) {
	app := newTestApp()
	app.ready = true

	updated, _ := app.Update(viewReadyMsg{})
	a := updated.(*App)

	assert.Equal(t, core.ViewPRList, a.view, "view after viewReadyMsg")
}

func TestAppRepoSwitch(t *testing.T) {
	app := newTestApp()
	app.ready = true
	app.view = core.ViewPRList

	msg := tea.KeyMsg{Type: tea.KeyCtrlR}
	updated, _ := app.Update(msg)
	a := updated.(*App)

	assert.Equal(t, core.ViewRepoSwitch, a.view)
	assert.Equal(t, core.ViewPRList, a.prevView)

	// Pressing Ctrl+R again shouldn't change anything (already in switcher).
	updated, _ = a.Update(msg)
	a = updated.(*App)
	assert.Equal(t, core.ViewRepoSwitch, a.view, "view should stay ViewRepoSwitch")
}

func TestAppAllViewsRender(t *testing.T) {
	allViews := []core.ViewState{
		core.ViewLoading,
		core.ViewPRList,
		core.ViewPRDetail,
		core.ViewDiff,
		core.ViewReview,
		core.ViewHelp,
		core.ViewRepoSwitch,
		core.ViewInbox,
		core.ViewFilter,
	}

	for _, v := range allViews {
		app := newTestApp()
		app.ready = true
		app.width = 120
		app.height = 40
		app.view = v

		view := app.View()
		assert.NotEmpty(t, view, "View() for state %d should not be empty", v)
	}
}

func TestAppViewNames(t *testing.T) {
	tests := []struct {
		view core.ViewState
		name string
	}{
		{core.ViewLoading, "Loading"},
		{core.ViewPRList, "PR List"},
		{core.ViewPRDetail, "PR Detail"},
		{core.ViewDiff, "Diff"},
		{core.ViewReview, "Review"},
		{core.ViewHelp, "Help"},
		{core.ViewRepoSwitch, "Repo Switch"},
		{core.ViewInbox, "Inbox"},
		{core.ViewFilter, "Filter"},
	}

	app := newTestApp()
	for _, tt := range tests {
		app.view = tt.view
		got := app.viewName()
		assert.Equal(t, tt.name, got)
	}
}

func TestAppSmallTerminal(t *testing.T) {
	app := newTestApp()
	app.ready = true
	app.width = 20
	app.height = 5
	app.view = core.ViewPRList

	// Should not panic.
	view := app.View()
	assert.NotEmpty(t, view, "small terminal view should not be empty")
}

func TestAppRepoDetected(t *testing.T) {
	app := newTestApp()
	app.ready = true

	msg := views.RepoDetectedMsg{
		Repo: domain.RepoRef{Owner: "test", Name: "repo"},
	}
	updated, _ := app.Update(msg)
	a := updated.(*App)

	assert.Equal(t, "test", a.repo.Owner)
	assert.Equal(t, "repo", a.repo.Name)
}

func TestAppRepoDetectedError(t *testing.T) {
	app := newTestApp()
	app.ready = true

	msg := views.RepoDetectedMsg{
		Err: fmt.Errorf("no git remote"),
	}
	updated, cmd := app.Update(msg)
	a := updated.(*App)

	assert.Equal(t, core.ViewPRList, a.view, "view after error")
	assert.NotNil(t, cmd, "should return toast cmd on error")
}

func TestAppPRsLoaded(t *testing.T) {
	app := newTestApp()
	app.ready = true
	app.view = core.ViewLoading // PRs load after banner dismisses
	app.banner.Hide()

	msg := views.PRsLoadedMsg{
		PRs: []domain.PR{{Number: 1, Title: "test"}},
	}
	updated, _ := app.Update(msg)
	a := updated.(*App)

	assert.Equal(t, core.ViewPRList, a.view)
}

func TestAppPRsLoadedWhileBannerVisible(t *testing.T) {
	app := newTestApp()
	app.ready = true
	// Keep view as ViewBanner (default)

	msg := views.PRsLoadedMsg{
		PRs: []domain.PR{{Number: 1, Title: "test"}},
	}
	updated, _ := app.Update(msg)
	a := updated.(*App)

	// View should stay as banner while banner is visible
	assert.Equal(t, core.ViewBanner, a.view, "PRs loaded while banner visible")
}

func TestAppOpenPR(t *testing.T) {
	app := newTestApp()
	app.ready = true
	app.view = core.ViewPRList

	msg := views.OpenPRMsg{Number: 42}
	updated, _ := app.Update(msg)
	a := updated.(*App)

	assert.Equal(t, core.ViewPRDetail, a.view)
}

func TestAppThemeCycle(t *testing.T) {
	app := newTestApp()
	app.ready = true
	app.width = 120
	app.height = 40
	app.view = core.ViewPRList

	origTheme := app.theme.Name

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T'}}
	updated, _ := app.Update(msg)
	a := updated.(*App)

	assert.NotEqual(t, origTheme, a.theme.Name, "theme should have changed after T key")
}
