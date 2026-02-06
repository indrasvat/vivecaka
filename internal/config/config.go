package config

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/pelletier/go-toml/v2"
)

// Config holds all application configuration.
type Config struct {
	General       GeneralConfig       `toml:"general"`
	Diff          DiffConfig          `toml:"diff"`
	Repos         ReposConfig         `toml:"repos"`
	Keybindings   map[string]string   `toml:"keybindings"`
	Notifications NotificationsConfig `toml:"notifications"`

	path string `toml:"-"` // source file path (not serialized)
}

// GeneralConfig holds general settings.
type GeneralConfig struct {
	Theme           string `toml:"theme"`
	RefreshInterval int    `toml:"refresh_interval"`
	DefaultSort     string `toml:"default_sort"`
	DefaultFilter   string `toml:"default_filter"`
	PageSize        int    `toml:"page_size"`
	ShowBanner      bool   `toml:"show_banner"`
	CacheTTL        int    `toml:"cache_ttl"`
	StaleDays       int    `toml:"stale_days"`
	Debug           bool   `toml:"debug"`
}

// DiffConfig holds diff viewer settings.
type DiffConfig struct {
	Mode          string `toml:"mode"`
	ExternalTool  string `toml:"external_tool"`
	LineNumbers   bool   `toml:"line_numbers"`
	ContextLines  int    `toml:"context_lines"`
	MarkdownStyle string `toml:"markdown_style"`
}

// ReposConfig holds repository settings.
type ReposConfig struct {
	Favorites []string `toml:"favorites"`
}

// NotificationsConfig holds notification settings.
type NotificationsConfig struct {
	NewPRs         bool `toml:"new_prs"`
	ReviewRequests bool `toml:"review_requests"`
	CIChanges      bool `toml:"ci_changes"`
}

// Default returns the default configuration.
func Default() *Config {
	return &Config{
		General: GeneralConfig{
			Theme:           "default-dark",
			RefreshInterval: 30,
			DefaultSort:     "updated",
			DefaultFilter:   "open",
			PageSize:        50,
			ShowBanner:      true,
			CacheTTL:        5,
			StaleDays:       7,
		},
		Diff: DiffConfig{
			Mode:          "unified",
			LineNumbers:   true,
			ContextLines:  3,
			MarkdownStyle: "dark",
		},
		Keybindings: make(map[string]string),
		Notifications: NotificationsConfig{
			NewPRs:         true,
			ReviewRequests: true,
			CIChanges:      true,
		},
	}
}

var (
	validSorts   = []string{"updated", "created", "number", "title", "author"}
	validFilters = []string{"open", "closed", "merged", "all"}
	validModes   = []string{"unified", "split"}
	validStyles  = []string{"dark", "light", "notty"}
)

// Validate checks config values and returns the first error found.
func (c *Config) Validate() error {
	if c.General.RefreshInterval < 0 {
		return fmt.Errorf("general.refresh_interval must be >= 0, got %d", c.General.RefreshInterval)
	}
	if c.General.PageSize <= 0 {
		return fmt.Errorf("general.page_size must be > 0, got %d", c.General.PageSize)
	}
	if c.General.CacheTTL < 0 {
		return fmt.Errorf("general.cache_ttl must be >= 0, got %d", c.General.CacheTTL)
	}
	if c.General.StaleDays < 0 {
		return fmt.Errorf("general.stale_days must be >= 0, got %d", c.General.StaleDays)
	}
	if c.General.DefaultSort != "" && !slices.Contains(validSorts, c.General.DefaultSort) {
		return fmt.Errorf("general.default_sort must be one of %v, got %q", validSorts, c.General.DefaultSort)
	}
	if c.General.DefaultFilter != "" && !slices.Contains(validFilters, c.General.DefaultFilter) {
		return fmt.Errorf("general.default_filter must be one of %v, got %q", validFilters, c.General.DefaultFilter)
	}
	if c.Diff.ContextLines < 0 {
		return fmt.Errorf("diff.context_lines must be >= 0, got %d", c.Diff.ContextLines)
	}
	if c.Diff.Mode != "" && !slices.Contains(validModes, c.Diff.Mode) {
		return fmt.Errorf("diff.mode must be one of %v, got %q", validModes, c.Diff.Mode)
	}
	if c.Diff.MarkdownStyle != "" && !slices.Contains(validStyles, c.Diff.MarkdownStyle) {
		return fmt.Errorf("diff.markdown_style must be one of %v, got %q", validStyles, c.Diff.MarkdownStyle)
	}
	return nil
}

// ConfigPath returns the path from which this config was loaded (empty if default).
func (c *Config) ConfigPath() string {
	return c.path
}

// UpdateFavorites writes the favorites list back to the config TOML file.
// If the config was loaded from a file, it updates that file. Otherwise it
// writes to the default XDG config path.
func (c *Config) UpdateFavorites(favorites []string) error {
	c.Repos.Favorites = favorites

	path := c.path
	if path == "" {
		path = filepath.Join(ConfigDir(), "config.toml")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	out, err := toml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	return os.WriteFile(path, out, 0o644)
}

// Load reads configuration from the XDG config path, creating defaults if needed.
func Load() (*Config, error) {
	return LoadFrom(filepath.Join(ConfigDir(), "config.toml"))
}

// LoadFrom reads configuration from a specific path, creating defaults if the file doesn't exist.
func LoadFrom(path string) (*Config, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			if mkErr := os.MkdirAll(filepath.Dir(path), 0o755); mkErr != nil {
				return cfg, nil
			}
			out, merr := toml.Marshal(cfg)
			if merr == nil {
				_ = os.WriteFile(path, out, 0o644)
			}
			return cfg, nil
		}
		return cfg, nil
	}

	if err := toml.Unmarshal(data, cfg); err != nil {
		return Default(), err
	}

	if err := cfg.Validate(); err != nil {
		return Default(), err
	}

	cfg.path = path
	return cfg, nil
}
