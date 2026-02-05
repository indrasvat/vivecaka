package views

import (
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/indrasvat/vivecaka/internal/config"
	"github.com/indrasvat/vivecaka/internal/tui/core"
)

// TutorialStep represents a single step in the first-launch tutorial.
type TutorialStep struct {
	Title string
	Body  string
}

var tutorialSteps = []TutorialStep{
	{
		Title: "Welcome to vivecaka!",
		Body:  "vivecaka is a keyboard-driven GitHub PR browser\nfor your terminal. Let's get you started.",
	},
	{
		Title: "Navigation",
		Body:  "Use j/k or arrow keys to move up and down.\ng goes to top, G goes to bottom.\nCtrl+d and Ctrl+u scroll half a page.",
	},
	{
		Title: "Opening PRs",
		Body:  "Press Enter to open a PR detail view.\nUse Tab to switch between Info, Files,\nChecks, and Comments panes.",
	},
	{
		Title: "Actions",
		Body:  "c  Checkout the PR branch\no  Open in browser\ny  Copy PR URL\nr  Submit a review",
	},
	{
		Title: "Getting Help",
		Body:  "Press ? at any time to see all keybindings\nfor the current view.\n\nYou're all set. Press Enter to begin!",
	},
}

// TutorialModel implements the first-launch tutorial overlay.
type TutorialModel struct {
	step    int
	width   int
	height  int
	styles  core.Styles
	visible bool
}

// NewTutorialModel creates a new tutorial model.
func NewTutorialModel(styles core.Styles) TutorialModel {
	return TutorialModel{styles: styles}
}

// SetSize updates the overlay dimensions.
func (m *TutorialModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// Show makes the tutorial visible.
func (m *TutorialModel) Show() {
	m.visible = true
	m.step = 0
}

// Visible returns whether the tutorial is being shown.
func (m *TutorialModel) Visible() bool {
	return m.visible
}

// TutorialDoneMsg is sent when the tutorial is dismissed.
type TutorialDoneMsg struct{}

// Update handles messages for the tutorial.
func (m *TutorialModel) Update(msg tea.Msg) tea.Cmd {
	if !m.visible {
		return nil
	}

	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.Type {
		case tea.KeyEscape:
			m.visible = false
			return func() tea.Msg { return TutorialDoneMsg{} }
		case tea.KeyEnter, tea.KeySpace:
			if m.step < len(tutorialSteps)-1 {
				m.step++
			} else {
				m.visible = false
				return func() tea.Msg { return TutorialDoneMsg{} }
			}
		}
	}
	return nil
}

// View renders the tutorial overlay.
func (m *TutorialModel) View() string {
	if !m.visible {
		return ""
	}

	t := m.styles.Theme
	step := tutorialSteps[m.step]

	titleStyle := lipgloss.NewStyle().Foreground(t.Primary).Bold(true)
	bodyStyle := lipgloss.NewStyle().Foreground(t.Fg)
	progressStyle := lipgloss.NewStyle().Foreground(t.Muted)
	hintStyle := lipgloss.NewStyle().Foreground(t.Info)

	title := titleStyle.Render(step.Title)
	body := bodyStyle.Render(step.Body)
	progress := progressStyle.Render(progressDots(m.step, len(tutorialSteps)))
	hint := hintStyle.Render("Enter/Space: next    Esc: skip")

	content := lipgloss.JoinVertical(lipgloss.Center, title, "", body, "", progress, "", hint)

	framed := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Primary).
		Padding(1, 3).
		Width(min(50, m.width-4)).
		Render(content)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, framed)
}

// progressDots returns a dot progress indicator like "● ● ○ ○ ○".
func progressDots(current, total int) string {
	var dots []byte
	for i := range total {
		if i > 0 {
			dots = append(dots, ' ')
		}
		if i <= current {
			dots = append(dots, "●"...)
		} else {
			dots = append(dots, "○"...)
		}
	}
	return string(dots)
}

const tutorialDoneFile = "tutorial_done"

// IsFirstLaunch checks if the tutorial has been shown before.
func IsFirstLaunch() bool {
	path := filepath.Join(config.DataDir(), tutorialDoneFile)
	_, err := os.Stat(path)
	return os.IsNotExist(err)
}

// MarkTutorialDone writes the flag file so the tutorial won't show again.
func MarkTutorialDone() error {
	dir := config.DataDir()
	if err := config.EnsureDir(dir); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, tutorialDoneFile), []byte("done"), 0o644)
}
