package views

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/indrasvat/vivecaka/internal/tui/core"
	"github.com/indrasvat/vivecaka/internal/usecase"
)

// checkoutDialogState tracks which phase the dialog is in.
type checkoutDialogState int

const (
	checkoutInactive       checkoutDialogState = iota
	checkoutWorktreeChoice                     // Mock B: CWD=correct repo, choose switch vs worktree
	checkoutKnownConfirm                       // Mock C: known path confirmation
	checkoutOptions                            // Mock D: clone/path/browser options
	checkoutCustomPath                         // Mock I: text input for custom clone path
	checkoutCloning                            // Mock E: clone in progress
	checkoutCheckingOut                        // Mock F: checkout running
	checkoutSuccess                            // Mock G/H: result with optional copy
	checkoutError                              // error result
)

// CheckoutDialogModel is the multi-state checkout dialog.
type CheckoutDialogModel struct {
	state  checkoutDialogState
	styles core.Styles
	keys   core.KeyMap
	width  int
	height int

	// Context
	repo     domain.RepoRef
	prNumber int
	branch   string

	// Strategy from SmartCheckout.Plan()
	plan usecase.CheckoutPlan

	// State-specific
	cursor       int             // for choice/options lists
	spinnerFrame int             // for loading states
	pathInput    textinput.Model // for custom path
	resultBranch string          // checkout result branch name
	resultPath   string          // where checkout happened
	resultErr    error           // error if any
	cwdCheckout  bool            // true if checkout was in CWD (no path to show)
}

// Checkout dialog messages.

// CheckoutStrategyChosenMsg is sent when user picks a strategy from the dialog.
type CheckoutStrategyChosenMsg struct {
	Strategy string // "switch", "worktree", "clone-cache", "clone-custom", "browser"
	Path     string // target path (for clone-custom or known-path)
	Repo     domain.RepoRef
	PRNumber int
	Branch   string
}

// CheckoutDialogCloseMsg is sent when the dialog should close.
type CheckoutDialogCloseMsg struct{}

// CopyCdCommandMsg is sent when user wants to copy the cd command.
type CopyCdCommandMsg struct {
	Path string
}

// CloneDoneMsg is sent when a clone operation finishes.
type CloneDoneMsg struct {
	Path string
	Err  error
}

// SmartCheckoutDoneMsg is sent when a smart checkout (at a non-CWD path) finishes.
type SmartCheckoutDoneMsg struct {
	Branch string
	Path   string
	Err    error
}

// checkoutDialogSpinnerTick drives the loading spinner.
type checkoutDialogSpinnerTick struct{}

// NewCheckoutDialogModel creates a new checkout dialog.
func NewCheckoutDialogModel(styles core.Styles, keys core.KeyMap) CheckoutDialogModel {
	ti := textinput.New()
	ti.Prompt = "❯ "
	ti.CharLimit = 256
	return CheckoutDialogModel{
		state:     checkoutInactive,
		styles:    styles,
		keys:      keys,
		pathInput: ti,
	}
}

// SetStyles updates styles without losing state.
func (m *CheckoutDialogModel) SetStyles(s core.Styles) { m.styles = s }

// SetSize updates dimensions.
func (m *CheckoutDialogModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// Active returns whether the dialog is currently visible.
func (m *CheckoutDialogModel) Active() bool {
	return m.state != checkoutInactive
}

// GetPRNumber returns the PR number the dialog is currently handling.
func (m *CheckoutDialogModel) GetPRNumber() int {
	return m.prNumber
}

// IsLoading returns whether the dialog is in a loading state.
func (m *CheckoutDialogModel) IsLoading() bool {
	return m.state == checkoutCloning || m.state == checkoutCheckingOut
}

// ShowWorktreeChoice shows the dialog for choosing between switch-branch and worktree.
func (m *CheckoutDialogModel) ShowWorktreeChoice(repo domain.RepoRef, prNumber int, branch string, plan usecase.CheckoutPlan) {
	m.state = checkoutWorktreeChoice
	m.repo = repo
	m.prNumber = prNumber
	m.branch = branch
	m.plan = plan
	m.cursor = 0
}

// ShowKnownConfirm shows the dialog for confirming checkout at a known path.
func (m *CheckoutDialogModel) ShowKnownConfirm(repo domain.RepoRef, prNumber int, branch string, plan usecase.CheckoutPlan) {
	m.state = checkoutKnownConfirm
	m.repo = repo
	m.prNumber = prNumber
	m.branch = branch
	m.plan = plan
}

// ShowOptions shows the dialog for clone/browser options when no local clone exists.
func (m *CheckoutDialogModel) ShowOptions(repo domain.RepoRef, prNumber int, branch string, plan usecase.CheckoutPlan) {
	m.state = checkoutOptions
	m.repo = repo
	m.prNumber = prNumber
	m.branch = branch
	m.plan = plan
	m.cursor = 0
}

// ShowCloning transitions to the clone-in-progress state.
func (m *CheckoutDialogModel) ShowCloning(path string) tea.Cmd {
	m.state = checkoutCloning
	m.resultPath = path
	m.spinnerFrame = 0
	return m.spinnerTick()
}

// ShowCheckingOut transitions to the checkout-in-progress state.
func (m *CheckoutDialogModel) ShowCheckingOut(path string) tea.Cmd {
	m.state = checkoutCheckingOut
	m.resultPath = path
	m.spinnerFrame = 0
	return m.spinnerTick()
}

// ShowSuccess transitions to the success result state.
func (m *CheckoutDialogModel) ShowSuccess(branch, path string, cwdCheckout bool) {
	m.state = checkoutSuccess
	m.resultBranch = branch
	m.resultPath = path
	m.cwdCheckout = cwdCheckout
}

// ShowError transitions to the error result state.
func (m *CheckoutDialogModel) ShowError(err error) {
	m.state = checkoutError
	m.resultErr = err
}

func (m *CheckoutDialogModel) spinnerTick() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(_ time.Time) tea.Msg {
		return checkoutDialogSpinnerTick{}
	})
}

// Update handles messages for the checkout dialog.
func (m *CheckoutDialogModel) Update(msg tea.Msg) tea.Cmd {
	switch typedMsg := msg.(type) {
	case checkoutDialogSpinnerTick:
		if m.IsLoading() {
			m.spinnerFrame++
			return m.spinnerTick()
		}
		return nil

	case tea.KeyMsg:
		return m.handleKey(typedMsg)
	}

	// Forward to text input if in custom path state.
	if m.state == checkoutCustomPath {
		var cmd tea.Cmd
		m.pathInput, cmd = m.pathInput.Update(msg)
		return cmd
	}

	return nil
}

func (m *CheckoutDialogModel) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch m.state {
	case checkoutWorktreeChoice:
		return m.handleWorktreeKey(msg)
	case checkoutKnownConfirm:
		return m.handleKnownConfirmKey(msg)
	case checkoutOptions:
		return m.handleOptionsKey(msg)
	case checkoutCustomPath:
		return m.handleCustomPathKey(msg)
	case checkoutCloning, checkoutCheckingOut:
		// No key interaction during loading (except Esc for cancel in future DR-4)
		return nil
	case checkoutSuccess:
		return m.handleSuccessKey(msg)
	case checkoutError:
		// Any key dismisses.
		m.reset()
		return func() tea.Msg { return CheckoutDialogCloseMsg{} }
	}
	return nil
}

func (m *CheckoutDialogModel) handleWorktreeKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case msg.Type == tea.KeyEscape:
		m.reset()
		return func() tea.Msg { return CheckoutDialogCloseMsg{} }
	case msg.Type == tea.KeyEnter:
		strategy := "switch"
		if m.cursor == 1 {
			strategy = "worktree"
		}
		return func() tea.Msg {
			return CheckoutStrategyChosenMsg{
				Strategy: strategy,
				Repo:     m.repo,
				PRNumber: m.prNumber,
				Branch:   m.branch,
			}
		}
	case key.Matches(msg, m.keys.Down) || msg.Type == tea.KeyDown:
		if m.cursor < 1 {
			m.cursor++
		}
	case key.Matches(msg, m.keys.Up) || msg.Type == tea.KeyUp:
		if m.cursor > 0 {
			m.cursor--
		}
	}
	return nil
}

func (m *CheckoutDialogModel) handleKnownConfirmKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.Type {
	case tea.KeyEscape:
		m.reset()
		return func() tea.Msg { return CheckoutDialogCloseMsg{} }
	case tea.KeyEnter:
		path := m.plan.TargetPath
		return func() tea.Msg {
			return CheckoutStrategyChosenMsg{
				Strategy: "known-path",
				Path:     path,
				Repo:     m.repo,
				PRNumber: m.prNumber,
				Branch:   m.branch,
			}
		}
	case tea.KeyRunes:
		if len(msg.Runes) == 1 && (msg.Runes[0] == 'n' || msg.Runes[0] == 'N') {
			m.reset()
			return func() tea.Msg { return CheckoutDialogCloseMsg{} }
		}
	}
	return nil
}

func (m *CheckoutDialogModel) handleOptionsKey(msg tea.KeyMsg) tea.Cmd {
	numOptions := 3 // clone-cache, clone-custom, browser
	switch {
	case msg.Type == tea.KeyEscape:
		m.reset()
		return func() tea.Msg { return CheckoutDialogCloseMsg{} }
	case msg.Type == tea.KeyEnter:
		return m.selectOption()
	case key.Matches(msg, m.keys.Down) || msg.Type == tea.KeyDown:
		if m.cursor < numOptions-1 {
			m.cursor++
		}
	case key.Matches(msg, m.keys.Up) || msg.Type == tea.KeyUp:
		if m.cursor > 0 {
			m.cursor--
		}
	}
	return nil
}

func (m *CheckoutDialogModel) selectOption() tea.Cmd {
	switch m.cursor {
	case 0: // Clone to vivecaka cache
		return func() tea.Msg {
			return CheckoutStrategyChosenMsg{
				Strategy: "clone-cache",
				Path:     m.plan.CacheClonePath,
				Repo:     m.repo,
				PRNumber: m.prNumber,
				Branch:   m.branch,
			}
		}
	case 1: // Clone to custom path
		m.state = checkoutCustomPath
		m.pathInput.SetValue(defaultClonePath(m.repo))
		m.pathInput.Focus()
		m.pathInput.CursorEnd()
		return textinput.Blink
	case 2: // Open on GitHub
		return func() tea.Msg {
			return CheckoutStrategyChosenMsg{
				Strategy: "browser",
				Repo:     m.repo,
				PRNumber: m.prNumber,
				Branch:   m.branch,
			}
		}
	}
	return nil
}

func (m *CheckoutDialogModel) handleCustomPathKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.Type {
	case tea.KeyEscape:
		// Go back to options, not close entirely.
		m.state = checkoutOptions
		m.cursor = 1
		m.pathInput.Blur()
		return nil
	case tea.KeyEnter:
		rawPath := m.pathInput.Value()
		resolved := expandPath(rawPath)
		if resolved == "" {
			return nil // Don't accept empty path.
		}
		m.pathInput.Blur()
		return func() tea.Msg {
			return CheckoutStrategyChosenMsg{
				Strategy: "clone-custom",
				Path:     resolved,
				Repo:     m.repo,
				PRNumber: m.prNumber,
				Branch:   m.branch,
			}
		}
	default:
		var cmd tea.Cmd
		m.pathInput, cmd = m.pathInput.Update(msg)
		return cmd
	}
}

func (m *CheckoutDialogModel) handleSuccessKey(msg tea.KeyMsg) tea.Cmd {
	if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 && (msg.Runes[0] == 'y' || msg.Runes[0] == 'Y') {
		if !m.cwdCheckout && m.resultPath != "" {
			path := m.resultPath
			m.reset()
			return func() tea.Msg { return CopyCdCommandMsg{Path: path} }
		}
	}
	// Any other key dismisses.
	m.reset()
	return func() tea.Msg { return CheckoutDialogCloseMsg{} }
}

func (m *CheckoutDialogModel) reset() {
	m.state = checkoutInactive
	m.repo = domain.RepoRef{}
	m.prNumber = 0
	m.branch = ""
	m.cursor = 0
	m.resultBranch = ""
	m.resultPath = ""
	m.resultErr = nil
	m.cwdCheckout = false
	m.pathInput.Blur()
	m.pathInput.SetValue("")
}

// View renders the dialog based on current state.
func (m *CheckoutDialogModel) View() string {
	switch m.state {
	case checkoutWorktreeChoice:
		return m.viewWorktreeChoice()
	case checkoutKnownConfirm:
		return m.viewKnownConfirm()
	case checkoutOptions:
		return m.viewOptions()
	case checkoutCustomPath:
		return m.viewCustomPath()
	case checkoutCloning:
		return m.viewCloning()
	case checkoutCheckingOut:
		return m.viewCheckingOut()
	case checkoutSuccess:
		return m.viewSuccess()
	case checkoutError:
		return m.viewError()
	}
	return ""
}

// StatusHint returns status bar text based on current dialog state.
func (m *CheckoutDialogModel) StatusHint() string {
	switch m.state {
	case checkoutCloning:
		return "Cloning..."
	case checkoutCheckingOut:
		return "Checking out..."
	case checkoutSuccess, checkoutError:
		return "Press any key to continue"
	case checkoutCustomPath:
		return "Enter path   Esc back"
	default:
		return "j/k select   Enter confirm   Esc cancel"
	}
}

// ── View renderers ──

func (m *CheckoutDialogModel) viewWorktreeChoice() string {
	t := m.styles.Theme
	boxWidth := m.boxWidth()

	title := m.titleStyle().Render(fmt.Sprintf("Checkout PR #%d", m.prNumber))
	branchLine := lipgloss.NewStyle().
		Foreground(t.Info).
		Render(m.branch)

	options := []struct {
		label string
		desc  string
	}{
		{"Switch branch", "Replaces current branch"},
		{"New worktree", ".worktrees/pr-" + fmt.Sprintf("%d-%s", m.prNumber, sanitizeBranch(m.branch))},
	}

	var optLines []string
	for i, opt := range options {
		cursor := "  "
		labelStyle := lipgloss.NewStyle().Foreground(t.Fg)
		if i == m.cursor {
			cursor = lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render("▸") + " "
			labelStyle = labelStyle.Bold(true)
		}
		descStyle := lipgloss.NewStyle().Foreground(t.Subtext).Width(boxWidth - 8)
		optLines = append(optLines, cursor+labelStyle.Render(opt.label))
		optLines = append(optLines, "    "+descStyle.Render(opt.desc))
		if i < len(options)-1 {
			optLines = append(optLines, "")
		}
	}

	hints := m.hintLine("j/k select   Enter confirm   Esc cancel")

	inner := lipgloss.JoinVertical(lipgloss.Left,
		append([]string{title, branchLine, ""},
			append(optLines, "", hints)...)...,
	)
	return m.renderBox(inner, boxWidth, string(t.Primary))
}

func (m *CheckoutDialogModel) viewKnownConfirm() string {
	t := m.styles.Theme
	boxWidth := m.boxWidth()

	title := m.titleStyle().Render(fmt.Sprintf("Checkout PR #%d", m.prNumber))
	repoLine := lipgloss.NewStyle().Foreground(t.Info).Render(m.repo.String()) +
		lipgloss.NewStyle().Foreground(t.Muted).Render(" → ") +
		lipgloss.NewStyle().Foreground(t.Info).Render(m.branch)

	pathLabel := lipgloss.NewStyle().Foreground(t.Fg).Render("Will check out in:")
	pathValue := lipgloss.NewStyle().
		Foreground(t.Warning).
		Bold(true).
		Width(boxWidth - 6).
		Render(shortenPath(m.plan.TargetPath))

	hints := m.confirmHints()

	inner := lipgloss.JoinVertical(lipgloss.Left,
		title, repoLine, "", pathLabel, pathValue, "", hints,
	)
	return m.renderBox(inner, boxWidth, string(t.Primary))
}

func (m *CheckoutDialogModel) viewOptions() string {
	t := m.styles.Theme
	boxWidth := m.boxWidth()

	warning := lipgloss.NewStyle().Foreground(t.Warning).Bold(true).Render("⚠ No local clone found")
	repoLine := lipgloss.NewStyle().Foreground(t.Fg).
		Width(boxWidth - 6).
		Render(m.repo.String() + " is not cloned locally.")

	options := []struct {
		label string
		desc  string
	}{
		{"Clone to vivecaka cache", shortenPath(m.plan.CacheClonePath)},
		{"Clone to custom path...", "Enter a directory path"},
		{"Open on GitHub", fmt.Sprintf("View PR #%d in browser", m.prNumber)},
	}

	var optLines []string
	for i, opt := range options {
		cursor := "  "
		labelStyle := lipgloss.NewStyle().Foreground(t.Fg)
		if i == m.cursor {
			cursor = lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render("▸") + " "
			labelStyle = labelStyle.Bold(true)
		}
		descStyle := lipgloss.NewStyle().Foreground(t.Subtext).Width(boxWidth - 8)
		optLines = append(optLines, cursor+labelStyle.Render(opt.label))
		optLines = append(optLines, "    "+descStyle.Render(opt.desc))
		if i < len(options)-1 {
			optLines = append(optLines, "")
		}
	}

	hints := m.hintLine("j/k select   Enter confirm   Esc cancel")

	inner := lipgloss.JoinVertical(lipgloss.Left,
		append([]string{warning, "", repoLine, ""},
			append(optLines, "", hints)...)...,
	)
	return m.renderBox(inner, boxWidth, string(t.Warning))
}

func (m *CheckoutDialogModel) viewCustomPath() string {
	t := m.styles.Theme
	boxWidth := m.boxWidth()

	title := m.titleStyle().Render(fmt.Sprintf("Clone %s", m.repo.String()))
	label := lipgloss.NewStyle().Foreground(t.Subtext).Render("Enter path:")

	m.pathInput.Width = boxWidth - 8
	inputView := m.pathInput.View()

	hints := m.confirmHints()

	inner := lipgloss.JoinVertical(lipgloss.Left,
		title, "", label, inputView, "", hints,
	)
	return m.renderBox(inner, boxWidth, string(t.Primary))
}

func (m *CheckoutDialogModel) viewCloning() string {
	t := m.styles.Theme
	boxWidth := m.boxWidth()

	title := m.titleStyle().Render("Cloning Repository")
	frame := spinnerFrames[m.spinnerFrame%len(spinnerFrames)]
	spinner := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render(frame)
	msg := lipgloss.NewStyle().Foreground(t.Fg).Render(fmt.Sprintf("Cloning %s...", m.repo.String()))
	pathLine := lipgloss.NewStyle().Foreground(t.Muted).Italic(true).
		Width(boxWidth - 6).Render(shortenPath(m.resultPath))

	inner := lipgloss.JoinVertical(lipgloss.Left,
		title, "", spinner+" "+msg, "  "+pathLine,
	)
	return m.renderBox(inner, boxWidth, string(t.Primary))
}

func (m *CheckoutDialogModel) viewCheckingOut() string {
	t := m.styles.Theme
	boxWidth := m.boxWidth()

	title := m.titleStyle().Render("Checking Out")
	frame := spinnerFrames[m.spinnerFrame%len(spinnerFrames)]
	spinner := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render(frame)
	msg := lipgloss.NewStyle().Foreground(t.Fg).
		Width(boxWidth - 8).
		Render(fmt.Sprintf("Checking out \"%s\" for PR #%d...", m.branch, m.prNumber))
	pathLine := lipgloss.NewStyle().Foreground(t.Muted).Italic(true).
		Width(boxWidth - 6).Render("in " + shortenPath(m.resultPath))

	inner := lipgloss.JoinVertical(lipgloss.Left,
		title, "", spinner+" "+msg, "  "+pathLine,
	)
	return m.renderBox(inner, boxWidth, string(t.Primary))
}

func (m *CheckoutDialogModel) viewSuccess() string {
	t := m.styles.Theme
	boxWidth := m.boxWidth()

	icon := lipgloss.NewStyle().Foreground(t.Success).Bold(true).Render("✓")
	title := lipgloss.NewStyle().Foreground(t.Success).Bold(true).Render("Checkout Complete")

	var lines []string
	lines = append(lines, icon+" "+title, "")

	if m.cwdCheckout {
		// Simple CWD checkout — same as existing behavior (Mock H).
		msg := lipgloss.NewStyle().Foreground(t.Fg).Render("Checked out branch: " +
			lipgloss.NewStyle().Foreground(t.Info).Render(m.resultBranch))
		lines = append(lines, msg)
	} else {
		// Remote/cached checkout — show path (Mock G).
		branchLabel := lipgloss.NewStyle().Foreground(t.Subtext).Render("Branch: ")
		branchValue := lipgloss.NewStyle().Foreground(t.Info).Render(m.resultBranch)
		pathLabel := lipgloss.NewStyle().Foreground(t.Subtext).Render("Path:   ")
		pathValue := lipgloss.NewStyle().Foreground(t.Warning).Bold(true).
			Width(boxWidth - 14).Render(shortenPath(m.resultPath))

		cdCmd := "cd " + m.resultPath
		cdStyle := lipgloss.NewStyle().
			Foreground(t.Fg).
			Background(t.BgDim).
			Width(boxWidth - 6).
			Render(cdCmd)

		lines = append(lines, branchLabel+branchValue, pathLabel+pathValue, "", cdStyle)
	}

	lines = append(lines, "")

	// Hints.
	if m.cwdCheckout {
		lines = append(lines, m.hintLine("Press any key to continue"))
	} else {
		yKey := lipgloss.NewStyle().Foreground(t.Info).Bold(true).Render("y")
		anyKey := lipgloss.NewStyle().Foreground(t.Muted).Render("any key")
		lines = append(lines, yKey+" copy cd command   "+anyKey+" dismiss")
	}

	inner := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return m.renderBox(inner, boxWidth, string(t.Success))
}

func (m *CheckoutDialogModel) viewError() string {
	t := m.styles.Theme
	boxWidth := m.boxWidth()

	icon := lipgloss.NewStyle().Foreground(t.Error).Bold(true).Render("✗")
	title := lipgloss.NewStyle().Foreground(t.Error).Bold(true).Render("Checkout Failed")

	errMsg := "Unknown error"
	if m.resultErr != nil {
		errMsg = m.resultErr.Error()
	}
	msgStyle := lipgloss.NewStyle().Foreground(t.Fg).Width(boxWidth - 6)

	hint := lipgloss.NewStyle().Foreground(t.Muted).Italic(true).Render("Press any key to continue")

	inner := lipgloss.JoinVertical(lipgloss.Left,
		icon+" "+title, "", msgStyle.Render(errMsg), "", hint,
	)
	return m.renderBox(inner, boxWidth, string(t.Error))
}

// ── Helpers ──

func (m *CheckoutDialogModel) boxWidth() int {
	return max(30, min(60, m.width-4))
}

func (m *CheckoutDialogModel) titleStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(m.styles.Theme.Primary).
		Bold(true)
}

func (m *CheckoutDialogModel) hintLine(text string) string {
	return lipgloss.NewStyle().
		Foreground(m.styles.Theme.Muted).
		Render(text)
}

func (m *CheckoutDialogModel) confirmHints() string {
	t := m.styles.Theme
	enterKey := lipgloss.NewStyle().Foreground(t.Success).Bold(true).Render("Enter")
	escKey := lipgloss.NewStyle().Foreground(t.Error).Bold(true).Render("Esc")
	return enterKey + " confirm   " + escKey + " cancel"
}

func (m *CheckoutDialogModel) renderBox(inner string, boxWidth int, borderColor string) string {
	boxStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(borderColor)).
		Padding(1, 2).
		Width(boxWidth)

	box := boxStyle.Render(inner)
	centered := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
	return ensureExactHeight(centered, m.height, m.width)
}

func defaultClonePath(repo domain.RepoRef) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return repo.Owner + "/" + repo.Name
	}
	return filepath.Join(home, "code", repo.Owner, repo.Name)
}

func expandPath(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "~/") || trimmed == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			trimmed = filepath.Join(home, trimmed[1:])
		}
	}
	abs, err := filepath.Abs(trimmed)
	if err != nil {
		return trimmed
	}
	return abs
}

func shortenPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if after, ok := strings.CutPrefix(path, home); ok {
		return "~" + after
	}
	return path
}

func sanitizeBranch(branch string) string {
	r := strings.NewReplacer("/", "-", "\\", "-", ":", "-", " ", "-")
	return r.Replace(branch)
}
