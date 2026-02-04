package plugin

import (
	"fmt"
	"sync"

	"github.com/indrasvat/vivecaka/internal/domain"
)

// Registry manages plugin lifecycle and capability discovery.
type Registry struct {
	mu        sync.RWMutex
	plugins   map[string]Plugin
	readers   []domain.PRReader
	reviewers []domain.PRReviewer
	writers   []domain.PRWriter
	views     []ViewRegistration
	keys      []KeyRegistration
	hooks     *HookManager
}

// NewRegistry creates a new plugin registry.
func NewRegistry() *Registry {
	return &Registry{
		plugins: make(map[string]Plugin),
		hooks:   NewHookManager(),
	}
}

// Register adds a plugin and auto-discovers its capabilities via type assertion.
func (r *Registry) Register(p Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	info := p.Info()
	if _, exists := r.plugins[info.Name]; exists {
		return fmt.Errorf("plugin %q already registered", info.Name)
	}
	r.plugins[info.Name] = p

	if reader, ok := p.(domain.PRReader); ok {
		r.readers = append(r.readers, reader)
	}
	if reviewer, ok := p.(domain.PRReviewer); ok {
		r.reviewers = append(r.reviewers, reviewer)
	}
	if writer, ok := p.(domain.PRWriter); ok {
		r.writers = append(r.writers, writer)
	}
	if vp, ok := p.(ViewPlugin); ok {
		r.views = append(r.views, vp.Views()...)
	}
	if kp, ok := p.(KeyPlugin); ok {
		r.keys = append(r.keys, kp.KeyBindings()...)
	}
	return nil
}

// Unregister removes a plugin by name.
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.plugins, name)
	// Note: capability slices are not pruned for simplicity.
	// Full cleanup would require tracking which plugin provided each capability.
}

// GetReaders returns all registered PRReader implementations.
func (r *Registry) GetReaders() []domain.PRReader {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.readers
}

// GetReviewers returns all registered PRReviewer implementations.
func (r *Registry) GetReviewers() []domain.PRReviewer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.reviewers
}

// GetWriters returns all registered PRWriter implementations.
func (r *Registry) GetWriters() []domain.PRWriter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.writers
}

// Hooks returns the hook manager.
func (r *Registry) Hooks() *HookManager {
	return r.hooks
}
