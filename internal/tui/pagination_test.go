package tui

import (
	"testing"

	"github.com/indrasvat/vivecaka/internal/config"
	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/indrasvat/vivecaka/internal/tui/views"
)

// TestPaginationFlow verifies the pagination state machine works correctly.
// Note: This tests the message handling flow. LoadMorePRsMsg requires adapters
// to be set up (which happens in the real app), so we focus on MorePRsLoadedMsg.
func TestPaginationFlow(t *testing.T) {
	cfg := config.Default()
	cfg.General.PageSize = 50
	app := New(cfg)

	// Simulate initial PR load (50 PRs)
	initialPRs := make([]domain.PR, 50)
	for i := range 50 {
		initialPRs[i] = domain.PR{Number: i + 1, Title: "PR " + string(rune('A'+i%26))}
	}

	// Process initial load
	app.Update(views.PRsLoadedMsg{PRs: initialPRs, Err: nil})

	// Verify initial state
	if got := app.prList.TotalPRs(); got != 50 {
		t.Errorf("After initial load: TotalPRs() = %d, want 50", got)
	}
	if got := app.prList.CurrentPage(); got != 1 {
		t.Errorf("After initial load: CurrentPage() = %d, want 1", got)
	}
	if !app.prList.HasMore() {
		t.Error("After initial load: HasMore() = false, want true")
	}

	// Note: LoadMorePRsMsg requires a.listPRs and a.repo to be set, which they aren't in this test.
	// In the real app, these are set via WithReader() and repo detection.
	// We skip testing LoadMorePRsMsg state transition here and focus on MorePRsLoadedMsg.

	// Manually set loading state (simulating what handleLoadMorePRs does)
	app.prList.SetLoadingMore(2)

	// Verify loading state was set
	if !app.prList.IsLoadingMore() {
		t.Error("After SetLoadingMore: IsLoadingMore() = false, want true")
	}
	if got := app.prList.CurrentPage(); got != 2 {
		t.Errorf("After SetLoadingMore: CurrentPage() = %d, want 2", got)
	}

	// Simulate more PRs loaded (another 50)
	morePRs := make([]domain.PR, 50)
	for i := range 50 {
		morePRs[i] = domain.PR{Number: 51 + i, Title: "PR " + string(rune('A'+(50+i)%26))}
	}

	app.Update(views.MorePRsLoadedMsg{PRs: morePRs, Page: 2, HasMore: true, Err: nil})

	// Verify state after pagination
	if got := app.prList.TotalPRs(); got != 100 {
		t.Errorf("After MorePRsLoadedMsg: TotalPRs() = %d, want 100", got)
	}
	if app.prList.IsLoadingMore() {
		t.Error("After MorePRsLoadedMsg: IsLoadingMore() = true, want false")
	}
	if !app.prList.HasMore() {
		t.Error("After MorePRsLoadedMsg: HasMore() = false, want true")
	}

	t.Logf("Pagination flow: 50 -> 100 PRs works correctly")
}

// TestPaginationNoMoreItems verifies pagination stops when no more items.
func TestPaginationNoMoreItems(t *testing.T) {
	cfg := config.Default()
	cfg.General.PageSize = 50
	app := New(cfg)

	// Initial load with fewer than perPage items (no more pages)
	initialPRs := make([]domain.PR, 30)
	for i := range 30 {
		initialPRs[i] = domain.PR{Number: i + 1}
	}

	app.Update(views.PRsLoadedMsg{PRs: initialPRs, Err: nil})

	if app.prList.HasMore() {
		t.Error("After partial load: HasMore() = true, want false (got fewer than perPage)")
	}
}

// TestPaginationEmptyResponse verifies handling of empty pagination response.
func TestPaginationEmptyResponse(t *testing.T) {
	cfg := config.Default()
	cfg.General.PageSize = 50
	app := New(cfg)

	// Initial load
	initialPRs := make([]domain.PR, 50)
	for i := range 50 {
		initialPRs[i] = domain.PR{Number: i + 1}
	}
	app.Update(views.PRsLoadedMsg{PRs: initialPRs, Err: nil})

	// Request more
	app.Update(views.LoadMorePRsMsg{Page: 2})

	// Empty response (no more items)
	app.Update(views.MorePRsLoadedMsg{PRs: []domain.PR{}, Page: 2, HasMore: false, Err: nil})

	// Should still have 50 PRs, hasMore should be false
	if got := app.prList.TotalPRs(); got != 50 {
		t.Errorf("After empty pagination: TotalPRs() = %d, want 50", got)
	}
	if app.prList.HasMore() {
		t.Error("After empty pagination: HasMore() = true, want false")
	}
}
