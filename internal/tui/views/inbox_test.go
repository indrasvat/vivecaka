package views

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testInboxPRs() []InboxPR {
	now := time.Now()
	return []InboxPR{
		{
			PR:   domain.PR{Number: 10, Title: "Fix auth", Author: "alice", CI: domain.CIFail, Review: domain.ReviewStatus{State: domain.ReviewPending, Approved: 0, Total: 1}, UpdatedAt: now.Add(-1 * time.Hour)},
			Repo: domain.RepoRef{Owner: "acme", Name: "webapp"},
		},
		{
			PR:   domain.PR{Number: 20, Title: "Add caching", Author: "indrasvat", CI: domain.CIPass, Review: domain.ReviewStatus{State: domain.ReviewApproved, Approved: 2, Total: 2}, UpdatedAt: now.Add(-3 * time.Hour)},
			Repo: domain.RepoRef{Owner: "acme", Name: "webapp"},
		},
		{
			PR:   domain.PR{Number: 5, Title: "New theme", Author: "bob", CI: domain.CIPending, Review: domain.ReviewStatus{State: domain.ReviewPending, Approved: 0, Total: 1}, UpdatedAt: now.Add(-48 * time.Hour)},
			Repo: domain.RepoRef{Owner: "indrasvat", Name: "vivecaka"},
		},
		{
			PR:   domain.PR{Number: 8, Title: "Update docs", Author: "indrasvat", CI: domain.CIPass, UpdatedAt: now.Add(-72 * time.Hour)},
			Repo: domain.RepoRef{Owner: "indrasvat", Name: "vivecaka"},
		},
	}
}

func TestNewInboxModel(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	assert.True(t, m.loading, "should be loading initially")
	assert.Equal(t, InboxAll, m.tab)
}

func TestInboxSetPRs(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	m.SetPRs(testInboxPRs())

	assert.False(t, m.loading, "should not be loading after SetPRs")
	assert.Len(t, m.filtered, 4)
}

func TestInboxSetSize(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	assert.Equal(t, 120, m.width)
	assert.Equal(t, 40, m.height)
}

func TestInboxSetUsername(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	m.SetUsername("indrasvat")
	assert.Equal(t, "indrasvat", m.username)
}

func TestInboxTabNavigation(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetUsername("indrasvat")
	m.SetPRs(testInboxPRs())

	tab := tea.KeyMsg{Type: tea.KeyTab}

	m.Update(tab)
	assert.Equal(t, InboxAssigned, m.tab)

	m.Update(tab)
	assert.Equal(t, InboxReviewRequested, m.tab)

	m.Update(tab)
	assert.Equal(t, InboxMyPRs, m.tab)

	m.Update(tab)
	assert.Equal(t, InboxAll, m.tab, "should wrap to InboxAll")
}

func TestInboxShiftTab(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetPRs(testInboxPRs())

	shiftTab := tea.KeyMsg{Type: tea.KeyShiftTab}
	m.Update(shiftTab)
	assert.Equal(t, InboxMyPRs, m.tab)
}

func TestInboxTabAllFilter(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetUsername("indrasvat")
	m.SetPRs(testInboxPRs())

	// All tab shows everything.
	assert.Len(t, m.filtered, 4)
}

func TestInboxTabMyPRs(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetUsername("indrasvat")
	m.SetPRs(testInboxPRs())

	// Switch to My PRs tab.
	m.tab = InboxMyPRs
	m.applyFilter()

	assert.Len(t, m.filtered, 2)
	for _, pr := range m.filtered {
		assert.Equal(t, "indrasvat", pr.Author, "My PRs should only have indrasvat")
	}
}

func TestInboxTabReviewRequested(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetUsername("indrasvat")
	m.SetPRs(testInboxPRs())

	// Switch to Review Requested tab.
	m.tab = InboxReviewRequested
	m.applyFilter()

	// PRs with pending review that are NOT authored by indrasvat.
	// PR#10 (alice, pending) and PR#5 (bob, pending) should appear.
	assert.Len(t, m.filtered, 2)
}

func TestInboxTabEmptyUsername(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetPRs(testInboxPRs())
	// No username set.

	m.tab = InboxMyPRs
	m.applyFilter()
	assert.Empty(t, m.filtered)
}

func TestInboxNavigation(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetPRs(testInboxPRs())

	down := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	m.Update(down)
	assert.Equal(t, 1, m.cursor)

	up := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	m.Update(up)
	assert.Equal(t, 0, m.cursor)
}

func TestInboxNavigationBounds(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetPRs(testInboxPRs())

	up := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	m.Update(up)
	assert.Equal(t, 0, m.cursor, "cursor should stay at 0")

	m.cursor = 3
	down := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	m.Update(down)
	assert.Equal(t, 3, m.cursor, "cursor at end should stay at 3")
}

func TestInboxEnter(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetPRs(testInboxPRs())

	enter := tea.KeyMsg{Type: tea.KeyEnter}
	cmd := m.Update(enter)
	require.NotNil(t, cmd, "Enter should produce a command")

	msg := cmd()
	open, ok := msg.(OpenInboxPRMsg)
	require.True(t, ok, "expected OpenInboxPRMsg, got %T", msg)
	assert.Equal(t, 10, open.Number)
	assert.Equal(t, "webapp", open.Repo.Name)
}

func TestInboxBack(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetPRs(testInboxPRs())

	back := tea.KeyMsg{Type: tea.KeyEscape}
	cmd := m.Update(back)
	require.NotNil(t, cmd, "Back should produce a command")

	msg := cmd()
	_, ok := msg.(CloseInboxMsg)
	assert.True(t, ok, "expected CloseInboxMsg, got %T", msg)
}

func TestInboxPRsLoadedMsg(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	m.SetSize(120, 40)

	m.Update(InboxPRsLoadedMsg{PRs: testInboxPRs()})
	assert.False(t, m.loading, "should not be loading after InboxPRsLoadedMsg")
}

func TestInboxViewLoading(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	m.SetSize(80, 24)

	view := m.View()
	assert.NotEmpty(t, view, "loading view should not be empty")
}

func TestInboxViewWithData(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetPRs(testInboxPRs())

	view := m.View()
	assert.NotEmpty(t, view, "view with data should not be empty")
}

func TestInboxViewEmptyTab(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetPRs(testInboxPRs())
	m.tab = InboxMyPRs
	m.applyFilter()
	// No username, so My PRs should be empty.

	view := m.View()
	assert.NotEmpty(t, view, "empty tab view should not be empty string")
}

func TestPrioritySort(t *testing.T) {
	prs := testInboxPRs()
	PrioritySort(prs, "indrasvat", 7)

	// First should be review-requested (PR#10 alice, pending, not by indrasvat).
	assert.Equal(t, 10, prs[0].Number, "first PR after sort should be #10 (review requested)")
}

func TestPrioritySortEmpty(t *testing.T) {
	// Should not panic.
	PrioritySort(nil, "user", 7)
	PrioritySort([]InboxPR{}, "user", 7)
}
