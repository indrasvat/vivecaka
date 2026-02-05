package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/indrasvat/vivecaka/internal/tui/core"
)

// HelpModel implements the context-aware help overlay.
type HelpModel struct {
	context core.ViewState
	width   int
	height  int
	styles  core.Styles
}

// NewHelpModel creates a new help overlay.
func NewHelpModel(styles core.Styles) HelpModel {
	return HelpModel{styles: styles}
}

// SetSize updates the overlay dimensions.
func (m *HelpModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetContext sets which view the help is for.
func (m *HelpModel) SetContext(view core.ViewState) {
	m.context = view
}

// CloseHelpMsg is sent when help is dismissed.
type CloseHelpMsg struct{}

// Update handles messages for the help overlay.
func (m *HelpModel) Update(msg tea.Msg) tea.Cmd {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.Type {
		case tea.KeyEscape:
			return func() tea.Msg { return CloseHelpMsg{} }
		case tea.KeyRunes:
			if len(msg.Runes) == 1 && msg.Runes[0] == '?' {
				return func() tea.Msg { return CloseHelpMsg{} }
			}
		}
	}
	return nil
}

// helpBinding describes a single key binding for display.
type helpBinding struct {
	key  string
	desc string
}

// View renders the help overlay.
func (m *HelpModel) View() string {
	t := m.styles.Theme

	titleStyle := lipgloss.NewStyle().Foreground(t.Primary).Bold(true)
	headingStyle := lipgloss.NewStyle().Foreground(t.Info).Bold(true).Underline(true)
	keyStyle := lipgloss.NewStyle().Foreground(t.Warning).Width(12)
	descStyle := lipgloss.NewStyle().Foreground(t.Fg)
	footerStyle := lipgloss.NewStyle().Foreground(t.Muted)

	title := titleStyle.Render(fmt.Sprintf("Help: %s", m.contextName()))

	// Build sections based on context.
	left, right := m.bindings()

	leftSection := m.renderSection(headingStyle, keyStyle, descStyle, left)
	rightSection := m.renderSection(headingStyle, keyStyle, descStyle, right)

	// Side by side.
	colWidth := max(20, (m.width-8)/2)

	leftCol := lipgloss.NewStyle().Width(colWidth).Render(leftSection)
	rightCol := lipgloss.NewStyle().Width(colWidth).Render(rightSection)

	content := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, "  ", rightCol)

	footer := footerStyle.Render("Press ? or Esc to close")

	box := lipgloss.JoinVertical(lipgloss.Left, title, "", content, "", footer)

	framed := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border).
		Padding(1, 2).
		Render(box)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, framed)
}

type helpSection struct {
	title    string
	bindings []helpBinding
}

func (m *HelpModel) renderSection(headingStyle, keyStyle, descStyle lipgloss.Style, sections []helpSection) string {
	var lines []string
	for i, s := range sections {
		if i > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, headingStyle.Render(s.title))
		for _, b := range s.bindings {
			lines = append(lines, keyStyle.Render(b.key)+descStyle.Render(b.desc))
		}
	}
	return strings.Join(lines, "\n")
}

func (m *HelpModel) contextName() string {
	switch m.context {
	case core.ViewPRList:
		return "PR List"
	case core.ViewPRDetail:
		return "PR Detail"
	case core.ViewDiff:
		return "Diff"
	case core.ViewReview:
		return "Review"
	case core.ViewInbox:
		return "Inbox"
	default:
		return "General"
	}
}

func (m *HelpModel) bindings() (left, right []helpSection) {
	global := helpSection{
		title: "Global",
		bindings: []helpBinding{
			{"Ctrl+R", "Switch repo"},
			{"T", "Cycle theme"},
			{"?", "Toggle help"},
			{"q", "Quit / Back"},
		},
	}

	switch m.context {
	case core.ViewPRList:
		left = []helpSection{
			{
				title: "Navigation",
				bindings: []helpBinding{
					{"j/k", "Move up/down"},
					{"g/G", "Top / bottom"},
					{"Ctrl+d", "Half page down"},
					{"Ctrl+u", "Half page up"},
				},
			},
			{
				title: "Search & Filter",
				bindings: []helpBinding{
					{"/", "Search PRs"},
					{"Esc", "Clear search"},
					{"s", "Cycle sort"},
				},
			},
		}
		right = []helpSection{
			{
				title: "Actions",
				bindings: []helpBinding{
					{"Enter", "Open PR detail"},
					{"c", "Checkout branch"},
					{"o", "Open in browser"},
					{"y", "Copy PR URL"},
					{"I", "Toggle inbox"},
				},
			},
			global,
		}

	case core.ViewPRDetail:
		left = []helpSection{
			{
				title: "Navigation",
				bindings: []helpBinding{
					{"j/k", "Scroll up/down"},
					{"Tab", "Next pane"},
					{"Shift+Tab", "Previous pane"},
				},
			},
		}
		right = []helpSection{
			{
				title: "Actions",
				bindings: []helpBinding{
					{"Enter", "Open diff (Files)"},
					{"r", "Submit review"},
					{"Esc", "Back to list"},
				},
			},
			global,
		}

	case core.ViewDiff:
		left = []helpSection{
			{
				title: "Navigation",
				bindings: []helpBinding{
					{"j/k", "Scroll up/down"},
					{"Ctrl+d/u", "Half page"},
					{"Tab", "Next file"},
					{"Shift+Tab", "Previous file"},
				},
			},
		}
		right = []helpSection{
			{
				title: "Actions",
				bindings: []helpBinding{
					{"/", "Search in diff"},
					{"Esc", "Back to detail"},
				},
			},
			global,
		}

	case core.ViewReview:
		left = []helpSection{
			{
				title: "Navigation",
				bindings: []helpBinding{
					{"j/k", "Move between fields"},
					{"Enter", "Cycle action / edit / submit"},
					{"Esc", "Stop editing body"},
				},
			},
		}
		right = []helpSection{global}

	case core.ViewInbox:
		left = []helpSection{
			{
				title: "Navigation",
				bindings: []helpBinding{
					{"j/k", "Move up/down"},
					{"Tab", "Next tab"},
					{"Shift+Tab", "Previous tab"},
				},
			},
		}
		right = []helpSection{
			{
				title: "Actions",
				bindings: []helpBinding{
					{"Enter", "Open PR"},
					{"Esc", "Back to list"},
				},
			},
			global,
		}

	default:
		left = []helpSection{
			{
				title: "Navigation",
				bindings: []helpBinding{
					{"j/k", "Move up/down"},
					{"Enter", "Select"},
					{"Esc", "Back"},
				},
			},
		}
		right = []helpSection{global}
	}

	return left, right
}

// StatusHints returns context-aware key hints for the status bar.
func StatusHints(view core.ViewState, width int) string {
	var hints string
	switch view {
	case core.ViewPRList:
		hints = "j/k navigate  Enter open  c checkout  / search  ? help  q quit"
	case core.ViewPRDetail:
		hints = "j/k scroll  Tab pane  r review  Enter diff  Esc back  ? help"
	case core.ViewDiff:
		hints = "j/k scroll  Tab file  / search  Esc back  ? help"
	case core.ViewReview:
		hints = "j/k field  Enter action  Esc back  ? help"
	case core.ViewInbox:
		hints = "j/k navigate  Tab tab  Enter open  Esc back  ? help"
	case core.ViewRepoSwitch:
		hints = "j/k navigate  Enter switch  Esc cancel"
	case core.ViewHelp:
		hints = "? or Esc to close"
	default:
		hints = "? help  q quit"
	}

	// Truncate for narrow terminals.
	if width > 0 && len(hints) > width-2 {
		hints = hints[:width-5] + "..."
	}

	return hints
}
