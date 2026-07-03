package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/indrasvat/vivecaka/internal/tui/core"
)

// InboxTab represents filtering tabs in the Unified PR Inbox.
type InboxTab int

const (
	InboxAll InboxTab = iota
	InboxAssigned
	InboxReviewRequested
	InboxMyPRs
)

// InboxPR wraps a PR with its source repo.
type InboxPR = domain.InboxItem

// InboxModel implements the Unified PR Inbox (S4).
type InboxModel struct {
	allPRs   []InboxPR
	filtered []InboxPR
	tab      InboxTab
	cursor   int
	offset   int
	width    int
	height   int
	styles   core.Styles
	keys     core.KeyMap
	loading  bool
	username string // current user for tab filtering
	rank     domain.InboxRankProfile
	cachedAt time.Time
	rate     domain.InboxRateLimit
	sources  []domain.InboxSourceStatus
	staged   []InboxPR
	insights map[string]domain.InboxInsight
}

// SetStyles updates the styles without losing state.
func (m *InboxModel) SetStyles(s core.Styles) { m.styles = s }

// NewInboxModel creates a new inbox view.
func NewInboxModel(styles core.Styles, keys core.KeyMap) InboxModel {
	return InboxModel{
		styles:   styles,
		keys:     keys,
		loading:  true,
		insights: make(map[string]domain.InboxInsight),
	}
}

// SetSize updates the view dimensions.
func (m *InboxModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetUsername sets the current GitHub username for filtering.
func (m *InboxModel) SetUsername(u string) {
	m.username = u
}

// SetPRs updates the inbox with PRs from all repos.
func (m *InboxModel) SetPRs(prs []InboxPR) {
	// Apply priority sort before storing.
	PrioritySort(prs, m.username, 7)
	m.allPRs = prs
	m.loading = false
	m.applyFilter()
}

// SetResult updates the inbox from the attention inbox use case.
func (m *InboxModel) SetResult(result domain.InboxResult, profile domain.InboxRankProfile, fresh bool) {
	selected := m.selectedKey()
	m.rank = profile
	m.cachedAt = result.CachedAt
	m.rate = result.Rate
	m.sources = result.Sources
	m.loading = false

	if !fresh || len(m.allPRs) == 0 {
		m.allPRs = result.Items
		m.applyFilter()
		m.restoreSelection(selected)
		return
	}

	if selected != "" && selected != inboxKey(result.Items, m.cursor) {
		m.staged = result.Items
		return
	}
	m.allPRs = result.Items
	m.applyFilter()
	m.restoreSelection(selected)
}

func (m *InboxModel) SetLoading() {
	m.loading = true
	m.staged = nil
}

// TotalPRs returns the number of loaded inbox rows.
func (m *InboxModel) TotalPRs() int { return len(m.allPRs) }

// Message types.
type (
	InboxPRsLoadedMsg   struct{ PRs []InboxPR }
	InboxItemsLoadedMsg struct {
		Result  domain.InboxResult
		Profile domain.InboxRankProfile
		Fresh   bool
		Err     error
	}
	InboxInsightLoadedMsg struct {
		Key     string
		Insight domain.InboxInsight
		Err     error
	}
	OpenInboxPRMsg struct {
		Repo   domain.RepoRef
		Number int
	}
	FocusInboxRepoMsg struct{ Repo domain.RepoRef }
	CloseInboxMsg     struct{}
)

// Update handles messages for the inbox.
func (m *InboxModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case InboxPRsLoadedMsg:
		m.SetPRs(msg.PRs)
	case InboxItemsLoadedMsg:
		if msg.Err == nil {
			m.SetResult(msg.Result, msg.Profile, msg.Fresh)
		} else if len(m.allPRs) == 0 {
			m.loading = false
		}
	case InboxInsightLoadedMsg:
		m.ApplyInsight(msg)
	}
	return nil
}

func (m *InboxModel) handleKey(msg tea.KeyMsg) tea.Cmd {
	listLen := len(m.filtered)

	switch {
	case key.Matches(msg, m.keys.Back):
		return func() tea.Msg { return CloseInboxMsg{} }
	case key.Matches(msg, m.keys.Tab):
		m.tab = (m.tab + 1) % 4
		m.applyFilter()
	case key.Matches(msg, m.keys.ShiftTab):
		m.tab = (m.tab + 3) % 4
		m.applyFilter()
	case key.Matches(msg, m.keys.Down):
		if listLen > 0 && m.cursor < listLen-1 {
			m.cursor++
			m.ensureVisible()
		}
	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
			m.ensureVisible()
		}
	case key.Matches(msg, m.keys.Enter):
		if listLen > 0 && m.cursor < listLen {
			pr := m.filtered[m.cursor]
			return func() tea.Msg {
				return OpenInboxPRMsg{Repo: pr.Repo, Number: pr.Number}
			}
		}
	case msg.Type == tea.KeyRunes && len(msg.Runes) == 1 && msg.Runes[0] == 'f':
		if listLen > 0 && m.cursor < listLen {
			pr := m.filtered[m.cursor]
			return func() tea.Msg { return FocusInboxRepoMsg{Repo: pr.Repo} }
		}
	case msg.Type == tea.KeyRunes && len(msg.Runes) == 1 && msg.Runes[0] == 'u':
		if len(m.staged) > 0 {
			selected := m.selectedKey()
			m.allPRs = m.staged
			m.staged = nil
			m.applyFilter()
			m.restoreSelection(selected)
		}
	}
	return nil
}

// SelectedItem returns the currently focused inbox item.
func (m *InboxModel) SelectedItem() (InboxPR, bool) {
	if m.cursor < 0 || m.cursor >= len(m.filtered) {
		return InboxPR{}, false
	}
	return m.filtered[m.cursor], true
}

// SelectedKey returns the stable key for the focused inbox item.
func (m *InboxModel) SelectedKey() string { return m.selectedKey() }

// HasInsight reports whether the selected item already has lazy insight data.
func (m *InboxModel) HasInsight(key string) bool {
	if m.insights == nil {
		return false
	}
	_, ok := m.insights[key]
	return ok
}

// ApplyInsight stores selected-row insight data if it matches a real row.
func (m *InboxModel) ApplyInsight(msg InboxInsightLoadedMsg) {
	if msg.Err != nil || msg.Key == "" {
		return
	}
	if m.insights == nil {
		m.insights = make(map[string]domain.InboxInsight)
	}
	m.insights[msg.Key] = msg.Insight
}

func (m *InboxModel) applyFilter() {
	switch m.tab {
	case InboxAssigned:
		m.filtered = filterByAssigned(m.allPRs)
	case InboxReviewRequested:
		m.filtered = filterByReviewRequested(m.allPRs, m.username)
	case InboxMyPRs:
		m.filtered = filterByAuthor(m.allPRs, m.username)
	default:
		m.filtered = m.allPRs
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
	m.offset = 0
}

func filterByAssigned(prs []InboxPR) []InboxPR {
	var out []InboxPR
	for _, pr := range prs {
		if inboxHasSource(pr, domain.InboxSourceAssigned) {
			out = append(out, pr)
		}
	}
	return out
}

func filterByReviewRequested(prs []InboxPR, username string) []InboxPR {
	if username == "" {
		return nil
	}
	var out []InboxPR
	for _, pr := range prs {
		// Include PRs where review is pending and user is not the author.
		if inboxHasSource(pr, domain.InboxSourceAttention) && !strings.EqualFold(pr.Author, username) {
			out = append(out, pr)
		}
	}
	return out
}

func filterByAuthor(prs []InboxPR, username string) []InboxPR {
	if username == "" {
		return nil
	}
	var out []InboxPR
	for _, pr := range prs {
		if strings.EqualFold(pr.Author, username) {
			out = append(out, pr)
		}
	}
	return out
}

func (m *InboxModel) visibleRows() int {
	rows := m.height - 4 // tabs + columns + separator + selected cue allowance
	if rows < 1 {
		return 1
	}
	return rows
}

func (m *InboxModel) ensureVisible() {
	visible := m.visibleRows()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+visible {
		m.offset = m.cursor - visible + 1
	}
}

// View renders the inbox view.
func (m *InboxModel) View() string {
	if m.loading {
		line := lipgloss.NewStyle().Foreground(m.styles.Theme.Muted).Render("  ◐ Inbox")
		return m.padView(line)
	}

	t := m.styles.Theme

	subtle := lipgloss.NewStyle().Foreground(t.Muted)
	tabs := m.renderTabs()

	if len(m.filtered) == 0 {
		empty := subtle.Render("  — no PRs")
		parts := m.inboxChrome(tabs)
		parts = append(parts, empty)
		return m.padView(lipgloss.JoinVertical(lipgloss.Left, parts...))
	}

	rowBudget := m.height - len(m.inboxChrome(tabs))
	if rowBudget < 1 {
		rowBudget = 1
	}

	var rows []string
	used := 0
	for i := m.offset; i < len(m.filtered); i++ {
		row := m.renderRow(i, m.filtered[i])
		rowLines := strings.Count(row, "\n") + 1
		if used > 0 && used+rowLines > rowBudget {
			break
		}
		rows = append(rows, row)
		used += rowLines
	}

	parts := m.inboxChrome(tabs)
	parts = append(parts, rows...)
	return m.padView(lipgloss.JoinVertical(lipgloss.Left, parts...))
}

func (m *InboxModel) inboxChrome(tabs string) []string {
	return []string{tabs, m.renderColumnHeader(), m.renderTableSeparator()}
}

func (m *InboxModel) renderTabs() string {
	t := m.styles.Theme
	active := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Padding(0, 1)
	inactive := lipgloss.NewStyle().Foreground(t.Muted).Padding(0, 1)

	tabs := []string{"All", "Assigned", "Review", "Mine"}
	var rendered []string
	for i, tab := range tabs {
		if InboxTab(i) == m.tab {
			rendered = append(rendered, active.Render(tab))
		} else {
			rendered = append(rendered, inactive.Render(tab))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
}

func (m *InboxModel) renderRow(idx int, pr InboxPR) string {
	t := m.styles.Theme
	selected := idx == m.cursor

	prefix := "  "
	if selected {
		prefix = lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render("│ ")
	}

	cols := m.columns()
	repoName := truncateDisplay(pr.Repo.String(), cols.repo)
	titleWidth := cols.title
	title := padRight(truncateDisplay(pr.Title, titleWidth), titleWidth)
	author := truncateDisplay("@"+pr.Author, cols.author)
	source := padRight(sourceIcons(pr.Sources), cols.sig)
	ci := styledCIIcon(pr.CI, t)
	age := relativeTime(pr.UpdatedAt)

	repoStyle := lipgloss.NewStyle().Foreground(t.Secondary)
	titleStyle := lipgloss.NewStyle().Foreground(t.Fg)
	if selected {
		repoStyle = repoStyle.Bold(true)
		titleStyle = titleStyle.Bold(true)
	}
	metaStyle := lipgloss.NewStyle().Foreground(t.Muted)
	scoreStyle := lipgloss.NewStyle().Foreground(t.Subtext)

	row := lipgloss.JoinHorizontal(lipgloss.Top,
		prefix,
		lipgloss.NewStyle().Foreground(t.Primary).Render(source),
		"  ",
		repoStyle.Render(padRight(repoName, cols.repo)),
		"  ",
		metaStyle.Render(fmt.Sprintf("%*d", cols.num, pr.Number)),
		"  ",
		titleStyle.Render(title),
		"  ",
		metaStyle.Render(padRight(author, cols.author)),
		"  ",
		ci,
		strings.Repeat(" ", max(0, cols.ci-lipgloss.Width(ci))),
		"  ",
		scoreStyle.Render(age),
	)

	row = fitLine(row, m.width)
	if selected {
		if cue := m.renderActionCue(pr); cue != "" {
			row += "\n" + cue
		}
	}
	return row
}

func (m *InboxModel) renderColumnHeader() string {
	t := m.styles.Theme
	header := lipgloss.NewStyle().Foreground(t.Muted)
	cols := m.columns()
	return header.Render(fmt.Sprintf("  %-*s  %-*s  %-*s  %-*s  %-*s  %-*s  %-*s",
		cols.sig, "Src",
		cols.repo, "Repo",
		cols.num, "#",
		cols.title, "Title",
		cols.author, "Author",
		cols.ci, "CI",
		cols.age, "Age",
	))
}

func (m *InboxModel) renderTableSeparator() string {
	t := m.styles.Theme
	cols := m.columns()
	sep := lipgloss.NewStyle().Foreground(t.Border)
	sigSep := strings.Repeat("-", cols.sig+1)
	repoSep := strings.Repeat("-", cols.repo+1)
	numSep := strings.Repeat("-", cols.num+1)
	titleSep := strings.Repeat("-", cols.title+1)
	authorSep := strings.Repeat("-", cols.author+1)
	ciSep := strings.Repeat("-", cols.ci+1)
	ageSep := strings.Repeat("-", cols.age+1)
	line := " " + sigSep + "+" + repoSep + "+" + numSep + "+" + titleSep + "+" + authorSep + "+" + ciSep + "+" + ageSep
	return sep.Render(line)
}

type inboxColWidths struct {
	sig, repo, num, title, author, ci, age int
}

func (m *InboxModel) columns() inboxColWidths {
	fixedWidth := 69
	titleWidth := max(20, m.width-fixedWidth)
	return inboxColWidths{
		sig:    5,
		repo:   24,
		num:    4,
		title:  titleWidth,
		author: 12,
		ci:     4,
		age:    5,
	}
}

// PrioritySort sorts inbox PRs by priority:
// review-requested > CI-failing > stale > updated.
func PrioritySort(prs []InboxPR, username string, staleDays int) {
	staleThreshold := time.Now().Add(-time.Duration(staleDays) * 24 * time.Hour)
	for i := 1; i < len(prs); i++ {
		for j := i; j > 0 && prPriority(prs[j], username, staleThreshold) > prPriority(prs[j-1], username, staleThreshold); j-- {
			prs[j], prs[j-1] = prs[j-1], prs[j]
		}
	}
}

func prPriority(pr InboxPR, username string, staleThreshold time.Time) int {
	// Higher = more important.
	if pr.Review.State == domain.ReviewPending && !strings.EqualFold(pr.Author, username) {
		return 4 // review requested
	}
	if pr.CI == domain.CIFail {
		return 3 // CI failing
	}
	if pr.UpdatedAt.Before(staleThreshold) {
		return 2 // stale
	}
	return 1 // normal
}

// FooterSignals returns compact, glyph-only health signals for the status bar.
func (m *InboxModel) FooterSignals() string {
	t := m.styles.Theme
	var failed bool
	for _, status := range m.sources {
		if status.Err != "" {
			failed = true
		}
	}
	warn := lipgloss.NewStyle().Foreground(t.Warning)
	muted := lipgloss.NewStyle().Foreground(t.Muted)
	parts := make([]string, 0, 4)
	if failed {
		parts = append(parts, warn.Render("⚠"))
	}
	if m.rate.SearchRemaining > 0 && m.rate.SearchRemaining <= 6 {
		parts = append(parts, warn.Render("◷"))
	}
	if !m.cachedAt.IsZero() {
		parts = append(parts, muted.Render("◌"))
	}
	if len(m.staged) > 0 {
		parts = append(parts, warn.Render("◆"))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " ")
}

func (m *InboxModel) renderActionCue(pr InboxPR) string {
	cues := m.inboxActionCues(pr)
	if len(cues) == 0 {
		return ""
	}
	t := m.styles.Theme
	branch := lipgloss.NewStyle().Foreground(t.Muted).Faint(true).Render("│ ↳ ")
	line := branch + renderInboxActionCues(cues, t)
	return fitLine(line, m.width)
}

type inboxActionCue struct {
	glyph string
	color lipgloss.Color
}

func (m *InboxModel) inboxActionCues(pr InboxPR) []inboxActionCue {
	cues := make([]inboxActionCue, 0, 5)
	switch {
	case pr.Review.State == domain.ReviewPending && !strings.EqualFold(pr.Author, m.username):
		cues = append(cues, inboxActionCue{glyph: "◆", color: m.styles.Theme.Primary})
	case pr.Review.State == domain.ReviewChangesRequested:
		cues = append(cues, inboxActionCue{glyph: "↩", color: m.styles.Theme.Warning})
	}

	hasActionCue := len(cues) > 0
	if hasActionCue && inboxHasSource(pr, domain.InboxSourceFavorite) {
		cues = append(cues, inboxActionCue{glyph: "★", color: m.styles.Theme.Warning})
	}
	if hasActionCue && inboxHasSource(pr, domain.InboxSourceHome) {
		cues = append(cues, inboxActionCue{glyph: "⌂", color: m.styles.Theme.Secondary})
	}
	if insight, ok := m.insights[inboxItemKey(pr)]; ok {
		if insight.CommitDelta > 0 {
			cues = append(cues, inboxActionCue{glyph: fmt.Sprintf("%d⎇", insight.CommitDelta), color: m.styles.Theme.Secondary})
		}
		if insight.FileDelta > 0 {
			cues = append(cues, inboxActionCue{glyph: fmt.Sprintf("%d▤", insight.FileDelta), color: m.styles.Theme.Info})
		}
		if insight.UnresolvedThreads > 0 {
			cues = append(cues, inboxActionCue{glyph: fmt.Sprintf("%d◌", insight.UnresolvedThreads), color: m.styles.Theme.Warning})
		}
	}
	if pr.LocalState == domain.InboxLocalReady {
		cues = append(cues, inboxActionCue{glyph: "↵", color: m.styles.Theme.Success})
	}
	if len(cues) > 5 {
		cues = cues[:5]
	}
	return cues
}

func renderInboxActionCues(cues []inboxActionCue, t core.Theme) string {
	parts := make([]string, 0, len(cues))
	for _, cue := range cues {
		color := cue.color
		if color == "" {
			color = inboxActionCueColor(cue.glyph, t)
		}
		parts = append(parts, lipgloss.NewStyle().Foreground(color).Faint(true).Render(cue.glyph))
	}
	sep := lipgloss.NewStyle().Foreground(t.Muted).Faint(true).Render(" · ")
	return strings.Join(parts, sep)
}

func inboxActionCueColor(glyph string, t core.Theme) lipgloss.Color {
	switch glyph {
	case "◆":
		return t.Primary
	case "↩":
		return t.Warning
	case "★":
		return t.Warning
	case "⌂":
		return t.Secondary
	case "↵":
		return t.Success
	default:
		return t.Muted
	}
}

func (m *InboxModel) selectedKey() string {
	if m.cursor < 0 || m.cursor >= len(m.filtered) {
		return ""
	}
	return inboxItemKey(m.filtered[m.cursor])
}

func inboxItemKey(pr InboxPR) string {
	return fmt.Sprintf("%s#%d", pr.Repo.String(), pr.Number)
}

func (m *InboxModel) restoreSelection(key string) {
	if key == "" {
		return
	}
	for i, pr := range m.filtered {
		if fmt.Sprintf("%s#%d", pr.Repo.String(), pr.Number) == key {
			m.cursor = i
			m.ensureVisible()
			return
		}
	}
}

func inboxKey(items []domain.InboxItem, idx int) string {
	if idx < 0 || idx >= len(items) {
		return ""
	}
	return fmt.Sprintf("%s#%d", items[idx].Repo.String(), items[idx].Number)
}

func inboxHasSource(pr InboxPR, source domain.InboxSource) bool {
	for _, s := range pr.Sources {
		if s == source {
			return true
		}
	}
	return false
}

func sourceIcons(sources []domain.InboxSource) string {
	seen := make(map[domain.InboxSource]bool, len(sources))
	for _, source := range sources {
		seen[source] = true
	}
	ordered := []struct {
		source domain.InboxSource
		icon   string
	}{
		{domain.InboxSourceAttention, "◆"},
		{domain.InboxSourceAssigned, "◈"},
		{domain.InboxSourceFavorite, "★"},
		{domain.InboxSourceOwned, "●"},
		{domain.InboxSourceHome, "⌂"},
	}
	var b strings.Builder
	for _, entry := range ordered {
		if seen[entry.source] {
			b.WriteString(entry.icon)
		}
	}
	if b.Len() == 0 {
		return "·"
	}
	return b.String()
}

func styledCIIcon(status domain.CIStatus, t core.Theme) string {
	style := lipgloss.NewStyle()
	switch status {
	case domain.CIPass:
		style = style.Foreground(t.Success)
	case domain.CIFail:
		style = style.Foreground(t.Error)
	case domain.CIPending:
		style = style.Foreground(t.Warning)
	default:
		style = style.Foreground(t.Muted)
	}
	return style.Render(ciIcon(status))
}

func truncateDisplay(s string, width int) string {
	if width <= 0 || lipgloss.Width(s) <= width {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes)+"…") > width {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}

func padRight(s string, width int) string {
	if lipgloss.Width(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-lipgloss.Width(s))
}

func fitLine(s string, width int) string {
	if width <= 0 {
		return s
	}
	if lipgloss.Width(s) > width {
		return truncateDisplay(s, width)
	}
	return s + strings.Repeat(" ", width-lipgloss.Width(s))
}

func (m *InboxModel) padView(view string) string {
	if m.width <= 0 || m.height <= 0 {
		return view
	}
	lines := strings.Split(view, "\n")
	for i := range lines {
		lines[i] = fitLine(lines[i], m.width)
	}
	for len(lines) < m.height {
		lines = append(lines, strings.Repeat(" ", m.width))
	}
	if len(lines) > m.height {
		lines = lines[:m.height]
	}
	return strings.Join(lines, "\n")
}
