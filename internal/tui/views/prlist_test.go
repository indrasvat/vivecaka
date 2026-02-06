package views

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/indrasvat/vivecaka/internal/tui/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	assert.True(t, m.loading, "new model should be in loading state")
	assert.Equal(t, "updated", m.sortField)
	assert.False(t, m.sortAsc, "default sort direction should be descending")
}

func TestSetPRs(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetPRs(testPRs())

	assert.False(t, m.loading, "loading should be false after SetPRs")
	assert.Len(t, m.filtered, 5)
}

func TestSelectedPR(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetPRs(testPRs())

	pr := m.SelectedPR()
	require.NotNil(t, pr)
	assert.Equal(t, 142, pr.Number)
}

func TestSelectedPREmpty(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetPRs(nil)

	pr := m.SelectedPR()
	assert.Nil(t, pr, "SelectedPR() should be nil for empty list")
}

func TestNavigationDown(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetSize(120, 30)
	m.SetPRs(testPRs())

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	m.Update(msg)

	assert.Equal(t, 1, m.cursor)
}

func TestNavigationUp(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetSize(120, 30)
	m.SetPRs(testPRs())
	m.cursor = 2

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	m.Update(msg)

	assert.Equal(t, 1, m.cursor)
}

func TestNavigationBounds(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetSize(120, 30)
	m.SetPRs(testPRs())

	// Already at top, going up should stay.
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	m.Update(msg)
	assert.Equal(t, 0, m.cursor)

	// Go to bottom.
	m.cursor = 4
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	m.Update(msg)
	assert.Equal(t, 4, m.cursor)
}

func TestTopBottom(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetSize(120, 30)
	m.SetPRs(testPRs())
	m.cursor = 2

	// G = bottom
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
	m.Update(msg)
	assert.Equal(t, 4, m.cursor)

	// g = top
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
	m.Update(msg)
	assert.Equal(t, 0, m.cursor)
}

func TestSearchFilter(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetSize(120, 30)
	m.SetPRs(testPRs())

	m.searchQuery = "alice"
	m.applyFilter()

	require.Len(t, m.filtered, 1)
	assert.Equal(t, "alice", m.filtered[0].Author)
}

func TestSearchFilterByTitle(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetPRs(testPRs())

	m.searchQuery = "plugin"
	m.applyFilter()

	assert.Len(t, m.filtered, 1)
}

func TestSearchFilterCaseInsensitive(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetPRs(testPRs())

	m.searchQuery = "ALICE"
	m.applyFilter()

	assert.Len(t, m.filtered, 1)
}

func TestCycleSort(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetPRs(testPRs())

	require.Equal(t, "updated", m.sortField)

	m.cycleSort()
	assert.Equal(t, "created", m.sortField)
	assert.False(t, m.sortAsc, "after 1st cycle sort should default to descending")

	m.cycleSort()
	assert.Equal(t, "created", m.sortField)
	assert.True(t, m.sortAsc, "after 2nd press sort should toggle to ascending")

	m.cycleSort()
	assert.Equal(t, "number", m.sortField)
	assert.False(t, m.sortAsc, "after 3rd press sort should reset to descending")
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

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPRListSortIndicator(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetSize(120, 30)
	m.SetPRs(testPRsForSort())

	m.sortField = "title"
	m.sortAsc = true
	header := m.renderColumnHeaders()

	assert.Contains(t, header, "Title▲")
	assert.NotContains(t, header, "Author▲")
}

func TestCurrentBranch(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetPRs(testPRs())
	m.SetCurrentBranch("feat/plugins")

	assert.Equal(t, "feat/plugins", m.currentBranch)
}

func TestPRListQuickFilterMyPRs(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetSize(120, 30)
	m.SetUsername("indrasvat")
	m.SetPRs(testPRs())

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	assert.Len(t, m.filtered, 2)
	for _, pr := range m.filtered {
		assert.Equal(t, "indrasvat", pr.Author, "unexpected author in My PRs filter")
	}
	require.NotNil(t, cmd, "expected filter change command")
	msg := cmd()
	filterMsg, ok := msg.(PRListFilterMsg)
	require.True(t, ok, "expected PRListFilterMsg, got %T", msg)
	assert.Equal(t, "My PRs", filterMsg.Label)

	// Toggle off.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	assert.Len(t, m.filtered, 5)
}

func TestPRListQuickFilterNeedsReview(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetSize(120, 30)
	m.SetUsername("indrasvat")
	m.SetPRs(testPRs())

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	require.Len(t, m.filtered, 1)
	if len(m.filtered) == 1 {
		assert.NotEqual(t, "indrasvat", m.filtered[0].Author, "needs review filter should exclude user's own PRs")
	}
	require.NotNil(t, cmd, "expected filter change command")
	msg := cmd()
	filterMsg, ok := msg.(PRListFilterMsg)
	require.True(t, ok, "expected PRListFilterMsg, got %T", msg)
	assert.Equal(t, "Needs Review", filterMsg.Label)
}

func TestPRListOpenFilterKey(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetSize(120, 30)
	m.SetPRs(testPRs())

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	require.NotNil(t, cmd, "expected command for filter key")
	_, ok := cmd().(OpenFilterMsg)
	require.True(t, ok, "expected OpenFilterMsg, got %T", cmd())
}

func TestViewLoading(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetSize(80, 24)

	view := m.View()
	assert.NotEmpty(t, view, "loading view should not be empty")
}

func TestViewEmpty(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetSize(80, 24)
	m.SetPRs(nil)

	view := m.View()
	assert.NotEmpty(t, view, "empty view should not be empty string")
}

func TestViewWithData(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetSize(120, 30)
	m.SetPRs(testPRs())

	view := m.View()
	assert.NotEmpty(t, view, "view with data should not be empty")
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
		got := ciIcon(tt.status)
		assert.Equal(t, tt.want, got)
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
		got := reviewText(tt.status)
		assert.Equal(t, tt.want, got)
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
		got := relativeTime(tt.t)
		assert.Equal(t, tt.want, got)
	}
}
