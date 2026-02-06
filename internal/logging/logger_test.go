package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitDisabled(t *testing.T) {
	// When debug=false, Init should be a no-op.
	if err := Init(false); err != nil {
		t.Fatalf("Init(false): %v", err)
	}
	// Log should discard.
	Log.Info("this should not appear anywhere")
}

func TestInitEnabled(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmp)

	if err := Init(true); err != nil {
		t.Fatalf("Init(true): %v", err)
	}
	defer Close()

	// Log a message.
	Log.Info("test message", "key", "value")

	// Verify file was created.
	path := LogPath()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("log file not found: %v", err)
	}

	// Verify content.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "logger initialized") {
		t.Error("expected 'logger initialized' in log")
	}
	if !strings.Contains(content, "test message") {
		t.Error("expected 'test message' in log")
	}
	if !strings.Contains(content, "key=value") {
		t.Error("expected 'key=value' in log")
	}
}

func TestLogPath(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmp)

	path := LogPath()
	expected := filepath.Join(tmp, "vivecaka", "debug.log")
	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}
}

func TestLogPathDefault(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "")
	path := LogPath()
	if !strings.Contains(path, ".local/state/vivecaka/debug.log") {
		t.Errorf("unexpected path: %s", path)
	}
}

func TestRotation(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmp)

	path := LogPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a large log file that exceeds maxLogSize.
	largeData := strings.Repeat("x", maxLogSize+100)
	if err := os.WriteFile(path, []byte(largeData), 0o644); err != nil {
		t.Fatal(err)
	}

	// Init should rotate.
	if err := Init(true); err != nil {
		t.Fatalf("Init: %v", err)
	}
	defer Close()

	// Old file should be rotated.
	rotated := path + ".1"
	if _, err := os.Stat(rotated); err != nil {
		t.Error("rotated file not found")
	}

	// New log file should be small (just the init message).
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() > 1000 {
		t.Errorf("expected small new log, got %d bytes", info.Size())
	}
}
