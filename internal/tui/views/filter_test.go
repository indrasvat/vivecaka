package views

import (
	"reflect"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/indrasvat/vivecaka/internal/domain"
)

func TestFilterDefaults(t *testing.T) {
	m := NewFilterModel(testStyles(), testKeys())
	opts := m.Opts()

	assert.Equal(t, domain.PRStateOpen, opts.State)
	assert.Empty(t, opts.Author)
	assert.Empty(t, opts.Labels)
	assert.Empty(t, opts.CI)
	assert.Empty(t, opts.Review)
	assert.Equal(t, domain.DraftInclude, opts.Draft)
}

func TestFilterSetOpts(t *testing.T) {
	m := NewFilterModel(testStyles(), testKeys())
	m.SetOpts(domain.ListOpts{
		State:  domain.PRStateClosed,
		Author: "alice",
		Labels: []string{"bug"},
		CI:     domain.CIFail,
		Review: domain.ReviewPending,
		Draft:  domain.DraftOnly,
	})

	opts := m.Opts()
	assert.Equal(t, domain.PRStateClosed, opts.State)
	assert.Equal(t, "alice", opts.Author)
	assert.True(t, reflect.DeepEqual(opts.Labels, []string{"bug"}))
	assert.Equal(t, domain.CIFail, opts.CI)
	assert.Equal(t, domain.ReviewPending, opts.Review)
	assert.Equal(t, domain.DraftOnly, opts.Draft)
}

func TestFilterApplyMessage(t *testing.T) {
	m := NewFilterModel(testStyles(), testKeys())
	m.focus = filterFieldApply

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd, "expected apply command")
	_, ok := cmd().(ApplyFilterMsg)
	assert.True(t, ok, "expected ApplyFilterMsg")
}

func TestFilterEnterAcceptsWithoutTogglingFocusedField(t *testing.T) {
	m := NewFilterModel(testStyles(), testKeys())
	m.focus = filterFieldStatus
	m.statusIdx = 0

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd, "expected accept command")

	_, ok := cmd().(ApplyFilterMsg)
	assert.True(t, ok, "expected ApplyFilterMsg")
	assert.Equal(t, 0, m.statusIdx, "enter should not toggle the status field")
}

func TestFilterCancelMessage(t *testing.T) {
	m := NewFilterModel(testStyles(), testKeys())
	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	require.NotNil(t, cmd, "expected close command")
	_, ok := cmd().(CloseFilterMsg)
	assert.True(t, ok, "expected CloseFilterMsg")
}

func TestFilterResetKey(t *testing.T) {
	m := NewFilterModel(testStyles(), testKeys())
	m.statusIdx = 2
	m.author = "bob"
	m.labelSelected["bug"] = true
	m.ciIdx = 2
	m.reviewIdx = 2
	m.draftIdx = 2

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	opts := m.Opts()
	assert.Equal(t, domain.PRStateOpen, opts.State)
	assert.Empty(t, opts.Author)
	assert.Empty(t, opts.Labels)
	assert.Equal(t, domain.DraftInclude, opts.Draft)
}

func TestFilterLabelToggle(t *testing.T) {
	m := NewFilterModel(testStyles(), testKeys())
	m.focus = filterFieldLabel

	m.Update(tea.KeyMsg{Type: tea.KeySpace})
	opts := m.Opts()
	assert.True(t, reflect.DeepEqual(opts.Labels, []string{"enhancement"}), "labels after toggle")

	m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m.Update(tea.KeyMsg{Type: tea.KeySpace})
	opts = m.Opts()
	assert.True(t, reflect.DeepEqual(opts.Labels, []string{"enhancement", "bug"}), "labels after second toggle")
}

func TestFilterMnemonics(t *testing.T) {
	m := NewFilterModel(testStyles(), testKeys())
	m.statusIdx = 2

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	require.NotNil(t, cmd, "accept mnemonic should return command")
	_, ok := cmd().(ApplyFilterMsg)
	assert.True(t, ok, "expected ApplyFilterMsg")

	cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	require.NotNil(t, cmd, "cancel mnemonic should return command")
	_, ok = cmd().(CloseFilterMsg)
	assert.True(t, ok, "expected CloseFilterMsg")

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	assert.Equal(t, 0, m.statusIdx, "reset mnemonic should restore default status")
}

func TestFilterMnemonicsDoNotHijackAuthorInput(t *testing.T) {
	m := NewFilterModel(testStyles(), testKeys())
	m.focus = filterFieldAuthor

	assert.Nil(t, m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}))
	assert.Nil(t, m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}))
	assert.Nil(t, m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}))
	assert.Equal(t, "arc", m.author)
}

func TestFilterViewShowsAcceptAction(t *testing.T) {
	m := NewFilterModel(testStyles(), testKeys())
	m.SetSize(120, 40)

	view := m.View()
	assert.Contains(t, view, "[ Accept ]")
	assert.Contains(t, view, "[ Reset ]")
	assert.Contains(t, view, "[ Cancel ]")
}
