package components

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/indrasvat/vivecaka/internal/tui/core"
)

// Banner renders the startup ASCII art banner.
type Banner struct {
	styles  core.Styles
	version string
	visible bool
	width   int
	height  int
}

// NewBanner creates a new Banner component.
func NewBanner(styles core.Styles, version string) *Banner {
	return &Banner{
		styles:  styles,
		version: version,
		visible: true,
	}
}

// SetSize updates the banner dimensions.
func (b *Banner) SetSize(w, h int) {
	b.width = w
	b.height = h
}

// Visible returns whether the banner is currently visible.
func (b *Banner) Visible() bool {
	return b.visible
}

// Hide hides the banner.
func (b *Banner) Hide() {
	b.visible = false
}

// BannerDismissMsg is sent when the banner auto-dismisses.
type BannerDismissMsg struct{}

// Update handles input for the banner (any key dismisses it).
func (b *Banner) Update(msg tea.Msg) tea.Cmd {
	if !b.visible {
		return nil
	}

	switch msg.(type) {
	case tea.KeyMsg:
		b.visible = false
		return nil
	case BannerDismissMsg:
		b.visible = false
		return nil
	}
	return nil
}

// StartAutoDismiss returns a command that dismisses the banner after a delay.
func (b *Banner) StartAutoDismiss(delay time.Duration) tea.Cmd {
	return tea.Tick(delay, func(_ time.Time) tea.Msg {
		return BannerDismissMsg{}
	})
}

// View renders the banner.
func (b *Banner) View() string {
	if !b.visible {
		return ""
	}

	t := b.styles.Theme

	// ASCII art logo - matches mocks.html
	// Using block characters that render well in terminals
	logoLines := []string{
		"              ██                                             ▄▄",
		"              ▀▀                                             ██",
		" ██▄  ▄██   ████     ██▄  ▄██   ▄████▄    ▄█████▄   ▄█████▄  ██ ▄██▀    ▄█████▄",
		"  ██  ██      ██      ██  ██   ██▄▄▄▄██  ██▀    ▀   ▀ ▄▄▄██  ██▄██      ▀ ▄▄▄██",
		"  ▀█▄▄█▀      ██      ▀█▄▄█▀   ██▀▀▀▀▀▀  ██        ▄██▀▀▀██  ██▀██▄    ▄██▀▀▀██",
		"   ████    ▄▄▄██▄▄▄    ████    ▀██▄▄▄▄█  ▀██▄▄▄▄█  ██▄▄▄███  ██  ▀█▄   ██▄▄▄███",
		"    ▀▀     ▀▀▀▀▀▀▀▀     ▀▀       ▀▀▀▀▀     ▀▀▀▀▀    ▀▀▀▀ ▀▀  ▀▀   ▀▀▀   ▀▀▀▀ ▀▀",
	}

	// Style the logo with alternating mauve/lavender colors
	mauve := lipgloss.NewStyle().Foreground(t.Primary)
	lavender := lipgloss.NewStyle().Foreground(t.Secondary).Bold(true)

	styledLines := make([]string, len(logoLines))
	for i, line := range logoLines {
		var style lipgloss.Style
		if i%2 == 0 {
			style = mauve
		} else {
			style = lavender
		}
		styledLines[i] = style.Render(line)
	}
	styledLogo := lipgloss.JoinVertical(lipgloss.Left, styledLines...)

	// Sanskrit name
	sanskrit := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F5E0DC")). // rosewater
		Render("विवेचक")

	// Tagline
	tagline := lipgloss.NewStyle().
		Foreground(t.Subtext).
		Render("─── the discerning reviewer ───")

	// Version
	version := lipgloss.NewStyle().
		Foreground(t.Muted).
		Render("v" + b.version + " · github.com/indrasvat/vivecaka")

	// Combine all elements
	content := lipgloss.JoinVertical(
		lipgloss.Center,
		styledLogo,
		"",
		sanskrit,
		"",
		tagline,
		"",
		version,
	)

	// Center in the available space
	// Note: No Background() needed - termenv.SetBackgroundColor() handles the terminal default
	return lipgloss.NewStyle().
		Width(b.width).
		Height(b.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(content)
}
