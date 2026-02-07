package views

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/indrasvat/vivecaka/internal/tui/core"
)

// ReviewModel implements the review submission form using huh library.
type ReviewModel struct {
	prNumber int
	width    int
	height   int
	styles   core.Styles
	keys     core.KeyMap
	form     *huh.Form
	action   domain.ReviewAction
	body     string
}

// SetStyles updates the styles without losing state.
func (m *ReviewModel) SetStyles(s core.Styles) { m.styles = s }

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
	if m.form != nil {
		m.form.WithWidth(w - 4)  // Account for padding
		m.form.WithHeight(h - 4) // Account for title and padding
	}
}

// SetPRNumber sets the PR being reviewed and initializes the form.
func (m *ReviewModel) SetPRNumber(n int) {
	m.prNumber = n
	m.action = domain.ReviewActionComment
	m.body = ""
	m.initForm()
}

// initForm creates the huh form with Select and Text fields.
func (m *ReviewModel) initForm() {
	// Action options
	actionOptions := []huh.Option[domain.ReviewAction]{
		huh.NewOption("Comment", domain.ReviewActionComment),
		huh.NewOption("Approve ✓", domain.ReviewActionApprove),
		huh.NewOption("Request Changes !", domain.ReviewActionRequestChanges),
	}

	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[domain.ReviewAction]().
				Title("Review Action").
				Description("Select the type of review to submit").
				Options(actionOptions...).
				Value(&m.action).
				Key("action"),
			huh.NewText().
				Title("Review Body").
				Description("Enter your review comments (optional)").
				Placeholder("Add your review comments here...").
				Value(&m.body).
				Lines(8).
				CharLimit(4000).
				Key("body"),
			huh.NewConfirm().
				Title("Submit Review?").
				Description("This will submit your review").
				Affirmative("Submit").
				Negative("Cancel").
				Key("confirm"),
		),
	)

	// Apply custom styling
	m.form.WithTheme(m.createTheme())
	if m.width > 4 {
		m.form.WithWidth(m.width - 4) // Account for padding
	}
	if m.height > 4 {
		m.form.WithHeight(m.height - 4)
	}
}

// createTheme returns a huh theme based on the app's styles.
func (m *ReviewModel) createTheme() *huh.Theme {
	t := m.styles.Theme
	theme := huh.ThemeDracula()

	// Customize to match vivecaka theme
	theme.Focused.Title = lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true)
	theme.Focused.Description = lipgloss.NewStyle().
		Foreground(t.Muted)
	theme.Focused.Base = lipgloss.NewStyle().
		BorderForeground(t.Primary).
		BorderStyle(lipgloss.RoundedBorder())
	theme.Focused.SelectedOption = lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true)
	theme.Focused.TextInput.Cursor = lipgloss.NewStyle().
		Foreground(t.Primary)

	return theme
}

// SubmitReviewMsg is sent when the user submits a review.
type SubmitReviewMsg struct {
	Number int
	Review domain.Review
}

// CloseReviewMsg is sent when the user cancels the review form.
type CloseReviewMsg struct{}

// Init returns the initial command for the form.
func (m *ReviewModel) Init() tea.Cmd {
	if m.form != nil {
		return m.form.Init()
	}
	return nil
}

// Update handles messages for the review form.
func (m *ReviewModel) Update(msg tea.Msg) tea.Cmd {
	// Check for Escape key to cancel (before nil form check so Escape
	// always works, even if the form hasn't initialized yet).
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.Type == tea.KeyEscape {
			return func() tea.Msg { return CloseReviewMsg{} }
		}
	}

	if m.form == nil {
		return nil
	}

	// Update the form
	model, cmd := m.form.Update(msg)
	if f, ok := model.(*huh.Form); ok {
		m.form = f
	}

	// Check form state after update
	switch m.form.State {
	case huh.StateCompleted:
		// User confirmed - check if they pressed Submit or Cancel
		confirm := m.form.GetBool("confirm")
		if confirm {
			return func() tea.Msg {
				return SubmitReviewMsg{
					Number: m.prNumber,
					Review: domain.Review{Action: m.action, Body: m.body},
				}
			}
		}
		// User pressed Cancel on confirm
		return func() tea.Msg { return CloseReviewMsg{} }
	case huh.StateAborted:
		return func() tea.Msg { return CloseReviewMsg{} }
	}

	return cmd
}

// View renders the review form.
func (m *ReviewModel) View() string {
	t := m.styles.Theme
	titleStyle := lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true).
		MarginBottom(1)

	var content string
	if m.form != nil {
		content = m.form.View()
	} else {
		content = "Loading form..."
	}

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Padding(1).
		Render(lipgloss.JoinVertical(
			lipgloss.Left,
			titleStyle.Render("Submit Review for PR #"+itoa(m.prNumber)),
			content,
		))
}

// itoa converts int to string (simple helper).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}
