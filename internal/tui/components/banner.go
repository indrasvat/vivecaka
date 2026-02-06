package components

import (
	"math/rand/v2"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/indrasvat/vivecaka/internal/tui/core"
)

// bannerGlyphs are decorative symbol trios shown under the logo, picked at random per launch.
var bannerGlyphs = [][3]string{
	{"⟁", "⟐", "⌬"},
	{"⌖", "⟡", "⊹"},
	{"⊛", "⟠", "⋈"},
	{"⏣", "◬", "⧉"},
	{"⎔", "⟐", "⟁"},
}

// Banner renders the startup ASCII art banner.
type Banner struct {
	styles     core.Styles
	version    string
	visible    bool
	width      int
	height     int
	glyphIndex int
}

// NewBanner creates a new Banner component.
func NewBanner(styles core.Styles, version string) *Banner {
	return &Banner{
		styles:     styles,
		version:    version,
		visible:    true,
		glyphIndex: rand.IntN(len(bannerGlyphs)),
	}
}

// BannerGlyphTickMsg advances to the next symbol trio.
type BannerGlyphTickMsg struct{}

// SetStyles updates the styles without losing state.
func (b *Banner) SetStyles(s core.Styles) { b.styles = s }

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
	case BannerGlyphTickMsg:
		b.glyphIndex = (b.glyphIndex + 1) % len(bannerGlyphs)
		return b.scheduleGlyphTick()
	}
	return nil
}

// scheduleGlyphTick returns a command that rotates glyphs after an interval.
func (b *Banner) scheduleGlyphTick() tea.Cmd {
	if !b.visible {
		return nil
	}
	return tea.Tick(400*time.Millisecond, func(_ time.Time) tea.Msg {
		return BannerGlyphTickMsg{}
	})
}

// StartAutoDismiss returns a command that dismisses the banner after a delay
// and starts the glyph rotation animation.
func (b *Banner) StartAutoDismiss(delay time.Duration) tea.Cmd {
	return tea.Batch(
		tea.Tick(delay, func(_ time.Time) tea.Msg {
			return BannerDismissMsg{}
		}),
		b.scheduleGlyphTick(),
	)
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

	// Decorative symbol trio (rotates while banner is visible, theme-colored)
	g := bannerGlyphs[b.glyphIndex]
	dot := lipgloss.NewStyle().Foreground(t.Muted).Render("·")
	sp := "  "
	glyphs := lipgloss.NewStyle().Foreground(t.Primary).Render(g[0]) +
		sp + dot + sp +
		lipgloss.NewStyle().Foreground(t.Secondary).Render(g[1]) +
		sp + dot + sp +
		lipgloss.NewStyle().Foreground(t.Primary).Render(g[2])

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
		glyphs,
		tagline,
		"",
		version,
	)

	// Use lipgloss.Place for reliable centering, then enforce exact height
	// with full-width padding lines. Avoids lipgloss.Height() which sets
	// MINIMUM height and can cause rendering artifacts (see CLAUDE.md).
	placed := lipgloss.Place(b.width, b.height, lipgloss.Center, lipgloss.Center, content)
	return bannerExactHeight(placed, b.height, b.width)
}

// bannerExactHeight pads or truncates content to exactly the specified height.
// Each padding line is full-width spaces to properly overwrite previous content.
func bannerExactHeight(content string, height, width int) string {
	lines := strings.Split(content, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	emptyLine := strings.Repeat(" ", width)
	for len(lines) < height {
		lines = append(lines, emptyLine)
	}
	return strings.Join(lines, "\n")
}
