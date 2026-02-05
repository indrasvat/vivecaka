package views

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/muesli/termenv"
)

func testDiff() *domain.Diff {
	return &domain.Diff{
		Files: []domain.FileDiff{
			{
				Path:    "internal/plugin/registry.go",
				OldPath: "internal/plugin/registry.go",
				Hunks: []domain.Hunk{
					{
						Header: "@@ -10,5 +10,8 @@",
						Lines: []domain.DiffLine{
							{Type: domain.DiffContext, Content: "import (", OldNum: 10, NewNum: 10},
							{Type: domain.DiffDelete, Content: "\t\"sync\"", OldNum: 11},
							{Type: domain.DiffAdd, Content: "\t\"sync\"", NewNum: 11},
							{Type: domain.DiffAdd, Content: "\t\"fmt\"", NewNum: 12},
							{Type: domain.DiffContext, Content: ")", OldNum: 12, NewNum: 13},
						},
					},
				},
			},
			{
				Path:    "internal/plugin/hooks.go",
				OldPath: "internal/plugin/hooks.go",
				Hunks: []domain.Hunk{
					{
						Header: "@@ -1,3 +1,4 @@",
						Lines: []domain.DiffLine{
							{Type: domain.DiffContext, Content: "package plugin", OldNum: 1, NewNum: 1},
							{Type: domain.DiffAdd, Content: "", NewNum: 2},
							{Type: domain.DiffContext, Content: "import (", OldNum: 2, NewNum: 3},
						},
					},
				},
			},
		},
	}
}

func TestNewDiffViewModel(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	if !m.loading {
		t.Error("new model should be in loading state")
	}
	if m.diff != nil {
		t.Error("diff should be nil initially")
	}
}

func TestDiffSetSize(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	if m.width != 120 || m.height != 40 {
		t.Errorf("size = %dx%d, want 120x40", m.width, m.height)
	}
}

func TestDiffSetDiff(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	d := testDiff()
	m.SetDiff(d)

	if m.loading {
		t.Error("loading should be false after SetDiff")
	}
	if m.diff != d {
		t.Error("diff should be set")
	}
	if m.fileIdx != 0 {
		t.Errorf("fileIdx = %d, want 0", m.fileIdx)
	}
	if m.scrollY != 0 {
		t.Errorf("scrollY = %d, want 0", m.scrollY)
	}
}

func TestDiffLoadedMsg(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)

	d := testDiff()
	m.Update(DiffLoadedMsg{Diff: d})

	if m.loading {
		t.Error("should not be loading after DiffLoadedMsg")
	}
	if m.diff != d {
		t.Error("diff should be set from message")
	}
}

func TestDiffScrollDown(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())

	down := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	m.Update(down)
	if m.scrollY != 1 {
		t.Errorf("scrollY after j = %d, want 1", m.scrollY)
	}
}

func TestDiffScrollUp(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())

	m.scrollY = 3
	up := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	m.Update(up)
	if m.scrollY != 2 {
		t.Errorf("scrollY after k = %d, want 2", m.scrollY)
	}

	// Can't go below 0.
	m.scrollY = 0
	m.Update(up)
	if m.scrollY != 0 {
		t.Errorf("scrollY shouldn't go below 0, got %d", m.scrollY)
	}
}

func TestDiffHalfPageScroll(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 20)
	m.SetDiff(testDiff())

	// Half page down.
	ctrlD := tea.KeyMsg{Type: tea.KeyCtrlD}
	m.Update(ctrlD)
	if m.scrollY != 10 {
		t.Errorf("scrollY after Ctrl+d = %d, want 10", m.scrollY)
	}

	// Half page up.
	ctrlU := tea.KeyMsg{Type: tea.KeyCtrlU}
	m.Update(ctrlU)
	if m.scrollY != 0 {
		t.Errorf("scrollY after Ctrl+u = %d, want 0", m.scrollY)
	}

	// Half page up at 0 stays at 0.
	m.Update(ctrlU)
	if m.scrollY != 0 {
		t.Errorf("scrollY shouldn't go below 0 with Ctrl+u, got %d", m.scrollY)
	}
}

func TestDiffFileNavigation(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())

	if m.fileIdx != 0 {
		t.Fatalf("initial fileIdx = %d, want 0", m.fileIdx)
	}

	// Tab to next file.
	tab := tea.KeyMsg{Type: tea.KeyTab}
	m.Update(tab)
	if m.fileIdx != 1 {
		t.Errorf("fileIdx after tab = %d, want 1", m.fileIdx)
	}

	// Tab at last file stays.
	m.Update(tab)
	if m.fileIdx != 1 {
		t.Errorf("fileIdx after tab at end = %d, want 1", m.fileIdx)
	}

	// Shift-tab back.
	shiftTab := tea.KeyMsg{Type: tea.KeyShiftTab}
	m.Update(shiftTab)
	if m.fileIdx != 0 {
		t.Errorf("fileIdx after shift-tab = %d, want 0", m.fileIdx)
	}

	// Shift-tab at first file stays.
	m.Update(shiftTab)
	if m.fileIdx != 0 {
		t.Errorf("fileIdx after shift-tab at start = %d, want 0", m.fileIdx)
	}
}

func TestDiffFileNavResetsScroll(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())

	m.scrollY = 5
	tab := tea.KeyMsg{Type: tea.KeyTab}
	m.Update(tab)
	if m.scrollY != 0 {
		t.Errorf("scrollY should reset on file switch, got %d", m.scrollY)
	}
}

func TestDiffSearch(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())

	// Enter search mode.
	slash := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	m.Update(slash)
	if !m.searching {
		t.Error("should be in search mode after /")
	}
	if m.searchQuery != "" {
		t.Errorf("searchQuery should be empty, got %q", m.searchQuery)
	}

	// Type query.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if m.searchQuery != "sync" {
		t.Errorf("searchQuery = %q, want %q", m.searchQuery, "sync")
	}
	if len(m.searchMatches) != 2 {
		t.Errorf("searchMatches = %d, want 2", len(m.searchMatches))
	}
	if m.currentMatch < 0 {
		t.Error("currentMatch should be set when matches exist")
	}

	// Backspace.
	m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.searchQuery != "syn" {
		t.Errorf("searchQuery after backspace = %q, want %q", m.searchQuery, "syn")
	}

	// Enter confirms search.
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.searching {
		t.Error("should exit search mode on Enter")
	}
	if m.searchQuery != "syn" {
		t.Errorf("query should be preserved after Enter, got %q", m.searchQuery)
	}
}

func TestDiffSearchEscape(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())

	// Enter search, type, then escape.
	slash := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	m.Update(slash)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if m.searching {
		t.Error("should exit search mode on Escape")
	}
	if m.searchQuery != "" {
		t.Errorf("query should be cleared on Escape, got %q", m.searchQuery)
	}
	if len(m.searchMatches) != 0 {
		t.Errorf("searchMatches should be cleared on Escape, got %d", len(m.searchMatches))
	}
	if m.currentMatch != -1 {
		t.Errorf("currentMatch should reset on Escape, got %d", m.currentMatch)
	}
}

func TestDiffSearchBackspaceEmpty(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())

	slash := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	m.Update(slash)

	// Backspace on empty query is safe.
	m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.searchQuery != "" {
		t.Errorf("backspace on empty should stay empty, got %q", m.searchQuery)
	}
}

func TestDiffSearchNavigation(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())

	m.searchQuery = "sync"
	m.updateSearchMatches()
	if len(m.searchMatches) == 0 {
		t.Fatal("expected matches for sync")
	}

	first := m.currentMatch
	m.nextMatch()
	if m.currentMatch == first {
		t.Error("nextMatch should advance currentMatch")
	}
	m.prevMatch()
	if m.currentMatch != first {
		t.Errorf("prevMatch should return to first, got %d", m.currentMatch)
	}
}

func TestDiffApplyHighlights(t *testing.T) {
	origProfile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() {
		lipgloss.SetColorProfile(origProfile)
	})

	base := lipgloss.NewStyle()
	match := lipgloss.NewStyle().Background(lipgloss.Color("1"))
	text := "sync sync"
	matches := []searchMatch{
		{colStart: 0, colEnd: 4},
		{colStart: 5, colEnd: 9},
	}

	got := applyHighlights(text, matches, base, match)
	if !strings.Contains(got, "\x1b[") {
		t.Error("expected highlight ANSI sequence in output")
	}
	if !strings.Contains(got, "sync") {
		t.Error("expected highlighted text to contain sync")
	}
}

func TestDiffViewLoading(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(80, 24)

	view := m.View()
	if view == "" {
		t.Error("loading view should not be empty")
	}
}

func TestDiffViewNoFiles(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(80, 24)
	m.SetDiff(&domain.Diff{Files: nil})

	view := m.View()
	if view == "" {
		t.Error("no-files view should not be empty")
	}
}

func TestDiffViewWithData(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())

	view := m.View()
	if view == "" {
		t.Error("diff view with data should not be empty")
	}
}

func TestDiffViewSearchBar(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())
	m.searching = true
	m.searchQuery = "test"

	view := m.View()
	if view == "" {
		t.Error("view with search bar should not be empty")
	}
}

func TestDiffViewScrollClamp(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 5)
	m.SetDiff(testDiff())

	// Set scroll way past content.
	m.scrollY = 9999
	view := m.View()
	if view == "" {
		t.Error("view with clamped scroll should not be empty")
	}
}

func TestDiffViewSmallHeight(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(80, 2) // Very small.
	m.SetDiff(testDiff())

	view := m.View()
	if view == "" {
		t.Error("small height view should not be empty")
	}
}

func TestDiffFileBarTruncation(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(&domain.Diff{
		Files: []domain.FileDiff{
			{Path: "this/is/a/very/long/path/to/a/file.go"},
			{Path: "short.go"},
		},
	})

	// Should not panic even with long paths.
	view := m.View()
	if view == "" {
		t.Error("view with long paths should not be empty")
	}
}
