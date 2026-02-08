package plugin

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHookManagerOnAndEmit(t *testing.T) {
	hm := NewHookManager()
	called := false

	hm.On(HookBeforeFetch, func(_ context.Context, _ any) error {
		called = true
		return nil
	})

	err := hm.Emit(context.Background(), HookBeforeFetch, nil)
	require.NoError(t, err)
	assert.True(t, called, "handler was not called")
}

func TestHookManagerEmitOrdering(t *testing.T) {
	hm := NewHookManager()
	var order []int

	for i := range 5 {
		n := i
		hm.On(HookAfterFetch, func(_ context.Context, _ any) error {
			order = append(order, n)
			return nil
		})
	}

	err := hm.Emit(context.Background(), HookAfterFetch, nil)
	require.NoError(t, err)

	require.Len(t, order, 5)
	for i, v := range order {
		assert.Equal(t, i, v)
	}
}

func TestHookManagerEmitErrorStopsChain(t *testing.T) {
	hm := NewHookManager()
	sentinel := errors.New("stop here")
	callCount := 0

	hm.On(HookBeforeRender, func(_ context.Context, _ any) error {
		callCount++
		return sentinel
	})
	hm.On(HookBeforeRender, func(_ context.Context, _ any) error {
		callCount++
		return nil
	})

	err := hm.Emit(context.Background(), HookBeforeRender, nil)
	assert.ErrorIs(t, err, sentinel)
	assert.Equal(t, 1, callCount, "second handler should not run")
}

func TestHookManagerEmitNoHandlers(t *testing.T) {
	hm := NewHookManager()

	err := hm.Emit(context.Background(), HookOnPRSelect, nil)
	assert.NoError(t, err, "Emit() with no handlers should return nil")
}

func TestHookManagerEmitPassesData(t *testing.T) {
	hm := NewHookManager()
	var received any

	hm.On(HookOnViewChange, func(_ context.Context, data any) error {
		received = data
		return nil
	})

	payload := "test-data"
	err := hm.Emit(context.Background(), HookOnViewChange, payload)
	require.NoError(t, err)
	assert.Equal(t, payload, received)
}

func TestHookManagerConcurrentAccess(t *testing.T) {
	hm := NewHookManager()
	var wg sync.WaitGroup

	// Concurrent registrations.
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			hm.On(HookBeforeFetch, func(_ context.Context, _ any) error {
				return nil
			})
		}()
	}

	// Concurrent emissions.
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = hm.Emit(context.Background(), HookBeforeFetch, nil)
		}()
	}

	wg.Wait()
}

func TestHookPointConstants(t *testing.T) {
	points := []HookPoint{
		HookBeforeFetch,
		HookAfterFetch,
		HookBeforeRender,
		HookOnPRSelect,
		HookOnViewChange,
	}
	seen := make(map[HookPoint]bool)
	for _, p := range points {
		assert.False(t, seen[p], "duplicate HookPoint value: %q", p)
		seen[p] = true
		assert.NotEmpty(t, string(p), "HookPoint should not be empty string")
	}
}
