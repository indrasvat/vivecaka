package views

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/indrasvat/vivecaka/internal/tui/core"
)

// DiffViewModel implements the diff viewer.
type DiffViewModel struct {
	diff          *domain.Diff
	width         int
	height        int
	styles        core.Styles
	keys          core.KeyMap
	fileIdx       int
	scrollY       int
	loading       bool
	searchQuery   string
	searching     bool
	searchMatches []searchMatch
	currentMatch  int
	pendingKey    rune
	collapsed     map[int]bool
}

// NewDiffViewModel creates a new diff viewer.
func NewDiffViewModel(styles core.Styles, keys core.KeyMap) DiffViewModel {
	return DiffViewModel{
		styles:       styles,
		keys:         keys,
		loading:      true,
		currentMatch: -1,
	}
}

// SetSize updates the view dimensions.
func (m *DiffViewModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetDiff updates the displayed diff.
func (m *DiffViewModel) SetDiff(d *domain.Diff) {
	m.diff = d
	m.loading = false
	m.fileIdx = 0
	m.scrollY = 0
	if m.searchQuery != "" {
		m.updateSearchMatches()
	}
}

// Message types.
type DiffLoadedMsg struct {
	Diff *domain.Diff
	Err  error
}

// Update handles messages for the diff view.
func (m *DiffViewModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case DiffLoadedMsg:
		m.SetDiff(msg.Diff)
	}
	return nil
}

func (m *DiffViewModel) handleKey(msg tea.KeyMsg) tea.Cmd {
	if m.searching {
		return m.handleSearchKey(msg)
	}

	if m.pendingKey != 0 && msg.Type != tea.KeyRunes {
		m.pendingKey = 0
	}

	if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 {
		r := msg.Runes[0]
		switch m.pendingKey {
		case 'g':
			m.pendingKey = 0
			if r == 'g' {
				m.scrollToTop()
				return nil
			}
		case 'z':
			m.pendingKey = 0
			if r == 'a' {
				m.toggleCollapse()
				return nil
			}
		}

		switch r {
		case 'n':
			m.nextMatch()
			return nil
		case 'N':
			m.prevMatch()
			return nil
		case 'g':
			m.pendingKey = 'g'
			return nil
		case 'G':
			m.scrollToBottom()
			return nil
		case '[':
			m.jumpToHunk(-1)
			return nil
		case ']':
			m.jumpToHunk(1)
			return nil
		case '{':
			m.prevFile()
			return nil
		case '}':
			m.nextFile()
			return nil
		case 'z':
			m.pendingKey = 'z'
			return nil
		}
	}

	switch {
	case key.Matches(msg, m.keys.Down):
		m.scrollY++
	case key.Matches(msg, m.keys.Up):
		if m.scrollY > 0 {
			m.scrollY--
		}
	case key.Matches(msg, m.keys.HalfPageDown):
		m.scrollY += m.height / 2
	case key.Matches(msg, m.keys.HalfPageUp):
		m.scrollY -= m.height / 2
		if m.scrollY < 0 {
			m.scrollY = 0
		}
	case key.Matches(msg, m.keys.Tab):
		// Next file.
		m.nextFile()
	case key.Matches(msg, m.keys.ShiftTab):
		// Previous file.
		m.prevFile()
	case key.Matches(msg, m.keys.Search):
		m.searching = true
		m.searchQuery = ""
		m.updateSearchMatches()
	}
	return nil
}

func (m *DiffViewModel) handleSearchKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.Type {
	case tea.KeyEscape:
		m.searching = false
		m.searchQuery = ""
		m.searchMatches = nil
		m.currentMatch = -1
	case tea.KeyEnter:
		m.searching = false
		if m.currentMatch >= 0 {
			m.jumpToMatch(m.currentMatch)
		}
	case tea.KeyBackspace:
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
			m.updateSearchMatches()
		}
	case tea.KeyRunes:
		m.searchQuery += string(msg.Runes)
		m.updateSearchMatches()
	}
	return nil
}

// View renders the diff viewer.
func (m *DiffViewModel) View() string {
	if m.loading || m.diff == nil {
		return lipgloss.NewStyle().
			Width(m.width).Height(m.height).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(m.styles.Theme.Muted).
			Render("Loading diff...")
	}

	if len(m.diff.Files) == 0 {
		return lipgloss.NewStyle().
			Width(m.width).Height(m.height).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(m.styles.Theme.Muted).
			Render("No files changed")
	}

	t := m.styles.Theme
	matchStyle := lipgloss.NewStyle().Background(t.Info).Foreground(t.Bg).Bold(true)
	lineMatches := m.matchesByLine(m.fileIdx)

	// File tab bar.
	fileBar := m.renderFileBar()

	// Diff content for current file.
	contentHeight := m.height - 2
	if contentHeight < 1 {
		contentHeight = 1
	}

	file := m.diff.Files[m.fileIdx]
	var lines []string
	lineIdx := 0

	if m.isCollapsed(m.fileIdx) {
		lines = append(lines, m.renderCollapsedFile(file))
	} else {
		for _, hunk := range file.Hunks {
			// Hunk header.
			hunkStyle := lipgloss.NewStyle().Foreground(t.Info)
			header := hunk.Header
			if matches := lineMatches[lineIdx]; len(matches) > 0 {
				header = applyHighlights(header, matches, hunkStyle, matchStyle)
				lines = append(lines, header)
			} else {
				lines = append(lines, hunkStyle.Render(header))
			}
			lineIdx++

			for _, dl := range hunk.Lines {
				matches := lineMatches[lineIdx]
				lineNum := ""
				switch dl.Type {
				case domain.DiffAdd:
					lineNum = fmt.Sprintf("%4s %4d ", "", dl.NewNum)
					line := renderDiffLine(lineNum, "+", dl.Content, m.styles.DiffAdd, matchStyle, matches)
					lines = append(lines, line)
				case domain.DiffDelete:
					lineNum = fmt.Sprintf("%4d %4s ", dl.OldNum, "")
					line := renderDiffLine(lineNum, "-", dl.Content, m.styles.DiffDelete, matchStyle, matches)
					lines = append(lines, line)
				default:
					lineNum = fmt.Sprintf("%4d %4d ", dl.OldNum, dl.NewNum)
					line := renderDiffLine(lineNum, " ", dl.Content, lipgloss.NewStyle().Foreground(t.Fg), matchStyle, matches)
					lines = append(lines, line)
				}
				lineIdx++
			}
		}
	}

	// Apply scroll.
	if m.scrollY >= len(lines) {
		m.scrollY = max(0, len(lines)-1)
	}
	end := m.scrollY + contentHeight
	if end > len(lines) {
		end = len(lines)
	}
	visible := lines[m.scrollY:end]

	content := strings.Join(visible, "\n")

	// Search bar.
	if m.searching {
		search := lipgloss.NewStyle().Foreground(t.Info).Render(m.searchBarText())
		content += "\n" + search
	}

	return lipgloss.JoinVertical(lipgloss.Left, fileBar, content)
}

func (m *DiffViewModel) renderFileBar() string {
	t := m.styles.Theme
	active := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Padding(0, 1)
	inactive := lipgloss.NewStyle().Foreground(t.Muted).Padding(0, 1)

	var tabs []string
	for i, f := range m.diff.Files {
		// Shorten path for display.
		name := f.Path
		if len(name) > 20 {
			name = "…" + name[len(name)-19:]
		}
		if i == m.fileIdx {
			tabs = append(tabs, active.Render(name))
		} else {
			tabs = append(tabs, inactive.Render(name))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
}

type searchMatch struct {
	fileIdx  int
	lineIdx  int
	colStart int
	colEnd   int
}

type matchSpan struct {
	start int
	end   int
}

func (m *DiffViewModel) updateSearchMatches() {
	m.searchMatches = nil
	m.currentMatch = -1

	query := strings.TrimSpace(m.searchQuery)
	if m.diff == nil || query == "" {
		return
	}

	lowerQuery := strings.ToLower(query)
	for fi, file := range m.diff.Files {
		lineIdx := 0
		for _, hunk := range file.Hunks {
			for _, span := range findMatchSpans(hunk.Header, lowerQuery) {
				m.searchMatches = append(m.searchMatches, searchMatch{
					fileIdx: fi, lineIdx: lineIdx, colStart: span.start, colEnd: span.end,
				})
			}
			lineIdx++
			for _, dl := range hunk.Lines {
				for _, span := range findMatchSpans(dl.Content, lowerQuery) {
					m.searchMatches = append(m.searchMatches, searchMatch{
						fileIdx: fi, lineIdx: lineIdx, colStart: span.start, colEnd: span.end,
					})
				}
				lineIdx++
			}
		}
	}

	if len(m.searchMatches) == 0 {
		return
	}

	m.currentMatch = 0
	if idx := m.firstMatchInFile(m.fileIdx); idx >= 0 {
		m.currentMatch = idx
	}
}

func (m *DiffViewModel) firstMatchInFile(fileIdx int) int {
	for i, match := range m.searchMatches {
		if match.fileIdx == fileIdx {
			return i
		}
	}
	return -1
}

func (m *DiffViewModel) nextMatch() {
	if len(m.searchMatches) == 0 {
		return
	}
	next := m.currentMatch + 1
	if next >= len(m.searchMatches) {
		next = 0
	}
	m.jumpToMatch(next)
}

func (m *DiffViewModel) prevMatch() {
	if len(m.searchMatches) == 0 {
		return
	}
	prev := m.currentMatch - 1
	if prev < 0 {
		prev = len(m.searchMatches) - 1
	}
	m.jumpToMatch(prev)
}

func (m *DiffViewModel) jumpToMatch(idx int) {
	if len(m.searchMatches) == 0 {
		return
	}
	if idx < 0 || idx >= len(m.searchMatches) {
		idx = 0
	}

	match := m.searchMatches[idx]
	m.currentMatch = idx
	if match.fileIdx != m.fileIdx {
		m.fileIdx = match.fileIdx
		m.scrollY = 0
	}

	lineCount := m.fileLineCount(m.fileIdx)
	target := match.lineIdx
	if target < 0 {
		target = 0
	}
	if target >= lineCount {
		target = lineCount - 1
	}

	visible := max(1, m.height-2)
	scroll := target - visible/2
	if scroll < 0 {
		scroll = 0
	}
	if scroll > lineCount-visible {
		scroll = max(0, lineCount-visible)
	}
	m.scrollY = scroll
}

func (m *DiffViewModel) fileLineCount(fileIdx int) int {
	if m.diff == nil || fileIdx < 0 || fileIdx >= len(m.diff.Files) {
		return 0
	}
	if m.isCollapsed(fileIdx) {
		return 1
	}
	count := 0
	for _, hunk := range m.diff.Files[fileIdx].Hunks {
		count++ // header
		count += len(hunk.Lines)
	}
	return count
}

func (m *DiffViewModel) matchesByLine(fileIdx int) map[int][]searchMatch {
	lineMatches := make(map[int][]searchMatch)
	for _, match := range m.searchMatches {
		if match.fileIdx != fileIdx {
			continue
		}
		lineMatches[match.lineIdx] = append(lineMatches[match.lineIdx], match)
	}
	for lineIdx, matches := range lineMatches {
		sort.Slice(matches, func(i, j int) bool {
			return matches[i].colStart < matches[j].colStart
		})
		lineMatches[lineIdx] = matches
	}
	return lineMatches
}

func (m *DiffViewModel) searchBarText() string {
	count := len(m.searchMatches)
	current := 0
	if m.currentMatch >= 0 {
		current = m.currentMatch + 1
	}
	if m.searchQuery == "" {
		return "/ ▎"
	}
	return fmt.Sprintf("/ %s [%d/%d]▎", m.searchQuery, current, count)
}

func (m *DiffViewModel) nextFile() {
	if m.diff == nil || m.fileIdx >= len(m.diff.Files)-1 {
		return
	}
	m.fileIdx++
	m.scrollY = 0
}

func (m *DiffViewModel) prevFile() {
	if m.diff == nil || m.fileIdx <= 0 {
		return
	}
	m.fileIdx--
	m.scrollY = 0
}

func (m *DiffViewModel) scrollToTop() {
	m.scrollY = 0
}

func (m *DiffViewModel) scrollToBottom() {
	lineCount := m.fileLineCount(m.fileIdx)
	visible := max(1, m.height-2)
	m.scrollY = max(0, lineCount-visible)
}

func (m *DiffViewModel) jumpToHunk(direction int) {
	if m.diff == nil || m.isCollapsed(m.fileIdx) {
		return
	}
	hunks := m.hunkLineIndexes(m.fileIdx)
	if len(hunks) == 0 {
		return
	}

	current := m.scrollY
	target := -1
	if direction > 0 {
		for _, h := range hunks {
			if h > current {
				target = h
				break
			}
		}
		if target == -1 {
			target = hunks[0]
		}
	} else {
		for i := len(hunks) - 1; i >= 0; i-- {
			if hunks[i] < current {
				target = hunks[i]
				break
			}
		}
		if target == -1 {
			target = hunks[len(hunks)-1]
		}
	}

	if target < 0 {
		return
	}
	m.scrollY = target
}

func (m *DiffViewModel) hunkLineIndexes(fileIdx int) []int {
	if m.diff == nil || fileIdx < 0 || fileIdx >= len(m.diff.Files) {
		return nil
	}
	var indexes []int
	lineIdx := 0
	for _, hunk := range m.diff.Files[fileIdx].Hunks {
		indexes = append(indexes, lineIdx)
		lineIdx++
		lineIdx += len(hunk.Lines)
	}
	return indexes
}

func (m *DiffViewModel) toggleCollapse() {
	if m.collapsed == nil {
		m.collapsed = make(map[int]bool)
	}
	m.collapsed[m.fileIdx] = !m.collapsed[m.fileIdx]
	m.scrollY = 0
}

func (m *DiffViewModel) isCollapsed(fileIdx int) bool {
	if m.collapsed == nil {
		return false
	}
	return m.collapsed[fileIdx]
}

func (m *DiffViewModel) renderCollapsedFile(file domain.FileDiff) string {
	t := m.styles.Theme
	adds, dels := countFileChanges(file)
	addStyle := lipgloss.NewStyle().Foreground(t.Success)
	delStyle := lipgloss.NewStyle().Foreground(t.Error)
	line := fmt.Sprintf("%s  %s %s",
		file.Path,
		addStyle.Render(fmt.Sprintf("+%d", adds)),
		delStyle.Render(fmt.Sprintf("-%d", dels)),
	)
	return lipgloss.NewStyle().Foreground(t.Muted).Render(line)
}

func countFileChanges(file domain.FileDiff) (int, int) {
	var adds, dels int
	for _, hunk := range file.Hunks {
		for _, line := range hunk.Lines {
			switch line.Type {
			case domain.DiffAdd:
				adds++
			case domain.DiffDelete:
				dels++
			}
		}
	}
	return adds, dels
}

func findMatchSpans(text, lowerQuery string) []matchSpan {
	if lowerQuery == "" {
		return nil
	}
	lowerText := strings.ToLower(text)
	var spans []matchSpan
	offset := 0
	for {
		idx := strings.Index(lowerText[offset:], lowerQuery)
		if idx == -1 {
			break
		}
		start := offset + idx
		end := start + len(lowerQuery)
		spans = append(spans, matchSpan{start: start, end: end})
		offset = end
	}
	return spans
}

func renderDiffLine(prefix, marker, content string, baseStyle, matchStyle lipgloss.Style, matches []searchMatch) string {
	prefixRendered := baseStyle.Render(prefix + marker)
	if len(matches) == 0 {
		return prefixRendered + baseStyle.Render(content)
	}
	return prefixRendered + applyHighlights(content, matches, baseStyle, matchStyle)
}

func applyHighlights(text string, matches []searchMatch, baseStyle, matchStyle lipgloss.Style) string {
	if len(matches) == 0 {
		return baseStyle.Render(text)
	}

	var b strings.Builder
	last := 0
	for _, match := range matches {
		if match.colStart < last || match.colStart >= len(text) {
			continue
		}
		end := match.colEnd
		if end > len(text) {
			end = len(text)
		}
		if match.colStart > last {
			b.WriteString(baseStyle.Render(text[last:match.colStart]))
		}
		b.WriteString(matchStyle.Render(text[match.colStart:end]))
		last = end
	}
	if last < len(text) {
		b.WriteString(baseStyle.Render(text[last:]))
	}
	return b.String()
}
