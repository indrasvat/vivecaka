package ghcli

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	require.NoError(t, err, "failed to read fixture %s", name)
	return data
}

func TestToDomainPR_FromFixture(t *testing.T) {
	data := loadFixture(t, "pr_list.json")
	var ghPRs []ghPR
	require.NoError(t, json.Unmarshal(data, &ghPRs))
	require.Len(t, ghPRs, 3)

	// First PR: approved, CI pass, not draft.
	pr := toDomainPR(ghPRs[0])
	assert.Equal(t, 42, pr.Number)
	assert.Equal(t, "Add user authentication", pr.Title)
	assert.Equal(t, "alice", pr.Author)
	assert.Equal(t, domain.PRStateOpen, pr.State)
	assert.False(t, pr.Draft)
	assert.Equal(t, "feat/auth", pr.Branch.Head)
	assert.Equal(t, "main", pr.Branch.Base)
	assert.Equal(t, []string{"enhancement", "security"}, pr.Labels)
	assert.Equal(t, domain.CIPass, pr.CI)
	assert.Equal(t, domain.ReviewApproved, pr.Review.State)
	assert.Equal(t, "https://github.com/owner/repo/pull/42", pr.URL)
	assert.False(t, pr.UpdatedAt.IsZero())
	assert.False(t, pr.CreatedAt.IsZero())

	// Second PR: draft, CI pending, review required.
	pr2 := toDomainPR(ghPRs[1])
	assert.Equal(t, 43, pr2.Number)
	assert.True(t, pr2.Draft)
	assert.Equal(t, domain.CIPending, pr2.CI)
	assert.Equal(t, domain.ReviewPending, pr2.Review.State)

	// Third PR: CI fail, changes requested.
	pr3 := toDomainPR(ghPRs[2])
	assert.Equal(t, 44, pr3.Number)
	assert.Equal(t, domain.CIFail, pr3.CI)
	assert.Equal(t, domain.ReviewChangesRequested, pr3.Review.State)
}

func TestToDomainPRDetail_FromFixture(t *testing.T) {
	data := loadFixture(t, "pr_detail.json")
	var g ghPR
	require.NoError(t, json.Unmarshal(data, &g))

	detail := toDomainPRDetail(g)

	assert.Equal(t, 42, detail.Number)
	assert.Equal(t, "alice", detail.Author)
	assert.Contains(t, detail.Body, "OAuth2-based authentication")
	assert.Equal(t, []string{"alice", "dave"}, detail.Assignees)

	// Reviewers: 2 from review requests + 2 from latest reviews.
	require.Len(t, detail.Reviewers, 4)
	assert.Equal(t, "eve", detail.Reviewers[0].Login)
	assert.Equal(t, domain.ReviewPending, detail.Reviewers[0].State)
	assert.Equal(t, "security-team", detail.Reviewers[1].Login)
	assert.Equal(t, domain.ReviewPending, detail.Reviewers[1].State)
	assert.Equal(t, "frank", detail.Reviewers[2].Login)
	assert.Equal(t, domain.ReviewApproved, detail.Reviewers[2].State)
	assert.Equal(t, "grace", detail.Reviewers[3].Login)
	assert.Equal(t, domain.ReviewChangesRequested, detail.Reviewers[3].State)

	// Files.
	require.Len(t, detail.Files, 3)
	assert.Equal(t, "internal/auth/middleware.go", detail.Files[0].Path)
	assert.Equal(t, 120, detail.Files[0].Additions)
	assert.Equal(t, 5, detail.Files[0].Deletions)

	// Checks.
	require.Len(t, detail.Checks, 2)
	assert.Equal(t, "CI", detail.Checks[0].Name)
	assert.Equal(t, domain.CIPass, detail.Checks[0].Status)
	assert.Equal(t, "Lint", detail.Checks[1].Name)
	assert.Equal(t, domain.CISkipped, detail.Checks[1].Status)
}

func TestGroupCommentsIntoThreads_FromFixture(t *testing.T) {
	data := loadFixture(t, "pr_comments.json")
	var ghComments []ghAPIComment
	require.NoError(t, json.Unmarshal(data, &ghComments))

	threads := groupCommentsIntoThreads(ghComments)

	// Should have 3 threads: comment 1001 (with reply 1002), 1003, 1004.
	require.Len(t, threads, 3)

	// Thread 1: root + reply.
	assert.Equal(t, "1001", threads[0].ID)
	assert.Equal(t, "internal/auth/middleware.go", threads[0].Path)
	assert.Equal(t, 45, threads[0].Line)
	require.Len(t, threads[0].Comments, 2)
	assert.Equal(t, "frank", threads[0].Comments[0].Author)
	assert.Equal(t, "alice", threads[0].Comments[1].Author)
	assert.Contains(t, threads[0].Comments[1].Body, "next commit")

	// Thread 2: standalone.
	assert.Equal(t, "1003", threads[1].ID)
	assert.Equal(t, "internal/auth/token.go", threads[1].Path)
	assert.Equal(t, 20, threads[1].Line)
	require.Len(t, threads[1].Comments, 1)

	// Thread 3: null line → line=0.
	assert.Equal(t, "1004", threads[2].ID)
	assert.Equal(t, 0, threads[2].Line)
}

func TestGroupCommentsIntoThreads_Empty(t *testing.T) {
	threads := groupCommentsIntoThreads(nil)
	assert.Empty(t, threads)
}

func TestToDomainCheck_AllStatuses(t *testing.T) {
	data := loadFixture(t, "pr_checks.json")
	var ghChecks []ghCheck
	require.NoError(t, json.Unmarshal(data, &ghChecks))
	require.Len(t, ghChecks, 4)

	// Build → SUCCESS.
	c0 := toDomainCheck(ghChecks[0])
	assert.Equal(t, "CI / Build", c0.Name)
	assert.Equal(t, domain.CIPass, c0.Status)
	assert.Equal(t, 5*time.Minute, c0.Duration)

	// Test → FAILURE.
	c1 := toDomainCheck(ghChecks[1])
	assert.Equal(t, domain.CIFail, c1.Status)

	// Lint → SKIPPED.
	c2 := toDomainCheck(ghChecks[2])
	assert.Equal(t, domain.CISkipped, c2.Status)
	assert.Zero(t, c2.Duration)

	// Deploy → IN_PROGRESS.
	c3 := toDomainCheck(ghChecks[3])
	assert.Equal(t, domain.CIPending, c3.Status)
}

func TestAggregateCI(t *testing.T) {
	tests := []struct {
		name   string
		checks []ghCheck
		want   domain.CIStatus
	}{
		{"empty", nil, domain.CINone},
		{"all pass", []ghCheck{{Status: "COMPLETED", Conclusion: "SUCCESS"}}, domain.CIPass},
		{"one fail", []ghCheck{
			{Status: "COMPLETED", Conclusion: "SUCCESS"},
			{Status: "COMPLETED", Conclusion: "FAILURE"},
		}, domain.CIFail},
		{"one pending", []ghCheck{
			{Status: "COMPLETED", Conclusion: "SUCCESS"},
			{Status: "IN_PROGRESS"},
		}, domain.CIPending},
		{"fail overrides pending", []ghCheck{
			{Status: "COMPLETED", Conclusion: "FAILURE"},
			{Status: "IN_PROGRESS"},
		}, domain.CIFail},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, aggregateCI(tt.checks))
		})
	}
}

func TestMapState(t *testing.T) {
	assert.Equal(t, domain.PRStateOpen, mapState("OPEN"))
	assert.Equal(t, domain.PRStateClosed, mapState("CLOSED"))
	assert.Equal(t, domain.PRStateMerged, mapState("MERGED"))
	assert.Equal(t, domain.PRStateOpen, mapState("UNKNOWN"))
}

func TestMapReviewDecision(t *testing.T) {
	assert.Equal(t, domain.ReviewApproved, mapReviewDecision("APPROVED").State)
	assert.Equal(t, domain.ReviewChangesRequested, mapReviewDecision("CHANGES_REQUESTED").State)
	assert.Equal(t, domain.ReviewPending, mapReviewDecision("REVIEW_REQUIRED").State)
	assert.Equal(t, domain.ReviewNone, mapReviewDecision("").State)
}

func TestMapReviewState(t *testing.T) {
	assert.Equal(t, domain.ReviewApproved, mapReviewState("APPROVED"))
	assert.Equal(t, domain.ReviewChangesRequested, mapReviewState("CHANGES_REQUESTED"))
	assert.Equal(t, domain.ReviewPending, mapReviewState("PENDING"))
	assert.Equal(t, domain.ReviewPending, mapReviewState("COMMENTED"))
	assert.Equal(t, domain.ReviewNone, mapReviewState(""))
}

func TestMapCheckStatus(t *testing.T) {
	assert.Equal(t, domain.CIPass, mapCheckStatus("COMPLETED", "SUCCESS"))
	assert.Equal(t, domain.CIFail, mapCheckStatus("COMPLETED", "FAILURE"))
	assert.Equal(t, domain.CIFail, mapCheckStatus("COMPLETED", "TIMED_OUT"))
	assert.Equal(t, domain.CIFail, mapCheckStatus("COMPLETED", "STARTUP_FAILURE"))
	assert.Equal(t, domain.CISkipped, mapCheckStatus("COMPLETED", "SKIPPED"))
	assert.Equal(t, domain.CISkipped, mapCheckStatus("COMPLETED", "NEUTRAL"))
	assert.Equal(t, domain.CIFail, mapCheckStatus("COMPLETED", "CANCELLED"))
	assert.Equal(t, domain.CIPending, mapCheckStatus("IN_PROGRESS", ""))
	assert.Equal(t, domain.CIPending, mapCheckStatus("QUEUED", ""))
	assert.Equal(t, domain.CIPending, mapCheckStatus("PENDING", ""))
	assert.Equal(t, domain.CIPending, mapCheckStatus("WAITING", ""))
	assert.Equal(t, domain.CIPending, mapCheckStatus("REQUESTED", ""))
	assert.Equal(t, domain.CINone, mapCheckStatus("UNKNOWN", ""))
}

func TestParseDiff_FromFixture(t *testing.T) {
	data := loadFixture(t, "pr_diff.txt")
	diff := ParseDiff(string(data))

	require.Len(t, diff.Files, 2)

	// First file: new file.
	assert.Equal(t, "internal/auth/middleware.go", diff.Files[0].Path)
	require.Len(t, diff.Files[0].Hunks, 1)
	assert.Equal(t, 15, len(diff.Files[0].Hunks[0].Lines))

	// Second file: modification.
	assert.Equal(t, "cmd/server/main.go", diff.Files[1].Path)
	require.Len(t, diff.Files[1].Hunks, 1)
}
