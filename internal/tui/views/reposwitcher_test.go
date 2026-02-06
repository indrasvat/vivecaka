package views

import (
	"strings"
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
	if len(m.favorites) != 0 {
		t.Error("favorites should be empty initially")
	}
	if len(m.visible) != 0 {
		t.Error("visible should be empty initially")
	}
}

func TestRepoSwitcherSetRepos(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetRepos(testRepoEntries())

	if len(m.favorites) != 4 {
		t.Errorf("favorites len = %d, want 4", len(m.favorites))
	}
	// All should be marked as favorites.
	for _, f := range m.favorites {
		if !f.Favorite {
			t.Errorf("repo %s should be marked favorite", f.Repo.String())
		}
		if f.Section != SectionFavorite {
			t.Errorf("repo %s section = %d, want SectionFavorite", f.Repo.String(), f.Section)
		}
	}
	if len(m.visible) != 4 {
		t.Errorf("visible len = %d, want 4", len(m.visible))
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
	if len(m.visible) != 1 {
		t.Errorf("visible = %d, want 1", len(m.visible))
	}
	if m.visible[0].Repo.Name != "dotfiles" {
		t.Errorf("visible[0] = %q, want dotfiles", m.visible[0].Repo.Name)
	}
}

func TestRepoSwitcherSearchBackspace(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetSize(100, 30)
	m.SetRepos(testRepoEntries())

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	if len(m.visible) != 0 {
		t.Errorf("visible for 'z' = %d, want 0", len(m.visible))
	}

	m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if len(m.visible) != 4 {
		t.Errorf("after backspace visible = %d, want 4", len(m.visible))
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
	// Should contain section header.
	if !strings.Contains(view, "FAVORITES") {
		t.Error("view should contain FAVORITES section header")
	}
	// Should contain title.
	if !strings.Contains(view, "Switch Repository") {
		t.Error("view should contain title")
	}
	// Should contain star.
	if !strings.Contains(view, "★") {
		t.Error("view should contain star for favorites")
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
	if !strings.Contains(view, "No repos matching query") {
		t.Error("empty view should show no-match message")
	}
}

func TestRepoSwitcherViewSmall(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetSize(40, 10)
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

	if m.cursor >= len(m.visible) {
		t.Errorf("cursor %d >= visible len %d", m.cursor, len(m.visible))
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

func TestRepoSwitcherMergeDiscovered(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetSize(100, 30)
	m.SetRepos([]RepoEntry{
		{Repo: domain.RepoRef{Owner: "acme", Name: "frontend"}, Favorite: true},
	})

	// Merge discovered — acme/frontend should be deduped.
	m.MergeDiscovered([]domain.RepoRef{
		{Owner: "acme", Name: "frontend"}, // duplicate
		{Owner: "acme", Name: "backend"},
		{Owner: "other", Name: "lib"},
	})

	if len(m.discovered) != 2 {
		t.Errorf("discovered = %d, want 2 (deduped)", len(m.discovered))
	}
	if len(m.visible) != 3 {
		t.Errorf("visible = %d, want 3 (1 fav + 2 disc)", len(m.visible))
	}
}

func TestRepoSwitcherSetCurrentRepo(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetSize(100, 30)
	m.SetRepos(testRepoEntries())

	m.SetCurrentRepo(domain.RepoRef{Owner: "indrasvat", Name: "dotfiles"})

	for _, v := range m.visible {
		if v.Repo.Name == "dotfiles" && !v.Current {
			t.Error("dotfiles should be marked current")
		}
		if v.Repo.Name == "vivecaka" && v.Current {
			t.Error("vivecaka should not be current anymore")
		}
	}
}

func TestRepoSwitcherToggleFavorite(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetSize(100, 30)
	m.SetRepos([]RepoEntry{
		{Repo: domain.RepoRef{Owner: "acme", Name: "frontend"}, Favorite: true},
	})
	m.MergeDiscovered([]domain.RepoRef{
		{Owner: "acme", Name: "backend"},
	})

	// Move cursor to discovered entry (index 1 = acme/backend).
	m.cursor = 1

	// Press 's' to toggle favorite.
	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if cmd == nil {
		t.Fatal("s should produce a command")
	}

	msg := cmd()
	toggle, ok := msg.(ToggleFavoriteMsg)
	if !ok {
		t.Fatalf("expected ToggleFavoriteMsg, got %T", msg)
	}
	if toggle.Repo.Name != "backend" {
		t.Errorf("toggle repo = %q, want backend", toggle.Repo.Name)
	}
	if !toggle.Favorite {
		t.Error("toggle should mark as favorite")
	}

	// Backend should now be in favorites.
	if len(m.favorites) != 2 {
		t.Errorf("favorites = %d, want 2", len(m.favorites))
	}
	if len(m.discovered) != 0 {
		t.Errorf("discovered = %d, want 0", len(m.discovered))
	}
}

func TestRepoSwitcherToggleFavoriteRemove(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetSize(100, 30)
	m.SetRepos([]RepoEntry{
		{Repo: domain.RepoRef{Owner: "acme", Name: "frontend"}, Favorite: true},
		{Repo: domain.RepoRef{Owner: "acme", Name: "backend"}, Favorite: true},
	})

	// Cursor on first entry (acme/frontend), press 's' to unfavorite.
	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if cmd == nil {
		t.Fatal("s should produce a command")
	}

	msg := cmd()
	toggle, ok := msg.(ToggleFavoriteMsg)
	if !ok {
		t.Fatalf("expected ToggleFavoriteMsg, got %T", msg)
	}
	if toggle.Favorite {
		t.Error("toggle should mark as NOT favorite")
	}

	// Frontend should move to discovered.
	if len(m.favorites) != 1 {
		t.Errorf("favorites = %d, want 1", len(m.favorites))
	}
	if len(m.discovered) != 1 {
		t.Errorf("discovered = %d, want 1", len(m.discovered))
	}
}

func TestRepoSwitcherGhostEntry(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetSize(100, 30)
	m.SetRepos(nil)

	// Type "acme/newrepo" — should show ghost entry.
	for _, r := range "acme/newrepo" {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	if len(m.visible) != 1 {
		t.Errorf("visible = %d, want 1 (ghost)", len(m.visible))
	}
	if m.visible[0].Section != SectionGhost {
		t.Error("entry should be ghost section")
	}

	// Enter on ghost should emit ValidateRepoRequestMsg.
	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter on ghost should produce a command")
	}
	msg := cmd()
	if val, ok := msg.(ValidateRepoRequestMsg); !ok {
		t.Fatalf("expected ValidateRepoRequestMsg, got %T", msg)
	} else if val.Repo.Owner != "acme" || val.Repo.Name != "newrepo" {
		t.Errorf("validate repo = %s, want acme/newrepo", val.Repo.String())
	}
}

func TestRepoSwitcherNeedsDiscovery(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	if !m.NeedsDiscovery() {
		t.Error("should need discovery initially")
	}
	m.SetDiscovering()
	if m.NeedsDiscovery() {
		t.Error("should not need discovery while discovering")
	}
	m.MergeDiscovered(nil)
	if m.NeedsDiscovery() {
		t.Error("should not need discovery after merge")
	}
}

func TestRepoSwitcherFavorites(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetRepos(testRepoEntries())

	favs := m.Favorites()
	if len(favs) != 4 {
		t.Errorf("Favorites() = %d, want 4", len(favs))
	}
}

func TestRepoSwitcherViewSections(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetSize(100, 40)
	m.SetRepos([]RepoEntry{
		{Repo: domain.RepoRef{Owner: "acme", Name: "frontend"}, Favorite: true},
	})
	m.MergeDiscovered([]domain.RepoRef{
		{Owner: "acme", Name: "backend"},
	})

	view := m.View()
	if !strings.Contains(view, "FAVORITES") {
		t.Error("view should contain FAVORITES header")
	}
	if !strings.Contains(view, "YOUR REPOS") {
		t.Error("view should contain YOUR REPOS header")
	}
}

func TestRepoSwitcherSToggleDoesNotFilterWhenQueryEmpty(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetSize(100, 30)
	m.SetRepos(testRepoEntries())

	// Press 's' with empty query — should toggle, not search for "s".
	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if cmd == nil {
		t.Fatal("s with empty query should toggle favorite, not search")
	}
	if m.query != "" {
		t.Errorf("query should still be empty, got %q", m.query)
	}
}

func TestRepoSwitcherSAppendsWhenQueryNonEmpty(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetSize(100, 30)
	m.SetRepos(testRepoEntries())

	// Type "a" then "s".
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})

	if m.query != "as" {
		t.Errorf("query = %q, want %q", m.query, "as")
	}
}
