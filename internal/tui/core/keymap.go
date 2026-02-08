package core

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all keybindings for the application.
type KeyMap struct {
	// Global
	Quit       key.Binding
	Help       key.Binding
	RepoSwitch key.Binding
	Refresh    key.Binding
	ThemeCycle key.Binding

	// Navigation
	Up           key.Binding
	Down         key.Binding
	PageUp       key.Binding
	PageDown     key.Binding
	HalfPageUp   key.Binding
	HalfPageDown key.Binding
	Top          key.Binding
	Bottom       key.Binding

	// Actions
	Enter    key.Binding
	Back     key.Binding
	Tab      key.Binding
	ShiftTab key.Binding
	Search   key.Binding
	Filter   key.Binding
	Sort     key.Binding
	Yank     key.Binding
	Open     key.Binding
	Checkout key.Binding
}

// DefaultKeyMap returns the default keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		RepoSwitch: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("C-r", "switch repo"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "refresh"),
		),
		ThemeCycle: key.NewBinding(
			key.WithKeys("T"),
			key.WithHelp("T", "cycle theme"),
		),

		// Navigation
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/↑", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/↓", "down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("pgdn", "page down"),
		),
		HalfPageUp: key.NewBinding(
			key.WithKeys("ctrl+u"),
			key.WithHelp("C-u", "½ page up"),
		),
		HalfPageDown: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("C-d", "½ page down"),
		),
		Top: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "top"),
		),
		Bottom: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "bottom"),
		),

		// Actions
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("↵", "open"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next pane"),
		),
		ShiftTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("S-tab", "prev pane"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		Filter: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "filter"),
		),
		Sort: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "sort"),
		),
		Yank: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "copy URL"),
		),
		Open: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "open in browser"),
		),
		Checkout: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "checkout"),
		),
	}
}

// ApplyOverrides applies keybinding overrides from a string map.
// Keys are binding names (e.g., "quit", "search"), values are key strings (e.g., "ctrl+q").
func (k *KeyMap) ApplyOverrides(overrides map[string]string) {
	for name, keyStr := range overrides {
		binding := k.bindingByName(name)
		if binding != nil {
			binding.SetKeys(keyStr)
		}
	}
}

func (k *KeyMap) bindingByName(name string) *key.Binding {
	switch name {
	case "quit":
		return &k.Quit
	case "help":
		return &k.Help
	case "repo_switch":
		return &k.RepoSwitch
	case "refresh":
		return &k.Refresh
	case "theme_cycle":
		return &k.ThemeCycle
	case "up":
		return &k.Up
	case "down":
		return &k.Down
	case "page_up":
		return &k.PageUp
	case "page_down":
		return &k.PageDown
	case "half_page_up":
		return &k.HalfPageUp
	case "half_page_down":
		return &k.HalfPageDown
	case "top":
		return &k.Top
	case "bottom":
		return &k.Bottom
	case "enter":
		return &k.Enter
	case "back":
		return &k.Back
	case "tab":
		return &k.Tab
	case "shift_tab":
		return &k.ShiftTab
	case "search":
		return &k.Search
	case "filter":
		return &k.Filter
	case "sort":
		return &k.Sort
	case "yank":
		return &k.Yank
	case "open":
		return &k.Open
	case "checkout":
		return &k.Checkout
	default:
		return nil
	}
}

// ShortHelp returns the key bindings for the condensed help view.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Back, k.Help, k.Quit}
}

// FullHelp returns the key bindings for the full help view.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.PageUp, k.PageDown, k.HalfPageUp, k.HalfPageDown, k.Top, k.Bottom},
		{k.Enter, k.Back, k.Tab, k.ShiftTab, k.Search, k.Filter, k.Sort},
		{k.Yank, k.Open, k.Checkout, k.Refresh, k.ThemeCycle, k.RepoSwitch},
		{k.Help, k.Quit},
	}
}
