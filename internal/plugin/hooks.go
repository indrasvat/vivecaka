package plugin

import (
	"context"
	"sync"
)

// HookPoint identifies a lifecycle event that plugins can hook into.
type HookPoint string

const (
	HookBeforeFetch  HookPoint = "before_fetch"
	HookAfterFetch   HookPoint = "after_fetch"
	HookBeforeRender HookPoint = "before_render"
	HookOnPRSelect   HookPoint = "on_pr_select"
	HookOnViewChange HookPoint = "on_view_change"
)

// HookHandler is a function that handles a lifecycle event.
type HookHandler func(ctx context.Context, data any) error

// HookManager manages event hooks with thread-safe registration and emission.
type HookManager struct {
	mu    sync.RWMutex
	hooks map[HookPoint][]HookHandler
}

// NewHookManager creates a new HookManager.
func NewHookManager() *HookManager {
	return &HookManager{
		hooks: make(map[HookPoint][]HookHandler),
	}
}

// On registers a handler for a hook point.
func (hm *HookManager) On(point HookPoint, handler HookHandler) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.hooks[point] = append(hm.hooks[point], handler)
}

// Emit calls all handlers for a hook point in registration order.
// If any handler returns an error, emission stops and the error is returned.
func (hm *HookManager) Emit(ctx context.Context, point HookPoint, data any) error {
	hm.mu.RLock()
	handlers := hm.hooks[point]
	hm.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler(ctx, data); err != nil {
			return err
		}
	}
	return nil
}
