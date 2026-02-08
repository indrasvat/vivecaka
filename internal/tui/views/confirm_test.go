package views

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfirmModel_ShowAndActive(t *testing.T) {
	m := NewConfirmModel(testStyles())
	assert.False(t, m.Active(), "newly created ConfirmModel should not be active")

	m.Show("Test Title", "Are you sure?", CheckoutPRMsg{Number: 1, Branch: "main"})
	assert.True(t, m.Active(), "ConfirmModel should be active after Show()")
}

func TestConfirmModel_ConfirmEnter(t *testing.T) {
	m := NewConfirmModel(testStyles())
	m.SetSize(80, 24)

	origMsg := CheckoutPRMsg{Number: 42, Branch: "feat/test"}
	m.Show("Checkout", "Check out?", origMsg)

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd, "Enter should produce a command")

	msg := cmd()
	result, ok := msg.(ConfirmResultMsg)
	require.True(t, ok, "expected ConfirmResultMsg, got %T", msg)
	assert.True(t, result.Confirmed, "Enter should set Confirmed=true")
	checkout, ok := result.Action.(CheckoutPRMsg)
	require.True(t, ok, "expected CheckoutPRMsg action, got %T", result.Action)
	assert.Equal(t, 42, checkout.Number)
	assert.Equal(t, "feat/test", checkout.Branch)
}

func TestConfirmModel_ConfirmY(t *testing.T) {
	m := NewConfirmModel(testStyles())
	m.SetSize(80, 24)
	m.Show("Checkout", "Check out?", CheckoutPRMsg{Number: 1, Branch: "main"})

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	require.NotNil(t, cmd, "'y' should produce a command")

	msg := cmd()
	result, ok := msg.(ConfirmResultMsg)
	require.True(t, ok, "expected ConfirmResultMsg, got %T", msg)
	assert.True(t, result.Confirmed, "'y' should set Confirmed=true")
}

func TestConfirmModel_ConfirmUpperY(t *testing.T) {
	m := NewConfirmModel(testStyles())
	m.SetSize(80, 24)
	m.Show("Checkout", "Check out?", CheckoutPRMsg{Number: 1, Branch: "main"})

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}})
	require.NotNil(t, cmd, "'Y' should produce a command")

	msg := cmd()
	result, ok := msg.(ConfirmResultMsg)
	require.True(t, ok, "expected ConfirmResultMsg, got %T", msg)
	assert.True(t, result.Confirmed, "'Y' should set Confirmed=true")
}

func TestConfirmModel_CancelEsc(t *testing.T) {
	m := NewConfirmModel(testStyles())
	m.SetSize(80, 24)
	m.Show("Checkout", "Check out?", CheckoutPRMsg{Number: 1, Branch: "main"})

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	require.NotNil(t, cmd, "Esc should produce a command")

	msg := cmd()
	_, ok := msg.(CloseConfirmMsg)
	assert.True(t, ok, "expected CloseConfirmMsg, got %T", msg)
	assert.False(t, m.Active(), "ConfirmModel should be inactive after cancel")
}

func TestConfirmModel_CancelN(t *testing.T) {
	m := NewConfirmModel(testStyles())
	m.SetSize(80, 24)
	m.Show("Checkout", "Check out?", CheckoutPRMsg{Number: 1, Branch: "main"})

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	require.NotNil(t, cmd, "'n' should produce a command")

	msg := cmd()
	_, ok := msg.(CloseConfirmMsg)
	assert.True(t, ok, "expected CloseConfirmMsg, got %T", msg)
}

func TestConfirmModel_CancelUpperN(t *testing.T) {
	m := NewConfirmModel(testStyles())
	m.SetSize(80, 24)
	m.Show("Checkout", "Check out?", CheckoutPRMsg{Number: 1, Branch: "main"})

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}})
	require.NotNil(t, cmd, "'N' should produce a command")

	msg := cmd()
	_, ok := msg.(CloseConfirmMsg)
	assert.True(t, ok, "expected CloseConfirmMsg, got %T", msg)
}

func TestConfirmModel_IgnoreOtherKeys(t *testing.T) {
	m := NewConfirmModel(testStyles())
	m.SetSize(80, 24)
	m.Show("Checkout", "Check out?", CheckoutPRMsg{Number: 1, Branch: "main"})

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	assert.Nil(t, cmd, "unrecognized key should not produce a command")
	assert.True(t, m.Active(), "ConfirmModel should remain active on unrecognized key")
}

func TestConfirmModel_IgnoreNonKeyMsg(t *testing.T) {
	m := NewConfirmModel(testStyles())
	m.SetSize(80, 24)
	m.Show("Checkout", "Check out?", CheckoutPRMsg{Number: 1, Branch: "main"})

	cmd := m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	assert.Nil(t, cmd, "non-key messages should not produce a command")
}

func TestConfirmModel_ViewPrompt(t *testing.T) {
	m := NewConfirmModel(testStyles())
	m.SetSize(80, 24)
	m.Show("Checkout Branch", "Check out branch \"feat/test\" for PR #42?", CheckoutPRMsg{Number: 42, Branch: "feat/test"})

	view := m.View()
	assert.NotEmpty(t, view, "View() should not be empty")
	assert.Contains(t, view, "Checkout Branch")
	assert.Contains(t, view, "feat/test")
	assert.Contains(t, view, "#42")
}

func TestConfirmModel_Loading(t *testing.T) {
	m := NewConfirmModel(testStyles())
	m.SetSize(80, 24)

	cmd := m.ShowLoading("Checkout Branch", "Checking out \"feat/test\"...")
	require.NotNil(t, cmd, "ShowLoading should return spinner tick cmd")
	assert.True(t, m.IsLoading(), "should be in loading state")
	assert.True(t, m.Active(), "should be active during loading")

	view := m.View()
	assert.Contains(t, view, "Checking out")

	// Keys should be ignored during loading
	cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	assert.Nil(t, cmd, "keys should be ignored during loading")
}

func TestConfirmModel_ResultSuccess(t *testing.T) {
	m := NewConfirmModel(testStyles())
	m.SetSize(80, 24)

	m.ShowResult("Checkout Complete", "Checked out branch: feat/test", true)

	view := m.View()
	assert.Contains(t, view, "Checkout Complete")
	assert.Contains(t, view, "feat/test")
	assert.Contains(t, view, "Press any key")
	assert.Contains(t, view, "✓")

	// Any key should dismiss
	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	require.NotNil(t, cmd, "any key should produce close cmd in result state")
	msg := cmd()
	_, ok := msg.(CloseConfirmMsg)
	assert.True(t, ok, "expected CloseConfirmMsg, got %T", msg)
}

func TestConfirmModel_ResultError(t *testing.T) {
	m := NewConfirmModel(testStyles())
	m.SetSize(80, 24)

	m.ShowResult("Checkout Failed", "could not checkout", false)

	view := m.View()
	assert.Contains(t, view, "Checkout Failed")
	assert.Contains(t, view, "✗")
}

func TestConfirmModel_StateHints(t *testing.T) {
	m := NewConfirmModel(testStyles())

	m.Show("Test", "msg", nil)
	assert.Contains(t, m.ConfirmStateHint(), "confirm")

	m.ShowLoading("Test", "loading...")
	assert.Contains(t, m.ConfirmStateHint(), "Checking")

	m.ShowResult("Done", "ok", true)
	assert.Contains(t, m.ConfirmStateHint(), "any key")
}
