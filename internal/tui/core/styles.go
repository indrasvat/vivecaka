package core

import "github.com/charmbracelet/lipgloss"

// Styles provides pre-built lipgloss styles based on the active theme.
type Styles struct {
	Theme Theme

	// Layout
	Header    lipgloss.Style
	Content   lipgloss.Style
	StatusBar lipgloss.Style

	// Text
	Title    lipgloss.Style
	Subtitle lipgloss.Style
	Muted    lipgloss.Style
	Bold     lipgloss.Style

	// Status indicators
	Success lipgloss.Style
	Error   lipgloss.Style
	Warning lipgloss.Style
	Info    lipgloss.Style

	// Interactive
	Selected    lipgloss.Style
	Focused     lipgloss.Style
	ActiveTab   lipgloss.Style
	InactiveTab lipgloss.Style

	// Diff
	DiffAdd    lipgloss.Style
	DiffDelete lipgloss.Style

	// Borders
	Border       lipgloss.Style
	ActiveBorder lipgloss.Style
}

// NewStyles creates a Styles from a Theme.
func NewStyles(t Theme) Styles {
	return Styles{
		Theme: t,

		Header: lipgloss.NewStyle().
			Foreground(t.Fg).
			Background(t.BgDim).
			Padding(0, 1),

		Content: lipgloss.NewStyle(),

		StatusBar: lipgloss.NewStyle().
			Foreground(t.Subtext).
			Background(t.BgDim).
			Padding(0, 1),

		Title: lipgloss.NewStyle().
			Foreground(t.Primary).
			Bold(true),

		Subtitle: lipgloss.NewStyle().
			Foreground(t.Subtext),

		Muted: lipgloss.NewStyle().
			Foreground(t.Muted),

		Bold: lipgloss.NewStyle().
			Bold(true),

		Success: lipgloss.NewStyle().
			Foreground(t.Success),

		Error: lipgloss.NewStyle().
			Foreground(t.Error),

		Warning: lipgloss.NewStyle().
			Foreground(t.Warning),

		Info: lipgloss.NewStyle().
			Foreground(t.Info),

		Selected: lipgloss.NewStyle().
			Foreground(t.Fg).
			Background(t.Border),

		Focused: lipgloss.NewStyle().
			Foreground(t.Primary).
			Bold(true),

		ActiveTab: lipgloss.NewStyle().
			Foreground(t.Fg).
			Background(t.Primary).
			Padding(0, 1),

		InactiveTab: lipgloss.NewStyle().
			Foreground(t.Muted).
			Padding(0, 1),

		DiffAdd: lipgloss.NewStyle().
			Foreground(t.Success),

		DiffDelete: lipgloss.NewStyle().
			Foreground(t.Error),

		Border: lipgloss.NewStyle().
			BorderForeground(t.Border),

		ActiveBorder: lipgloss.NewStyle().
			BorderForeground(t.Primary),
	}
}
