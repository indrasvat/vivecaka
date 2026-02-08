package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitDisabled(t *testing.T) {
	// When debug=false, Init should be a no-op.
	err := Init(false)
	require.NoError(t, err)
	// Log should discard.
	Log.Info("this should not appear anywhere")
}

func TestInitEnabled(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmp)

	err := Init(true)
	require.NoError(t, err)
	defer Close()

	// Log a message.
	Log.Info("test message", "key", "value")

	// Verify file was created.
	path := LogPath()
	_, err = os.Stat(path)
	require.NoError(t, err, "log file should exist")

	// Verify content.
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "logger initialized")
	assert.Contains(t, content, "test message")
	assert.Contains(t, content, "key=value")
}

func TestLogPath(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmp)

	path := LogPath()
	expected := filepath.Join(tmp, "vivecaka", "debug.log")
	assert.Equal(t, expected, path)
}

func TestLogPathDefault(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "")
	path := LogPath()
	assert.Contains(t, path, ".local/state/vivecaka/debug.log")
}

func TestRotation(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmp)

	path := LogPath()
	err := os.MkdirAll(filepath.Dir(path), 0o755)
	require.NoError(t, err)

	// Create a large log file that exceeds maxLogSize.
	largeData := strings.Repeat("x", maxLogSize+100)
	err = os.WriteFile(path, []byte(largeData), 0o644)
	require.NoError(t, err)

	// Init should rotate.
	err = Init(true)
	require.NoError(t, err)
	defer Close()

	// Old file should be rotated.
	rotated := path + ".1"
	_, err = os.Stat(rotated)
	assert.NoError(t, err, "rotated file should exist")

	// New log file should be small (just the init message).
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.LessOrEqual(t, info.Size(), int64(1000), "expected small new log")
}
