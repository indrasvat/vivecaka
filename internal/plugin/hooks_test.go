package plugin

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func TestHookManagerOnAndEmit(t *testing.T) {
	hm := NewHookManager()
	called := false

	hm.On(HookBeforeFetch, func(_ context.Context, _ any) error {
		called = true
		return nil
	})

	if err := hm.Emit(context.Background(), HookBeforeFetch, nil); err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	if !called {
		t.Error("handler was not called")
	}
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

	if err := hm.Emit(context.Background(), HookAfterFetch, nil); err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	if len(order) != 5 {
		t.Fatalf("expected 5 calls, got %d", len(order))
	}
	for i, v := range order {
		if v != i {
			t.Errorf("order[%d] = %d, want %d", i, v, i)
		}
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
	if !errors.Is(err, sentinel) {
		t.Errorf("Emit() error = %v, want %v", err, sentinel)
	}
	if callCount != 1 {
		t.Errorf("callCount = %d, want 1 (second handler should not run)", callCount)
	}
}

func TestHookManagerEmitNoHandlers(t *testing.T) {
	hm := NewHookManager()

	if err := hm.Emit(context.Background(), HookOnPRSelect, nil); err != nil {
		t.Errorf("Emit() with no handlers should return nil, got %v", err)
	}
}

func TestHookManagerEmitPassesData(t *testing.T) {
	hm := NewHookManager()
	var received any

	hm.On(HookOnViewChange, func(_ context.Context, data any) error {
		received = data
		return nil
	})

	payload := "test-data"
	if err := hm.Emit(context.Background(), HookOnViewChange, payload); err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	if received != payload {
		t.Errorf("handler received %v, want %v", received, payload)
	}
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
		if seen[p] {
			t.Errorf("duplicate HookPoint value: %q", p)
		}
		seen[p] = true
		if string(p) == "" {
			t.Error("HookPoint should not be empty string")
		}
	}
}
