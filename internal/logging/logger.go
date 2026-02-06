package logging

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

const (
	maxLogSize = 10 * 1024 * 1024 // 10 MB
	appName    = "vivecaka"
)

// Log is the global logger. It's a no-op by default until Init is called.
var Log *slog.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))

// logFile holds the open file handle for cleanup.
var logFile *os.File

// Init sets up the logger. If debug is false, it stays as a no-op.
func Init(debug bool) error {
	if !debug {
		return nil
	}

	path := LogPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	// Rotate if the log file is too large.
	if info, err := os.Stat(path); err == nil && info.Size() > maxLogSize {
		rotated := path + ".1"
		_ = os.Remove(rotated)
		_ = os.Rename(path, rotated)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	logFile = f

	Log = slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	Log.Info("logger initialized", "path", path)
	return nil
}

// Close flushes and closes the log file.
func Close() {
	if logFile != nil {
		_ = logFile.Close()
		logFile = nil
	}
}

// LogPath returns the log file path in the XDG state directory.
func LogPath() string {
	if dir := os.Getenv("XDG_STATE_HOME"); dir != "" {
		return filepath.Join(dir, appName, "debug.log")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "state", appName, "debug.log")
}
