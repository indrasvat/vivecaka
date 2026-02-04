package views

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/indrasvat/vivecaka/internal/domain"
)

func testRepoEntries() []RepoEntry {
	return []RepoEntry{
		{Repo: domain.RepoRef{Owner: "indrasvat", Name: "vivecaka"}, Favorite: true, Current: true, OpenCount: 5},
		{Repo: domain.RepoRef{Owner: "indrasvat", Name: "dotfiles"}, Favorite: true, OpenCount: 12},
		{Repo: domain.RepoRef{Owner: "indrasvat", Name: "cli-tools"}, Favorite: false, OpenCount: 3},
		{Repo: domain.RepoRef{Owner: "acme", Name: "webapp"}, Favorite: false, OpenCount: 0},
	}
}

func TestNewRepoSwitcherModel(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	if len(m.repos) != 0 {
		t.Error("repos should be empty initially")
	}
}

func TestRepoSwitcherSetRepos(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetRepos(testRepoEntries())

	if len(m.repos) != 4 {
		t.Errorf("repos len = %d, want 4", len(m.repos))
	}
	if len(m.filtered) != 4 {
		t.Errorf("filtered len = %d, want 4", len(m.filtered))
	}
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
	if m.query != "" {
		t.Errorf("query = %q, want empty", m.query)
	}
}

func TestRepoSwitcherSetSize(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetSize(100, 30)
	if m.width != 100 || m.height != 30 {
		t.Errorf("size = %dx%d, want 100x30", m.width, m.height)
	}
}

func TestRepoSwitcherNavigation(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetSize(100, 30)
	m.SetRepos(testRepoEntries())

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

	// Can't go below 0.
	m.Update(up)
	if m.cursor != 0 {
		t.Errorf("cursor should stay at 0, got %d", m.cursor)
	}

	// Go to end.
	m.cursor = 3
	m.Update(down)
	if m.cursor != 3 {
		t.Errorf("cursor at end should stay at 3, got %d", m.cursor)
	}
}

func TestRepoSwitcherFuzzySearch(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetSize(100, 30)
	m.SetRepos(testRepoEntries())

	// Type "dot".
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})

	if m.query != "dot" {
		t.Errorf("query = %q, want %q", m.query, "dot")
	}
	if len(m.filtered) != 1 {
		t.Errorf("filtered = %d, want 1", len(m.filtered))
	}
	if m.filtered[0].Repo.Name != "dotfiles" {
		t.Errorf("filtered[0] = %q, want dotfiles", m.filtered[0].Repo.Name)
	}
}

func TestRepoSwitcherSearchBackspace(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetSize(100, 30)
	m.SetRepos(testRepoEntries())

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	if len(m.filtered) != 0 {
		t.Errorf("filtered for 'z' = %d, want 0", len(m.filtered))
	}

	m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if len(m.filtered) != 4 {
		t.Errorf("after backspace filtered = %d, want 4", len(m.filtered))
	}
}

func TestRepoSwitcherSearchBackspaceEmpty(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetSize(100, 30)
	m.SetRepos(testRepoEntries())

	// Backspace on empty query is safe.
	m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.query != "" {
		t.Errorf("query should stay empty, got %q", m.query)
	}
}

func TestRepoSwitcherSelect(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetSize(100, 30)
	m.SetRepos(testRepoEntries())

	// Move to second entry and select.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter should produce a command")
	}

	msg := cmd()
	sw, ok := msg.(SwitchRepoMsg)
	if !ok {
		t.Fatalf("expected SwitchRepoMsg, got %T", msg)
	}
	if sw.Repo.Name != "dotfiles" {
		t.Errorf("selected repo = %q, want dotfiles", sw.Repo.Name)
	}
}

func TestRepoSwitcherSelectEmpty(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetSize(100, 30)
	m.SetRepos(nil)

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("Enter on empty list should not produce a command")
	}
}

func TestRepoSwitcherEscape(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetSize(100, 30)
	m.SetRepos(testRepoEntries())

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("Escape should produce a command")
	}

	msg := cmd()
	if _, ok := msg.(CloseRepoSwitcherMsg); !ok {
		t.Errorf("expected CloseRepoSwitcherMsg, got %T", msg)
	}
}

func TestRepoSwitcherView(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetSize(100, 30)
	m.SetRepos(testRepoEntries())

	view := m.View()
	if view == "" {
		t.Error("view should not be empty")
	}
}

func TestRepoSwitcherViewEmpty(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetSize(100, 30)
	m.SetRepos(nil)

	view := m.View()
	if view == "" {
		t.Error("empty view should not be empty string")
	}
}

func TestRepoSwitcherViewSmall(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetSize(40, 10) // Small.
	m.SetRepos(testRepoEntries())

	view := m.View()
	if view == "" {
		t.Error("small view should not be empty")
	}
}

func TestRepoSwitcherCursorClamp(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetSize(100, 30)
	m.SetRepos(testRepoEntries())
	m.cursor = 3

	// Filter to 1 result — cursor should clamp.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})

	if m.cursor >= len(m.filtered) {
		t.Errorf("cursor %d >= filtered len %d", m.cursor, len(m.filtered))
	}
}

func TestFuzzyMatch(t *testing.T) {
	tests := []struct {
		text, query string
		want        bool
	}{
		{"indrasvat/vivecaka", "viv", true},
		{"indrasvat/vivecaka", "ivk", true},
		{"indrasvat/vivecaka", "xyz", false},
		{"indrasvat/dotfiles", "dot", true},
		{"indrasvat/dotfiles", "idf", true},
		{"acme/webapp", "aw", true},
		{"acme/webapp", "za", false},
		{"anything", "", true},
	}
	for _, tt := range tests {
		if got := fuzzyMatch(tt.text, tt.query); got != tt.want {
			t.Errorf("fuzzyMatch(%q, %q) = %v, want %v", tt.text, tt.query, got, tt.want)
		}
	}
}

func TestRepoSwitcherNonKeyMsg(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetSize(100, 30)

	cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if cmd != nil {
		t.Error("non-key messages should return nil cmd")
	}
}
