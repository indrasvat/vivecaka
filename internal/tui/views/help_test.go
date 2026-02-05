package views

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/indrasvat/vivecaka/internal/tui/core"
)

func TestNewHelpModel(t *testing.T) {
	m := NewHelpModel(testStyles())
	if m.context != core.ViewLoading {
		t.Errorf("default context = %d, want ViewLoading", m.context)
	}
}

func TestHelpSetSize(t *testing.T) {
	m := NewHelpModel(testStyles())
	m.SetSize(120, 40)
	if m.width != 120 || m.height != 40 {
		t.Errorf("size = %dx%d, want 120x40", m.width, m.height)
	}
}

func TestHelpSetContext(t *testing.T) {
	m := NewHelpModel(testStyles())
	m.SetContext(core.ViewPRList)
	if m.context != core.ViewPRList {
		t.Errorf("context = %d, want ViewPRList", m.context)
	}
}

func TestHelpEscClose(t *testing.T) {
	m := NewHelpModel(testStyles())
	m.SetSize(120, 40)
	m.SetContext(core.ViewPRList)

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("Escape should produce a command")
	}
	msg := cmd()
	if _, ok := msg.(CloseHelpMsg); !ok {
		t.Errorf("expected CloseHelpMsg, got %T", msg)
	}
}

func TestHelpQuestionMarkClose(t *testing.T) {
	m := NewHelpModel(testStyles())
	m.SetSize(120, 40)

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if cmd == nil {
		t.Fatal("? should produce a command")
	}
	msg := cmd()
	if _, ok := msg.(CloseHelpMsg); !ok {
		t.Errorf("expected CloseHelpMsg, got %T", msg)
	}
}

func TestHelpOtherKeysIgnored(t *testing.T) {
	m := NewHelpModel(testStyles())
	m.SetSize(120, 40)

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd != nil {
		t.Error("other keys should return nil cmd")
	}
}

func TestHelpNonKeyMsg(t *testing.T) {
	m := NewHelpModel(testStyles())
	cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if cmd != nil {
		t.Error("non-key messages should return nil cmd")
	}
}

func TestHelpViewPRList(t *testing.T) {
	m := NewHelpModel(testStyles())
	m.SetSize(120, 40)
	m.SetContext(core.ViewPRList)

	view := m.View()
	if view == "" {
		t.Error("PRList help view should not be empty")
	}
}

func TestHelpViewPRDetail(t *testing.T) {
	m := NewHelpModel(testStyles())
	m.SetSize(120, 40)
	m.SetContext(core.ViewPRDetail)

	view := m.View()
	if view == "" {
		t.Error("PRDetail help view should not be empty")
	}
}

func TestHelpViewDiff(t *testing.T) {
	m := NewHelpModel(testStyles())
	m.SetSize(120, 40)
	m.SetContext(core.ViewDiff)

	view := m.View()
	if view == "" {
		t.Error("Diff help view should not be empty")
	}
}

func TestHelpViewReview(t *testing.T) {
	m := NewHelpModel(testStyles())
	m.SetSize(120, 40)
	m.SetContext(core.ViewReview)

	view := m.View()
	if view == "" {
		t.Error("Review help view should not be empty")
	}
}

func TestHelpViewInbox(t *testing.T) {
	m := NewHelpModel(testStyles())
	m.SetSize(120, 40)
	m.SetContext(core.ViewInbox)

	view := m.View()
	if view == "" {
		t.Error("Inbox help view should not be empty")
	}
}

func TestHelpViewDefault(t *testing.T) {
	m := NewHelpModel(testStyles())
	m.SetSize(120, 40)
	m.SetContext(core.ViewLoading)

	view := m.View()
	if view == "" {
		t.Error("default help view should not be empty")
	}
}

func TestHelpViewSmall(t *testing.T) {
	m := NewHelpModel(testStyles())
	m.SetSize(40, 15) // Small terminal.
	m.SetContext(core.ViewPRList)

	view := m.View()
	if view == "" {
		t.Error("small help view should not be empty")
	}
}

func TestStatusHints(t *testing.T) {
	views := []core.ViewState{
		core.ViewPRList,
		core.ViewPRDetail,
		core.ViewDiff,
		core.ViewReview,
		core.ViewInbox,
		core.ViewRepoSwitch,
		core.ViewHelp,
		core.ViewLoading,
	}

	for _, v := range views {
		hints := StatusHints(v, 120)
		if hints == "" {
			t.Errorf("StatusHints(%d, 120) should not be empty", v)
		}
	}
}

func TestStatusHintsTruncation(t *testing.T) {
	hints := StatusHints(core.ViewPRList, 30)
	if len(hints) > 30 {
		t.Errorf("hints should be truncated to width, got len=%d", len(hints))
	}
}

func TestStatusHintsWideTerminal(t *testing.T) {
	hints := StatusHints(core.ViewPRList, 200)
	if hints == "" {
		t.Error("wide terminal hints should not be empty")
	}
}
