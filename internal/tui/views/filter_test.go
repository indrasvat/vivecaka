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

func TestFilterKeyboardFlowCoversFieldsAndActions(t *testing.T) {
	m := NewFilterModel(testStyles(), testKeys())
	m.SetSize(90, 30)
	m.SetOpts(domain.ListOpts{
		State:   domain.PRStateClosed,
		Author:  "alice",
		Labels:  []string{"bug", "docs"},
		CI:      domain.CIFail,
		Review:  domain.ReviewPending,
		Draft:   domain.DraftOnly,
		PerPage: 25,
	})

	opts := m.Opts()
	assert.Equal(t, domain.PRStateClosed, opts.State)
	assert.Equal(t, "alice", opts.Author)
	assert.ElementsMatch(t, []string{"bug", "docs"}, opts.Labels)
	assert.Equal(t, domain.CIFail, opts.CI)
	assert.Equal(t, domain.ReviewPending, opts.Review)
	assert.Equal(t, domain.DraftOnly, opts.Draft)
	assert.Equal(t, 25, opts.PerPage)

	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, filterFieldAuthor, m.focus)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'!'}})
	assert.Equal(t, "alice!", m.author)
	m.Update(tea.KeyMsg{Type: tea.KeySpace})
	assert.Equal(t, "alice! ", m.author)
	m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	assert.Equal(t, "alice!", m.author)

	m.focus = filterFieldLabel
	m.Update(tea.KeyMsg{Type: tea.KeyRight})
	assert.Equal(t, 1, m.labelCursor)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	assert.Equal(t, 0, m.labelCursor)
	m.Update(tea.KeyMsg{Type: tea.KeySpace})
	assert.True(t, m.labelSelected[m.labelOptions[0]])

	m.focus = filterFieldStatus
	m.Update(tea.KeyMsg{Type: tea.KeySpace})
	assert.Equal(t, 2, m.statusIdx)
	m.focus = filterFieldCI
	m.Update(tea.KeyMsg{Type: tea.KeySpace})
	assert.NotEqual(t, domain.CIFail, m.Opts().CI)
	m.focus = filterFieldReview
	m.Update(tea.KeyMsg{Type: tea.KeySpace})
	assert.NotEqual(t, domain.ReviewPending, m.Opts().Review)
	m.focus = filterFieldDraft
	m.Update(tea.KeyMsg{Type: tea.KeySpace})
	assert.NotEqual(t, domain.DraftOnly, m.Opts().Draft)

	m.focus = filterFieldApply
	msg := m.Update(tea.KeyMsg{Type: tea.KeyEnter})()
	_, ok := msg.(ApplyFilterMsg)
	assert.True(t, ok)

	m.focus = filterFieldCancel
	msg = m.Update(tea.KeyMsg{Type: tea.KeySpace})()
	_, ok = msg.(CloseFilterMsg)
	assert.True(t, ok)

	m.focus = filterFieldReset
	assert.Nil(t, m.Update(tea.KeyMsg{Type: tea.KeyEnter}))
	assert.Empty(t, m.author)
	assert.Empty(t, m.Opts().Labels)
}

func TestFilterNavigationHelpersAndBounds(t *testing.T) {
	m := NewFilterModel(testStyles(), testKeys())
	m.labelOptions = nil
	m.labelCursor = 10
	m.moveLabelCursor(1)
	assert.Equal(t, 0, m.labelCursor)

	m.focus = 0
	m.prevField()
	assert.Equal(t, filterFieldCount-1, m.focus)
	m.nextField()
	assert.Equal(t, 0, m.focus)

	assert.Equal(t, 0, indexOfCI(domain.CIStatus("unknown")))
	assert.Equal(t, 0, indexOfReview(domain.ReviewState("unknown")))
	assert.Equal(t, 0, indexOfDraft(domain.DraftFilter("unknown")))
	assert.Equal(t, "abc", appendRune("abc", 'd', 3))
	assert.Equal(t, "abcd", appendRune("abc", 'd', 0))
	assert.Equal(t, "", backspace(""))
	assert.Equal(t, "ab", backspace("abc"))

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	_, ok := cmd().(CloseFilterMsg)
	assert.True(t, ok)
	assert.Nil(t, m.Update("not a key"))
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
