package views

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/indrasvat/vivecaka/internal/tui/core"
	"github.com/indrasvat/vivecaka/internal/usecase"
)

func testCheckoutDialog() CheckoutDialogModel {
	styles := core.NewStyles(core.ThemeByName("catppuccin-mocha"))
	keys := core.DefaultKeyMap()
	m := NewCheckoutDialogModel(styles, keys)
	m.SetSize(80, 24)
	return m
}

func testRepo() domain.RepoRef {
	return domain.RepoRef{Owner: "steipete", Name: "CodexBar"}
}

func TestCheckoutDialogInactiveByDefault(t *testing.T) {
	m := testCheckoutDialog()
	assert.False(t, m.Active())
	assert.Empty(t, m.View())
}

func TestCheckoutDialogWorktreeChoice(t *testing.T) {
	m := testCheckoutDialog()
	plan := usecase.CheckoutPlan{Strategy: usecase.StrategyLocal, TargetPath: "/path"}
	m.ShowWorktreeChoice(testRepo(), 255, "feat/oauth", plan)

	assert.True(t, m.Active())
	view := m.View()
	assert.Contains(t, view, "Checkout PR #255")
	assert.Contains(t, view, "feat/oauth")
	assert.Contains(t, view, "Switch branch")
	assert.Contains(t, view, "New worktree")
}

func TestCheckoutDialogWorktreeChoiceNavigation(t *testing.T) {
	m := testCheckoutDialog()
	plan := usecase.CheckoutPlan{Strategy: usecase.StrategyLocal}
	m.ShowWorktreeChoice(testRepo(), 255, "feat/oauth", plan)

	assert.Equal(t, 0, m.cursor)

	// Navigate down.
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, m.cursor)

	// Can't go past last.
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, m.cursor)

	// Navigate up.
	m.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 0, m.cursor)
}

func TestCheckoutDialogWorktreeChoiceSwitch(t *testing.T) {
	m := testCheckoutDialog()
	plan := usecase.CheckoutPlan{Strategy: usecase.StrategyLocal}
	m.ShowWorktreeChoice(testRepo(), 255, "feat/oauth", plan)

	// Cursor on "Switch branch" (index 0), press Enter.
	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	msg := cmd()
	chosen, ok := msg.(CheckoutStrategyChosenMsg)
	require.True(t, ok)
	assert.Equal(t, "switch", chosen.Strategy)
	assert.Equal(t, 255, chosen.PRNumber)
}

func TestCheckoutDialogWorktreeChoiceWorktree(t *testing.T) {
	m := testCheckoutDialog()
	plan := usecase.CheckoutPlan{Strategy: usecase.StrategyLocal}
	m.ShowWorktreeChoice(testRepo(), 255, "feat/oauth", plan)

	// Navigate to "New worktree".
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	msg := cmd()
	chosen, ok := msg.(CheckoutStrategyChosenMsg)
	require.True(t, ok)
	assert.Equal(t, "worktree", chosen.Strategy)
}

func TestCheckoutDialogWorktreeChoiceEsc(t *testing.T) {
	m := testCheckoutDialog()
	plan := usecase.CheckoutPlan{Strategy: usecase.StrategyLocal}
	m.ShowWorktreeChoice(testRepo(), 255, "feat/oauth", plan)

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(CheckoutDialogCloseMsg)
	assert.True(t, ok)
	assert.False(t, m.Active())
}

func TestCheckoutDialogKnownConfirm(t *testing.T) {
	m := testCheckoutDialog()
	plan := usecase.CheckoutPlan{
		Strategy:   usecase.StrategyKnownPath,
		TargetPath: "/Users/test/code/steipete/CodexBar",
	}
	m.ShowKnownConfirm(testRepo(), 255, "feat/oauth", plan)

	assert.True(t, m.Active())
	view := m.View()
	assert.Contains(t, view, "Checkout PR #255")
	assert.Contains(t, view, "Will check out in")
}

func TestCheckoutDialogKnownConfirmEnter(t *testing.T) {
	m := testCheckoutDialog()
	plan := usecase.CheckoutPlan{
		Strategy:   usecase.StrategyKnownPath,
		TargetPath: "/some/path",
	}
	m.ShowKnownConfirm(testRepo(), 255, "feat/oauth", plan)

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	msg := cmd()
	chosen, ok := msg.(CheckoutStrategyChosenMsg)
	require.True(t, ok)
	assert.Equal(t, "known-path", chosen.Strategy)
	assert.Equal(t, "/some/path", chosen.Path)
}

func TestCheckoutDialogKnownConfirmEsc(t *testing.T) {
	m := testCheckoutDialog()
	plan := usecase.CheckoutPlan{Strategy: usecase.StrategyKnownPath, TargetPath: "/path"}
	m.ShowKnownConfirm(testRepo(), 255, "feat/oauth", plan)

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	msg := cmd()
	_, ok := msg.(CheckoutDialogCloseMsg)
	assert.True(t, ok)
}

func TestCheckoutDialogOptions(t *testing.T) {
	m := testCheckoutDialog()
	plan := usecase.CheckoutPlan{
		Strategy:       usecase.StrategyNeedsClone,
		CacheClonePath: "/cache/clones/steipete/CodexBar",
	}
	m.ShowOptions(testRepo(), 255, "feat/oauth", plan)

	view := m.View()
	assert.Contains(t, view, "No local clone found")
	assert.Contains(t, view, "Clone to vivecaka cache")
	assert.Contains(t, view, "Clone to custom path")
	assert.Contains(t, view, "Open on GitHub")
}

func TestCheckoutDialogOptionsCloneCache(t *testing.T) {
	m := testCheckoutDialog()
	plan := usecase.CheckoutPlan{
		Strategy:       usecase.StrategyNeedsClone,
		CacheClonePath: "/cache/path",
	}
	m.ShowOptions(testRepo(), 255, "feat/oauth", plan)

	// Cursor on "Clone to vivecaka cache" (index 0).
	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	msg := cmd()
	chosen, ok := msg.(CheckoutStrategyChosenMsg)
	require.True(t, ok)
	assert.Equal(t, "clone-cache", chosen.Strategy)
	assert.Equal(t, "/cache/path", chosen.Path)
}

func TestCheckoutDialogOptionsCustomPath(t *testing.T) {
	m := testCheckoutDialog()
	plan := usecase.CheckoutPlan{Strategy: usecase.StrategyNeedsClone, CacheClonePath: "/cache"}
	m.ShowOptions(testRepo(), 255, "feat/oauth", plan)

	// Navigate to "Clone to custom path..." (index 1).
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Should transition to custom path input.
	assert.Equal(t, checkoutCustomPath, m.state)
	view := m.View()
	assert.Contains(t, view, "Clone steipete/CodexBar")
	assert.Contains(t, view, "Enter path")
}

func TestCheckoutDialogOptionsBrowser(t *testing.T) {
	m := testCheckoutDialog()
	plan := usecase.CheckoutPlan{Strategy: usecase.StrategyNeedsClone, CacheClonePath: "/cache"}
	m.ShowOptions(testRepo(), 255, "feat/oauth", plan)

	// Navigate to "Open on GitHub" (index 2).
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	msg := cmd()
	chosen, ok := msg.(CheckoutStrategyChosenMsg)
	require.True(t, ok)
	assert.Equal(t, "browser", chosen.Strategy)
}

func TestCheckoutDialogOptionsEsc(t *testing.T) {
	m := testCheckoutDialog()
	plan := usecase.CheckoutPlan{Strategy: usecase.StrategyNeedsClone}
	m.ShowOptions(testRepo(), 255, "feat/oauth", plan)

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	msg := cmd()
	_, ok := msg.(CheckoutDialogCloseMsg)
	assert.True(t, ok)
}

func TestCheckoutDialogCustomPathEscGoesBack(t *testing.T) {
	m := testCheckoutDialog()
	plan := usecase.CheckoutPlan{Strategy: usecase.StrategyNeedsClone, CacheClonePath: "/cache"}
	m.ShowOptions(testRepo(), 255, "feat/oauth", plan)

	// Navigate to custom path option and select it.
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, checkoutCustomPath, m.state)

	// Esc should go back to options, not close.
	m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	assert.Equal(t, checkoutOptions, m.state)
	assert.True(t, m.Active())
}

func TestCheckoutDialogLoadingStates(t *testing.T) {
	m := testCheckoutDialog()

	cmd := m.ShowCloning("/cache/path")
	assert.True(t, m.IsLoading())
	assert.NotNil(t, cmd)
	view := m.View()
	assert.Contains(t, view, "Cloning Repository")

	_ = m.ShowCheckingOut("/cache/path")
	assert.True(t, m.IsLoading())
	view = m.View()
	assert.Contains(t, view, "Checking Out")
}

func TestCheckoutDialogSpinnerTick(t *testing.T) {
	m := testCheckoutDialog()
	m.ShowCloning("/path")
	initialFrame := m.spinnerFrame

	cmd := m.Update(checkoutDialogSpinnerTick{})
	assert.Equal(t, initialFrame+1, m.spinnerFrame)
	assert.NotNil(t, cmd, "should return next tick")
}

func TestCheckoutDialogSuccessCWD(t *testing.T) {
	m := testCheckoutDialog()
	m.ShowSuccess("feat/oauth", "/path", true)

	assert.True(t, m.Active())
	view := m.View()
	assert.Contains(t, view, "Checkout Complete")
	assert.Contains(t, view, "feat/oauth")
	assert.Contains(t, view, "Press any key to continue")
	assert.NotContains(t, view, "copy cd command")
}

func TestCheckoutDialogSuccessRemote(t *testing.T) {
	m := testCheckoutDialog()
	m.ShowSuccess("feat/oauth", "/cache/path", false)

	view := m.View()
	assert.Contains(t, view, "Checkout Complete")
	assert.Contains(t, view, "feat/oauth")
	assert.Contains(t, view, "copy cd command")
	assert.Contains(t, view, "cd /cache/path")
}

func TestCheckoutDialogSuccessCopyKey(t *testing.T) {
	m := testCheckoutDialog()
	m.ShowSuccess("feat/oauth", "/cache/path", false)

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	require.NotNil(t, cmd)
	msg := cmd()
	copyMsg, ok := msg.(CopyCdCommandMsg)
	require.True(t, ok)
	assert.Equal(t, "/cache/path", copyMsg.Path)
}

func TestCheckoutDialogSuccessDismiss(t *testing.T) {
	m := testCheckoutDialog()
	m.ShowSuccess("feat/oauth", "/path", true)

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(CheckoutDialogCloseMsg)
	assert.True(t, ok)
	assert.False(t, m.Active())
}

func TestCheckoutDialogError(t *testing.T) {
	m := testCheckoutDialog()
	m.ShowError(errors.New("not a git repository"))

	assert.True(t, m.Active())
	view := m.View()
	assert.Contains(t, view, "Checkout Failed")
	assert.Contains(t, view, "not a git repository")
}

func TestCheckoutDialogErrorDismiss(t *testing.T) {
	m := testCheckoutDialog()
	m.ShowError(errors.New("fail"))

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(CheckoutDialogCloseMsg)
	assert.True(t, ok)
	assert.False(t, m.Active())
}

func TestCheckoutDialogStatusHints(t *testing.T) {
	m := testCheckoutDialog()

	m.ShowCloning("/path")
	assert.Equal(t, "Cloning...", m.StatusHint())

	m.ShowCheckingOut("/path")
	assert.Equal(t, "Checking out...", m.StatusHint())

	m.ShowSuccess("branch", "/path", false)
	assert.Equal(t, "Press any key to continue", m.StatusHint())

	m.ShowError(errors.New("fail"))
	assert.Equal(t, "Press any key to continue", m.StatusHint())
}

func TestExpandPath(t *testing.T) {
	assert.Empty(t, expandPath(""))
	assert.Empty(t, expandPath("   "))

	// Absolute path stays absolute.
	assert.Equal(t, "/usr/local/bin", expandPath("/usr/local/bin"))
}

func TestShortenPath(t *testing.T) {
	// Non-home path stays as-is.
	assert.Equal(t, "/tmp/test", shortenPath("/tmp/test"))
}

func TestSanitizeBranch(t *testing.T) {
	assert.Equal(t, "feat-oauth", sanitizeBranch("feat/oauth"))
	assert.Equal(t, "simple", sanitizeBranch("simple"))
}
