package main

import (
	"fmt"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"

	"github.com/indrasvat/vivecaka/internal/adapter/ghcli"
	"github.com/indrasvat/vivecaka/internal/config"
	"github.com/indrasvat/vivecaka/internal/logging"
	"github.com/indrasvat/vivecaka/internal/tui"
)

const (
	terminalBackgroundHex = "#1E1E2E"
	resetBackgroundOSC    = "\x1b]111\x07"
)

func runApp(opts cliOptions) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	debug := opts.debug || cfg.General.Debug
	if err := logging.Init(debug); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "warning: failed to initialize logger: %v\n", err)
	}

	repoOverride := ""
	if opts.repo.Owner != "" {
		repoOverride = opts.repo.String()
	}

	logging.Log.Info("starting vivecaka",
		"version", version,
		"commit", commit,
		"debug_flag", opts.debug,
		"debug_config", cfg.General.Debug,
		"repo_override", repoOverride,
	)

	adapter := ghcli.New()
	if err := adapter.Check(); err != nil {
		return err
	}

	appOptions := []tui.Option{
		tui.WithVersion(version),
		tui.WithReader(adapter),
		tui.WithReviewer(adapter),
		tui.WithWriter(adapter),
		tui.WithRepoManager(adapter),
	}
	if opts.repo.Owner != "" {
		appOptions = append(appOptions, tui.WithRepo(opts.repo))
	}

	app := tui.New(cfg, appOptions...)

	// Set the terminal background for the TUI's lifetime so empty cells render
	// correctly, then explicitly reset it with OSC 111 on exit. output.Reset()
	// only sends SGR reset (\x1b[0m), which does not undo OSC 11.
	setTerminalBackground(os.Stdout)
	defer resetTerminalBackground(os.Stdout)

	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, runErr := p.Run()

	logging.Log.Info("shutting down", "error", runErr)
	logging.Close()

	if runErr != nil {
		return runErr
	}
	return nil
}

func setTerminalBackground(w io.Writer) {
	output := termenv.NewOutput(w)
	output.SetBackgroundColor(termenv.RGBColor(terminalBackgroundHex))
}

func resetTerminalBackground(w io.Writer) {
	_, _ = io.WriteString(w, resetBackgroundOSC)
}
