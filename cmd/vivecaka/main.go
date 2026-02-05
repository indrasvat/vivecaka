package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"

	"github.com/indrasvat/vivecaka/internal/adapter/ghcli"
	"github.com/indrasvat/vivecaka/internal/config"
	"github.com/indrasvat/vivecaka/internal/tui"
)

// Build-time variables injected via ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("vivecaka %s (%s) built %s\n", version, commit, date)
		os.Exit(0)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}

	adapter := ghcli.New()

	app := tui.New(cfg,
		tui.WithVersion(version),
		tui.WithReader(adapter),
		tui.WithReviewer(adapter),
		tui.WithWriter(adapter),
	)

	// Set terminal background color to match theme (Catppuccin Mocha base)
	// This fills ALL cells including empty ones, preventing background bleeding
	output := termenv.NewOutput(os.Stdout)
	output.SetBackgroundColor(output.Color("#1E1E2E"))

	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, runErr := p.Run()

	// Reset terminal colors BEFORE os.Exit (defer won't run after os.Exit)
	output.Reset()

	if runErr != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", runErr)
		os.Exit(1)
	}
}
