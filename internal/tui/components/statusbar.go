package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/indrasvat/vivecaka/internal/tui/core"
)

// StatusBar renders the bottom bar with key hints and notifications.
type StatusBar struct {
	styles core.Styles
	hints  []string
	msg    string
	msgErr bool
	width  int
}

// SetStyles updates the styles without losing state.
func (s *StatusBar) SetStyles(st core.Styles) { s.styles = st }

// NewStatusBar creates a new StatusBar component.
func NewStatusBar(styles core.Styles) *StatusBar {
	return &StatusBar{styles: styles}
}

// SetHints updates the key hints.
func (s *StatusBar) SetHints(hints []string) { s.hints = hints }

// SetMessage sets a transient message (error or success).
func (s *StatusBar) SetMessage(msg string, isError bool) {
	s.msg = msg
	s.msgErr = isError
}

// ClearMessage clears the transient message.
func (s *StatusBar) ClearMessage() { s.msg = "" }

// SetWidth updates the status bar width.
func (s *StatusBar) SetWidth(w int) { s.width = w }

// View renders the status bar.
func (sb *StatusBar) View() string {
	t := sb.styles.Theme

	var left string
	if sb.msg != "" {
		style := lipgloss.NewStyle().Foreground(t.Success)
		if sb.msgErr {
			style = lipgloss.NewStyle().Foreground(t.Error)
		}
		left = style.Render(sb.msg)
	} else {
		hintStyle := lipgloss.NewStyle().Foreground(t.Muted)
		parts := make([]string, len(sb.hints))
		for i, h := range sb.hints {
			parts[i] = hintStyle.Render(h)
		}
		left = strings.Join(parts, "  ")
	}

	return sb.styles.StatusBar.Width(sb.width).Render(left)
}
