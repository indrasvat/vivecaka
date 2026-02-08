package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := Default()

	assert.Equal(t, "default-dark", cfg.General.Theme)
	assert.Equal(t, 30, cfg.General.RefreshInterval)
	assert.Equal(t, "updated", cfg.General.DefaultSort)
	assert.Equal(t, "open", cfg.General.DefaultFilter)
	assert.Equal(t, 50, cfg.General.PageSize)
	assert.True(t, cfg.General.ShowBanner)
	assert.Equal(t, 5, cfg.General.CacheTTL)
	assert.Equal(t, 7, cfg.General.StaleDays)
	assert.Equal(t, "unified", cfg.Diff.Mode)
	assert.True(t, cfg.Diff.LineNumbers)
	assert.Equal(t, 3, cfg.Diff.ContextLines)
	assert.Equal(t, "dark", cfg.Diff.MarkdownStyle)
	assert.True(t, cfg.Notifications.NewPRs)
	assert.True(t, cfg.Notifications.ReviewRequests)
	assert.True(t, cfg.Notifications.CIChanges)
}

func TestDefaultConfigValidates(t *testing.T) {
	cfg := Default()
	err := cfg.Validate()
	assert.NoError(t, err)
}

func TestValidateInvalidRefreshInterval(t *testing.T) {
	cfg := Default()
	cfg.General.RefreshInterval = -1
	err := cfg.Validate()
	assert.Error(t, err, "Validate() with negative refresh_interval should return error")
}

func TestValidateInvalidPageSize(t *testing.T) {
	cfg := Default()
	cfg.General.PageSize = 0
	err := cfg.Validate()
	assert.Error(t, err, "Validate() with zero page_size should return error")
}

func TestValidateInvalidCacheTTL(t *testing.T) {
	cfg := Default()
	cfg.General.CacheTTL = -1
	err := cfg.Validate()
	assert.Error(t, err, "Validate() with negative cache_ttl should return error")
}

func TestValidateInvalidStaleDays(t *testing.T) {
	cfg := Default()
	cfg.General.StaleDays = -1
	err := cfg.Validate()
	assert.Error(t, err, "Validate() with negative stale_days should return error")
}

func TestValidateInvalidSort(t *testing.T) {
	cfg := Default()
	cfg.General.DefaultSort = "invalid"
	err := cfg.Validate()
	assert.Error(t, err, "Validate() with invalid default_sort should return error")
}

func TestValidateInvalidFilter(t *testing.T) {
	cfg := Default()
	cfg.General.DefaultFilter = "bogus"
	err := cfg.Validate()
	assert.Error(t, err, "Validate() with invalid default_filter should return error")
}

func TestValidateInvalidContextLines(t *testing.T) {
	cfg := Default()
	cfg.Diff.ContextLines = -1
	err := cfg.Validate()
	assert.Error(t, err, "Validate() with negative context_lines should return error")
}

func TestValidateInvalidDiffMode(t *testing.T) {
	cfg := Default()
	cfg.Diff.Mode = "side-by-side"
	err := cfg.Validate()
	assert.Error(t, err, "Validate() with invalid diff mode should return error")
}

func TestValidateInvalidMarkdownStyle(t *testing.T) {
	cfg := Default()
	cfg.Diff.MarkdownStyle = "neon"
	err := cfg.Validate()
	assert.Error(t, err, "Validate() with invalid markdown_style should return error")
}

func TestValidateAcceptsAllValidSorts(t *testing.T) {
	for _, sort := range validSorts {
		cfg := Default()
		cfg.General.DefaultSort = sort
		err := cfg.Validate()
		assert.NoError(t, err, "Validate() with sort=%q should not return error", sort)
	}
}

func TestValidateAcceptsAllValidFilters(t *testing.T) {
	for _, filter := range validFilters {
		cfg := Default()
		cfg.General.DefaultFilter = filter
		err := cfg.Validate()
		assert.NoError(t, err, "Validate() with filter=%q should not return error", filter)
	}
}

func TestValidateRefreshIntervalZeroIsValid(t *testing.T) {
	cfg := Default()
	cfg.General.RefreshInterval = 0
	err := cfg.Validate()
	assert.NoError(t, err, "Validate() with refresh_interval=0 should be valid")
}

func TestLoadFromCreatesDefaultOnMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vivecaka", "config.toml")

	cfg, err := LoadFrom(path)
	require.NoError(t, err)

	// Should return defaults.
	assert.Equal(t, "default-dark", cfg.General.Theme)

	// Should have created the file.
	_, err = os.Stat(path)
	assert.NoError(t, err, "config file should be created")
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
	err := os.WriteFile(path, []byte(custom), 0o644)
	require.NoError(t, err)

	cfg, err := LoadFrom(path)
	require.NoError(t, err)

	assert.Equal(t, "catppuccin", cfg.General.Theme)
	assert.Equal(t, 25, cfg.General.PageSize)
	assert.Equal(t, 60, cfg.General.RefreshInterval)
	assert.Equal(t, "split", cfg.Diff.Mode)
}

func TestLoadFromMergesOverDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	// Partial config - only override theme.
	partial := `
[general]
theme = "mocha"
`
	err := os.WriteFile(path, []byte(partial), 0o644)
	require.NoError(t, err)

	cfg, err := LoadFrom(path)
	require.NoError(t, err)

	// Overridden value.
	assert.Equal(t, "mocha", cfg.General.Theme)
	// Default values should still be present.
	assert.Equal(t, 3, cfg.Diff.ContextLines, "context_lines should be default")
	assert.True(t, cfg.Notifications.NewPRs, "new_prs should default to true")
}

func TestLoadFromInvalidTOML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	err := os.WriteFile(path, []byte("not valid toml [[["), 0o644)
	require.NoError(t, err)

	_, err = LoadFrom(path)
	assert.Error(t, err, "LoadFrom() with invalid TOML should return error")
}

func TestUpdateFavorites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	initial := `
[general]
theme = "default-dark"

[repos]
favorites = ["acme/frontend"]
`
	err := os.WriteFile(path, []byte(initial), 0o644)
	require.NoError(t, err)

	cfg, err := LoadFrom(path)
	require.NoError(t, err)

	// Update favorites.
	err = cfg.UpdateFavorites([]string{"acme/frontend", "acme/backend", "other/lib"})
	require.NoError(t, err)

	// Reload and verify.
	cfg2, err := LoadFrom(path)
	require.NoError(t, err)
	require.Len(t, cfg2.Repos.Favorites, 3)
	assert.Equal(t, "acme/frontend", cfg2.Repos.Favorites[0])
	assert.Equal(t, "acme/backend", cfg2.Repos.Favorites[1])
}

func TestUpdateFavoritesNewFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "config.toml")

	cfg := Default()
	// path is not set, but we set it manually.
	cfg2 := *cfg
	cfg2.path = path

	err := cfg2.UpdateFavorites([]string{"acme/frontend"})
	require.NoError(t, err)

	// Verify file was created.
	reloaded, err := LoadFrom(path)
	require.NoError(t, err)
	assert.Len(t, reloaded.Repos.Favorites, 1)
}

func TestLoadFromInvalidValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	invalid := `
[general]
page_size = -5
`
	err := os.WriteFile(path, []byte(invalid), 0o644)
	require.NoError(t, err)

	_, err = LoadFrom(path)
	assert.Error(t, err, "LoadFrom() with invalid values should return error")
}
