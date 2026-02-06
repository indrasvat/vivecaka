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
	filter        string
	refreshSecs   int
	refreshPaused bool
	branch        string
	width         int
}

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
	repo := repoStyle.Render(h.repo.String())

	// Format count: show "loaded/total" when total is known
	var countText string
	if h.totalCount > 0 {
		countText = fmt.Sprintf("%d/%d open", h.prCount, h.totalCount)
	} else {
		countText = fmt.Sprintf("%d open", h.prCount)
	}
	count := countStyle.Render(countText)

	// Determine filter label
	filterLabel := "All PRs"
	if h.filter != "" && h.filter != "all" && h.filter != "All PRs" {
		filterLabel = h.filter
	}
	filter := filterStyle.Render(filterLabel)

	// Build left side: brand      repo      count      filter
	// Use 6 spaces between each element for visual separation (matches mock's flexbox gaps)
	left := brand + "      " + repo + "      " + count + "      " + filter

	// Build right side: branch + refresh timer
	var rightParts []string
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
		left + lipgloss.NewStyle().Width(gap).Render("") + right,
	)
	return bar
}
