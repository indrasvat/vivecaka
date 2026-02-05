package components

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/indrasvat/vivecaka/internal/tui/core"
)

// Toast represents a single notification.
type Toast struct {
	Message  string
	Level    domain.ToastLevel
	Duration time.Duration
	id       int
}

// ToastManager manages a stack of toast notifications.
type ToastManager struct {
	styles core.Styles
	toasts []Toast
	nextID int
	width  int
}

// NewToastManager creates a new ToastManager.
func NewToastManager(styles core.Styles) *ToastManager {
	return &ToastManager{styles: styles}
}

// SetWidth updates the toast rendering width.
func (tm *ToastManager) SetWidth(w int) { tm.width = w }

// Add creates a new toast and returns a command to auto-dismiss it.
func (tm *ToastManager) Add(msg string, level domain.ToastLevel, duration time.Duration) tea.Cmd {
	id := tm.nextID
	tm.nextID++
	tm.toasts = append(tm.toasts, Toast{
		Message:  msg,
		Level:    level,
		Duration: duration,
		id:       id,
	})
	return tea.Tick(duration, func(_ time.Time) tea.Msg {
		return DismissToastMsg{ID: id}
	})
}

// DismissToastMsg is sent when a toast's duration expires.
type DismissToastMsg struct {
	ID int
}

// Update processes toast dismiss messages.
func (tm *ToastManager) Update(msg tea.Msg) tea.Cmd {
	if m, ok := msg.(DismissToastMsg); ok {
		for i, t := range tm.toasts {
			if t.id == m.ID {
				tm.toasts = append(tm.toasts[:i], tm.toasts[i+1:]...)
				break
			}
		}
	}
	return nil
}

// HasToasts returns true if there are active toasts.
func (tm *ToastManager) HasToasts() bool {
	return len(tm.toasts) > 0
}

// View renders all active toasts stacked in the top-right corner.
func (tm *ToastManager) View() string {
	if len(tm.toasts) == 0 {
		return ""
	}

	var rendered []string
	for _, t := range tm.toasts {
		style := tm.toastStyle(t.Level)
		rendered = append(rendered, style.Render(tm.icon(t.Level)+" "+t.Message))
	}

	return lipgloss.JoinVertical(lipgloss.Right, rendered...)
}

func (tm *ToastManager) toastStyle(level domain.ToastLevel) lipgloss.Style {
	theme := tm.styles.Theme
	base := lipgloss.NewStyle().
		Padding(0, 1).
		Bold(true)

	switch level {
	case domain.ToastSuccess:
		return base.Foreground(theme.Success)
	case domain.ToastError:
		return base.Foreground(theme.Error)
	case domain.ToastWarning:
		return base.Foreground(theme.Warning)
	default:
		return base.Foreground(theme.Info)
	}
}

func (tm *ToastManager) icon(level domain.ToastLevel) string {
	switch level {
	case domain.ToastSuccess:
		return "✓"
	case domain.ToastError:
		return "✗"
	case domain.ToastWarning:
		return "△"
	default:
		return "●"
	}
}
