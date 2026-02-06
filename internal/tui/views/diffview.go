package views

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/indrasvat/vivecaka/internal/tui/core"
)

// syntaxHighlighter provides language-aware code highlighting for diff lines.
type syntaxHighlighter struct {
	mu         sync.RWMutex
	lexerCache map[string]chroma.Lexer
	formatter  chroma.Formatter
	style      *chroma.Style
}

// newSyntaxHighlighter creates a highlighter with terminal256 formatter and monokai style.
func newSyntaxHighlighter() *syntaxHighlighter {
	return &syntaxHighlighter{
		lexerCache: make(map[string]chroma.Lexer),
		formatter:  formatters.TTY256,
		style:      styles.Get("monokai"),
	}
}

// getLexer returns a cached lexer for the given filename.
func (h *syntaxHighlighter) getLexer(filename string) chroma.Lexer {
	ext := filepath.Ext(filename)
	if ext == "" {
		ext = filename // for files like Makefile, Dockerfile
	}

	h.mu.RLock()
	if lexer, ok := h.lexerCache[ext]; ok {
		h.mu.RUnlock()
		return lexer
	}
	h.mu.RUnlock()

	h.mu.Lock()
	defer h.mu.Unlock()

	// Double-check after acquiring write lock
	if lexer, ok := h.lexerCache[ext]; ok {
		return lexer
	}

	lexer := lexers.Match(filename)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)
	h.lexerCache[ext] = lexer
	return lexer
}

// highlight applies syntax highlighting to a code line.
// Returns the original content if highlighting fails.
func (h *syntaxHighlighter) highlight(content, filename string) string {
	if content == "" {
		return content
	}

	lexer := h.getLexer(filename)
	if lexer == nil || lexer == lexers.Fallback {
		return content
	}

	iterator, err := lexer.Tokenise(nil, content)
	if err != nil {
		return content
	}

	var buf bytes.Buffer
	if err := h.formatter.Format(&buf, h.style, iterator); err != nil {
		return content
	}

	// Chroma adds trailing newline; strip it
	result := buf.String()
	result = strings.TrimSuffix(result, "\n")
	return result
}

// OpenExternalDiffMsg is sent when the user wants to open an external diff tool.
type OpenExternalDiffMsg struct {
	Number int
}

// DiffViewModel implements the diff viewer.
type DiffViewModel struct {
	diff          *domain.Diff
	prNumber      int
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
	highlighter   *syntaxHighlighter

	// Two-pane layout: file tree on left, content on right.
	treeFocus bool // true when file tree pane has focus
	treeWidth int  // computed tree pane width
}

// NewDiffViewModel creates a new diff viewer.
func NewDiffViewModel(styles core.Styles, keys core.KeyMap) DiffViewModel {
	return DiffViewModel{
		styles:       styles,
		keys:         keys,
		loading:      true,
		currentMatch: -1,
		highlighter:  newSyntaxHighlighter(),
	}
}

// SetSize updates the view dimensions.
func (m *DiffViewModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetPRNumber sets the PR number for external tool launches.
func (m *DiffViewModel) SetPRNumber(n int) { m.prNumber = n }

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

// IsTreeFocus returns whether the file tree pane has focus.
func (m *DiffViewModel) IsTreeFocus() bool { return m.treeFocus }

func (m *DiffViewModel) handleKey(msg tea.KeyMsg) tea.Cmd {
	if m.searching {
		return m.handleSearchKey(msg)
	}

	// Tab toggles focus between file tree and content pane.
	if key.Matches(msg, m.keys.Tab) || key.Matches(msg, m.keys.ShiftTab) {
		m.treeFocus = !m.treeFocus
		return nil
	}

	// When tree pane is focused, handle tree-specific keys.
	if m.treeFocus {
		return m.handleTreeKey(msg)
	}

	return m.handleContentKey(msg)
}

func (m *DiffViewModel) handleTreeKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.Down):
		m.nextFile()
	case key.Matches(msg, m.keys.Up):
		m.prevFile()
	case key.Matches(msg, m.keys.Enter):
		// Select file and switch focus to content.
		m.treeFocus = false
		m.scrollY = 0
	case key.Matches(msg, m.keys.Search):
		m.searching = true
		m.searchQuery = ""
		m.updateSearchMatches()
	}

	// Rune-based keys in tree.
	if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 && msg.Runes[0] == 'e' {
		n := m.prNumber
		return func() tea.Msg { return OpenExternalDiffMsg{Number: n} }
	}
	return nil
}

func (m *DiffViewModel) handleContentKey(msg tea.KeyMsg) tea.Cmd {
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
		case 'e':
			n := m.prNumber
			return func() tea.Msg { return OpenExternalDiffMsg{Number: n} }
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

// computeTreeWidth calculates the file tree pane width.
func (m *DiffViewModel) computeTreeWidth() int {
	w := m.width / 4
	if w < 20 {
		w = 20
	}
	if w > 40 {
		w = 40
	}
	if w >= m.width-20 {
		w = max(10, m.width/3)
	}
	return w
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

	m.treeWidth = m.computeTreeWidth()
	treePane := m.renderFileTree()
	contentPane := m.renderContentPane()

	return lipgloss.JoinHorizontal(lipgloss.Top, treePane, contentPane)
}

// renderFileTree renders the left file tree pane.
func (m *DiffViewModel) renderFileTree() string {
	t := m.styles.Theme
	tw := m.treeWidth - 2 // subtract border

	borderColor := t.Muted
	if m.treeFocus {
		borderColor = t.Primary
	}

	var lines []string
	for i, f := range m.diff.Files {
		adds, dels := countFileChanges(f)
		// Status icon.
		icon := "~"
		if adds > 0 && dels == 0 {
			icon = "+"
		} else if dels > 0 && adds == 0 {
			icon = "-"
		}

		// Shorten path for tree display.
		name := filepath.Base(f.Path)
		dir := filepath.Dir(f.Path)
		if dir != "." {
			maxDir := tw - len(name) - 12
			if maxDir > 0 && len(dir) > maxDir {
				dir = "…" + dir[len(dir)-maxDir+1:]
			}
			name = dir + "/" + name
		}
		if len(name) > tw-8 {
			name = "…" + name[len(name)-tw+9:]
		}

		stat := fmt.Sprintf("+%d -%d", adds, dels)
		// Pad or truncate to fit.
		padding := tw - len(name) - len(stat) - 3 // icon + spaces
		if padding < 1 {
			padding = 1
		}
		line := fmt.Sprintf(" %s %s%s%s", icon, name, strings.Repeat(" ", padding), stat)
		if len(line) > tw {
			line = line[:tw]
		}

		style := lipgloss.NewStyle().Foreground(t.Fg)
		if i == m.fileIdx {
			if m.treeFocus {
				style = style.Background(t.BgDim).Bold(true).Foreground(t.Primary)
			} else {
				style = style.Bold(true).Foreground(t.Primary)
			}
		} else {
			style = style.Foreground(t.Muted)
		}
		lines = append(lines, style.Render(line))
	}

	// Pad to fill height.
	for len(lines) < m.height {
		lines = append(lines, strings.Repeat(" ", tw))
	}
	if len(lines) > m.height {
		lines = lines[:m.height]
	}

	content := strings.Join(lines, "\n")
	return lipgloss.NewStyle().
		Width(m.treeWidth).
		BorderRight(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(borderColor).
		Render(content)
}

// renderContentPane renders the right diff content pane.
func (m *DiffViewModel) renderContentPane() string {
	t := m.styles.Theme
	matchStyle := lipgloss.NewStyle().Background(t.Info).Foreground(t.Bg).Bold(true)
	lineMatches := m.matchesByLine(m.fileIdx)

	contentWidth := m.width - m.treeWidth - 1 // subtract tree + border
	contentHeight := m.height - 1
	contentHeight = max(1, contentHeight)

	// File header line.
	file := m.diff.Files[m.fileIdx]
	fileHeader := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render(file.Path)

	var lines []string
	lineIdx := 0

	if m.isCollapsed(m.fileIdx) {
		lines = append(lines, m.renderCollapsedFile(file))
	} else {
		for _, hunk := range file.Hunks {
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
				highlightedContent := m.highlighter.highlight(dl.Content, file.Path)
				switch dl.Type {
				case domain.DiffAdd:
					lineNum = fmt.Sprintf("%4s %4d ", "", dl.NewNum)
					line := renderDiffLineWithSyntax(lineNum, "+", dl.Content, highlightedContent, m.styles.DiffAdd, matchStyle, matches)
					lines = append(lines, line)
				case domain.DiffDelete:
					lineNum = fmt.Sprintf("%4d %4s ", dl.OldNum, "")
					line := renderDiffLineWithSyntax(lineNum, "-", dl.Content, highlightedContent, m.styles.DiffDelete, matchStyle, matches)
					lines = append(lines, line)
				default:
					lineNum = fmt.Sprintf("%4d %4d ", dl.OldNum, dl.NewNum)
					line := renderDiffLineWithSyntax(lineNum, " ", dl.Content, highlightedContent, lipgloss.NewStyle().Foreground(t.Fg), matchStyle, matches)
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
	end = min(end, len(lines))
	visible := lines[m.scrollY:end]

	content := strings.Join(visible, "\n")

	// Search bar.
	if m.searching {
		search := lipgloss.NewStyle().Foreground(t.Info).Render(m.searchBarText())
		content += "\n" + search
	}

	return lipgloss.NewStyle().
		Width(contentWidth).
		Render(lipgloss.JoinVertical(lipgloss.Left, fileHeader, content))
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

// renderDiffLineWithSyntax renders a diff line with syntax highlighting.
// When search matches exist, falls back to plain styling for correct highlighting.
// Otherwise, uses Chroma syntax colors with a background tint for add/delete lines.
func renderDiffLineWithSyntax(prefix, marker, rawContent, highlightedContent string, baseStyle, matchStyle lipgloss.Style, matches []searchMatch) string {
	// Marker and prefix keep their original style
	prefixRendered := baseStyle.Render(prefix + marker)

	// If there are search matches, use raw content with match highlighting
	// (syntax colors would interfere with search highlight visibility)
	if len(matches) > 0 {
		return prefixRendered + applyHighlights(rawContent, matches, baseStyle, matchStyle)
	}

	// If highlighting produced no change (fallback lexer or error), use base style
	if highlightedContent == rawContent {
		return prefixRendered + baseStyle.Render(rawContent)
	}

	// Use syntax highlighted content
	return prefixRendered + highlightedContent
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
