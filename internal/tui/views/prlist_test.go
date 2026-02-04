package views

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/indrasvat/vivecaka/internal/tui"
)

func testStyles() tui.Styles {
	return tui.NewStyles(tui.ThemeByName("catppuccin-mocha"))
}

func testKeys() tui.KeyMap {
	return tui.DefaultKeyMap()
}

func testPRs() []domain.PR {
	now := time.Now()
	return []domain.PR{
		{Number: 142, Title: "Add plugin architecture", Author: "indrasvat", CI: domain.CIPass, Review: domain.ReviewStatus{State: domain.ReviewApproved, Approved: 2, Total: 2}, UpdatedAt: now.Add(-2 * time.Hour), Branch: domain.BranchInfo{Head: "feat/plugins"}},
		{Number: 141, Title: "Fix diff viewer alignment", Author: "alice", CI: domain.CIFail, Review: domain.ReviewStatus{State: domain.ReviewPending, Approved: 0, Total: 1}, UpdatedAt: now.Add(-5 * time.Hour)},
		{Number: 140, Title: "Update CI pipeline", Author: "bob", CI: domain.CIPending, UpdatedAt: now.Add(-24 * time.Hour)},
		{Number: 139, Title: "New theme engine", Author: "indrasvat", Draft: true, UpdatedAt: now.Add(-48 * time.Hour)},
		{Number: 138, Title: "Refactor config loader", Author: "carol", CI: domain.CIPass, Review: domain.ReviewStatus{State: domain.ReviewChangesRequested, Approved: 1, Total: 2}, UpdatedAt: now.Add(-72 * time.Hour)},
	}
}

func TestNewPRListModel(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	if !m.loading {
		t.Error("new model should be in loading state")
	}
	if m.sortField != "updated" {
		t.Errorf("default sort = %q, want %q", m.sortField, "updated")
	}
}

func TestSetPRs(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetPRs(testPRs())

	if m.loading {
		t.Error("loading should be false after SetPRs")
	}
	if len(m.filtered) != 5 {
		t.Errorf("filtered len = %d, want 5", len(m.filtered))
	}
}

func TestSelectedPR(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetPRs(testPRs())

	pr := m.SelectedPR()
	if pr == nil {
		t.Fatal("SelectedPR() should not be nil")
	}
	if pr.Number != 142 {
		t.Errorf("selected PR number = %d, want 142", pr.Number)
	}
}

func TestSelectedPREmpty(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetPRs(nil)

	if pr := m.SelectedPR(); pr != nil {
		t.Error("SelectedPR() should be nil for empty list")
	}
}

func TestNavigationDown(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetSize(120, 30)
	m.SetPRs(testPRs())

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	m.Update(msg)

	if m.cursor != 1 {
		t.Errorf("cursor after j = %d, want 1", m.cursor)
	}
}

func TestNavigationUp(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetSize(120, 30)
	m.SetPRs(testPRs())
	m.cursor = 2

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	m.Update(msg)

	if m.cursor != 1 {
		t.Errorf("cursor after k = %d, want 1", m.cursor)
	}
}

func TestNavigationBounds(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetSize(120, 30)
	m.SetPRs(testPRs())

	// Already at top, going up should stay.
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	m.Update(msg)
	if m.cursor != 0 {
		t.Errorf("cursor at top after k = %d, want 0", m.cursor)
	}

	// Go to bottom.
	m.cursor = 4
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	m.Update(msg)
	if m.cursor != 4 {
		t.Errorf("cursor at bottom after j = %d, want 4", m.cursor)
	}
}

func TestTopBottom(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetSize(120, 30)
	m.SetPRs(testPRs())
	m.cursor = 2

	// G = bottom
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
	m.Update(msg)
	if m.cursor != 4 {
		t.Errorf("cursor after G = %d, want 4", m.cursor)
	}

	// g = top
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
	m.Update(msg)
	if m.cursor != 0 {
		t.Errorf("cursor after g = %d, want 0", m.cursor)
	}
}

func TestSearchFilter(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetSize(120, 30)
	m.SetPRs(testPRs())

	m.searchQuery = "alice"
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Errorf("filtered count after search = %d, want 1", len(m.filtered))
	}
	if m.filtered[0].Author != "alice" {
		t.Errorf("filtered[0].Author = %q, want %q", m.filtered[0].Author, "alice")
	}
}

func TestSearchFilterByTitle(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetPRs(testPRs())

	m.searchQuery = "plugin"
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Errorf("filtered count = %d, want 1", len(m.filtered))
	}
}

func TestSearchFilterCaseInsensitive(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetPRs(testPRs())

	m.searchQuery = "ALICE"
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Errorf("case-insensitive search: filtered count = %d, want 1", len(m.filtered))
	}
}

func TestCycleSort(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetPRs(testPRs())

	if m.sortField != "updated" {
		t.Fatalf("initial sort = %q", m.sortField)
	}

	m.cycleSort()
	if m.sortField != "created" {
		t.Errorf("after 1st cycle = %q, want %q", m.sortField, "created")
	}

	m.cycleSort()
	if m.sortField != "number" {
		t.Errorf("after 2nd cycle = %q, want %q", m.sortField, "number")
	}
}

func TestCurrentBranch(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetPRs(testPRs())
	m.SetCurrentBranch("feat/plugins")

	if m.currentBranch != "feat/plugins" {
		t.Errorf("currentBranch = %q, want %q", m.currentBranch, "feat/plugins")
	}
}

func TestViewLoading(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetSize(80, 24)

	view := m.View()
	if view == "" {
		t.Error("loading view should not be empty")
	}
}

func TestViewEmpty(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetSize(80, 24)
	m.SetPRs(nil)

	view := m.View()
	if view == "" {
		t.Error("empty view should not be empty string")
	}
}

func TestViewWithData(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetSize(120, 30)
	m.SetPRs(testPRs())

	view := m.View()
	if view == "" {
		t.Error("view with data should not be empty")
	}
}

func TestCIIcon(t *testing.T) {
	tests := []struct {
		status domain.CIStatus
		want   string
	}{
		{domain.CIPass, "✓"},
		{domain.CIFail, "✗"},
		{domain.CIPending, "◐"},
		{domain.CISkipped, "○"},
		{domain.CINone, "—"},
	}
	for _, tt := range tests {
		if got := ciIcon(tt.status); got != tt.want {
			t.Errorf("ciIcon(%q) = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestReviewText(t *testing.T) {
	tests := []struct {
		status domain.ReviewStatus
		want   string
	}{
		{domain.ReviewStatus{State: domain.ReviewApproved, Approved: 2, Total: 2}, "✓ 2/2"},
		{domain.ReviewStatus{State: domain.ReviewChangesRequested, Approved: 1, Total: 2}, "! 1/2"},
		{domain.ReviewStatus{State: domain.ReviewPending, Approved: 0, Total: 1}, "● 0/1"},
		{domain.ReviewStatus{State: domain.ReviewNone}, "—"},
	}
	for _, tt := range tests {
		if got := reviewText(tt.status); got != tt.want {
			t.Errorf("reviewText(%+v) = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestRelativeTime(t *testing.T) {
	now := time.Now()
	tests := []struct {
		t    time.Time
		want string
	}{
		{now.Add(-30 * time.Second), "<1m"},
		{now.Add(-5 * time.Minute), "5m"},
		{now.Add(-3 * time.Hour), "3h"},
		{now.Add(-48 * time.Hour), "2d"},
	}
	for _, tt := range tests {
		if got := relativeTime(tt.t); got != tt.want {
			t.Errorf("relativeTime(%v ago) = %q, want %q", time.Since(tt.t), got, tt.want)
		}
	}
}
