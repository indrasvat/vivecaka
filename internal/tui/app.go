package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/indrasvat/vivecaka/internal/cache"
	"github.com/indrasvat/vivecaka/internal/config"
	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/indrasvat/vivecaka/internal/repolocator"
	"github.com/indrasvat/vivecaka/internal/reviewprogress"
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

// WithRepoManager sets the RepoManager adapter for smart checkout.
func WithRepoManager(rm domain.RepoManager) Option {
	return func(a *App) { a.repoManager = rm }
}

// WithRepo sets the initial repo (skips auto-detection).
func WithRepo(r domain.RepoRef) Option {
	return func(a *App) {
		a.repo = r
		a.repoExplicit = true
	}
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
	repoExplicit bool
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
	reader      domain.PRReader
	reviewer    domain.PRReviewer
	writer      domain.PRWriter
	repoManager domain.RepoManager

	// Smart checkout
	cwdRepo       domain.RepoRef // CWD repo identity (detected on startup)
	cwdPath       string         // CWD path (captured on startup)
	repoLocator   *repolocator.Locator
	smartCheckout *usecase.SmartCheckout

	// Use cases
	listPRs          *usecase.ListPRs
	getPRDetail      *usecase.GetPRDetail
	getReviewContext *usecase.GetReviewContext
	reviewPR         *usecase.ReviewPR
	checkoutPR       *usecase.CheckoutPR
	addComment       *usecase.AddComment
	resolveThread    *usecase.ResolveThread
	getInboxPRs      *usecase.GetInboxPRs

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
	confirmDialog  views.ConfirmModel
	checkoutDialog views.CheckoutDialogModel

	// Filters
	filterOpts domain.ListOpts

	// Per-repo state persistence
	repoState            cache.RepoState
	currentReviewContext *reviewprogress.Context
	currentReviewDiff    *domain.Diff
	currentReviewPR      int

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
		prList:         views.NewPRListModel(styles, keys),
		prDetail:       views.NewPRDetailModel(styles, keys),
		diffView:       views.NewDiffViewModel(styles, keys),
		reviewForm:     views.NewReviewModel(styles, keys),
		repoSwitcher:   views.NewRepoSwitcherModel(styles, keys),
		helpOverlay:    views.NewHelpModel(styles),
		inbox:          views.NewInboxModel(styles, keys),
		tutorial:       views.NewTutorialModel(styles),
		filterPanel:    views.NewFilterModel(styles, keys),
		confirmDialog:  views.NewConfirmModel(styles),
		checkoutDialog: views.NewCheckoutDialogModel(styles, keys),

		// Infrastructure
		repoLocator: repolocator.New(),

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
		a.getReviewContext = usecase.NewGetReviewContext(a.reader)
		a.getInboxPRs = usecase.NewGetInboxPRs(a.reader)
	}
	if a.reviewer != nil {
		a.reviewPR = usecase.NewReviewPR(a.reviewer)
		a.addComment = usecase.NewAddComment(a.reviewer)
		a.resolveThread = usecase.NewResolveThread(a.reviewer)
	}
	if a.writer != nil {
		a.checkoutPR = usecase.NewCheckoutPR(a.writer)
	}
	if a.repoManager != nil {
		a.smartCheckout = usecase.NewSmartCheckout(a.repoManager, a.repoLocator)
	}

	// Capture CWD path on startup.
	a.cwdPath, _ = os.Getwd()

	// Initialize repo switcher with favorites from config.
	a.initRepoSwitcherFavorites()

	return a
}

func (a *App) Init() tea.Cmd {
	cmds := []tea.Cmd{
		detectUserCmd(),
		a.banner.StartAutoDismiss(2 * time.Second), // Show banner for 2 seconds
	}
	if !a.repoExplicit {
		cmds = append(cmds, detectBranchCmd())
	}
	// Show tutorial on first launch.
	if views.IsFirstLaunch() {
		a.tutorial.Show()
	}
	// If repo was provided via WithRepo, skip detection and load PRs directly.
	if a.repo.Owner != "" {
		cmds = append(cmds, a.startRepoLoad(a.repo, false)...)
		return tea.Batch(cmds...)
	}
	// Otherwise, detect repo from git remote.
	cmds = append(cmds, detectRepoCmd())
	return tea.Batch(cmds...)
}

func (a *App) startRepoLoad(repo domain.RepoRef, includeCache bool) []tea.Cmd {
	a.repo = repo
	a.header.SetRepo(a.repo)
	a.header.SetTotalCount(0)
	a.loadRepoState()
	a.repoSwitcher.SetCurrentRepo(a.repo)

	if a.listPRs == nil {
		return []tea.Cmd{func() tea.Msg { return viewReadyMsg{} }}
	}

	state := a.filterOpts.State
	if state == "" {
		state = domain.PRStateOpen
	}

	cmds := make([]tea.Cmd, 0, 3)
	if includeCache {
		cmds = append(cmds, loadCachedPRsCmd(a.repo))
	}
	cmds = append(cmds,
		loadPRsCmd(a.listPRs, a.repo, a.filterOpts),
		loadPRCountCmd(a.reader, a.repo, state),
	)
	return cmds
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

	switch typedMsg := msg.(type) {
	case tea.WindowSizeMsg:
		return a.handleWindowSize(typedMsg)

	case tea.KeyMsg:
		return a.handleKey(typedMsg)

	case viewReadyMsg:
		a.view = core.ViewPRList
		return a, nil
	}

	if handled, cmd := a.handleAppMessage(msg); handled {
		return a, cmd
	}

	// Dispatch to active view model for unhandled messages.
	cmd := a.updateActiveView(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	return a, tea.Batch(cmds...)
}

func (a *App) handleAppMessage(msg tea.Msg) (bool, tea.Cmd) {
	switch typedMsg := msg.(type) {
	case views.RepoDetectedMsg:
		_, cmd := a.handleRepoDetected(typedMsg)
		return true, cmd
	case views.UserDetectedMsg:
		_, cmd := a.handleUserDetected(typedMsg)
		return true, cmd
	case cachedPRsLoadedMsg:
		if len(typedMsg.PRs) > 0 && a.prList.IsLoading() {
			a.prList.SetPRs(typedMsg.PRs)
			a.header.SetPRCount(a.prList.TotalPRs())
		}
		return true, nil
	case views.BranchDetectedMsg:
		if typedMsg.Err == nil && typedMsg.Branch != "" {
			a.prList.SetCurrentBranch(typedMsg.Branch)
			a.header.SetBranch(typedMsg.Branch)
		}
		return true, nil
	case views.PRsLoadedMsg:
		_, cmd := a.handlePRsLoaded(typedMsg)
		return true, cmd
	case views.LoadMorePRsMsg:
		_, cmd := a.handleLoadMorePRs(typedMsg)
		return true, cmd
	case views.MorePRsLoadedMsg:
		_, cmd := a.handleMorePRsLoaded(typedMsg)
		return true, cmd
	case views.PRCountLoadedMsg:
		a.handlePRCountLoaded(typedMsg)
		return true, nil
	case views.OpenPRMsg:
		_, cmd := a.handleOpenPR(typedMsg)
		return true, cmd
	case views.PRDetailLoadedMsg:
		_, cmd := a.handlePRDetailLoaded(typedMsg)
		return true, cmd
	case views.ReviewContextLoadedMsg:
		a.handleReviewContextLoaded(typedMsg)
		return true, nil
	case views.OpenDiffMsg:
		_, cmd := a.handleOpenDiff(typedMsg)
		return true, cmd
	case views.OpenExternalDiffMsg:
		_, cmd := a.handleOpenExternalDiff(typedMsg)
		return true, cmd
	case views.OpenFilterMsg:
		a.prevView = a.view
		a.view = core.ViewFilter
		a.filterPanel.SetOpts(a.filterOpts)
		return true, nil
	case views.ApplyFilterMsg:
		a.filterOpts = typedMsg.Opts
		a.prList.SetFilter(typedMsg.Opts)
		a.header.SetFilter(a.prList.FilterLabel())
		a.repoState.LastFilter = typedMsg.Opts
		a.saveRepoState()
		a.view = a.prevView
		if a.listPRs != nil && a.repo.Owner != "" {
			return true, loadPRsCmd(a.listPRs, a.repo, typedMsg.Opts)
		}
		return true, nil
	case views.CloseFilterMsg:
		a.view = a.prevView
		return true, nil
	case views.DiffLoadedMsg:
		return true, a.handleDiffLoaded(typedMsg)
	case views.AddInlineCommentMsg:
		_, cmd := a.handleAddInlineComment(typedMsg)
		return true, cmd
	case views.InlineCommentAddedMsg:
		_, cmd := a.handleInlineCommentAdded(typedMsg)
		return true, cmd
	case views.StartReviewMsg:
		a.prevView = a.view
		a.view = core.ViewReview
		a.reviewForm.SetPRNumber(typedMsg.Number)
		return true, a.reviewForm.Init()
	case views.SubmitReviewMsg:
		_, cmd := a.handleSubmitReview(typedMsg)
		return true, cmd
	case views.ReviewSubmittedMsg:
		_, cmd := a.handleReviewSubmitted(typedMsg)
		return true, cmd
	case views.CycleReviewScopeMsg:
		a.handleCycleReviewScope()
		return true, nil
	case views.JumpNextReviewTargetMsg:
		a.handleJumpNextReviewTarget(typedMsg)
		return true, nil
	case views.ToggleViewedFileMsg:
		a.handleToggleViewedFile(typedMsg)
		return true, nil
	case views.CloseReviewMsg:
		a.view = core.ViewPRDetail
		return true, nil
	case views.CheckoutPRMsg:
		_, cmd := a.handleSmartCheckout(typedMsg)
		return true, cmd
	case views.ConfirmResultMsg:
		_, cmd := a.handleConfirmResult(typedMsg)
		return true, cmd
	case views.CloseConfirmMsg:
		a.view = a.prevView
		return true, nil
	case views.CheckoutDoneMsg:
		_, cmd := a.handleCheckoutDone(typedMsg)
		return true, cmd
	case views.CheckoutStrategyChosenMsg:
		_, cmd := a.handleCheckoutStrategyChosen(typedMsg)
		return true, cmd
	case views.CloneDoneMsg:
		_, cmd := a.handleCloneDone(typedMsg)
		return true, cmd
	case views.SmartCheckoutDoneMsg:
		a.handleSmartCheckoutDone(typedMsg)
		return true, nil
	case views.CheckoutDialogCloseMsg:
		a.view = a.prevView
		return true, nil
	case views.CopyCdCommandMsg:
		return true, a.handleCopyCdCommand(typedMsg)
	case views.CopyURLMsg:
		return true, a.handleCopyURL(typedMsg)
	case views.OpenBrowserMsg:
		return true, a.handleOpenBrowser(typedMsg)
	case views.BatchCopyURLsMsg:
		return true, a.handleBatchCopyURLs(typedMsg)
	case views.BatchOpenBrowserMsg:
		return true, a.handleBatchOpenBrowser(typedMsg)
	case views.PRListFilterMsg:
		a.header.SetFilter(typedMsg.Label)
		return true, nil
	case views.SwitchRepoMsg:
		_, cmd := a.handleSwitchRepo(typedMsg)
		return true, cmd
	case views.CloseRepoSwitcherMsg:
		a.view = a.prevView
		return true, nil
	case views.ReposDiscoveredMsg:
		_, cmd := a.handleReposDiscovered(typedMsg)
		return true, cmd
	case views.ToggleFavoriteMsg:
		_, cmd := a.handleToggleFavorite(typedMsg)
		return true, cmd
	case views.ValidateRepoRequestMsg:
		return true, validateRepoCmd(typedMsg.Repo)
	case views.RepoValidatedMsg:
		_, cmd := a.handleRepoValidated(typedMsg)
		return true, cmd
	case views.CloseHelpMsg:
		a.view = a.prevView
		return true, nil
	case views.OpenInboxPRMsg:
		_, cmd := a.handleOpenInboxPR(typedMsg)
		return true, cmd
	case views.CloseInboxMsg:
		a.view = core.ViewPRList
		return true, nil
	case views.ResolveThreadMsg:
		_, cmd := a.handleResolveThread(typedMsg)
		return true, cmd
	case views.UnresolveThreadMsg:
		return true, a.toasts.Add("Unresolve not implemented yet", domain.ToastInfo, 3*time.Second)
	case views.ReplyToThreadMsg:
		return true, a.toasts.Add("Reply not implemented yet", domain.ToastInfo, 3*time.Second)
	case resolveThreadDoneMsg:
		_, cmd := a.handleResolveThreadDone(typedMsg)
		return true, cmd
	case views.TutorialDoneMsg:
		if err := views.MarkTutorialDone(); err != nil {
			return true, a.toasts.Add(
				fmt.Sprintf("Failed to persist tutorial state: %v", err),
				domain.ToastError, 5*time.Second,
			)
		}
		return true, nil
	case components.DismissToastMsg:
		a.toasts.Update(typedMsg)
		return true, nil
	case components.BannerGlyphTickMsg:
		return true, a.banner.Update(typedMsg)
	case components.BannerDismissMsg:
		return true, a.handleBannerDismiss(typedMsg)
	case loadingTickMsg:
		if a.view == core.ViewLoading {
			a.loadingFrame++
			return true, a.loadingTick()
		}
		return true, nil
	case refreshTickMsg:
		_, cmd := a.handleRefreshTick()
		return true, cmd
	case externalDiffDoneMsg:
		if typedMsg.Err != nil {
			return true, a.toasts.Add(
				fmt.Sprintf("External diff tool error: %v", typedMsg.Err),
				domain.ToastError, 5*time.Second,
			)
		}
		return true, nil
	default:
		return false, nil
	}
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
	a.checkoutDialog.SetSize(a.width, contentHeight)

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

	// Smart checkout dialog intercepts all keys when visible.
	if a.view == core.ViewSmartCheckout {
		cmd := a.checkoutDialog.Update(msg)
		return a, cmd
	}

	// Tutorial intercepts all keys when visible.
	if a.tutorial.Visible() {
		cmd := a.tutorial.Update(msg)
		return a, cmd
	}

	// Views with text input intercept all keys so global shortcuts
	// (q, ?, T, R) don't fire while the user is typing.
	// Only Ctrl+C is allowed to quit globally from these views.
	switch a.view {
	case core.ViewRepoSwitch:
		if msg.Type == tea.KeyCtrlC {
			return a, tea.Quit
		}
		cmd := a.repoSwitcher.Update(msg)
		return a, cmd
	case core.ViewFilter:
		if msg.Type == tea.KeyCtrlC {
			return a, tea.Quit
		}
		cmd := a.filterPanel.Update(msg)
		return a, cmd
	case core.ViewReview:
		if msg.Type == tea.KeyCtrlC {
			return a, tea.Quit
		}
		cmd := a.reviewForm.Update(msg)
		return a, cmd
	}

	// Global keys always active.
	switch {
	case key.Matches(msg, a.keys.Quit):
		a.finalizeCurrentPRVisit()
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
		a.finalizeCurrentPRVisit()
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
	a.cwdRepo = msg.Repo // Record CWD repo identity for smart checkout.

	// Auto-learn: register CWD repo in known-repos (smart checkout mechanism 1).
	if a.repoLocator != nil && a.cwdPath != "" {
		_ = a.repoLocator.Register(msg.Repo, a.cwdPath, "detected")
	}

	// Prepend CWD repo to favorites if not already there.
	a.ensureCWDRepoInFavorites()

	if a.listPRs != nil {
		return a, tea.Batch(a.startRepoLoad(a.repo, true)...)
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
		// Only transition to PR list from loading state — don't dismiss modals/dialogs.
		if a.view == core.ViewLoading {
			a.view = core.ViewPRList
		}
		a.prList.SetPRs(nil)
		return a, cmd
	}
	// Store PRs but don't switch view while the user is in a modal or detail view.
	// Only transition from loading/banner states.
	a.prList.SetPRs(msg.PRs)
	a.header.SetPRCount(a.prList.TotalPRs())
	a.header.SetFilter(a.prList.FilterLabel())
	if a.view == core.ViewLoading || a.view == core.ViewPRList {
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

func (a *App) handlePRCountLoaded(msg views.PRCountLoadedMsg) tea.Model {
	if msg.Err != nil {
		// Silently ignore count errors - not critical
		return a
	}
	a.header.SetTotalCount(msg.Total)
	return a
}

func (a *App) handleDiffLoaded(msg views.DiffLoadedMsg) tea.Cmd {
	if msg.Number != 0 && msg.Number != a.currentReviewPR {
		return nil
	}

	var cmds []tea.Cmd
	if msg.Err == nil && msg.Diff != nil {
		a.currentReviewDiff = msg.Diff
	}
	cmd := a.diffView.Update(msg)
	if msg.Err == nil && msg.Diff != nil && a.currentReviewContext != nil {
		if next := a.nextReviewTargetPath(""); next != "" {
			a.diffView.JumpToFile(next)
		}
	}
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	return tea.Batch(cmds...)
}

func (a *App) handleCopyCdCommand(msg views.CopyCdCommandMsg) tea.Cmd {
	a.view = a.prevView
	if err := copyToClipboard("cd " + msg.Path); err != nil {
		return a.toasts.Add(
			fmt.Sprintf("Copy failed: %v", err),
			domain.ToastError, 5*time.Second,
		)
	}
	return a.toasts.Add("Copied cd command", domain.ToastSuccess, 3*time.Second)
}

func (a *App) handleCopyURL(msg views.CopyURLMsg) tea.Cmd {
	if err := copyToClipboard(msg.URL); err != nil {
		return a.toasts.Add(
			fmt.Sprintf("Copy failed: %v", err),
			domain.ToastError, 5*time.Second,
		)
	}
	return a.toasts.Add("Copied PR URL", domain.ToastSuccess, 3*time.Second)
}

func (a *App) handleOpenBrowser(msg views.OpenBrowserMsg) tea.Cmd {
	if err := openBrowser(msg.URL); err != nil {
		return a.toasts.Add(
			fmt.Sprintf("Open browser failed: %v", err),
			domain.ToastError, 5*time.Second,
		)
	}
	return nil
}

func (a *App) handleBatchCopyURLs(msg views.BatchCopyURLsMsg) tea.Cmd {
	combined := strings.Join(msg.URLs, "\n")
	if err := copyToClipboard(combined); err != nil {
		return a.toasts.Add(
			fmt.Sprintf("Copy failed: %v", err),
			domain.ToastError, 5*time.Second,
		)
	}
	return a.toasts.Add(
		fmt.Sprintf("Copied %d PR URLs", len(msg.URLs)),
		domain.ToastSuccess, 3*time.Second,
	)
}

func (a *App) handleBatchOpenBrowser(msg views.BatchOpenBrowserMsg) tea.Cmd {
	for _, url := range msg.URLs {
		if err := openBrowser(url); err != nil {
			return a.toasts.Add(
				fmt.Sprintf("Open browser failed: %v", err),
				domain.ToastError, 5*time.Second,
			)
		}
	}
	return a.toasts.Add(
		fmt.Sprintf("Opened %d PRs in browser", len(msg.URLs)),
		domain.ToastSuccess, 3*time.Second,
	)
}

func (a *App) handleBannerDismiss(msg components.BannerDismissMsg) tea.Cmd {
	a.banner.Update(msg)
	if a.view == core.ViewBanner {
		if a.prList.HasPRs() {
			a.view = core.ViewPRList
		} else {
			a.view = core.ViewLoading
		}
	}
	return tea.Batch(tea.ClearScreen, a.loadingTick())
}

func (a *App) handleOpenPR(msg views.OpenPRMsg) (tea.Model, tea.Cmd) {
	a.finalizeCurrentPRVisit()
	a.view = core.ViewPRDetail
	// Mark PR as viewed for unread tracking.
	a.repoState.MarkPRViewed(msg.Number)
	a.saveRepoState()
	a.currentReviewContext = nil
	a.currentReviewDiff = nil
	a.currentReviewPR = msg.Number
	a.prDetail.SetReviewContext(nil)
	a.diffView.SetReviewContext(nil)
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
	if a.getReviewContext != nil && a.repo.Owner != "" {
		state := a.repoState.ReviewState(msg.Detail.Number)
		return a, loadReviewContextCmd(a.getReviewContext, a.repo, msg.Detail.Number, msg.Detail, state)
	}
	return a, nil
}

func (a *App) handleReviewContextLoaded(msg views.ReviewContextLoadedMsg) tea.Model {
	if msg.Err != nil || msg.Number != a.currentReviewPR {
		return a
	}
	a.currentReviewContext = msg.Context
	a.currentReviewDiff = msg.Diff
	a.prDetail.SetReviewContext(msg.Context)
	a.diffView.SetReviewContext(msg.Context)
	if a.view == core.ViewDiff && msg.Diff != nil {
		a.diffView.SetDiff(msg.Diff)
		if next := a.nextReviewTargetPath(a.diffView.CurrentFilePath()); next != "" && a.diffView.CurrentFilePath() == "" {
			a.diffView.JumpToFile(next)
		}
	}
	return a
}

// detectDiffTool probes git config and PATH for a diff tool.
// Returns empty string if nothing found.
func detectDiffTool() string {
	ctx := context.Background()
	// 1. git config diff.external
	if out, err := exec.CommandContext(ctx, "git", "config", "diff.external").Output(); err == nil {
		if t := strings.TrimSpace(string(out)); t != "" {
			return t
		}
	}
	// 2. git config diff.tool
	if out, err := exec.CommandContext(ctx, "git", "config", "diff.tool").Output(); err == nil {
		if t := strings.TrimSpace(string(out)); t != "" {
			return t
		}
	}
	// 3. Common diff tools in PATH.
	for _, t := range []string{"difft", "delta", "difftastic", "icdiff", "colordiff"} {
		if _, err := exec.LookPath(t); err == nil {
			return t
		}
	}
	return ""
}

func (a *App) handleOpenExternalDiff(msg views.OpenExternalDiffMsg) (tea.Model, tea.Cmd) {
	tool := a.cfg.Diff.ExternalTool
	if tool == "" {
		tool = detectDiffTool()
	}
	if tool == "" {
		cmd := a.toasts.Add(
			"No external diff tool found. Set [diff] external_tool in config.",
			domain.ToastWarning, 5*time.Second,
		)
		return a, cmd
	}

	// If the API diff failed (e.g. too large), fall back to local git diff.
	if msg.LoadErr != nil {
		branch := a.prDetail.GetBranch()
		if branch.Base != "" && branch.Head != "" {
			repoDir := a.findRepoDir()
			if repoDir == "" {
				cmd := a.toasts.Add(
					"No local repo found. Press Esc → c to checkout the branch first.",
					domain.ToastWarning, 5*time.Second,
				)
				return a, cmd
			}
			c := exec.Command("sh", "-c", //nolint:noctx
				`git fetch origin "$1" "$2" && GIT_EXTERNAL_DIFF="$3" git diff "origin/$1"..."origin/$2"`,
				"_", branch.Base, branch.Head, tool,
			)
			c.Dir = repoDir
			return a, tea.ExecProcess(c, func(err error) tea.Msg {
				return externalDiffDoneMsg{Err: err}
			})
		}
	}

	// Default: pipe gh pr diff through the tool as GH_PAGER.
	args := []string{"pr", "diff", fmt.Sprintf("%d", msg.Number)}
	c := exec.Command("gh", args...) //nolint:noctx // tea.ExecProcess requires raw *exec.Cmd, no context available
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
	a.diffView.SetHeadBranch(a.prDetail.GetBranch().Head)
	// Pass inline comments from the loaded PR detail to the diff view.
	a.diffView.SetComments(a.prDetail.GetInlineComments())
	a.diffView.SetReviewContext(a.currentReviewContext)
	if a.currentReviewPR == msg.Number && a.currentReviewDiff != nil {
		a.diffView.SetDiff(a.currentReviewDiff)
		if next := a.nextReviewTargetPath(""); next != "" {
			a.diffView.JumpToFile(next)
		}
		return a, nil
	}
	spinnerCmd := a.diffView.StartLoading()
	if a.reader != nil && a.repo.Owner != "" {
		return a, tea.Batch(spinnerCmd, loadDiffCmd(a.reader, a.repo, msg.Number))
	}
	return a, spinnerCmd
}

func (a *App) handleAddInlineComment(msg views.AddInlineCommentMsg) (tea.Model, tea.Cmd) {
	if a.addComment != nil && a.repo.Owner != "" {
		return a, addInlineCommentCmd(a.addComment, a.repo, msg.Number, msg.Input)
	}
	return a, nil
}

func (a *App) handleInlineCommentAdded(msg views.InlineCommentAddedMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		cmd := a.toasts.Add(
			fmt.Sprintf("Comment failed: %v", msg.Err),
			domain.ToastError, 5*time.Second,
		)
		return a, cmd
	}
	cmd := a.toasts.Add("Comment added", domain.ToastSuccess, 3*time.Second)
	// Refresh PR detail to show the new comment.
	if a.getPRDetail != nil && a.prDetail.GetPRNumber() > 0 {
		return a, tea.Batch(cmd, loadPRDetailCmd(a.getPRDetail, a.repo, a.prDetail.GetPRNumber()))
	}
	return a, cmd
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
	a.markCurrentPRReviewed()
	cmd := a.toasts.Add("Review submitted", domain.ToastSuccess, 3*time.Second)
	a.view = core.ViewPRDetail
	if a.getPRDetail != nil && a.prDetail.GetPRNumber() > 0 {
		return a, tea.Batch(cmd, loadPRDetailCmd(a.getPRDetail, a.repo, a.prDetail.GetPRNumber()))
	}
	return a, cmd
}

func (a *App) handleSmartCheckout(msg views.CheckoutPRMsg) (tea.Model, tea.Cmd) {
	// If no RepoManager, use legacy path (DR-2: only safe when repos match).
	if a.smartCheckout == nil {
		if reposMatchRef(a.repo, a.cwdRepo) {
			return a.handleCheckoutConfirm(msg)
		}
		// Unsafe: can't checkout in wrong repo without RepoManager.
		a.prevView = a.view
		a.view = core.ViewSmartCheckout
		a.checkoutDialog.ShowError(fmt.Errorf("cannot check out PRs from a different repo without smart checkout capability"))
		return a, nil
	}

	// Validate known-repos entry (I/O — DR-1).
	knownPath, knownValid := a.repoLocator.Validate(a.repo)

	ctx := usecase.CheckoutContext{
		BrowsingRepo: a.repo,
		CWDRepo:      a.cwdRepo,
		CWDPath:      a.cwdPath,
	}
	plan := a.smartCheckout.Plan(ctx, knownPath, knownValid)

	a.prevView = a.view
	switch plan.Strategy {
	case usecase.StrategyLocal:
		// CWD matches — show worktree choice dialog.
		a.view = core.ViewSmartCheckout
		a.checkoutDialog.ShowWorktreeChoice(a.repo, msg.Number, msg.Branch, plan)
		return a, nil

	case usecase.StrategyKnownPath:
		// Known path — show confirmation dialog.
		a.view = core.ViewSmartCheckout
		a.checkoutDialog.ShowKnownConfirm(a.repo, msg.Number, msg.Branch, plan)
		return a, nil

	case usecase.StrategyNeedsClone:
		// No local clone — show options dialog.
		a.view = core.ViewSmartCheckout
		a.checkoutDialog.ShowOptions(a.repo, msg.Number, msg.Branch, plan)
		return a, nil
	}
	return a, nil
}

func (a *App) handleCheckoutStrategyChosen(msg views.CheckoutStrategyChosenMsg) (tea.Model, tea.Cmd) {
	switch msg.Strategy {
	case "switch":
		// Direct branch switch in CWD (existing behavior).
		spinnerCmd := a.checkoutDialog.ShowCheckingOut(a.cwdPath)
		return a, tea.Batch(spinnerCmd,
			smartCheckoutCmd(a.smartCheckout, msg.Repo, msg.PRNumber, a.cwdPath, true))

	case "worktree":
		// Create worktree in CWD repo.
		spinnerCmd := a.checkoutDialog.ShowCheckingOut(a.cwdPath)
		return a, tea.Batch(spinnerCmd,
			worktreeCmd(a.smartCheckout, msg.Repo, msg.PRNumber, msg.Branch, a.cwdPath))

	case "known-path":
		// Checkout at known path.
		spinnerCmd := a.checkoutDialog.ShowCheckingOut(msg.Path)
		return a, tea.Batch(spinnerCmd,
			smartCheckoutCmd(a.smartCheckout, msg.Repo, msg.PRNumber, msg.Path, false))

	case "clone-cache", "clone-custom":
		// Clone then checkout.
		spinnerCmd := a.checkoutDialog.ShowCloning(msg.Path)
		return a, tea.Batch(spinnerCmd,
			cloneRepoCmd(a.smartCheckout, msg.Repo, msg.PRNumber, msg.Branch, msg.Path))

	case "browser":
		// Open in browser.
		url := fmt.Sprintf("https://github.com/%s/pull/%d", msg.Repo.String(), msg.PRNumber)
		a.view = a.prevView
		if err := openBrowser(url); err != nil {
			cmd := a.toasts.Add(
				fmt.Sprintf("Open browser failed: %v", err),
				domain.ToastError, 5*time.Second,
			)
			return a, cmd
		}
		return a, nil
	}
	return a, nil
}

func (a *App) handleCloneDone(msg views.CloneDoneMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		a.checkoutDialog.ShowError(msg.Err)
		return a, nil
	}
	// Clone succeeded — now checkout the PR.
	spinnerCmd := a.checkoutDialog.ShowCheckingOut(msg.Path)
	// Get the PR number and repo from the dialog state.
	return a, tea.Batch(spinnerCmd,
		smartCheckoutCmd(a.smartCheckout, a.repo, a.checkoutDialog.GetPRNumber(), msg.Path, false))
}

func (a *App) handleSmartCheckoutDone(msg views.SmartCheckoutDoneMsg) tea.Model {
	if msg.Err != nil {
		a.checkoutDialog.ShowError(msg.Err)
		return a
	}
	a.prList.SetCurrentBranch(msg.Branch)
	cwdCheckout := msg.Path == a.cwdPath
	a.checkoutDialog.ShowSuccess(msg.Branch, msg.Path, cwdCheckout)
	return a
}

func reposMatchRef(a, b domain.RepoRef) bool {
	return strings.EqualFold(a.Owner, b.Owner) && strings.EqualFold(a.Name, b.Name)
}

// findRepoDir returns a local directory for the currently browsed repo.
// It checks the CWD first, then the repo locator's known paths.
func (a *App) findRepoDir() string {
	if reposMatchRef(a.repo, a.cwdRepo) && a.cwdPath != "" {
		return a.cwdPath
	}
	if a.repoLocator != nil {
		if p, ok := a.repoLocator.Validate(a.repo); ok {
			return p
		}
	}
	return ""
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
	a.finalizeCurrentPRVisit()
	a.repo = msg.Repo
	a.header.SetRepo(a.repo)
	a.header.SetTotalCount(0) // Reset total count for new repo
	a.loadRepoState()
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
	a.finalizeCurrentPRVisit()
	a.repo = msg.Repo
	a.header.SetRepo(a.repo)
	a.loadRepoState()
	a.view = core.ViewPRDetail
	a.currentReviewContext = nil
	a.currentReviewDiff = nil
	a.currentReviewPR = msg.Number
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

func (a *App) handleCycleReviewScope() tea.Model {
	state := a.repoState.ReviewState(a.currentReviewPR)
	scope := reviewprogress.Scope(state.ActiveScope)
	if scope == "" && a.currentReviewContext != nil {
		scope = a.currentReviewContext.Scope
	}
	scope = scope.Cycle()
	state.ActiveScope = string(scope)
	a.repoState.SetReviewState(a.currentReviewPR, state)
	a.saveRepoState()
	a.rebuildReviewContext()
	return a
}

func (a *App) handleJumpNextReviewTarget(msg views.JumpNextReviewTargetMsg) tea.Model {
	path := a.nextReviewTargetPath(msg.CurrentPath)
	if path == "" {
		return a
	}
	switch a.view {
	case core.ViewDiff:
		a.diffView.JumpToFile(path)
	default:
		a.prDetail.JumpToFile(path)
	}
	return a
}

func (a *App) handleToggleViewedFile(msg views.ToggleViewedFileMsg) tea.Model {
	if msg.Path == "" || a.currentReviewContext == nil {
		return a
	}
	file, ok := a.currentReviewContext.FindFile(msg.Path)
	if !ok {
		return a
	}

	state := a.repoState.ReviewState(a.currentReviewPR)
	if state.ViewedFiles == nil {
		state.ViewedFiles = make(map[string]cache.FileReviewState)
	}
	if snap, ok := state.ViewedFiles[msg.Path]; ok && snap.PatchDigest == file.PatchDigest {
		delete(state.ViewedFiles, msg.Path)
	} else {
		state.ViewedFiles[msg.Path] = cache.FileReviewState{
			ViewedAt:      time.Now(),
			ViewedHeadSHA: a.currentReviewContext.HeadSHA,
			PatchDigest:   file.PatchDigest,
		}
	}
	a.repoState.SetReviewState(a.currentReviewPR, state)
	a.saveRepoState()
	a.rebuildReviewContext()
	return a
}

func (a *App) rebuildReviewContext() {
	if a.currentReviewContext == nil || a.prDetail.GetPRNumber() == 0 {
		return
	}
	detail := a.prDetail.GetDetail()
	if detail == nil {
		return
	}
	state := a.repoState.ReviewState(detail.Number)
	digests := a.currentReviewContext.CurrentDigests
	a.currentReviewContext = reviewprogress.Build(detail, digests, state, a.currentReviewContext.DegradedDigestSource)
	a.prDetail.SetReviewContext(a.currentReviewContext)
	a.diffView.SetReviewContext(a.currentReviewContext)
}

func (a *App) nextReviewTargetPath(current string) string {
	if a.currentReviewContext == nil {
		return ""
	}
	return a.currentReviewContext.NextActionableAfter(current)
}

func (a *App) finalizeCurrentPRVisit() {
	if a.currentReviewPR == 0 || a.currentReviewContext == nil {
		return
	}
	state := a.repoState.ReviewState(a.currentReviewPR)
	headSHA, files := reviewprogress.SnapshotFromContext(a.currentReviewContext, time.Now())
	state.LastVisitAt = time.Now()
	state.LastVisitHeadSHA = headSHA
	state.LastVisitFiles = files
	if state.ActiveScope == "" {
		state.ActiveScope = string(a.currentReviewContext.Scope)
	}
	a.repoState.SetReviewState(a.currentReviewPR, state)
	a.saveRepoState()
}

func (a *App) markCurrentPRReviewed() {
	if a.currentReviewPR == 0 || a.currentReviewContext == nil {
		return
	}
	state := a.repoState.ReviewState(a.currentReviewPR)
	now := time.Now()
	headSHA, files := reviewprogress.SnapshotFromContext(a.currentReviewContext, now)
	state.LastReviewAt = now
	state.LastReviewHeadSHA = headSHA
	state.LastReviewFiles = files
	state.LastVisitAt = now
	state.LastVisitHeadSHA = headSHA
	state.LastVisitFiles = files
	if state.ViewedFiles == nil {
		state.ViewedFiles = make(map[string]cache.FileReviewState)
	}
	markVisibleOnly := a.currentReviewContext.Scope != reviewprogress.ScopeAll
	for path, digest := range files {
		if markVisibleOnly {
			file, ok := a.currentReviewContext.FindFile(path)
			if !ok || !file.Actionable {
				continue
			}
		}
		state.ViewedFiles[path] = cache.FileReviewState{
			ViewedAt:      now,
			ViewedHeadSHA: headSHA,
			PatchDigest:   digest,
		}
	}
	if state.ActiveScope == "" {
		state.ActiveScope = string(a.currentReviewContext.Scope)
	}
	a.repoState.SetReviewState(a.currentReviewPR, state)
	a.saveRepoState()
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
	case core.ViewSmartCheckout:
		return a.checkoutDialog.Update(msg)
	}
	return nil
}

// rebuildStyles propagates the updated theme to all view models and components
// without recreating them, preserving all state (PR data, cursor positions, etc.).
func (a *App) rebuildStyles() {
	s := a.styles

	// Update styles on all view models (preserves state).
	a.prList.SetStyles(s)
	a.prDetail.SetStyles(s)
	a.diffView.SetStyles(s)
	a.reviewForm.SetStyles(s)
	a.repoSwitcher.SetStyles(s)
	a.helpOverlay.SetStyles(s)
	a.inbox.SetStyles(s)
	a.tutorial.SetStyles(s)
	a.filterPanel.SetStyles(s)
	a.confirmDialog.SetStyles(s)
	a.checkoutDialog.SetStyles(s)

	// Update styles on components (preserves state).
	a.banner.SetStyles(s)
	a.header.SetStyles(s)
	a.status.SetStyles(s)
	a.toasts.SetStyles(s)
}

func (a *App) contentHeight() int {
	return max(1, a.height-2) // header + status bar
}

func (a *App) loadRepoState() {
	if a.repo.Owner == "" {
		return
	}
	state, err := cache.LoadRepoState(a.repo)
	if err != nil {
		return
	}
	a.repoState = state
	// Apply saved filter if it has non-default values.
	if state.LastFilter.State != "" {
		a.filterOpts = state.LastFilter
		a.prList.SetFilter(a.filterOpts)
		a.header.SetFilter(a.prList.FilterLabel())
	}
}

func (a *App) saveRepoState() {
	if a.repo.Owner == "" {
		return
	}
	_ = cache.SaveRepoState(a.repo, a.repoState)
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
	case a.view == core.ViewSmartCheckout:
		a.status.SetHints([]string{a.checkoutDialog.StatusHint()})
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

	case core.ViewSmartCheckout:
		return a.checkoutDialog.View()

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
	case core.ViewSmartCheckout:
		return "Smart Checkout"
	default:
		return ""
	}
}
