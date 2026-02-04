package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/indrasvat/vivecaka/internal/config"
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
	if app.view != ViewLoading {
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
	app.view = ViewPRList

	// Press ? to open help.
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	updated, _ := app.Update(msg)
	a := updated.(*App)

	if a.view != ViewHelp {
		t.Errorf("view = %d, want ViewHelp", a.view)
	}
	if a.prevView != ViewPRList {
		t.Errorf("prevView = %d, want ViewPRList", a.prevView)
	}

	// Press ? again to close help.
	updated, _ = a.Update(msg)
	a = updated.(*App)

	if a.view != ViewPRList {
		t.Errorf("view = %d, want ViewPRList after closing help", a.view)
	}
}

func TestAppBackNavigation(t *testing.T) {
	app := newTestApp()
	app.ready = true

	tests := []struct {
		from ViewState
		to   ViewState
	}{
		{ViewHelp, ViewPRList},
		{ViewPRDetail, ViewPRList},
		{ViewDiff, ViewPRDetail},
		{ViewReview, ViewPRDetail},
		{ViewRepoSwitch, ViewPRList},
		{ViewInbox, ViewPRList},
	}

	for _, tt := range tests {
		app.view = tt.from
		app.prevView = ViewPRList // For help/repo-switch navigation.

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

	if a.view != ViewPRList {
		t.Errorf("view after viewReadyMsg = %d, want ViewPRList", a.view)
	}
}

func TestAppRepoSwitch(t *testing.T) {
	app := newTestApp()
	app.ready = true
	app.view = ViewPRList

	msg := tea.KeyMsg{Type: tea.KeyCtrlR}
	updated, _ := app.Update(msg)
	a := updated.(*App)

	if a.view != ViewRepoSwitch {
		t.Errorf("view = %d, want ViewRepoSwitch", a.view)
	}
	if a.prevView != ViewPRList {
		t.Errorf("prevView = %d, want ViewPRList", a.prevView)
	}

	// Pressing Ctrl+R again shouldn't change anything (already in switcher).
	updated, _ = a.Update(msg)
	a = updated.(*App)
	if a.view != ViewRepoSwitch {
		t.Errorf("view should stay ViewRepoSwitch, got %d", a.view)
	}
}

func TestAppAllViewsRender(t *testing.T) {
	views := []ViewState{
		ViewLoading,
		ViewPRList,
		ViewPRDetail,
		ViewDiff,
		ViewReview,
		ViewHelp,
		ViewRepoSwitch,
		ViewInbox,
	}

	for _, v := range views {
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
		view ViewState
		name string
	}{
		{ViewLoading, "Loading"},
		{ViewPRList, "PR List"},
		{ViewPRDetail, "PR Detail"},
		{ViewDiff, "Diff"},
		{ViewReview, "Review"},
		{ViewHelp, "Help"},
		{ViewRepoSwitch, "Repo Switch"},
		{ViewInbox, "Inbox"},
	}

	app := newTestApp()
	for _, tt := range tests {
		app.view = tt.view
		if got := app.viewName(); got != tt.name {
			t.Errorf("viewName(%d) = %q, want %q", tt.view, got, tt.name)
		}
	}
}

func TestAppStatusHints(t *testing.T) {
	views := []ViewState{
		ViewPRList,
		ViewPRDetail,
		ViewDiff,
		ViewReview,
		ViewInbox,
		ViewRepoSwitch,
		ViewHelp,
		ViewLoading,
	}

	app := newTestApp()
	app.width = 120
	for _, v := range views {
		app.view = v
		hints := app.statusHints()
		if hints == "" {
			t.Errorf("statusHints(%d) should not be empty", v)
		}
	}
}

func TestTruncateHints(t *testing.T) {
	long := "j/k navigate  Enter open  c checkout  / search  ? help  q quit"

	// No truncation when wide enough.
	got := truncateHints(long, 200)
	if got != long {
		t.Errorf("should not truncate when wide enough")
	}

	// Truncation for narrow.
	got = truncateHints(long, 20)
	if len(got) > 20 {
		t.Errorf("truncated hints len=%d should be <= 20", len(got))
	}

	// Zero width doesn't truncate.
	got = truncateHints(long, 0)
	if got != long {
		t.Errorf("zero width should not truncate")
	}
}

func TestAppSmallTerminal(t *testing.T) {
	app := newTestApp()
	app.ready = true
	app.width = 20
	app.height = 5
	app.view = ViewPRList

	// Should not panic.
	view := app.View()
	if view == "" {
		t.Error("small terminal view should not be empty")
	}
}
