package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestThemeByNameReturnsKnown(t *testing.T) {
	names := []string{"default-dark", "catppuccin-mocha", "catppuccin-frappe", "tokyo-night", "dracula"}
	for _, name := range names {
		theme := ThemeByName(name)
		assert.Equal(t, name, theme.Name)
	}
}

func TestThemeByNameFallback(t *testing.T) {
	theme := ThemeByName("nonexistent")
	assert.Equal(t, "catppuccin-mocha", theme.Name)
}

func TestThemeNamesReturnsAll(t *testing.T) {
	names := ThemeNames()
	assert.Len(t, names, 5)
}

func TestNextThemeCycles(t *testing.T) {
	// Should cycle through all themes in order.
	current := "catppuccin-mocha"
	expected := []string{"catppuccin-frappe", "tokyo-night", "dracula", "default-dark", "catppuccin-mocha"}
	for _, want := range expected {
		next := NextTheme(current)
		assert.Equal(t, want, next.Name)
		current = next.Name
	}
}

func TestNextThemeUnknownFallsToFirst(t *testing.T) {
	theme := NextTheme("nonexistent")
	assert.Equal(t, "catppuccin-mocha", theme.Name)
}

func TestThemeColorsNotEmpty(t *testing.T) {
	for _, name := range ThemeNames() {
		theme := ThemeByName(name)
		assert.NotEmpty(t, theme.Primary, "theme %q should have primary color", name)
		assert.NotEmpty(t, theme.Error, "theme %q should have error color", name)
		assert.NotEmpty(t, theme.Success, "theme %q should have success color", name)
	}
}
