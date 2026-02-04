package tui

import "github.com/charmbracelet/lipgloss"

// Theme defines semantic colors for the application UI.
type Theme struct {
	Name      string
	Primary   lipgloss.Color
	Secondary lipgloss.Color
	Success   lipgloss.Color
	Error     lipgloss.Color
	Warning   lipgloss.Color
	Info      lipgloss.Color
	Muted     lipgloss.Color
	Subtext   lipgloss.Color
	Border    lipgloss.Color
	Bg        lipgloss.Color
	Fg        lipgloss.Color
	BgDim     lipgloss.Color
}

// Built-in themes.
var themes = map[string]Theme{
	"default-dark":      defaultDark,
	"catppuccin-mocha":  catppuccinMocha,
	"catppuccin-frappe": catppuccinFrappe,
	"tokyo-night":       tokyoNight,
	"dracula":           dracula,
}

var defaultDark = Theme{
	Name:      "default-dark",
	Primary:   lipgloss.Color("#7C3AED"),
	Secondary: lipgloss.Color("#06B6D4"),
	Success:   lipgloss.Color("#22C55E"),
	Error:     lipgloss.Color("#EF4444"),
	Warning:   lipgloss.Color("#EAB308"),
	Info:      lipgloss.Color("#3B82F6"),
	Muted:     lipgloss.Color("#6B7280"),
	Subtext:   lipgloss.Color("#9CA3AF"),
	Border:    lipgloss.Color("#4B5563"),
	Bg:        lipgloss.Color("#1F2937"),
	Fg:        lipgloss.Color("#F9FAFB"),
	BgDim:     lipgloss.Color("#111827"),
}

var catppuccinMocha = Theme{
	Name:      "catppuccin-mocha",
	Primary:   lipgloss.Color("#CBA6F7"), // mauve
	Secondary: lipgloss.Color("#89B4FA"), // blue
	Success:   lipgloss.Color("#A6E3A1"), // green
	Error:     lipgloss.Color("#F38BA8"), // red
	Warning:   lipgloss.Color("#F9E2AF"), // yellow
	Info:      lipgloss.Color("#89DCEB"), // sky
	Muted:     lipgloss.Color("#6C7086"), // overlay0
	Subtext:   lipgloss.Color("#A6ADC8"), // subtext0
	Border:    lipgloss.Color("#585B70"), // surface2
	Bg:        lipgloss.Color("#1E1E2E"), // base
	Fg:        lipgloss.Color("#CDD6F4"), // text
	BgDim:     lipgloss.Color("#181825"), // mantle
}

var catppuccinFrappe = Theme{
	Name:      "catppuccin-frappe",
	Primary:   lipgloss.Color("#CA9EE6"), // mauve
	Secondary: lipgloss.Color("#8CAAEE"), // blue
	Success:   lipgloss.Color("#A6D189"), // green
	Error:     lipgloss.Color("#E78284"), // red
	Warning:   lipgloss.Color("#E5C890"), // yellow
	Info:      lipgloss.Color("#99D1DB"), // sky
	Muted:     lipgloss.Color("#737994"), // overlay0
	Subtext:   lipgloss.Color("#A5ADCE"), // subtext0
	Border:    lipgloss.Color("#626880"), // surface2
	Bg:        lipgloss.Color("#303446"), // base
	Fg:        lipgloss.Color("#C6D0F5"), // text
	BgDim:     lipgloss.Color("#292C3C"), // mantle
}

var tokyoNight = Theme{
	Name:      "tokyo-night",
	Primary:   lipgloss.Color("#BB9AF7"),
	Secondary: lipgloss.Color("#7AA2F7"),
	Success:   lipgloss.Color("#9ECE6A"),
	Error:     lipgloss.Color("#F7768E"),
	Warning:   lipgloss.Color("#E0AF68"),
	Info:      lipgloss.Color("#7DCFFF"),
	Muted:     lipgloss.Color("#565F89"),
	Subtext:   lipgloss.Color("#A9B1D6"),
	Border:    lipgloss.Color("#3B4261"),
	Bg:        lipgloss.Color("#1A1B26"),
	Fg:        lipgloss.Color("#C0CAF5"),
	BgDim:     lipgloss.Color("#16161E"),
}

var dracula = Theme{
	Name:      "dracula",
	Primary:   lipgloss.Color("#BD93F9"),
	Secondary: lipgloss.Color("#8BE9FD"),
	Success:   lipgloss.Color("#50FA7B"),
	Error:     lipgloss.Color("#FF5555"),
	Warning:   lipgloss.Color("#F1FA8C"),
	Info:      lipgloss.Color("#8BE9FD"),
	Muted:     lipgloss.Color("#6272A4"),
	Subtext:   lipgloss.Color("#BFBFBF"),
	Border:    lipgloss.Color("#6272A4"),
	Bg:        lipgloss.Color("#282A36"),
	Fg:        lipgloss.Color("#F8F8F2"),
	BgDim:     lipgloss.Color("#21222C"),
}

// ThemeByName returns a theme by name. Falls back to catppuccin-mocha.
func ThemeByName(name string) Theme {
	if t, ok := themes[name]; ok {
		return t
	}
	return catppuccinMocha
}

// ThemeNames returns all available theme names.
func ThemeNames() []string {
	names := make([]string, 0, len(themes))
	for k := range themes {
		names = append(names, k)
	}
	return names
}

// themeOrder defines the cycling order for themes.
var themeOrder = []string{
	"catppuccin-mocha",
	"catppuccin-frappe",
	"tokyo-night",
	"dracula",
	"default-dark",
}

// NextTheme returns the next theme in the cycle after the given theme name.
func NextTheme(current string) Theme {
	for i, name := range themeOrder {
		if name == current {
			next := themeOrder[(i+1)%len(themeOrder)]
			return themes[next]
		}
	}
	return themes[themeOrder[0]]
}
