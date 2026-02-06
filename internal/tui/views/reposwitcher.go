package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/indrasvat/vivecaka/internal/tui/core"
)

// RepoSection categorizes a repo entry.
type RepoSection int

const (
	SectionFavorite   RepoSection = iota
	SectionDiscovered             // from gh repo list
	SectionGhost                  // manual add entry
)

// RepoEntry represents a repo in the switcher list.
type RepoEntry struct {
	Repo      domain.RepoRef
	Favorite  bool
	OpenCount int // number of open PRs (-1 means unknown)
	Current   bool
	Section   RepoSection
}

// RepoSwitcherModel implements the repo switcher overlay.
type RepoSwitcherModel struct {
	favorites  []RepoEntry
	discovered []RepoEntry
	visible    []RepoEntry // combined, filtered list (favorites then discovered)
	cursor     int
	scroll     int // scroll offset
	query      string
	width      int
	height     int
	styles     core.Styles
	keys       core.KeyMap

	reposDiscovered bool // true after first gh repo list fetch
	discovering     bool // true while fetching
}

// SetStyles updates the styles without losing state.
func (m *RepoSwitcherModel) SetStyles(s core.Styles) { m.styles = s }

// NewRepoSwitcherModel creates a new repo switcher.
func NewRepoSwitcherModel(styles core.Styles, keys core.KeyMap) RepoSwitcherModel {
	return RepoSwitcherModel{
		styles: styles,
		keys:   keys,
	}
}

// SetSize updates the overlay dimensions.
func (m *RepoSwitcherModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetRepos sets the favorites list. Called on init.
func (m *RepoSwitcherModel) SetRepos(repos []RepoEntry) {
	m.favorites = nil
	for _, r := range repos {
		r.Section = SectionFavorite
		r.Favorite = true
		m.favorites = append(m.favorites, r)
	}
	m.query = ""
	m.cursor = 0
	m.scroll = 0
	m.rebuildVisible()
}

// SetCurrentRepo marks a repo as the current one.
func (m *RepoSwitcherModel) SetCurrentRepo(repo domain.RepoRef) {
	setCurrent := func(entries []RepoEntry) {
		for i := range entries {
			entries[i].Current = entries[i].Repo == repo
		}
	}
	setCurrent(m.favorites)
	setCurrent(m.discovered)
	m.rebuildVisible()
}

// MergeDiscovered adds repos from gh repo list, deduplicating against favorites.
func (m *RepoSwitcherModel) MergeDiscovered(repos []domain.RepoRef) {
	m.reposDiscovered = true
	m.discovering = false

	// Build set of favorites for dedup.
	favSet := make(map[string]bool, len(m.favorites))
	for _, f := range m.favorites {
		favSet[f.Repo.String()] = true
	}

	m.discovered = nil
	for _, r := range repos {
		if favSet[r.String()] {
			continue
		}
		m.discovered = append(m.discovered, RepoEntry{
			Repo:      r,
			Section:   SectionDiscovered,
			OpenCount: -1,
		})
	}
	m.rebuildVisible()
}

// NeedsDiscovery returns true if repos haven't been fetched yet.
func (m *RepoSwitcherModel) NeedsDiscovery() bool {
	return !m.reposDiscovered && !m.discovering
}

// SetDiscovering marks that discovery is in progress.
func (m *RepoSwitcherModel) SetDiscovering() {
	m.discovering = true
}

// IsFavorite returns true if the repo at cursor is a favorite.
func (m *RepoSwitcherModel) IsFavorite() bool {
	if m.cursor >= 0 && m.cursor < len(m.visible) {
		return m.visible[m.cursor].Favorite
	}
	return false
}

// SelectedRepo returns the repo at cursor, if any.
func (m *RepoSwitcherModel) SelectedRepo() (RepoEntry, bool) {
	if m.cursor >= 0 && m.cursor < len(m.visible) {
		return m.visible[m.cursor], true
	}
	return RepoEntry{}, false
}

// Favorites returns the current favorites list.
func (m *RepoSwitcherModel) Favorites() []RepoEntry {
	return m.favorites
}

// SwitchRepoMsg is sent when the user selects a repo.
type SwitchRepoMsg struct {
	Repo domain.RepoRef
}

// CloseRepoSwitcherMsg is sent when the overlay is dismissed.
type CloseRepoSwitcherMsg struct{}

// Update handles messages for the repo switcher.
func (m *RepoSwitcherModel) Update(msg tea.Msg) tea.Cmd {
	typedMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}
	return m.handleKey(typedMsg)
}

func (m *RepoSwitcherModel) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.Type {
	case tea.KeyEscape:
		return func() tea.Msg { return CloseRepoSwitcherMsg{} }

	case tea.KeyEnter:
		if len(m.visible) > 0 && m.cursor < len(m.visible) {
			entry := m.visible[m.cursor]
			if entry.Section == SectionGhost {
				repo := entry.Repo
				return func() tea.Msg {
					return ValidateRepoRequestMsg{Repo: repo}
				}
			}
			return func() tea.Msg { return SwitchRepoMsg{Repo: entry.Repo} }
		}
		return nil

	case tea.KeyBackspace:
		if len(m.query) > 0 {
			m.query = m.query[:len(m.query)-1]
			m.rebuildVisible()
		}
		return nil

	case tea.KeyUp:
		if m.cursor > 0 {
			m.cursor--
			m.ensureVisible()
		}
		return nil

	case tea.KeyDown:
		if m.cursor < len(m.visible)-1 {
			m.cursor++
			m.ensureVisible()
		}
		return nil

	case tea.KeyRunes:
		r := string(msg.Runes)
		// Toggle favorite with 's' only when query is empty.
		if r == "s" && m.query == "" {
			return m.toggleFavorite()
		}
		m.query += r
		m.rebuildVisible()
		return nil
	}
	return nil
}

func (m *RepoSwitcherModel) toggleFavorite() tea.Cmd {
	if m.cursor < 0 || m.cursor >= len(m.visible) {
		return nil
	}
	entry := m.visible[m.cursor]
	if entry.Section == SectionGhost {
		return nil
	}
	newFav := !entry.Favorite
	repo := entry.Repo

	// Update internal state.
	if newFav {
		// Move from discovered to favorites.
		m.removeFromDiscovered(repo)
		entry.Favorite = true
		entry.Section = SectionFavorite
		m.favorites = append(m.favorites, entry)
	} else {
		// Move from favorites to discovered.
		m.removeFromFavorites(repo)
		entry.Favorite = false
		entry.Section = SectionDiscovered
		m.discovered = append(m.discovered, entry)
	}
	m.rebuildVisible()

	return func() tea.Msg {
		return ToggleFavoriteMsg{Repo: repo, Favorite: newFav}
	}
}

func (m *RepoSwitcherModel) removeFromFavorites(repo domain.RepoRef) {
	for i, f := range m.favorites {
		if f.Repo == repo {
			m.favorites = append(m.favorites[:i], m.favorites[i+1:]...)
			return
		}
	}
}

func (m *RepoSwitcherModel) removeFromDiscovered(repo domain.RepoRef) {
	for i, d := range m.discovered {
		if d.Repo == repo {
			m.discovered = append(m.discovered[:i], m.discovered[i+1:]...)
			return
		}
	}
}

func (m *RepoSwitcherModel) rebuildVisible() {
	q := strings.ToLower(m.query)
	m.visible = nil

	// Filter and add favorites.
	for _, r := range m.favorites {
		if q == "" || fuzzyMatch(strings.ToLower(r.Repo.String()), q) {
			m.visible = append(m.visible, r)
		}
	}
	// Filter and add discovered.
	for _, r := range m.discovered {
		if q == "" || fuzzyMatch(strings.ToLower(r.Repo.String()), q) {
			m.visible = append(m.visible, r)
		}
	}

	// Ghost add entry: when query looks like owner/repo and no exact match.
	if strings.Contains(m.query, "/") && len(m.query) >= 3 {
		parts := strings.SplitN(m.query, "/", 2)
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			ghostRepo := domain.RepoRef{Owner: parts[0], Name: parts[1]}
			// Check if it already exists.
			found := false
			for _, v := range m.visible {
				if v.Repo == ghostRepo {
					found = true
					break
				}
			}
			if !found {
				m.visible = append(m.visible, RepoEntry{
					Repo:      ghostRepo,
					Section:   SectionGhost,
					OpenCount: -1,
				})
			}
		}
	}

	if m.cursor >= len(m.visible) {
		m.cursor = max(0, len(m.visible)-1)
	}
	m.ensureVisible()
}

func (m *RepoSwitcherModel) maxVisibleRows() int {
	return max(5, m.height/2-8)
}

func (m *RepoSwitcherModel) ensureVisible() {
	maxRows := m.maxVisibleRows()
	if m.cursor < m.scroll {
		m.scroll = m.cursor
	}
	if m.cursor >= m.scroll+maxRows {
		m.scroll = m.cursor - maxRows + 1
	}
}

// fuzzyMatch checks if all characters in query appear in order within text.
func fuzzyMatch(text, query string) bool {
	qi := 0
	for i := range len(text) {
		if qi < len(query) && text[i] == query[qi] {
			qi++
		}
	}
	return qi == len(query)
}

// View renders the repo switcher overlay.
func (m *RepoSwitcherModel) View() string {
	t := m.styles.Theme

	boxWidth := max(40, min(60, m.width-4))
	innerWidth := boxWidth - 4 // border + padding

	// Styles.
	titleIcon := lipgloss.NewStyle().Foreground(t.Secondary).Render("⟡")
	titleText := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render("Switch Repository")
	searchPrompt := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render("❯")
	searchText := lipgloss.NewStyle().Foreground(t.Fg).Render(m.query)
	cursorChar := lipgloss.NewStyle().Foreground(t.Primary).Render("▎")
	mutedSty := lipgloss.NewStyle().Foreground(t.Muted)
	subtextSty := lipgloss.NewStyle().Foreground(t.Subtext)
	fgBoldSty := lipgloss.NewStyle().Foreground(t.Fg).Bold(true)
	starSty := lipgloss.NewStyle().Foreground(t.Warning)
	currentSty := lipgloss.NewStyle().Foreground(t.Success)
	ghostSty := lipgloss.NewStyle().Foreground(t.Muted).Italic(true)
	infoSty := lipgloss.NewStyle().Foreground(t.Info)

	pad := strings.Repeat(" ", innerWidth)

	// Build content lines.
	var lines []string

	// Title line.
	titleLine := fmt.Sprintf("  %s %s", titleIcon, titleText)
	lines = append(lines, m.padLine(titleLine, innerWidth))
	lines = append(lines, pad)

	// Search line.
	searchLine := fmt.Sprintf("  %s %s%s", searchPrompt, searchText, cursorChar)
	lines = append(lines, m.padLine(searchLine, innerWidth))
	lines = append(lines, pad)

	// Build the list items with section headers.
	type listItem struct {
		entry     *RepoEntry
		isHeader  bool
		headerTxt string
	}
	var items []listItem

	// Count visible favorites and discovered.
	var favItems, discItems []RepoEntry
	for i := range m.visible {
		switch m.visible[i].Section {
		case SectionFavorite:
			favItems = append(favItems, m.visible[i])
		case SectionDiscovered:
			discItems = append(discItems, m.visible[i])
		case SectionGhost:
			// Ghost goes at end.
		}
	}

	if len(favItems) > 0 {
		items = append(items, listItem{isHeader: true, headerTxt: "FAVORITES"})
		for i := range favItems {
			items = append(items, listItem{entry: &favItems[i]})
		}
	}
	if len(discItems) > 0 {
		items = append(items, listItem{isHeader: true, headerTxt: "YOUR REPOS"})
		for i := range discItems {
			items = append(items, listItem{entry: &discItems[i]})
		}
	}
	// Ghost entries.
	for i := range m.visible {
		if m.visible[i].Section == SectionGhost {
			items = append(items, listItem{entry: &m.visible[i]})
		}
	}

	// Map visible cursor index (into m.visible) to list item index.
	cursorInVisible := m.cursor
	visibleIdx := 0
	cursorItemIdx := -1
	for itemIdx, item := range items {
		if item.isHeader {
			continue
		}
		if visibleIdx == cursorInVisible {
			cursorItemIdx = itemIdx
			break
		}
		visibleIdx++
	}

	// Compute scroll window over items.
	maxRows := m.maxVisibleRows()

	// Adjust scroll to keep cursor item visible.
	if cursorItemIdx >= 0 {
		if cursorItemIdx < m.scroll {
			m.scroll = cursorItemIdx
		}
		if cursorItemIdx >= m.scroll+maxRows {
			m.scroll = cursorItemIdx - maxRows + 1
		}
	}
	if m.scroll < 0 {
		m.scroll = 0
	}

	endIdx := min(m.scroll+maxRows, len(items))
	canScrollUp := m.scroll > 0
	canScrollDown := endIdx < len(items)

	// Render visible items.
	visibleCounter := -1
	for idx := m.scroll; idx < endIdx; idx++ {
		item := items[idx]
		if item.isHeader {
			header := m.renderSectionHeader(item.headerTxt, innerWidth, subtextSty, mutedSty)
			lines = append(lines, header)
			continue
		}
		visibleCounter++
		entry := item.entry

		// Determine if this is the selected item.
		isSelected := (idx == cursorItemIdx)

		if entry.Section == SectionGhost {
			ghostLine := ghostSty.Render(fmt.Sprintf("  + add %s", entry.Repo.String()))
			if isSelected {
				line := lipgloss.NewStyle().
					Background(t.BgDim).
					Width(innerWidth).
					Render(ghostLine)
				lines = append(lines, line)
			} else {
				lines = append(lines, m.padLine(ghostLine, innerWidth))
			}
			continue
		}

		// Build repo line: cursor + star + owner/repo + count/current.
		var parts []string
		if isSelected {
			parts = append(parts, lipgloss.NewStyle().Foreground(t.Primary).Render("▸"))
		} else {
			parts = append(parts, " ")
		}

		if entry.Favorite {
			parts = append(parts, " "+starSty.Render("★"))
		} else {
			parts = append(parts, "  ")
		}

		// Owner/repo styled.
		ownerPart := subtextSty.Render(entry.Repo.Owner)
		slashPart := mutedSty.Render(" / ")
		namePart := fgBoldSty.Render(entry.Repo.Name)
		parts = append(parts, " "+ownerPart+slashPart+namePart)

		leftContent := strings.Join(parts, "")

		// Right side: current dot or PR count.
		var rightContent string
		if entry.Current {
			rightContent = currentSty.Render("●")
		} else if entry.OpenCount > 0 {
			rightContent = mutedSty.Render(fmt.Sprintf("%d", entry.OpenCount))
		}

		// Render the line. We need to manually compute widths for alignment.
		if isSelected {
			line := m.renderRepoLine(leftContent, rightContent, innerWidth, &t.BgDim)
			lines = append(lines, line)
		} else {
			line := m.renderRepoLine(leftContent, rightContent, innerWidth, nil)
			lines = append(lines, line)
		}
	}

	// Scroll indicators.
	if canScrollUp {
		scrollUpLine := lipgloss.NewStyle().Foreground(t.Muted).Width(innerWidth).Align(lipgloss.Right).Render("▲")
		lines = append([]string{lines[0], lines[1], lines[2], lines[3], scrollUpLine}, lines[4:]...)
	}
	if canScrollDown {
		scrollDownLine := lipgloss.NewStyle().Foreground(t.Muted).Width(innerWidth).Align(lipgloss.Right).Render("▼")
		lines = append(lines, scrollDownLine)
	}

	// Empty state.
	if len(m.visible) == 0 {
		if m.discovering {
			lines = append(lines, mutedSty.Render("  Loading repos..."))
		} else {
			lines = append(lines, mutedSty.Render("  No repos matching query"))
		}
	}

	lines = append(lines, pad)

	// Key hints.
	sKey := infoSty.Render("s")
	sLabel := mutedSty.Render(" star   ")
	enterKey := infoSty.Render("↵")
	enterLabel := mutedSty.Render(" switch   ")
	escKey := infoSty.Render("Esc")
	escLabel := mutedSty.Render(" close")
	hintLine := "  " + sKey + sLabel + enterKey + enterLabel + escKey + escLabel
	lines = append(lines, m.padLine(hintLine, innerWidth))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Primary).
		Padding(1).
		Width(boxWidth).
		Render(content)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m *RepoSwitcherModel) renderSectionHeader(label string, width int, subtextSty, mutedSty lipgloss.Style) string {
	labelRendered := subtextSty.Render(label)
	lineChar := mutedSty.Render("╶──")
	endChar := mutedSty.Render("──╴")

	// Compute remaining width for right line.
	labelWidth := lipgloss.Width(labelRendered)
	leftLineWidth := lipgloss.Width(lineChar)
	endLineWidth := lipgloss.Width(endChar)
	usedWidth := 2 + leftLineWidth + 1 + labelWidth + 1 + endLineWidth // "  " + line + " " + label + " " + endline

	rightLineLen := width - usedWidth
	var rightLine string
	if rightLineLen > 0 {
		rightLine = mutedSty.Render(strings.Repeat("─", rightLineLen))
	}

	return fmt.Sprintf("  %s %s %s%s", lineChar, labelRendered, rightLine, endChar)
}

func (m *RepoSwitcherModel) renderRepoLine(left, right string, width int, bgColor *lipgloss.Color) string {
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)

	gap := width - leftWidth - rightWidth
	if gap < 1 {
		gap = 1
	}

	line := left + strings.Repeat(" ", gap) + right

	if bgColor != nil {
		return lipgloss.NewStyle().
			Background(*bgColor).
			Width(width).
			Render(line)
	}
	return m.padLine(line, width)
}

func (m *RepoSwitcherModel) padLine(content string, width int) string {
	w := lipgloss.Width(content)
	if w >= width {
		return content
	}
	return content + strings.Repeat(" ", width-w)
}
