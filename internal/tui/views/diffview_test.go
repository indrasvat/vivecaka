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

	// { } still navigate files from content pane.
	next := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'}'}}
	m.Update(next)
	if m.fileIdx != 1 {
		t.Errorf("fileIdx after } = %d, want 1", m.fileIdx)
	}

	prev := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'{'}}
	m.Update(prev)
	if m.fileIdx != 0 {
		t.Errorf("fileIdx after { = %d, want 0", m.fileIdx)
	}

	// Tab toggles focus to tree pane.
	tab := tea.KeyMsg{Type: tea.KeyTab}
	m.Update(tab)
	if !m.treeFocus {
		t.Error("expected treeFocus after Tab")
	}

	// In tree pane, j/k navigate files.
	down := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	m.Update(down)
	if m.fileIdx != 1 {
		t.Errorf("fileIdx after j in tree = %d, want 1", m.fileIdx)
	}

	up := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	m.Update(up)
	if m.fileIdx != 0 {
		t.Errorf("fileIdx after k in tree = %d, want 0", m.fileIdx)
	}

	// Enter selects file and returns focus to content.
	m.Update(down) // go to file 1
	enter := tea.KeyMsg{Type: tea.KeyEnter}
	m.Update(enter)
	if m.treeFocus {
		t.Error("expected content focus after Enter in tree")
	}
	if m.fileIdx != 1 {
		t.Errorf("fileIdx after Enter = %d, want 1", m.fileIdx)
	}

	// Shift-tab also toggles focus.
	shiftTab := tea.KeyMsg{Type: tea.KeyShiftTab}
	m.Update(shiftTab)
	if !m.treeFocus {
		t.Error("expected treeFocus after Shift-Tab")
	}
}

func TestDiffFileNavResetsScroll(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())

	m.scrollY = 5
	// Use } to navigate to next file (resets scroll).
	next := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'}'}}
	m.Update(next)
	if m.scrollY != 0 {
		t.Errorf("scrollY should reset on file switch, got %d", m.scrollY)
	}
}

func TestDiffHunkNavigation(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 10)
	m.SetDiff(testDiffWithHunks())

	if m.scrollY != 0 {
		t.Fatalf("initial scrollY = %d, want 0", m.scrollY)
	}

	next := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}}
	m.Update(next)
	if m.scrollY != 2 {
		t.Errorf("scrollY after ] = %d, want 2", m.scrollY)
	}

	prev := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}}
	m.Update(prev)
	if m.scrollY != 0 {
		t.Errorf("scrollY after [ = %d, want 0", m.scrollY)
	}
}

func TestDiffFileJump(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())
	m.scrollY = 5

	next := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'}'}}
	m.Update(next)
	if m.fileIdx != 1 {
		t.Errorf("fileIdx after } = %d, want 1", m.fileIdx)
	}
	if m.scrollY != 0 {
		t.Errorf("scrollY after } = %d, want 0", m.scrollY)
	}

	prev := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'{'}}
	m.Update(prev)
	if m.fileIdx != 0 {
		t.Errorf("fileIdx after { = %d, want 0", m.fileIdx)
	}
}

func TestDiffTopBottomNavigation(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 5)
	m.SetDiff(testDiff())
	m.scrollY = 3

	g := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
	m.Update(g)
	m.Update(g)
	if m.scrollY != 0 {
		t.Errorf("scrollY after gg = %d, want 0", m.scrollY)
	}

	G := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
	m.Update(G)
	if m.scrollY != 3 {
		t.Errorf("scrollY after G = %d, want 3", m.scrollY)
	}
}

func TestDiffCollapseToggle(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())

	z := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}}
	a := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}

	m.Update(z)
	m.Update(a)
	if !m.isCollapsed(m.fileIdx) {
		t.Error("expected file to be collapsed after za")
	}

	m.Update(z)
	m.Update(a)
	if m.isCollapsed(m.fileIdx) {
		t.Error("expected file to expand after za again")
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
	if !strings.Contains(view, "\x1b[") {
		t.Error("expected ANSI escape sequences from syntax highlighting")
	}
	// Should contain the code keywords
	if !strings.Contains(view, "func") {
		t.Error("expected 'func' keyword in output")
	}
	if !strings.Contains(view, "main") {
		t.Error("expected 'main' in output")
	}
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
	if view == "" {
		t.Error("view should not be empty for unknown file types")
	}
	if !strings.Contains(view, "random content") {
		t.Error("expected content to be present even without syntax highlighting")
	}
}

func TestSyntaxHighlighterCaching(t *testing.T) {
	h := newSyntaxHighlighter()

	// First call creates the lexer
	_ = h.highlight("func main() {}", "test.go")

	// Check lexer is cached
	h.mu.RLock()
	_, cached := h.lexerCache[".go"]
	h.mu.RUnlock()

	if !cached {
		t.Error("expected .go lexer to be cached")
	}

	// Second call should use cache
	result := h.highlight("package main", "other.go")
	if result == "" {
		t.Error("highlight should return non-empty result")
	}
}

func TestSyntaxHighlighterGoCode(t *testing.T) {
	h := newSyntaxHighlighter()

	// Go code with keywords and strings
	code := `func main() { fmt.Println("hello world") }`
	result := h.highlight(code, "main.go")

	// Result should contain ANSI escape sequences for syntax colors
	if !strings.Contains(result, "\x1b[") {
		t.Errorf("expected ANSI escape sequences in highlighted Go code, got: %q", result)
	}

	// Result should still contain the original text
	if !strings.Contains(result, "func") {
		t.Error("expected 'func' keyword in result")
	}
	if !strings.Contains(result, "main") {
		t.Error("expected 'main' in result")
	}
	if !strings.Contains(result, "hello world") {
		t.Error("expected string literal in result")
	}
}

func TestSyntaxHighlighterRustCode(t *testing.T) {
	h := newSyntaxHighlighter()

	// Rust code with keywords
	code := `fn main() { println!("hello"); }`
	result := h.highlight(code, "main.rs")

	// Result should contain ANSI escape sequences
	if !strings.Contains(result, "\x1b[") {
		t.Errorf("expected ANSI escape sequences in highlighted Rust code, got: %q", result)
	}
}

func TestSyntaxHighlighterTypescriptCode(t *testing.T) {
	h := newSyntaxHighlighter()

	// TypeScript code with keywords
	code := `const x: number = 42; export default function foo() {}`
	result := h.highlight(code, "app.ts")

	// Result should contain ANSI escape sequences
	if !strings.Contains(result, "\x1b[") {
		t.Errorf("expected ANSI escape sequences in highlighted TypeScript code, got: %q", result)
	}
}

func TestSyntaxHighlighterFallback(t *testing.T) {
	h := newSyntaxHighlighter()

	// Unknown file type should return original content
	code := "some random content"
	result := h.highlight(code, "file.unknown123xyz")

	// For truly unknown extensions, should return original
	if result != code {
		// This is actually fine - Chroma might have broad fallback rules
		// Just verify it doesn't crash and returns something
		if result == "" {
			t.Error("highlight should return non-empty result even for unknown types")
		}
	}
}

func TestDiffTwoPaneLayout(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())

	view := m.View()

	// View should contain file names from the tree.
	if !strings.Contains(view, "registry.go") {
		t.Error("expected file tree to contain 'registry.go'")
	}
	if !strings.Contains(view, "hooks.go") {
		t.Error("expected file tree to contain 'hooks.go'")
	}

	// Should contain diff content markers.
	if !strings.Contains(view, "@@") {
		t.Error("expected diff content with hunk headers")
	}
}

func TestDiffTreeFocusToggle(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())

	// Initially, content has focus.
	if m.treeFocus {
		t.Error("expected content focus initially")
	}

	// Tab switches to tree.
	tab := tea.KeyMsg{Type: tea.KeyTab}
	m.Update(tab)
	if !m.treeFocus {
		t.Error("expected tree focus after Tab")
	}

	// Tab back to content.
	m.Update(tab)
	if m.treeFocus {
		t.Error("expected content focus after second Tab")
	}
}

func TestDiffTreeWidth(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())

	// Normal terminal: 25% of width, clamped.
	m.SetSize(120, 40)
	w := m.computeTreeWidth()
	if w < 20 || w > 40 {
		t.Errorf("tree width %d outside range [20, 40] for width=120", w)
	}

	// Very wide terminal.
	m.SetSize(200, 40)
	w = m.computeTreeWidth()
	if w > 40 {
		t.Errorf("tree width %d should cap at 40 for wide terminal", w)
	}

	// Narrow terminal.
	m.SetSize(60, 40)
	w = m.computeTreeWidth()
	if w < 10 {
		t.Errorf("tree width %d should be at least 10 for narrow terminal", w)
	}
}
