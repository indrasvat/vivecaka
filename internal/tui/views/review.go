package views

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/indrasvat/vivecaka/internal/tui/core"
)

// ReviewModel implements the review submission form.
type ReviewModel struct {
	prNumber int
	width    int
	height   int
	styles   core.Styles
	keys     core.KeyMap
	action   domain.ReviewAction
	body     string
	cursor   int // 0=action, 1=body, 2=submit
	editing  bool
}

// NewReviewModel creates a new review form.
func NewReviewModel(styles core.Styles, keys core.KeyMap) ReviewModel {
	return ReviewModel{
		styles: styles,
		keys:   keys,
		action: domain.ReviewActionComment,
	}
}

// SetSize updates the view dimensions.
func (m *ReviewModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetPRNumber sets the PR being reviewed.
func (m *ReviewModel) SetPRNumber(n int) {
	m.prNumber = n
	m.action = domain.ReviewActionComment
	m.body = ""
	m.cursor = 0
}

// SubmitReviewMsg is sent when the user submits a review.
type SubmitReviewMsg struct {
	Number int
	Review domain.Review
}

// Update handles messages for the review form.
func (m *ReviewModel) Update(msg tea.Msg) tea.Cmd {
	if msg, ok := msg.(tea.KeyMsg); ok {
		return m.handleKey(msg)
	}
	return nil
}

func (m *ReviewModel) handleKey(msg tea.KeyMsg) tea.Cmd {
	if m.editing {
		return m.handleEditKey(msg)
	}

	switch {
	case key.Matches(msg, m.keys.Down):
		if m.cursor < 2 {
			m.cursor++
		}
	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
	case key.Matches(msg, m.keys.Enter):
		switch m.cursor {
		case 0:
			// Cycle action.
			m.cycleAction()
		case 1:
			// Start editing body.
			m.editing = true
		case 2:
			// Submit.
			return func() tea.Msg {
				return SubmitReviewMsg{
					Number: m.prNumber,
					Review: domain.Review{Action: m.action, Body: m.body},
				}
			}
		}
	}
	return nil
}

func (m *ReviewModel) handleEditKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.Type {
	case tea.KeyEscape:
		m.editing = false
	case tea.KeyBackspace:
		if len(m.body) > 0 {
			m.body = m.body[:len(m.body)-1]
		}
	case tea.KeyEnter:
		m.body += "\n"
	case tea.KeyRunes:
		m.body += string(msg.Runes)
	}
	return nil
}

func (m *ReviewModel) cycleAction() {
	switch m.action {
	case domain.ReviewActionComment:
		m.action = domain.ReviewActionApprove
	case domain.ReviewActionApprove:
		m.action = domain.ReviewActionRequestChanges
	case domain.ReviewActionRequestChanges:
		m.action = domain.ReviewActionComment
	}
}

// View renders the review form.
func (m *ReviewModel) View() string {
	t := m.styles.Theme
	titleStyle := lipgloss.NewStyle().Foreground(t.Primary).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(t.Muted)
	selectedStyle := lipgloss.NewStyle().Foreground(t.Fg).Bold(true)
	normalStyle := lipgloss.NewStyle().Foreground(t.Subtext)

	var lines []string
	lines = append(lines, titleStyle.Render("Submit Review"))
	lines = append(lines, "")

	// Action selector.
	actionLabel := "  Action: "
	actionValue := actionDisplay(m.action)
	if m.cursor == 0 {
		lines = append(lines, selectedStyle.Render("▸ "+actionLabel+actionValue+" (Enter to cycle)"))
	} else {
		lines = append(lines, normalStyle.Render("  "+actionLabel+actionValue))
	}

	// Body editor.
	bodyLabel := "  Body:   "
	bodyPreview := m.body
	if bodyPreview == "" {
		bodyPreview = "(empty)"
	}
	switch {
	case m.editing:
		lines = append(lines, selectedStyle.Render("▸ "+bodyLabel+bodyPreview+"▎"))
	case m.cursor == 1:
		lines = append(lines, selectedStyle.Render("▸ "+bodyLabel+bodyPreview+" (Enter to edit)"))
	default:
		lines = append(lines, normalStyle.Render("  "+bodyLabel+bodyPreview))
	}

	lines = append(lines, "")

	// Submit button.
	if m.cursor == 2 {
		lines = append(lines, selectedStyle.Render("▸ [ Submit ]"))
	} else {
		lines = append(lines, labelStyle.Render("  [ Submit ]"))
	}

	return lipgloss.NewStyle().Width(m.width).Height(m.height).Padding(1).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func actionDisplay(a domain.ReviewAction) string {
	switch a {
	case domain.ReviewActionApprove:
		return "Approve ✓"
	case domain.ReviewActionRequestChanges:
		return "Request Changes !"
	default:
		return "Comment"
	}
}
