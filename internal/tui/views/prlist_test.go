package views

import (
	"reflect"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/indrasvat/vivecaka/internal/tui/core"
)

func testStyles() core.Styles {
	return core.NewStyles(core.ThemeByName("catppuccin-mocha"))
}

func testKeys() core.KeyMap {
	return core.DefaultKeyMap()
}

func testPRs() []domain.PR {
	now := time.Now()
	return []domain.PR{
		{
			Number:    142,
			Title:     "Add plugin architecture",
			Author:    "indrasvat",
			State:     domain.PRStateOpen,
			CI:        domain.CIPass,
			Review:    domain.ReviewStatus{State: domain.ReviewApproved, Approved: 2, Total: 2},
			UpdatedAt: now.Add(-2 * time.Hour),
			CreatedAt: now.Add(-10 * time.Hour),
			Branch:    domain.BranchInfo{Head: "feat/plugins"},
		},
		{
			Number:    141,
			Title:     "Fix diff viewer alignment",
			Author:    "alice",
			State:     domain.PRStateOpen,
			CI:        domain.CIFail,
			Review:    domain.ReviewStatus{State: domain.ReviewPending, Approved: 0, Total: 1},
			UpdatedAt: now.Add(-5 * time.Hour),
			CreatedAt: now.Add(-12 * time.Hour),
		},
		{
			Number:    140,
			Title:     "Update CI pipeline",
			Author:    "bob",
			State:     domain.PRStateOpen,
			CI:        domain.CIPending,
			UpdatedAt: now.Add(-24 * time.Hour),
			CreatedAt: now.Add(-24 * time.Hour),
		},
		{
			Number:    139,
			Title:     "New theme engine",
			Author:    "indrasvat",
			State:     domain.PRStateOpen,
			Draft:     true,
			UpdatedAt: now.Add(-48 * time.Hour),
			CreatedAt: now.Add(-60 * time.Hour),
		},
		{
			Number:    138,
			Title:     "Refactor config loader",
			Author:    "carol",
			State:     domain.PRStateOpen,
			CI:        domain.CIPass,
			Review:    domain.ReviewStatus{State: domain.ReviewChangesRequested, Approved: 1, Total: 2},
			UpdatedAt: now.Add(-72 * time.Hour),
			CreatedAt: now.Add(-96 * time.Hour),
		},
	}
}

func testPRsForSort() []domain.PR {
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	return []domain.PR{
		{Number: 3, Title: "Zulu", Author: "zoe", State: domain.PRStateOpen, UpdatedAt: base.Add(5 * time.Hour), CreatedAt: base.Add(-1 * time.Hour)},
		{Number: 1, Title: "Alpha", Author: "alice", State: domain.PRStateOpen, UpdatedAt: base.Add(1 * time.Hour), CreatedAt: base.Add(-5 * time.Hour)},
		{Number: 5, Title: "Echo", Author: "mike", State: domain.PRStateOpen, UpdatedAt: base.Add(3 * time.Hour), CreatedAt: base.Add(-2 * time.Hour)},
		{Number: 2, Title: "Bravo", Author: "bob", State: domain.PRStateOpen, UpdatedAt: base.Add(2 * time.Hour), CreatedAt: base.Add(-4 * time.Hour)},
		{Number: 4, Title: "Delta", Author: "carol", State: domain.PRStateOpen, UpdatedAt: base.Add(4 * time.Hour), CreatedAt: base.Add(-3 * time.Hour)},
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
	if m.sortAsc {
		t.Error("default sort direction should be descending")
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
	if m.sortAsc {
		t.Error("after 1st cycle sort should default to descending")
	}

	m.cycleSort()
	if m.sortField != "created" {
		t.Errorf("after 2nd press = %q, want %q", m.sortField, "created")
	}
	if !m.sortAsc {
		t.Error("after 2nd press sort should toggle to ascending")
	}

	m.cycleSort()
	if m.sortField != "number" {
		t.Errorf("after 3rd press = %q, want %q", m.sortField, "number")
	}
	if m.sortAsc {
		t.Error("after 3rd press sort should reset to descending")
	}
}

func TestPRListSortApplyFilter(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetPRs(testPRsForSort())

	tests := []struct {
		name  string
		field string
		asc   bool
		want  []int
	}{
		{name: "updated desc", field: "updated", asc: false, want: []int{3, 4, 5, 2, 1}},
		{name: "updated asc", field: "updated", asc: true, want: []int{1, 2, 5, 4, 3}},
		{name: "created desc", field: "created", asc: false, want: []int{3, 5, 4, 2, 1}},
		{name: "created asc", field: "created", asc: true, want: []int{1, 2, 4, 5, 3}},
		{name: "number asc", field: "number", asc: true, want: []int{1, 2, 3, 4, 5}},
		{name: "number desc", field: "number", asc: false, want: []int{5, 4, 3, 2, 1}},
		{name: "title asc", field: "title", asc: true, want: []int{1, 2, 4, 5, 3}},
		{name: "title desc", field: "title", asc: false, want: []int{3, 5, 4, 2, 1}},
		{name: "author asc", field: "author", asc: true, want: []int{1, 2, 4, 5, 3}},
		{name: "author desc", field: "author", asc: false, want: []int{3, 5, 4, 2, 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m.sortField = tt.field
			m.sortAsc = tt.asc
			m.searchQuery = ""
			m.applyFilter()

			got := make([]int, 0, len(m.filtered))
			for _, pr := range m.filtered {
				got = append(got, pr.Number)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("order = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPRListSortIndicator(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetSize(120, 30)
	m.SetPRs(testPRsForSort())

	m.sortField = "title"
	m.sortAsc = true
	header := m.renderHeaderRow()

	if !strings.Contains(header, "Title▲") {
		t.Errorf("header missing sort indicator for title: %q", header)
	}
	if strings.Contains(header, "Author▲") {
		t.Errorf("unexpected indicator on non-active column: %q", header)
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

func TestPRListQuickFilterMyPRs(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetSize(120, 30)
	m.SetUsername("indrasvat")
	m.SetPRs(testPRs())

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	if len(m.filtered) != 2 {
		t.Errorf("filtered len = %d, want 2", len(m.filtered))
	}
	for _, pr := range m.filtered {
		if pr.Author != "indrasvat" {
			t.Errorf("unexpected author %q in My PRs filter", pr.Author)
		}
	}
	if cmd == nil {
		t.Fatal("expected filter change command")
	}
	if msg, ok := cmd().(PRListFilterMsg); !ok {
		t.Errorf("expected PRListFilterMsg, got %T", msg)
	} else if msg.Label != "My PRs" {
		t.Errorf("filter label = %q, want My PRs", msg.Label)
	}

	// Toggle off.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	if len(m.filtered) != 5 {
		t.Errorf("filtered len after toggle off = %d, want 5", len(m.filtered))
	}
}

func TestPRListQuickFilterNeedsReview(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetSize(120, 30)
	m.SetUsername("indrasvat")
	m.SetPRs(testPRs())

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if len(m.filtered) != 1 {
		t.Errorf("filtered len = %d, want 1", len(m.filtered))
	}
	if len(m.filtered) == 1 && m.filtered[0].Author == "indrasvat" {
		t.Error("needs review filter should exclude user's own PRs")
	}
	if cmd == nil {
		t.Fatal("expected filter change command")
	}
	if msg, ok := cmd().(PRListFilterMsg); !ok {
		t.Errorf("expected PRListFilterMsg, got %T", msg)
	} else if msg.Label != "Needs Review" {
		t.Errorf("filter label = %q, want Needs Review", msg.Label)
	}
}

func TestPRListOpenFilterKey(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetSize(120, 30)
	m.SetPRs(testPRs())

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	if cmd == nil {
		t.Fatal("expected command for filter key")
	}
	if _, ok := cmd().(OpenFilterMsg); !ok {
		t.Fatalf("expected OpenFilterMsg, got %T", cmd())
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
