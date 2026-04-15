package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/indrasvat/vivecaka/internal/reviewprogress"
	"github.com/indrasvat/vivecaka/internal/tui/core"
)

func reviewFileMarker(file reviewprogress.File) string {
	switch {
	case file.Viewed:
		return "✓"
	case file.Actionable:
		return "●"
	default:
		return "◌"
	}
}

func reviewFileStateText(file reviewprogress.File) string {
	parts := make([]string, 0, 2)
	switch {
	case file.ChangedSinceReview:
		parts = append(parts, "changed since review")
	case file.ChangedSinceVisit:
		parts = append(parts, "changed since visit")
	}

	if file.Viewed {
		parts = append(parts, "viewed")
	}

	if len(parts) == 0 {
		return "unviewed"
	}
	return strings.Join(parts, " · ")
}

func reviewFileMeta(file reviewprogress.File, theme core.Theme) string {
	parts := make([]string, 0, 2)
	switch {
	case file.ChangedSinceReview:
		parts = append(parts, lipgloss.NewStyle().Foreground(theme.Warning).Render("changed since review"))
	case file.ChangedSinceVisit:
		parts = append(parts, lipgloss.NewStyle().Foreground(theme.Info).Render("changed since visit"))
	}

	if file.Viewed {
		parts = append(parts, lipgloss.NewStyle().Foreground(theme.Success).Render("viewed"))
	}

	if len(parts) == 0 {
		return lipgloss.NewStyle().Foreground(theme.Muted).Render("unviewed")
	}
	return strings.Join(parts, lipgloss.NewStyle().Foreground(theme.Muted).Render(" · "))
}
