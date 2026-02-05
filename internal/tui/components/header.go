package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/indrasvat/vivecaka/internal/tui/core"
)

// Header renders the top bar showing repo info and filter state.
type Header struct {
	styles      core.Styles
	repo        domain.RepoRef
	prCount     int
	filter      string
	refreshSecs int
	width       int
}

// NewHeader creates a new Header component.
func NewHeader(styles core.Styles) *Header {
	return &Header{styles: styles}
}

// SetRepo updates the displayed repository.
func (h *Header) SetRepo(repo domain.RepoRef) { h.repo = repo }

// SetPRCount updates the displayed PR count.
func (h *Header) SetPRCount(n int) { h.prCount = n }

// SetFilter updates the displayed filter name.
func (h *Header) SetFilter(f string) { h.filter = f }

// SetRefreshSecs updates the refresh countdown display.
func (h *Header) SetRefreshSecs(s int) { h.refreshSecs = s }

// SetWidth updates the header width for responsive layout.
func (h *Header) SetWidth(w int) { h.width = w }

// View renders the header bar.
func (h *Header) View() string {
	t := h.styles.Theme

	repoStyle := lipgloss.NewStyle().Foreground(t.Primary).Bold(true)
	countStyle := lipgloss.NewStyle().Foreground(t.Success)
	filterStyle := lipgloss.NewStyle().Foreground(t.Info)
	refreshStyle := lipgloss.NewStyle().Foreground(t.Muted)

	repo := repoStyle.Render("◉ " + h.repo.String())
	count := countStyle.Render(fmt.Sprintf("%d open", h.prCount))

	left := repo + "  " + count

	if h.filter != "" && h.filter != "all" {
		left += "  " + filterStyle.Render("⊘ "+h.filter)
	}

	right := ""
	if h.refreshSecs > 0 {
		right = refreshStyle.Render(fmt.Sprintf("↻ %ds", h.refreshSecs))
	}

	// Pad between left and right.
	gap := h.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}

	bar := h.styles.Header.Width(h.width).Render(
		left + lipgloss.NewStyle().Width(gap).Render("") + right,
	)
	return bar
}
