package core

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
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
		if help.Key == "" {
			t.Errorf("binding %s has no help key", b.name)
		}
		if help.Desc == "" {
			t.Errorf("binding %s has no help description", b.name)
		}
	}
}

func TestShortHelpNotEmpty(t *testing.T) {
	km := DefaultKeyMap()
	short := km.ShortHelp()
	if len(short) == 0 {
		t.Error("ShortHelp() should return bindings")
	}
}

func TestFullHelpNotEmpty(t *testing.T) {
	km := DefaultKeyMap()
	full := km.FullHelp()
	if len(full) == 0 {
		t.Error("FullHelp() should return binding groups")
	}
	for i, group := range full {
		if len(group) == 0 {
			t.Errorf("FullHelp() group %d is empty", i)
		}
	}
}
