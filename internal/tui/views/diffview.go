package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/indrasvat/vivecaka/internal/tui"
)

// DiffViewModel implements the diff viewer.
type DiffViewModel struct {
	diff        *domain.Diff
	width       int
	height      int
	styles      tui.Styles
	keys        tui.KeyMap
	fileIdx     int
	scrollY     int
	loading     bool
	searchQuery string
	searching   bool
}

// NewDiffViewModel creates a new diff viewer.
func NewDiffViewModel(styles tui.Styles, keys tui.KeyMap) DiffViewModel {
	return DiffViewModel{
		styles:  styles,
		keys:    keys,
		loading: true,
	}
}

// SetSize updates the view dimensions.
func (m *DiffViewModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetDiff updates the displayed diff.
func (m *DiffViewModel) SetDiff(d *domain.Diff) {
	m.diff = d
	m.loading = false
	m.fileIdx = 0
	m.scrollY = 0
}

// Message types.
type DiffLoadedMsg struct{ Diff *domain.Diff }

// Update handles messages for the diff view.
func (m *DiffViewModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case DiffLoadedMsg:
		m.SetDiff(msg.Diff)
	}
	return nil
}

func (m *DiffViewModel) handleKey(msg tea.KeyMsg) tea.Cmd {
	if m.searching {
		return m.handleSearchKey(msg)
	}

	switch {
	case key.Matches(msg, m.keys.Down):
		m.scrollY++
	case key.Matches(msg, m.keys.Up):
		if m.scrollY > 0 {
			m.scrollY--
		}
	case key.Matches(msg, m.keys.HalfPageDown):
		m.scrollY += m.height / 2
	case key.Matches(msg, m.keys.HalfPageUp):
		m.scrollY -= m.height / 2
		if m.scrollY < 0 {
			m.scrollY = 0
		}
	case key.Matches(msg, m.keys.Tab):
		// Next file.
		if m.diff != nil && m.fileIdx < len(m.diff.Files)-1 {
			m.fileIdx++
			m.scrollY = 0
		}
	case key.Matches(msg, m.keys.ShiftTab):
		// Previous file.
		if m.fileIdx > 0 {
			m.fileIdx--
			m.scrollY = 0
		}
	case key.Matches(msg, m.keys.Search):
		m.searching = true
		m.searchQuery = ""
	}
	return nil
}

func (m *DiffViewModel) handleSearchKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.Type {
	case tea.KeyEscape:
		m.searching = false
		m.searchQuery = ""
	case tea.KeyEnter:
		m.searching = false
	case tea.KeyBackspace:
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
		}
	case tea.KeyRunes:
		m.searchQuery += string(msg.Runes)
	}
	return nil
}

// View renders the diff viewer.
func (m *DiffViewModel) View() string {
	if m.loading || m.diff == nil {
		return lipgloss.NewStyle().
			Width(m.width).Height(m.height).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(m.styles.Theme.Muted).
			Render("Loading diff...")
	}

	if len(m.diff.Files) == 0 {
		return lipgloss.NewStyle().
			Width(m.width).Height(m.height).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(m.styles.Theme.Muted).
			Render("No files changed")
	}

	t := m.styles.Theme

	// File tab bar.
	fileBar := m.renderFileBar()

	// Diff content for current file.
	contentHeight := m.height - 2
	if contentHeight < 1 {
		contentHeight = 1
	}

	file := m.diff.Files[m.fileIdx]
	var lines []string

	for _, hunk := range file.Hunks {
		// Hunk header.
		hunkStyle := lipgloss.NewStyle().Foreground(t.Info)
		lines = append(lines, hunkStyle.Render(hunk.Header))

		for _, dl := range hunk.Lines {
			lineNum := ""
			switch dl.Type {
			case domain.DiffAdd:
				lineNum = fmt.Sprintf("%4s %4d ", "", dl.NewNum)
				line := m.styles.DiffAdd.Render(lineNum + "+" + dl.Content)
				lines = append(lines, line)
			case domain.DiffDelete:
				lineNum = fmt.Sprintf("%4d %4s ", dl.OldNum, "")
				line := m.styles.DiffDelete.Render(lineNum + "-" + dl.Content)
				lines = append(lines, line)
			default:
				lineNum = fmt.Sprintf("%4d %4d ", dl.OldNum, dl.NewNum)
				line := lipgloss.NewStyle().Foreground(t.Fg).Render(lineNum + " " + dl.Content)
				lines = append(lines, line)
			}
		}
	}

	// Apply scroll.
	if m.scrollY >= len(lines) {
		m.scrollY = max(0, len(lines)-1)
	}
	end := m.scrollY + contentHeight
	if end > len(lines) {
		end = len(lines)
	}
	visible := lines[m.scrollY:end]

	content := strings.Join(visible, "\n")

	// Search bar.
	if m.searching {
		search := lipgloss.NewStyle().Foreground(t.Info).Render(fmt.Sprintf("/ %s▎", m.searchQuery))
		content += "\n" + search
	}

	return lipgloss.JoinVertical(lipgloss.Left, fileBar, content)
}

func (m *DiffViewModel) renderFileBar() string {
	t := m.styles.Theme
	active := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Padding(0, 1)
	inactive := lipgloss.NewStyle().Foreground(t.Muted).Padding(0, 1)

	var tabs []string
	for i, f := range m.diff.Files {
		// Shorten path for display.
		name := f.Path
		if len(name) > 20 {
			name = "…" + name[len(name)-19:]
		}
		if i == m.fileIdx {
			tabs = append(tabs, active.Render(name))
		} else {
			tabs = append(tabs, inactive.Render(name))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
}
