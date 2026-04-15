package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"

	"github.com/indrasvat/vivecaka/internal/adapter/ghcli"
	"github.com/indrasvat/vivecaka/internal/config"
	"github.com/indrasvat/vivecaka/internal/logging"
	"github.com/indrasvat/vivecaka/internal/tui"
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

	// Set terminal background color to match theme (Catppuccin Mocha base)
	// This fills ALL cells including empty ones, preventing background bleeding.
	output := termenv.NewOutput(os.Stdout)
	output.SetBackgroundColor(output.Color("#1E1E2E"))

	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, runErr := p.Run()

	// Reset terminal colors BEFORE returning (defer won't help with os.Exit in main).
	output.Reset()

	logging.Log.Info("shutting down", "error", runErr)
	logging.Close()

	if runErr != nil {
		return runErr
	}
	return nil
}
