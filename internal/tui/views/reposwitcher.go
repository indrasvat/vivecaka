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

// RepoEntry represents a repo in the switcher list.
type RepoEntry struct {
	Repo      domain.RepoRef
	Favorite  bool
	OpenCount int // number of open PRs
	Current   bool
}

// RepoSwitcherModel implements the repo switcher overlay.
type RepoSwitcherModel struct {
	repos    []RepoEntry
	filtered []RepoEntry
	cursor   int
	query    string
	width    int
	height   int
	styles   tui.Styles
	keys     tui.KeyMap
}

// NewRepoSwitcherModel creates a new repo switcher.
func NewRepoSwitcherModel(styles tui.Styles, keys tui.KeyMap) RepoSwitcherModel {
	return RepoSwitcherModel{
		styles: styles,
		keys:   keys,
	}
}

// SetSize updates the overlay dimensions.
func (m *RepoSwitcherModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetRepos updates the repo list.
func (m *RepoSwitcherModel) SetRepos(repos []RepoEntry) {
	m.repos = repos
	m.query = ""
	m.cursor = 0
	m.applyFilter()
}

// SwitchRepoMsg is sent when the user selects a repo.
type SwitchRepoMsg struct {
	Repo domain.RepoRef
}

// CloseRepoSwitcherMsg is sent when the overlay is dismissed.
type CloseRepoSwitcherMsg struct{}

// Update handles messages for the repo switcher.
func (m *RepoSwitcherModel) Update(msg tea.Msg) tea.Cmd {
	if msg, ok := msg.(tea.KeyMsg); ok {
		return m.handleKey(msg)
	}
	return nil
}

func (m *RepoSwitcherModel) handleKey(msg tea.KeyMsg) tea.Cmd {
	// Check navigation bindings first (j/k/arrows).
	switch {
	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
		}
		return nil
	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
		return nil
	}

	switch msg.Type {
	case tea.KeyEscape:
		return func() tea.Msg { return CloseRepoSwitcherMsg{} }
	case tea.KeyEnter:
		if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
			return func() tea.Msg { return SwitchRepoMsg{Repo: m.filtered[m.cursor].Repo} }
		}
	case tea.KeyBackspace:
		if len(m.query) > 0 {
			m.query = m.query[:len(m.query)-1]
			m.applyFilter()
		}
	case tea.KeyRunes:
		m.query += string(msg.Runes)
		m.applyFilter()
	}
	return nil
}

func (m *RepoSwitcherModel) applyFilter() {
	if m.query == "" {
		m.filtered = m.repos
	} else {
		q := strings.ToLower(m.query)
		m.filtered = nil
		for _, r := range m.repos {
			name := strings.ToLower(r.Repo.String())
			if fuzzyMatch(name, q) {
				m.filtered = append(m.filtered, r)
			}
		}
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

// fuzzyMatch checks if all characters in query appear in order within text.
func fuzzyMatch(text, query string) bool {
	qi := 0
	for i := range len(text) {
		if qi < len(query) && text[i] == query[qi] {
			qi++
		}
	}
	return qi == len(query)
}

// View renders the repo switcher overlay.
func (m *RepoSwitcherModel) View() string {
	t := m.styles.Theme

	boxWidth := max(30, min(60, m.width-4))
	innerWidth := boxWidth - 4 // border + padding

	titleStyle := lipgloss.NewStyle().Foreground(t.Primary).Bold(true)
	searchStyle := lipgloss.NewStyle().Foreground(t.Info)
	activeStyle := lipgloss.NewStyle().Foreground(t.Fg).Background(t.Border).Width(innerWidth)
	normalStyle := lipgloss.NewStyle().Foreground(t.Fg).Width(innerWidth)
	mutedStyle := lipgloss.NewStyle().Foreground(t.Muted)
	favStyle := lipgloss.NewStyle().Foreground(t.Warning)
	currentStyle := lipgloss.NewStyle().Foreground(t.Success)

	var lines []string
	lines = append(lines, titleStyle.Render("Switch Repo"))
	lines = append(lines, "")
	lines = append(lines, searchStyle.Render(fmt.Sprintf("  Search: %s▎", m.query)))
	lines = append(lines, "")

	maxVisible := max(3, m.height-10)
	start := 0
	if m.cursor >= maxVisible {
		start = m.cursor - maxVisible + 1
	}
	end := min(start+maxVisible, len(m.filtered))

	for i := start; i < end; i++ {
		r := m.filtered[i]
		prefix := "  "
		if r.Favorite {
			prefix = favStyle.Render("★") + " "
		}

		name := r.Repo.String()
		suffix := ""
		if r.Current {
			suffix = currentStyle.Render(" (current)")
		} else if r.OpenCount > 0 {
			suffix = mutedStyle.Render(fmt.Sprintf(" %d open PRs", r.OpenCount))
		}

		entry := prefix + name + suffix
		if i == m.cursor {
			lines = append(lines, activeStyle.Render(entry))
		} else {
			lines = append(lines, normalStyle.Render(entry))
		}
	}

	if len(m.filtered) == 0 {
		lines = append(lines, mutedStyle.Render("  No repos matching query"))
	}

	lines = append(lines, "")
	lines = append(lines, mutedStyle.Render("  ★ = favorite    ↵ switch    Esc cancel"))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border).
		Padding(1).
		Width(boxWidth).
		Render(content)

	// Center the overlay.
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}
