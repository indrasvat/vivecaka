package ghcli

import (
	"bytes"
	"fmt"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/indrasvat/vivecaka/internal/plugin"
)

// Adapter implements the ghcli plugin providing PR data via the gh CLI.
// It implements plugin.Plugin, domain.PRReader, domain.PRReviewer, and domain.PRWriter.
type Adapter struct {
	// ghPath is the resolved path to the gh binary.
	ghPath string
}

// New creates a new GH CLI adapter.
func New() *Adapter {
	return &Adapter{}
}

// Info returns plugin metadata.
func (a *Adapter) Info() plugin.PluginInfo {
	return plugin.PluginInfo{
		Name:        "ghcli",
		Version:     "1.0.0",
		Description: "GitHub CLI adapter using go-gh",
		Provides:    []string{"pr-reader", "pr-reviewer", "pr-writer"},
	}
}

// Check verifies that the gh CLI is installed and authenticated.
// Call this before starting the TUI to fail fast with a clear message.
func (a *Adapter) Check() error {
	ghPath, err := exec.LookPath("gh")
	if err != nil {
		return fmt.Errorf("gh CLI not found: install from https://cli.github.com")
	}
	a.ghPath = ghPath

	cmd := exec.Command(ghPath, "auth", "status")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh not authenticated: run 'gh auth login'")
	}
	return nil
}

// Init satisfies the plugin.Plugin interface.
func (a *Adapter) Init(_ plugin.AppContext) tea.Cmd {
	return nil
}
