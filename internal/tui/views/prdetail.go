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

// PRDetailModel implements the PR detail view.
type PRDetailModel struct {
	detail  *domain.PRDetail
	width   int
	height  int
	styles  core.Styles
	keys    core.KeyMap
	pane    DetailPane
	scrollY int
	loading bool
	spinner spinner.Model

	bodyCache    markdownCache
	commentCache map[string]markdownCache
	pendingNum   int

	// Comment pane state
	commentCollapsed map[int]bool // collapsed state per thread index
	commentCursor    int          // selected thread index
	pendingCollapseZ bool         // waiting for 'a' after 'z'
}

// DetailPane represents the active pane in detail view.
type DetailPane int

const (
	PaneInfo DetailPane = iota
	PaneFiles
	PaneChecks
	PaneComments
)

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

	// Comment thread messages
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
		if msg.Runes[0] == 'a' && m.pane == PaneComments {
			m.toggleCommentCollapse()
			return nil
		}
	}

	switch {
	case key.Matches(msg, m.keys.Tab):
		m.pane = (m.pane + 1) % 4
		m.scrollY = 0
	case key.Matches(msg, m.keys.ShiftTab):
		m.pane = (m.pane + 3) % 4
		m.scrollY = 0
	case key.Matches(msg, m.keys.Down):
		if m.pane == PaneComments && m.detail != nil && len(m.detail.Comments) > 0 {
			if m.commentCursor < len(m.detail.Comments)-1 {
				m.commentCursor++
			}
		} else {
			m.scrollY++
		}
	case key.Matches(msg, m.keys.Up):
		if m.pane == PaneComments && m.detail != nil && len(m.detail.Comments) > 0 {
			if m.commentCursor > 0 {
				m.commentCursor--
			}
		} else if m.scrollY > 0 {
			m.scrollY--
		}
	case key.Matches(msg, m.keys.Enter):
		if m.detail != nil {
			num := m.detail.Number
			if m.pane == PaneFiles {
				return func() tea.Msg { return OpenDiffMsg{Number: num} }
			}
		}
	case key.Matches(msg, m.keys.Open):
		if url := m.openURL(); url != "" {
			return func() tea.Msg { return OpenBrowserMsg{URL: url} }
		}
	case key.Matches(msg, m.keys.Checkout):
		if m.detail != nil {
			return func() tea.Msg { return CheckoutPRMsg{Number: m.detail.Number} }
		}
	}

	// Handle single-key commands
	if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 {
		r := msg.Runes[0]
		switch r {
		case 'r':
			if m.detail != nil {
				// In comments pane, 'r' starts a reply to the current thread
				// In other panes, 'r' starts a review
				if m.pane == PaneComments && len(m.detail.Comments) > 0 {
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
			if m.pane == PaneComments {
				m.pendingCollapseZ = true
			}
		case ' ':
			// Space toggles collapse in comments pane
			if m.pane == PaneComments {
				m.toggleCommentCollapse()
			}
		case 'x':
			// Resolve thread
			if m.pane == PaneComments && m.detail != nil && len(m.detail.Comments) > 0 {
				thread := m.detail.Comments[m.commentCursor]
				if !thread.Resolved {
					return func() tea.Msg { return ResolveThreadMsg{ThreadID: thread.ID} }
				}
			}
		case 'X':
			// Unresolve thread
			if m.pane == PaneComments && m.detail != nil && len(m.detail.Comments) > 0 {
				thread := m.detail.Comments[m.commentCursor]
				if thread.Resolved {
					return func() tea.Msg { return UnresolveThreadMsg{ThreadID: thread.ID} }
				}
			}
		}
	}
	return nil
}

// toggleCommentCollapse toggles the collapsed state of the current thread.
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
	if m.pane == PaneChecks {
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

// View renders the PR detail view.
func (m *PRDetailModel) View() string {
	// Show loading state while loading OR if detail hasn't arrived yet
	// Note: Use m.loading for spinner control, m.detail for content check
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

	// Handle case where loading finished but detail is nil (error case)
	if m.detail == nil {
		return lipgloss.NewStyle().
			Width(m.width).Height(m.height).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(m.styles.Theme.Muted).
			Render("No PR detail available")
	}

	var content string

	switch m.pane {
	case PaneComments:
		// Comments view uses full width
		content = m.renderCommentsPane(m.height)
	default:
		// Default view: two-column layout for Info/Checks + Description + Files
		content = m.renderDefaultLayout(m.height)
	}

	// PR title header is integrated into the pane layout now
	// Ensure exact height to prevent overflow
	return ensureExactHeight(content, m.height, m.width)
}

// renderPRHeader renders the PR header line within the detail view.
func (m *PRDetailModel) renderPRHeader() string {
	t := m.styles.Theme
	d := m.detail

	prNumStyle := lipgloss.NewStyle().Foreground(t.Primary).Bold(true)
	titleStyle := lipgloss.NewStyle().Foreground(t.Fg).Bold(true)
	authorStyle := lipgloss.NewStyle().Foreground(t.Muted)
	branchStyle := lipgloss.NewStyle().Foreground(t.Info)
	arrowStyle := lipgloss.NewStyle().Foreground(t.Muted)

	return fmt.Sprintf("%s  %s  %s %s %s",
		prNumStyle.Render(fmt.Sprintf("#%d", d.Number)),
		titleStyle.Render(truncateStr(d.Title, m.width/2)),
		authorStyle.Render(d.Author),
		arrowStyle.Render("→"),
		branchStyle.Render(d.Branch.Base),
	)
}

// renderDefaultLayout renders the two-column layout with Info/Checks, Description, and Files.
func (m *PRDetailModel) renderDefaultLayout(height int) string {
	t := m.styles.Theme
	d := m.detail

	// PR Header line at top
	prHeader := m.renderPRHeader()

	// Calculate pane widths - each column gets half the width minus gap
	// Border takes 2 chars (left + right), so inner width = paneWidth - 2
	totalWidth := max(60, m.width)
	paneWidth := (totalWidth - 1) / 2 // -1 for single space gap between panes
	innerWidth := paneWidth - 2       // account for border

	// Pane styles with fixed height for top row
	topPaneHeight := 7 // Fixed height for Info/Checks panes
	paneStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border).
		Width(paneWidth - 2). // Set inner width (lipgloss Width is content width)
		Height(topPaneHeight)

	fullWidthPaneStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border).
		Width(totalWidth - 2) // Full width minus border

	paneHeaderStyle := lipgloss.NewStyle().
		Foreground(t.Muted).
		Bold(true)

	activePaneHeaderStyle := paneHeaderStyle.
		Foreground(t.Primary)

	// === Info Pane (left) ===
	infoHeader := paneHeaderStyle.Render(" Info")
	if m.pane == PaneInfo {
		infoHeader = activePaneHeaderStyle.Render(" Info")
	}
	infoContent := m.renderInfoContent(innerWidth - 2)
	infoPaneContent := lipgloss.JoinVertical(lipgloss.Left, infoHeader, infoContent)
	infoPane := paneStyle.Render(infoPaneContent)

	// === Checks Pane (right) ===
	checksHeader := paneHeaderStyle.Render(" Checks")
	if m.pane == PaneChecks {
		checksHeader = activePaneHeaderStyle.Render(" Checks")
	}
	checksContent := m.renderChecksContent(innerWidth - 2)
	checksPaneContent := lipgloss.JoinVertical(lipgloss.Left, checksHeader, checksContent)
	checksPane := paneStyle.Render(checksPaneContent)

	// Join Info and Checks horizontally with proper alignment
	topRow := lipgloss.JoinHorizontal(lipgloss.Top, infoPane, " ", checksPane)

	// === Description Pane ===
	descHeader := paneHeaderStyle.Render(" Description")
	descContent := m.renderDescriptionContent(totalWidth - 4)
	descPaneContent := lipgloss.JoinVertical(lipgloss.Left, descHeader, descContent)
	descPane := fullWidthPaneStyle.Render(descPaneContent)

	// === Files Pane ===
	filesHeaderText := " Files Changed"
	if len(d.Files) > 0 {
		var adds, dels int
		for _, f := range d.Files {
			adds += f.Additions
			dels += f.Deletions
		}
		addStyle := lipgloss.NewStyle().Foreground(t.Success)
		delStyle := lipgloss.NewStyle().Foreground(t.Error)
		filesHeaderText = fmt.Sprintf(" Files Changed (%d)  %s %s",
			len(d.Files),
			addStyle.Render(fmt.Sprintf("+%d", adds)),
			delStyle.Render(fmt.Sprintf("-%d", dels)),
		)
	}
	filesHeader := paneHeaderStyle.Render(filesHeaderText)
	if m.pane == PaneFiles {
		filesHeader = activePaneHeaderStyle.Render(filesHeaderText)
	}
	filesContent := m.renderFilesContent(totalWidth - 4)
	filesPaneContent := lipgloss.JoinVertical(lipgloss.Left, filesHeader, filesContent)
	filesPane := fullWidthPaneStyle.Render(filesPaneContent)

	// Join all sections vertically with breathing room between sections
	// Use full-width space lines for proper rendering (not empty strings per CLAUDE.md)
	spacer := strings.Repeat(" ", totalWidth)
	return lipgloss.JoinVertical(lipgloss.Left,
		prHeader,
		spacer, // breathing room after header
		topRow,
		spacer, // breathing room after Info/Checks
		descPane,
		spacer, // breathing room after Description
		filesPane,
	)
}

// renderInfoContent renders the info pane content without borders.
func (m *PRDetailModel) renderInfoContent(width int) string {
	t := m.styles.Theme
	d := m.detail

	labelStyle := lipgloss.NewStyle().Foreground(t.Muted)
	valueStyle := lipgloss.NewStyle().Foreground(t.Fg)
	branchStyle := lipgloss.NewStyle().Foreground(t.Info)
	badgeStyle := lipgloss.NewStyle().Foreground(t.Info)

	var lines []string

	// Branch
	lines = append(lines, fmt.Sprintf("%s  %s %s %s",
		labelStyle.Render("Branch:"),
		branchStyle.Render(truncateStr(d.Branch.Head, width/2)),
		labelStyle.Render("→"),
		branchStyle.Render(d.Branch.Base),
	))

	// Labels
	if len(d.Labels) > 0 {
		var labelBadges []string
		for _, l := range d.Labels {
			labelBadges = append(labelBadges, badgeStyle.Render(l))
		}
		lines = append(lines, fmt.Sprintf("%s  %s",
			labelStyle.Render("Labels:"),
			strings.Join(labelBadges, " "),
		))
	}

	// Assignees
	if len(d.Assignees) > 0 {
		lines = append(lines, fmt.Sprintf("%s  %s",
			labelStyle.Render("Assignees:"),
			valueStyle.Render(strings.Join(d.Assignees, ", ")),
		))
	}

	// Reviewers
	if len(d.Reviewers) > 0 {
		var revs []string
		for _, r := range d.Reviewers {
			icon := "●" // pending
			style := lipgloss.NewStyle().Foreground(t.Warning)
			switch r.State {
			case domain.ReviewApproved:
				icon = "✓"
				style = lipgloss.NewStyle().Foreground(t.Success)
			case domain.ReviewChangesRequested:
				icon = "✗"
				style = lipgloss.NewStyle().Foreground(t.Error)
			}
			revs = append(revs, fmt.Sprintf("%s %s", style.Render(r.Login), style.Render(icon)))
		}
		lines = append(lines, fmt.Sprintf("%s  %s",
			labelStyle.Render("Reviewers:"),
			strings.Join(revs, "  "),
		))
	}

	// Created/Updated times
	created := formatRelativeTime(d.CreatedAt)
	updated := formatRelativeTime(d.UpdatedAt)
	lines = append(lines, fmt.Sprintf("%s %s  %s %s",
		labelStyle.Render("Created:"),
		valueStyle.Render(created),
		labelStyle.Render("Updated:"),
		valueStyle.Render(updated),
	))

	return strings.Join(lines, "\n")
}

// renderChecksContent renders the checks pane content without borders.
func (m *PRDetailModel) renderChecksContent(width int) string {
	t := m.styles.Theme
	d := m.detail

	if len(d.Checks) == 0 {
		return lipgloss.NewStyle().Foreground(t.Muted).Render("No CI checks.")
	}

	var lines []string

	// Individual checks
	maxChecks := 5 // Show at most 5 checks
	for i, c := range d.Checks {
		if i >= maxChecks {
			remaining := len(d.Checks) - maxChecks
			lines = append(lines, lipgloss.NewStyle().Foreground(t.Muted).Render(
				fmt.Sprintf("... %d more checks", remaining),
			))
			break
		}
		icon := ciIcon(c.Status)
		dur := ""
		if c.Duration > 0 {
			dur = lipgloss.NewStyle().Foreground(t.Muted).Render(
				fmt.Sprintf(" %s", c.Duration.Truncate(1e9)),
			)
		}
		checkName := truncateStr(c.Name, width-15)
		lines = append(lines, fmt.Sprintf("%s %s%s", icon, checkName, dur))
	}

	// Summary line
	lines = append(lines, "")
	summary := formatCheckSummary(d.Checks)
	if summary != "" {
		summaryStyle := lipgloss.NewStyle().Foreground(t.Success).Bold(true)
		// Check if any failing
		for _, c := range d.Checks {
			if c.Status == domain.CIFail {
				summaryStyle = lipgloss.NewStyle().Foreground(t.Error).Bold(true)
				break
			}
		}
		lines = append(lines, summaryStyle.Render(summary))
	}

	return strings.Join(lines, "\n")
}

// renderDescriptionContent renders the description pane content.
func (m *PRDetailModel) renderDescriptionContent(width int) string {
	t := m.styles.Theme
	d := m.detail

	if d.Body == "" {
		return lipgloss.NewStyle().Foreground(t.Muted).Render("No description provided.")
	}

	bodyWidth := max(20, width)
	return renderMarkdownCached(&m.bodyCache, d.Body, bodyWidth)
}

// renderFilesContent renders the files pane content.
func (m *PRDetailModel) renderFilesContent(width int) string {
	t := m.styles.Theme
	d := m.detail

	if len(d.Files) == 0 {
		return lipgloss.NewStyle().Foreground(t.Muted).Render("No files changed.")
	}

	var lines []string
	maxFiles := 6 // Show at most 6 files
	for i, f := range d.Files {
		if i >= maxFiles {
			remaining := len(d.Files) - maxFiles
			lines = append(lines, lipgloss.NewStyle().Foreground(t.Muted).Render(
				fmt.Sprintf("... %d more files", remaining),
			))
			break
		}
		addStyle := lipgloss.NewStyle().Foreground(t.Success)
		delStyle := lipgloss.NewStyle().Foreground(t.Error)

		// Truncate path if needed
		pathWidth := max(10, width-20)
		path := truncateStr(f.Path, pathWidth)

		line := fmt.Sprintf("%s  %s %s",
			path,
			addStyle.Render(fmt.Sprintf("+%d", f.Additions)),
			delStyle.Render(fmt.Sprintf("-%d", f.Deletions)),
		)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func (m *PRDetailModel) renderCommentsPane(height int) string {
	t := m.styles.Theme
	d := m.detail

	if len(d.Comments) == 0 {
		return lipgloss.NewStyle().Foreground(t.Muted).Render("No comments.")
	}

	var lines []string
	commentIndent := "      "
	commentWidth := max(20, m.width-len(commentIndent))
	if m.commentCache == nil {
		m.commentCache = make(map[string]markdownCache)
	}
	if m.commentCollapsed == nil {
		m.commentCollapsed = make(map[int]bool)
	}

	for i, thread := range d.Comments {
		isSelected := i == m.commentCursor
		isCollapsed := m.commentCollapsed[i]

		// Build header
		marker := "▼"
		if isCollapsed {
			marker = "▶"
		}
		cursorMark := " "
		if isSelected {
			cursorMark = ">"
		}

		header := fmt.Sprintf("%s%s %s:%d", cursorMark, marker, thread.Path, thread.Line)
		if thread.Resolved {
			header += " [resolved]"
		}
		if isCollapsed && len(thread.Comments) > 0 {
			// Show first comment preview and reply count
			firstBody := thread.Comments[0].Body
			firstBody = strings.ReplaceAll(firstBody, "\n", " ")
			if len(firstBody) > 30 {
				firstBody = firstBody[:30] + "..."
			}
			header += fmt.Sprintf(": %s (%d replies)", firstBody, len(thread.Comments))
		}

		// Style the header
		headerStyle := lipgloss.NewStyle().Foreground(t.Info).Bold(true)
		if isSelected {
			headerStyle = headerStyle.Background(t.Primary).Foreground(t.Bg)
		}
		lines = append(lines, headerStyle.Render(header))

		// If not collapsed, show comment details
		if !isCollapsed {
			for _, c := range thread.Comments {
				authorLine := fmt.Sprintf("    @%s:", c.Author)
				lines = append(lines, lipgloss.NewStyle().Foreground(t.Muted).Render(authorLine))
				rendered := renderMarkdownCachedMap(m.commentCache, c.ID, c.Body, commentWidth)
				lines = append(lines, indentLines(rendered, commentIndent))
			}
		}
		lines = append(lines, "")
	}

	return lipgloss.NewStyle().Width(m.width).Height(height).
		Render(strings.Join(lines, "\n"))
}

func formatCheckSummary(checks []domain.Check) string {
	if len(checks) == 0 {
		return ""
	}

	var pass, fail, pending, skipped, none int
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
		default:
			none++
		}
	}

	total := len(checks)
	parts := []string{fmt.Sprintf("%d/%d passing", pass, total)}
	if fail > 0 {
		parts = append(parts, fmt.Sprintf("%d failing", fail))
	}
	if pending > 0 {
		parts = append(parts, fmt.Sprintf("%d pending", pending))
	}
	if skipped > 0 {
		parts = append(parts, fmt.Sprintf("%d skipped", skipped))
	}
	if none > 0 {
		parts = append(parts, fmt.Sprintf("%d no status", none))
	}
	return strings.Join(parts, ", ")
}

func newDetailSpinner(styles core.Styles) spinner.Model {
	sp := spinner.New()
	sp.Spinner = spinner.Line
	sp.Style = lipgloss.NewStyle().Foreground(styles.Theme.Info)
	return sp
}

func renderMarkdown(content string, width int) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}
	if width < 20 {
		width = 20
	}
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return content
	}
	out, err := renderer.Render(content)
	if err != nil {
		return content
	}
	return strings.TrimRight(out, "\n")
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

func indentLines(text, prefix string) string {
	if text == "" {
		return ""
	}
	lines := strings.Split(text, "\n")
	for i := range lines {
		lines[i] = prefix + lines[i]
	}
	return strings.Join(lines, "\n")
}

// truncateStr truncates a string to maxLen, adding "..." if truncated.
func truncateStr(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// formatRelativeTime formats a time as a relative duration (e.g., "2h ago").
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
