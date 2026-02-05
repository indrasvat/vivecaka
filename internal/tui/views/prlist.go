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
	sortDesc      bool
	currentBranch string
	filter        domain.ListOpts
}

// NewPRListModel creates a new PR list view.
func NewPRListModel(styles core.Styles, keys core.KeyMap) PRListModel {
	return PRListModel{
		styles:    styles,
		keys:      keys,
		loading:   true,
		sortField: "updated",
		sortDesc:  true,
		filter:    domain.ListOpts{State: domain.PRStateOpen},
	}
}

// SetSize updates the view dimensions.
func (m *PRListModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetPRs updates the PR list data.
func (m *PRListModel) SetPRs(prs []domain.PR) {
	m.prs = prs
	m.loading = false
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

// PRListMsg types for communication with parent.
type (
	PRsLoadedMsg struct {
		PRs []domain.PR
		Err error
	}
	OpenPRMsg      struct{ Number int }
	CheckoutPRMsg  struct{ Number int }
	CopyURLMsg     struct{ URL string }
	OpenBrowserMsg struct{ URL string }
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
	}
	return nil
}

func (m *PRListModel) handleKey(msg tea.KeyMsg) tea.Cmd {
	listLen := len(m.filtered)
	if listLen == 0 {
		if key.Matches(msg, m.keys.Search) {
			m.searching = true
			m.searchQuery = ""
		}
		return nil
	}

	switch {
	case key.Matches(msg, m.keys.Down):
		if m.cursor < listLen-1 {
			m.cursor++
			m.ensureVisible()
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
	case key.Matches(msg, m.keys.Enter):
		if pr := m.SelectedPR(); pr != nil {
			return func() tea.Msg { return OpenPRMsg{Number: pr.Number} }
		}
	case key.Matches(msg, m.keys.Checkout):
		if pr := m.SelectedPR(); pr != nil {
			return func() tea.Msg { return CheckoutPRMsg{Number: pr.Number} }
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
	m.applyFilter()
}

func (m *PRListModel) applyFilter() {
	result := make([]domain.PR, 0, len(m.prs))
	query := strings.ToLower(m.searchQuery)

	for _, pr := range m.prs {
		if query != "" {
			titleMatch := strings.Contains(strings.ToLower(pr.Title), query)
			authorMatch := strings.Contains(strings.ToLower(pr.Author), query)
			if !titleMatch && !authorMatch {
				continue
			}
		}
		result = append(result, pr)
	}

	m.filtered = result
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
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

// View renders the PR list.
func (m *PRListModel) View() string {
	if m.loading {
		return lipgloss.NewStyle().
			Width(m.width).
			Height(m.height).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(m.styles.Theme.Muted).
			Render("Loading PRs...")
	}

	if len(m.filtered) == 0 {
		msg := "No pull requests found"
		if m.searchQuery != "" {
			msg = fmt.Sprintf("No PRs matching %q", m.searchQuery)
		}
		return lipgloss.NewStyle().
			Width(m.width).
			Height(m.height).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(m.styles.Theme.Muted).
			Render(msg)
	}

	var rows []string

	// Header row.
	rows = append(rows, m.renderHeaderRow())
	rows = append(rows, m.renderSeparator())

	// PR rows.
	visible := m.visibleRows()
	end := m.offset + visible
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	for i := m.offset; i < end; i++ {
		rows = append(rows, m.renderPRRow(i, m.filtered[i]))
	}

	// Search bar.
	if m.searching {
		rows = append(rows, m.renderSearchBar())
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (m *PRListModel) renderHeaderRow() string {
	t := m.styles.Theme
	header := lipgloss.NewStyle().Foreground(t.Muted).Bold(true)

	cols := m.columns()
	return header.Render(fmt.Sprintf("  %-*s %-*s %-*s %-*s %-*s %-*s",
		cols.num, "#",
		cols.title, "Title",
		cols.author, "Author",
		cols.ci, "CI",
		cols.review, "Review",
		cols.age, "Age",
	))
}

func (m *PRListModel) renderSeparator() string {
	return lipgloss.NewStyle().Foreground(m.styles.Theme.Border).
		Render(strings.Repeat("─", m.width))
}

type colWidths struct {
	num, title, author, ci, review, age int
}

func (m *PRListModel) columns() colWidths {
	fixed := 4 + 10 + 4 + 7 + 5 + 6 // padding between columns
	titleWidth := m.width - fixed
	if titleWidth < 15 {
		titleWidth = 15
	}
	return colWidths{
		num:    4,
		title:  titleWidth,
		author: 10,
		ci:     3,
		review: 7,
		age:    4,
	}
}

func (m *PRListModel) renderPRRow(idx int, pr domain.PR) string {
	t := m.styles.Theme
	cols := m.columns()

	selected := idx == m.cursor
	isBranch := m.currentBranch != "" && pr.Branch.Head == m.currentBranch
	isDraft := pr.Draft

	// Selection indicator.
	prefix := "  "
	if selected {
		prefix = "▸ "
	} else if isBranch {
		prefix = "◉ "
	}

	// Title with draft prefix.
	title := pr.Title
	if isDraft {
		title = "[DRAFT] " + title
	}
	if len(title) > cols.title {
		title = title[:cols.title-1] + "…"
	}

	// Author.
	author := pr.Author
	if len(author) > cols.author {
		author = author[:cols.author-1] + "…"
	}

	// CI icon.
	ci := ciIcon(pr.CI)

	// Review.
	review := reviewText(pr.Review)

	// Age.
	age := relativeTime(pr.UpdatedAt)

	row := fmt.Sprintf("%s%-*d %-*s %-*s %-*s %-*s %-*s",
		prefix,
		cols.num, pr.Number,
		cols.title, title,
		cols.author, author,
		cols.ci, ci,
		cols.review, review,
		cols.age, age,
	)

	style := lipgloss.NewStyle()
	switch {
	case selected:
		style = style.Background(t.Border).Foreground(t.Fg)
	case isDraft:
		style = style.Foreground(t.Muted)
	case isBranch:
		style = style.Foreground(t.Primary)
	default:
		style = style.Foreground(t.Fg)
	}

	return style.Width(m.width).Render(row)
}

func (m *PRListModel) renderSearchBar() string {
	t := m.styles.Theme
	style := lipgloss.NewStyle().Foreground(t.Info)
	return style.Render(fmt.Sprintf("/ %s▎", m.searchQuery))
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
