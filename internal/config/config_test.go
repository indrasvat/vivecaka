package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := Default()

	if cfg.General.Theme != "default-dark" {
		t.Errorf("default theme = %q, want %q", cfg.General.Theme, "default-dark")
	}
	if cfg.General.RefreshInterval != 30 {
		t.Errorf("default refresh_interval = %d, want 30", cfg.General.RefreshInterval)
	}
	if cfg.General.DefaultSort != "updated" {
		t.Errorf("default default_sort = %q, want %q", cfg.General.DefaultSort, "updated")
	}
	if cfg.General.DefaultFilter != "open" {
		t.Errorf("default default_filter = %q, want %q", cfg.General.DefaultFilter, "open")
	}
	if cfg.General.PageSize != 50 {
		t.Errorf("default page_size = %d, want 50", cfg.General.PageSize)
	}
	if !cfg.General.ShowBanner {
		t.Error("default show_banner = false, want true")
	}
	if cfg.General.CacheTTL != 5 {
		t.Errorf("default cache_ttl = %d, want 5", cfg.General.CacheTTL)
	}
	if cfg.General.StaleDays != 7 {
		t.Errorf("default stale_days = %d, want 7", cfg.General.StaleDays)
	}
	if cfg.Diff.Mode != "unified" {
		t.Errorf("default diff mode = %q, want %q", cfg.Diff.Mode, "unified")
	}
	if !cfg.Diff.LineNumbers {
		t.Error("default line_numbers = false, want true")
	}
	if cfg.Diff.ContextLines != 3 {
		t.Errorf("default context_lines = %d, want 3", cfg.Diff.ContextLines)
	}
	if cfg.Diff.MarkdownStyle != "dark" {
		t.Errorf("default markdown_style = %q, want %q", cfg.Diff.MarkdownStyle, "dark")
	}
	if !cfg.Notifications.NewPRs {
		t.Error("default new_prs = false, want true")
	}
	if !cfg.Notifications.ReviewRequests {
		t.Error("default review_requests = false, want true")
	}
	if !cfg.Notifications.CIChanges {
		t.Error("default ci_changes = false, want true")
	}
}

func TestDefaultConfigValidates(t *testing.T) {
	cfg := Default()
	if err := cfg.Validate(); err != nil {
		t.Errorf("Default().Validate() = %v, want nil", err)
	}
}

func TestValidateInvalidRefreshInterval(t *testing.T) {
	cfg := Default()
	cfg.General.RefreshInterval = -1
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() with negative refresh_interval should return error")
	}
}

func TestValidateInvalidPageSize(t *testing.T) {
	cfg := Default()
	cfg.General.PageSize = 0
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() with zero page_size should return error")
	}
}

func TestValidateInvalidCacheTTL(t *testing.T) {
	cfg := Default()
	cfg.General.CacheTTL = -1
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() with negative cache_ttl should return error")
	}
}

func TestValidateInvalidStaleDays(t *testing.T) {
	cfg := Default()
	cfg.General.StaleDays = -1
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() with negative stale_days should return error")
	}
}

func TestValidateInvalidSort(t *testing.T) {
	cfg := Default()
	cfg.General.DefaultSort = "invalid"
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() with invalid default_sort should return error")
	}
}

func TestValidateInvalidFilter(t *testing.T) {
	cfg := Default()
	cfg.General.DefaultFilter = "bogus"
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() with invalid default_filter should return error")
	}
}

func TestValidateInvalidContextLines(t *testing.T) {
	cfg := Default()
	cfg.Diff.ContextLines = -1
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() with negative context_lines should return error")
	}
}

func TestValidateInvalidDiffMode(t *testing.T) {
	cfg := Default()
	cfg.Diff.Mode = "side-by-side"
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() with invalid diff mode should return error")
	}
}

func TestValidateInvalidMarkdownStyle(t *testing.T) {
	cfg := Default()
	cfg.Diff.MarkdownStyle = "neon"
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() with invalid markdown_style should return error")
	}
}

func TestValidateAcceptsAllValidSorts(t *testing.T) {
	for _, sort := range validSorts {
		cfg := Default()
		cfg.General.DefaultSort = sort
		if err := cfg.Validate(); err != nil {
			t.Errorf("Validate() with sort=%q returned error: %v", sort, err)
		}
	}
}

func TestValidateAcceptsAllValidFilters(t *testing.T) {
	for _, filter := range validFilters {
		cfg := Default()
		cfg.General.DefaultFilter = filter
		if err := cfg.Validate(); err != nil {
			t.Errorf("Validate() with filter=%q returned error: %v", filter, err)
		}
	}
}

func TestValidateRefreshIntervalZeroIsValid(t *testing.T) {
	cfg := Default()
	cfg.General.RefreshInterval = 0
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() with refresh_interval=0 should be valid, got: %v", err)
	}
}

func TestLoadFromCreatesDefaultOnMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vivecaka", "config.toml")

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}

	// Should return defaults.
	if cfg.General.Theme != "default-dark" {
		t.Errorf("theme = %q, want default", cfg.General.Theme)
	}

	// Should have created the file.
	if _, err := os.Stat(path); err != nil {
		t.Errorf("config file not created: %v", err)
	}
}

func TestLoadFromCustomConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	custom := `
[general]
theme = "catppuccin"
page_size = 25
refresh_interval = 60

[diff]
mode = "split"
`
	if err := os.WriteFile(path, []byte(custom), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}

	if cfg.General.Theme != "catppuccin" {
		t.Errorf("theme = %q, want %q", cfg.General.Theme, "catppuccin")
	}
	if cfg.General.PageSize != 25 {
		t.Errorf("page_size = %d, want 25", cfg.General.PageSize)
	}
	if cfg.General.RefreshInterval != 60 {
		t.Errorf("refresh_interval = %d, want 60", cfg.General.RefreshInterval)
	}
	if cfg.Diff.Mode != "split" {
		t.Errorf("diff.mode = %q, want %q", cfg.Diff.Mode, "split")
	}
}

func TestLoadFromMergesOverDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	// Partial config - only override theme.
	partial := `
[general]
theme = "mocha"
`
	if err := os.WriteFile(path, []byte(partial), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}

	// Overridden value.
	if cfg.General.Theme != "mocha" {
		t.Errorf("theme = %q, want %q", cfg.General.Theme, "mocha")
	}
	// Default values should still be present.
	if cfg.Diff.ContextLines != 3 {
		t.Errorf("context_lines = %d, want 3 (default)", cfg.Diff.ContextLines)
	}
	if !cfg.Notifications.NewPRs {
		t.Error("new_prs should default to true")
	}
}

func TestLoadFromInvalidTOML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	if err := os.WriteFile(path, []byte("not valid toml [[["), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	_, err := LoadFrom(path)
	if err == nil {
		t.Error("LoadFrom() with invalid TOML should return error")
	}
}

func TestLoadFromInvalidValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	invalid := `
[general]
page_size = -5
`
	if err := os.WriteFile(path, []byte(invalid), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	_, err := LoadFrom(path)
	if err == nil {
		t.Error("LoadFrom() with invalid values should return error")
	}
}
