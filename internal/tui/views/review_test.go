package views

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/indrasvat/vivecaka/internal/domain"
)

func TestNewReviewModel(t *testing.T) {
	m := NewReviewModel(testStyles(), testKeys())
	if m.action != domain.ReviewActionComment {
		t.Errorf("default action = %q, want %q", m.action, domain.ReviewActionComment)
	}
	if m.cursor != 0 {
		t.Errorf("default cursor = %d, want 0", m.cursor)
	}
	if m.editing {
		t.Error("should not be editing initially")
	}
}

func TestReviewSetSize(t *testing.T) {
	m := NewReviewModel(testStyles(), testKeys())
	m.SetSize(80, 24)
	if m.width != 80 || m.height != 24 {
		t.Errorf("size = %dx%d, want 80x24", m.width, m.height)
	}
}

func TestReviewSetPRNumber(t *testing.T) {
	m := NewReviewModel(testStyles(), testKeys())
	m.action = domain.ReviewActionApprove
	m.body = "old body"
	m.cursor = 2

	m.SetPRNumber(42)
	if m.prNumber != 42 {
		t.Errorf("prNumber = %d, want 42", m.prNumber)
	}
	if m.action != domain.ReviewActionComment {
		t.Errorf("action should reset to Comment, got %q", m.action)
	}
	if m.body != "" {
		t.Errorf("body should reset, got %q", m.body)
	}
	if m.cursor != 0 {
		t.Errorf("cursor should reset, got %d", m.cursor)
	}
}

func TestReviewNavigationDown(t *testing.T) {
	m := NewReviewModel(testStyles(), testKeys())
	m.SetSize(80, 24)

	down := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	m.Update(down)
	if m.cursor != 1 {
		t.Errorf("cursor after j = %d, want 1", m.cursor)
	}

	m.Update(down)
	if m.cursor != 2 {
		t.Errorf("cursor after 2nd j = %d, want 2", m.cursor)
	}

	// Can't go past 2.
	m.Update(down)
	if m.cursor != 2 {
		t.Errorf("cursor should cap at 2, got %d", m.cursor)
	}
}

func TestReviewNavigationUp(t *testing.T) {
	m := NewReviewModel(testStyles(), testKeys())
	m.SetSize(80, 24)
	m.cursor = 2

	up := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	m.Update(up)
	if m.cursor != 1 {
		t.Errorf("cursor after k = %d, want 1", m.cursor)
	}

	m.Update(up)
	if m.cursor != 0 {
		t.Errorf("cursor after 2nd k = %d, want 0", m.cursor)
	}

	// Can't go below 0.
	m.Update(up)
	if m.cursor != 0 {
		t.Errorf("cursor should stay at 0, got %d", m.cursor)
	}
}

func TestReviewCycleAction(t *testing.T) {
	m := NewReviewModel(testStyles(), testKeys())
	m.SetSize(80, 24)
	m.cursor = 0

	enter := tea.KeyMsg{Type: tea.KeyEnter}

	// Comment → Approve.
	m.Update(enter)
	if m.action != domain.ReviewActionApprove {
		t.Errorf("after 1st cycle = %q, want Approve", m.action)
	}

	// Approve → RequestChanges.
	m.Update(enter)
	if m.action != domain.ReviewActionRequestChanges {
		t.Errorf("after 2nd cycle = %q, want RequestChanges", m.action)
	}

	// RequestChanges → Comment.
	m.Update(enter)
	if m.action != domain.ReviewActionComment {
		t.Errorf("after 3rd cycle = %q, want Comment", m.action)
	}
}

func TestReviewEditBody(t *testing.T) {
	m := NewReviewModel(testStyles(), testKeys())
	m.SetSize(80, 24)

	// Navigate to body field.
	down := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	m.Update(down)
	if m.cursor != 1 {
		t.Fatal("expected cursor at body field")
	}

	// Enter editing mode.
	enter := tea.KeyMsg{Type: tea.KeyEnter}
	m.Update(enter)
	if !m.editing {
		t.Fatal("should be in editing mode")
	}

	// Type text.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	if m.body != "Hi" {
		t.Errorf("body = %q, want %q", m.body, "Hi")
	}

	// Newline.
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.body != "Hi\n" {
		t.Errorf("body after enter = %q, want %q", m.body, "Hi\n")
	}

	// Backspace.
	m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.body != "Hi" {
		t.Errorf("body after backspace = %q, want %q", m.body, "Hi")
	}

	// Escape exits editing.
	m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if m.editing {
		t.Error("should exit editing on Escape")
	}
}

func TestReviewEditBackspaceEmpty(t *testing.T) {
	m := NewReviewModel(testStyles(), testKeys())
	m.SetSize(80, 24)
	m.cursor = 1
	m.editing = true

	// Backspace on empty body is safe.
	m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.body != "" {
		t.Errorf("body should stay empty, got %q", m.body)
	}
}

func TestReviewSubmit(t *testing.T) {
	m := NewReviewModel(testStyles(), testKeys())
	m.SetSize(80, 24)
	m.SetPRNumber(42)
	m.action = domain.ReviewActionApprove
	m.body = "LGTM"
	m.cursor = 2

	enter := tea.KeyMsg{Type: tea.KeyEnter}
	cmd := m.Update(enter)
	if cmd == nil {
		t.Fatal("Submit should produce a command")
	}

	msg := cmd()
	sub, ok := msg.(SubmitReviewMsg)
	if !ok {
		t.Fatalf("expected SubmitReviewMsg, got %T", msg)
	}
	if sub.Number != 42 {
		t.Errorf("Number = %d, want 42", sub.Number)
	}
	if sub.Review.Action != domain.ReviewActionApprove {
		t.Errorf("Action = %q, want Approve", sub.Review.Action)
	}
	if sub.Review.Body != "LGTM" {
		t.Errorf("Body = %q, want %q", sub.Review.Body, "LGTM")
	}
}

func TestReviewViewCursor0(t *testing.T) {
	m := NewReviewModel(testStyles(), testKeys())
	m.SetSize(80, 24)
	m.SetPRNumber(42)
	m.cursor = 0

	view := m.View()
	if view == "" {
		t.Error("view should not be empty")
	}
}

func TestReviewViewCursor1(t *testing.T) {
	m := NewReviewModel(testStyles(), testKeys())
	m.SetSize(80, 24)
	m.cursor = 1

	view := m.View()
	if view == "" {
		t.Error("view with cursor on body should not be empty")
	}
}

func TestReviewViewEditing(t *testing.T) {
	m := NewReviewModel(testStyles(), testKeys())
	m.SetSize(80, 24)
	m.cursor = 1
	m.editing = true
	m.body = "typing..."

	view := m.View()
	if view == "" {
		t.Error("editing view should not be empty")
	}
}

func TestReviewViewCursor2(t *testing.T) {
	m := NewReviewModel(testStyles(), testKeys())
	m.SetSize(80, 24)
	m.cursor = 2

	view := m.View()
	if view == "" {
		t.Error("view with cursor on submit should not be empty")
	}
}

func TestReviewViewEmptyBody(t *testing.T) {
	m := NewReviewModel(testStyles(), testKeys())
	m.SetSize(80, 24)
	m.body = ""

	view := m.View()
	if view == "" {
		t.Error("view with empty body should not be empty")
	}
}

func TestActionDisplay(t *testing.T) {
	tests := []struct {
		action domain.ReviewAction
		want   string
	}{
		{domain.ReviewActionApprove, "Approve ✓"},
		{domain.ReviewActionRequestChanges, "Request Changes !"},
		{domain.ReviewActionComment, "Comment"},
	}
	for _, tt := range tests {
		if got := actionDisplay(tt.action); got != tt.want {
			t.Errorf("actionDisplay(%q) = %q, want %q", tt.action, got, tt.want)
		}
	}
}

func TestReviewNonKeyMsg(t *testing.T) {
	m := NewReviewModel(testStyles(), testKeys())
	m.SetSize(80, 24)

	// Non-key messages should be handled gracefully.
	cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if cmd != nil {
		t.Error("non-key messages should return nil cmd")
	}
}
