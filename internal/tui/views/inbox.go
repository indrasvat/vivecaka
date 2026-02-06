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
type InboxPR struct {
	domain.PR
	Repo domain.RepoRef
}

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
}

// NewInboxModel creates a new inbox view.
func NewInboxModel(styles core.Styles, keys core.KeyMap) InboxModel {
	return InboxModel{
		styles:  styles,
		keys:    keys,
		loading: true,
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

// Message types.
type (
	InboxPRsLoadedMsg struct{ PRs []InboxPR }
	OpenInboxPRMsg    struct {
		Repo   domain.RepoRef
		Number int
	}
	CloseInboxMsg struct{}
)

// Update handles messages for the inbox.
func (m *InboxModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case InboxPRsLoadedMsg:
		m.SetPRs(msg.PRs)
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
	}
	return nil
}

func (m *InboxModel) applyFilter() {
	switch m.tab {
	case InboxAssigned:
		m.filtered = filterByAssigned(m.allPRs, m.username)
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

func filterByAssigned(prs []InboxPR, username string) []InboxPR {
	if username == "" {
		return nil
	}
	var out []InboxPR
	for _, pr := range prs {
		for _, a := range pr.Labels {
			// In the inbox, we use a simple heuristic: check if username appears
			// as assignee. Since the domain PR doesn't have an Assignees field
			// (only PRDetail does), we check the Author for "assigned to me".
			_ = a
		}
		// Simplified: when PRDetail is loaded, we'd check assignees.
		// For now, include PRs where user is author (placeholder for proper assignee filtering).
		if strings.EqualFold(pr.Author, username) {
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
		if pr.Review.State == domain.ReviewPending && !strings.EqualFold(pr.Author, username) {
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
	rows := m.height - 5 // tabs + header + separator + padding
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
		return lipgloss.NewStyle().
			Width(m.width).Height(m.height).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(m.styles.Theme.Muted).
			Render("Loading inbox...")
	}

	t := m.styles.Theme

	// Title.
	titleStyle := lipgloss.NewStyle().Foreground(t.Primary).Bold(true)
	title := titleStyle.Render("Unified PR Inbox")

	// Tab bar.
	tabs := m.renderTabs()

	if len(m.filtered) == 0 {
		empty := lipgloss.NewStyle().Foreground(t.Muted).
			Render("  No PRs in this tab")
		return lipgloss.JoinVertical(lipgloss.Left, title, tabs, "", empty)
	}

	// Header.
	headerStyle := lipgloss.NewStyle().Foreground(t.Muted).Bold(true)
	header := headerStyle.Render(fmt.Sprintf("  %-25s %-5s %-30s %-10s %-4s %-5s",
		"Repo", "#", "Title", "Author", "CI", "Age"))
	sep := lipgloss.NewStyle().Foreground(t.Border).
		Render(strings.Repeat("─", m.width))

	// Rows.
	visible := m.visibleRows()
	end := min(m.offset+visible, len(m.filtered))

	var rows []string
	for i := m.offset; i < end; i++ {
		rows = append(rows, m.renderRow(i, m.filtered[i]))
	}

	parts := []string{title, tabs, header, sep}
	parts = append(parts, rows...)
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m *InboxModel) renderTabs() string {
	t := m.styles.Theme
	active := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Padding(0, 1)
	inactive := lipgloss.NewStyle().Foreground(t.Muted).Padding(0, 1)

	tabs := []string{"All", "Assigned", "Review Requested", "My PRs"}
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
		prefix = "▸ "
	}

	repoName := pr.Repo.String()
	if len(repoName) > 23 {
		repoName = repoName[:22] + "…"
	}

	title := pr.Title
	if len(title) > 28 {
		title = title[:27] + "…"
	}

	author := pr.Author
	if len(author) > 8 {
		author = author[:7] + "…"
	}

	ci := ciIcon(pr.CI)
	age := relativeTime(pr.UpdatedAt)

	row := fmt.Sprintf("%s%-25s %-5d %-30s %-10s %-4s %-5s",
		prefix, repoName, pr.Number, title, author, ci, age)

	style := lipgloss.NewStyle()
	if selected {
		style = style.Background(t.Border).Foreground(t.Fg)
	} else {
		style = style.Foreground(t.Fg)
	}
	return style.Width(m.width).Render(row)
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
