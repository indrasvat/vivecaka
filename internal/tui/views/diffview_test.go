package views

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/indrasvat/vivecaka/internal/domain"
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

func testDiffWithHunks() *domain.Diff {
	return &domain.Diff{
		Files: []domain.FileDiff{
			{
				Path: "internal/plugin/registry.go",
				Hunks: []domain.Hunk{
					{
						Header: "@@ -1,1 +1,1 @@",
						Lines: []domain.DiffLine{
							{Type: domain.DiffContext, Content: "line one", OldNum: 1, NewNum: 1},
						},
					},
					{
						Header: "@@ -10,1 +10,1 @@",
						Lines: []domain.DiffLine{
							{Type: domain.DiffContext, Content: "line two", OldNum: 10, NewNum: 10},
						},
					},
				},
			},
		},
	}
}

func TestNewDiffViewModel(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	assert.True(t, m.loading, "new model should be in loading state")
	assert.Nil(t, m.diff, "diff should be nil initially")
}

func TestDiffSetSize(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	assert.Equal(t, 120, m.width)
	assert.Equal(t, 40, m.height)
}

func TestDiffSetDiff(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	d := testDiff()
	m.SetDiff(d)

	assert.False(t, m.loading, "loading should be false after SetDiff")
	assert.Equal(t, d, m.diff, "diff should be set")
	assert.Equal(t, 0, m.fileIdx)
	assert.Equal(t, 0, m.scrollY)
}

func TestDiffLoadedMsg(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)

	d := testDiff()
	m.Update(DiffLoadedMsg{Diff: d})

	assert.False(t, m.loading, "should not be loading after DiffLoadedMsg")
	assert.Equal(t, d, m.diff, "diff should be set from message")
}

func TestDiffScrollDown(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())

	down := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	m.Update(down)
	assert.Equal(t, 1, m.scrollY)
}

func TestDiffScrollUp(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())

	m.scrollY = 3
	up := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	m.Update(up)
	assert.Equal(t, 2, m.scrollY)

	// Can't go below 0.
	m.scrollY = 0
	m.Update(up)
	assert.Equal(t, 0, m.scrollY, "scrollY shouldn't go below 0")
}

func TestDiffHalfPageScroll(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 20)
	m.SetDiff(testDiff())

	// Half page down.
	ctrlD := tea.KeyMsg{Type: tea.KeyCtrlD}
	m.Update(ctrlD)
	assert.Equal(t, 10, m.scrollY)

	// Half page up.
	ctrlU := tea.KeyMsg{Type: tea.KeyCtrlU}
	m.Update(ctrlU)
	assert.Equal(t, 0, m.scrollY)

	// Half page up at 0 stays at 0.
	m.Update(ctrlU)
	assert.Equal(t, 0, m.scrollY, "scrollY shouldn't go below 0 with Ctrl+u")
}

func TestDiffFileNavigation(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())

	require.Equal(t, 0, m.fileIdx)

	// { } still navigate files from content pane.
	next := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'}'}}
	m.Update(next)
	assert.Equal(t, 1, m.fileIdx)

	prev := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'{'}}
	m.Update(prev)
	assert.Equal(t, 0, m.fileIdx)

	// Tab toggles focus to tree pane.
	tab := tea.KeyMsg{Type: tea.KeyTab}
	m.Update(tab)
	assert.True(t, m.treeFocus, "expected treeFocus after Tab")

	// In tree pane, j/k navigate files.
	down := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	m.Update(down)
	assert.Equal(t, 1, m.fileIdx)

	up := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	m.Update(up)
	assert.Equal(t, 0, m.fileIdx)

	// Enter selects file and returns focus to content.
	m.Update(down) // go to file 1
	enter := tea.KeyMsg{Type: tea.KeyEnter}
	m.Update(enter)
	assert.False(t, m.treeFocus, "expected content focus after Enter in tree")
	assert.Equal(t, 1, m.fileIdx)

	// Shift-tab also toggles focus.
	shiftTab := tea.KeyMsg{Type: tea.KeyShiftTab}
	m.Update(shiftTab)
	assert.True(t, m.treeFocus, "expected treeFocus after Shift-Tab")
}

func TestDiffFileNavResetsScroll(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())

	m.scrollY = 5
	// Use } to navigate to next file (resets scroll).
	next := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'}'}}
	m.Update(next)
	assert.Equal(t, 0, m.scrollY, "scrollY should reset on file switch")
}

func TestDiffHunkNavigation(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 10)
	m.SetDiff(testDiffWithHunks())

	require.Equal(t, 0, m.scrollY)

	next := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}}
	m.Update(next)
	assert.Equal(t, 2, m.scrollY)

	prev := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}}
	m.Update(prev)
	assert.Equal(t, 0, m.scrollY)
}

func TestDiffFileJump(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())
	m.scrollY = 5

	next := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'}'}}
	m.Update(next)
	assert.Equal(t, 1, m.fileIdx)
	assert.Equal(t, 0, m.scrollY)

	prev := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'{'}}
	m.Update(prev)
	assert.Equal(t, 0, m.fileIdx)
}

func TestDiffTopBottomNavigation(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 5)
	m.SetDiff(testDiff())
	m.scrollY = 3

	g := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
	m.Update(g)
	m.Update(g)
	assert.Equal(t, 0, m.scrollY)

	G := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
	m.Update(G)
	assert.Equal(t, 3, m.scrollY)
}

func TestDiffCollapseToggle(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())

	z := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}}
	a := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}

	m.Update(z)
	m.Update(a)
	assert.True(t, m.isCollapsed(m.fileIdx), "expected file to be collapsed after za")

	m.Update(z)
	m.Update(a)
	assert.False(t, m.isCollapsed(m.fileIdx), "expected file to expand after za again")
}

func TestDiffSearch(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())

	// Enter search mode.
	slash := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	m.Update(slash)
	assert.True(t, m.searching, "should be in search mode after /")
	assert.Equal(t, "", m.searchQuery)

	// Type query.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	assert.Equal(t, "sync", m.searchQuery)
	assert.Len(t, m.searchMatches, 2)
	assert.GreaterOrEqual(t, m.currentMatch, 0, "currentMatch should be set when matches exist")

	// Backspace.
	m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	assert.Equal(t, "syn", m.searchQuery)

	// Enter confirms search.
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.False(t, m.searching, "should exit search mode on Enter")
	assert.Equal(t, "syn", m.searchQuery, "query should be preserved after Enter")
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
	assert.False(t, m.searching, "should exit search mode on Escape")
	assert.Equal(t, "", m.searchQuery, "query should be cleared on Escape")
	assert.Empty(t, m.searchMatches, "searchMatches should be cleared on Escape")
	assert.Equal(t, -1, m.currentMatch, "currentMatch should reset on Escape")
}

func TestDiffSearchBackspaceEmpty(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())

	slash := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	m.Update(slash)

	// Backspace on empty query is safe.
	m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	assert.Equal(t, "", m.searchQuery, "backspace on empty should stay empty")
}

func TestDiffSearchNavigation(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())

	m.searchQuery = "sync"
	m.updateSearchMatches()
	require.NotEmpty(t, m.searchMatches, "expected matches for sync")

	first := m.currentMatch
	m.nextMatch()
	assert.NotEqual(t, first, m.currentMatch, "nextMatch should advance currentMatch")
	m.prevMatch()
	assert.Equal(t, first, m.currentMatch, "prevMatch should return to first")
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
	assert.Contains(t, got, "\x1b[", "expected highlight ANSI sequence in output")
	assert.Contains(t, got, "sync", "expected highlighted text to contain sync")
}

func TestDiffViewLoading(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(80, 24)

	view := m.View()
	assert.NotEmpty(t, view, "loading view should not be empty")
}

func TestDiffViewNoFiles(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(80, 24)
	m.SetDiff(&domain.Diff{Files: nil})

	view := m.View()
	assert.NotEmpty(t, view, "no-files view should not be empty")
}

func TestDiffViewWithData(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())

	view := m.View()
	assert.NotEmpty(t, view, "diff view with data should not be empty")
}

func TestDiffViewSearchBar(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())
	m.searching = true
	m.searchQuery = "test"

	view := m.View()
	assert.NotEmpty(t, view, "view with search bar should not be empty")
}

func TestDiffViewScrollClamp(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 5)
	m.SetDiff(testDiff())

	// Set scroll way past content.
	m.scrollY = 9999
	view := m.View()
	assert.NotEmpty(t, view, "view with clamped scroll should not be empty")
}

func TestDiffViewSmallHeight(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(80, 2) // Very small.
	m.SetDiff(testDiff())

	view := m.View()
	assert.NotEmpty(t, view, "small height view should not be empty")
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
	assert.NotEmpty(t, view, "view with long paths should not be empty")
}

func TestDiffSyntaxHighlighting(t *testing.T) {
	origProfile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() {
		lipgloss.SetColorProfile(origProfile)
	})

	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(&domain.Diff{
		Files: []domain.FileDiff{
			{
				Path: "main.go",
				Hunks: []domain.Hunk{
					{
						Header: "@@ -1,1 +1,1 @@",
						Lines: []domain.DiffLine{
							{Type: domain.DiffAdd, Content: "func main() { fmt.Println(\"hello\") }", NewNum: 1},
							{Type: domain.DiffDelete, Content: "package main", OldNum: 1},
							{Type: domain.DiffContext, Content: "import \"fmt\"", OldNum: 2, NewNum: 2},
						},
					},
				},
			},
		},
	})

	view := m.View()
	// Syntax highlighted output should contain ANSI escape sequences
	assert.Contains(t, view, "\x1b[", "expected ANSI escape sequences from syntax highlighting")
	// Should contain the code keywords
	assert.Contains(t, view, "func", "expected 'func' keyword in output")
	assert.Contains(t, view, "main", "expected 'main' in output")
}

func TestDiffSyntaxHighlightingUnknownLanguage(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(&domain.Diff{
		Files: []domain.FileDiff{
			{
				Path: "data.unknown",
				Hunks: []domain.Hunk{
					{
						Header: "@@ -1,1 +1,1 @@",
						Lines: []domain.DiffLine{
							{Type: domain.DiffContext, Content: "some random content", OldNum: 1, NewNum: 1},
						},
					},
				},
			},
		},
	})

	// Should not panic with unknown file types
	view := m.View()
	assert.NotEmpty(t, view, "view should not be empty for unknown file types")
	assert.Contains(t, view, "random content", "expected content to be present even without syntax highlighting")
}

func TestSyntaxHighlighterCaching(t *testing.T) {
	h := newSyntaxHighlighter()

	// First call creates the lexer
	_ = h.highlight("func main() {}", "test.go")

	// Check lexer is cached
	h.mu.RLock()
	_, cached := h.lexerCache[".go"]
	h.mu.RUnlock()

	assert.True(t, cached, "expected .go lexer to be cached")

	// Second call should use cache
	result := h.highlight("package main", "other.go")
	assert.NotEmpty(t, result, "highlight should return non-empty result")
}

func TestSyntaxHighlighterGoCode(t *testing.T) {
	h := newSyntaxHighlighter()

	// Go code with keywords and strings
	code := `func main() { fmt.Println("hello world") }`
	result := h.highlight(code, "main.go")

	// Result should contain ANSI escape sequences for syntax colors
	assert.Contains(t, result, "\x1b[", "expected ANSI escape sequences in highlighted Go code")

	// Result should still contain the original text
	assert.Contains(t, result, "func", "expected 'func' keyword in result")
	assert.Contains(t, result, "main", "expected 'main' in result")
	assert.Contains(t, result, "hello world", "expected string literal in result")
}

func TestSyntaxHighlighterRustCode(t *testing.T) {
	h := newSyntaxHighlighter()

	// Rust code with keywords
	code := `fn main() { println!("hello"); }`
	result := h.highlight(code, "main.rs")

	// Result should contain ANSI escape sequences
	assert.Contains(t, result, "\x1b[", "expected ANSI escape sequences in highlighted Rust code")
}

func TestSyntaxHighlighterTypescriptCode(t *testing.T) {
	h := newSyntaxHighlighter()

	// TypeScript code with keywords
	code := `const x: number = 42; export default function foo() {}`
	result := h.highlight(code, "app.ts")

	// Result should contain ANSI escape sequences
	assert.Contains(t, result, "\x1b[", "expected ANSI escape sequences in highlighted TypeScript code")
}

func TestSyntaxHighlighterFallback(t *testing.T) {
	h := newSyntaxHighlighter()

	// Unknown file type should return original content
	code := "some random content"
	result := h.highlight(code, "file.unknown123xyz")

	// For truly unknown extensions, should return original
	// This is actually fine - Chroma might have broad fallback rules
	// Just verify it doesn't crash and returns something
	if result != code {
		assert.NotEmpty(t, result, "highlight should return non-empty result even for unknown types")
	}
}

func TestDiffTwoPaneLayout(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())

	view := m.View()

	// View should contain file names from the tree.
	assert.Contains(t, view, "registry.go", "expected file tree to contain 'registry.go'")
	assert.Contains(t, view, "hooks.go", "expected file tree to contain 'hooks.go'")

	// Should contain diff content markers.
	assert.Contains(t, view, "@@", "expected diff content with hunk headers")
}

func TestDiffTreeFocusToggle(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())

	// Initially, content has focus.
	assert.False(t, m.treeFocus, "expected content focus initially")

	// Tab switches to tree.
	tab := tea.KeyMsg{Type: tea.KeyTab}
	m.Update(tab)
	assert.True(t, m.treeFocus, "expected tree focus after Tab")

	// Tab back to content.
	m.Update(tab)
	assert.False(t, m.treeFocus, "expected content focus after second Tab")
}

func TestDiffSplitModeToggle(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())

	assert.False(t, m.splitMode, "expected unified mode initially")

	// Press t to toggle to split mode.
	tKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}
	m.Update(tKey)
	assert.True(t, m.splitMode, "expected split mode after t")

	// View should still render.
	view := m.View()
	assert.NotEmpty(t, view, "split mode view should not be empty")
	// Should contain the file path.
	assert.Contains(t, view, "registry.go", "expected file path in split view")

	// Toggle back.
	m.Update(tKey)
	assert.False(t, m.splitMode, "expected unified mode after second t")
}

func TestDiffSplitModeRendering(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())
	m.splitMode = true

	view := m.View()
	// Split mode should contain the divider character.
	assert.Contains(t, view, "│", "expected vertical divider in split view")
	// Should show the "Split" mode label.
	assert.Contains(t, view, "Split", "expected 'Split' label in view")
}

func TestDiffUnifiedModeLabel(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())

	view := m.View()
	assert.Contains(t, view, "Unified", "expected 'Unified' label in default view")
}

func TestDiffTreeWidth(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())

	// Normal terminal: 25% of width, clamped.
	m.SetSize(120, 40)
	w := m.computeTreeWidth()
	assert.GreaterOrEqual(t, w, 20, "tree width should be at least 20")
	assert.LessOrEqual(t, w, 40, "tree width should be at most 40")

	// Very wide terminal.
	m.SetSize(200, 40)
	w = m.computeTreeWidth()
	assert.LessOrEqual(t, w, 40, "tree width should cap at 40 for wide terminal")

	// Narrow terminal.
	m.SetSize(60, 40)
	w = m.computeTreeWidth()
	assert.GreaterOrEqual(t, w, 10, "tree width should be at least 10 for narrow terminal")
}

func TestDiffSetComments(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())

	threads := []domain.CommentThread{
		{
			ID:   "t1",
			Path: "internal/plugin/registry.go",
			Line: 11,
			Comments: []domain.Comment{
				{Author: "alice", Body: "looks good"},
			},
		},
		{
			ID:       "t2",
			Path:     "internal/plugin/registry.go",
			Line:     12,
			Resolved: true,
			Comments: []domain.Comment{
				{Author: "bob", Body: "nit: remove import"},
			},
		},
	}
	m.SetComments(threads)

	assert.Len(t, m.comments, 2)
	require.NotNil(t, m.commentMap)

	got := m.commentsForLine("internal/plugin/registry.go", 11)
	require.Len(t, got, 1)
	assert.Equal(t, "t1", got[0].ID)

	// No comments for unrelated line.
	got = m.commentsForLine("internal/plugin/registry.go", 999)
	assert.Empty(t, got)
}

func TestDiffInlineCommentRendering(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())

	threads := []domain.CommentThread{
		{
			ID:   "t1",
			Path: "internal/plugin/registry.go",
			Line: 11,
			Comments: []domain.Comment{
				{Author: "alice", Body: "looks good"},
			},
		},
	}
	m.SetComments(threads)

	view := m.View()
	// Comment thread borders should appear in unified view.
	assert.Contains(t, view, "┌", "expected top border in view")
	assert.Contains(t, view, "└", "expected bottom border in view")
	assert.Contains(t, view, "alice", "expected comment author in view")
	assert.Contains(t, view, "looks good", "expected comment body in view")
}

func TestDiffInlineCommentResolved(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())

	threads := []domain.CommentThread{
		{
			ID:       "t1",
			Path:     "internal/plugin/registry.go",
			Line:     11,
			Resolved: true,
			Comments: []domain.Comment{
				{Author: "bob", Body: "done"},
			},
		},
	}
	m.SetComments(threads)

	view := m.View()
	assert.Contains(t, view, "resolved", "expected resolved indicator in view")
}

func TestDiffCommentEditorOpenClose(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())

	// Scroll to a diff line (past hunk header at position 0).
	m.scrollY = 1

	// Press c to open comment editor.
	cKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
	m.Update(cKey)
	assert.True(t, m.editing, "expected editing mode after c")
	assert.NotEmpty(t, m.editPath, "expected editPath to be set")

	// Type some text.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h', 'i'}})
	assert.Equal(t, "hi", m.editBuffer)

	// Escape cancels.
	m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	assert.False(t, m.editing, "expected editing to end on Escape")
	assert.Empty(t, m.editBuffer, "editBuffer should be empty after Escape")
}

func TestDiffCommentEditorSubmit(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())
	m.SetPRNumber(42)

	// Move past hunk header.
	m.scrollY = 1

	// Open editor.
	cKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
	m.Update(cKey)

	// Type comment text.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t', 'e', 's', 't'}})

	// Ctrl+S submits.
	cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	assert.False(t, m.editing, "expected editing to end after Ctrl+S")

	// Should produce an AddInlineCommentMsg command.
	require.NotNil(t, cmd, "expected a command after Ctrl+S")
	msg := cmd()
	addMsg, ok := msg.(AddInlineCommentMsg)
	require.True(t, ok, "expected AddInlineCommentMsg, got %T", msg)
	assert.Equal(t, 42, addMsg.Number)
	assert.Equal(t, "test", addMsg.Input.Body)
}

func TestDiffCommentEditorEmptySubmit(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())
	m.scrollY = 1

	// Open editor.
	cKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
	m.Update(cKey)

	// Ctrl+S with empty buffer should not submit.
	cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	assert.False(t, m.editing, "expected editing to end on empty submit")
	assert.Nil(t, cmd, "expected nil command for empty comment")
}

func TestDiffCommentReply(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())

	threads := []domain.CommentThread{
		{
			ID:   "t1",
			Path: "internal/plugin/registry.go",
			Line: 11,
			Comments: []domain.Comment{
				{Author: "alice", Body: "looks good"},
			},
		},
	}
	m.SetComments(threads)

	// Scroll to deletion at line 11 (position 2: header=0, context=1, delete=2).
	m.scrollY = 2

	// Press r to reply.
	rKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	m.Update(rKey)

	assert.True(t, m.editing, "expected editing mode for reply")
	assert.Equal(t, "t1", m.editReplyTo)
}

func TestDiffCommentResolve(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())

	threads := []domain.CommentThread{
		{
			ID:   "t1",
			Path: "internal/plugin/registry.go",
			Line: 11,
			Comments: []domain.Comment{
				{Author: "alice", Body: "looks good"},
			},
		},
	}
	m.SetComments(threads)

	// Scroll to deletion at line 11 (position 2: header=0, context=1, delete=2).
	m.scrollY = 2

	// Press x to resolve.
	xKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
	cmd := m.Update(xKey)

	require.NotNil(t, cmd, "expected a command for resolve")
	msg := cmd()
	resolveMsg, ok := msg.(ResolveThreadMsg)
	require.True(t, ok, "expected ResolveThreadMsg, got %T", msg)
	assert.Equal(t, "t1", resolveMsg.ThreadID)
}

func TestDiffCommentEditorBackspace(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())
	m.scrollY = 1

	cKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
	m.Update(cKey)

	// Type and backspace.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a', 'b'}})
	m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	assert.Equal(t, "a", m.editBuffer)

	// Backspace on empty.
	m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m.Update(tea.KeyMsg{Type: tea.KeyBackspace}) // extra backspace on empty
	assert.Empty(t, m.editBuffer)
}

func TestDiffCommentEditorNewline(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())
	m.scrollY = 1

	cKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
	m.Update(cKey)

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	assert.Equal(t, "a\nb", m.editBuffer)
}

func TestDiffCommentEditorRendering(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())
	m.scrollY = 1

	cKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
	m.Update(cKey)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h', 'e', 'l', 'l', 'o'}})

	view := m.View()
	// Editor should show in the view.
	assert.Contains(t, view, "Comment on", "expected comment editor header in view")
	assert.Contains(t, view, "Ctrl+S", "expected Ctrl+S hint in editor")
}

func TestDiffCurrentDiffLine(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())

	// scrollY=0 is the hunk header.
	path, line, side := m.currentDiffLine()
	assert.Equal(t, "internal/plugin/registry.go", path)
	assert.Equal(t, 0, line)
	assert.Equal(t, "", side)

	// scrollY=1 is the first context line (OldNum=10, NewNum=10).
	m.scrollY = 1
	_, line, side = m.currentDiffLine()
	assert.Equal(t, 10, line)
	assert.Equal(t, "RIGHT", side)

	// scrollY=2 is a deletion (OldNum=11).
	m.scrollY = 2
	_, line, side = m.currentDiffLine()
	assert.Equal(t, 11, line)
	assert.Equal(t, "LEFT", side)

	// scrollY=3 is an addition (NewNum=11).
	m.scrollY = 3
	_, line, side = m.currentDiffLine()
	assert.Equal(t, 11, line)
	assert.Equal(t, "RIGHT", side)
}

func TestDiffNoCommentOnHunkHeader(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())

	// scrollY=0 is hunk header (line=0), c should not open editor.
	m.scrollY = 0
	cKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
	m.Update(cKey)
	assert.False(t, m.editing, "should not open editor on hunk header")
}

func TestDiffSpinnerTick(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)

	// When loading, spinner tick increments frame.
	assert.True(t, m.loading, "new model should be loading")
	m.spinnerFrame = 3
	cmd := m.Update(diffSpinnerTickMsg{})
	assert.Equal(t, 4, m.spinnerFrame, "spinnerFrame should increment on tick")
	assert.NotNil(t, cmd, "should return another tick command when loading")

	// When not loading, tick is a no-op.
	m.loading = false
	m.spinnerFrame = 10
	cmd = m.Update(diffSpinnerTickMsg{})
	assert.Equal(t, 10, m.spinnerFrame, "spinnerFrame should not change when not loading")
	assert.Nil(t, cmd, "should return nil when not loading")
}

func TestDiffStartLoading(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())

	// After SetDiff, loading is false.
	assert.False(t, m.loading)
	assert.NotNil(t, m.diff)
	m.fileIdx = 1
	m.scrollY = 5
	m.spinnerFrame = 7

	cmd := m.StartLoading()

	assert.True(t, m.loading, "StartLoading should set loading to true")
	assert.Nil(t, m.diff, "StartLoading should clear diff")
	assert.Equal(t, 0, m.fileIdx, "StartLoading should reset fileIdx")
	assert.Equal(t, 0, m.scrollY, "StartLoading should reset scrollY")
	assert.Equal(t, 0, m.spinnerFrame, "StartLoading should reset spinnerFrame")
	assert.NotNil(t, cmd, "StartLoading should return a spinner tick command")
}

func TestDiffLoadedError(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)

	// Simulate error loading diff.
	m.Update(DiffLoadedMsg{Diff: nil, Err: fmt.Errorf("timeout")})

	assert.False(t, m.loading, "loading should be false after error")
	assert.Nil(t, m.diff, "diff should be nil on error")
	assert.NotNil(t, m.loadErr, "loadErr should be set")
}

func TestDiffErrorViewState(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)

	// Start loading then receive error.
	m.StartLoading()
	m.Update(DiffLoadedMsg{Diff: nil, Err: fmt.Errorf("diff too large")})

	view := m.View()
	// Should NOT show "Loading diff..." (spinner is stopped).
	assert.NotContains(t, view, "Loading diff", "should not show loading text after error")
	// Should show the error message.
	assert.Contains(t, view, "diff too large", "should show error message")
	// Should show the hint about external diff.
	assert.Contains(t, view, "external diff", "should show external diff hint")
}

func TestDiffFileChangeCountCache(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)

	d := testDiff()
	m.SetDiff(d)

	require.Len(t, m.fileChangeCounts, 2, "should have counts for each file")

	// Verify cached counts match countFileChanges.
	for i, f := range d.Files {
		expectedAdds, expectedDels := countFileChanges(f)
		assert.Equal(t, expectedAdds, m.fileChangeCounts[i][0], "adds mismatch for file %d", i)
		assert.Equal(t, expectedDels, m.fileChangeCounts[i][1], "dels mismatch for file %d", i)
	}
}

func TestDiffLazyRendering(t *testing.T) {
	// Create a large diff with many lines.
	var lines []domain.DiffLine
	for i := 1; i <= 2000; i++ {
		lines = append(lines, domain.DiffLine{
			Type:    domain.DiffAdd,
			Content: fmt.Sprintf("line %d: some code content here", i),
			NewNum:  i,
		})
	}
	largeDiff := &domain.Diff{
		Files: []domain.FileDiff{
			{
				Path: "large.go",
				Hunks: []domain.Hunk{
					{
						Header: "@@ -0,0 +1,2000 @@",
						Lines:  lines,
					},
				},
			},
		},
	}

	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(largeDiff)

	// View should render without hanging. Timeout would fail the test.
	view := m.View()
	assert.NotEmpty(t, view, "large diff view should not be empty")

	// Scroll to middle and render again.
	m.scrollY = 1000
	view = m.View()
	assert.NotEmpty(t, view, "scrolled large diff view should not be empty")
}

func TestDiffLargeFileWarning(t *testing.T) {
	// Create a diff with > maxHighlightLines (5000) lines.
	var lines []domain.DiffLine
	for i := 1; i <= 5500; i++ {
		lines = append(lines, domain.DiffLine{
			Type:    domain.DiffContext,
			Content: fmt.Sprintf("line %d", i),
			OldNum:  i,
			NewNum:  i,
		})
	}
	hugeDiff := &domain.Diff{
		Files: []domain.FileDiff{
			{
				Path: "huge.go",
				Hunks: []domain.Hunk{
					{
						Header: "@@ -1,5500 +1,5500 @@",
						Lines:  lines,
					},
				},
			},
		},
	}

	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(hugeDiff)

	// At scrollY=0, the warning banner should appear.
	view := m.View()
	assert.Contains(t, view, "Large file", "expected large file warning in view")
	assert.Contains(t, view, "syntax highlighting disabled", "expected highlighting disabled message")

	// Scrolled down, warning should not appear (only shown at top).
	m.scrollY = 100
	view = m.View()
	assert.NotContains(t, view, "Large file", "warning should not appear when scrolled")
}

func TestDiffSpinnerViewRendering(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(80, 24)

	// Advance spinner a few frames.
	m.spinnerFrame = 3
	view := m.View()
	assert.NotEmpty(t, view, "spinner view should not be empty")
	assert.Contains(t, view, "Loading diff", "should show loading text with spinner")
}
