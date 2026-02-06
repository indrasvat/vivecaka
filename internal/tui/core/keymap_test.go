package core

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	"github.com/stretchr/testify/assert"
)

func TestDefaultKeyMapCreation(t *testing.T) {
	km := DefaultKeyMap()

	// Verify key bindings exist and have help text.
	bindings := []struct {
		name    string
		binding key.Binding
	}{
		{"Quit", km.Quit},
		{"Help", km.Help},
		{"Up", km.Up},
		{"Down", km.Down},
		{"Enter", km.Enter},
		{"Back", km.Back},
		{"Search", km.Search},
		{"Filter", km.Filter},
		{"Sort", km.Sort},
		{"Yank", km.Yank},
		{"Open", km.Open},
		{"Checkout", km.Checkout},
		{"Refresh", km.Refresh},
		{"RepoSwitch", km.RepoSwitch},
		{"ThemeCycle", km.ThemeCycle},
	}

	for _, b := range bindings {
		help := b.binding.Help()
		assert.NotEmpty(t, help.Key, "binding %s should have help key", b.name)
		assert.NotEmpty(t, help.Desc, "binding %s should have help description", b.name)
	}
}

func TestShortHelpNotEmpty(t *testing.T) {
	km := DefaultKeyMap()
	short := km.ShortHelp()
	assert.NotEmpty(t, short, "ShortHelp() should return bindings")
}

func TestApplyOverrides(t *testing.T) {
	km := DefaultKeyMap()

	// Override quit to ctrl+q.
	km.ApplyOverrides(map[string]string{
		"quit":   "ctrl+q",
		"search": "ctrl+f",
	})

	// Verify quit was overridden.
	quitKeys := km.Quit.Keys()
	assert.Contains(t, quitKeys, "ctrl+q")

	// Verify search was overridden.
	searchKeys := km.Search.Keys()
	assert.Contains(t, searchKeys, "ctrl+f")
}

func TestApplyOverridesUnknown(t *testing.T) {
	km := DefaultKeyMap()
	// Should not panic on unknown key name.
	km.ApplyOverrides(map[string]string{"nonexistent": "x"})
}

func TestFullHelpNotEmpty(t *testing.T) {
	km := DefaultKeyMap()
	full := km.FullHelp()
	assert.NotEmpty(t, full, "FullHelp() should return binding groups")
	for i, group := range full {
		assert.NotEmpty(t, group, "FullHelp() group %d should not be empty", i)
	}
}
