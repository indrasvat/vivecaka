package views

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	assert.Empty(t, m.favorites, "favorites should be empty initially")
	assert.Empty(t, m.visible, "visible should be empty initially")
}

func TestRepoSwitcherSetRepos(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetRepos(testRepoEntries())

	assert.Len(t, m.favorites, 4)
	// All should be marked as favorites.
	for _, f := range m.favorites {
		assert.True(t, f.Favorite, "repo %s should be marked favorite", f.Repo.String())
		assert.Equal(t, SectionFavorite, f.Section, "repo %s section should be SectionFavorite", f.Repo.String())
	}
	assert.Len(t, m.visible, 4)
	assert.Equal(t, 0, m.cursor)
	assert.Empty(t, m.query)
}

func TestRepoSwitcherSetSize(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetSize(100, 30)
	assert.Equal(t, 100, m.width)
	assert.Equal(t, 30, m.height)
}

func TestRepoSwitcherNavigation(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetSize(100, 30)
	m.SetRepos(testRepoEntries())

	// Repo switcher uses arrow keys for navigation (j/k type into search).
	down := tea.KeyMsg{Type: tea.KeyDown}
	m.Update(down)
	assert.Equal(t, 1, m.cursor)

	up := tea.KeyMsg{Type: tea.KeyUp}
	m.Update(up)
	assert.Equal(t, 0, m.cursor)

	// Can't go below 0.
	m.Update(up)
	assert.Equal(t, 0, m.cursor, "cursor should stay at 0")

	// Go to end.
	m.cursor = 3
	m.Update(down)
	assert.Equal(t, 3, m.cursor, "cursor at end should stay at 3")
}

func TestRepoSwitcherFuzzySearch(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetSize(100, 30)
	m.SetRepos(testRepoEntries())

	// Type "dot".
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})

	assert.Equal(t, "dot", m.query)
	require.Len(t, m.visible, 1)
	assert.Equal(t, "dotfiles", m.visible[0].Repo.Name)
}

func TestRepoSwitcherSearchBackspace(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetSize(100, 30)
	m.SetRepos(testRepoEntries())

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	assert.Empty(t, m.visible)

	m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	assert.Len(t, m.visible, 4)
}

func TestRepoSwitcherSearchBackspaceEmpty(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetSize(100, 30)
	m.SetRepos(testRepoEntries())

	// Backspace on empty query is safe.
	m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	assert.Empty(t, m.query, "query should stay empty")
}

func TestRepoSwitcherSelect(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetSize(100, 30)
	m.SetRepos(testRepoEntries())

	// Move to second entry (arrow key) and select.
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd, "Enter should produce a command")

	msg := cmd()
	sw, ok := msg.(SwitchRepoMsg)
	require.True(t, ok, "expected SwitchRepoMsg, got %T", msg)
	assert.Equal(t, "dotfiles", sw.Repo.Name)
}

func TestRepoSwitcherSelectEmpty(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetSize(100, 30)
	m.SetRepos(nil)

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd, "Enter on empty list should not produce a command")
}

func TestRepoSwitcherEscape(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetSize(100, 30)
	m.SetRepos(testRepoEntries())

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	require.NotNil(t, cmd, "Escape should produce a command")

	msg := cmd()
	_, ok := msg.(CloseRepoSwitcherMsg)
	assert.True(t, ok, "expected CloseRepoSwitcherMsg, got %T", msg)
}

func TestRepoSwitcherView(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetSize(100, 30)
	m.SetRepos(testRepoEntries())

	view := m.View()
	assert.NotEmpty(t, view, "view should not be empty")
	assert.Contains(t, view, "FAVORITES", "view should contain FAVORITES section header")
	assert.Contains(t, view, "Switch Repository", "view should contain title")
	assert.Contains(t, view, "★", "view should contain star for favorites")
}

func TestRepoSwitcherViewEmpty(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetSize(100, 30)
	m.SetRepos(nil)

	view := m.View()
	assert.NotEmpty(t, view, "empty view should not be empty string")
	assert.Contains(t, view, "No repos matching query", "empty view should show no-match message")
}

func TestRepoSwitcherViewSmall(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetSize(40, 10)
	m.SetRepos(testRepoEntries())

	view := m.View()
	assert.NotEmpty(t, view, "small view should not be empty")
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

	assert.Less(t, m.cursor, len(m.visible))
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
		got := fuzzyMatch(tt.text, tt.query)
		assert.Equal(t, tt.want, got, "fuzzyMatch(%q, %q)", tt.text, tt.query)
	}
}

func TestRepoSwitcherNonKeyMsg(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetSize(100, 30)

	cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	assert.Nil(t, cmd, "non-key messages should return nil cmd")
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

	assert.Len(t, m.discovered, 2, "deduped discovered")
	assert.Len(t, m.visible, 3, "1 fav + 2 disc")
}

func TestRepoSwitcherSetCurrentRepo(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetSize(100, 30)
	m.SetRepos(testRepoEntries())

	m.SetCurrentRepo(domain.RepoRef{Owner: "indrasvat", Name: "dotfiles"})

	for _, v := range m.visible {
		if v.Repo.Name == "dotfiles" {
			assert.True(t, v.Current, "dotfiles should be marked current")
		}
		if v.Repo.Name == "vivecaka" {
			assert.False(t, v.Current, "vivecaka should not be current anymore")
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
	require.NotNil(t, cmd, "s should produce a command")

	msg := cmd()
	toggle, ok := msg.(ToggleFavoriteMsg)
	require.True(t, ok, "expected ToggleFavoriteMsg, got %T", msg)
	assert.Equal(t, "backend", toggle.Repo.Name)
	assert.True(t, toggle.Favorite, "toggle should mark as favorite")

	// Backend should now be in favorites.
	assert.Len(t, m.favorites, 2)
	assert.Empty(t, m.discovered)
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
	require.NotNil(t, cmd, "s should produce a command")

	msg := cmd()
	toggle, ok := msg.(ToggleFavoriteMsg)
	require.True(t, ok, "expected ToggleFavoriteMsg, got %T", msg)
	assert.False(t, toggle.Favorite, "toggle should mark as NOT favorite")

	// Frontend should move to discovered.
	assert.Len(t, m.favorites, 1)
	assert.Len(t, m.discovered, 1)
}

func TestRepoSwitcherGhostEntry(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetSize(100, 30)
	m.SetRepos(nil)

	// Type "acme/newrepo" — should show ghost entry.
	for _, r := range "acme/newrepo" {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	require.Len(t, m.visible, 1, "ghost entry")
	assert.Equal(t, SectionGhost, m.visible[0].Section, "entry should be ghost section")

	// Enter on ghost should emit ValidateRepoRequestMsg.
	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd, "Enter on ghost should produce a command")
	msg := cmd()
	val, ok := msg.(ValidateRepoRequestMsg)
	require.True(t, ok, "expected ValidateRepoRequestMsg, got %T", msg)
	assert.Equal(t, "acme", val.Repo.Owner)
	assert.Equal(t, "newrepo", val.Repo.Name)
}

func TestRepoSwitcherNeedsDiscovery(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	assert.True(t, m.NeedsDiscovery(), "should need discovery initially")
	m.SetDiscovering()
	assert.False(t, m.NeedsDiscovery(), "should not need discovery while discovering")
	m.MergeDiscovered(nil)
	assert.False(t, m.NeedsDiscovery(), "should not need discovery after merge")
}

func TestRepoSwitcherFavorites(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetRepos(testRepoEntries())

	favs := m.Favorites()
	assert.Len(t, favs, 4)
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
	assert.Contains(t, view, "FAVORITES", "view should contain FAVORITES header")
	assert.Contains(t, view, "YOUR REPOS", "view should contain YOUR REPOS header")
}

func TestRepoSwitcherSToggleDoesNotFilterWhenQueryEmpty(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetSize(100, 30)
	m.SetRepos(testRepoEntries())

	// Press 's' with empty query — should toggle, not search for "s".
	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	require.NotNil(t, cmd, "s with empty query should toggle favorite, not search")
	assert.Empty(t, m.query, "query should still be empty")
}

func TestRepoSwitcherSAppendsWhenQueryNonEmpty(t *testing.T) {
	m := NewRepoSwitcherModel(testStyles(), testKeys())
	m.SetSize(100, 30)
	m.SetRepos(testRepoEntries())

	// Type "a" then "s".
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})

	assert.Equal(t, "as", m.query)
}
