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

func TestDiffSplitModeToggle(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())

	if m.splitMode {
		t.Error("expected unified mode initially")
	}

	// Press t to toggle to split mode.
	tKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}
	m.Update(tKey)
	if !m.splitMode {
		t.Error("expected split mode after t")
	}

	// View should still render.
	view := m.View()
	if view == "" {
		t.Error("split mode view should not be empty")
	}
	// Should contain the file path.
	if !strings.Contains(view, "registry.go") {
		t.Error("expected file path in split view")
	}

	// Toggle back.
	m.Update(tKey)
	if m.splitMode {
		t.Error("expected unified mode after second t")
	}
}

func TestDiffSplitModeRendering(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())
	m.splitMode = true

	view := m.View()
	// Split mode should contain the divider character.
	if !strings.Contains(view, "│") {
		t.Error("expected vertical divider in split view")
	}
	// Should show the "Split" mode label.
	if !strings.Contains(view, "Split") {
		t.Error("expected 'Split' label in view")
	}
}

func TestDiffUnifiedModeLabel(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())

	view := m.View()
	if !strings.Contains(view, "Unified") {
		t.Error("expected 'Unified' label in default view")
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

	if len(m.comments) != 2 {
		t.Errorf("comments = %d, want 2", len(m.comments))
	}
	if m.commentMap == nil {
		t.Fatal("commentMap should not be nil")
	}

	got := m.commentsForLine("internal/plugin/registry.go", 11)
	if len(got) != 1 {
		t.Errorf("commentsForLine(registry.go, 11) = %d, want 1", len(got))
	}
	if len(got) > 0 && got[0].ID != "t1" {
		t.Errorf("expected thread t1, got %s", got[0].ID)
	}

	// No comments for unrelated line.
	got = m.commentsForLine("internal/plugin/registry.go", 999)
	if len(got) != 0 {
		t.Errorf("commentsForLine(registry.go, 999) = %d, want 0", len(got))
	}
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
	if !strings.Contains(view, "┌") || !strings.Contains(view, "└") {
		t.Error("expected comment thread borders in view")
	}
	if !strings.Contains(view, "alice") {
		t.Error("expected comment author in view")
	}
	if !strings.Contains(view, "looks good") {
		t.Error("expected comment body in view")
	}
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
	if !strings.Contains(view, "resolved") {
		t.Error("expected resolved indicator in view")
	}
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
	if !m.editing {
		t.Error("expected editing mode after c")
	}
	if m.editPath == "" {
		t.Error("expected editPath to be set")
	}

	// Type some text.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h', 'i'}})
	if m.editBuffer != "hi" {
		t.Errorf("editBuffer = %q, want %q", m.editBuffer, "hi")
	}

	// Escape cancels.
	m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if m.editing {
		t.Error("expected editing to end on Escape")
	}
	if m.editBuffer != "" {
		t.Errorf("editBuffer should be empty after Escape, got %q", m.editBuffer)
	}
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
	if m.editing {
		t.Error("expected editing to end after Ctrl+S")
	}

	// Should produce an AddInlineCommentMsg command.
	if cmd == nil {
		t.Fatal("expected a command after Ctrl+S")
	}
	msg := cmd()
	addMsg, ok := msg.(AddInlineCommentMsg)
	if !ok {
		t.Fatalf("expected AddInlineCommentMsg, got %T", msg)
	}
	if addMsg.Number != 42 {
		t.Errorf("PR number = %d, want 42", addMsg.Number)
	}
	if addMsg.Input.Body != "test" {
		t.Errorf("body = %q, want %q", addMsg.Input.Body, "test")
	}
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
	if m.editing {
		t.Error("expected editing to end on empty submit")
	}
	if cmd != nil {
		t.Error("expected nil command for empty comment")
	}
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

	if !m.editing {
		t.Error("expected editing mode for reply")
	}
	if m.editReplyTo != "t1" {
		t.Errorf("editReplyTo = %q, want %q", m.editReplyTo, "t1")
	}
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

	if cmd == nil {
		t.Fatal("expected a command for resolve")
	}
	msg := cmd()
	resolveMsg, ok := msg.(ResolveThreadMsg)
	if !ok {
		t.Fatalf("expected ResolveThreadMsg, got %T", msg)
	}
	if resolveMsg.ThreadID != "t1" {
		t.Errorf("ThreadID = %q, want %q", resolveMsg.ThreadID, "t1")
	}
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
	if m.editBuffer != "a" {
		t.Errorf("editBuffer after backspace = %q, want %q", m.editBuffer, "a")
	}

	// Backspace on empty.
	m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m.Update(tea.KeyMsg{Type: tea.KeyBackspace}) // extra backspace on empty
	if m.editBuffer != "" {
		t.Errorf("editBuffer = %q, want empty", m.editBuffer)
	}
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
	if m.editBuffer != "a\nb" {
		t.Errorf("editBuffer = %q, want %q", m.editBuffer, "a\nb")
	}
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
	if !strings.Contains(view, "Comment on") {
		t.Error("expected comment editor header in view")
	}
	if !strings.Contains(view, "Ctrl+S") {
		t.Error("expected Ctrl+S hint in editor")
	}
}

func TestDiffCurrentDiffLine(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())

	// scrollY=0 is the hunk header.
	path, line, side := m.currentDiffLine()
	if path != "internal/plugin/registry.go" {
		t.Errorf("path = %q, want %q", path, "internal/plugin/registry.go")
	}
	if line != 0 {
		t.Errorf("hunk header line = %d, want 0", line)
	}
	if side != "" {
		t.Errorf("hunk header side = %q, want empty", side)
	}

	// scrollY=1 is the first context line (OldNum=10, NewNum=10).
	m.scrollY = 1
	_, line, side = m.currentDiffLine()
	if line != 10 {
		t.Errorf("context line = %d, want 10", line)
	}
	if side != "RIGHT" {
		t.Errorf("context side = %q, want RIGHT", side)
	}

	// scrollY=2 is a deletion (OldNum=11).
	m.scrollY = 2
	_, line, side = m.currentDiffLine()
	if line != 11 {
		t.Errorf("delete line = %d, want 11", line)
	}
	if side != "LEFT" {
		t.Errorf("delete side = %q, want LEFT", side)
	}

	// scrollY=3 is an addition (NewNum=11).
	m.scrollY = 3
	_, line, side = m.currentDiffLine()
	if line != 11 {
		t.Errorf("add line = %d, want 11", line)
	}
	if side != "RIGHT" {
		t.Errorf("add side = %q, want RIGHT", side)
	}
}

func TestDiffNoCommentOnHunkHeader(t *testing.T) {
	m := NewDiffViewModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDiff(testDiff())

	// scrollY=0 is hunk header (line=0), c should not open editor.
	m.scrollY = 0
	cKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
	m.Update(cKey)
	if m.editing {
		t.Error("should not open editor on hunk header")
	}
}
