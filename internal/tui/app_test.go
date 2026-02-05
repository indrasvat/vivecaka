package tui

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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

	if app.version != "test" {
		t.Errorf("version = %q, want %q", app.version, "test")
	}
	if app.view != core.ViewLoading {
		t.Errorf("initial view = %d, want ViewLoading", app.view)
	}
}

func TestAppInitReturnsCmd(t *testing.T) {
	app := newTestApp()
	cmd := app.Init()
	if cmd == nil {
		t.Error("Init() should return a cmd")
	}
}

func TestAppWindowSizeMsg(t *testing.T) {
	app := newTestApp()

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updated, _ := app.Update(msg)
	a := updated.(*App)

	if a.width != 120 {
		t.Errorf("width = %d, want 120", a.width)
	}
	if a.height != 40 {
		t.Errorf("height = %d, want 40", a.height)
	}
	if !a.ready {
		t.Error("ready should be true after WindowSizeMsg")
	}
}

func TestAppQuitKey(t *testing.T) {
	app := newTestApp()
	app.ready = true

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := app.Update(msg)

	if cmd == nil {
		t.Fatal("quit key should return a cmd")
	}
}

func TestAppHelpToggle(t *testing.T) {
	app := newTestApp()
	app.ready = true
	app.view = core.ViewPRList

	// Press ? to open help.
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	updated, _ := app.Update(msg)
	a := updated.(*App)

	if a.view != core.ViewHelp {
		t.Errorf("view = %d, want ViewHelp", a.view)
	}
	if a.prevView != core.ViewPRList {
		t.Errorf("prevView = %d, want ViewPRList", a.prevView)
	}

	// Press ? again to close help.
	updated, _ = a.Update(msg)
	a = updated.(*App)

	if a.view != core.ViewPRList {
		t.Errorf("view = %d, want ViewPRList after closing help", a.view)
	}
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
	}

	for _, tt := range tests {
		app.view = tt.from
		app.prevView = core.ViewPRList // For help/repo-switch navigation.

		msg := tea.KeyMsg{Type: tea.KeyEscape}
		updated, _ := app.Update(msg)
		a := updated.(*App)

		if a.view != tt.to {
			t.Errorf("back from %d: view = %d, want %d", tt.from, a.view, tt.to)
		}
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
	if view == "" {
		t.Error("View() should not be empty when ready")
	}
}

func TestAppViewNotReady(t *testing.T) {
	app := newTestApp()
	view := app.View()
	if view != "" {
		t.Errorf("View() should be empty when not ready, got %q", view)
	}
}

func TestAppViewReadyMsg(t *testing.T) {
	app := newTestApp()
	app.ready = true

	updated, _ := app.Update(viewReadyMsg{})
	a := updated.(*App)

	if a.view != core.ViewPRList {
		t.Errorf("view after viewReadyMsg = %d, want ViewPRList", a.view)
	}
}

func TestAppRepoSwitch(t *testing.T) {
	app := newTestApp()
	app.ready = true
	app.view = core.ViewPRList

	msg := tea.KeyMsg{Type: tea.KeyCtrlR}
	updated, _ := app.Update(msg)
	a := updated.(*App)

	if a.view != core.ViewRepoSwitch {
		t.Errorf("view = %d, want ViewRepoSwitch", a.view)
	}
	if a.prevView != core.ViewPRList {
		t.Errorf("prevView = %d, want ViewPRList", a.prevView)
	}

	// Pressing Ctrl+R again shouldn't change anything (already in switcher).
	updated, _ = a.Update(msg)
	a = updated.(*App)
	if a.view != core.ViewRepoSwitch {
		t.Errorf("view should stay ViewRepoSwitch, got %d", a.view)
	}
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
	}

	for _, v := range allViews {
		app := newTestApp()
		app.ready = true
		app.width = 120
		app.height = 40
		app.view = v

		view := app.View()
		if view == "" {
			t.Errorf("View() for state %d should not be empty", v)
		}
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
	}

	app := newTestApp()
	for _, tt := range tests {
		app.view = tt.view
		if got := app.viewName(); got != tt.name {
			t.Errorf("viewName(%d) = %q, want %q", tt.view, got, tt.name)
		}
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
	if view == "" {
		t.Error("small terminal view should not be empty")
	}
}

func TestAppRepoDetected(t *testing.T) {
	app := newTestApp()
	app.ready = true

	msg := views.RepoDetectedMsg{
		Repo: domain.RepoRef{Owner: "test", Name: "repo"},
	}
	updated, _ := app.Update(msg)
	a := updated.(*App)

	if a.repo.Owner != "test" || a.repo.Name != "repo" {
		t.Errorf("repo = %v, want test/repo", a.repo)
	}
}

func TestAppRepoDetectedError(t *testing.T) {
	app := newTestApp()
	app.ready = true

	msg := views.RepoDetectedMsg{
		Err: fmt.Errorf("no git remote"),
	}
	updated, cmd := app.Update(msg)
	a := updated.(*App)

	if a.view != core.ViewPRList {
		t.Errorf("view after error = %d, want ViewPRList", a.view)
	}
	if cmd == nil {
		t.Error("should return toast cmd on error")
	}
}

func TestAppPRsLoaded(t *testing.T) {
	app := newTestApp()
	app.ready = true

	msg := views.PRsLoadedMsg{
		PRs: []domain.PR{{Number: 1, Title: "test"}},
	}
	updated, _ := app.Update(msg)
	a := updated.(*App)

	if a.view != core.ViewPRList {
		t.Errorf("view = %d, want ViewPRList", a.view)
	}
}

func TestAppOpenPR(t *testing.T) {
	app := newTestApp()
	app.ready = true
	app.view = core.ViewPRList

	msg := views.OpenPRMsg{Number: 42}
	updated, _ := app.Update(msg)
	a := updated.(*App)

	if a.view != core.ViewPRDetail {
		t.Errorf("view = %d, want ViewPRDetail", a.view)
	}
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

	if a.theme.Name == origTheme {
		t.Error("theme should have changed after T key")
	}
}
