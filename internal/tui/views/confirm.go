package views

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/indrasvat/vivecaka/internal/tui/core"
)

// confirmState tracks which phase the dialog is in.
type confirmState int

const (
	confirmPrompt  confirmState = iota // y/n confirmation
	confirmLoading                     // spinner while action runs
	confirmResult                      // success/error result
)

// ConfirmModel is a reusable action dialog.
// It shows a centered bordered box that transitions through:
// prompt → loading (spinner) → result (success/error) → dismiss.
type ConfirmModel struct {
	state        confirmState
	title        string
	message      string
	confirmLabel string
	cancelLabel  string
	onConfirm    tea.Msg // message to emit when confirmed
	width        int
	height       int
	styles       core.Styles

	// Loading state
	spinnerFrame int

	// Result state
	resultSuccess bool
	resultMessage string
}

// SetStyles updates the styles without losing state.
func (m *ConfirmModel) SetStyles(s core.Styles) { m.styles = s }

// NewConfirmModel creates a new confirmation dialog.
func NewConfirmModel(styles core.Styles) ConfirmModel {
	return ConfirmModel{
		styles:       styles,
		confirmLabel: "Yes",
		cancelLabel:  "No",
	}
}

// ConfirmResultMsg is emitted when the user confirms.
type ConfirmResultMsg struct {
	Confirmed bool
	Action    tea.Msg
}

// CloseConfirmMsg is emitted when the dialog should close.
type CloseConfirmMsg struct{}

// confirmSpinnerTickMsg drives the loading spinner.
type confirmSpinnerTickMsg struct{}

// Show configures and activates the confirmation prompt.
func (m *ConfirmModel) Show(title, message string, onConfirm tea.Msg) {
	m.state = confirmPrompt
	m.title = title
	m.message = message
	m.onConfirm = onConfirm
}

// ShowLoading transitions the dialog to a loading spinner state.
func (m *ConfirmModel) ShowLoading(title, message string) tea.Cmd {
	m.state = confirmLoading
	m.title = title
	m.message = message
	m.spinnerFrame = 0
	return m.spinnerTick()
}

// ShowResult transitions the dialog to show a success or error result.
func (m *ConfirmModel) ShowResult(title, message string, success bool) {
	m.state = confirmResult
	m.title = title
	m.resultMessage = message
	m.resultSuccess = success
}

// Active returns whether the dialog is currently visible.
func (m *ConfirmModel) Active() bool {
	return m.state != confirmPrompt || m.title != ""
}

// IsLoading returns whether the dialog is in loading state.
func (m *ConfirmModel) IsLoading() bool {
	return m.state == confirmLoading
}

// SetSize updates dimensions.
func (m *ConfirmModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *ConfirmModel) spinnerTick() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(_ time.Time) tea.Msg {
		return confirmSpinnerTickMsg{}
	})
}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Update handles messages for the confirmation dialog.
func (m *ConfirmModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case confirmSpinnerTickMsg:
		if m.state == confirmLoading {
			m.spinnerFrame++
			return m.spinnerTick()
		}
		return nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return nil
}

func (m *ConfirmModel) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch m.state {
	case confirmPrompt:
		return m.handlePromptKey(msg)
	case confirmLoading:
		// No key interaction during loading
		return nil
	case confirmResult:
		// Any key dismisses the result
		m.reset()
		return func() tea.Msg { return CloseConfirmMsg{} }
	}
	return nil
}

func (m *ConfirmModel) handlePromptKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.Type {
	case tea.KeyEnter:
		action := m.onConfirm
		return func() tea.Msg { return ConfirmResultMsg{Confirmed: true, Action: action} }
	case tea.KeyEscape:
		m.reset()
		return func() tea.Msg { return CloseConfirmMsg{} }
	case tea.KeyRunes:
		if len(msg.Runes) == 1 {
			switch msg.Runes[0] {
			case 'y', 'Y':
				action := m.onConfirm
				return func() tea.Msg { return ConfirmResultMsg{Confirmed: true, Action: action} }
			case 'n', 'N':
				m.reset()
				return func() tea.Msg { return CloseConfirmMsg{} }
			}
		}
	}
	return nil
}

func (m *ConfirmModel) reset() {
	m.state = confirmPrompt
	m.title = ""
	m.message = ""
	m.resultMessage = ""
	m.onConfirm = nil
}

// View renders the dialog based on current state.
func (m *ConfirmModel) View() string {
	switch m.state {
	case confirmPrompt:
		return m.viewPrompt()
	case confirmLoading:
		return m.viewLoading()
	case confirmResult:
		return m.viewResult()
	}
	return ""
}

func (m *ConfirmModel) viewPrompt() string {
	t := m.styles.Theme
	boxWidth := max(30, min(60, m.width-4))

	titleStyle := lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true)

	msgStyle := lipgloss.NewStyle().
		Foreground(t.Fg).
		Width(boxWidth - 6)

	confirmKey := lipgloss.NewStyle().
		Foreground(t.Success).
		Bold(true).
		Render("Enter/y")
	cancelKey := lipgloss.NewStyle().
		Foreground(t.Error).
		Bold(true).
		Render("Esc/n")

	hints := fmt.Sprintf("%s %s   %s %s",
		confirmKey, m.confirmLabel, cancelKey, m.cancelLabel)

	inner := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(m.title),
		"",
		msgStyle.Render(m.message),
		"",
		hints,
	)

	return m.renderBox(inner, boxWidth, string(t.Primary))
}

func (m *ConfirmModel) viewLoading() string {
	t := m.styles.Theme
	boxWidth := max(30, min(60, m.width-4))

	titleStyle := lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true)

	frame := spinnerFrames[m.spinnerFrame%len(spinnerFrames)]
	spinner := lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true).
		Render(frame)

	msgStyle := lipgloss.NewStyle().
		Foreground(t.Fg).
		Width(boxWidth - 6)

	inner := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(m.title),
		"",
		spinner+" "+msgStyle.Render(m.message),
	)

	return m.renderBox(inner, boxWidth, string(t.Primary))
}

func (m *ConfirmModel) viewResult() string {
	t := m.styles.Theme
	boxWidth := max(30, min(60, m.width-4))

	var icon string
	var titleColor, borderColor lipgloss.Color
	if m.resultSuccess {
		icon = "✓"
		titleColor = t.Success
		borderColor = t.Success
	} else {
		icon = "✗"
		titleColor = t.Error
		borderColor = t.Error
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(titleColor).
		Bold(true)

	msgStyle := lipgloss.NewStyle().
		Foreground(t.Fg).
		Width(boxWidth - 6)

	hintStyle := lipgloss.NewStyle().
		Foreground(t.Muted).
		Italic(true)

	inner := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(icon+" "+m.title),
		"",
		msgStyle.Render(m.resultMessage),
		"",
		hintStyle.Render("Press any key to continue"),
	)

	return m.renderBox(inner, boxWidth, string(borderColor))
}

func (m *ConfirmModel) renderBox(inner string, boxWidth int, borderColor string) string {
	boxStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(borderColor)).
		Padding(1, 2).
		Width(boxWidth)

	box := boxStyle.Render(inner)
	centered := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
	return ensureExactHeight(centered, m.height, m.width)
}

// ConfirmStateHint returns status bar text based on confirm dialog state.
func (m *ConfirmModel) ConfirmStateHint() string {
	switch m.state {
	case confirmLoading:
		return "Checking out..."
	case confirmResult:
		return "Press any key to continue"
	default:
		return "Enter/y confirm  Esc/n cancel"
	}
}
