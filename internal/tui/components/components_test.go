package components

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/indrasvat/vivecaka/internal/tui/core"
)

func componentStyles() core.Styles {
	return core.NewStyles(core.ThemeByName("catppuccin-mocha"))
}

func TestHeaderRendersRepoCountsFilterRefreshAndBranch(t *testing.T) {
	h := NewHeader(componentStyles())
	h.SetWidth(120)
	h.SetRepo(domain.RepoRef{Owner: "owner", Name: "repo"})
	h.SetPRCount(12)
	h.SetTotalCount(34)
	h.SetFilter("Needs Review")
	h.SetRefreshCountdown(9, false)
	h.SetBranch("feat/auth")

	view := h.View()
	assert.Contains(t, view, "vivecaka")
	assert.Contains(t, view, "owner/repo")
	assert.Contains(t, view, "12/34 open")
	assert.Contains(t, view, "Needs Review")
	assert.Contains(t, view, "feat/auth")
	assert.Contains(t, view, "9s")

	h.SetTotalCount(0)
	h.SetFilter("")
	h.SetRefreshSecs(0)
	h.SetRefreshCountdown(3, true)
	view = h.View()
	assert.Contains(t, view, "12 open")
	assert.Contains(t, view, "All PRs")
	assert.Contains(t, view, "paused")

	h.SetStyles(componentStyles())
}

func TestStatusBarRendersHintsMessagesAndClears(t *testing.T) {
	s := NewStatusBar(componentStyles())
	s.SetWidth(80)
	s.SetHints([]string{"j/k move", "enter open"})
	view := s.View()
	assert.Contains(t, view, "j/k move")
	assert.Contains(t, view, "enter open")

	s.SetMessage("saved", false)
	assert.Contains(t, s.View(), "saved")
	s.SetMessage("failed", true)
	assert.Contains(t, s.View(), "failed")
	s.ClearMessage()
	assert.NotContains(t, s.View(), "failed")
	s.SetStyles(componentStyles())
}

func TestToastManagerLifecycleAndStyles(t *testing.T) {
	tm := NewToastManager(componentStyles())
	tm.SetWidth(80)
	assert.False(t, tm.HasToasts())
	assert.Empty(t, tm.View())

	for _, level := range []domain.ToastLevel{
		domain.ToastInfo,
		domain.ToastSuccess,
		domain.ToastWarning,
		domain.ToastError,
	} {
		cmd := tm.Add(string(level), level, time.Millisecond)
		assert.NotNil(t, cmd)
		assert.NotEmpty(t, tm.icon(level))
		assert.NotEmpty(t, tm.toastStyle(level).Render("x"))
	}

	assert.True(t, tm.HasToasts())
	view := tm.View()
	assert.Contains(t, view, "info")
	assert.Contains(t, view, "success")
	assert.Contains(t, view, "warning")
	assert.Contains(t, view, "error")

	tm.Update(DismissToastMsg{ID: 1})
	assert.NotContains(t, tm.View(), "success")
	tm.Update("not a toast")
	tm.SetStyles(componentStyles())
}

func TestBannerVisibilityTicksAndExactHeight(t *testing.T) {
	b := NewBanner(componentStyles(), "test")
	b.SetSize(80, 12)
	assert.True(t, b.Visible())
	assert.NotEmpty(t, b.View())

	initial := b.glyphIndex
	cmd := b.Update(BannerGlyphTickMsg{})
	assert.NotNil(t, cmd)
	assert.Equal(t, (initial+1)%len(bannerGlyphs), b.glyphIndex)

	b.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	assert.False(t, b.Visible())
	assert.Empty(t, b.View())
	assert.Nil(t, b.Update(BannerGlyphTickMsg{}))
	assert.Nil(t, b.scheduleGlyphTick())

	b = NewBanner(componentStyles(), "test")
	b.Update(BannerDismissMsg{})
	assert.False(t, b.Visible())
	b.Hide()
	assert.False(t, b.Visible())
	b.SetStyles(componentStyles())

	exact := bannerExactHeight("a\nb\nc", 2, 4)
	assert.Equal(t, 2, len(strings.Split(exact, "\n")))
	padded := bannerExactHeight("a", 3, 4)
	lines := strings.Split(padded, "\n")
	assert.Len(t, lines, 3)
	assert.Equal(t, strings.Repeat(" ", 4), lines[1])
}
