package views

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestConfirmModel_ShowAndActive(t *testing.T) {
	m := NewConfirmModel(testStyles())
	if m.Active() {
		t.Error("newly created ConfirmModel should not be active")
	}

	m.Show("Test Title", "Are you sure?", CheckoutPRMsg{Number: 1, Branch: "main"})
	if !m.Active() {
		t.Error("ConfirmModel should be active after Show()")
	}
}

func TestConfirmModel_ConfirmEnter(t *testing.T) {
	m := NewConfirmModel(testStyles())
	m.SetSize(80, 24)

	origMsg := CheckoutPRMsg{Number: 42, Branch: "feat/test"}
	m.Show("Checkout", "Check out?", origMsg)

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter should produce a command")
	}

	msg := cmd()
	result, ok := msg.(ConfirmResultMsg)
	if !ok {
		t.Fatalf("expected ConfirmResultMsg, got %T", msg)
	}
	if !result.Confirmed {
		t.Error("Enter should set Confirmed=true")
	}
	checkout, ok := result.Action.(CheckoutPRMsg)
	if !ok {
		t.Fatalf("expected CheckoutPRMsg action, got %T", result.Action)
	}
	if checkout.Number != 42 {
		t.Errorf("Action.Number = %d, want 42", checkout.Number)
	}
	if checkout.Branch != "feat/test" {
		t.Errorf("Action.Branch = %q, want %q", checkout.Branch, "feat/test")
	}
}

func TestConfirmModel_ConfirmY(t *testing.T) {
	m := NewConfirmModel(testStyles())
	m.SetSize(80, 24)
	m.Show("Checkout", "Check out?", CheckoutPRMsg{Number: 1, Branch: "main"})

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatal("'y' should produce a command")
	}

	msg := cmd()
	result, ok := msg.(ConfirmResultMsg)
	if !ok {
		t.Fatalf("expected ConfirmResultMsg, got %T", msg)
	}
	if !result.Confirmed {
		t.Error("'y' should set Confirmed=true")
	}
}

func TestConfirmModel_ConfirmUpperY(t *testing.T) {
	m := NewConfirmModel(testStyles())
	m.SetSize(80, 24)
	m.Show("Checkout", "Check out?", CheckoutPRMsg{Number: 1, Branch: "main"})

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}})
	if cmd == nil {
		t.Fatal("'Y' should produce a command")
	}

	msg := cmd()
	result, ok := msg.(ConfirmResultMsg)
	if !ok {
		t.Fatalf("expected ConfirmResultMsg, got %T", msg)
	}
	if !result.Confirmed {
		t.Error("'Y' should set Confirmed=true")
	}
}

func TestConfirmModel_CancelEsc(t *testing.T) {
	m := NewConfirmModel(testStyles())
	m.SetSize(80, 24)
	m.Show("Checkout", "Check out?", CheckoutPRMsg{Number: 1, Branch: "main"})

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("Esc should produce a command")
	}

	msg := cmd()
	if _, ok := msg.(CloseConfirmMsg); !ok {
		t.Fatalf("expected CloseConfirmMsg, got %T", msg)
	}
	if m.Active() {
		t.Error("ConfirmModel should be inactive after cancel")
	}
}

func TestConfirmModel_CancelN(t *testing.T) {
	m := NewConfirmModel(testStyles())
	m.SetSize(80, 24)
	m.Show("Checkout", "Check out?", CheckoutPRMsg{Number: 1, Branch: "main"})

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if cmd == nil {
		t.Fatal("'n' should produce a command")
	}

	msg := cmd()
	if _, ok := msg.(CloseConfirmMsg); !ok {
		t.Fatalf("expected CloseConfirmMsg, got %T", msg)
	}
}

func TestConfirmModel_CancelUpperN(t *testing.T) {
	m := NewConfirmModel(testStyles())
	m.SetSize(80, 24)
	m.Show("Checkout", "Check out?", CheckoutPRMsg{Number: 1, Branch: "main"})

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}})
	if cmd == nil {
		t.Fatal("'N' should produce a command")
	}

	msg := cmd()
	if _, ok := msg.(CloseConfirmMsg); !ok {
		t.Fatalf("expected CloseConfirmMsg, got %T", msg)
	}
}

func TestConfirmModel_IgnoreOtherKeys(t *testing.T) {
	m := NewConfirmModel(testStyles())
	m.SetSize(80, 24)
	m.Show("Checkout", "Check out?", CheckoutPRMsg{Number: 1, Branch: "main"})

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if cmd != nil {
		t.Error("unrecognized key should not produce a command")
	}
	if !m.Active() {
		t.Error("ConfirmModel should remain active on unrecognized key")
	}
}

func TestConfirmModel_IgnoreNonKeyMsg(t *testing.T) {
	m := NewConfirmModel(testStyles())
	m.SetSize(80, 24)
	m.Show("Checkout", "Check out?", CheckoutPRMsg{Number: 1, Branch: "main"})

	cmd := m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	if cmd != nil {
		t.Error("non-key messages should not produce a command")
	}
}

func TestConfirmModel_ViewPrompt(t *testing.T) {
	m := NewConfirmModel(testStyles())
	m.SetSize(80, 24)
	m.Show("Checkout Branch", "Check out branch \"feat/test\" for PR #42?", CheckoutPRMsg{Number: 42, Branch: "feat/test"})

	view := m.View()
	if view == "" {
		t.Error("View() should not be empty")
	}
	if !strings.Contains(view, "Checkout Branch") {
		t.Error("View should contain the title")
	}
	if !strings.Contains(view, "feat/test") {
		t.Error("View should contain the branch name")
	}
	if !strings.Contains(view, "#42") {
		t.Error("View should contain the PR number")
	}
}

func TestConfirmModel_Loading(t *testing.T) {
	m := NewConfirmModel(testStyles())
	m.SetSize(80, 24)

	cmd := m.ShowLoading("Checkout Branch", "Checking out \"feat/test\"...")
	if cmd == nil {
		t.Fatal("ShowLoading should return spinner tick cmd")
	}
	if !m.IsLoading() {
		t.Error("should be in loading state")
	}
	if !m.Active() {
		t.Error("should be active during loading")
	}

	view := m.View()
	if !strings.Contains(view, "Checking out") {
		t.Error("loading view should contain the message")
	}

	// Keys should be ignored during loading
	cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd != nil {
		t.Error("keys should be ignored during loading")
	}
}

func TestConfirmModel_ResultSuccess(t *testing.T) {
	m := NewConfirmModel(testStyles())
	m.SetSize(80, 24)

	m.ShowResult("Checkout Complete", "Checked out branch: feat/test", true)

	view := m.View()
	if !strings.Contains(view, "Checkout Complete") {
		t.Error("result view should contain title")
	}
	if !strings.Contains(view, "feat/test") {
		t.Error("result view should contain branch name")
	}
	if !strings.Contains(view, "Press any key") {
		t.Error("result view should contain dismiss hint")
	}
	if !strings.Contains(view, "✓") {
		t.Error("success result should contain checkmark")
	}

	// Any key should dismiss
	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if cmd == nil {
		t.Fatal("any key should produce close cmd in result state")
	}
	msg := cmd()
	if _, ok := msg.(CloseConfirmMsg); !ok {
		t.Fatalf("expected CloseConfirmMsg, got %T", msg)
	}
}

func TestConfirmModel_ResultError(t *testing.T) {
	m := NewConfirmModel(testStyles())
	m.SetSize(80, 24)

	m.ShowResult("Checkout Failed", "could not checkout", false)

	view := m.View()
	if !strings.Contains(view, "Checkout Failed") {
		t.Error("error result should contain title")
	}
	if !strings.Contains(view, "✗") {
		t.Error("error result should contain X mark")
	}
}

func TestConfirmModel_StateHints(t *testing.T) {
	m := NewConfirmModel(testStyles())

	m.Show("Test", "msg", nil)
	if h := m.ConfirmStateHint(); !strings.Contains(h, "confirm") {
		t.Errorf("prompt hint should contain 'confirm', got %q", h)
	}

	m.ShowLoading("Test", "loading...")
	if h := m.ConfirmStateHint(); !strings.Contains(h, "Checking") {
		t.Errorf("loading hint should contain 'Checking', got %q", h)
	}

	m.ShowResult("Done", "ok", true)
	if h := m.ConfirmStateHint(); !strings.Contains(h, "any key") {
		t.Errorf("result hint should contain 'any key', got %q", h)
	}
}
