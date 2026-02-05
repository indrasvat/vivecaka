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
	if m.Active() {
		t.Error("ConfirmModel should be inactive after confirmation")
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

func TestConfirmModel_View(t *testing.T) {
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

func TestStatusHintsConfirm(t *testing.T) {
	hints := StatusHintsConfirm()
	if !strings.Contains(hints, "confirm") {
		t.Error("hints should contain 'confirm'")
	}
	if !strings.Contains(hints, "cancel") {
		t.Error("hints should contain 'cancel'")
	}
}
