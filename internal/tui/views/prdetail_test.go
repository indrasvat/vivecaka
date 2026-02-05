package views

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/indrasvat/vivecaka/internal/domain"
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
		Comments: []domain.CommentThread{
			{
				Path: "plugin.go", Line: 42, Resolved: false,
				Comments: []domain.Comment{
					{Author: "alice", Body: "Needs error handling here."},
					{Author: "indrasvat", Body: "Good catch, fixing."},
				},
			},
			{
				Path: "registry.go", Line: 10, Resolved: true,
				Comments: []domain.Comment{
					{Author: "bob", Body: "Consider using sync.Map."},
				},
			},
		},
	}
}

func TestNewPRDetailModel(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	if !m.loading {
		t.Error("new model should be in loading state")
	}
	if m.pane != PaneInfo {
		t.Errorf("default pane = %d, want PaneInfo", m.pane)
	}
}

func TestSetDetail(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetDetail(testDetail())

	if m.loading {
		t.Error("loading should be false after SetDetail")
	}
	if m.detail.Number != 42 {
		t.Errorf("detail.Number = %d, want 42", m.detail.Number)
	}
	if m.pendingNum != 0 {
		t.Errorf("pendingNum = %d, want 0", m.pendingNum)
	}
}

func TestDetailStartLoading(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	cmd := m.StartLoading(99)

	if !m.loading {
		t.Error("StartLoading should set loading = true")
	}
	if m.pendingNum != 99 {
		t.Errorf("pendingNum = %d, want 99", m.pendingNum)
	}

	view := m.View()
	if !strings.Contains(view, "Loading PR #99") {
		t.Errorf("loading view should include PR number, got %q", view)
	}
	if cmd == nil {
		t.Error("StartLoading should return a spinner command")
	}
}

func TestDetailSpinnerTick(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	cmd := m.StartLoading(1)
	if cmd == nil {
		t.Fatal("StartLoading should return a spinner command")
	}
	first := m.spinner.View()
	msg := cmd()
	next := m.Update(msg)
	second := m.spinner.View()
	if first == second {
		t.Errorf("spinner frame should advance, got %q", second)
	}
	if next == nil {
		t.Error("spinner tick should return a follow-up command")
	}
}

func TestDetailSetSize(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	if m.width != 120 || m.height != 40 {
		t.Errorf("size = %dx%d, want 120x40", m.width, m.height)
	}
}

func TestDetailTabNavigation(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDetail(testDetail())

	// Tab forward through all panes.
	tab := tea.KeyMsg{Type: tea.KeyTab}
	m.Update(tab)
	if m.pane != PaneFiles {
		t.Errorf("after 1 tab pane = %d, want PaneFiles", m.pane)
	}

	m.Update(tab)
	if m.pane != PaneChecks {
		t.Errorf("after 2 tabs pane = %d, want PaneChecks", m.pane)
	}

	m.Update(tab)
	if m.pane != PaneComments {
		t.Errorf("after 3 tabs pane = %d, want PaneComments", m.pane)
	}

	m.Update(tab)
	if m.pane != PaneInfo {
		t.Errorf("after 4 tabs pane = %d, want PaneInfo (wrap)", m.pane)
	}
}

func TestDetailShiftTabNavigation(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDetail(testDetail())

	// Shift-tab wraps backward.
	shiftTab := tea.KeyMsg{Type: tea.KeyShiftTab}
	m.Update(shiftTab)
	if m.pane != PaneComments {
		t.Errorf("after shift-tab pane = %d, want PaneComments", m.pane)
	}
}

func TestDetailScrolling(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDetail(testDetail())

	down := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	m.Update(down)
	if m.scrollY != 1 {
		t.Errorf("scrollY after j = %d, want 1", m.scrollY)
	}

	up := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	m.Update(up)
	if m.scrollY != 0 {
		t.Errorf("scrollY after k = %d, want 0", m.scrollY)
	}

	// Up at 0 stays at 0.
	m.Update(up)
	if m.scrollY != 0 {
		t.Errorf("scrollY shouldn't go below 0, got %d", m.scrollY)
	}
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
	if m.scrollY != 0 {
		t.Errorf("scrollY should reset on tab switch, got %d", m.scrollY)
	}
}

func TestDetailEnterOnFilesPane(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDetail(testDetail())

	// Switch to Files pane.
	tab := tea.KeyMsg{Type: tea.KeyTab}
	m.Update(tab)
	if m.pane != PaneFiles {
		t.Fatal("expected PaneFiles")
	}

	// Enter should produce OpenDiffMsg.
	enter := tea.KeyMsg{Type: tea.KeyEnter}
	cmd := m.Update(enter)
	if cmd == nil {
		t.Fatal("Enter on Files pane should produce a command")
	}

	msg := cmd()
	if diff, ok := msg.(OpenDiffMsg); !ok {
		t.Errorf("expected OpenDiffMsg, got %T", msg)
	} else if diff.Number != 42 {
		t.Errorf("OpenDiffMsg.Number = %d, want 42", diff.Number)
	}
}

func TestDetailEnterOnInfoPane(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDetail(testDetail())

	// Enter on Info pane does nothing special.
	enter := tea.KeyMsg{Type: tea.KeyEnter}
	cmd := m.Update(enter)
	if cmd != nil {
		t.Error("Enter on Info pane should not produce a command")
	}
}

func TestPRDetailDiffKey(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDetail(testDetail())

	d := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	cmd := m.Update(d)
	if cmd == nil {
		t.Fatal("'d' key should produce a command")
	}

	msg := cmd()
	if diff, ok := msg.(OpenDiffMsg); !ok {
		t.Errorf("expected OpenDiffMsg, got %T", msg)
	} else if diff.Number != 42 {
		t.Errorf("OpenDiffMsg.Number = %d, want 42", diff.Number)
	}
}

func TestPRDetailCheckoutKey(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDetail(testDetail())

	c := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
	cmd := m.Update(c)
	if cmd == nil {
		t.Fatal("'c' key should produce a command")
	}

	msg := cmd()
	if checkout, ok := msg.(CheckoutPRMsg); !ok {
		t.Errorf("expected CheckoutPRMsg, got %T", msg)
	} else if checkout.Number != 42 {
		t.Errorf("CheckoutPRMsg.Number = %d, want 42", checkout.Number)
	}
}

func TestPRDetailOpenKeyChecksPane(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDetail(testDetail())
	m.pane = PaneChecks
	m.scrollY = 1

	o := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}}
	cmd := m.Update(o)
	if cmd == nil {
		t.Fatal("'o' key should produce a command on checks pane")
	}

	msg := cmd()
	open, ok := msg.(OpenBrowserMsg)
	if !ok {
		t.Fatalf("expected OpenBrowserMsg, got %T", msg)
	}
	if open.URL != "https://example.com/checks/lint" {
		t.Errorf("OpenBrowserMsg.URL = %q, want lint URL", open.URL)
	}
}

func TestPRDetailOpenKeyInfoPane(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDetail(testDetail())
	m.pane = PaneInfo

	o := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}}
	cmd := m.Update(o)
	if cmd == nil {
		t.Fatal("'o' key should produce a command on info pane")
	}

	msg := cmd()
	open, ok := msg.(OpenBrowserMsg)
	if !ok {
		t.Fatalf("expected OpenBrowserMsg, got %T", msg)
	}
	if open.URL != "https://example.com/pr/42" {
		t.Errorf("OpenBrowserMsg.URL = %q, want PR URL", open.URL)
	}
}

func TestDetailReviewKey(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDetail(testDetail())

	// 'r' should produce StartReviewMsg.
	r := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	cmd := m.Update(r)
	if cmd == nil {
		t.Fatal("'r' key should produce a command")
	}

	msg := cmd()
	if rev, ok := msg.(StartReviewMsg); !ok {
		t.Errorf("expected StartReviewMsg, got %T", msg)
	} else if rev.Number != 42 {
		t.Errorf("StartReviewMsg.Number = %d, want 42", rev.Number)
	}
}

func TestPRDetailFormatCheckSummary(t *testing.T) {
	detail := testDetail()
	got := formatCheckSummary(detail.Checks)
	want := "1/3 passing, 1 failing, 1 pending"
	if got != want {
		t.Errorf("summary = %q, want %q", got, want)
	}
}

func TestDetailReviewKeyNoDetail(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	// No detail set.

	r := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	cmd := m.Update(r)
	if cmd != nil {
		t.Error("'r' with no detail should not produce a command")
	}
}

func TestDetailLoadedMsg(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)

	detail := testDetail()
	m.Update(PRDetailLoadedMsg{Detail: detail})

	if m.loading {
		t.Error("should not be loading after PRDetailLoadedMsg")
	}
	if m.detail != detail {
		t.Error("detail should be set from message")
	}
}

func TestDetailViewLoading(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(80, 24)

	view := m.View()
	if view == "" {
		t.Error("loading view should not be empty")
	}
}

func TestDetailViewInfoPane(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDetail(testDetail())

	view := m.View()
	if view == "" {
		t.Error("info pane view should not be empty")
	}
}

func TestDetailViewFilesPane(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDetail(testDetail())
	m.pane = PaneFiles

	view := m.View()
	if view == "" {
		t.Error("files pane view should not be empty")
	}
}

func TestDetailViewChecksPane(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDetail(testDetail())
	m.pane = PaneChecks

	view := m.View()
	if view == "" {
		t.Error("checks pane view should not be empty")
	}
}

func TestDetailViewCommentsPane(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	m.SetDetail(testDetail())
	m.pane = PaneComments

	view := m.View()
	if view == "" {
		t.Error("comments pane view should not be empty")
	}
}

func TestDetailViewNoBody(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	d := testDetail()
	d.Body = ""
	m.SetDetail(d)

	view := m.View()
	if view == "" {
		t.Error("view with no body should not be empty")
	}
}

func TestDetailViewEmptyFiles(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	d := testDetail()
	d.Files = nil
	m.SetDetail(d)
	m.pane = PaneFiles

	view := m.View()
	if view == "" {
		t.Error("empty files pane view should not be empty")
	}
}

func TestDetailViewEmptyChecks(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	d := testDetail()
	d.Checks = nil
	m.SetDetail(d)
	m.pane = PaneChecks

	view := m.View()
	if view == "" {
		t.Error("empty checks pane view should not be empty")
	}
}

func TestDetailViewEmptyComments(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(120, 40)
	d := testDetail()
	d.Comments = nil
	m.SetDetail(d)
	m.pane = PaneComments

	view := m.View()
	if view == "" {
		t.Error("empty comments pane view should not be empty")
	}
}

func TestDetailViewSmallHeight(t *testing.T) {
	m := NewPRDetailModel(testStyles(), testKeys())
	m.SetSize(80, 2) // Very small height.
	m.SetDetail(testDetail())

	view := m.View()
	if view == "" {
		t.Error("small height view should not be empty")
	}
}

func TestRenderMarkdownBold(t *testing.T) {
	out := renderMarkdown("**bold**", 40)
	if !strings.Contains(out, "bold") {
		t.Errorf("rendered markdown should contain text, got %q", out)
	}
}

func TestRenderMarkdownList(t *testing.T) {
	out := renderMarkdown("- item", 40)
	if !strings.Contains(out, "item") {
		t.Errorf("rendered list should contain item, got %q", out)
	}
	if !strings.Contains(out, "•") {
		t.Errorf("rendered list should include bullet, got %q", out)
	}
}
