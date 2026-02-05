package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/indrasvat/vivecaka/internal/tui/core"
)

// PRDetailModel implements the PR detail view with horizontal tabs.
type PRDetailModel struct {
	detail  *domain.PRDetail
	width   int
	height  int
	styles  core.Styles
	keys    core.KeyMap
	tab     DetailTab
	scrollY int
	loading bool
	spinner spinner.Model

	bodyCache    markdownCache
	commentCache map[string]markdownCache
	pendingNum   int

	// Comment state
	commentCollapsed map[int]bool
	commentCursor    int
	pendingCollapseZ bool
}

// DetailTab represents the active tab in detail view.
type DetailTab int

const (
	TabDescription DetailTab = iota
	TabChecks
	TabFiles
	TabComments
)

const numTabs = 4

type markdownCache struct {
	width    int
	source   string
	rendered string
}

// NewPRDetailModel creates a new PR detail view.
func NewPRDetailModel(styles core.Styles, keys core.KeyMap) PRDetailModel {
	return PRDetailModel{
		styles:  styles,
		keys:    keys,
		loading: true,
		spinner: newDetailSpinner(styles),
		tab:     TabDescription,
	}
}

// SetSize updates the view dimensions.
func (m *PRDetailModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetDetail updates the displayed PR detail.
func (m *PRDetailModel) SetDetail(d *domain.PRDetail) {
	m.detail = d
	m.loading = false
	m.scrollY = 0
	m.bodyCache = markdownCache{}
	m.commentCache = make(map[string]markdownCache)
	m.pendingNum = 0
	m.commentCollapsed = make(map[int]bool)
	m.commentCursor = 0
}

// GetPRNumber returns the current PR number (0 if no PR loaded).
func (m *PRDetailModel) GetPRNumber() int {
	if m.detail != nil {
		return m.detail.Number
	}
	return m.pendingNum
}

// StartLoading shows loading state while detail is fetched.
func (m *PRDetailModel) StartLoading(number int) tea.Cmd {
	m.loading = true
	m.pendingNum = number
	m.scrollY = 0
	m.detail = nil
	m.spinner = newDetailSpinner(m.styles)
	return m.spinner.Tick
}

// StopLoading clears loading state without mutating detail.
func (m *PRDetailModel) StopLoading() {
	m.loading = false
	m.pendingNum = 0
}

// Message types.
type (
	PRDetailLoadedMsg struct {
		Detail *domain.PRDetail
		Err    error
	}
	OpenDiffMsg    struct{ Number int }
	StartReviewMsg struct{ Number int }

	ReplyToThreadMsg struct {
		ThreadID string
		Body     string
	}
	ResolveThreadMsg struct {
		ThreadID string
	}
	UnresolveThreadMsg struct {
		ThreadID string
	}
)

// Update handles messages for the detail view.
func (m *PRDetailModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case PRDetailLoadedMsg:
		m.SetDetail(msg.Detail)
		return nil
	case spinner.TickMsg:
		if !m.loading {
			return nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return cmd
	}
	return nil
}

func (m *PRDetailModel) handleKey(msg tea.KeyMsg) tea.Cmd {
	// Handle za sequence for collapse
	if m.pendingCollapseZ && msg.Type == tea.KeyRunes && len(msg.Runes) == 1 {
		m.pendingCollapseZ = false
		if msg.Runes[0] == 'a' && m.tab == TabComments {
			m.toggleCommentCollapse()
			return nil
		}
	}

	// Tab navigation
	switch {
	case key.Matches(msg, m.keys.Tab):
		m.tab = (m.tab + 1) % numTabs
		m.scrollY = 0
		return nil
	case key.Matches(msg, m.keys.ShiftTab):
		m.tab = (m.tab + numTabs - 1) % numTabs
		m.scrollY = 0
		return nil
	case key.Matches(msg, m.keys.Down):
		if m.tab == TabComments && m.detail != nil && len(m.detail.Comments) > 0 {
			if m.commentCursor < len(m.detail.Comments)-1 {
				m.commentCursor++
			}
		} else {
			m.scrollY++
		}
		return nil
	case key.Matches(msg, m.keys.Up):
		if m.tab == TabComments && m.detail != nil && len(m.detail.Comments) > 0 {
			if m.commentCursor > 0 {
				m.commentCursor--
			}
		} else if m.scrollY > 0 {
			m.scrollY--
		}
		return nil
	case key.Matches(msg, m.keys.Enter):
		if m.detail != nil && m.tab == TabFiles {
			return func() tea.Msg { return OpenDiffMsg{Number: m.detail.Number} }
		}
		return nil
	case key.Matches(msg, m.keys.Open):
		if url := m.openURL(); url != "" {
			return func() tea.Msg { return OpenBrowserMsg{URL: url} }
		}
		return nil
	case key.Matches(msg, m.keys.Checkout):
		if m.detail != nil {
			return func() tea.Msg { return CheckoutPRMsg{Number: m.detail.Number} }
		}
		return nil
	}

	// Number keys for direct tab access
	if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 {
		r := msg.Runes[0]
		switch r {
		case '1':
			m.tab = TabDescription
			m.scrollY = 0
			return nil
		case '2':
			m.tab = TabChecks
			m.scrollY = 0
			return nil
		case '3':
			m.tab = TabFiles
			m.scrollY = 0
			return nil
		case '4':
			m.tab = TabComments
			m.scrollY = 0
			return nil
		case 'r':
			if m.detail != nil {
				if m.tab == TabComments && len(m.detail.Comments) > 0 {
					thread := m.detail.Comments[m.commentCursor]
					return func() tea.Msg {
						return ReplyToThreadMsg{ThreadID: thread.ID, Body: ""}
					}
				}
				return func() tea.Msg { return StartReviewMsg{Number: m.detail.Number} }
			}
		case 'd':
			if m.detail != nil {
				return func() tea.Msg { return OpenDiffMsg{Number: m.detail.Number} }
			}
		case 'z':
			if m.tab == TabComments {
				m.pendingCollapseZ = true
			}
		case ' ':
			if m.tab == TabComments {
				m.toggleCommentCollapse()
			}
		case 'x':
			if m.tab == TabComments && m.detail != nil && len(m.detail.Comments) > 0 {
				thread := m.detail.Comments[m.commentCursor]
				if !thread.Resolved {
					return func() tea.Msg { return ResolveThreadMsg{ThreadID: thread.ID} }
				}
			}
		case 'X':
			if m.tab == TabComments && m.detail != nil && len(m.detail.Comments) > 0 {
				thread := m.detail.Comments[m.commentCursor]
				if thread.Resolved {
					return func() tea.Msg { return UnresolveThreadMsg{ThreadID: thread.ID} }
				}
			}
		case 'g':
			m.scrollY = 0
		case 'G':
			// Scroll to bottom - will be clamped in render
			m.scrollY = 9999
		}
	}
	return nil
}

func (m *PRDetailModel) toggleCommentCollapse() {
	if m.detail == nil || len(m.detail.Comments) == 0 {
		return
	}
	if m.commentCollapsed == nil {
		m.commentCollapsed = make(map[int]bool)
	}
	m.commentCollapsed[m.commentCursor] = !m.commentCollapsed[m.commentCursor]
}

func (m *PRDetailModel) openURL() string {
	if m.detail == nil {
		return ""
	}
	if m.tab == TabChecks {
		if url := m.selectedCheckURL(); url != "" {
			return url
		}
	}
	return m.detail.URL
}

func (m *PRDetailModel) selectedCheckURL() string {
	if m.detail == nil || len(m.detail.Checks) == 0 {
		return ""
	}
	idx := m.scrollY
	if idx < 0 {
		idx = 0
	}
	if idx >= len(m.detail.Checks) {
		idx = len(m.detail.Checks) - 1
	}
	if url := m.detail.Checks[idx].URL; url != "" {
		return url
	}
	for _, c := range m.detail.Checks {
		if c.URL != "" {
			return c.URL
		}
	}
	return ""
}

// View renders the PR detail view with tabs.
func (m *PRDetailModel) View() string {
	if m.loading {
		msg := "Loading PR detail..."
		if m.pendingNum > 0 {
			msg = fmt.Sprintf("%s Loading PR #%d...", m.spinner.View(), m.pendingNum)
		}
		return lipgloss.NewStyle().
			Width(m.width).Height(m.height).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(m.styles.Theme.Muted).
			Render(msg)
	}

	if m.detail == nil {
		return lipgloss.NewStyle().
			Width(m.width).Height(m.height).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(m.styles.Theme.Muted).
			Render("No PR detail available")
	}

	// Layout:
	// Line 1: PR header
	// Line 2: Tab bar
	// Line 3+: Content area (scrollable)
	prHeader := m.renderPRHeader()
	tabBar := m.renderTabBar()
	contentHeight := max(1, m.height-2) // -2 for header and tab bar
	content := m.renderTabContent(contentHeight)

	return lipgloss.JoinVertical(lipgloss.Left, prHeader, tabBar, content)
}

// renderPRHeader renders the PR title line.
func (m *PRDetailModel) renderPRHeader() string {
	t := m.styles.Theme
	d := m.detail

	prNumStyle := lipgloss.NewStyle().Foreground(t.Primary).Bold(true)
	titleStyle := lipgloss.NewStyle().Foreground(t.Fg).Bold(true)
	authorStyle := lipgloss.NewStyle().Foreground(t.Muted)
	branchStyle := lipgloss.NewStyle().Foreground(t.Info)
	arrowStyle := lipgloss.NewStyle().Foreground(t.Muted)
	stateStyle := lipgloss.NewStyle().Foreground(t.Success)
	if d.State == domain.PRStateClosed {
		stateStyle = lipgloss.NewStyle().Foreground(t.Error)
	}

	// Truncate title to fit
	maxTitleLen := max(10, m.width-50)
	title := d.Title
	if len(title) > maxTitleLen {
		title = title[:maxTitleLen-3] + "..."
	}

	return fmt.Sprintf("%s  %s  %s %s %s  %s",
		prNumStyle.Render(fmt.Sprintf("#%d", d.Number)),
		titleStyle.Render(title),
		authorStyle.Render(d.Author),
		arrowStyle.Render("→"),
		branchStyle.Render(d.Branch.Base),
		stateStyle.Render(string(d.State)),
	)
}

// renderTabBar renders the horizontal tab bar with counts.
func (m *PRDetailModel) renderTabBar() string {
	t := m.styles.Theme
	d := m.detail

	// Tab styles
	activeStyle := lipgloss.NewStyle().
		Foreground(t.Bg).
		Background(t.Primary).
		Bold(true).
		Padding(0, 1)

	inactiveStyle := lipgloss.NewStyle().
		Foreground(t.Muted).
		Padding(0, 1)

	countStyle := lipgloss.NewStyle().
		Foreground(t.Info)

	// Build tab labels with counts
	tabs := []struct {
		name  string
		count int
		tab   DetailTab
	}{
		{"Description", 0, TabDescription},
		{"Checks", len(d.Checks), TabChecks},
		{"Files", len(d.Files), TabFiles},
		{"Comments", len(d.Comments), TabComments},
	}

	var tabStrings []string
	for _, tab := range tabs {
		label := tab.name
		if tab.count > 0 {
			label = fmt.Sprintf("%s %s", tab.name, countStyle.Render(fmt.Sprintf("(%d)", tab.count)))
		}

		if m.tab == tab.tab {
			tabStrings = append(tabStrings, activeStyle.Render(label))
		} else {
			tabStrings = append(tabStrings, inactiveStyle.Render(label))
		}
	}

	// Join tabs with separator
	separator := lipgloss.NewStyle().Foreground(t.Border).Render("  ")
	tabBar := strings.Join(tabStrings, separator)

	// Add bottom border
	borderStyle := lipgloss.NewStyle().
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(t.Border).
		Width(m.width)

	return borderStyle.Render(tabBar)
}

// renderTabContent renders the content for the active tab.
func (m *PRDetailModel) renderTabContent(height int) string {
	var content string
	switch m.tab {
	case TabDescription:
		content = m.renderDescriptionTab()
	case TabChecks:
		content = m.renderChecksTab()
	case TabFiles:
		content = m.renderFilesTab()
	case TabComments:
		content = m.renderCommentsTab()
	}

	// Apply scrolling
	lines := strings.Split(content, "\n")
	totalLines := len(lines)

	// Clamp scroll
	maxScroll := max(0, totalLines-height)
	if m.scrollY > maxScroll {
		m.scrollY = maxScroll
	}
	if m.scrollY < 0 {
		m.scrollY = 0
	}

	// Get visible lines
	start := m.scrollY
	end := min(start+height, totalLines)
	visibleLines := lines[start:end]

	// Add scroll indicator if needed
	if totalLines > height {
		indicator := fmt.Sprintf(" ↓ %d more", totalLines-end)
		if end >= totalLines {
			indicator = " ↑ scroll up"
		}
		// Pad to fill height, then add indicator on last line
		for len(visibleLines) < height-1 {
			visibleLines = append(visibleLines, "")
		}
		indicatorStyle := lipgloss.NewStyle().Foreground(m.styles.Theme.Muted).Italic(true)
		if len(visibleLines) >= height {
			visibleLines[height-1] = indicatorStyle.Render(indicator)
		} else {
			visibleLines = append(visibleLines, indicatorStyle.Render(indicator))
		}
	}

	// Ensure exact height with full-width padding
	return ensureExactHeight(strings.Join(visibleLines, "\n"), height, m.width)
}

// renderDescriptionTab renders the Description tab content.
func (m *PRDetailModel) renderDescriptionTab() string {
	t := m.styles.Theme
	d := m.detail

	var lines []string

	// Branch info
	labelStyle := lipgloss.NewStyle().Foreground(t.Muted)
	valueStyle := lipgloss.NewStyle().Foreground(t.Fg)
	branchStyle := lipgloss.NewStyle().Foreground(t.Info)

	lines = append(lines, fmt.Sprintf("%s  %s → %s",
		labelStyle.Render("Branch:"),
		branchStyle.Render(d.Branch.Head),
		branchStyle.Render(d.Branch.Base),
	))

	// Labels
	if len(d.Labels) > 0 {
		badgeStyle := lipgloss.NewStyle().Foreground(t.Info)
		var badges []string
		for _, l := range d.Labels {
			badges = append(badges, badgeStyle.Render(l))
		}
		lines = append(lines, fmt.Sprintf("%s  %s",
			labelStyle.Render("Labels:"),
			strings.Join(badges, " "),
		))
	}

	// Reviewers
	if len(d.Reviewers) > 0 {
		var revs []string
		for _, r := range d.Reviewers {
			icon := "●"
			style := lipgloss.NewStyle().Foreground(t.Warning)
			switch r.State {
			case domain.ReviewApproved:
				icon = "✓"
				style = lipgloss.NewStyle().Foreground(t.Success)
			case domain.ReviewChangesRequested:
				icon = "✗"
				style = lipgloss.NewStyle().Foreground(t.Error)
			}
			revs = append(revs, fmt.Sprintf("%s %s", style.Render(icon), r.Login))
		}
		lines = append(lines, fmt.Sprintf("%s  %s",
			labelStyle.Render("Reviewers:"),
			strings.Join(revs, "  "),
		))
	}

	// Dates
	created := formatRelativeTime(d.CreatedAt)
	updated := formatRelativeTime(d.UpdatedAt)
	lines = append(lines, fmt.Sprintf("%s %s  %s %s",
		labelStyle.Render("Created:"),
		valueStyle.Render(created),
		labelStyle.Render("Updated:"),
		valueStyle.Render(updated),
	))

	lines = append(lines, "") // Spacer

	// PR body
	if d.Body == "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(t.Muted).Italic(true).Render("No description provided."))
	} else {
		bodyWidth := max(20, m.width-4)
		rendered := renderMarkdownCached(&m.bodyCache, d.Body, bodyWidth)
		lines = append(lines, strings.Split(rendered, "\n")...)
	}

	return strings.Join(lines, "\n")
}

// renderChecksTab renders the Checks tab content.
func (m *PRDetailModel) renderChecksTab() string {
	t := m.styles.Theme
	d := m.detail

	if len(d.Checks) == 0 {
		return lipgloss.NewStyle().Foreground(t.Muted).Italic(true).Render("No CI checks.")
	}

	var lines []string

	// Summary line
	var pass, fail, pending, skipped int
	for _, c := range d.Checks {
		switch c.Status {
		case domain.CIPass:
			pass++
		case domain.CIFail:
			fail++
		case domain.CIPending:
			pending++
		case domain.CISkipped:
			skipped++
		}
	}

	summaryParts := []string{fmt.Sprintf("%d/%d passing", pass, len(d.Checks))}
	if fail > 0 {
		summaryParts = append(summaryParts, lipgloss.NewStyle().Foreground(t.Error).Render(fmt.Sprintf("%d failing", fail)))
	}
	if pending > 0 {
		summaryParts = append(summaryParts, lipgloss.NewStyle().Foreground(t.Warning).Render(fmt.Sprintf("%d pending", pending)))
	}
	if skipped > 0 {
		summaryParts = append(summaryParts, lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf("%d skipped", skipped)))
	}

	summaryStyle := lipgloss.NewStyle().Bold(true)
	if fail > 0 {
		summaryStyle = summaryStyle.Foreground(t.Error)
	} else {
		summaryStyle = summaryStyle.Foreground(t.Success)
	}
	lines = append(lines, summaryStyle.Render(strings.Join(summaryParts, ", ")))
	lines = append(lines, "") // Spacer

	// Individual checks
	for i, c := range d.Checks {
		icon := detailCIIcon(c.Status)
		cursor := "  "
		if i == m.scrollY {
			cursor = "> "
		}
		dur := ""
		if c.Duration > 0 {
			dur = lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf(" %s", c.Duration.Truncate(time.Second)))
		}
		lines = append(lines, fmt.Sprintf("%s%s %s%s", cursor, icon, c.Name, dur))
	}

	return strings.Join(lines, "\n")
}

// formatCheckSummary returns a plain text summary of check statuses (for testing).
func formatCheckSummary(checks []domain.Check) string {
	var pass, fail, pending, skipped int
	for _, c := range checks {
		switch c.Status {
		case domain.CIPass:
			pass++
		case domain.CIFail:
			fail++
		case domain.CIPending:
			pending++
		case domain.CISkipped:
			skipped++
		}
	}

	parts := []string{fmt.Sprintf("%d/%d passing", pass, len(checks))}
	if fail > 0 {
		parts = append(parts, fmt.Sprintf("%d failing", fail))
	}
	if pending > 0 {
		parts = append(parts, fmt.Sprintf("%d pending", pending))
	}
	if skipped > 0 {
		parts = append(parts, fmt.Sprintf("%d skipped", skipped))
	}

	return strings.Join(parts, ", ")
}

// renderFilesTab renders the Files tab content.
func (m *PRDetailModel) renderFilesTab() string {
	t := m.styles.Theme
	d := m.detail

	if len(d.Files) == 0 {
		return lipgloss.NewStyle().Foreground(t.Muted).Italic(true).Render("No files changed.")
	}

	var lines []string

	// Summary
	var totalAdds, totalDels int
	for _, f := range d.Files {
		totalAdds += f.Additions
		totalDels += f.Deletions
	}
	summaryStyle := lipgloss.NewStyle().Bold(true).Foreground(t.Fg)
	addStyle := lipgloss.NewStyle().Foreground(t.Success)
	delStyle := lipgloss.NewStyle().Foreground(t.Error)
	lines = append(lines, summaryStyle.Render(fmt.Sprintf("%d files changed  %s  %s",
		len(d.Files),
		addStyle.Render(fmt.Sprintf("+%d", totalAdds)),
		delStyle.Render(fmt.Sprintf("-%d", totalDels)),
	)))
	lines = append(lines, "") // Spacer

	// File list
	maxPathLen := max(20, m.width-25)
	for _, f := range d.Files {
		path := f.Path
		if len(path) > maxPathLen {
			path = "..." + path[len(path)-maxPathLen+3:]
		}
		line := fmt.Sprintf("  %s  %s %s",
			path,
			addStyle.Render(fmt.Sprintf("+%d", f.Additions)),
			delStyle.Render(fmt.Sprintf("-%d", f.Deletions)),
		)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// renderCommentsTab renders the Comments tab content.
func (m *PRDetailModel) renderCommentsTab() string {
	t := m.styles.Theme
	d := m.detail

	if len(d.Comments) == 0 {
		return lipgloss.NewStyle().Foreground(t.Muted).Italic(true).Render("No comments.")
	}

	var lines []string
	commentWidth := max(20, m.width-10)

	if m.commentCache == nil {
		m.commentCache = make(map[string]markdownCache)
	}
	if m.commentCollapsed == nil {
		m.commentCollapsed = make(map[int]bool)
	}

	for i, thread := range d.Comments {
		isSelected := i == m.commentCursor
		isCollapsed := m.commentCollapsed[i]

		// Thread header
		marker := "▼"
		if isCollapsed {
			marker = "▶"
		}
		cursor := " "
		if isSelected {
			cursor = ">"
		}

		header := fmt.Sprintf("%s%s %s:%d", cursor, marker, thread.Path, thread.Line)
		if thread.Resolved {
			header += lipgloss.NewStyle().Foreground(t.Success).Render(" [resolved]")
		}
		if isCollapsed && len(thread.Comments) > 0 {
			preview := thread.Comments[0].Body
			preview = strings.ReplaceAll(preview, "\n", " ")
			if len(preview) > 40 {
				preview = preview[:40] + "..."
			}
			header += lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf(" %s", preview))
		}

		headerStyle := lipgloss.NewStyle().Foreground(t.Info).Bold(true)
		if isSelected {
			headerStyle = headerStyle.Background(t.Primary).Foreground(t.Bg)
		}
		lines = append(lines, headerStyle.Render(header))

		// Thread comments (if expanded)
		if !isCollapsed {
			for _, c := range thread.Comments {
				authorLine := fmt.Sprintf("    @%s:", c.Author)
				lines = append(lines, lipgloss.NewStyle().Foreground(t.Muted).Render(authorLine))
				rendered := renderMarkdownCachedMap(m.commentCache, c.ID, c.Body, commentWidth)
				for _, l := range strings.Split(rendered, "\n") {
					lines = append(lines, "      "+l)
				}
			}
		}
		lines = append(lines, "") // Spacer between threads
	}

	return strings.Join(lines, "\n")
}

// Helper functions

func newDetailSpinner(styles core.Styles) spinner.Model {
	sp := spinner.New()
	sp.Spinner = spinner.Line
	sp.Style = lipgloss.NewStyle().Foreground(styles.Theme.Info)
	return sp
}

// Cached glamour renderer - use dracula style for colorful markdown.
// Avoid WithAutoStyle() which does slow terminal detection (~5 seconds).
var glamourRenderer *glamour.TermRenderer

func getGlamourRenderer() *glamour.TermRenderer {
	if glamourRenderer == nil {
		var err error
		glamourRenderer, err = glamour.NewTermRenderer(
			glamour.WithStandardStyle("dracula"),
			glamour.WithWordWrap(100),
		)
		if err != nil {
			return nil
		}
	}
	return glamourRenderer
}

func renderMarkdown(content string, _ int) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}

	renderer := getGlamourRenderer()
	if renderer == nil {
		return simpleMarkdown(content)
	}

	out, err := renderer.Render(content)
	if err != nil {
		return simpleMarkdown(content)
	}
	return strings.TrimRight(out, "\n")
}

func simpleMarkdown(content string) string {
	lines := strings.Split(content, "\n")
	var result []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			result = append(result, "")
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			for strings.HasPrefix(trimmed, "#") {
				trimmed = strings.TrimPrefix(trimmed, "#")
			}
			result = append(result, strings.TrimSpace(trimmed))
			continue
		}
		if strings.HasPrefix(trimmed, "- ") {
			result = append(result, "• "+strings.TrimPrefix(trimmed, "- "))
			continue
		}
		if strings.HasPrefix(trimmed, "* ") {
			result = append(result, "• "+strings.TrimPrefix(trimmed, "* "))
			continue
		}
		result = append(result, trimmed)
	}

	return strings.TrimRight(strings.Join(result, "\n"), "\n")
}

func renderMarkdownCached(cache *markdownCache, content string, width int) string {
	if cache != nil && cache.source == content && cache.width == width && cache.rendered != "" {
		return cache.rendered
	}
	rendered := renderMarkdown(content, width)
	if cache != nil {
		cache.source = content
		cache.width = width
		cache.rendered = rendered
	}
	return rendered
}

func renderMarkdownCachedMap(cache map[string]markdownCache, key, content string, width int) string {
	if cache != nil {
		if entry, ok := cache[key]; ok && entry.source == content && entry.width == width && entry.rendered != "" {
			return entry.rendered
		}
	}
	rendered := renderMarkdown(content, width)
	if cache != nil {
		cache[key] = markdownCache{width: width, source: content, rendered: rendered}
	}
	return rendered
}

func formatRelativeTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return t.Format("Jan 2")
	}
}

func detailCIIcon(status domain.CIStatus) string {
	switch status {
	case domain.CIPass:
		return "✓"
	case domain.CIFail:
		return "✗"
	case domain.CIPending:
		return "●"
	case domain.CISkipped:
		return "—"
	default:
		return "?"
	}
}
