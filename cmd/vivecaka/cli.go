package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/indrasvat/vivecaka/internal/config"
	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/indrasvat/vivecaka/internal/tui/core"
)

const (
	repoEnvVar  = "VIVECAKA_REPO"
	debugEnvVar = "VIVECAKA_DEBUG"
)

type cliOptions struct {
	debug       bool
	repo        domain.RepoRef
	repoSource  string
	showVersion bool
}

type cliEnvDefaults struct {
	debug   bool
	repoRaw string
}

func executeCLI(args []string, stdout, stderr io.Writer, getenv func(string) string) error {
	envDefaults := optionsFromEnv(getenv)
	cmd := newRootCommand(envDefaults, stdout, stderr, runApp)
	cmd.SetArgs(args)
	return cmd.Execute()
}

func newRootCommand(envDefaults cliEnvDefaults, stdout, stderr io.Writer, run func(cliOptions) error) *cobra.Command {
	opts := cliOptions{
		debug: envDefaults.debug,
	}
	theme := helpTheme()
	helpRenderer := newCLIHelpRenderer(theme, opts, envDefaults)

	cmd := &cobra.Command{
		Use:           "vivecaka",
		Short:         "Keyboard-first GitHub PR TUI",
		Long:          "vivecaka is a keyboard-first terminal UI for reviewing and triaging GitHub pull requests.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.showVersion {
				_, err := fmt.Fprint(cmd.OutOrStdout(), formatVersion())
				return err
			}
			if opts.repo.Owner == "" && envDefaults.repoRaw != "" {
				repo, err := parseRepoRef(envDefaults.repoRaw)
				if err != nil {
					return fmt.Errorf("invalid %s: %w", repoEnvVar, err)
				}
				opts.repo = repo
				opts.repoSource = repoEnvVar
			}
			return run(opts)
		},
	}

	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		_, _ = fmt.Fprint(cmd.OutOrStdout(), helpRenderer.Render(cmd))
	})
	cmd.SetUsageFunc(func(cmd *cobra.Command) error {
		_, err := fmt.Fprint(cmd.ErrOrStderr(), helpRenderer.Render(cmd))
		return err
	})

	cmd.Flags().BoolVarP(&opts.showVersion, "version", "v", false, "Show version information")
	cmd.Flags().BoolVarP(&opts.debug, "debug", "d", opts.debug, "Enable debug logging")

	repoValue := newRepoFlagValue(&opts.repo, &opts.repoSource)
	if opts.repo.Owner != "" {
		repoValue.set = true
	}
	cmd.Flags().Var(&repoValue, "repo", "Start in a specific repository (owner/name)")

	return cmd
}

func formatVersion() string {
	return fmt.Sprintf("vivecaka %s\n  commit:  %s\n  built:   %s\n  go:      %s\n", version, commit, date, goVersion)
}

func optionsFromEnv(getenv func(string) string) cliEnvDefaults {
	return cliEnvDefaults{
		debug:   parseBoolEnv(getenv(debugEnvVar)),
		repoRaw: strings.TrimSpace(getenv(repoEnvVar)),
	}
}

func parseBoolEnv(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func parseRepoRef(value string) (domain.RepoRef, error) {
	repo := strings.TrimSpace(value)
	owner, name, found := strings.Cut(repo, "/")
	if !found || owner == "" || name == "" || strings.Contains(name, "/") {
		return domain.RepoRef{}, fmt.Errorf("expected owner/name, got %q", value)
	}
	return domain.RepoRef{Owner: owner, Name: name}, nil
}

type repoFlagValue struct {
	target       *domain.RepoRef
	sourceTarget *string
	set          bool
}

func newRepoFlagValue(target *domain.RepoRef, sourceTarget *string) repoFlagValue {
	return repoFlagValue{target: target, sourceTarget: sourceTarget}
}

func (v *repoFlagValue) String() string {
	if v == nil || !v.set || v.target == nil || v.target.Owner == "" {
		return ""
	}
	return v.target.String()
}

func (v *repoFlagValue) Set(value string) error {
	repo, err := parseRepoRef(value)
	if err != nil {
		return err
	}
	if v.target != nil {
		*v.target = repo
	}
	if v.sourceTarget != nil {
		*v.sourceTarget = "flag"
	}
	v.set = true
	return nil
}

func (v *repoFlagValue) Type() string { return "owner/name" }

func helpTheme() core.Theme {
	cfg, err := config.Load()
	if err != nil {
		return core.ThemeByName(config.Default().General.Theme)
	}
	return core.ThemeByName(cfg.General.Theme)
}

type cliHelpRenderer struct {
	theme       core.Theme
	opts        cliOptions
	envDefaults cliEnvDefaults
}

func newCLIHelpRenderer(theme core.Theme, opts cliOptions, envDefaults cliEnvDefaults) cliHelpRenderer {
	return cliHelpRenderer{theme: theme, opts: opts, envDefaults: envDefaults}
}

func (r cliHelpRenderer) Render(cmd *cobra.Command) string {
	t := r.theme

	frameStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border).
		Background(t.Bg).
		Padding(1, 2).
		Width(78)

	titleStyle := lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true)
	subtitleStyle := lipgloss.NewStyle().
		Foreground(t.Subtext)
	sectionStyle := lipgloss.NewStyle().
		Foreground(t.Info).
		Bold(true)
	keyStyle := lipgloss.NewStyle().
		Foreground(t.Secondary).
		Bold(true)
	textStyle := lipgloss.NewStyle().
		Foreground(t.Fg)
	mutedStyle := lipgloss.NewStyle().
		Foreground(t.Muted)
	valueStyle := lipgloss.NewStyle().
		Foreground(t.Warning)

	flagLabelWidth := 20
	renderRow := func(label, desc string) string {
		return keyStyle.Render(fmt.Sprintf("%-*s", flagLabelWidth, label)) + textStyle.Render(desc)
	}

	usage := []string{
		sectionStyle.Render("Usage"),
		textStyle.Render("  vivecaka [flags]"),
		"",
		sectionStyle.Render("Examples"),
		textStyle.Render("  vivecaka"),
		textStyle.Render("  vivecaka --repo indrasvat/vivecaka"),
		textStyle.Render("  vivecaka --debug"),
		textStyle.Render("  vivecaka --help"),
		"",
		sectionStyle.Render("Flags"),
		renderRow("-h, --help", "Show this help"),
		renderRow("-v, --version", "Show version information"),
		renderRow("-d, --debug", "Enable debug logging"),
		renderRow("--repo owner/name", "Start in a specific repository"),
		"",
		sectionStyle.Render("Environment"),
		renderRow(debugEnvVar, "Enable debug logging when set to 1/true"),
		renderRow(repoEnvVar, "Default repository override (owner/name)"),
	}

	if r.opts.repo.Owner != "" {
		usage = append(usage,
			"",
			sectionStyle.Render("Resolved Defaults"),
			renderRow("repo", valueStyle.Render(r.opts.repo.String())+" "+mutedStyle.Render("("+r.opts.repoSource+")")),
		)
	} else if r.envDefaults.repoRaw != "" {
		usage = append(usage,
			"",
			sectionStyle.Render("Resolved Defaults"),
			renderRow("repo", mutedStyle.Render(r.envDefaults.repoRaw)+" "+mutedStyle.Render("(from env; validated on run)")),
		)
	}
	if r.opts.debug {
		usage = append(usage, renderRow("debug", valueStyle.Render("enabled")+" "+mutedStyle.Render("(env/flag default)")))
	}

	body := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(cmd.Name()+"  "+mutedStyle.Render(version)),
		subtitleStyle.Render("Keyboard-first GitHub PR triage with themed terminal workflows."),
		"",
		lipgloss.JoinVertical(lipgloss.Left, usage...),
	)

	return frameStyle.Render(body) + "\n"
}
