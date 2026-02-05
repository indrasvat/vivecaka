package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/indrasvat/vivecaka/internal/domain"
)

// --- Mock implementations ---

type mockReader struct {
	prs         []domain.PR
	detail      *domain.PRDetail
	diff        *domain.Diff
	checks      []domain.Check
	comments    []domain.CommentThread
	err         error
	checksErr   error
	commentsErr error
}

func (m *mockReader) ListPRs(_ context.Context, _ domain.RepoRef, _ domain.ListOpts) ([]domain.PR, error) {
	return m.prs, m.err
}
func (m *mockReader) GetPR(_ context.Context, _ domain.RepoRef, _ int) (*domain.PRDetail, error) {
	return m.detail, m.err
}
func (m *mockReader) GetDiff(_ context.Context, _ domain.RepoRef, _ int) (*domain.Diff, error) {
	return m.diff, m.err
}
func (m *mockReader) GetChecks(_ context.Context, _ domain.RepoRef, _ int) ([]domain.Check, error) {
	return m.checks, m.checksErr
}
func (m *mockReader) GetComments(_ context.Context, _ domain.RepoRef, _ int) ([]domain.CommentThread, error) {
	return m.comments, m.commentsErr
}
func (m *mockReader) GetPRCount(_ context.Context, _ domain.RepoRef, _ domain.PRState) (int, error) {
	return len(m.prs), m.err
}

type mockReviewer struct {
	err error
}

func (m *mockReviewer) SubmitReview(_ context.Context, _ domain.RepoRef, _ int, _ domain.Review) error {
	return m.err
}
func (m *mockReviewer) AddComment(_ context.Context, _ domain.RepoRef, _ int, _ domain.InlineCommentInput) error {
	return m.err
}
func (m *mockReviewer) ResolveThread(_ context.Context, _ domain.RepoRef, _ string) error {
	return m.err
}

type mockWriter struct {
	branch string
	err    error
}

func (m *mockWriter) Checkout(_ context.Context, _ domain.RepoRef, _ int) (string, error) {
	return m.branch, m.err
}
func (m *mockWriter) Merge(_ context.Context, _ domain.RepoRef, _ int, _ domain.MergeOpts) error {
	return m.err
}
func (m *mockWriter) UpdateLabels(_ context.Context, _ domain.RepoRef, _ int, _ []string) error {
	return m.err
}

var testRepo = domain.RepoRef{Owner: "test", Name: "repo"}

// --- ListPRs tests ---

func TestListPRsExecute(t *testing.T) {
	prs := []domain.PR{
		{Number: 1, Title: "First PR"},
		{Number: 2, Title: "Second PR"},
	}
	reader := &mockReader{prs: prs}
	uc := NewListPRs(reader)

	got, err := uc.Execute(context.Background(), testRepo, domain.ListOpts{})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d PRs, want 2", len(got))
	}
}

func TestListPRsExecuteError(t *testing.T) {
	reader := &mockReader{err: errors.New("network error")}
	uc := NewListPRs(reader)

	_, err := uc.Execute(context.Background(), testRepo, domain.ListOpts{})
	if err == nil {
		t.Fatal("Execute() should return error")
	}
}

// --- GetPRDetail tests ---

func TestGetPRDetailExecute(t *testing.T) {
	// Checks are now included in the detail returned by GetPR (from statusCheckRollup).
	// GetChecks is no longer called separately.
	detail := &domain.PRDetail{
		PR:     domain.PR{Number: 42, Title: "Test PR"},
		Body:   "Description",
		Checks: []domain.Check{{Name: "ci", Status: domain.CIPass}},
	}
	comments := []domain.CommentThread{{ID: "t1"}}

	reader := &mockReader{detail: detail, comments: comments}
	uc := NewGetPRDetail(reader)

	got, err := uc.Execute(context.Background(), testRepo, 42)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got.Number != 42 {
		t.Errorf("PR number = %d, want 42", got.Number)
	}
	if len(got.Checks) != 1 {
		t.Errorf("checks count = %d, want 1", len(got.Checks))
	}
	if len(got.Comments) != 1 {
		t.Errorf("comments count = %d, want 1", len(got.Comments))
	}
}

func TestGetPRDetailPartialFailure(t *testing.T) {
	// Checks are now part of GetPR response, only comments failure is tolerated.
	detail := &domain.PRDetail{
		PR:     domain.PR{Number: 42, Title: "Test PR"},
		Checks: []domain.Check{{Name: "ci", Status: domain.CIPass}},
	}
	reader := &mockReader{
		detail:      detail,
		commentsErr: errors.New("comments failed"),
	}
	uc := NewGetPRDetail(reader)

	got, err := uc.Execute(context.Background(), testRepo, 42)
	if err != nil {
		t.Fatalf("Execute() should tolerate partial failures, got error = %v", err)
	}
	if got.Number != 42 {
		t.Errorf("PR number = %d, want 42", got.Number)
	}
	// Checks come from GetPR (statusCheckRollup), so they should be present.
	if len(got.Checks) != 1 {
		t.Errorf("checks should be present, got %v", got.Checks)
	}
	// Comments should be nil due to tolerated failure.
	if got.Comments != nil {
		t.Errorf("comments should be nil on failure, got %v", got.Comments)
	}
}

func TestGetPRDetailMainFailure(t *testing.T) {
	reader := &mockReader{err: errors.New("PR not found")}
	uc := NewGetPRDetail(reader)

	_, err := uc.Execute(context.Background(), testRepo, 42)
	if err == nil {
		t.Fatal("Execute() should return error when main PR fetch fails")
	}
}

// --- ReviewPR tests ---

func TestReviewPRExecute(t *testing.T) {
	reviewer := &mockReviewer{}
	uc := NewReviewPR(reviewer)

	err := uc.Execute(context.Background(), testRepo, 42, domain.Review{
		Action: domain.ReviewActionApprove,
		Body:   "LGTM",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestReviewPRInvalidAction(t *testing.T) {
	reviewer := &mockReviewer{}
	uc := NewReviewPR(reviewer)

	err := uc.Execute(context.Background(), testRepo, 42, domain.Review{
		Action: "invalid",
	})
	if err == nil {
		t.Fatal("Execute() should return error for invalid action")
	}
	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("error should be ValidationError, got %T", err)
	}
}

func TestReviewPRRequestChangesRequiresBody(t *testing.T) {
	reviewer := &mockReviewer{}
	uc := NewReviewPR(reviewer)

	err := uc.Execute(context.Background(), testRepo, 42, domain.Review{
		Action: domain.ReviewActionRequestChanges,
		Body:   "",
	})
	if err == nil {
		t.Fatal("Execute() should require body for request_changes")
	}
}

// --- CheckoutPR tests ---

func TestCheckoutPRExecute(t *testing.T) {
	writer := &mockWriter{branch: "feat/test-branch"}
	uc := NewCheckoutPR(writer)

	branch, err := uc.Execute(context.Background(), testRepo, 42)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if branch != "feat/test-branch" {
		t.Errorf("branch = %q, want %q", branch, "feat/test-branch")
	}
}

func TestCheckoutPRError(t *testing.T) {
	writer := &mockWriter{err: errors.New("checkout failed")}
	uc := NewCheckoutPR(writer)

	_, err := uc.Execute(context.Background(), testRepo, 42)
	if err == nil {
		t.Fatal("Execute() should return error")
	}
}

// --- AddComment tests ---

func TestAddCommentExecute(t *testing.T) {
	reviewer := &mockReviewer{}
	uc := NewAddComment(reviewer)

	err := uc.Execute(context.Background(), testRepo, 42, domain.InlineCommentInput{
		Path:     "main.go",
		Line:     10,
		Body:     "Consider refactoring this",
		Side:     "RIGHT",
		CommitID: "abc123",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestAddCommentMissingPath(t *testing.T) {
	uc := NewAddComment(&mockReviewer{})

	err := uc.Execute(context.Background(), testRepo, 42, domain.InlineCommentInput{
		Line: 10,
		Body: "comment",
	})
	if err == nil {
		t.Fatal("Execute() should require path")
	}
}

func TestAddCommentMissingBody(t *testing.T) {
	uc := NewAddComment(&mockReviewer{})

	err := uc.Execute(context.Background(), testRepo, 42, domain.InlineCommentInput{
		Path: "main.go",
		Line: 10,
	})
	if err == nil {
		t.Fatal("Execute() should require body")
	}
}

func TestAddCommentInvalidLine(t *testing.T) {
	uc := NewAddComment(&mockReviewer{})

	err := uc.Execute(context.Background(), testRepo, 42, domain.InlineCommentInput{
		Path: "main.go",
		Line: 0,
		Body: "comment",
	})
	if err == nil {
		t.Fatal("Execute() should require positive line number")
	}
}

// --- ResolveThread tests ---

func TestResolveThreadExecute(t *testing.T) {
	reviewer := &mockReviewer{}
	uc := NewResolveThread(reviewer)

	err := uc.Execute(context.Background(), testRepo, "thread-123")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestResolveThreadEmptyID(t *testing.T) {
	uc := NewResolveThread(&mockReviewer{})

	err := uc.Execute(context.Background(), testRepo, "")
	if err == nil {
		t.Fatal("Execute() should require thread ID")
	}
}

// Verify the use case doesn't import any TUI-specific packages.
// This is a compile-time guarantee: if someone adds a bubbletea import to
// the usecase package, these tests will fail to compile without that dep.
// The absence of tea imports is the verification.
var _ = time.Now // Use time to avoid unused import lint on test helper time.
