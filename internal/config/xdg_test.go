package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigDirDefault(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	dir := ConfigDir()
	assert.True(t, strings.HasSuffix(dir, filepath.Join(".config", "vivecaka")))
}

func TestConfigDirXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/custom/config")
	dir := ConfigDir()
	want := filepath.Join("/custom/config", "vivecaka")
	assert.Equal(t, want, dir)
}

func TestDataDirDefault(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "")
	dir := DataDir()
	assert.True(t, strings.HasSuffix(dir, filepath.Join(".local", "share", "vivecaka")))
}

func TestDataDirXDG(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/custom/data")
	dir := DataDir()
	want := filepath.Join("/custom/data", "vivecaka")
	assert.Equal(t, want, dir)
}

func TestCacheDirDefault(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "")
	dir := CacheDir()
	assert.True(t, strings.HasSuffix(dir, filepath.Join(".cache", "vivecaka")))
}

func TestCacheDirXDG(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "/custom/cache")
	dir := CacheDir()
	want := filepath.Join("/custom/cache", "vivecaka")
	assert.Equal(t, want, dir)
}

func TestEnsureDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "a", "b", "c")

	err := EnsureDir(dir)
	require.NoError(t, err)

	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.True(t, info.IsDir(), "EnsureDir() should create a directory")
}

func TestEnsureDirIdempotent(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "exists")

	err := EnsureDir(dir)
	require.NoError(t, err)
	err = EnsureDir(dir)
	require.NoError(t, err, "second EnsureDir() should succeed")
}
