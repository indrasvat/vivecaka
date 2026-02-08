package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/indrasvat/vivecaka/internal/tui/core"
)

const (
	filterFieldStatus = iota
	filterFieldAuthor
	filterFieldLabel
	filterFieldCI
	filterFieldReview
	filterFieldDraft
	filterFieldApply
	filterFieldReset
	filterFieldCancel
	filterFieldCount
)

var (
	filterStatusLabels = []string{"Open", "Closed", "Merged"}
	filterStatusValues = []domain.PRState{domain.PRStateOpen, domain.PRStateClosed, domain.PRStateMerged}

	filterCILabels = []string{"All", "Passing", "Failing"}
	filterCIValues = []domain.CIStatus{"", domain.CIPass, domain.CIFail}

	filterReviewLabels = []string{"All", "Approved", "Pending"}
	filterReviewValues = []domain.ReviewState{"", domain.ReviewApproved, domain.ReviewPending}

	filterDraftLabels = []string{"Include", "Exclude", "Only"}
	filterDraftValues = []domain.DraftFilter{domain.DraftInclude, domain.DraftExclude, domain.DraftOnly}
)

// FilterModel implements the filter panel overlay.
type FilterModel struct {
	width  int
	height int
	styles core.Styles
	keys   core.KeyMap

	focus int

	statusIdx int
	author    string

	labelOptions  []string
	labelSelected map[string]bool
	labelCursor   int

	ciIdx     int
	reviewIdx int
	draftIdx  int

	// Preserve pagination settings from original opts
	perPage int
}

// SetStyles updates the styles without losing state.
func (m *FilterModel) SetStyles(s core.Styles) { m.styles = s }

// NewFilterModel creates a new filter panel.
func NewFilterModel(styles core.Styles, keys core.KeyMap) FilterModel {
	m := FilterModel{
		styles:        styles,
		keys:          keys,
		labelOptions:  []string{"enhancement", "bug", "docs"},
		labelSelected: make(map[string]bool),
	}
	m.Reset()
	return m
}

// SetSize updates the overlay dimensions.
func (m *FilterModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetOpts sets the filter state from ListOpts.
func (m *FilterModel) SetOpts(opts domain.ListOpts) {
	m.Reset()
	m.focus = filterFieldStatus

	// Preserve pagination settings
	m.perPage = opts.PerPage

	switch opts.State {
	case domain.PRStateClosed:
		m.statusIdx = 1
	case domain.PRStateMerged:
		m.statusIdx = 2
	default:
		m.statusIdx = 0
	}

	m.author = strings.TrimSpace(opts.Author)

	for _, label := range opts.Labels {
		for _, option := range m.labelOptions {
			if strings.EqualFold(option, label) {
				m.labelSelected[option] = true
			}
		}
	}

	m.ciIdx = indexOfCI(opts.CI)
	m.reviewIdx = indexOfReview(opts.Review)
	m.draftIdx = indexOfDraft(opts.Draft)
}

// Opts returns the current filter selections as ListOpts.
func (m *FilterModel) Opts() domain.ListOpts {
	opts := domain.ListOpts{
		State:   filterStatusValues[m.statusIdx],
		Author:  strings.TrimSpace(m.author),
		CI:      filterCIValues[m.ciIdx],
		Review:  filterReviewValues[m.reviewIdx],
		Draft:   filterDraftValues[m.draftIdx],
		PerPage: m.perPage, // Preserve pagination settings
	}

	for _, label := range m.labelOptions {
		if m.labelSelected[label] {
			opts.Labels = append(opts.Labels, label)
		}
	}

	return opts
}

// Update handles messages for the filter panel.
func (m *FilterModel) Update(msg tea.Msg) tea.Cmd {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}

	if keyMsg.Type == tea.KeyEscape {
		return func() tea.Msg { return CloseFilterMsg{} }
	}

	if keyMsg.Type == tea.KeyTab {
		m.nextField()
		return nil
	}
	if keyMsg.Type == tea.KeyShiftTab {
		m.prevField()
		return nil
	}

	switch keyMsg.Type {
	case tea.KeyUp:
		m.prevField()
		return nil
	case tea.KeyDown:
		m.nextField()
		return nil
	case tea.KeyLeft:
		if m.focus == filterFieldLabel {
			m.moveLabelCursor(-1)
			return nil
		}
	case tea.KeyRight:
		if m.focus == filterFieldLabel {
			m.moveLabelCursor(1)
			return nil
		}
	case tea.KeyEnter:
		switch m.focus {
		case filterFieldApply:
			return func() tea.Msg { return ApplyFilterMsg{Opts: m.Opts()} }
		case filterFieldReset:
			m.Reset()
			return nil
		case filterFieldCancel:
			return func() tea.Msg { return CloseFilterMsg{} }
		default:
			m.toggleFocused()
			return nil
		}
	case tea.KeySpace:
		if m.isTextField() {
			m.author = appendRune(m.author, ' ', 20)
			return nil
		}
		m.toggleFocused()
		return nil
	case tea.KeyBackspace:
		if m.isTextField() {
			m.author = backspace(m.author)
			return nil
		}
	case tea.KeyRunes:
		if len(keyMsg.Runes) != 1 {
			return nil
		}
		r := keyMsg.Runes[0]
		if r == 'r' && !m.isTextField() {
			m.Reset()
			return nil
		}
		if m.focus == filterFieldLabel && (r == 'h' || r == 'l') {
			if r == 'h' {
				m.moveLabelCursor(-1)
			} else {
				m.moveLabelCursor(1)
			}
			return nil
		}
		if m.isTextField() {
			m.author = appendRune(m.author, r, 20)
			return nil
		}
		if r == 'j' {
			m.nextField()
			return nil
		}
		if r == 'k' {
			m.prevField()
			return nil
		}
	}

	return nil
}

// Reset clears all filter selections to defaults.
func (m *FilterModel) Reset() {
	m.statusIdx = 0
	m.author = ""
	for _, label := range m.labelOptions {
		m.labelSelected[label] = false
	}
	m.labelCursor = 0
	m.ciIdx = 0
	m.reviewIdx = 0
	m.draftIdx = 0
}

func (m *FilterModel) nextField() {
	m.focus = (m.focus + 1) % filterFieldCount
}

func (m *FilterModel) prevField() {
	m.focus = (m.focus - 1 + filterFieldCount) % filterFieldCount
}

func (m *FilterModel) moveLabelCursor(delta int) {
	if len(m.labelOptions) == 0 {
		m.labelCursor = 0
		return
	}
	m.labelCursor = (m.labelCursor + delta + len(m.labelOptions)) % len(m.labelOptions)
}

func (m *FilterModel) toggleFocused() {
	switch m.focus {
	case filterFieldStatus:
		m.statusIdx = (m.statusIdx + 1) % len(filterStatusLabels)
	case filterFieldLabel:
		if len(m.labelOptions) > 0 {
			label := m.labelOptions[m.labelCursor]
			m.labelSelected[label] = !m.labelSelected[label]
		}
	case filterFieldCI:
		m.ciIdx = (m.ciIdx + 1) % len(filterCILabels)
	case filterFieldReview:
		m.reviewIdx = (m.reviewIdx + 1) % len(filterReviewLabels)
	case filterFieldDraft:
		m.draftIdx = (m.draftIdx + 1) % len(filterDraftLabels)
	}
}

func (m *FilterModel) isTextField() bool {
	return m.focus == filterFieldAuthor
}

// View renders the filter overlay.
func (m *FilterModel) View() string {
	t := m.styles.Theme
	labelWidth := 9
	boxWidth := max(50, min(70, m.width-4))
	innerWidth := max(10, boxWidth-4)

	titleStyle := lipgloss.NewStyle().Foreground(t.Primary).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(t.Subtext)
	focusedLabelStyle := lipgloss.NewStyle().Foreground(t.Info).Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(t.Muted)
	textStyle := lipgloss.NewStyle().Foreground(t.Fg)
	applyStyle := lipgloss.NewStyle().Foreground(t.Success).Bold(true)
	resetStyle := lipgloss.NewStyle().Foreground(t.Subtext)
	cancelStyle := lipgloss.NewStyle().Foreground(t.Muted)
	keyStyle := lipgloss.NewStyle().Foreground(t.Info)

	divider := strings.Repeat("─", innerWidth)

	label := func(name string, focused bool) string {
		if focused {
			return focusedLabelStyle.Render(fmt.Sprintf("%-*s", labelWidth, name))
		}
		return labelStyle.Render(fmt.Sprintf("%-*s", labelWidth, name))
	}

	checkbox := func(checked bool) string {
		if checked {
			return m.styles.Success.Render("[✓]")
		}
		return mutedStyle.Render("[ ]")
	}

	input := func(value string, focused bool) string {
		width := 8
		if value == "" {
			placeholder := strings.Repeat("_", width)
			if focused && width > 0 {
				placeholder = placeholder[:width-1] + "▎"
			}
			return mutedStyle.Render(placeholder)
		}
		trimmed := value
		if len(trimmed) > width {
			trimmed = trimmed[:width]
		}
		if len(trimmed) < width {
			trimmed += strings.Repeat(" ", width-len(trimmed))
		}
		if focused && len(trimmed) > 0 {
			trimmed = trimmed[:len(trimmed)-1] + "▎"
		}
		return textStyle.Render(trimmed)
	}

	radio := func(labels []string, selected int) string {
		parts := make([]string, 0, len(labels))
		for i, name := range labels {
			box := checkbox(i == selected)
			parts = append(parts, box+" "+textStyle.Render(name))
		}
		row := strings.Join(parts, "  ")
		return row
	}

	labelRow := func() string {
		parts := make([]string, 0, len(m.labelOptions))
		for i, name := range m.labelOptions {
			box := checkbox(m.labelSelected[name])
			text := textStyle.Render(name)
			if m.focus == filterFieldLabel && i == m.labelCursor {
				text = lipgloss.NewStyle().Foreground(t.Info).Bold(true).Render(name)
			}
			parts = append(parts, box+" "+text)
		}
		return strings.Join(parts, "  ")
	}

	statusLine := label("Status:", m.focus == filterFieldStatus) + " " + radio(filterStatusLabels, m.statusIdx)
	authorLine := label("Author:", m.focus == filterFieldAuthor) + " " + input(m.author, m.focus == filterFieldAuthor) + "  " + mutedStyle.Render("(fuzzy)")
	labelLine := label("Label:", m.focus == filterFieldLabel) + " " + labelRow()
	ciLine := label("CI:", m.focus == filterFieldCI) + " " + radio(filterCILabels, m.ciIdx)
	reviewLine := label("Review:", m.focus == filterFieldReview) + " " + radio(filterReviewLabels, m.reviewIdx)
	draftLine := label("Draft:", m.focus == filterFieldDraft) + " " + radio(filterDraftLabels, m.draftIdx)

	quickLine := label("Quick:", false) + " " + keyStyle.Render("[m]") + " " + textStyle.Render("My PRs") + "  " + keyStyle.Render("[n]") + " " + textStyle.Render("Needs My Review")

	apply := applyStyle.Render("[ Apply ]")
	reset := resetStyle.Render("[ Reset ]")
	cancel := cancelStyle.Render("[ Cancel ]")
	if m.focus == filterFieldApply {
		apply = applyStyle.Underline(true).Render("[ Apply ]")
	}
	if m.focus == filterFieldReset {
		reset = resetStyle.Underline(true).Render("[ Reset ]")
	}
	if m.focus == filterFieldCancel {
		cancel = cancelStyle.Underline(true).Render("[ Cancel ]")
	}
	buttons := apply + "  " + reset + "  " + cancel

	lines := []string{
		titleStyle.Render("Filters"),
		divider,
		"",
		statusLine,
		authorLine,
		labelLine,
		ciLine,
		reviewLine,
		draftLine,
		"",
		quickLine,
		"",
		buttons,
	}

	for i, line := range lines {
		lines[i] = lipgloss.NewStyle().Width(innerWidth).Render(line)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border).
		Padding(1).
		Width(boxWidth).
		Render(content)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func indexOfCI(ci domain.CIStatus) int {
	for i, v := range filterCIValues {
		if v == ci {
			return i
		}
	}
	return 0
}

func indexOfReview(review domain.ReviewState) int {
	for i, v := range filterReviewValues {
		if v == review {
			return i
		}
	}
	return 0
}

func indexOfDraft(draft domain.DraftFilter) int {
	for i, v := range filterDraftValues {
		if v == draft {
			return i
		}
	}
	return 0
}

func appendRune(s string, r rune, maxLen int) string {
	if maxLen > 0 && len(s) >= maxLen {
		return s
	}
	return s + string(r)
}

func backspace(s string) string {
	if len(s) == 0 {
		return s
	}
	return s[:len(s)-1]
}
