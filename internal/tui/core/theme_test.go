package core

import "testing"

func TestThemeByNameReturnsKnown(t *testing.T) {
	names := []string{"default-dark", "catppuccin-mocha", "catppuccin-frappe", "tokyo-night", "dracula"}
	for _, name := range names {
		theme := ThemeByName(name)
		if theme.Name != name {
			t.Errorf("ThemeByName(%q).Name = %q", name, theme.Name)
		}
	}
}

func TestThemeByNameFallback(t *testing.T) {
	theme := ThemeByName("nonexistent")
	if theme.Name != "catppuccin-mocha" {
		t.Errorf("ThemeByName(unknown).Name = %q, want %q", theme.Name, "catppuccin-mocha")
	}
}

func TestThemeNamesReturnsAll(t *testing.T) {
	names := ThemeNames()
	if len(names) != 5 {
		t.Errorf("ThemeNames() len = %d, want 5", len(names))
	}
}

func TestNextThemeCycles(t *testing.T) {
	// Should cycle through all themes in order.
	current := "catppuccin-mocha"
	expected := []string{"catppuccin-frappe", "tokyo-night", "dracula", "default-dark", "catppuccin-mocha"}
	for _, want := range expected {
		next := NextTheme(current)
		if next.Name != want {
			t.Errorf("NextTheme(%q) = %q, want %q", current, next.Name, want)
		}
		current = next.Name
	}
}

func TestNextThemeUnknownFallsToFirst(t *testing.T) {
	theme := NextTheme("nonexistent")
	if theme.Name != "catppuccin-mocha" {
		t.Errorf("NextTheme(unknown) = %q, want catppuccin-mocha", theme.Name)
	}
}

func TestThemeColorsNotEmpty(t *testing.T) {
	for _, name := range ThemeNames() {
		theme := ThemeByName(name)
		if theme.Primary == "" {
			t.Errorf("theme %q has empty primary color", name)
		}
		if theme.Error == "" {
			t.Errorf("theme %q has empty error color", name)
		}
		if theme.Success == "" {
			t.Errorf("theme %q has empty success color", name)
		}
	}
}
