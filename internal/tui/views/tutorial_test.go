package views

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTutorialModel(t *testing.T) {
	m := NewTutorialModel(testStyles())
	assert.False(t, m.visible, "tutorial should not be visible initially")
}

func TestTutorialShow(t *testing.T) {
	m := NewTutorialModel(testStyles())
	m.SetSize(80, 24)
	m.Show()

	assert.True(t, m.Visible(), "should be visible after Show()")
	assert.Equal(t, 0, m.step)
}

func TestTutorialSetSize(t *testing.T) {
	m := NewTutorialModel(testStyles())
	m.SetSize(120, 40)
	assert.Equal(t, 120, m.width)
	assert.Equal(t, 40, m.height)
}

func TestTutorialStepThrough(t *testing.T) {
	m := NewTutorialModel(testStyles())
	m.SetSize(80, 24)
	m.Show()

	total := len(tutorialSteps)

	// Step through all but last.
	for i := range total - 1 {
		cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		assert.Nil(t, cmd, "step %d should not produce cmd before last step", i)
		assert.Equal(t, i+1, m.step)
		assert.True(t, m.visible, "should still be visible at step %d", i+1)
	}

	// Enter on last step dismisses.
	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd, "Enter on last step should produce a command")
	msg := cmd()
	_, ok := msg.(TutorialDoneMsg)
	assert.True(t, ok, "expected TutorialDoneMsg")
	assert.False(t, m.visible, "should not be visible after last step Enter")
}

func TestTutorialSpaceAdvances(t *testing.T) {
	m := NewTutorialModel(testStyles())
	m.SetSize(80, 24)
	m.Show()

	m.Update(tea.KeyMsg{Type: tea.KeySpace})
	assert.Equal(t, 1, m.step, "step after Space")
}

func TestTutorialEscapeSkips(t *testing.T) {
	m := NewTutorialModel(testStyles())
	m.SetSize(80, 24)
	m.Show()

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	require.NotNil(t, cmd, "Escape should produce a command")
	msg := cmd()
	_, ok := msg.(TutorialDoneMsg)
	assert.True(t, ok, "expected TutorialDoneMsg")
	assert.False(t, m.visible, "should not be visible after Escape")
}

func TestTutorialInvisibleIgnoresKeys(t *testing.T) {
	m := NewTutorialModel(testStyles())
	m.SetSize(80, 24)
	// Not shown.

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd, "invisible tutorial should not respond to keys")
}

func TestTutorialView(t *testing.T) {
	m := NewTutorialModel(testStyles())
	m.SetSize(80, 24)
	m.Show()

	view := m.View()
	assert.NotEmpty(t, view, "tutorial view should not be empty")
}

func TestTutorialViewInvisible(t *testing.T) {
	m := NewTutorialModel(testStyles())
	m.SetSize(80, 24)

	view := m.View()
	assert.Empty(t, view, "invisible tutorial view should be empty")
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
		assert.Equal(t, tt.want, got)
	}
}

func TestFirstLaunchDetection(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)

	// No flag file yet — should be first launch.
	assert.True(t, IsFirstLaunch(), "should be first launch")

	// Mark done.
	err := MarkTutorialDone()
	require.NoError(t, err)

	// Should no longer be first launch.
	assert.False(t, IsFirstLaunch(), "should not be first launch after marking done")

	// Verify file exists.
	path := filepath.Join(tmpDir, "vivecaka", "tutorial_done")
	_, err = os.Stat(path)
	assert.NoError(t, err, "tutorial_done file should exist")
}
