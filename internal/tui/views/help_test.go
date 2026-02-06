package views

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/indrasvat/vivecaka/internal/tui/core"
)

func TestNewHelpModel(t *testing.T) {
	m := NewHelpModel(testStyles())
	// Default context is ViewBanner (iota 0), but that's fine - context gets set when help opens
	assert.Equal(t, core.ViewBanner, m.context, "default context should be ViewBanner")
}

func TestHelpSetSize(t *testing.T) {
	m := NewHelpModel(testStyles())
	m.SetSize(120, 40)
	assert.Equal(t, 120, m.width)
	assert.Equal(t, 40, m.height)
}

func TestHelpSetContext(t *testing.T) {
	m := NewHelpModel(testStyles())
	m.SetContext(core.ViewPRList)
	assert.Equal(t, core.ViewPRList, m.context)
}

func TestHelpEscClose(t *testing.T) {
	m := NewHelpModel(testStyles())
	m.SetSize(120, 40)
	m.SetContext(core.ViewPRList)

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	require.NotNil(t, cmd, "Escape should produce a command")
	msg := cmd()
	_, ok := msg.(CloseHelpMsg)
	assert.True(t, ok, "expected CloseHelpMsg")
}

func TestHelpQuestionMarkClose(t *testing.T) {
	m := NewHelpModel(testStyles())
	m.SetSize(120, 40)

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	require.NotNil(t, cmd, "? should produce a command")
	msg := cmd()
	_, ok := msg.(CloseHelpMsg)
	assert.True(t, ok, "expected CloseHelpMsg")
}

func TestHelpOtherKeysIgnored(t *testing.T) {
	m := NewHelpModel(testStyles())
	m.SetSize(120, 40)

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	assert.Nil(t, cmd, "other keys should return nil cmd")
}

func TestHelpNonKeyMsg(t *testing.T) {
	m := NewHelpModel(testStyles())
	cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	assert.Nil(t, cmd, "non-key messages should return nil cmd")
}

func TestHelpViewPRList(t *testing.T) {
	m := NewHelpModel(testStyles())
	m.SetSize(120, 40)
	m.SetContext(core.ViewPRList)

	view := m.View()
	assert.NotEmpty(t, view, "PRList help view should not be empty")
}

func TestHelpViewPRDetail(t *testing.T) {
	m := NewHelpModel(testStyles())
	m.SetSize(120, 40)
	m.SetContext(core.ViewPRDetail)

	view := m.View()
	assert.NotEmpty(t, view, "PRDetail help view should not be empty")
}

func TestHelpViewDiff(t *testing.T) {
	m := NewHelpModel(testStyles())
	m.SetSize(120, 40)
	m.SetContext(core.ViewDiff)

	view := m.View()
	assert.NotEmpty(t, view, "Diff help view should not be empty")
}

func TestHelpViewReview(t *testing.T) {
	m := NewHelpModel(testStyles())
	m.SetSize(120, 40)
	m.SetContext(core.ViewReview)

	view := m.View()
	assert.NotEmpty(t, view, "Review help view should not be empty")
}

func TestHelpViewInbox(t *testing.T) {
	m := NewHelpModel(testStyles())
	m.SetSize(120, 40)
	m.SetContext(core.ViewInbox)

	view := m.View()
	assert.NotEmpty(t, view, "Inbox help view should not be empty")
}

func TestHelpViewDefault(t *testing.T) {
	m := NewHelpModel(testStyles())
	m.SetSize(120, 40)
	m.SetContext(core.ViewLoading)

	view := m.View()
	assert.NotEmpty(t, view, "default help view should not be empty")
}

func TestHelpViewSmall(t *testing.T) {
	m := NewHelpModel(testStyles())
	m.SetSize(40, 15) // Small terminal.
	m.SetContext(core.ViewPRList)

	view := m.View()
	assert.NotEmpty(t, view, "small help view should not be empty")
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
		core.ViewFilter,
		core.ViewLoading,
	}

	for _, v := range views {
		hints := StatusHints(v, 120)
		assert.NotEmpty(t, hints, "StatusHints(%d, 120) should not be empty", v)
	}
}

func TestStatusHintsTruncation(t *testing.T) {
	hints := StatusHints(core.ViewPRList, 30)
	assert.LessOrEqual(t, len(hints), 30, "hints should be truncated to width")
}

func TestStatusHintsWideTerminal(t *testing.T) {
	hints := StatusHints(core.ViewPRList, 200)
	assert.NotEmpty(t, hints, "wide terminal hints should not be empty")
}
