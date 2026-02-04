package plugin

import (
	"context"
	"fmt"
	"sync"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/indrasvat/vivecaka/internal/domain"
)

// mockPlugin is a basic plugin for testing.
type mockPlugin struct {
	name string
}

func (m *mockPlugin) Info() PluginInfo {
	return PluginInfo{
		Name:        m.name,
		Version:     "1.0.0",
		Description: "test plugin",
		Provides:    []string{"test"},
	}
}

func (m *mockPlugin) Init(_ AppContext) tea.Cmd { return nil }

// mockReaderPlugin implements Plugin + domain.PRReader.
type mockReaderPlugin struct {
	mockPlugin
}

func (m *mockReaderPlugin) ListPRs(_ context.Context, _ domain.RepoRef, _ domain.ListOpts) ([]domain.PR, error) {
	return nil, nil
}
func (m *mockReaderPlugin) GetPR(_ context.Context, _ domain.RepoRef, _ int) (*domain.PRDetail, error) {
	return nil, nil
}
func (m *mockReaderPlugin) GetDiff(_ context.Context, _ domain.RepoRef, _ int) (*domain.Diff, error) {
	return nil, nil
}
func (m *mockReaderPlugin) GetChecks(_ context.Context, _ domain.RepoRef, _ int) ([]domain.Check, error) {
	return nil, nil
}
func (m *mockReaderPlugin) GetComments(_ context.Context, _ domain.RepoRef, _ int) ([]domain.CommentThread, error) {
	return nil, nil
}

// mockReviewerPlugin implements Plugin + domain.PRReviewer.
type mockReviewerPlugin struct {
	mockPlugin
}

func (m *mockReviewerPlugin) SubmitReview(_ context.Context, _ domain.RepoRef, _ int, _ domain.Review) error {
	return nil
}
func (m *mockReviewerPlugin) AddComment(_ context.Context, _ domain.RepoRef, _ int, _ domain.InlineCommentInput) error {
	return nil
}
func (m *mockReviewerPlugin) ResolveThread(_ context.Context, _ domain.RepoRef, _ string) error {
	return nil
}

// mockWriterPlugin implements Plugin + domain.PRWriter.
type mockWriterPlugin struct {
	mockPlugin
}

func (m *mockWriterPlugin) Checkout(_ context.Context, _ domain.RepoRef, _ int) (string, error) {
	return "", nil
}
func (m *mockWriterPlugin) Merge(_ context.Context, _ domain.RepoRef, _ int, _ domain.MergeOpts) error {
	return nil
}
func (m *mockWriterPlugin) UpdateLabels(_ context.Context, _ domain.RepoRef, _ int, _ []string) error {
	return nil
}

// mockFullPlugin implements Plugin + all three domain interfaces.
type mockFullPlugin struct {
	mockReaderPlugin
	mockReviewerPlugin
	mockWriterPlugin
	info PluginInfo
}

func (m *mockFullPlugin) Info() PluginInfo          { return m.info }
func (m *mockFullPlugin) Init(_ AppContext) tea.Cmd { return nil }

func TestRegistryRegister(t *testing.T) {
	reg := NewRegistry()
	p := &mockPlugin{name: "test-plugin"}

	if err := reg.Register(p); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
}

func TestRegistryDuplicateReturnsError(t *testing.T) {
	reg := NewRegistry()
	p := &mockPlugin{name: "dup"}

	if err := reg.Register(p); err != nil {
		t.Fatalf("first Register() error = %v", err)
	}
	if err := reg.Register(p); err == nil {
		t.Fatal("second Register() with same name should return error")
	}
}

func TestRegistryAutoDiscoverReader(t *testing.T) {
	reg := NewRegistry()
	p := &mockReaderPlugin{mockPlugin: mockPlugin{name: "reader"}}

	if err := reg.Register(p); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	readers := reg.GetReaders()
	if len(readers) != 1 {
		t.Fatalf("GetReaders() len = %d, want 1", len(readers))
	}
}

func TestRegistryAutoDiscoverReviewer(t *testing.T) {
	reg := NewRegistry()
	p := &mockReviewerPlugin{mockPlugin: mockPlugin{name: "reviewer"}}

	if err := reg.Register(p); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	reviewers := reg.GetReviewers()
	if len(reviewers) != 1 {
		t.Fatalf("GetReviewers() len = %d, want 1", len(reviewers))
	}
}

func TestRegistryAutoDiscoverWriter(t *testing.T) {
	reg := NewRegistry()
	p := &mockWriterPlugin{mockPlugin: mockPlugin{name: "writer"}}

	if err := reg.Register(p); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	writers := reg.GetWriters()
	if len(writers) != 1 {
		t.Fatalf("GetWriters() len = %d, want 1", len(writers))
	}
}

func TestRegistryNoCapabilities(t *testing.T) {
	reg := NewRegistry()
	p := &mockPlugin{name: "bare"}

	if err := reg.Register(p); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if got := len(reg.GetReaders()); got != 0 {
		t.Errorf("GetReaders() len = %d, want 0", got)
	}
	if got := len(reg.GetReviewers()); got != 0 {
		t.Errorf("GetReviewers() len = %d, want 0", got)
	}
	if got := len(reg.GetWriters()); got != 0 {
		t.Errorf("GetWriters() len = %d, want 0", got)
	}
}

func TestRegistryUnregister(t *testing.T) {
	reg := NewRegistry()
	p := &mockPlugin{name: "removeme"}

	if err := reg.Register(p); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	reg.Unregister("removeme")

	// Should be able to re-register with same name after unregister.
	if err := reg.Register(p); err != nil {
		t.Fatalf("re-Register() after Unregister() error = %v", err)
	}
}

func TestRegistryAutoDiscoverAllCapabilities(t *testing.T) {
	reg := NewRegistry()
	p := &mockFullPlugin{
		info: PluginInfo{
			Name:        "full",
			Version:     "1.0.0",
			Description: "full capability plugin",
			Provides:    []string{"pr-reader", "pr-reviewer", "pr-writer"},
		},
	}

	if err := reg.Register(p); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if got := len(reg.GetReaders()); got != 1 {
		t.Errorf("GetReaders() len = %d, want 1", got)
	}
	if got := len(reg.GetReviewers()); got != 1 {
		t.Errorf("GetReviewers() len = %d, want 1", got)
	}
	if got := len(reg.GetWriters()); got != 1 {
		t.Errorf("GetWriters() len = %d, want 1", got)
	}
}

func TestRegistryHooksNotNil(t *testing.T) {
	reg := NewRegistry()
	if reg.Hooks() == nil {
		t.Fatal("Hooks() returned nil")
	}
}

func TestRegistryConcurrentAccess(t *testing.T) {
	reg := NewRegistry()
	var wg sync.WaitGroup

	// Concurrent registrations with unique names.
	for i := range 50 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			p := &mockPlugin{name: fmt.Sprintf("plugin-%d", n)}
			_ = reg.Register(p)
		}(i)
	}

	// Concurrent reads while registering.
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = reg.GetReaders()
			_ = reg.GetReviewers()
			_ = reg.GetWriters()
		}()
	}

	wg.Wait()
}
