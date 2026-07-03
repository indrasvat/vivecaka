package views

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/indrasvat/vivecaka/internal/domain"
)

func testInboxPRs() []InboxPR {
	now := time.Now()
	return []InboxPR{
		{
			PR:      domain.PR{Number: 10, Title: "Fix auth", Author: "alice", CI: domain.CIFail, Review: domain.ReviewStatus{State: domain.ReviewPending, Approved: 0, Total: 1}, UpdatedAt: now.Add(-1 * time.Hour)},
			Repo:    domain.RepoRef{Owner: "acme", Name: "webapp"},
			Sources: []domain.InboxSource{domain.InboxSourceAttention},
		},
		{
			PR:   domain.PR{Number: 20, Title: "Add caching", Author: "indrasvat", CI: domain.CIPass, Review: domain.ReviewStatus{State: domain.ReviewApproved, Approved: 2, Total: 2}, UpdatedAt: now.Add(-3 * time.Hour)},
			Repo: domain.RepoRef{Owner: "acme", Name: "webapp"},
		},
		{
			PR:      domain.PR{Number: 5, Title: "New theme", Author: "bob", CI: domain.CIPending, Review: domain.ReviewStatus{State: domain.ReviewPending, Approved: 0, Total: 1}, UpdatedAt: now.Add(-48 * time.Hour)},
			Repo:    domain.RepoRef{Owner: "indrasvat", Name: "vivecaka"},
			Sources: []domain.InboxSource{domain.InboxSourceAttention},
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

func TestInboxTabAssignedOnlyShowsAssignedSource(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetUsername("indrasvat")
	m.SetPRs([]InboxPR{
		{
			PR:      domain.PR{Number: 1, Title: "review", Author: "alice", Review: domain.ReviewStatus{State: domain.ReviewPending}},
			Repo:    domain.RepoRef{Owner: "owner", Name: "review"},
			Sources: []domain.InboxSource{domain.InboxSourceAttention},
		},
		{
			PR:      domain.PR{Number: 2, Title: "assigned", Author: "bob"},
			Repo:    domain.RepoRef{Owner: "owner", Name: "assigned"},
			Sources: []domain.InboxSource{domain.InboxSourceAssigned},
		},
	})

	m.tab = InboxAssigned
	m.applyFilter()

	require.Len(t, m.filtered, 1)
	assert.Equal(t, 2, m.filtered[0].Number)
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

func TestInboxRowsUseStableVisualColumns(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	m.SetSize(128, 34)
	m.SetUsername("indrasvat")
	m.SetPRs([]InboxPR{
		{
			PR:      domain.PR{Number: 1, Title: "short", Author: "alice", CI: domain.CIFail, Review: domain.ReviewStatus{State: domain.ReviewPending}, UpdatedAt: time.Now()},
			Repo:    domain.RepoRef{Owner: "owner", Name: "repo"},
			Sources: []domain.InboxSource{domain.InboxSourceHome, domain.InboxSourceAttention, domain.InboxSourceFavorite, domain.InboxSourceOwned},
			Reason:  "review",
		},
		{
			PR:      domain.PR{Number: 2, Title: "a much longer title that should still keep author and ci aligned", Author: "indrasvat", CI: domain.CINone, UpdatedAt: time.Now()},
			Repo:    domain.RepoRef{Owner: "owner", Name: "other"},
			Sources: []domain.InboxSource{domain.InboxSourceOwned},
			Reason:  "watch",
		},
	})

	first := m.renderRow(0, m.filtered[0])
	second := m.renderRow(1, m.filtered[1])
	firstPlain := ansi.Strip(first)
	secondPlain := ansi.Strip(second)

	assert.Equal(t, 128, lipgloss.Width(first))
	assert.Equal(t, 128, lipgloss.Width(second))
	assert.Contains(t, firstPlain, "◆★●⌂", "source glyphs must render in canonical slots")
	assert.Contains(t, secondPlain, "●   ", "short source clusters must be padded before repo")
	assert.Contains(t, firstPlain, "@alice")
	assert.Contains(t, secondPlain, "@indrasvat")
	assert.Equal(t, visualColumn(firstPlain, "@alice"), visualColumn(secondPlain, "@indrasvat"), "author column should not zig-zag")
}

func TestInboxSelectedRowShowsActionCueOnlyOnSelectedRow(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	m.SetSize(128, 34)
	m.SetUsername("indrasvat")
	m.SetPRs([]InboxPR{
		{
			PR:         domain.PR{Number: 1, Title: "needs review", Author: "alice", CI: domain.CIFail, Review: domain.ReviewStatus{State: domain.ReviewPending}, UpdatedAt: time.Now().Add(-9 * 24 * time.Hour)},
			Repo:       domain.RepoRef{Owner: "owner", Name: "repo"},
			Sources:    []domain.InboxSource{domain.InboxSourceAttention, domain.InboxSourceFavorite},
			LocalState: domain.InboxLocalReady,
			Reason:     "review",
		},
		{
			PR:      domain.PR{Number: 2, Title: "approved", Author: "indrasvat", CI: domain.CIPass, Review: domain.ReviewStatus{State: domain.ReviewApproved}, UpdatedAt: time.Now()},
			Repo:    domain.RepoRef{Owner: "owner", Name: "other"},
			Sources: []domain.InboxSource{domain.InboxSourceOwned},
			Reason:  "watch",
		},
	})

	selected := ansi.Strip(m.renderRow(0, m.filtered[0]))
	unselected := ansi.Strip(m.renderRow(1, m.filtered[1]))
	cue := strings.SplitN(selected, "\n", 2)[1]

	assert.Contains(t, selected, "\n", "selected row should get one secondary cue line")
	assert.Contains(t, cue, "◆")
	assert.NotContains(t, cue, "✕")
	assert.Contains(t, cue, "★")
	assert.Contains(t, cue, "↵")
	assert.NotContains(t, cue, "review")
	assert.NotContains(t, cue, "fix CI")
	assert.NotContains(t, cue, "stale")
	assert.NotContains(t, cue, "owned +")
	assert.NotContains(t, cue, "candidate")
	assert.NotContains(t, unselected, "\n", "unselected rows should stay compact")
	assert.Equal(t, testStyles().Theme.Primary, inboxActionCueColor("◆", testStyles().Theme))
	assert.Equal(t, testStyles().Theme.Warning, inboxActionCueColor("★", testStyles().Theme))
	assert.Equal(t, testStyles().Theme.Success, inboxActionCueColor("↵", testStyles().Theme))
}

func TestInboxSelectedRowSuppressesCIOnlyCue(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	m.SetSize(128, 34)
	m.SetUsername("indrasvat")
	m.SetPRs([]InboxPR{{
		PR:      domain.PR{Number: 1, Title: "ci failed", Author: "indrasvat", CI: domain.CIFail, UpdatedAt: time.Now()},
		Repo:    domain.RepoRef{Owner: "owner", Name: "repo"},
		Sources: []domain.InboxSource{domain.InboxSourceOwned},
	}})

	selected := ansi.Strip(m.renderRow(0, m.filtered[0]))

	assert.NotContains(t, selected, "\n", "CI is already visible in the CI column")
	assert.Contains(t, selected, "✗")
}

func TestInboxSelectedRowShowsLazyInsightAtoms(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	m.SetSize(128, 34)
	m.SetUsername("indrasvat")
	m.SetPRs([]InboxPR{{
		PR:      domain.PR{Number: 7, Title: "changed since review", Author: "alice", Review: domain.ReviewStatus{State: domain.ReviewPending}, UpdatedAt: time.Now()},
		Repo:    domain.RepoRef{Owner: "owner", Name: "repo"},
		Sources: []domain.InboxSource{domain.InboxSourceAttention},
	}})
	m.ApplyInsight(InboxInsightLoadedMsg{
		Key: "owner/repo#7",
		Insight: domain.InboxInsight{
			Repo:              domain.RepoRef{Owner: "owner", Name: "repo"},
			Number:            7,
			CommitDelta:       3,
			FileDelta:         2,
			UnresolvedThreads: 1,
		},
	})

	selected := ansi.Strip(m.renderRow(0, m.filtered[0]))
	cue := strings.SplitN(selected, "\n", 2)[1]

	assert.Contains(t, cue, "↳")
	assert.Contains(t, cue, "◆")
	assert.Contains(t, cue, "3⎇")
	assert.Contains(t, cue, "2▤")
	assert.Contains(t, cue, "1◌")
	assert.NotContains(t, cue, "commits")
	assert.NotContains(t, cue, "files")
	assert.NotContains(t, cue, "threads")
}

func TestInboxSelectedRowSuppressesAgeOnlyStaleCue(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	m.SetSize(128, 34)
	m.SetUsername("indrasvat")
	m.SetPRs([]InboxPR{{
		PR:      domain.PR{Number: 1, Title: "old but otherwise quiet", Author: "indrasvat", CI: domain.CINone, UpdatedAt: time.Now().Add(-90 * 24 * time.Hour)},
		Repo:    domain.RepoRef{Owner: "owner", Name: "repo"},
		Sources: []domain.InboxSource{domain.InboxSourceOwned},
	}})

	selected := ansi.Strip(m.renderRow(0, m.filtered[0]))

	assert.NotContains(t, selected, "\n", "age-only stale rows should not render redundant secondary cues")
	assert.NotContains(t, selected, "stale")
}

func TestInboxFooterSignalsAreGlyphOnlyAndIssueScoped(t *testing.T) {
	m := NewInboxModel(testStyles(), testKeys())
	m.sources = []domain.InboxSourceStatus{
		{Source: domain.InboxSourceAttention, Label: "review", Count: 2},
		{Source: domain.InboxSourceFavorite, Label: "favs", Count: 3},
		{Source: domain.InboxSourceOwned, Label: "owned", Count: 4},
	}

	assert.Empty(t, ansi.Strip(m.FooterSignals()))

	m.rate.SearchRemaining = 3
	assert.Equal(t, "◷", ansi.Strip(m.FooterSignals()))
	assert.NotContains(t, ansi.Strip(m.FooterSignals()), "gh")

	m.sources = append(m.sources, domain.InboxSourceStatus{Source: domain.InboxSourceAssigned, Label: "assigned", Err: "timeout"})
	m.staged = []InboxPR{{PR: domain.PR{Number: 1}}}
	m.cachedAt = time.Now()
	signals := ansi.Strip(m.FooterSignals())

	assert.Contains(t, signals, "⚠")
	assert.Contains(t, signals, "◷")
	assert.Contains(t, signals, "◌")
	assert.Contains(t, signals, "◆")
	assert.NotContains(t, signals, "partial")
	assert.NotContains(t, signals, "limited")
	assert.NotContains(t, signals, "cache")
}

func visualColumn(line, token string) int {
	idx := strings.Index(line, token)
	if idx < 0 {
		return -1
	}
	return lipgloss.Width(line[:idx])
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
