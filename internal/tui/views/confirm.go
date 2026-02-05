package views

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/indrasvat/vivecaka/internal/tui/core"
)

// ConfirmModel is a reusable confirmation dialog.
// It shows a centered bordered box with a message and confirm/cancel options.
type ConfirmModel struct {
	title        string
	message      string
	confirmLabel string
	cancelLabel  string
	onConfirm    tea.Msg // message to emit when confirmed
	active       bool
	width        int
	height       int
	styles       core.Styles
}

// NewConfirmModel creates a new confirmation dialog.
func NewConfirmModel(styles core.Styles) ConfirmModel {
	return ConfirmModel{
		styles:       styles,
		confirmLabel: "Yes",
		cancelLabel:  "No",
	}
}

// ConfirmResultMsg is emitted when the user confirms.
// It wraps the original action message to be re-dispatched.
type ConfirmResultMsg struct {
	Confirmed bool
	Action    tea.Msg
}

// CloseConfirmMsg is emitted when the user cancels.
type CloseConfirmMsg struct{}

// Show configures and activates the confirmation dialog.
func (m *ConfirmModel) Show(title, message string, onConfirm tea.Msg) {
	m.title = title
	m.message = message
	m.onConfirm = onConfirm
	m.active = true
}

// Active returns whether the dialog is currently visible.
func (m *ConfirmModel) Active() bool {
	return m.active
}

// SetSize updates dimensions.
func (m *ConfirmModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// Update handles key messages for the confirmation dialog.
func (m *ConfirmModel) Update(msg tea.Msg) tea.Cmd {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}

	switch keyMsg.Type {
	case tea.KeyEnter:
		m.active = false
		action := m.onConfirm
		return func() tea.Msg { return ConfirmResultMsg{Confirmed: true, Action: action} }
	case tea.KeyEscape:
		m.active = false
		return func() tea.Msg { return CloseConfirmMsg{} }
	case tea.KeyRunes:
		if len(keyMsg.Runes) == 1 {
			switch keyMsg.Runes[0] {
			case 'y', 'Y':
				m.active = false
				action := m.onConfirm
				return func() tea.Msg { return ConfirmResultMsg{Confirmed: true, Action: action} }
			case 'n', 'N':
				m.active = false
				return func() tea.Msg { return CloseConfirmMsg{} }
			}
		}
	}
	return nil
}

// View renders the confirmation dialog as a centered box.
func (m *ConfirmModel) View() string {
	t := m.styles.Theme

	// Dialog box dimensions
	boxWidth := max(30, min(60, m.width-4))

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Primary)).
		Bold(true)

	// Message
	msgStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Fg)).
		Width(boxWidth - 4)

	// Key hints
	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Muted)).
		Italic(true)

	confirmKey := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Success)).
		Bold(true).
		Render("Enter/y")
	cancelKey := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Error)).
		Bold(true).
		Render("Esc/n")

	hints := fmt.Sprintf("%s %s   %s %s",
		confirmKey, m.confirmLabel, cancelKey, m.cancelLabel)

	// Build inner content
	inner := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(m.title),
		"",
		msgStyle.Render(m.message),
		"",
		hintStyle.Render(hints),
	)

	// Box border
	boxStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.Primary)).
		Padding(1, 2).
		Width(boxWidth)

	box := boxStyle.Render(inner)

	// Center in the available space
	centered := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)

	// Ensure exact height with full-width padding
	return ensureExactHeight(centered, m.height, m.width)
}

// StatusHintsConfirm returns status bar hints for the confirm dialog.
func StatusHintsConfirm() string {
	return "Enter/y confirm  Esc/n cancel"
}
