package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigDirDefault(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	dir := ConfigDir()
	if !strings.HasSuffix(dir, filepath.Join(".config", "vivecaka")) {
		t.Errorf("ConfigDir() = %q, want suffix %q", dir, filepath.Join(".config", "vivecaka"))
	}
}

func TestConfigDirXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/custom/config")
	dir := ConfigDir()
	want := filepath.Join("/custom/config", "vivecaka")
	if dir != want {
		t.Errorf("ConfigDir() = %q, want %q", dir, want)
	}
}

func TestDataDirDefault(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "")
	dir := DataDir()
	if !strings.HasSuffix(dir, filepath.Join(".local", "share", "vivecaka")) {
		t.Errorf("DataDir() = %q, want suffix %q", dir, filepath.Join(".local", "share", "vivecaka"))
	}
}

func TestDataDirXDG(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/custom/data")
	dir := DataDir()
	want := filepath.Join("/custom/data", "vivecaka")
	if dir != want {
		t.Errorf("DataDir() = %q, want %q", dir, want)
	}
}

func TestCacheDirDefault(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "")
	dir := CacheDir()
	if !strings.HasSuffix(dir, filepath.Join(".cache", "vivecaka")) {
		t.Errorf("CacheDir() = %q, want suffix %q", dir, filepath.Join(".cache", "vivecaka"))
	}
}

func TestCacheDirXDG(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "/custom/cache")
	dir := CacheDir()
	want := filepath.Join("/custom/cache", "vivecaka")
	if dir != want {
		t.Errorf("CacheDir() = %q, want %q", dir, want)
	}
}

func TestEnsureDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "a", "b", "c")

	if err := EnsureDir(dir); err != nil {
		t.Fatalf("EnsureDir() error = %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Stat() after EnsureDir() error = %v", err)
	}
	if !info.IsDir() {
		t.Error("EnsureDir() should create a directory")
	}
}

func TestEnsureDirIdempotent(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "exists")

	if err := EnsureDir(dir); err != nil {
		t.Fatalf("first EnsureDir() error = %v", err)
	}
	if err := EnsureDir(dir); err != nil {
		t.Fatalf("second EnsureDir() should succeed, error = %v", err)
	}
}
