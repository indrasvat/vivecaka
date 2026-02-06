package views

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/indrasvat/vivecaka/internal/domain"
)

func TestNewReviewModel(t *testing.T) {
	m := NewReviewModel(testStyles(), testKeys())
	assert.Equal(t, domain.ReviewActionComment, m.action)
	assert.Nil(t, m.form, "form should be nil before SetPRNumber")
}

func TestReviewSetSize(t *testing.T) {
	m := NewReviewModel(testStyles(), testKeys())
	m.SetSize(80, 24)
	assert.Equal(t, 80, m.width)
	assert.Equal(t, 24, m.height)
}

func TestReviewSetPRNumber(t *testing.T) {
	m := NewReviewModel(testStyles(), testKeys())
	m.SetSize(80, 24)
	m.action = domain.ReviewActionApprove
	m.body = "old body"

	m.SetPRNumber(42)
	assert.Equal(t, 42, m.prNumber)
	assert.Equal(t, domain.ReviewActionComment, m.action, "action should reset to Comment")
	assert.Empty(t, m.body, "body should reset")
	assert.NotNil(t, m.form, "form should be initialized after SetPRNumber")
}

func TestReviewEscapeKey(t *testing.T) {
	m := NewReviewModel(testStyles(), testKeys())
	m.SetSize(80, 24)
	m.SetPRNumber(42)

	// Escape should produce CloseReviewMsg
	esc := tea.KeyMsg{Type: tea.KeyEscape}
	cmd := m.Update(esc)
	require.NotNil(t, cmd, "Escape should produce a command")

	msg := cmd()
	_, ok := msg.(CloseReviewMsg)
	assert.True(t, ok, "expected CloseReviewMsg")
}

func TestReviewInit(t *testing.T) {
	m := NewReviewModel(testStyles(), testKeys())
	m.SetSize(80, 24)

	// Before SetPRNumber, Init should return nil
	cmd := m.Init()
	assert.Nil(t, cmd, "Init should return nil before form is initialized")

	m.SetPRNumber(42)

	// After SetPRNumber, Init should return the form's init command
	cmd = m.Init()
	// The form's Init() returns a command - we just verify it doesn't panic
	// and returns something (huh forms return an init cmd)
	assert.NotNil(t, cmd, "Init should return form's init command after SetPRNumber")
}

func TestReviewView(t *testing.T) {
	m := NewReviewModel(testStyles(), testKeys())
	m.SetSize(80, 24)
	m.SetPRNumber(42)

	view := m.View()
	assert.NotEmpty(t, view, "view should not be empty")
}

func TestReviewViewBeforeInit(t *testing.T) {
	m := NewReviewModel(testStyles(), testKeys())
	m.SetSize(80, 24)

	// Before SetPRNumber, view should show loading
	view := m.View()
	assert.NotEmpty(t, view, "view should not be empty even before form init")
}

func TestReviewNonKeyMsg(t *testing.T) {
	m := NewReviewModel(testStyles(), testKeys())
	m.SetSize(80, 24)

	// Non-key messages before form init should return nil
	cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	assert.Nil(t, cmd, "non-key messages without form should return nil cmd")
}

func TestReviewFormExists(t *testing.T) {
	m := NewReviewModel(testStyles(), testKeys())
	m.SetSize(80, 24)
	m.SetPRNumber(42)

	// Form should be created
	assert.NotNil(t, m.form, "form should be created after SetPRNumber")
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
		got := itoa(tt.n)
		assert.Equal(t, tt.want, got)
	}
}
