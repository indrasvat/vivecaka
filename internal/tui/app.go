package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/indrasvat/vivecaka/internal/config"
)

// ViewState represents which view is currently active.
type ViewState int

const (
	ViewLoading ViewState = iota
	ViewPRList
	ViewPRDetail
	ViewDiff
	ViewReview
	ViewHelp
	ViewRepoSwitch
	ViewInbox
)

// Option is a functional option for configuring the app.
type Option func(*App)

// WithVersion sets the version string.
func WithVersion(v string) Option {
	return func(a *App) { a.version = v }
}

// App is the root BubbleTea model.
type App struct {
	cfg      *config.Config
	version  string
	view     ViewState
	prevView ViewState
	width    int
	height   int
	ready    bool
	keys     KeyMap
	styles   Styles
	theme    Theme
}

// New creates a new App model.
func New(cfg *config.Config, opts ...Option) *App {
	theme := ThemeByName(cfg.General.Theme)
	a := &App{
		cfg:    cfg,
		view:   ViewLoading,
		keys:   DefaultKeyMap(),
		theme:  theme,
		styles: NewStyles(theme),
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

func (a *App) Init() tea.Cmd {
	// Transition to PR list view immediately (data loading will happen in Phase 7).
	return func() tea.Msg { return viewReadyMsg{} }
}

type viewReadyMsg struct{}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return a.handleKey(msg)

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.ready = true
		return a, nil

	case viewReadyMsg:
		a.view = ViewPRList
		return a, nil
	}

	return a, nil
}

func (a *App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global keys always active.
	switch {
	case key.Matches(msg, a.keys.Quit):
		return a, tea.Quit

	case key.Matches(msg, a.keys.Help):
		if a.view == ViewHelp {
			a.view = a.prevView
		} else {
			a.prevView = a.view
			a.view = ViewHelp
		}
		return a, nil

	case key.Matches(msg, a.keys.ThemeCycle):
		a.theme = NextTheme(a.theme.Name)
		a.styles = NewStyles(a.theme)
		return a, nil

	case key.Matches(msg, a.keys.RepoSwitch):
		if a.view != ViewRepoSwitch {
			a.prevView = a.view
			a.view = ViewRepoSwitch
		}
		return a, nil

	case key.Matches(msg, a.keys.Back):
		switch a.view {
		case ViewHelp:
			a.view = a.prevView
		case ViewRepoSwitch:
			a.view = a.prevView
		case ViewPRDetail:
			a.view = ViewPRList
		case ViewDiff:
			a.view = ViewPRDetail
		case ViewReview:
			a.view = ViewPRDetail
		case ViewInbox:
			a.view = ViewPRList
		}
		return a, nil
	}

	return a, nil
}

func (a *App) View() string {
	if !a.ready {
		return ""
	}

	// Header.
	header := a.renderHeader()

	// Content.
	contentHeight := max(1, a.height-2) // header + status bar
	content := a.renderContent(contentHeight)

	// Status bar.
	statusBar := a.renderStatusBar()

	return lipgloss.JoinVertical(lipgloss.Left, header, content, statusBar)
}

func (a *App) renderHeader() string {
	t := a.theme
	repoStyle := lipgloss.NewStyle().Foreground(t.Primary).Bold(true)
	viewStyle := lipgloss.NewStyle().Foreground(t.Subtext)

	repo := repoStyle.Render("◉ vivecaka")
	view := viewStyle.Render(a.viewName())

	gap := max(1, a.width-lipgloss.Width(repo)-lipgloss.Width(view)-2)

	return a.styles.Header.Width(a.width).Render(
		repo + lipgloss.NewStyle().Width(gap).Render("") + view,
	)
}

func (a *App) renderContent(height int) string {
	style := lipgloss.NewStyle().
		Width(a.width).
		Height(height)

	switch a.view {
	case ViewLoading:
		return style.Align(lipgloss.Center, lipgloss.Center).
			Foreground(a.theme.Muted).
			Render("Loading...")

	case ViewPRList:
		return style.Align(lipgloss.Center, lipgloss.Center).
			Foreground(a.theme.Subtext).
			Render("PR List (Phase 7)")

	case ViewPRDetail:
		return style.Align(lipgloss.Center, lipgloss.Center).
			Foreground(a.theme.Subtext).
			Render("PR Detail (Phase 8)")

	case ViewDiff:
		return style.Align(lipgloss.Center, lipgloss.Center).
			Foreground(a.theme.Subtext).
			Render("Diff Viewer (Phase 9)")

	case ViewReview:
		return style.Align(lipgloss.Center, lipgloss.Center).
			Foreground(a.theme.Subtext).
			Render("Review Form (Phase 10)")

	case ViewHelp:
		return style.Align(lipgloss.Center, lipgloss.Center).
			Foreground(a.theme.Info).
			Render("Help — press ? or Esc to close")

	case ViewRepoSwitch:
		return style.Align(lipgloss.Center, lipgloss.Center).
			Foreground(a.theme.Info).
			Render("Repo Switcher — Ctrl+R")

	case ViewInbox:
		return style.Align(lipgloss.Center, lipgloss.Center).
			Foreground(a.theme.Subtext).
			Render("Unified PR Inbox")

	default:
		return style.Render("")
	}
}

func (a *App) renderStatusBar() string {
	t := a.theme
	hintStyle := lipgloss.NewStyle().Foreground(t.Muted)

	hints := hintStyle.Render(a.statusHints())

	return a.styles.StatusBar.Width(a.width).Render(hints)
}

func (a *App) statusHints() string {
	switch a.view {
	case ViewPRList:
		return truncateHints("j/k navigate  Enter open  c checkout  / search  ? help  q quit", a.width)
	case ViewPRDetail:
		return truncateHints("j/k scroll  Tab pane  r review  Enter diff  Esc back  ? help", a.width)
	case ViewDiff:
		return truncateHints("j/k scroll  Tab file  / search  Esc back  ? help", a.width)
	case ViewReview:
		return truncateHints("j/k field  Enter action  Esc back  ? help", a.width)
	case ViewInbox:
		return truncateHints("j/k navigate  Tab tab  Enter open  Esc back  ? help", a.width)
	case ViewRepoSwitch:
		return truncateHints("j/k navigate  Enter switch  Esc cancel", a.width)
	case ViewHelp:
		return "? or Esc to close"
	default:
		return "? help  q quit"
	}
}

func truncateHints(hints string, width int) string {
	if width > 0 && len(hints) > width-2 {
		return hints[:width-5] + "..."
	}
	return hints
}

func (a *App) viewName() string {
	switch a.view {
	case ViewLoading:
		return "Loading"
	case ViewPRList:
		return "PR List"
	case ViewPRDetail:
		return "PR Detail"
	case ViewDiff:
		return "Diff"
	case ViewReview:
		return "Review"
	case ViewHelp:
		return "Help"
	case ViewRepoSwitch:
		return "Repo Switch"
	case ViewInbox:
		return "Inbox"
	default:
		return ""
	}
}
