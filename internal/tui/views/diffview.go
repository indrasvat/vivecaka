package views

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

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

// maxHighlightLines is the threshold above which syntax highlighting is disabled
// for a single file to prevent Chroma tokenization from stalling the UI.
const maxHighlightLines = 5000

// OpenExternalDiffMsg is sent when the user wants to open an external diff tool.
type OpenExternalDiffMsg struct {
	Number int
}

// AddInlineCommentMsg is sent when the user submits an inline comment.
type AddInlineCommentMsg struct {
	Number int
	Input  domain.InlineCommentInput
}

// InlineCommentAddedMsg is sent after a comment is successfully added.
type InlineCommentAddedMsg struct {
	Err error
}

// DiffViewModel implements the diff viewer.
type DiffViewModel struct {
	diff             *domain.Diff
	prNumber         int
	width            int
	height           int
	styles           core.Styles
	keys             core.KeyMap
	fileIdx          int
	scrollY          int
	loading          bool
	searchQuery      string
	searching        bool
	searchMatches    []searchMatch
	currentMatch     int
	pendingKey       rune
	collapsed        map[int]bool
	highlighter      *syntaxHighlighter
	spinnerFrame     int
	fileChangeCounts [][2]int // cached [adds, dels] per file

	// Two-pane layout: file tree on left, content on right.
	treeFocus bool // true when file tree pane has focus
	treeWidth int  // computed tree pane width

	// Side-by-side mode.
	splitMode bool // true for side-by-side, false for unified

	// Inline comments.
	comments    []domain.CommentThread            // all comments for this PR
	commentMap  map[string][]domain.CommentThread // path:line → threads
	editing     bool                              // true when comment editor is open
	editBuffer  string                            // current editor text
	editLine    int                               // line number being commented
	editPath    string                            // file path being commented
	editSide    string                            // "LEFT" or "RIGHT"
	editReplyTo string                            // thread ID if replying
}

// SetStyles updates the styles without losing state.
func (m *DiffViewModel) SetStyles(s core.Styles) { m.styles = s }

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
	// Pre-compute file change counts so renderFileTree doesn't recount every frame.
	if d != nil {
		m.fileChangeCounts = make([][2]int, len(d.Files))
		for i, f := range d.Files {
			adds, dels := countFileChanges(f)
			m.fileChangeCounts[i] = [2]int{adds, dels}
		}
	} else {
		m.fileChangeCounts = nil
	}
	if m.searchQuery != "" {
		m.updateSearchMatches()
	}
}

// SetComments sets the inline comments for display in the diff.
func (m *DiffViewModel) SetComments(threads []domain.CommentThread) {
	m.comments = threads
	m.commentMap = make(map[string][]domain.CommentThread)
	for _, t := range threads {
		key := fmt.Sprintf("%s:%d", t.Path, t.Line)
		m.commentMap[key] = append(m.commentMap[key], t)
	}
}

func (m *DiffViewModel) spinnerTick() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(_ time.Time) tea.Msg {
		return diffSpinnerTickMsg{}
	})
}

// StartLoading resets the diff view to loading state and starts the spinner.
func (m *DiffViewModel) StartLoading() tea.Cmd {
	m.loading = true
	m.diff = nil
	m.fileIdx = 0
	m.scrollY = 0
	m.spinnerFrame = 0
	return m.spinnerTick()
}

// commentsForLine returns comment threads anchored to a specific file and line.
func (m *DiffViewModel) commentsForLine(path string, line int) []domain.CommentThread {
	if m.commentMap == nil {
		return nil
	}
	return m.commentMap[fmt.Sprintf("%s:%d", path, line)]
}

// Message types.
type DiffLoadedMsg struct {
	Diff *domain.Diff
	Err  error
}

// diffSpinnerTickMsg drives the diff loading spinner animation.
type diffSpinnerTickMsg struct{}

// Update handles messages for the diff view.
func (m *DiffViewModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case diffSpinnerTickMsg:
		if m.loading {
			m.spinnerFrame++
			return m.spinnerTick()
		}
		return nil
	case DiffLoadedMsg:
		if msg.Err != nil {
			m.loading = false
			return nil
		}
		m.SetDiff(msg.Diff)
	}
	return nil
}

// IsTreeFocus returns whether the file tree pane has focus.
func (m *DiffViewModel) IsTreeFocus() bool { return m.treeFocus }

// IsSplitMode returns whether the diff is in side-by-side mode.
func (m *DiffViewModel) IsSplitMode() bool { return m.splitMode }

func (m *DiffViewModel) handleKey(msg tea.KeyMsg) tea.Cmd {
	// Comment editor intercepts all keys.
	if m.editing {
		return m.handleEditKey(msg)
	}

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

func (m *DiffViewModel) handleEditKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.Type {
	case tea.KeyEscape:
		m.editing = false
		m.editBuffer = ""
		m.editReplyTo = ""
	case tea.KeyCtrlS:
		// Submit the comment.
		if strings.TrimSpace(m.editBuffer) == "" {
			m.editing = false
			m.editBuffer = ""
			return nil
		}
		input := domain.InlineCommentInput{
			Path:      m.editPath,
			Line:      m.editLine,
			Side:      m.editSide,
			Body:      m.editBuffer,
			InReplyTo: m.editReplyTo,
		}
		n := m.prNumber
		m.editing = false
		m.editBuffer = ""
		m.editReplyTo = ""
		return func() tea.Msg {
			return AddInlineCommentMsg{Number: n, Input: input}
		}
	case tea.KeyEnter:
		m.editBuffer += "\n"
	case tea.KeyBackspace:
		if len(m.editBuffer) > 0 {
			m.editBuffer = m.editBuffer[:len(m.editBuffer)-1]
		}
	case tea.KeyRunes:
		m.editBuffer += string(msg.Runes)
	}
	return nil
}

// currentDiffLine returns the diff line info at the current scroll position.
func (m *DiffViewModel) currentDiffLine() (path string, line int, side string) {
	if m.diff == nil || m.fileIdx >= len(m.diff.Files) {
		return "", 0, ""
	}
	file := m.diff.Files[m.fileIdx]
	pos := m.scrollY
	for _, hunk := range file.Hunks {
		pos-- // hunk header
		if pos < 0 {
			return file.Path, 0, ""
		}
		for _, dl := range hunk.Lines {
			if pos == 0 {
				switch dl.Type {
				case domain.DiffAdd:
					return file.Path, dl.NewNum, "RIGHT"
				case domain.DiffDelete:
					return file.Path, dl.OldNum, "LEFT"
				default:
					return file.Path, dl.NewNum, "RIGHT"
				}
			}
			pos--
		}
	}
	return file.Path, 0, ""
}

// threadAtCurrentLine returns the first comment thread at the current scroll position.
func (m *DiffViewModel) threadAtCurrentLine() *domain.CommentThread {
	path, line, _ := m.currentDiffLine()
	if path == "" || line == 0 {
		return nil
	}
	threads := m.commentsForLine(path, line)
	if len(threads) > 0 {
		return &threads[0]
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
		case 't':
			m.splitMode = !m.splitMode
			m.scrollY = 0
			return nil
		case 'c':
			// Open comment editor at current line.
			path, line, side := m.currentDiffLine()
			if path != "" && line > 0 {
				m.editing = true
				m.editBuffer = ""
				m.editPath = path
				m.editLine = line
				m.editSide = side
				m.editReplyTo = ""
			}
			return nil
		case 'r':
			// Reply to thread at current line.
			thread := m.threadAtCurrentLine()
			if thread != nil {
				m.editing = true
				m.editBuffer = ""
				m.editPath = thread.Path
				m.editLine = thread.Line
				m.editSide = "RIGHT"
				m.editReplyTo = thread.ID
			}
			return nil
		case 'x':
			// Resolve thread at current line.
			thread := m.threadAtCurrentLine()
			if thread != nil {
				threadID := thread.ID
				return func() tea.Msg { return ResolveThreadMsg{ThreadID: threadID} }
			}
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
		frame := spinnerFrames[m.spinnerFrame%len(spinnerFrames)]
		spinner := lipgloss.NewStyle().Foreground(m.styles.Theme.Primary).Render(frame)
		text := lipgloss.NewStyle().Foreground(m.styles.Theme.Muted).Render(" Loading diff...")
		content := spinner + text
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
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
		var adds, dels int
		if i < len(m.fileChangeCounts) {
			adds, dels = m.fileChangeCounts[i][0], m.fileChangeCounts[i][1]
		} else {
			adds, dels = countFileChanges(f)
		}
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
	if m.splitMode {
		return m.renderSplitContent()
	}
	return m.renderUnifiedContent()
}

// renderUnifiedContent renders the standard unified diff view.
// Only syntax-highlights lines in the visible window for performance.
func (m *DiffViewModel) renderUnifiedContent() string {
	t := m.styles.Theme
	matchStyle := lipgloss.NewStyle().Background(t.Info).Foreground(t.Bg).Bold(true)
	lineMatches := m.matchesByLine(m.fileIdx)

	contentWidth := m.width - m.treeWidth - 1
	contentHeight := max(1, m.height-1)

	file := m.diff.Files[m.fileIdx]
	modeLabel := lipgloss.NewStyle().Foreground(t.Muted).Render(" Unified")
	fileHeader := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render(file.Path) + modeLabel

	if m.isCollapsed(m.fileIdx) {
		content := m.renderCollapsedFile(file, m.fileIdx)
		if m.searching {
			content += "\n" + lipgloss.NewStyle().Foreground(t.Info).Render(m.searchBarText())
		}
		return lipgloss.NewStyle().Width(contentWidth).
			Render(lipgloss.JoinVertical(lipgloss.Left, fileHeader, content))
	}

	// Clamp scrollY using diff line count.
	totalLines := m.fileLineCount(m.fileIdx)
	if m.scrollY >= totalLines {
		m.scrollY = max(0, totalLines-1)
	}

	// Disable syntax highlighting for very large files to prevent UI stalls.
	largeFile := totalLines > maxHighlightLines

	// Build only the visible slice: skip lines before scrollY, break after viewport.
	var visible []string
	lineIdx := 0
	visibleCount := 0

	if largeFile && m.scrollY == 0 {
		warnStyle := lipgloss.NewStyle().Foreground(t.Warning).Bold(true)
		visible = append(visible, warnStyle.Render("  ⚠ Large file — syntax highlighting disabled. Press 'e' for external diff."))
		visibleCount++
	}

	for _, hunk := range file.Hunks {
		if visibleCount >= contentHeight {
			break
		}

		// Hunk header.
		if lineIdx >= m.scrollY {
			hunkStyle := lipgloss.NewStyle().Foreground(t.Info)
			header := hunk.Header
			if matches := lineMatches[lineIdx]; len(matches) > 0 {
				header = applyHighlights(header, matches, hunkStyle, matchStyle)
				visible = append(visible, header)
			} else {
				visible = append(visible, hunkStyle.Render(header))
			}
			visibleCount++
		}
		lineIdx++

		for _, dl := range hunk.Lines {
			if visibleCount >= contentHeight {
				break
			}

			if lineIdx >= m.scrollY {
				matches := lineMatches[lineIdx]
				var highlightedContent string
				if largeFile {
					highlightedContent = dl.Content // skip Chroma tokenization
				} else {
					highlightedContent = m.highlighter.highlight(dl.Content, file.Path)
				}
				switch dl.Type {
				case domain.DiffAdd:
					lineNum := fmt.Sprintf("%4s %4d ", "", dl.NewNum)
					visible = append(visible, renderDiffLineWithSyntax(lineNum, "+", dl.Content, highlightedContent, m.styles.DiffAdd, matchStyle, matches))
				case domain.DiffDelete:
					lineNum := fmt.Sprintf("%4d %4s ", dl.OldNum, "")
					visible = append(visible, renderDiffLineWithSyntax(lineNum, "-", dl.Content, highlightedContent, m.styles.DiffDelete, matchStyle, matches))
				default:
					lineNum := fmt.Sprintf("%4d %4d ", dl.OldNum, dl.NewNum)
					visible = append(visible, renderDiffLineWithSyntax(lineNum, " ", dl.Content, highlightedContent, lipgloss.NewStyle().Foreground(t.Fg), matchStyle, matches))
				}
				visibleCount++

				// Render inline comments anchored to this line.
				commentLine := dl.NewNum
				if dl.Type == domain.DiffDelete {
					commentLine = dl.OldNum
				}
				for _, thread := range m.commentsForLine(file.Path, commentLine) {
					for _, cl := range m.renderCommentThread(thread) {
						if visibleCount >= contentHeight {
							break
						}
						visible = append(visible, cl)
						visibleCount++
					}
				}
			}
			lineIdx++
		}
	}

	content := strings.Join(visible, "\n")
	if m.searching {
		content += "\n" + lipgloss.NewStyle().Foreground(t.Info).Render(m.searchBarText())
	}

	// Show comment editor if active.
	if m.editing {
		content += "\n" + m.renderCommentEditor()
	}

	return lipgloss.NewStyle().Width(contentWidth).
		Render(lipgloss.JoinVertical(lipgloss.Left, fileHeader, content))
}

// renderCommentThread renders an inline comment thread as indented lines.
func (m *DiffViewModel) renderCommentThread(thread domain.CommentThread) []string {
	t := m.styles.Theme
	var lines []string

	borderStyle := lipgloss.NewStyle().Foreground(t.Border)
	authorStyle := lipgloss.NewStyle().Foreground(t.Warning).Bold(true)
	bodyStyle := lipgloss.NewStyle().Foreground(t.Muted)
	resolvedStyle := lipgloss.NewStyle().Foreground(t.Success)

	prefix := "    │ "
	topBorder := borderStyle.Render("    ┌─── ")
	if thread.Resolved {
		topBorder += resolvedStyle.Render("✓ resolved")
	}
	lines = append(lines, topBorder)

	for _, c := range thread.Comments {
		header := prefix + authorStyle.Render(c.Author) + bodyStyle.Render(" "+c.CreatedAt.Format("Jan 2"))
		lines = append(lines, header)
		for _, bodyLine := range strings.Split(c.Body, "\n") {
			lines = append(lines, borderStyle.Render(prefix)+bodyStyle.Render(bodyLine))
		}
	}
	lines = append(lines, borderStyle.Render("    └───"))
	return lines
}

// renderCommentEditor renders the inline comment editor.
func (m *DiffViewModel) renderCommentEditor() string {
	t := m.styles.Theme
	border := lipgloss.NewStyle().Foreground(t.Primary)
	hint := lipgloss.NewStyle().Foreground(t.Muted)

	var lines []string
	lines = append(lines, border.Render("    ╔══ Comment on "+m.editPath+fmt.Sprintf(":%d", m.editLine)))
	if m.editReplyTo != "" {
		lines = append(lines, border.Render("    ║ ")+hint.Render("(reply to thread)"))
	}

	// Show buffer content.
	bufLines := strings.Split(m.editBuffer, "\n")
	for _, bl := range bufLines {
		lines = append(lines, border.Render("    ║ ")+bl+"▎")
	}
	if m.editBuffer == "" {
		lines = append(lines, border.Render("    ║ ")+"▎")
	}

	lines = append(lines, border.Render("    ╚══ ")+hint.Render("Ctrl+S submit  Esc cancel"))
	return strings.Join(lines, "\n")
}

// splitRow holds one row of the side-by-side view.
type splitRow struct {
	leftNum   string
	leftText  string
	leftType  domain.DiffLineType
	rightNum  string
	rightText string
	rightType domain.DiffLineType
}

// renderSplitContent renders side-by-side diff columns.
// Only renders rows in the visible window for performance.
func (m *DiffViewModel) renderSplitContent() string {
	t := m.styles.Theme
	contentWidth := m.width - m.treeWidth - 1
	contentHeight := max(1, m.height-1)
	colWidth := (contentWidth - 3) / 2 // 3 for divider " │ "

	file := m.diff.Files[m.fileIdx]
	modeLabel := lipgloss.NewStyle().Foreground(t.Muted).Render(" Split")
	fileHeader := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render(file.Path) + modeLabel

	rows := m.buildSplitRows(file)

	// Clamp scrollY.
	if m.scrollY >= len(rows) {
		m.scrollY = max(0, len(rows)-1)
	}
	end := min(m.scrollY+contentHeight, len(rows))

	// Only render rows in the visible window.
	var visible []string
	divider := lipgloss.NewStyle().Foreground(t.Border).Render(" │ ")
	lineNumWidth := 5

	for _, row := range rows[m.scrollY:end] {
		leftStyle := m.splitLineStyle(row.leftType)
		rightStyle := m.splitLineStyle(row.rightType)

		leftLine := m.renderSplitHalf(row.leftNum, row.leftText, leftStyle, lineNumWidth, colWidth)
		rightLine := m.renderSplitHalf(row.rightNum, row.rightText, rightStyle, lineNumWidth, colWidth)
		visible = append(visible, leftLine+divider+rightLine)
	}

	content := strings.Join(visible, "\n")
	if m.searching {
		content += "\n" + lipgloss.NewStyle().Foreground(t.Info).Render(m.searchBarText())
	}

	return lipgloss.NewStyle().Width(contentWidth).
		Render(lipgloss.JoinVertical(lipgloss.Left, fileHeader, content))
}

func (m *DiffViewModel) splitLineStyle(lineType domain.DiffLineType) lipgloss.Style {
	switch lineType {
	case domain.DiffAdd:
		return m.styles.DiffAdd
	case domain.DiffDelete:
		return m.styles.DiffDelete
	default:
		return lipgloss.NewStyle().Foreground(m.styles.Theme.Fg)
	}
}

func (m *DiffViewModel) renderSplitHalf(num, text string, style lipgloss.Style, numW, colW int) string {
	numStr := fmt.Sprintf("%*s ", numW, num)
	maxText := colW - numW - 2
	if maxText < 0 {
		maxText = 0
	}
	if len(text) > maxText {
		text = text[:maxText]
	}
	return style.Render(numStr + text)
}

func (m *DiffViewModel) buildSplitRows(file domain.FileDiff) []splitRow {
	var rows []splitRow

	for _, hunk := range file.Hunks {
		// Hunk header spans both sides.
		rows = append(rows, splitRow{
			leftText: hunk.Header, leftType: domain.DiffContext,
			rightText: hunk.Header, rightType: domain.DiffContext,
		})

		// Pair up deletions and additions within the hunk.
		var delBuf, addBuf []domain.DiffLine
		flushPairs := func() {
			maxLen := max(len(delBuf), len(addBuf))
			for i := range maxLen {
				var row splitRow
				if i < len(delBuf) {
					row.leftNum = fmt.Sprintf("%d", delBuf[i].OldNum)
					row.leftText = delBuf[i].Content
					row.leftType = domain.DiffDelete
				}
				if i < len(addBuf) {
					row.rightNum = fmt.Sprintf("%d", addBuf[i].NewNum)
					row.rightText = addBuf[i].Content
					row.rightType = domain.DiffAdd
				}
				rows = append(rows, row)
			}
			delBuf = delBuf[:0]
			addBuf = addBuf[:0]
		}

		for _, dl := range hunk.Lines {
			switch dl.Type {
			case domain.DiffDelete:
				delBuf = append(delBuf, dl)
			case domain.DiffAdd:
				addBuf = append(addBuf, dl)
			default:
				// Flush any pending pairs before context.
				flushPairs()
				rows = append(rows, splitRow{
					leftNum: fmt.Sprintf("%d", dl.OldNum), leftText: dl.Content, leftType: domain.DiffContext,
					rightNum: fmt.Sprintf("%d", dl.NewNum), rightText: dl.Content, rightType: domain.DiffContext,
				})
			}
		}
		flushPairs()
	}

	return rows
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

func (m *DiffViewModel) renderCollapsedFile(file domain.FileDiff, fileIdx int) string {
	t := m.styles.Theme
	var adds, dels int
	if fileIdx >= 0 && fileIdx < len(m.fileChangeCounts) {
		adds, dels = m.fileChangeCounts[fileIdx][0], m.fileChangeCounts[fileIdx][1]
	} else {
		adds, dels = countFileChanges(file)
	}
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
