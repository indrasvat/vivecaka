package views

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewTutorialModel(t *testing.T) {
	m := NewTutorialModel(testStyles())
	if m.visible {
		t.Error("tutorial should not be visible initially")
	}
}

func TestTutorialShow(t *testing.T) {
	m := NewTutorialModel(testStyles())
	m.SetSize(80, 24)
	m.Show()

	if !m.Visible() {
		t.Error("should be visible after Show()")
	}
	if m.step != 0 {
		t.Errorf("step = %d, want 0", m.step)
	}
}

func TestTutorialSetSize(t *testing.T) {
	m := NewTutorialModel(testStyles())
	m.SetSize(120, 40)
	if m.width != 120 || m.height != 40 {
		t.Errorf("size = %dx%d, want 120x40", m.width, m.height)
	}
}

func TestTutorialStepThrough(t *testing.T) {
	m := NewTutorialModel(testStyles())
	m.SetSize(80, 24)
	m.Show()

	total := len(tutorialSteps)

	// Step through all but last.
	for i := range total - 1 {
		cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd != nil {
			t.Errorf("step %d: should not produce cmd before last step", i)
		}
		if m.step != i+1 {
			t.Errorf("step = %d, want %d", m.step, i+1)
		}
		if !m.visible {
			t.Errorf("should still be visible at step %d", i+1)
		}
	}

	// Enter on last step dismisses.
	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter on last step should produce a command")
	}
	msg := cmd()
	if _, ok := msg.(TutorialDoneMsg); !ok {
		t.Errorf("expected TutorialDoneMsg, got %T", msg)
	}
	if m.visible {
		t.Error("should not be visible after last step Enter")
	}
}

func TestTutorialSpaceAdvances(t *testing.T) {
	m := NewTutorialModel(testStyles())
	m.SetSize(80, 24)
	m.Show()

	m.Update(tea.KeyMsg{Type: tea.KeySpace})
	if m.step != 1 {
		t.Errorf("step after Space = %d, want 1", m.step)
	}
}

func TestTutorialEscapeSkips(t *testing.T) {
	m := NewTutorialModel(testStyles())
	m.SetSize(80, 24)
	m.Show()

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("Escape should produce a command")
	}
	msg := cmd()
	if _, ok := msg.(TutorialDoneMsg); !ok {
		t.Errorf("expected TutorialDoneMsg, got %T", msg)
	}
	if m.visible {
		t.Error("should not be visible after Escape")
	}
}

func TestTutorialInvisibleIgnoresKeys(t *testing.T) {
	m := NewTutorialModel(testStyles())
	m.SetSize(80, 24)
	// Not shown.

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("invisible tutorial should not respond to keys")
	}
}

func TestTutorialView(t *testing.T) {
	m := NewTutorialModel(testStyles())
	m.SetSize(80, 24)
	m.Show()

	view := m.View()
	if view == "" {
		t.Error("tutorial view should not be empty")
	}
}

func TestTutorialViewInvisible(t *testing.T) {
	m := NewTutorialModel(testStyles())
	m.SetSize(80, 24)

	view := m.View()
	if view != "" {
		t.Error("invisible tutorial view should be empty")
	}
}

func TestProgressDots(t *testing.T) {
	tests := []struct {
		current, total int
		want           string
	}{
		{0, 3, "● ○ ○"},
		{1, 3, "● ● ○"},
		{2, 3, "● ● ●"},
		{0, 1, "●"},
	}
	for _, tt := range tests {
		got := progressDots(tt.current, tt.total)
		if got != tt.want {
			t.Errorf("progressDots(%d, %d) = %q, want %q", tt.current, tt.total, got, tt.want)
		}
	}
}

func TestFirstLaunchDetection(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)

	// No flag file yet — should be first launch.
	if !IsFirstLaunch() {
		t.Error("should be first launch")
	}

	// Mark done.
	if err := MarkTutorialDone(); err != nil {
		t.Fatalf("MarkTutorialDone: %v", err)
	}

	// Should no longer be first launch.
	if IsFirstLaunch() {
		t.Error("should not be first launch after marking done")
	}

	// Verify file exists.
	path := filepath.Join(tmpDir, "vivecaka", "tutorial_done")
	if _, err := os.Stat(path); err != nil {
		t.Errorf("tutorial_done file should exist: %v", err)
	}
}
