package views

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/indrasvat/vivecaka/internal/tui/core"
)

// PRListModel implements the PR list view.
type PRListModel struct {
	prs           []domain.PR
	filtered      []domain.PR
	cursor        int
	offset        int
	width         int
	height        int
	styles        core.Styles
	keys          core.KeyMap
	loading       bool
	searchQuery   string
	searching     bool
	sortField     string
	sortAsc       bool
	sortPending   bool
	currentBranch string
	filter        domain.ListOpts
	username      string
	quickFilter   quickFilter
	panelLabel    string

	// Pagination state
	page        int  // current page (1-based)
	perPage     int  // items per page
	hasMore     bool // are there more PRs to load?
	loadingMore bool // currently loading more PRs?

	// Spinner animation
	spinnerFrame int
}

// NewPRListModel creates a new PR list view.
func NewPRListModel(styles core.Styles, keys core.KeyMap) PRListModel {
	return PRListModel{
		styles:    styles,
		keys:      keys,
		loading:   true,
		sortField: "updated",
		sortAsc:   false,
		filter:    domain.ListOpts{State: domain.PRStateOpen},
		page:      1,
		perPage:   50, // default, can be overridden via SetPerPage
		hasMore:   true,
	}
}

// SetPerPage sets the page size for pagination.
func (m *PRListModel) SetPerPage(n int) {
	if n > 0 {
		m.perPage = n
	}
}

// SetSize updates the view dimensions.
func (m *PRListModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetPRs updates the PR list data (initial load, replaces all PRs).
func (m *PRListModel) SetPRs(prs []domain.PR) {
	m.prs = prs
	m.loading = false
	m.page = 1
	// If we got fewer than perPage, there are no more pages
	m.hasMore = len(prs) >= m.perPage
	m.loadingMore = false
	m.applyFilter()
}

// AppendPRs adds more PRs to the existing list (pagination).
func (m *PRListModel) AppendPRs(prs []domain.PR, hasMore bool) {
	m.prs = append(m.prs, prs...)
	m.hasMore = hasMore
	m.loadingMore = false
	m.applyFilter()
}

// IsLoadingMore returns true if more PRs are being loaded.
func (m *PRListModel) IsLoadingMore() bool {
	return m.loadingMore
}

// HasMore returns true if there are more PRs to load.
func (m *PRListModel) HasMore() bool {
	return m.hasMore
}

// CurrentPage returns the current page number.
func (m *PRListModel) CurrentPage() int {
	return m.page
}

// PerPage returns the page size.
func (m *PRListModel) PerPage() int {
	return m.perPage
}

// SetLoadingMore marks that more PRs are being loaded and starts the spinner.
// Returns a command to start the spinner animation.
func (m *PRListModel) SetLoadingMore(page int) tea.Cmd {
	m.loadingMore = true
	m.page = page
	m.spinnerFrame = 0
	return m.spinnerTick()
}

// SetFilter updates the active filter options.
// This resets pagination state since filters change the result set.
func (m *PRListModel) SetFilter(opts domain.ListOpts) {
	m.filter = opts
	m.panelLabel = filterLabelFromOpts(opts)
	m.cursor = 0
	m.offset = 0
	// Reset pagination when filter changes
	m.page = 1
	m.hasMore = true
	m.loadingMore = false
	m.applyFilter()
}

// SetCurrentBranch updates the current git branch for highlight.
func (m *PRListModel) SetCurrentBranch(branch string) {
	m.currentBranch = branch
}

// SelectedPR returns the currently selected PR, if any.
func (m *PRListModel) SelectedPR() *domain.PR {
	if len(m.filtered) == 0 || m.cursor >= len(m.filtered) {
		return nil
	}
	pr := m.filtered[m.cursor]
	return &pr
}

// IsLoading returns true if the PR list is still loading.
func (m *PRListModel) IsLoading() bool {
	return m.loading
}

// HasPRs returns true if PRs have been loaded (even if filtered list is empty).
func (m *PRListModel) HasPRs() bool {
	return !m.loading
}

// FilteredPRs returns the current filtered PR list.
func (m *PRListModel) FilteredPRs() []domain.PR {
	return m.filtered
}

// TotalPRs returns the total number of PRs loaded (before filtering).
func (m *PRListModel) TotalPRs() int {
	return len(m.prs)
}

// PRListMsg types for communication with parent.
type (
	PRsLoadedMsg struct {
		PRs []domain.PR
		Err error
	}
	OpenPRMsg     struct{ Number int }
	CheckoutPRMsg struct {
		Number int
		Branch string
	}
	CopyURLMsg     struct{ URL string }
	OpenBrowserMsg struct{ URL string }
)

// SpinnerTickMsg is sent to animate the loading spinner.
type SpinnerTickMsg struct{}

type quickFilter string

const (
	quickFilterNone        quickFilter = ""
	quickFilterMyPRs       quickFilter = "my_prs"
	quickFilterNeedsReview quickFilter = "needs_review"
)

// Update handles messages for the PR list view.
func (m *PRListModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.searching {
			return m.handleSearchKey(msg)
		}
		return m.handleKey(msg)
	case PRsLoadedMsg:
		m.SetPRs(msg.PRs)
	case SpinnerTickMsg:
		if m.loadingMore {
			m.spinnerFrame++
			return m.spinnerTick()
		}
	}
	return nil
}

// spinnerTick returns a command that sends a SpinnerTickMsg after a delay.
func (m *PRListModel) spinnerTick() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg {
		return SpinnerTickMsg{}
	})
}

func (m *PRListModel) handleKey(msg tea.KeyMsg) tea.Cmd {
	listLen := len(m.filtered)
	if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 {
		switch msg.Runes[0] {
		case 'm':
			return m.toggleQuickFilter(quickFilterMyPRs)
		case 'n':
			return m.toggleQuickFilter(quickFilterNeedsReview)
		}
	}

	if listLen == 0 {
		switch {
		case key.Matches(msg, m.keys.Search):
			m.searching = true
			m.searchQuery = ""
		case key.Matches(msg, m.keys.Filter):
			return func() tea.Msg { return OpenFilterMsg{} }
		}
		return nil
	}

	switch {
	case key.Matches(msg, m.keys.Down):
		if m.cursor < listLen-1 {
			m.cursor++
			m.ensureVisible()
			// Check if we need to load more PRs
			if cmd := m.checkLoadMore(); cmd != nil {
				return cmd
			}
		}
	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
			m.ensureVisible()
		}
	case key.Matches(msg, m.keys.HalfPageDown):
		visible := m.visibleRows()
		m.cursor += visible / 2
		if m.cursor >= listLen {
			m.cursor = listLen - 1
		}
		m.ensureVisible()
		// Check if we need to load more PRs
		if cmd := m.checkLoadMore(); cmd != nil {
			return cmd
		}
	case key.Matches(msg, m.keys.HalfPageUp):
		visible := m.visibleRows()
		m.cursor -= visible / 2
		if m.cursor < 0 {
			m.cursor = 0
		}
		m.ensureVisible()
	case key.Matches(msg, m.keys.Top):
		m.cursor = 0
		m.offset = 0
	case key.Matches(msg, m.keys.Bottom):
		m.cursor = listLen - 1
		m.ensureVisible()
		// Check if we need to load more PRs
		if cmd := m.checkLoadMore(); cmd != nil {
			return cmd
		}
	case key.Matches(msg, m.keys.Enter):
		if pr := m.SelectedPR(); pr != nil {
			return func() tea.Msg { return OpenPRMsg{Number: pr.Number} }
		}
	case key.Matches(msg, m.keys.Checkout):
		if pr := m.SelectedPR(); pr != nil {
			return func() tea.Msg { return CheckoutPRMsg{Number: pr.Number, Branch: pr.Branch.Head} }
		}
	case key.Matches(msg, m.keys.Yank):
		if pr := m.SelectedPR(); pr != nil {
			return func() tea.Msg { return CopyURLMsg{URL: pr.URL} }
		}
	case key.Matches(msg, m.keys.Open):
		if pr := m.SelectedPR(); pr != nil {
			return func() tea.Msg { return OpenBrowserMsg{URL: pr.URL} }
		}
	case key.Matches(msg, m.keys.Search):
		m.searching = true
		m.searchQuery = ""
	case key.Matches(msg, m.keys.Filter):
		return func() tea.Msg { return OpenFilterMsg{} }
	case key.Matches(msg, m.keys.Sort):
		m.cycleSort()
	}
	return nil
}

func (m *PRListModel) handleSearchKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.Type {
	case tea.KeyEscape:
		m.searching = false
		m.searchQuery = ""
		m.applyFilter()
	case tea.KeyEnter:
		m.searching = false
		m.applyFilter()
	case tea.KeyBackspace:
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
			m.applyFilter()
		}
	case tea.KeyRunes:
		m.searchQuery += string(msg.Runes)
		m.applyFilter()
	}
	return nil
}

func (m *PRListModel) cycleSort() {
	fields := []string{"updated", "created", "number", "title", "author"}
	if m.sortPending {
		m.sortAsc = !m.sortAsc
		m.sortPending = false
		m.applyFilter()
		return
	}

	for i, f := range fields {
		if f == m.sortField {
			if i+1 < len(fields) {
				m.sortField = fields[i+1]
			} else {
				m.sortField = fields[0]
			}
			break
		}
	}
	m.sortAsc = false
	m.sortPending = true
	m.applyFilter()
}

func (m *PRListModel) applyFilter() {
	result := make([]domain.PR, 0, len(m.prs))
	query := strings.ToLower(m.searchQuery)

	for _, pr := range m.prs {
		if !m.matchesQuickFilter(pr) {
			continue
		}
		if !m.matchesPanelFilter(pr) {
			continue
		}
		if query != "" {
			titleMatch := strings.Contains(strings.ToLower(pr.Title), query)
			authorMatch := strings.Contains(strings.ToLower(pr.Author), query)
			if !titleMatch && !authorMatch {
				continue
			}
		}
		result = append(result, pr)
	}

	sort.SliceStable(result, func(i, j int) bool {
		cmp := m.comparePR(result[i], result[j])
		if cmp == 0 {
			return false
		}
		if m.sortAsc {
			return cmp < 0
		}
		return cmp > 0
	})

	m.filtered = result
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
	m.ensureVisible()
}

func (m *PRListModel) visibleRows() int {
	rows := m.height - 3 // header row + separator + status
	if rows < 1 {
		return 1
	}
	return rows
}

func (m *PRListModel) ensureVisible() {
	visible := m.visibleRows()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+visible {
		m.offset = m.cursor - visible + 1
	}
}

// checkLoadMore checks if we should load more PRs and returns a command if so.
// Triggers when cursor is within 5 items of the bottom and more PRs are available.
func (m *PRListModel) checkLoadMore() tea.Cmd {
	// Don't load more if already loading, no more pages, or list is empty
	if m.loadingMore || !m.hasMore || len(m.filtered) == 0 {
		return nil
	}

	// Trigger load when within 5 items of bottom
	distanceFromBottom := len(m.filtered) - 1 - m.cursor
	if distanceFromBottom <= 5 {
		nextPage := m.page + 1
		return func() tea.Msg {
			return LoadMorePRsMsg{Page: nextPage}
		}
	}
	return nil
}

// View renders the PR list.
func (m *PRListModel) View() string {
	if m.loading {
		content := lipgloss.NewStyle().
			Foreground(m.styles.Theme.Muted).
			Render("Loading PRs...")
		centered := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
		return ensureExactHeight(centered, m.height, m.width)
	}

	if len(m.filtered) == 0 {
		msg := "No pull requests found"
		if m.searchQuery != "" {
			msg = fmt.Sprintf("No PRs matching %q", m.searchQuery)
		}
		content := lipgloss.NewStyle().
			Foreground(m.styles.Theme.Muted).
			Render(msg)
		centered := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
		return ensureExactHeight(centered, m.height, m.width)
	}

	var rows []string

	// Column headers row.
	rows = append(rows, m.renderColumnHeaders())
	rows = append(rows, m.renderTableSeparator())

	// PR rows.
	visible := m.visibleRows()
	end := min(m.offset+visible, len(m.filtered))

	for i := m.offset; i < end; i++ {
		rows = append(rows, m.renderPRRow(i, m.filtered[i]))
	}

	// Loading more indicator with animated spinner (shows at bottom when fetching next page).
	if m.loadingMore {
		spinnerFrames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		frame := spinnerFrames[m.spinnerFrame%len(spinnerFrames)]
		spinnerStyle := lipgloss.NewStyle().Foreground(m.styles.Theme.Primary)
		textStyle := lipgloss.NewStyle().Foreground(m.styles.Theme.Muted).Italic(true)
		indicator := spinnerStyle.Render(frame) + textStyle.Render(" Loading more PRs...")
		rows = append(rows, "  "+indicator)
	}

	// Search bar (replaces one row if active).
	if m.searching {
		rows = append(rows, m.renderSearchBar())
	}

	// Join rows into content
	content := lipgloss.JoinVertical(lipgloss.Left, rows...)

	// Ensure exact height with full-width padding lines
	// This is critical for overwriting previous screen content (e.g., banner)
	return ensureExactHeight(content, m.height, m.width)
}

// ensureExactHeight pads or truncates content to exactly the specified height.
// Each padding line is full-width spaces to properly overwrite previous content.
// This pattern is from yukti - see CLAUDE.md for details.
func ensureExactHeight(content string, height, width int) string {
	lines := strings.Split(content, "\n")

	// Truncate if too many lines
	if len(lines) > height {
		lines = lines[:height]
	}

	// Create a full-width empty line for padding
	emptyLine := strings.Repeat(" ", width)

	// Pad if too few lines - use full-width empty lines
	for len(lines) < height {
		lines = append(lines, emptyLine)
	}

	return strings.Join(lines, "\n")
}

func (m *PRListModel) renderColumnHeaders() string {
	t := m.styles.Theme
	header := lipgloss.NewStyle().Foreground(t.Muted)

	cols := m.columns()
	ageLabel := "Age"
	switch m.sortField {
	case "updated":
		ageLabel = m.sortLabel("updated", "Age")
	case "created":
		ageLabel = m.sortLabel("created", "Age")
	}

	// Format: "  #   Title                              Author       CI  Review   Age"
	// Left-padded by 2 spaces to align with row indicator column
	return header.Render(fmt.Sprintf("  %-*s  %-*s  %-*s  %-*s  %-*s  %-*s",
		cols.num, m.sortLabel("number", "#"),
		cols.title, m.sortLabel("title", "Title"),
		cols.author, m.sortLabel("author", "Author"),
		cols.ci, "CI",
		cols.review, "Review",
		cols.age, ageLabel,
	))
}

func (m *PRListModel) renderTableSeparator() string {
	t := m.styles.Theme
	cols := m.columns()

	// Build separator with + characters at column boundaries like mock:
	// " -----+---------------------------------+------------+----+--------+-----"
	sep := lipgloss.NewStyle().Foreground(t.Border)

	// Use ASCII hyphen (-) not box-drawing character, matches mock
	numSep := strings.Repeat("-", cols.num+1)
	titleSep := strings.Repeat("-", cols.title+1)
	authorSep := strings.Repeat("-", cols.author+1)
	ciSep := strings.Repeat("-", cols.ci+1)
	reviewSep := strings.Repeat("-", cols.review+1)
	ageSep := strings.Repeat("-", cols.age+1)

	line := " " + numSep + "+" + titleSep + "+" + authorSep + "+" + ciSep + "+" + reviewSep + "+" + ageSep

	return sep.Render(line)
}

type colWidths struct {
	num, title, author, ci, review, age int
}

func (m *PRListModel) columns() colWidths {
	// Fixed columns: num(4) + author(12) + ci(4) + review(8) + age(5) = 33
	// Plus spaces between columns: 6 columns * 2 spaces = 12
	// Plus left indicator: 2 chars
	// Total fixed: 33 + 12 + 2 = 47
	fixedWidth := 47
	titleWidth := max(20, m.width-fixedWidth)
	return colWidths{
		num:    4,
		title:  titleWidth,
		author: 12,
		ci:     4,
		review: 8,
		age:    5,
	}
}

func (m *PRListModel) renderPRRow(idx int, pr domain.PR) string {
	t := m.styles.Theme
	cols := m.columns()

	selected := idx == m.cursor
	isBranch := m.currentBranch != "" && pr.Branch.Head == m.currentBranch
	isDraft := pr.Draft

	// Title with draft prefix.
	title := pr.Title
	if isDraft {
		title = "[DRAFT] " + title
	}
	if len(title) > cols.title {
		title = title[:cols.title-1] + "…"
	}

	// Determine styles based on row state
	var numStyle, titleStyle, authorStyle, ageStyle lipgloss.Style
	leftIndicator := " " // space for non-selected

	switch {
	case selected:
		// Selected row: mauve left border, bold number
		leftIndicator = lipgloss.NewStyle().Foreground(t.Primary).Render("│")
		numStyle = lipgloss.NewStyle().Foreground(t.Primary).Bold(true)
		titleStyle = lipgloss.NewStyle().Foreground(t.Fg).Bold(true)
		authorStyle = lipgloss.NewStyle().Foreground(t.Secondary)
		ageStyle = lipgloss.NewStyle().Foreground(t.Subtext)
	case isDraft:
		// Draft: all dimmed
		numStyle = lipgloss.NewStyle().Foreground(t.Muted)
		titleStyle = lipgloss.NewStyle().Foreground(t.Muted)
		authorStyle = lipgloss.NewStyle().Foreground(t.Muted)
		ageStyle = lipgloss.NewStyle().Foreground(t.Muted)
	case isBranch:
		// Current branch: highlighted number
		numStyle = lipgloss.NewStyle().Foreground(t.Primary)
		titleStyle = lipgloss.NewStyle().Foreground(t.Fg)
		authorStyle = lipgloss.NewStyle().Foreground(t.Secondary)
		ageStyle = lipgloss.NewStyle().Foreground(t.Subtext)
	default:
		numStyle = lipgloss.NewStyle().Foreground(t.Fg)
		titleStyle = lipgloss.NewStyle().Foreground(t.Fg)
		authorStyle = lipgloss.NewStyle().Foreground(t.Secondary)
		ageStyle = lipgloss.NewStyle().Foreground(t.Subtext)
	}

	// Author truncation
	author := pr.Author
	if len(author) > cols.author {
		author = author[:cols.author-1] + "…"
	}

	// CI icon with proper color
	ci := m.renderCIIcon(pr.CI)

	// Review with proper color
	review := m.renderReviewText(pr.Review)

	// Age
	age := relativeTime(pr.UpdatedAt)

	// Build the row with consistent column spacing
	// Format matches column headers: "  #   Title...  Author  CI  Review  Age"
	num := numStyle.Render(fmt.Sprintf("%*d", cols.num, pr.Number))
	titleText := titleStyle.Render(fmt.Sprintf("%-*s", cols.title, title))
	authorText := authorStyle.Render(fmt.Sprintf("%-*s", cols.author, author))
	ageText := ageStyle.Render(fmt.Sprintf("%-*s", cols.age, age))

	// Pad CI and review using visual width (ANSI codes don't count toward visual width)
	ciPad := max(0, cols.ci-lipgloss.Width(ci))
	ciPadded := ci + strings.Repeat(" ", ciPad)

	reviewPad := max(0, cols.review-lipgloss.Width(review))
	reviewPadded := review + strings.Repeat(" ", reviewPad)

	// Build row: indicator + space + num + spaces + title + spaces + author + spaces + ci + spaces + review + spaces + age
	row := fmt.Sprintf("%s %s  %s  %s  %s  %s  %s",
		leftIndicator,
		num,
		titleText,
		authorText,
		ciPadded,
		reviewPadded,
		ageText,
	)

	return row
}

// renderCIIcon returns the colored CI status icon.
func (m *PRListModel) renderCIIcon(status domain.CIStatus) string {
	t := m.styles.Theme
	switch status {
	case domain.CIPass:
		return lipgloss.NewStyle().Foreground(t.Success).Render("✓")
	case domain.CIFail:
		return lipgloss.NewStyle().Foreground(t.Error).Render("✗")
	case domain.CIPending:
		return lipgloss.NewStyle().Foreground(t.Warning).Render("◐")
	case domain.CISkipped:
		return lipgloss.NewStyle().Foreground(t.Muted).Render("○")
	default:
		return lipgloss.NewStyle().Foreground(t.Muted).Render("—")
	}
}

// renderReviewText returns the colored review status text.
func (m *PRListModel) renderReviewText(r domain.ReviewStatus) string {
	t := m.styles.Theme
	switch r.State {
	case domain.ReviewApproved:
		return lipgloss.NewStyle().Foreground(t.Success).Render(fmt.Sprintf("✓ %d/%d", r.Approved, r.Total))
	case domain.ReviewChangesRequested:
		// Use warning/peach color for changes requested
		return lipgloss.NewStyle().Foreground(t.Warning).Render(fmt.Sprintf("! %d/%d", r.Approved, r.Total))
	case domain.ReviewPending:
		return lipgloss.NewStyle().Foreground(t.Warning).Render(fmt.Sprintf("● %d/%d", r.Approved, r.Total))
	default:
		return lipgloss.NewStyle().Foreground(t.Muted).Render("—")
	}
}

func (m *PRListModel) renderSearchBar() string {
	t := m.styles.Theme
	style := lipgloss.NewStyle().Foreground(t.Info)
	return style.Render(fmt.Sprintf("/ %s▎", m.searchQuery))
}

// SetUsername sets the current GitHub username for quick filters.
func (m *PRListModel) SetUsername(username string) {
	m.username = username
}

// FilterLabel returns the label for the active quick filter.
func (m *PRListModel) FilterLabel() string {
	switch m.quickFilter {
	case quickFilterMyPRs:
		return "My PRs"
	case quickFilterNeedsReview:
		return "Needs Review"
	default:
		return m.panelLabel
	}
}

func (m *PRListModel) toggleQuickFilter(filter quickFilter) tea.Cmd {
	if m.quickFilter == filter {
		m.quickFilter = quickFilterNone
	} else {
		m.quickFilter = filter
	}
	m.cursor = 0
	m.offset = 0
	m.applyFilter()
	return func() tea.Msg { return PRListFilterMsg{Label: m.FilterLabel()} }
}

func (m *PRListModel) matchesQuickFilter(pr domain.PR) bool {
	switch m.quickFilter {
	case quickFilterMyPRs:
		return m.username != "" && strings.EqualFold(pr.Author, m.username)
	case quickFilterNeedsReview:
		if m.username == "" {
			return false
		}
		return pr.Review.State == domain.ReviewPending && !strings.EqualFold(pr.Author, m.username)
	default:
		return true
	}
}

func (m *PRListModel) matchesPanelFilter(pr domain.PR) bool {
	if m.filter.State != "" && m.filter.State != "all" && pr.State != m.filter.State {
		return false
	}
	if m.filter.Author != "" && !strings.Contains(strings.ToLower(pr.Author), strings.ToLower(m.filter.Author)) {
		return false
	}
	if len(m.filter.Labels) > 0 {
		for _, label := range m.filter.Labels {
			if !hasLabel(pr.Labels, label) {
				return false
			}
		}
	}
	if m.filter.CI != "" && pr.CI != m.filter.CI {
		return false
	}
	if m.filter.Review != "" && pr.Review.State != m.filter.Review {
		return false
	}
	if m.filter.Draft == domain.DraftExclude && pr.Draft {
		return false
	}
	if m.filter.Draft == domain.DraftOnly && !pr.Draft {
		return false
	}
	return true
}

func hasLabel(labels []string, want string) bool {
	for _, label := range labels {
		if strings.EqualFold(label, want) {
			return true
		}
	}
	return false
}

func filterLabelFromOpts(opts domain.ListOpts) string {
	active := (opts.State != "" && opts.State != domain.PRStateOpen && opts.State != "all") ||
		strings.TrimSpace(opts.Author) != "" ||
		len(opts.Labels) > 0 ||
		opts.CI != "" ||
		opts.Review != "" ||
		(opts.Draft != "" && opts.Draft != domain.DraftInclude)

	if active {
		return "Filtered"
	}
	return ""
}

func (m *PRListModel) sortLabel(field, label string) string {
	if m.sortField != field {
		return label
	}
	if m.sortAsc {
		return label + "▲"
	}
	return label + "▼"
}

func (m *PRListModel) comparePR(a, b domain.PR) int {
	switch m.sortField {
	case "updated":
		return compareTime(a.UpdatedAt, b.UpdatedAt)
	case "created":
		return compareTime(nonZeroTime(a.CreatedAt, a.UpdatedAt), nonZeroTime(b.CreatedAt, b.UpdatedAt))
	case "number":
		return compareInt(a.Number, b.Number)
	case "title":
		return compareStringFold(a.Title, b.Title)
	case "author":
		return compareStringFold(a.Author, b.Author)
	default:
		return compareTime(a.UpdatedAt, b.UpdatedAt)
	}
}

func compareTime(a, b time.Time) int {
	switch {
	case a.Before(b):
		return -1
	case a.After(b):
		return 1
	default:
		return 0
	}
}

func compareInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func compareStringFold(a, b string) int {
	return strings.Compare(strings.ToLower(a), strings.ToLower(b))
}

func nonZeroTime(primary, fallback time.Time) time.Time {
	if primary.IsZero() {
		return fallback
	}
	return primary
}

// ciIcon returns the Unicode symbol for a CI status.
func ciIcon(status domain.CIStatus) string {
	switch status {
	case domain.CIPass:
		return "✓"
	case domain.CIFail:
		return "✗"
	case domain.CIPending:
		return "◐"
	case domain.CISkipped:
		return "○"
	default:
		return "—"
	}
}

// reviewText returns the formatted review status string.
func reviewText(r domain.ReviewStatus) string {
	switch r.State {
	case domain.ReviewApproved:
		return fmt.Sprintf("✓ %d/%d", r.Approved, r.Total)
	case domain.ReviewChangesRequested:
		return fmt.Sprintf("! %d/%d", r.Approved, r.Total)
	case domain.ReviewPending:
		return fmt.Sprintf("● %d/%d", r.Approved, r.Total)
	default:
		return "—"
	}
}

// relativeTime formats a time as a relative duration string.
func relativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "<1m"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	default:
		return fmt.Sprintf("%dmo", int(d.Hours()/(24*30)))
	}
}
