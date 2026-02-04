package views

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/indrasvat/vivecaka/internal/domain"
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
	if !m.loading {
		t.Error("should be loading initially")
	}
	if m.tab != InboxAll {
		t.Errorf("default tab = %d, want InboxAll", m.tab)
	}
}

func TestInboxSetPRs(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	m.SetPRs(testInboxPRs())

	if m.loading {
		t.Error("should not be loading after SetPRs")
	}
	if len(m.filtered) != 4 {
		t.Errorf("filtered = %d, want 4", len(m.filtered))
	}
}

func TestInboxSetSize(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	if m.width != 120 || m.height != 40 {
		t.Errorf("size = %dx%d, want 120x40", m.width, m.height)
	}
}

func TestInboxSetUsername(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	m.SetUsername("indrasvat")
	if m.username != "indrasvat" {
		t.Errorf("username = %q, want %q", m.username, "indrasvat")
	}
}

func TestInboxTabNavigation(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetUsername("indrasvat")
	m.SetPRs(testInboxPRs())

	tab := tea.KeyMsg{Type: tea.KeyTab}

	m.Update(tab)
	if m.tab != InboxAssigned {
		t.Errorf("after 1 tab = %d, want InboxAssigned", m.tab)
	}

	m.Update(tab)
	if m.tab != InboxReviewRequested {
		t.Errorf("after 2 tabs = %d, want InboxReviewRequested", m.tab)
	}

	m.Update(tab)
	if m.tab != InboxMyPRs {
		t.Errorf("after 3 tabs = %d, want InboxMyPRs", m.tab)
	}

	m.Update(tab)
	if m.tab != InboxAll {
		t.Errorf("after 4 tabs = %d, want InboxAll (wrap)", m.tab)
	}
}

func TestInboxShiftTab(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetPRs(testInboxPRs())

	shiftTab := tea.KeyMsg{Type: tea.KeyShiftTab}
	m.Update(shiftTab)
	if m.tab != InboxMyPRs {
		t.Errorf("after shift-tab = %d, want InboxMyPRs", m.tab)
	}
}

func TestInboxTabAllFilter(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetUsername("indrasvat")
	m.SetPRs(testInboxPRs())

	// All tab shows everything.
	if len(m.filtered) != 4 {
		t.Errorf("All tab: filtered = %d, want 4", len(m.filtered))
	}
}

func TestInboxTabMyPRs(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetUsername("indrasvat")
	m.SetPRs(testInboxPRs())

	// Switch to My PRs tab.
	m.tab = InboxMyPRs
	m.applyFilter()

	if len(m.filtered) != 2 {
		t.Errorf("My PRs tab: filtered = %d, want 2", len(m.filtered))
	}
	for _, pr := range m.filtered {
		if pr.Author != "indrasvat" {
			t.Errorf("My PRs should only have indrasvat, got %q", pr.Author)
		}
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
	if len(m.filtered) != 2 {
		t.Errorf("Review Requested tab: filtered = %d, want 2", len(m.filtered))
	}
}

func TestInboxTabEmptyUsername(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetPRs(testInboxPRs())
	// No username set.

	m.tab = InboxMyPRs
	m.applyFilter()
	if len(m.filtered) != 0 {
		t.Errorf("My PRs with no username: filtered = %d, want 0", len(m.filtered))
	}
}

func TestInboxNavigation(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetPRs(testInboxPRs())

	down := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	m.Update(down)
	if m.cursor != 1 {
		t.Errorf("cursor after j = %d, want 1", m.cursor)
	}

	up := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	m.Update(up)
	if m.cursor != 0 {
		t.Errorf("cursor after k = %d, want 0", m.cursor)
	}
}

func TestInboxNavigationBounds(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetPRs(testInboxPRs())

	up := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	m.Update(up)
	if m.cursor != 0 {
		t.Errorf("cursor should stay at 0, got %d", m.cursor)
	}

	m.cursor = 3
	down := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	m.Update(down)
	if m.cursor != 3 {
		t.Errorf("cursor at end should stay at 3, got %d", m.cursor)
	}
}

func TestInboxEnter(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetPRs(testInboxPRs())

	enter := tea.KeyMsg{Type: tea.KeyEnter}
	cmd := m.Update(enter)
	if cmd == nil {
		t.Fatal("Enter should produce a command")
	}

	msg := cmd()
	open, ok := msg.(OpenInboxPRMsg)
	if !ok {
		t.Fatalf("expected OpenInboxPRMsg, got %T", msg)
	}
	if open.Number != 10 {
		t.Errorf("Number = %d, want 10", open.Number)
	}
	if open.Repo.Name != "webapp" {
		t.Errorf("Repo = %q, want webapp", open.Repo.Name)
	}
}

func TestInboxBack(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetPRs(testInboxPRs())

	back := tea.KeyMsg{Type: tea.KeyEscape}
	cmd := m.Update(back)
	if cmd == nil {
		t.Fatal("Back should produce a command")
	}

	msg := cmd()
	if _, ok := msg.(CloseInboxMsg); !ok {
		t.Errorf("expected CloseInboxMsg, got %T", msg)
	}
}

func TestInboxPRsLoadedMsg(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	m.SetSize(120, 40)

	m.Update(InboxPRsLoadedMsg{PRs: testInboxPRs()})
	if m.loading {
		t.Error("should not be loading after InboxPRsLoadedMsg")
	}
}

func TestInboxViewLoading(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	m.SetSize(80, 24)

	view := m.View()
	if view == "" {
		t.Error("loading view should not be empty")
	}
}

func TestInboxViewWithData(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetPRs(testInboxPRs())

	view := m.View()
	if view == "" {
		t.Error("view with data should not be empty")
	}
}

func TestInboxViewEmptyTab(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetPRs(testInboxPRs())
	m.tab = InboxMyPRs
	m.applyFilter()
	// No username, so My PRs should be empty.

	view := m.View()
	if view == "" {
		t.Error("empty tab view should not be empty string")
	}
}

func TestPrioritySort(t *testing.T) {
	prs := testInboxPRs()
	PrioritySort(prs, "indrasvat", 7)

	// First should be review-requested (PR#10 alice, pending, not by indrasvat).
	if prs[0].Number != 10 {
		t.Errorf("first PR after sort = #%d, want #10 (review requested)", prs[0].Number)
	}
}

func TestPrioritySortEmpty(t *testing.T) {
	// Should not panic.
	PrioritySort(nil, "user", 7)
	PrioritySort([]InboxPR{}, "user", 7)
}
