package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/indrasvat/vivecaka/internal/tui/core"
)

// PRDetailModel implements the PR detail view.
type PRDetailModel struct {
	detail  *domain.PRDetail
	width   int
	height  int
	styles  core.Styles
	keys    core.KeyMap
	pane    DetailPane
	scrollY int
	loading bool
}

// DetailPane represents the active pane in detail view.
type DetailPane int

const (
	PaneInfo DetailPane = iota
	PaneFiles
	PaneChecks
	PaneComments
)

// NewPRDetailModel creates a new PR detail view.
func NewPRDetailModel(styles core.Styles, keys core.KeyMap) PRDetailModel {
	return PRDetailModel{
		styles:  styles,
		keys:    keys,
		loading: true,
	}
}

// SetSize updates the view dimensions.
func (m *PRDetailModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetDetail updates the displayed PR detail.
func (m *PRDetailModel) SetDetail(d *domain.PRDetail) {
	m.detail = d
	m.loading = false
	m.scrollY = 0
}

// Message types.
type (
	PRDetailLoadedMsg struct {
		Detail *domain.PRDetail
		Err    error
	}
	OpenDiffMsg    struct{ Number int }
	StartReviewMsg struct{ Number int }
)

// Update handles messages for the detail view.
func (m *PRDetailModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case PRDetailLoadedMsg:
		m.SetDetail(msg.Detail)
	}
	return nil
}

func (m *PRDetailModel) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.Tab):
		m.pane = (m.pane + 1) % 4
		m.scrollY = 0
	case key.Matches(msg, m.keys.ShiftTab):
		m.pane = (m.pane + 3) % 4
		m.scrollY = 0
	case key.Matches(msg, m.keys.Down):
		m.scrollY++
	case key.Matches(msg, m.keys.Up):
		if m.scrollY > 0 {
			m.scrollY--
		}
	case key.Matches(msg, m.keys.Enter):
		if m.detail != nil {
			num := m.detail.Number
			if m.pane == PaneFiles {
				return func() tea.Msg { return OpenDiffMsg{Number: num} }
			}
		}
	}
	// 'r' key for review.
	if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 && msg.Runes[0] == 'r' {
		if m.detail != nil {
			return func() tea.Msg { return StartReviewMsg{Number: m.detail.Number} }
		}
	}
	return nil
}

// View renders the PR detail view.
func (m *PRDetailModel) View() string {
	if m.loading || m.detail == nil {
		return lipgloss.NewStyle().
			Width(m.width).Height(m.height).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(m.styles.Theme.Muted).
			Render("Loading PR detail...")
	}

	t := m.styles.Theme
	d := m.detail

	// Title bar.
	titleStyle := lipgloss.NewStyle().Foreground(t.Primary).Bold(true)
	title := titleStyle.Render(fmt.Sprintf("#%d %s", d.Number, d.Title))

	// Tab bar.
	tabs := m.renderTabs()

	// Content based on active pane.
	contentHeight := max(1, m.height-4) // title + tabs + padding
	content := m.renderPane(contentHeight)

	return lipgloss.JoinVertical(lipgloss.Left, title, tabs, content)
}

func (m *PRDetailModel) renderTabs() string {
	t := m.styles.Theme
	active := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Padding(0, 1)
	inactive := lipgloss.NewStyle().Foreground(t.Muted).Padding(0, 1)

	tabs := []string{"Info", "Files", "Checks", "Comments"}
	var rendered []string
	for i, tab := range tabs {
		if DetailPane(i) == m.pane {
			rendered = append(rendered, active.Render(tab))
		} else {
			rendered = append(rendered, inactive.Render(tab))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
}

func (m *PRDetailModel) renderPane(height int) string {
	switch m.pane {
	case PaneInfo:
		return m.renderInfoPane(height)
	case PaneFiles:
		return m.renderFilesPane(height)
	case PaneChecks:
		return m.renderChecksPane(height)
	case PaneComments:
		return m.renderCommentsPane(height)
	default:
		return ""
	}
}

func (m *PRDetailModel) renderInfoPane(height int) string {
	t := m.styles.Theme
	d := m.detail

	labelStyle := lipgloss.NewStyle().Foreground(t.Muted)
	valueStyle := lipgloss.NewStyle().Foreground(t.Fg)

	var lines []string
	lines = append(lines, labelStyle.Render("Author: ")+valueStyle.Render(d.Author))
	lines = append(lines, labelStyle.Render("Branch: ")+valueStyle.Render(d.Branch.Head+" → "+d.Branch.Base))
	lines = append(lines, labelStyle.Render("State:  ")+valueStyle.Render(string(d.State)))

	if len(d.Labels) > 0 {
		lines = append(lines, labelStyle.Render("Labels: ")+valueStyle.Render(strings.Join(d.Labels, ", ")))
	}
	if len(d.Assignees) > 0 {
		lines = append(lines, labelStyle.Render("Assign: ")+valueStyle.Render(strings.Join(d.Assignees, ", ")))
	}
	if len(d.Reviewers) > 0 {
		var revs []string
		for _, r := range d.Reviewers {
			revs = append(revs, fmt.Sprintf("%s (%s)", r.Login, r.State))
		}
		lines = append(lines, labelStyle.Render("Review: ")+valueStyle.Render(strings.Join(revs, ", ")))
	}

	lines = append(lines, "")
	if d.Body != "" {
		lines = append(lines, d.Body)
	} else {
		lines = append(lines, lipgloss.NewStyle().Foreground(t.Muted).Render("No description provided."))
	}

	content := strings.Join(lines, "\n")
	return lipgloss.NewStyle().Width(m.width).Height(height).Render(content)
}

func (m *PRDetailModel) renderFilesPane(height int) string {
	t := m.styles.Theme
	d := m.detail

	if len(d.Files) == 0 {
		return lipgloss.NewStyle().Foreground(t.Muted).Render("No files changed.")
	}

	var lines []string
	for _, f := range d.Files {
		addStyle := lipgloss.NewStyle().Foreground(t.Success)
		delStyle := lipgloss.NewStyle().Foreground(t.Error)
		line := fmt.Sprintf("  %s  %s %s",
			f.Path,
			addStyle.Render(fmt.Sprintf("+%d", f.Additions)),
			delStyle.Render(fmt.Sprintf("-%d", f.Deletions)),
		)
		lines = append(lines, line)
	}

	return lipgloss.NewStyle().Width(m.width).Height(height).
		Render(strings.Join(lines, "\n"))
}

func (m *PRDetailModel) renderChecksPane(height int) string {
	t := m.styles.Theme
	d := m.detail

	if len(d.Checks) == 0 {
		return lipgloss.NewStyle().Foreground(t.Muted).Render("No CI checks.")
	}

	var lines []string
	for _, c := range d.Checks {
		icon := ciIcon(c.Status)
		dur := ""
		if c.Duration > 0 {
			dur = fmt.Sprintf(" (%s)", c.Duration.Truncate(1e9))
		}
		lines = append(lines, fmt.Sprintf("  %s %s%s", icon, c.Name, dur))
	}

	return lipgloss.NewStyle().Width(m.width).Height(height).
		Render(strings.Join(lines, "\n"))
}

func (m *PRDetailModel) renderCommentsPane(height int) string {
	t := m.styles.Theme
	d := m.detail

	if len(d.Comments) == 0 {
		return lipgloss.NewStyle().Foreground(t.Muted).Render("No comments.")
	}

	var lines []string
	for _, thread := range d.Comments {
		header := fmt.Sprintf("  %s:%d", thread.Path, thread.Line)
		if thread.Resolved {
			header += " [resolved]"
		}
		lines = append(lines, lipgloss.NewStyle().Foreground(t.Info).Bold(true).Render(header))

		for _, c := range thread.Comments {
			lines = append(lines,
				fmt.Sprintf("    @%s: %s", c.Author, c.Body),
			)
		}
		lines = append(lines, "")
	}

	return lipgloss.NewStyle().Width(m.width).Height(height).
		Render(strings.Join(lines, "\n"))
}
