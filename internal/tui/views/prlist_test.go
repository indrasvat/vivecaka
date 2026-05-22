package views

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

func TestPRListPaginationAccessorsAndAppend(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetSize(80, 8)
	m.SetPerPage(2)
	m.SetPRs(testPRs()[:2])

	assert.False(t, m.IsLoading())
	assert.True(t, m.HasPRs())
	assert.Equal(t, 1, m.CurrentPage())
	assert.Equal(t, 2, m.PerPage())
	assert.True(t, m.HasMore())
	assert.False(t, m.IsLoadingMore())
	assert.Len(t, m.FilteredPRs(), 2)
	assert.Equal(t, 2, m.TotalPRs())

	cmd := m.SetLoadingMore(2)
	assert.NotNil(t, cmd)
	assert.True(t, m.IsLoadingMore())
	assert.Equal(t, 2, m.CurrentPage())

	m.AppendPRs(testPRs()[2:4], false)
	assert.False(t, m.IsLoadingMore())
	assert.False(t, m.HasMore())
	assert.Equal(t, 4, m.TotalPRs())
	assert.Len(t, m.FilteredPRs(), 4)

	m.SetPerPage(0)
	assert.Equal(t, 2, m.PerPage(), "invalid page size should be ignored")
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

func TestPRListSearchModeEditsQueryAndCanCancel(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	m.SetPRs(testPRs())

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	assert.True(t, m.searching)

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("plugin")})
	require.Len(t, m.filtered, 1)
	assert.Equal(t, "plugin", m.searchQuery)

	m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	assert.Equal(t, "plugi", m.searchQuery)

	m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	assert.False(t, m.searching)
	assert.Empty(t, m.searchQuery)
	assert.Len(t, m.filtered, len(testPRs()))

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("alice")})
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.False(t, m.searching)
	require.Len(t, m.filtered, 1)
	assert.Equal(t, "alice", m.filtered[0].Author)
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

func TestPRListSelectionModeBatchActions(t *testing.T) {
	prs := testPRs()
	for i := range prs {
		prs[i].URL = "https://example.test/pr/" + string(rune('a'+i))
	}

	m := NewPRListModel(testStyles(), testKeys())
	m.SetPRs(prs)

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	assert.True(t, m.IsSelectionMode())
	assert.Equal(t, 0, m.SelectionCount())

	m.Update(tea.KeyMsg{Type: tea.KeySpace})
	assert.Equal(t, 1, m.SelectionCount())
	assert.Equal(t, []string{prs[0].URL}, m.selectedURLs())

	msg := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})()
	copyMsg := msg.(BatchCopyURLsMsg)
	assert.Equal(t, []string{prs[0].URL}, copyMsg.URLs)

	msg = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})()
	openMsg := msg.(BatchOpenBrowserMsg)
	assert.Equal(t, []string{prs[0].URL}, openMsg.URLs)

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	assert.Equal(t, len(prs), m.SelectionCount())

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	assert.False(t, m.IsSelectionMode())
	assert.Equal(t, 0, m.SelectionCount())
}

func TestPRListPanelFilterRequiresAllCriteria(t *testing.T) {
	m := NewPRListModel(testStyles(), testKeys())
	prs := testPRs()
	prs[0].Labels = []string{"Backend", "Security"}
	prs[1].Labels = []string{"backend"}
	prs[1].Draft = true
	m.SetPRs(prs)

	m.SetFilter(domain.ListOpts{
		State:  domain.PRStateOpen,
		Author: "INDRA",
		Labels: []string{"security"},
		CI:     domain.CIPass,
		Review: domain.ReviewApproved,
		Draft:  domain.DraftExclude,
	})
	require.Len(t, m.FilteredPRs(), 1)
	assert.Equal(t, 142, m.FilteredPRs()[0].Number)
	assert.Equal(t, "Filtered", m.FilterLabel())

	m.SetFilter(domain.ListOpts{Draft: domain.DraftOnly})
	require.Len(t, m.FilteredPRs(), 2)
	for _, pr := range m.FilteredPRs() {
		assert.True(t, pr.Draft)
	}

	m.SetFilter(domain.ListOpts{State: domain.PRStateClosed})
	assert.Empty(t, m.FilteredPRs())

	assert.True(t, hasLabel([]string{"Backend"}, "backend"))
	assert.False(t, hasLabel([]string{"Backend"}, "frontend"))
	assert.Equal(t, "", filterLabelFromOpts(domain.ListOpts{State: domain.PRStateOpen}))
	assert.Equal(t, "Filtered", filterLabelFromOpts(domain.ListOpts{State: domain.PRStateMerged}))
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
