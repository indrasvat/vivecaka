package tui

import (
	"fmt"
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
	view     core.ViewState
	prevView core.ViewState
	width    int
	height   int
	ready    bool
	repo     domain.RepoRef
	username string

	// Shared
	keys   core.KeyMap
	styles core.Styles
	theme  core.Theme

	// Domain (injected)
	reader   domain.PRReader
	reviewer domain.PRReviewer
	writer   domain.PRWriter

	// Use cases
	listPRs     *usecase.ListPRs
	getPRDetail *usecase.GetPRDetail
	reviewPR    *usecase.ReviewPR
	checkoutPR  *usecase.CheckoutPR

	// View models
	prList       views.PRListModel
	prDetail     views.PRDetailModel
	diffView     views.DiffViewModel
	reviewForm   views.ReviewModel
	repoSwitcher views.RepoSwitcherModel
	helpOverlay  views.HelpModel
	inbox        views.InboxModel
	tutorial     views.TutorialModel

	// Components
	header *components.Header
	status *components.StatusBar
	toasts *components.ToastManager
}

// New creates a new App model.
func New(cfg *config.Config, opts ...Option) *App {
	theme := core.ThemeByName(cfg.General.Theme)
	styles := core.NewStyles(theme)
	keys := core.DefaultKeyMap()

	a := &App{
		cfg:    cfg,
		view:   core.ViewLoading,
		keys:   keys,
		theme:  theme,
		styles: styles,

		// View models
		prList:       views.NewPRListModel(styles, keys),
		prDetail:     views.NewPRDetailModel(styles, keys),
		diffView:     views.NewDiffViewModel(styles, keys),
		reviewForm:   views.NewReviewModel(styles, keys),
		repoSwitcher: views.NewRepoSwitcherModel(styles, keys),
		helpOverlay:  views.NewHelpModel(styles),
		inbox:        views.NewInboxModel(styles, keys),
		tutorial:     views.NewTutorialModel(styles),

		// Components
		header: components.NewHeader(styles),
		status: components.NewStatusBar(styles),
		toasts: components.NewToastManager(styles),
	}

	for _, opt := range opts {
		opt(a)
	}

	// Wire use cases from injected adapters.
	if a.reader != nil {
		a.listPRs = usecase.NewListPRs(a.reader)
		a.getPRDetail = usecase.NewGetPRDetail(a.reader)
	}
	if a.reviewer != nil {
		a.reviewPR = usecase.NewReviewPR(a.reviewer)
	}
	if a.writer != nil {
		a.checkoutPR = usecase.NewCheckoutPR(a.writer)
	}

	return a
}

func (a *App) Init() tea.Cmd {
	cmds := []tea.Cmd{detectUserCmd()}
	// If repo was provided via WithRepo, skip detection and load PRs directly.
	if a.repo.Owner != "" {
		a.header.SetRepo(a.repo)
		if a.listPRs != nil {
			cmds = append(cmds, loadPRsCmd(a.listPRs, a.repo, domain.ListOpts{State: domain.PRStateOpen}))
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

	case views.PRsLoadedMsg:
		return a.handlePRsLoaded(msg)

	case views.OpenPRMsg:
		return a.handleOpenPR(msg)

	case views.PRDetailLoadedMsg:
		return a.handlePRDetailLoaded(msg)

	case views.OpenDiffMsg:
		return a.handleOpenDiff(msg)

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
		return a, nil

	case views.SubmitReviewMsg:
		return a.handleSubmitReview(msg)

	case views.ReviewSubmittedMsg:
		return a.handleReviewSubmitted(msg)

	case views.CheckoutPRMsg:
		return a.handleCheckoutPR(msg)

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

	case views.PRListFilterMsg:
		a.header.SetFilter(msg.Label)
		return a, nil

	case views.SwitchRepoMsg:
		return a.handleSwitchRepo(msg)

	case views.CloseRepoSwitcherMsg:
		a.view = a.prevView
		return a, nil

	case views.CloseHelpMsg:
		a.view = a.prevView
		return a, nil

	case views.CloseInboxMsg:
		a.view = core.ViewPRList
		return a, nil

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

	// Propagate to all view models.
	a.prList.SetSize(a.width, contentHeight)
	a.prDetail.SetSize(a.width, contentHeight)
	a.diffView.SetSize(a.width, contentHeight)
	a.reviewForm.SetSize(a.width, contentHeight)
	a.repoSwitcher.SetSize(a.width, contentHeight)
	a.helpOverlay.SetSize(a.width, contentHeight)
	a.inbox.SetSize(a.width, contentHeight)
	a.tutorial.SetSize(a.width, contentHeight)

	// Components.
	a.header.SetWidth(a.width)
	a.status.SetWidth(a.width)
	a.toasts.SetWidth(a.width)

	return a, nil
}

func (a *App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
		}
		return a, nil

	case key.Matches(msg, a.keys.Refresh):
		if a.view == core.ViewPRList && a.listPRs != nil && a.repo.Owner != "" {
			return a, loadPRsCmd(a.listPRs, a.repo, domain.ListOpts{State: domain.PRStateOpen})
		}
		return a, nil

	case key.Matches(msg, a.keys.Back):
		return a.handleBack()
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

	if a.listPRs != nil {
		return a, loadPRsCmd(a.listPRs, a.repo, domain.ListOpts{State: domain.PRStateOpen})
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
		a.view = core.ViewPRList
		a.prList.SetPRs(nil)
		return a, cmd
	}
	a.view = core.ViewPRList
	a.prList.SetPRs(msg.PRs)
	a.header.SetPRCount(len(msg.PRs))
	a.header.SetFilter(a.prList.FilterLabel())
	return a, nil
}

func (a *App) handleOpenPR(msg views.OpenPRMsg) (tea.Model, tea.Cmd) {
	a.view = core.ViewPRDetail
	if a.getPRDetail != nil && a.repo.Owner != "" {
		return a, loadPRDetailCmd(a.getPRDetail, a.repo, msg.Number)
	}
	return a, nil
}

func (a *App) handlePRDetailLoaded(msg views.PRDetailLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		cmd := a.toasts.Add(
			fmt.Sprintf("Error loading PR detail: %v", msg.Err),
			domain.ToastError, 5*time.Second,
		)
		return a, cmd
	}
	a.prDetail.SetDetail(msg.Detail)
	return a, nil
}

func (a *App) handleOpenDiff(msg views.OpenDiffMsg) (tea.Model, tea.Cmd) {
	a.view = core.ViewDiff
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

func (a *App) handleCheckoutPR(msg views.CheckoutPRMsg) (tea.Model, tea.Cmd) {
	if a.checkoutPR != nil && a.repo.Owner != "" {
		return a, checkoutPRCmd(a.checkoutPR, a.repo, msg.Number)
	}
	return a, nil
}

func (a *App) handleCheckoutDone(msg views.CheckoutDoneMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		cmd := a.toasts.Add(
			fmt.Sprintf("Checkout failed: %v", msg.Err),
			domain.ToastError, 5*time.Second,
		)
		return a, cmd
	}
	cmd := a.toasts.Add(
		fmt.Sprintf("Checked out branch: %s", msg.Branch),
		domain.ToastSuccess, 3*time.Second,
	)
	a.prList.SetCurrentBranch(msg.Branch)
	return a, cmd
}

func (a *App) handleSwitchRepo(msg views.SwitchRepoMsg) (tea.Model, tea.Cmd) {
	a.repo = msg.Repo
	a.header.SetRepo(a.repo)
	a.view = core.ViewPRList

	if a.listPRs != nil {
		return a, loadPRsCmd(a.listPRs, a.repo, domain.ListOpts{State: domain.PRStateOpen})
	}
	return a, nil
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

	a.header = components.NewHeader(s)
	a.status = components.NewStatusBar(s)
	a.toasts = components.NewToastManager(s)

	// Re-set repo on header.
	a.header.SetRepo(a.repo)

	// Re-set sizes.
	ch := a.contentHeight()
	a.prList.SetSize(a.width, ch)
	a.prDetail.SetSize(a.width, ch)
	a.diffView.SetSize(a.width, ch)
	a.reviewForm.SetSize(a.width, ch)
	a.repoSwitcher.SetSize(a.width, ch)
	a.helpOverlay.SetSize(a.width, ch)
	a.inbox.SetSize(a.width, ch)
	a.tutorial.SetSize(a.width, ch)
	a.header.SetWidth(a.width)
	a.status.SetWidth(a.width)
	a.toasts.SetWidth(a.width)
}

func (a *App) contentHeight() int {
	return max(1, a.height-2) // header + status bar
}

func (a *App) View() string {
	if !a.ready {
		return ""
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

	// Status bar.
	a.status.SetHints([]string{views.StatusHints(a.view, a.width)})
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
		return lipgloss.NewStyle().
			Width(a.width).Height(height).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(a.theme.Muted).
			Render("Loading...")

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
	default:
		return ""
	}
}
