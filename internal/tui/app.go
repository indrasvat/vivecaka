package tui

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/indrasvat/vivecaka/internal/config"
	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/indrasvat/vivecaka/internal/tui/components"
	"github.com/indrasvat/vivecaka/internal/tui/core"
	"github.com/indrasvat/vivecaka/internal/tui/views"
	"github.com/indrasvat/vivecaka/internal/usecase"
)

// Option is a functional option for configuring the app.
type Option func(*App)

// WithVersion sets the version string.
func WithVersion(v string) Option {
	return func(a *App) { a.version = v }
}

// WithReader sets the PRReader adapter.
func WithReader(r domain.PRReader) Option {
	return func(a *App) { a.reader = r }
}

// WithReviewer sets the PRReviewer adapter.
func WithReviewer(r domain.PRReviewer) Option {
	return func(a *App) { a.reviewer = r }
}

// WithWriter sets the PRWriter adapter.
func WithWriter(w domain.PRWriter) Option {
	return func(a *App) { a.writer = w }
}

// WithRepo sets the initial repo (skips auto-detection).
func WithRepo(r domain.RepoRef) Option {
	return func(a *App) { a.repo = r }
}

// App is the root BubbleTea model.
type App struct {
	cfg     *config.Config
	version string

	// State
	view         core.ViewState
	prevView     core.ViewState
	width        int
	height       int
	ready        bool
	repo         domain.RepoRef
	username     string
	loadingFrame int // animation frame for loading spinner

	// Auto-refresh
	refreshCountdown int  // seconds until next refresh
	refreshPaused    bool // true when paused via 'p'
	refreshInterval  int  // from config (seconds); 0 = disabled
	prevPRCount      int  // track count for new-PR detection

	// Shared
	keys   core.KeyMap
	styles core.Styles
	theme  core.Theme

	// Domain (injected)
	reader   domain.PRReader
	reviewer domain.PRReviewer
	writer   domain.PRWriter

	// Use cases
	listPRs       *usecase.ListPRs
	getPRDetail   *usecase.GetPRDetail
	reviewPR      *usecase.ReviewPR
	checkoutPR    *usecase.CheckoutPR
	resolveThread *usecase.ResolveThread
	getInboxPRs   *usecase.GetInboxPRs

	// View models
	prList       views.PRListModel
	prDetail     views.PRDetailModel
	diffView     views.DiffViewModel
	reviewForm   views.ReviewModel
	repoSwitcher views.RepoSwitcherModel
	helpOverlay  views.HelpModel
	inbox        views.InboxModel
	tutorial     views.TutorialModel
	filterPanel  views.FilterModel

	// Overlays
	confirmDialog views.ConfirmModel

	// Filters
	filterOpts domain.ListOpts

	// Components
	banner *components.Banner
	header *components.Header
	status *components.StatusBar
	toasts *components.ToastManager
}

// New creates a new App model.
func New(cfg *config.Config, opts ...Option) *App {
	theme := core.ThemeByName(cfg.General.Theme)
	styles := core.NewStyles(theme)
	keys := core.DefaultKeyMap()

	// Apply keybinding overrides from config.
	if len(cfg.Keybindings) > 0 {
		keys.ApplyOverrides(cfg.Keybindings)
	}

	a := &App{
		cfg:    cfg,
		view:   core.ViewBanner,
		keys:   keys,
		theme:  theme,
		styles: styles,

		// View models
		prList:        views.NewPRListModel(styles, keys),
		prDetail:      views.NewPRDetailModel(styles, keys),
		diffView:      views.NewDiffViewModel(styles, keys),
		reviewForm:    views.NewReviewModel(styles, keys),
		repoSwitcher:  views.NewRepoSwitcherModel(styles, keys),
		helpOverlay:   views.NewHelpModel(styles),
		inbox:         views.NewInboxModel(styles, keys),
		tutorial:      views.NewTutorialModel(styles),
		filterPanel:   views.NewFilterModel(styles, keys),
		confirmDialog: views.NewConfirmModel(styles),

		// Components
		banner: nil, // initialized after options apply
		header: components.NewHeader(styles),
		status: components.NewStatusBar(styles),
		toasts: components.NewToastManager(styles),

		filterOpts:      domain.ListOpts{State: domain.PRStateOpen, Draft: domain.DraftInclude, PerPage: cfg.General.PageSize},
		refreshInterval: cfg.General.RefreshInterval,
	}

	for _, opt := range opts {
		opt(a)
	}

	// Initialize banner with version
	a.banner = components.NewBanner(styles, a.version)
	a.prList.SetPerPage(cfg.General.PageSize)
	a.prList.SetFilter(a.filterOpts)

	// Wire use cases from injected adapters.
	if a.reader != nil {
		a.listPRs = usecase.NewListPRs(a.reader)
		a.getPRDetail = usecase.NewGetPRDetail(a.reader)
		a.getInboxPRs = usecase.NewGetInboxPRs(a.reader)
	}
	if a.reviewer != nil {
		a.reviewPR = usecase.NewReviewPR(a.reviewer)
		a.resolveThread = usecase.NewResolveThread(a.reviewer)
	}
	if a.writer != nil {
		a.checkoutPR = usecase.NewCheckoutPR(a.writer)
	}

	// Initialize repo switcher with favorites from config.
	a.initRepoSwitcherFavorites()

	return a
}

func (a *App) Init() tea.Cmd {
	cmds := []tea.Cmd{
		detectUserCmd(),
		detectBranchCmd(),
		a.banner.StartAutoDismiss(2 * time.Second), // Show banner for 2 seconds
	}
	// Show tutorial on first launch.
	if views.IsFirstLaunch() {
		a.tutorial.Show()
	}
	// If repo was provided via WithRepo, skip detection and load PRs directly.
	if a.repo.Owner != "" {
		a.header.SetRepo(a.repo)
		if a.listPRs != nil {
			state := a.filterOpts.State
			if state == "" {
				state = domain.PRStateOpen
			}
			cmds = append(cmds, loadPRsCmd(a.listPRs, a.repo, a.filterOpts))
			cmds = append(cmds, loadPRCountCmd(a.reader, a.repo, state))
			return tea.Batch(cmds...)
		}
		cmds = append(cmds, func() tea.Msg { return viewReadyMsg{} })
		return tea.Batch(cmds...)
	}
	// Otherwise, detect repo from git remote.
	cmds = append(cmds, detectRepoCmd())
	return tea.Batch(cmds...)
}

type viewReadyMsg struct{}

// loadingTickMsg drives the loading screen spinner animation.
type loadingTickMsg struct{}

// refreshTickMsg fires every second for auto-refresh countdown.
type refreshTickMsg struct{}

func (a *App) loadingTick() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(_ time.Time) tea.Msg {
		return loadingTickMsg{}
	})
}

func (a *App) refreshTick() tea.Cmd {
	if a.refreshInterval <= 0 {
		return nil
	}
	return tea.Tick(1*time.Second, func(_ time.Time) tea.Msg {
		return refreshTickMsg{}
	})
}

func (a *App) startRefreshTimer() tea.Cmd {
	if a.refreshInterval <= 0 {
		return nil
	}
	a.refreshCountdown = a.refreshInterval
	a.header.SetRefreshCountdown(a.refreshCountdown, a.refreshPaused)
	return a.refreshTick()
}

func (a *App) handleRefreshTick() (tea.Model, tea.Cmd) {
	if a.refreshInterval <= 0 {
		return a, nil
	}
	if a.refreshPaused {
		a.header.SetRefreshCountdown(a.refreshCountdown, true)
		return a, a.refreshTick()
	}
	a.refreshCountdown--
	if a.refreshCountdown <= 0 {
		// Trigger auto-refresh.
		a.refreshCountdown = a.refreshInterval
		a.header.SetRefreshCountdown(a.refreshCountdown, false)
		if a.listPRs != nil && a.repo.Owner != "" && a.view == core.ViewPRList {
			a.prevPRCount = a.prList.TotalPRs()
			return a, tea.Batch(a.refreshTick(), loadPRsCmd(a.listPRs, a.repo, a.filterOpts))
		}
		return a, a.refreshTick()
	}
	a.header.SetRefreshCountdown(a.refreshCountdown, false)
	return a, a.refreshTick()
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return a.handleWindowSize(msg)

	case tea.KeyMsg:
		return a.handleKey(msg)

	case viewReadyMsg:
		a.view = core.ViewPRList
		return a, nil

	case views.RepoDetectedMsg:
		return a.handleRepoDetected(msg)

	case views.UserDetectedMsg:
		return a.handleUserDetected(msg)

	case cachedPRsLoadedMsg:
		if len(msg.PRs) > 0 && a.prList.IsLoading() {
			// Show cached data immediately while fresh load is in progress.
			a.prList.SetPRs(msg.PRs)
			a.header.SetPRCount(a.prList.TotalPRs())
		}
		return a, nil

	case views.BranchDetectedMsg:
		if msg.Err == nil && msg.Branch != "" {
			a.prList.SetCurrentBranch(msg.Branch)
			a.header.SetBranch(msg.Branch)
		}
		return a, nil

	case views.PRsLoadedMsg:
		return a.handlePRsLoaded(msg)

	case views.LoadMorePRsMsg:
		return a.handleLoadMorePRs(msg)

	case views.MorePRsLoadedMsg:
		return a.handleMorePRsLoaded(msg)

	case views.PRCountLoadedMsg:
		return a.handlePRCountLoaded(msg)

	case views.OpenPRMsg:
		return a.handleOpenPR(msg)

	case views.PRDetailLoadedMsg:
		return a.handlePRDetailLoaded(msg)

	case views.OpenDiffMsg:
		return a.handleOpenDiff(msg)

	case views.OpenExternalDiffMsg:
		return a.handleOpenExternalDiff(msg)

	case views.OpenFilterMsg:
		a.prevView = a.view
		a.view = core.ViewFilter
		a.filterPanel.SetOpts(a.filterOpts)
		return a, nil

	case views.ApplyFilterMsg:
		a.filterOpts = msg.Opts
		a.prList.SetFilter(msg.Opts)
		a.header.SetFilter(a.prList.FilterLabel())
		a.view = a.prevView
		if a.listPRs != nil && a.repo.Owner != "" {
			return a, loadPRsCmd(a.listPRs, a.repo, msg.Opts)
		}
		return a, nil

	case views.CloseFilterMsg:
		a.view = a.prevView
		return a, nil

	case views.DiffLoadedMsg:
		cmd := a.diffView.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return a, tea.Batch(cmds...)

	case views.StartReviewMsg:
		a.prevView = a.view
		a.view = core.ViewReview
		a.reviewForm.SetPRNumber(msg.Number)
		return a, a.reviewForm.Init()

	case views.SubmitReviewMsg:
		return a.handleSubmitReview(msg)

	case views.ReviewSubmittedMsg:
		return a.handleReviewSubmitted(msg)

	case views.CloseReviewMsg:
		a.view = core.ViewPRDetail
		return a, nil

	case views.CheckoutPRMsg:
		return a.handleCheckoutConfirm(msg)

	case views.ConfirmResultMsg:
		return a.handleConfirmResult(msg)

	case views.CloseConfirmMsg:
		a.view = a.prevView
		return a, nil

	case views.CheckoutDoneMsg:
		return a.handleCheckoutDone(msg)

	case views.CopyURLMsg:
		if err := copyToClipboard(msg.URL); err != nil {
			cmd := a.toasts.Add(
				fmt.Sprintf("Copy failed: %v", err),
				domain.ToastError, 5*time.Second,
			)
			return a, cmd
		}
		cmd := a.toasts.Add("Copied PR URL", domain.ToastSuccess, 3*time.Second)
		return a, cmd

	case views.OpenBrowserMsg:
		if err := openBrowser(msg.URL); err != nil {
			cmd := a.toasts.Add(
				fmt.Sprintf("Open browser failed: %v", err),
				domain.ToastError, 5*time.Second,
			)
			return a, cmd
		}
		return a, nil

	case views.BatchCopyURLsMsg:
		combined := strings.Join(msg.URLs, "\n")
		if err := copyToClipboard(combined); err != nil {
			cmd := a.toasts.Add(
				fmt.Sprintf("Copy failed: %v", err),
				domain.ToastError, 5*time.Second,
			)
			return a, cmd
		}
		cmd := a.toasts.Add(
			fmt.Sprintf("Copied %d PR URLs", len(msg.URLs)),
			domain.ToastSuccess, 3*time.Second,
		)
		return a, cmd

	case views.BatchOpenBrowserMsg:
		for _, url := range msg.URLs {
			if err := openBrowser(url); err != nil {
				cmd := a.toasts.Add(
					fmt.Sprintf("Open browser failed: %v", err),
					domain.ToastError, 5*time.Second,
				)
				return a, cmd
			}
		}
		cmd := a.toasts.Add(
			fmt.Sprintf("Opened %d PRs in browser", len(msg.URLs)),
			domain.ToastSuccess, 3*time.Second,
		)
		return a, cmd

	case views.PRListFilterMsg:
		a.header.SetFilter(msg.Label)
		return a, nil

	case views.SwitchRepoMsg:
		return a.handleSwitchRepo(msg)

	case views.CloseRepoSwitcherMsg:
		a.view = a.prevView
		return a, nil

	case views.ReposDiscoveredMsg:
		return a.handleReposDiscovered(msg)

	case views.ToggleFavoriteMsg:
		return a.handleToggleFavorite(msg)

	case views.ValidateRepoRequestMsg:
		return a, validateRepoCmd(msg.Repo)

	case views.RepoValidatedMsg:
		return a.handleRepoValidated(msg)

	case views.CloseHelpMsg:
		a.view = a.prevView
		return a, nil

	case views.OpenInboxPRMsg:
		return a.handleOpenInboxPR(msg)

	case views.CloseInboxMsg:
		a.view = core.ViewPRList
		return a, nil

	case views.ResolveThreadMsg:
		return a.handleResolveThread(msg)

	case views.UnresolveThreadMsg:
		// TODO: Implement unresolve (needs API support)
		cmd := a.toasts.Add("Unresolve not implemented yet", domain.ToastInfo, 3*time.Second)
		return a, cmd

	case views.ReplyToThreadMsg:
		// TODO: Implement reply (needs text input UI)
		cmd := a.toasts.Add("Reply not implemented yet", domain.ToastInfo, 3*time.Second)
		return a, cmd

	case resolveThreadDoneMsg:
		return a.handleResolveThreadDone(msg)

	case views.TutorialDoneMsg:
		if err := views.MarkTutorialDone(); err != nil {
			cmd := a.toasts.Add(
				fmt.Sprintf("Failed to persist tutorial state: %v", err),
				domain.ToastError, 5*time.Second,
			)
			return a, cmd
		}
		return a, nil

	case components.DismissToastMsg:
		a.toasts.Update(msg)
		return a, nil

	case components.BannerGlyphTickMsg:
		cmd := a.banner.Update(msg)
		return a, cmd

	case components.BannerDismissMsg:
		a.banner.Update(msg)
		if a.view == core.ViewBanner {
			// If PRs already loaded, go directly to PR list; otherwise loading
			if a.prList.HasPRs() {
				a.view = core.ViewPRList
			} else {
				a.view = core.ViewLoading
			}
		}
		// Force a full screen redraw to clear banner remnants
		// The tea.ClearScreen command clears the alt screen buffer
		return a, tea.Batch(tea.ClearScreen, a.loadingTick())

	case loadingTickMsg:
		if a.view == core.ViewLoading {
			a.loadingFrame++
			return a, a.loadingTick()
		}
		return a, nil

	case refreshTickMsg:
		return a.handleRefreshTick()

	case externalDiffDoneMsg:
		if msg.Err != nil {
			cmd := a.toasts.Add(
				fmt.Sprintf("External diff tool error: %v", msg.Err),
				domain.ToastError, 5*time.Second,
			)
			return a, cmd
		}
		return a, nil
	}

	// Dispatch to active view model for unhandled messages.
	cmd := a.updateActiveView(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	return a, tea.Batch(cmds...)
}

func (a *App) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	a.width = msg.Width
	a.height = msg.Height
	a.ready = true

	contentHeight := a.contentHeight()

	// Banner gets full screen.
	a.banner.SetSize(a.width, a.height)

	// Propagate to all view models.
	a.prList.SetSize(a.width, contentHeight)
	a.prDetail.SetSize(a.width, contentHeight)
	a.diffView.SetSize(a.width, contentHeight)
	a.reviewForm.SetSize(a.width, contentHeight)
	a.repoSwitcher.SetSize(a.width, contentHeight)
	a.helpOverlay.SetSize(a.width, contentHeight)
	a.inbox.SetSize(a.width, contentHeight)
	a.tutorial.SetSize(a.width, contentHeight)
	a.filterPanel.SetSize(a.width, contentHeight)
	a.confirmDialog.SetSize(a.width, contentHeight)

	// Components.
	a.header.SetWidth(a.width)
	a.status.SetWidth(a.width)
	a.toasts.SetWidth(a.width)

	return a, nil
}

func (a *App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Banner intercepts all keys when visible.
	if a.view == core.ViewBanner && a.banner.Visible() {
		// Quit key should exit immediately, not just dismiss the banner.
		if key.Matches(msg, a.keys.Quit) {
			return a, tea.Quit
		}
		a.banner.Update(msg)
		if a.prList.HasPRs() {
			a.view = core.ViewPRList
		} else {
			a.view = core.ViewLoading
		}
		return a, tea.Batch(tea.ClearScreen, a.loadingTick())
	}

	// Confirm dialog intercepts all keys when visible.
	if a.view == core.ViewConfirm {
		cmd := a.confirmDialog.Update(msg)
		return a, cmd
	}

	// Tutorial intercepts all keys when visible.
	if a.tutorial.Visible() {
		cmd := a.tutorial.Update(msg)
		return a, cmd
	}

	// Global keys always active.
	switch {
	case key.Matches(msg, a.keys.Quit):
		return a, tea.Quit

	case key.Matches(msg, a.keys.Help):
		if a.view == core.ViewHelp {
			a.view = a.prevView
		} else {
			a.prevView = a.view
			a.view = core.ViewHelp
			a.helpOverlay.SetContext(a.prevView)
		}
		return a, nil

	case key.Matches(msg, a.keys.ThemeCycle):
		a.theme = core.NextTheme(a.theme.Name)
		a.styles = core.NewStyles(a.theme)
		a.rebuildStyles()
		return a, nil

	case key.Matches(msg, a.keys.RepoSwitch):
		if a.view != core.ViewRepoSwitch {
			a.prevView = a.view
			a.view = core.ViewRepoSwitch
			// Mark current repo in the switcher.
			a.repoSwitcher.SetCurrentRepo(a.repo)
			// Trigger discovery on first open.
			if a.repoSwitcher.NeedsDiscovery() {
				a.repoSwitcher.SetDiscovering()
				return a, discoverReposCmd()
			}
		}
		return a, nil

	case key.Matches(msg, a.keys.Refresh):
		if a.view == core.ViewPRList && a.listPRs != nil && a.repo.Owner != "" {
			cmd := a.startRefreshTimer()
			return a, tea.Batch(cmd, loadPRsCmd(a.listPRs, a.repo, a.filterOpts))
		}
		return a, nil

	case key.Matches(msg, a.keys.Back):
		return a.handleBack()
	}

	// View-specific global keys (not in keymap).
	if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 {
		switch msg.Runes[0] {
		case 'p':
			if a.view == core.ViewPRList && a.refreshInterval > 0 {
				a.refreshPaused = !a.refreshPaused
				a.header.SetRefreshCountdown(a.refreshCountdown, a.refreshPaused)
				return a, nil
			}
		case 'I':
			if a.view == core.ViewPRList {
				return a.openInbox()
			}
		}
	}

	// Dispatch to active view.
	cmd := a.dispatchKeyToView(msg)
	return a, cmd
}

func (a *App) handleBack() (tea.Model, tea.Cmd) {
	switch a.view {
	case core.ViewHelp:
		a.view = a.prevView
	case core.ViewRepoSwitch:
		a.view = a.prevView
	case core.ViewPRDetail:
		a.view = core.ViewPRList
	case core.ViewDiff:
		a.view = core.ViewPRDetail
	case core.ViewReview:
		a.view = core.ViewPRDetail
	case core.ViewInbox:
		a.view = core.ViewPRList
	case core.ViewFilter:
		a.view = a.prevView
	case core.ViewConfirm:
		a.view = a.prevView
	}
	return a, nil
}

func (a *App) handleRepoDetected(msg views.RepoDetectedMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		cmd := a.toasts.Add(
			fmt.Sprintf("Could not detect repo: %v", msg.Err),
			domain.ToastError, 5*time.Second,
		)
		a.view = core.ViewPRList
		return a, cmd
	}
	a.repo = msg.Repo
	a.header.SetRepo(a.repo)
	a.header.SetTotalCount(0) // Reset total count for new repo
	// Prepend CWD repo to favorites if not already there.
	a.ensureCWDRepoInFavorites()
	a.repoSwitcher.SetCurrentRepo(a.repo)

	if a.listPRs != nil {
		// Load PRs and fetch total count in parallel.
		// Also try loading cached PRs for instant display.
		state := a.filterOpts.State
		if state == "" {
			state = domain.PRStateOpen
		}
		return a, tea.Batch(
			loadCachedPRsCmd(a.repo),
			loadPRsCmd(a.listPRs, a.repo, a.filterOpts),
			loadPRCountCmd(a.reader, a.repo, state),
		)
	}
	a.view = core.ViewPRList
	return a, nil
}

func (a *App) handleUserDetected(msg views.UserDetectedMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		cmd := a.toasts.Add(
			fmt.Sprintf("Could not detect user: %v", msg.Err),
			domain.ToastWarning, 5*time.Second,
		)
		return a, cmd
	}
	a.username = msg.Username
	a.prList.SetUsername(msg.Username)
	a.inbox.SetUsername(msg.Username)
	return a, nil
}

func (a *App) handlePRsLoaded(msg views.PRsLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		cmd := a.toasts.Add(
			fmt.Sprintf("Error loading PRs: %v", msg.Err),
			domain.ToastError, 5*time.Second,
		)
		// Don't switch to PR list while banner is still visible
		if a.view != core.ViewBanner {
			a.view = core.ViewPRList
		}
		a.prList.SetPRs(nil)
		return a, cmd
	}
	// Store PRs but don't switch view while banner is visible
	// The banner dismiss handler will transition to the PR list
	a.prList.SetPRs(msg.PRs)
	a.header.SetPRCount(a.prList.TotalPRs())
	a.header.SetFilter(a.prList.FilterLabel())
	if a.view != core.ViewBanner {
		a.view = core.ViewPRList
	}

	// Save fresh PRs to cache (fire and forget).
	var cmds []tea.Cmd
	if a.repo.Owner != "" && len(msg.PRs) > 0 {
		cmds = append(cmds, saveCacheCmd(a.repo, msg.PRs))
	}

	// Detect new PRs on auto-refresh.
	newCount := a.prList.TotalPRs()
	if a.prevPRCount > 0 && newCount > a.prevPRCount {
		diff := newCount - a.prevPRCount
		cmds = append(cmds, a.toasts.Add(
			fmt.Sprintf("%d new PR(s)", diff),
			domain.ToastInfo, 5*time.Second,
		))
	}
	a.prevPRCount = newCount

	// Start refresh timer (reset countdown).
	if a.refreshInterval > 0 && a.refreshCountdown <= 0 {
		cmds = append(cmds, a.startRefreshTimer())
	}
	return a, tea.Batch(cmds...)
}

func (a *App) handleLoadMorePRs(msg views.LoadMorePRsMsg) (tea.Model, tea.Cmd) {
	if a.listPRs == nil || a.repo.Owner == "" {
		return a, nil
	}
	// Mark that we're loading more and start spinner
	spinnerCmd := a.prList.SetLoadingMore(msg.Page)
	// Create opts with pagination
	opts := a.filterOpts
	opts.Page = msg.Page
	return a, tea.Batch(spinnerCmd, loadMorePRsCmd(a.listPRs, a.repo, opts, msg.Page))
}

func (a *App) handleMorePRsLoaded(msg views.MorePRsLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		cmd := a.toasts.Add(
			fmt.Sprintf("Error loading more PRs: %v", msg.Err),
			domain.ToastError, 5*time.Second,
		)
		// Reset loading state
		a.prList.AppendPRs(nil, false)
		return a, cmd
	}
	// Append the new PRs
	a.prList.AppendPRs(msg.PRs, msg.HasMore)
	// Update header count to show total loaded PRs
	a.header.SetPRCount(a.prList.TotalPRs())
	return a, nil
}

func (a *App) handlePRCountLoaded(msg views.PRCountLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		// Silently ignore count errors - not critical
		return a, nil
	}
	a.header.SetTotalCount(msg.Total)
	return a, nil
}

func (a *App) handleOpenPR(msg views.OpenPRMsg) (tea.Model, tea.Cmd) {
	a.view = core.ViewPRDetail
	spinCmd := a.prDetail.StartLoading(msg.Number)

	if a.getPRDetail != nil && a.repo.Owner != "" {
		return a, tea.Batch(spinCmd, loadPRDetailCmd(a.getPRDetail, a.repo, msg.Number))
	}
	return a, spinCmd
}

func (a *App) handlePRDetailLoaded(msg views.PRDetailLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		a.prDetail.StopLoading()
		cmd := a.toasts.Add(
			fmt.Sprintf("Error loading PR detail: %v", msg.Err),
			domain.ToastError, 5*time.Second,
		)
		return a, cmd
	}
	a.prDetail.SetDetail(msg.Detail)
	return a, nil
}

func (a *App) handleOpenExternalDiff(msg views.OpenExternalDiffMsg) (tea.Model, tea.Cmd) {
	tool := a.cfg.Diff.ExternalTool
	if tool == "" {
		cmd := a.toasts.Add(
			"No external diff tool configured. Set [diff] external_tool in config.",
			domain.ToastWarning, 5*time.Second,
		)
		return a, cmd
	}
	// Use tea.ExecProcess to suspend TUI and run the external tool.
	args := []string{"pr", "diff", fmt.Sprintf("%d", msg.Number)}
	c := exec.Command("gh", args...)
	c.Env = append(c.Environ(), fmt.Sprintf("GH_PAGER=%s", tool))
	return a, tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil {
			return externalDiffDoneMsg{Err: err}
		}
		return externalDiffDoneMsg{}
	})
}

type externalDiffDoneMsg struct {
	Err error
}

func (a *App) handleOpenDiff(msg views.OpenDiffMsg) (tea.Model, tea.Cmd) {
	a.view = core.ViewDiff
	a.diffView.SetPRNumber(msg.Number)
	if a.reader != nil && a.repo.Owner != "" {
		return a, loadDiffCmd(a.reader, a.repo, msg.Number)
	}
	return a, nil
}

func (a *App) handleSubmitReview(msg views.SubmitReviewMsg) (tea.Model, tea.Cmd) {
	if a.reviewPR != nil && a.repo.Owner != "" {
		return a, submitReviewCmd(a.reviewPR, a.repo, msg.Number, msg.Review)
	}
	return a, nil
}

func (a *App) handleReviewSubmitted(msg views.ReviewSubmittedMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		cmd := a.toasts.Add(
			fmt.Sprintf("Review failed: %v", msg.Err),
			domain.ToastError, 5*time.Second,
		)
		return a, cmd
	}
	cmd := a.toasts.Add("Review submitted", domain.ToastSuccess, 3*time.Second)
	a.view = core.ViewPRDetail
	return a, cmd
}

func (a *App) handleCheckoutConfirm(msg views.CheckoutPRMsg) (tea.Model, tea.Cmd) {
	branch := msg.Branch
	if branch == "" {
		branch = fmt.Sprintf("PR #%d", msg.Number)
	}
	a.prevView = a.view
	a.view = core.ViewConfirm
	a.confirmDialog.Show(
		"Checkout Branch",
		fmt.Sprintf("Check out branch \"%s\" for PR #%d?", branch, msg.Number),
		msg,
	)
	return a, nil
}

func (a *App) handleConfirmResult(msg views.ConfirmResultMsg) (tea.Model, tea.Cmd) {
	if !msg.Confirmed {
		a.view = a.prevView
		return a, nil
	}
	// Keep dialog open and show loading spinner while the action runs.
	checkoutMsg, ok := msg.Action.(views.CheckoutPRMsg)
	if ok && a.checkoutPR != nil && a.repo.Owner != "" {
		branch := checkoutMsg.Branch
		if branch == "" {
			branch = fmt.Sprintf("PR #%d", checkoutMsg.Number)
		}
		spinnerCmd := a.confirmDialog.ShowLoading(
			"Checkout Branch",
			fmt.Sprintf("Checking out \"%s\" for PR #%d...", branch, checkoutMsg.Number),
		)
		return a, tea.Batch(spinnerCmd, checkoutPRCmd(a.checkoutPR, a.repo, checkoutMsg.Number))
	}
	a.view = a.prevView
	return a, nil
}

func (a *App) handleCheckoutDone(msg views.CheckoutDoneMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		// Show error in the dialog if it's still open, otherwise toast.
		if a.view == core.ViewConfirm {
			a.confirmDialog.ShowResult("Checkout Failed", msg.Err.Error(), false)
			return a, nil
		}
		cmd := a.toasts.Add(
			fmt.Sprintf("Checkout failed: %v", msg.Err),
			domain.ToastError, 5*time.Second,
		)
		return a, cmd
	}
	a.prList.SetCurrentBranch(msg.Branch)
	// Show success in the dialog if it's still open, otherwise toast.
	if a.view == core.ViewConfirm {
		a.confirmDialog.ShowResult("Checkout Complete", fmt.Sprintf("Checked out branch: %s", msg.Branch), true)
		return a, nil
	}
	cmd := a.toasts.Add(
		fmt.Sprintf("Checked out branch: %s", msg.Branch),
		domain.ToastSuccess, 3*time.Second,
	)
	return a, cmd
}

func (a *App) handleSwitchRepo(msg views.SwitchRepoMsg) (tea.Model, tea.Cmd) {
	a.repo = msg.Repo
	a.header.SetRepo(a.repo)
	a.header.SetTotalCount(0) // Reset total count for new repo
	a.repoSwitcher.SetCurrentRepo(a.repo)
	a.view = core.ViewPRList

	if a.listPRs != nil {
		state := a.filterOpts.State
		if state == "" {
			state = domain.PRStateOpen
		}
		return a, tea.Batch(
			loadPRsCmd(a.listPRs, a.repo, a.filterOpts),
			loadPRCountCmd(a.reader, a.repo, state),
		)
	}
	return a, nil
}

func (a *App) handleReposDiscovered(msg views.ReposDiscoveredMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		cmd := a.toasts.Add(
			fmt.Sprintf("Could not list repos: %v", msg.Err),
			domain.ToastWarning, 5*time.Second,
		)
		return a, cmd
	}
	a.repoSwitcher.MergeDiscovered(msg.Repos)
	return a, nil
}

func (a *App) handleToggleFavorite(msg views.ToggleFavoriteMsg) (tea.Model, tea.Cmd) {
	// Collect current favorite repos from the switcher.
	favs := a.collectFavoriteStrings()
	if err := a.cfg.UpdateFavorites(favs); err != nil {
		cmd := a.toasts.Add(
			fmt.Sprintf("Failed to save favorites: %v", err),
			domain.ToastError, 5*time.Second,
		)
		return a, cmd
	}
	action := "added to"
	if !msg.Favorite {
		action = "removed from"
	}
	cmd := a.toasts.Add(
		fmt.Sprintf("%s %s favorites", msg.Repo.String(), action),
		domain.ToastSuccess, 3*time.Second,
	)
	return a, cmd
}

func (a *App) openInbox() (tea.Model, tea.Cmd) {
	a.prevView = a.view
	a.view = core.ViewInbox
	a.inbox.SetUsername(a.username)

	// Collect repos from favorites.
	var repos []domain.RepoRef
	for _, entry := range a.repoSwitcher.Favorites() {
		repos = append(repos, entry.Repo)
	}
	if len(repos) == 0 && a.repo.Owner != "" {
		repos = []domain.RepoRef{a.repo}
	}

	if a.getInboxPRs != nil && len(repos) > 0 {
		return a, loadInboxCmd(a.getInboxPRs, repos)
	}
	return a, nil
}

func (a *App) handleOpenInboxPR(msg views.OpenInboxPRMsg) (tea.Model, tea.Cmd) {
	// Switch repo context to the inbox PR's repo and open detail.
	a.repo = msg.Repo
	a.header.SetRepo(a.repo)
	a.view = core.ViewPRDetail
	spinCmd := a.prDetail.StartLoading(msg.Number)

	if a.getPRDetail != nil {
		return a, tea.Batch(spinCmd, loadPRDetailCmd(a.getPRDetail, a.repo, msg.Number))
	}
	return a, spinCmd
}

func (a *App) handleRepoValidated(msg views.RepoValidatedMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		cmd := a.toasts.Add(
			fmt.Sprintf("Repository not found: %s", msg.Repo.String()),
			domain.ToastError, 5*time.Second,
		)
		return a, cmd
	}
	// Valid repo — switch to it and add to favorites.
	return a.handleSwitchRepo(views.SwitchRepoMsg{Repo: msg.Repo})
}

func (a *App) ensureCWDRepoInFavorites() {
	if a.repo.Owner == "" {
		return
	}
	// Check if already in favorites.
	for _, f := range a.repoSwitcher.Favorites() {
		if f.Repo == a.repo {
			return
		}
	}
	// Prepend CWD repo.
	entry := views.RepoEntry{
		Repo:      a.repo,
		Favorite:  true,
		Section:   views.SectionFavorite,
		Current:   true,
		OpenCount: -1,
	}
	current := a.repoSwitcher.Favorites()
	all := append([]views.RepoEntry{entry}, current...)
	a.repoSwitcher.SetRepos(all)
}

func (a *App) collectFavoriteStrings() []string {
	// Walk the repo switcher favorites to build the config list.
	// Access via the SetRepos/MergeDiscovered approach — we need the switcher state.
	// Since we just toggled, we read from config and update based on the msg.
	// Simpler: read the favorites from the switcher model.
	var favs []string
	for _, entry := range a.repoSwitcher.Favorites() {
		favs = append(favs, entry.Repo.String())
	}
	return favs
}

// resolveThreadDoneMsg is sent when a resolve/unresolve thread operation completes.
type resolveThreadDoneMsg struct {
	ThreadID string
	Err      error
}

func (a *App) handleResolveThread(msg views.ResolveThreadMsg) (tea.Model, tea.Cmd) {
	if a.resolveThread != nil && a.repo.Owner != "" {
		return a, resolveThreadCmd(a.resolveThread, a.repo, msg.ThreadID)
	}
	return a, nil
}

func (a *App) handleResolveThreadDone(msg resolveThreadDoneMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		cmd := a.toasts.Add(
			fmt.Sprintf("Resolve failed: %v", msg.Err),
			domain.ToastError, 5*time.Second,
		)
		return a, cmd
	}
	cmd := a.toasts.Add("Thread resolved", domain.ToastSuccess, 3*time.Second)
	// Refresh PR detail to show updated resolved status
	if a.getPRDetail != nil && a.prDetail.GetPRNumber() > 0 {
		return a, tea.Batch(cmd, loadPRDetailCmd(a.getPRDetail, a.repo, a.prDetail.GetPRNumber()))
	}
	return a, cmd
}

func (a *App) dispatchKeyToView(msg tea.KeyMsg) tea.Cmd {
	switch a.view {
	case core.ViewPRList:
		return a.prList.Update(msg)
	case core.ViewPRDetail:
		return a.prDetail.Update(msg)
	case core.ViewDiff:
		return a.diffView.Update(msg)
	case core.ViewReview:
		return a.reviewForm.Update(msg)
	case core.ViewRepoSwitch:
		return a.repoSwitcher.Update(msg)
	case core.ViewHelp:
		return a.helpOverlay.Update(msg)
	case core.ViewInbox:
		return a.inbox.Update(msg)
	case core.ViewFilter:
		return a.filterPanel.Update(msg)
	case core.ViewConfirm:
		return a.confirmDialog.Update(msg)
	}
	return nil
}

func (a *App) updateActiveView(msg tea.Msg) tea.Cmd {
	switch a.view {
	case core.ViewPRList:
		return a.prList.Update(msg)
	case core.ViewPRDetail:
		return a.prDetail.Update(msg)
	case core.ViewDiff:
		return a.diffView.Update(msg)
	case core.ViewReview:
		return a.reviewForm.Update(msg)
	case core.ViewRepoSwitch:
		return a.repoSwitcher.Update(msg)
	case core.ViewHelp:
		return a.helpOverlay.Update(msg)
	case core.ViewInbox:
		return a.inbox.Update(msg)
	case core.ViewFilter:
		return a.filterPanel.Update(msg)
	case core.ViewConfirm:
		return a.confirmDialog.Update(msg)
	}
	return nil
}

// rebuildStyles propagates the updated theme to all view models and components.
func (a *App) rebuildStyles() {
	s := a.styles
	k := a.keys

	a.prList = views.NewPRListModel(s, k)
	a.prDetail = views.NewPRDetailModel(s, k)
	a.diffView = views.NewDiffViewModel(s, k)
	a.reviewForm = views.NewReviewModel(s, k)
	a.repoSwitcher = views.NewRepoSwitcherModel(s, k)
	a.helpOverlay = views.NewHelpModel(s)
	a.inbox = views.NewInboxModel(s, k)
	a.tutorial = views.NewTutorialModel(s)
	a.filterPanel = views.NewFilterModel(s, k)
	a.confirmDialog = views.NewConfirmModel(s)

	a.banner = components.NewBanner(s, a.version)
	a.header = components.NewHeader(s)
	a.status = components.NewStatusBar(s)
	a.toasts = components.NewToastManager(s)

	// Re-set repo on header.
	a.header.SetRepo(a.repo)

	// Re-set sizes.
	ch := a.contentHeight()
	a.banner.SetSize(a.width, a.height)
	a.prList.SetSize(a.width, ch)
	a.prDetail.SetSize(a.width, ch)
	a.diffView.SetSize(a.width, ch)
	a.reviewForm.SetSize(a.width, ch)
	a.repoSwitcher.SetSize(a.width, ch)
	a.helpOverlay.SetSize(a.width, ch)
	a.inbox.SetSize(a.width, ch)
	a.tutorial.SetSize(a.width, ch)
	a.filterPanel.SetSize(a.width, ch)
	a.confirmDialog.SetSize(a.width, ch)
	a.header.SetWidth(a.width)
	a.status.SetWidth(a.width)
	a.toasts.SetWidth(a.width)
}

func (a *App) contentHeight() int {
	return max(1, a.height-2) // header + status bar
}

// initRepoSwitcherFavorites populates the repo switcher from config favorites.
func (a *App) initRepoSwitcherFavorites() {
	var entries []views.RepoEntry
	for _, fav := range a.cfg.Repos.Favorites {
		parts := strings.SplitN(fav, "/", 2)
		if len(parts) == 2 {
			entries = append(entries, views.RepoEntry{
				Repo:      domain.RepoRef{Owner: parts[0], Name: parts[1]},
				Favorite:  true,
				Section:   views.SectionFavorite,
				OpenCount: -1,
			})
		}
	}
	a.repoSwitcher.SetRepos(entries)
}

func (a *App) View() string {
	if !a.ready {
		return ""
	}

	// Banner supersedes everything when visible.
	if a.view == core.ViewBanner && a.banner.Visible() {
		return a.banner.View()
	}

	// Tutorial overlay supersedes everything.
	if a.tutorial.Visible() {
		return a.tutorial.View()
	}

	// Header.
	headerView := a.header.View()

	// Content area.
	contentHeight := a.contentHeight()
	content := a.renderContent(contentHeight)

	// Status bar — context-specific hints.
	switch {
	case a.view == core.ViewConfirm:
		a.status.SetHints([]string{a.confirmDialog.ConfirmStateHint()})
	case a.view == core.ViewPRList && a.prList.IsSelectionMode():
		n := a.prList.SelectionCount()
		a.status.SetHints([]string{fmt.Sprintf("%d selected  Space toggle  a all  y copy  o open  v exit  Esc cancel", n)})
	default:
		a.status.SetHints([]string{views.StatusHints(a.view, a.width)})
	}
	statusView := a.status.View()

	// Stack vertically: header, content, status bar.
	// If toasts are active, render them between header and content.
	if a.toasts.HasToasts() {
		toastView := a.toasts.View()
		return lipgloss.JoinVertical(lipgloss.Left, headerView, toastView, content, statusView)
	}

	return lipgloss.JoinVertical(lipgloss.Left, headerView, content, statusView)
}

func (a *App) renderContent(height int) string {
	switch a.view {
	case core.ViewLoading:
		spinnerFrames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		frame := spinnerFrames[a.loadingFrame%len(spinnerFrames)]
		spinner := lipgloss.NewStyle().Foreground(a.theme.Primary).Render(frame)
		text := lipgloss.NewStyle().Foreground(a.theme.Muted).Render(" Loading...")
		content := spinner + text
		return lipgloss.Place(a.width, height, lipgloss.Center, lipgloss.Center, content)

	case core.ViewPRList:
		return a.prList.View()

	case core.ViewPRDetail:
		return a.prDetail.View()

	case core.ViewDiff:
		return a.diffView.View()

	case core.ViewReview:
		return a.reviewForm.View()

	case core.ViewHelp:
		return a.helpOverlay.View()

	case core.ViewRepoSwitch:
		return a.repoSwitcher.View()

	case core.ViewInbox:
		return a.inbox.View()
	case core.ViewFilter:
		return a.filterPanel.View()

	case core.ViewConfirm:
		return a.confirmDialog.View()

	default:
		return ""
	}
}

func (a *App) viewName() string {
	switch a.view {
	case core.ViewLoading:
		return "Loading"
	case core.ViewPRList:
		return "PR List"
	case core.ViewPRDetail:
		return "PR Detail"
	case core.ViewDiff:
		return "Diff"
	case core.ViewReview:
		return "Review"
	case core.ViewHelp:
		return "Help"
	case core.ViewRepoSwitch:
		return "Repo Switch"
	case core.ViewInbox:
		return "Inbox"
	case core.ViewFilter:
		return "Filter"
	case core.ViewConfirm:
		return "Confirm"
	default:
		return ""
	}
}
