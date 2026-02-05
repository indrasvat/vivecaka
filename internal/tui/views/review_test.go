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
	if m.form != nil {
		t.Error("form should be nil before SetPRNumber")
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
	m.SetSize(80, 24)
	m.action = domain.ReviewActionApprove
	m.body = "old body"

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
	if m.form == nil {
		t.Error("form should be initialized after SetPRNumber")
	}
}

func TestReviewEscapeKey(t *testing.T) {
	m := NewReviewModel(testStyles(), testKeys())
	m.SetSize(80, 24)
	m.SetPRNumber(42)

	// Escape should produce CloseReviewMsg
	esc := tea.KeyMsg{Type: tea.KeyEscape}
	cmd := m.Update(esc)
	if cmd == nil {
		t.Fatal("Escape should produce a command")
	}

	msg := cmd()
	if _, ok := msg.(CloseReviewMsg); !ok {
		t.Fatalf("expected CloseReviewMsg, got %T", msg)
	}
}

func TestReviewInit(t *testing.T) {
	m := NewReviewModel(testStyles(), testKeys())
	m.SetSize(80, 24)

	// Before SetPRNumber, Init should return nil
	cmd := m.Init()
	if cmd != nil {
		t.Error("Init should return nil before form is initialized")
	}

	m.SetPRNumber(42)

	// After SetPRNumber, Init should return the form's init command
	cmd = m.Init()
	// The form's Init() returns a command - we just verify it doesn't panic
	// and returns something (huh forms return an init cmd)
	if cmd == nil {
		t.Error("Init should return form's init command after SetPRNumber")
	}
}

func TestReviewView(t *testing.T) {
	m := NewReviewModel(testStyles(), testKeys())
	m.SetSize(80, 24)
	m.SetPRNumber(42)

	view := m.View()
	if view == "" {
		t.Error("view should not be empty")
	}
}

func TestReviewViewBeforeInit(t *testing.T) {
	m := NewReviewModel(testStyles(), testKeys())
	m.SetSize(80, 24)

	// Before SetPRNumber, view should show loading
	view := m.View()
	if view == "" {
		t.Error("view should not be empty even before form init")
	}
}

func TestReviewNonKeyMsg(t *testing.T) {
	m := NewReviewModel(testStyles(), testKeys())
	m.SetSize(80, 24)

	// Non-key messages before form init should return nil
	cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if cmd != nil {
		t.Error("non-key messages without form should return nil cmd")
	}
}

func TestReviewFormExists(t *testing.T) {
	m := NewReviewModel(testStyles(), testKeys())
	m.SetSize(80, 24)
	m.SetPRNumber(42)

	// Form should be created
	if m.form == nil {
		t.Fatal("form should be created after SetPRNumber")
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "0"},
		{1, "1"},
		{42, "42"},
		{123, "123"},
		{-1, "-1"},
		{-42, "-42"},
	}
	for _, tt := range tests {
		if got := itoa(tt.n); got != tt.want {
			t.Errorf("itoa(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}
