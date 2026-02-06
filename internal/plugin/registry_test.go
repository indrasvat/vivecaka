package plugin

import (
	"context"
	"fmt"
	"sync"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
func (m *mockReaderPlugin) GetPRCount(_ context.Context, _ domain.RepoRef, _ domain.PRState) (int, error) {
	return 0, nil
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

// mockRepoManagerPlugin implements Plugin + domain.RepoManager.
type mockRepoManagerPlugin struct {
	mockPlugin
}

func (m *mockRepoManagerPlugin) CheckoutAt(_ context.Context, _ domain.RepoRef, _ int, _ string) (string, error) {
	return "", nil
}
func (m *mockRepoManagerPlugin) CloneRepo(_ context.Context, _ domain.RepoRef, _ string) error {
	return nil
}
func (m *mockRepoManagerPlugin) CreateWorktree(_ context.Context, _ string, _ int, _, _ string) error {
	return nil
}

// mockFullPlugin implements Plugin + all domain interfaces including RepoManager.
type mockFullPlugin struct {
	mockReaderPlugin
	mockReviewerPlugin
	mockWriterPlugin
	mockRepoManagerPlugin
	info PluginInfo
}

func (m *mockFullPlugin) Info() PluginInfo          { return m.info }
func (m *mockFullPlugin) Init(_ AppContext) tea.Cmd { return nil }

func TestRegistryRegister(t *testing.T) {
	reg := NewRegistry()
	p := &mockPlugin{name: "test-plugin"}

	err := reg.Register(p)
	require.NoError(t, err)
}

func TestRegistryDuplicateReturnsError(t *testing.T) {
	reg := NewRegistry()
	p := &mockPlugin{name: "dup"}

	err := reg.Register(p)
	require.NoError(t, err)
	err = reg.Register(p)
	assert.Error(t, err, "second Register() with same name should return error")
}

func TestRegistryAutoDiscoverReader(t *testing.T) {
	reg := NewRegistry()
	p := &mockReaderPlugin{mockPlugin: mockPlugin{name: "reader"}}

	err := reg.Register(p)
	require.NoError(t, err)

	readers := reg.GetReaders()
	assert.Len(t, readers, 1)
}

func TestRegistryAutoDiscoverReviewer(t *testing.T) {
	reg := NewRegistry()
	p := &mockReviewerPlugin{mockPlugin: mockPlugin{name: "reviewer"}}

	err := reg.Register(p)
	require.NoError(t, err)

	reviewers := reg.GetReviewers()
	assert.Len(t, reviewers, 1)
}

func TestRegistryAutoDiscoverWriter(t *testing.T) {
	reg := NewRegistry()
	p := &mockWriterPlugin{mockPlugin: mockPlugin{name: "writer"}}

	err := reg.Register(p)
	require.NoError(t, err)

	writers := reg.GetWriters()
	assert.Len(t, writers, 1)
}

func TestRegistryAutoDiscoverRepoManager(t *testing.T) {
	reg := NewRegistry()
	p := &mockRepoManagerPlugin{mockPlugin: mockPlugin{name: "repomgr"}}

	err := reg.Register(p)
	require.NoError(t, err)

	rms := reg.GetRepoManagers()
	assert.Len(t, rms, 1)
}

func TestRegistryNoCapabilities(t *testing.T) {
	reg := NewRegistry()
	p := &mockPlugin{name: "bare"}

	err := reg.Register(p)
	require.NoError(t, err)

	assert.Empty(t, reg.GetReaders())
	assert.Empty(t, reg.GetReviewers())
	assert.Empty(t, reg.GetWriters())
}

func TestRegistryUnregister(t *testing.T) {
	reg := NewRegistry()
	p := &mockPlugin{name: "removeme"}

	err := reg.Register(p)
	require.NoError(t, err)

	reg.Unregister("removeme")

	// Should be able to re-register with same name after unregister.
	err = reg.Register(p)
	require.NoError(t, err)
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

	err := reg.Register(p)
	require.NoError(t, err)

	assert.Len(t, reg.GetReaders(), 1)
	assert.Len(t, reg.GetReviewers(), 1)
	assert.Len(t, reg.GetWriters(), 1)
	assert.Len(t, reg.GetRepoManagers(), 1)
}

func TestRegistryHooksNotNil(t *testing.T) {
	reg := NewRegistry()
	assert.NotNil(t, reg.Hooks())
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
