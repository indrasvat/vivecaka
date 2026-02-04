package plugin

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/indrasvat/vivecaka/internal/domain"
)

// Plugin is the base interface all plugins must implement.
type Plugin interface {
	Info() PluginInfo
	Init(app AppContext) tea.Cmd
}

// PluginInfo contains metadata about a plugin.
type PluginInfo struct {
	Name        string   // unique identifier: "ghcli", "github-api"
	Version     string   // semver
	Description string   // human-readable
	Provides    []string // capabilities: "pr-reader", "pr-reviewer"
}

// AppContext provides plugins access to application state during Init().
type AppContext interface {
	ConfigValue(key string) any
	ThemeName() string
	CurrentRepo() domain.RepoRef
	SendMessage(tea.Msg)
}

// ViewPlugin provides custom UI views.
type ViewPlugin interface {
	Plugin
	Views() []ViewRegistration
}

// ViewRegistration describes a custom view provided by a plugin.
type ViewRegistration struct {
	Name     string    // Unique view name
	Title    string    // Display title
	Position string    // "tab", "overlay", "pane"
	Model    tea.Model // BubbleTea model for the view
}

// KeyPlugin provides custom key bindings.
type KeyPlugin interface {
	Plugin
	KeyBindings() []KeyRegistration
}

// KeyRegistration describes a custom key binding provided by a plugin.
type KeyRegistration struct {
	Key    key.Binding    // The key binding
	View   string         // Which view this applies to ("" = global)
	Action func() tea.Cmd // Action to execute
}
