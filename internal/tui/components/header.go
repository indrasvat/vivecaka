package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/indrasvat/vivecaka/internal/tui/core"
)

// Header renders the top bar showing repo info and filter state.
type Header struct {
	styles        core.Styles
	repo          domain.RepoRef
	prCount       int
	totalCount    int // total PRs available (0 = unknown)
	countLabel    string
	countOverride string
	filter        string
	refreshSecs   int
	refreshPaused bool
	branch        string
	width         int
}

// SetStyles updates the styles without losing state.
func (h *Header) SetStyles(s core.Styles) { h.styles = s }

// NewHeader creates a new Header component.
func NewHeader(styles core.Styles) *Header {
	return &Header{styles: styles}
}

// SetRepo updates the displayed repository.
func (h *Header) SetRepo(repo domain.RepoRef) { h.repo = repo }

// SetPRCount updates the displayed loaded PR count.
func (h *Header) SetPRCount(n int) { h.prCount = n }

// SetTotalCount updates the displayed total PR count.
func (h *Header) SetTotalCount(n int) { h.totalCount = n }

// SetCountLabel updates the noun used after the displayed count.
func (h *Header) SetCountLabel(label string) { h.countLabel = label }

// SetCountOverride replaces the generated count text when a view needs a
// compact non-count state such as "#17".
func (h *Header) SetCountOverride(text string) { h.countOverride = text }

// SetFilter updates the displayed filter name.
func (h *Header) SetFilter(f string) { h.filter = f }

// SetRefreshSecs updates the refresh countdown display.
func (h *Header) SetRefreshSecs(s int) { h.refreshSecs = s }

// SetRefreshCountdown updates the refresh countdown and pause state.
func (h *Header) SetRefreshCountdown(secs int, paused bool) {
	h.refreshSecs = secs
	h.refreshPaused = paused
}

// SetBranch updates the displayed branch name.
func (h *Header) SetBranch(branch string) { h.branch = branch }

// SetWidth updates the header width for responsive layout.
func (h *Header) SetWidth(w int) { h.width = w }

// View renders the header bar.
func (h *Header) View() string {
	t := h.styles.Theme

	// Brand name in mauve, bold
	brandStyle := lipgloss.NewStyle().Foreground(t.Primary).Bold(true)
	// Repo name in teal/secondary
	repoStyle := lipgloss.NewStyle().Foreground(t.Secondary)
	// PR count in subtext
	countStyle := lipgloss.NewStyle().Foreground(t.Subtext)
	// Filter in info/blue
	filterStyle := lipgloss.NewStyle().Foreground(t.Info)
	// Refresh timer in muted
	refreshStyle := lipgloss.NewStyle().Foreground(t.Muted)

	brand := brandStyle.Render(" vivecaka")
	var left string
	if h.repo.Owner != "" || h.repo.Name != "" {
		repo := repoStyle.Render(h.repo.String())
		left = brand + "      " + repo
	} else {
		left = brand
	}

	// Format count: show "loaded/total" when total is known
	countLabel := h.countLabel
	if countLabel == "" {
		countLabel = "open"
	}
	var countText string
	switch {
	case h.countOverride != "":
		countText = h.countOverride
	case h.totalCount > 0:
		countText = fmt.Sprintf("%d/%d %s", h.prCount, h.totalCount, countLabel)
	default:
		countText = fmt.Sprintf("%d %s", h.prCount, countLabel)
	}
	count := countStyle.Render(countText)

	// Determine filter label
	filterLabel := "All PRs"
	if h.filter != "" && h.filter != "all" && h.filter != "All PRs" {
		filterLabel = h.filter
	}
	filter := filterStyle.Render(filterLabel)

	var rightParts []string
	rightParts = append(rightParts, count, filter)
	if h.branch != "" {
		branchStyle := lipgloss.NewStyle().Foreground(t.Info)
		rightParts = append(rightParts, branchStyle.Render("⎇ "+h.branch))
	}
	if h.refreshPaused {
		rightParts = append(rightParts, refreshStyle.Render("⏸ paused"))
	} else if h.refreshSecs > 0 {
		rightParts = append(rightParts, refreshStyle.Render(fmt.Sprintf("↻ %ds", h.refreshSecs)))
	}
	right := strings.Join(rightParts, "  ")

	// Pad between left and right
	gap := max(1, h.width-lipgloss.Width(left)-lipgloss.Width(right))

	// Use inline to prevent any background styling issues
	bar := lipgloss.NewStyle().Width(h.width).Render(
		left + strings.Repeat(" ", gap) + right,
	)
	return bar
}
