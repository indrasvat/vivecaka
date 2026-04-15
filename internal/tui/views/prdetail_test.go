package views

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/indrasvat/vivecaka/internal/reviewprogress"
)

func testDetail() *domain.PRDetail {
	return &domain.PRDetail{
		PR: domain.PR{
			Number: 42,
			Title:  "Add plugin architecture",
			Author: "indrasvat",
			State:  domain.PRStateOpen,
			Branch: domain.BranchInfo{Head: "feat/plugins", Base: "main"},
			Labels: []string{"enhancement", "v1"},
			URL:    "https://example.com/pr/42",
		},
		Body:      "This PR adds plugin support.",
		Assignees: []string{"indrasvat"},
		Reviewers: []domain.ReviewerInfo{
			{Login: "alice", State: "APPROVED"},
			{Login: "bob", State: "PENDING"},
		},
		Files: []domain.FileChange{
			{Path: "plugin.go", Additions: 120, Deletions: 5},
			{Path: "registry.go", Additions: 80, Deletions: 0},
		},
		Checks: []domain.Check{
			{Name: "ci/build", Status: domain.CIPass, Duration: 45 * time.Second, URL: "https://example.com/checks/build"},
			{Name: "ci/lint", Status: domain.CIFail, Duration: 12 * time.Second, URL: "https://example.com/checks/lint"},
			{Name: "ci/test", Status: domain.CIPending},
		},
		InlineComments: []domain.CommentThread{
			{
				ID: "comment-1", ThreadID: "thread-1", ReplyToID: "comment-1",
				Path: "plugin.go", Line: 42, Resolved: false,
				Comments: []domain.Comment{
					{Author: "alice", Body: "Needs error handling here."},
					{Author: "indrasvat", Body: "Good catch, fixing."},
				},
			},
			{
				ID: "comment-2", ThreadID: "thread-2", ReplyToID: "comment-2",
				Path: "registry.go", Line: 10, Resolved: true,
				Comments: []domain.Comment{
					{Author: "bob", Body: "Consider using sync.Map."},
				},
			},
		},
		Discussion: []domain.DiscussionItem{
			{
				ID:         "review-1",
				Kind:       domain.DiscussionReview,
				StateLabel: "approved",
				CreatedAt:  time.Now().Add(-2 * time.Hour),
				URL:        "https://example.com/reviews/1",
				Comments: []domain.Comment{
					{ID: "review-1", Author: "indrasvat", Body: "LGTM", CreatedAt: time.Now().Add(-2 * time.Hour)},
				},
			},
			{
				ID:        "conversation-1",
				Kind:      domain.DiscussionComment,
				CreatedAt: time.Now().Add(-time.Hour),
				URL:       "https://example.com/comments/1",
				Comments: []domain.Comment{
					{ID: "conversation-1", Author: "alice", Body: "Please verify the migration path.", CreatedAt: time.Now().Add(-time.Hour)},
				},
			},
			{
				ID:        "comment-1",
				Kind:      domain.DiscussionInlineThread,
				Path:      "plugin.go",
				Line:      42,
				ThreadID:  "thread-1",
				ReplyToID: "comment-1",
				CreatedAt: time.Now().Add(-30 * time.Minute),
				Comments: []domain.Comment{
					{ID: "comment-1", Author: "alice", Body: "Needs error handling here.", CreatedAt: time.Now().Add(-30 * time.Minute)},
					{ID: "comment-1-reply", Author: "indrasvat", Body: "Good catch, fixing.", CreatedAt: time.Now().Add(-25 * time.Minute)},
				},
			},
			{
				ID:        "comment-2",
				Kind:      domain.DiscussionInlineThread,
				Path:      "registry.go",
				Line:      10,
				Resolved:  true,
				ThreadID:  "thread-2",
				ReplyToID: "comment-2",
				CreatedAt: time.Now().Add(-20 * time.Minute),
				Comments: []domain.Comment{
					{ID: "comment-2", Author: "bob", Body: "Consider using sync.Map.", CreatedAt: time.Now().Add(-20 * time.Minute)},
				},
			},
		},
	}
}

func TestNewPRDetailModel(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	assert.True(t, m.loading, "new model should be in loading state")
	assert.Equal(t, TabDescription, m.tab)
}

func TestSetDetail(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetDetail(testDetail())

	assert.False(t, m.loading, "loading should be false after SetDetail")
	assert.Equal(t, 42, m.detail.Number)
	assert.Equal(t, 0, m.pendingNum)
}

func TestSelectedFilePathWithNilDetail(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())

	assert.Equal(t, "", m.SelectedFilePath())
}

func TestDetailReviewContextRenders(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 24)
	m.SetDetail(testDetail())
	m.SetReviewContext(&reviewprogress.Context{
		Scope:              reviewprogress.ScopeSinceReview,
		ViewedFiles:        1,
		TotalFiles:         2,
		SinceReviewFiles:   1,
		NextActionablePath: "plugin.go",
	})

	view := m.View()
	assert.Contains(t, view, "Review")
	assert.Contains(t, view, "Since Review")
	assert.Contains(t, view, "plugin.go")
}

func TestReviewFileMarkerPrefersViewedState(t *testing.T) {
	file := reviewprogress.File{
		Path:               "plugin.go",
		Viewed:             true,
		ChangedSinceReview: true,
		Actionable:         true,
	}

	assert.Equal(t, "✓", reviewFileMarker(file))
	assert.Contains(t, reviewFileMeta(file, testStyles().Theme), "viewed")
	assert.Contains(t, reviewFileMeta(file, testStyles().Theme), "changed since review")
}

func TestDetailViewFitsAssignedHeight(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 24)
	m.SetDetail(testDetail())
	m.SetReviewContext(&reviewprogress.Context{
		Scope:            reviewprogress.ScopeSinceReview,
		ViewedFiles:      1,
		TotalFiles:       2,
		SinceReviewFiles: 1,
	})

	view := m.View()
	lines := strings.Split(view, "\n")

	assert.Len(t, lines, 24)
	assert.Contains(t, view, "#42")
	assert.Contains(t, view, "Description")
	assert.Contains(t, view, "Review")
}

func TestDetailStartLoading(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	cmd := m.StartLoading(99)

	assert.True(t, m.loading, "StartLoading should set loading = true")
	assert.Equal(t, 99, m.pendingNum)

	view := m.View()
	assert.Contains(t, view, "Loading PR #99")
	assert.NotNil(t, cmd, "StartLoading should return a spinner command")
}

func TestDetailSpinnerTick(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	cmd := m.StartLoading(1)
	require.NotNil(t, cmd, "StartLoading should return a spinner command")

	first := m.spinner.View()
	msg := cmd()
	next := m.Update(msg)
	second := m.spinner.View()
	assert.NotEqual(t, first, second, "spinner frame should advance")
	assert.NotNil(t, next, "spinner tick should return a follow-up command")
}

func TestDetailSetSize(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	assert.Equal(t, 120, m.width)
	assert.Equal(t, 40, m.height)
}

func TestDetailTabNavigation(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDetail(testDetail())

	// Tab forward through all tabs: Description → Checks → Files → Comments → Description
	tab := tea.KeyMsg{Type: tea.KeyTab}
	m.Update(tab)
	assert.Equal(t, TabChecks, m.tab)

	m.Update(tab)
	assert.Equal(t, TabFiles, m.tab)

	m.Update(tab)
	assert.Equal(t, TabComments, m.tab)

	m.Update(tab)
	assert.Equal(t, TabDescription, m.tab)
}

func TestDetailShiftTabNavigation(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDetail(testDetail())

	// Shift-tab wraps backward.
	shiftTab := tea.KeyMsg{Type: tea.KeyShiftTab}
	m.Update(shiftTab)
	assert.Equal(t, TabComments, m.tab)
}

func TestDetailScrolling(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDetail(testDetail())

	down := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	m.Update(down)
	assert.Equal(t, 1, m.scrollY)

	up := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	m.Update(up)
	assert.Equal(t, 0, m.scrollY)

	// Up at 0 stays at 0.
	m.Update(up)
	assert.Equal(t, 0, m.scrollY)
}

func TestDetailScrollResetsOnTabSwitch(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDetail(testDetail())

	// Scroll down, then switch tab.
	down := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	m.Update(down)
	m.Update(down)

	tab := tea.KeyMsg{Type: tea.KeyTab}
	m.Update(tab)
	assert.Equal(t, 0, m.scrollY, "scrollY should reset on tab switch")
}

func TestDetailEnterOnFilesPane(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDetail(testDetail())

	// Switch to Files tab (Description → Checks → Files)
	m.tab = TabFiles

	// Enter should produce OpenDiffMsg.
	enter := tea.KeyMsg{Type: tea.KeyEnter}
	cmd := m.Update(enter)
	require.NotNil(t, cmd, "Enter on Files pane should produce a command")

	msg := cmd()
	diff, ok := msg.(OpenDiffMsg)
	require.True(t, ok, "expected OpenDiffMsg, got %T", msg)
	assert.Equal(t, 42, diff.Number)
}

func TestDetailCycleScopeKey(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 24)
	m.SetDetail(testDetail())

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	require.NotNil(t, cmd)
	_, ok := cmd().(CycleReviewScopeMsg)
	assert.True(t, ok)
}

func TestDetailToggleViewedKeyInFilesTab(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 24)
	m.SetDetail(testDetail())
	m.tab = TabFiles
	m.SetReviewContext(&reviewprogress.Context{
		Scope: reviewprogress.ScopeAll,
		Files: []reviewprogress.File{
			{Path: "plugin.go", Additions: 120, Deletions: 5},
			{Path: "registry.go", Additions: 80, Deletions: 0},
		},
	})

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'V'}})
	require.NotNil(t, cmd)
	msg, ok := cmd().(ToggleViewedFileMsg)
	require.True(t, ok)
	assert.Equal(t, "plugin.go", msg.Path)
}

func TestDetailEnterOnInfoPane(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDetail(testDetail())

	// Enter on Info pane does nothing special.
	enter := tea.KeyMsg{Type: tea.KeyEnter}
	cmd := m.Update(enter)
	assert.Nil(t, cmd, "Enter on Info pane should not produce a command")
}

func TestPRDetailDiffKey(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDetail(testDetail())

	d := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	cmd := m.Update(d)
	require.NotNil(t, cmd, "'d' key should produce a command")

	msg := cmd()
	diff, ok := msg.(OpenDiffMsg)
	require.True(t, ok, "expected OpenDiffMsg, got %T", msg)
	assert.Equal(t, 42, diff.Number)
}

func TestPRDetailCheckoutKey(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDetail(testDetail())

	c := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
	cmd := m.Update(c)
	require.NotNil(t, cmd, "'c' key should produce a command")

	msg := cmd()
	checkout, ok := msg.(CheckoutPRMsg)
	require.True(t, ok, "expected CheckoutPRMsg, got %T", msg)
	assert.Equal(t, 42, checkout.Number)
	assert.Equal(t, "feat/plugins", checkout.Branch)
}

func TestPRDetailOpenKeyChecksPane(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDetail(testDetail())
	m.tab = TabChecks
	m.scrollY = 1

	o := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}}
	cmd := m.Update(o)
	require.NotNil(t, cmd, "'o' key should produce a command on checks pane")

	msg := cmd()
	open, ok := msg.(OpenBrowserMsg)
	require.True(t, ok, "expected OpenBrowserMsg, got %T", msg)
	assert.Equal(t, "https://example.com/checks/lint", open.URL)
}

func TestPRDetailOpenKeyInfoPane(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDetail(testDetail())
	m.tab = TabDescription

	o := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}}
	cmd := m.Update(o)
	require.NotNil(t, cmd, "'o' key should produce a command on info pane")

	msg := cmd()
	open, ok := msg.(OpenBrowserMsg)
	require.True(t, ok, "expected OpenBrowserMsg, got %T", msg)
	assert.Equal(t, "https://example.com/pr/42", open.URL)
}

func TestDetailReviewKey(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDetail(testDetail())

	// 'r' should produce StartReviewMsg.
	r := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	cmd := m.Update(r)
	require.NotNil(t, cmd, "'r' key should produce a command")

	msg := cmd()
	rev, ok := msg.(StartReviewMsg)
	require.True(t, ok, "expected StartReviewMsg, got %T", msg)
	assert.Equal(t, 42, rev.Number)
}

func TestPRDetailFormatCheckSummary(t *testing.T) {
	detail := testDetail()
	got := formatCheckSummary(detail.Checks)
	assert.Equal(t, "1/3 passing, 1 failing, 1 pending", got)
}

func TestDetailReviewKeyNoDetail(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	// No detail set.

	r := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	cmd := m.Update(r)
	assert.Nil(t, cmd, "'r' with no detail should not produce a command")
}

func TestDetailLoadedMsg(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)

	detail := testDetail()
	m.Update(PRDetailLoadedMsg{Detail: detail})

	assert.False(t, m.loading, "should not be loading after PRDetailLoadedMsg")
	assert.Equal(t, detail, m.detail)
}

func TestDetailViewLoading(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(80, 24)

	view := m.View()
	assert.NotEmpty(t, view, "loading view should not be empty")
}

func TestDetailViewInfoPane(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDetail(testDetail())

	view := m.View()
	assert.NotEmpty(t, view, "info pane view should not be empty")
}

func TestDetailViewFilesPane(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDetail(testDetail())
	m.tab = TabFiles

	view := m.View()
	assert.NotEmpty(t, view, "files pane view should not be empty")
}

func TestDetailViewChecksPane(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDetail(testDetail())
	m.tab = TabChecks

	view := m.View()
	assert.NotEmpty(t, view, "checks pane view should not be empty")
}

func TestDetailViewCommentsPane(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDetail(testDetail())
	m.tab = TabComments

	view := m.View()
	assert.NotEmpty(t, view, "comments pane view should not be empty")
}

func TestDetailViewNoBody(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	d := testDetail()
	d.Body = ""
	m.SetDetail(d)

	view := m.View()
	assert.NotEmpty(t, view, "view with no body should not be empty")
}

func TestDetailViewEmptyFiles(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	d := testDetail()
	d.Files = nil
	m.SetDetail(d)
	m.tab = TabFiles

	view := m.View()
	assert.NotEmpty(t, view, "empty files pane view should not be empty")
}

func TestDetailViewEmptyChecks(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	d := testDetail()
	d.Checks = nil
	m.SetDetail(d)
	m.tab = TabChecks

	view := m.View()
	assert.NotEmpty(t, view, "empty checks pane view should not be empty")
}

func TestDetailViewEmptyComments(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	d := testDetail()
	d.Discussion = nil
	m.SetDetail(d)
	m.tab = TabComments

	view := m.View()
	assert.NotEmpty(t, view, "empty comments pane view should not be empty")
}

func TestDetailViewSmallHeight(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(80, 2) // Very small height.
	m.SetDetail(testDetail())

	view := m.View()
	assert.NotEmpty(t, view, "small height view should not be empty")
}

func TestRenderMarkdownBold(t *testing.T) {
	out := renderMarkdown("**bold**", 40)
	assert.Contains(t, out, "bold")
}

func TestRenderMarkdownList(t *testing.T) {
	out := renderMarkdown("- item", 40)
	assert.Contains(t, out, "item")
	assert.Contains(t, out, "•")
}

// Comment pane tests

func TestCommentPaneNavigation(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDetail(testDetail())
	m.tab = TabComments

	// Start at first thread
	assert.Equal(t, 0, m.commentCursor)

	// Navigate down
	down := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	m.Update(down)
	assert.Equal(t, 1, m.commentCursor)

	// Navigate up
	up := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	m.Update(up)
	assert.Equal(t, 0, m.commentCursor)

	// Can't go below 0
	m.Update(up)
	assert.Equal(t, 0, m.commentCursor)
}

func TestCommentPaneCollapseExpand(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDetail(testDetail())
	m.tab = TabComments

	// Initially not collapsed
	assert.False(t, m.commentCollapsed[0], "thread should not be collapsed initially")

	// Collapse with Space
	space := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	m.Update(space)
	assert.True(t, m.commentCollapsed[0], "thread should be collapsed after Space")

	// Expand again
	m.Update(space)
	assert.False(t, m.commentCollapsed[0], "thread should be expanded after second Space")

	// Collapse with za
	z := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}}
	a := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	m.Update(z)
	m.Update(a)
	assert.True(t, m.commentCollapsed[0], "thread should be collapsed after za")
}

func TestCommentPaneResolve(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	detail := testDetail()
	detail.Discussion[2].ThreadID = "thread-1"
	m.SetDetail(detail)
	m.tab = TabComments
	m.commentCursor = 2

	// 'x' should produce ResolveThreadMsg for unresolved thread
	x := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
	cmd := m.Update(x)
	require.NotNil(t, cmd, "'x' key should produce a command")

	msg := cmd()
	resolve, ok := msg.(ResolveThreadMsg)
	require.True(t, ok, "expected ResolveThreadMsg, got %T", msg)
	assert.Equal(t, "thread-1", resolve.ThreadID)
}

func TestCommentPaneUnresolve(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	detail := testDetail()
	detail.Discussion[3].ThreadID = "thread-2" // This one is resolved
	m.SetDetail(detail)
	m.tab = TabComments
	m.commentCursor = 3 // Move to resolved thread

	// 'X' should produce UnresolveThreadMsg for resolved thread
	X := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}}
	cmd := m.Update(X)
	require.NotNil(t, cmd, "'X' key should produce a command")

	msg := cmd()
	unresolve, ok := msg.(UnresolveThreadMsg)
	require.True(t, ok, "expected UnresolveThreadMsg, got %T", msg)
	assert.Equal(t, "thread-2", unresolve.ThreadID)
}

func TestCommentPaneReplyKey(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	detail := testDetail()
	detail.Discussion[2].ReplyToID = "comment-1"
	m.SetDetail(detail)
	m.tab = TabComments
	m.commentCursor = 2

	// 'r' in comments pane should produce ReplyToThreadMsg
	r := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	cmd := m.Update(r)
	require.NotNil(t, cmd, "'r' key should produce a command in comments pane")

	msg := cmd()
	reply, ok := msg.(ReplyToThreadMsg)
	require.True(t, ok, "expected ReplyToThreadMsg, got %T", msg)
	assert.Equal(t, "comment-1", reply.ThreadID)
}

func TestCommentPaneViewRendering(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDetail(testDetail())
	m.tab = TabComments

	view := m.View()
	assert.NotEmpty(t, view, "comments pane view should not be empty")
	// Should contain thread info
	assert.Contains(t, view, "plugin.go")
	assert.Contains(t, view, "review by @indrasvat")
}

func TestCommentPaneCollapsedView(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDetail(testDetail())
	m.tab = TabComments
	m.commentCollapsed[0] = true

	view := m.View()
	// Collapsed view should show preview of first comment
	assert.Contains(t, view, "Needs error handling")
}

func TestEnsureSelectedCommentVisible_AnchorsTallSelectedBlockAtHeader(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(80, 8)

	detail := testDetail()
	detail.Discussion = []domain.DiscussionItem{
		{
			ID:        "short",
			Kind:      domain.DiscussionComment,
			CreatedAt: time.Now().Add(-time.Hour),
			Comments: []domain.Comment{
				{ID: "short", Author: "alice", Body: "short comment", CreatedAt: time.Now().Add(-time.Hour)},
			},
		},
		{
			ID:        "long",
			Kind:      domain.DiscussionComment,
			CreatedAt: time.Now(),
			Comments: []domain.Comment{
				{ID: "long", Author: "indrasvat", Body: strings.Repeat("very long comment line\n", 24), CreatedAt: time.Now()},
			},
		},
	}
	m.SetDetail(detail)
	m.tab = TabComments
	m.commentCursor = 1

	commentWidth := max(20, m.width-10)
	linePos := m.commentBlockHeight(detail.Discussion[0], 0, commentWidth)
	blockHeight := m.commentBlockHeight(detail.Discussion[1], 1, commentWidth)
	require.Greater(t, blockHeight, 8, "selected block must exceed viewport height for this test")

	m.ensureSelectedCommentVisible(8, commentWidth)

	assert.Equal(t, linePos, m.scrollY, "oversized selected block should stay anchored at its header")
}

func TestGetPRNumber(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)

	// No detail set
	assert.Equal(t, 0, m.GetPRNumber(), "GetPRNumber should return 0 when no detail")

	// After setting detail
	m.SetDetail(testDetail())
	assert.Equal(t, 42, m.GetPRNumber())

	// When loading
	m.detail = nil
	m.pendingNum = 123
	assert.Equal(t, 123, m.GetPRNumber())
}
