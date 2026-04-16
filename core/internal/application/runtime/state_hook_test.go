package runtime

import (
	"fmt"
	"sync"
	"testing"
)

func TestMemoryStateHookSetGetAndOverwrite(t *testing.T) {
	t.Helper()

	hook := NewStateHook()

	if _, ok := hook.Get("missing"); ok {
		t.Fatal("expected missing key lookup to fail")
	}

	hook.Set("service", "alpha")
	hook.Set("service", "beta")

	value, ok := hook.Get("service")
	if !ok {
		t.Fatal("expected stored value to be present")
	}
	if got, want := value, "beta"; got != want {
		t.Fatalf("unexpected stored value: got %v want %v", got, want)
	}
}

func TestMemoryStateHookDoesNotExposeInternalMapStorage(t *testing.T) {
	t.Helper()

	hook := NewStateHook()

	original := map[string]any{
		"namespace": "s3",
		"meta": map[string]any{
			"bucket": "alpha",
		},
	}

	hook.Set("state", original)
	original["namespace"] = "mutated"
	original["meta"].(map[string]any)["bucket"] = "mutated"

	value, ok := hook.Get("state")
	if !ok {
		t.Fatal("expected stored map value to be present")
	}

	state := value.(map[string]any)
	if got, want := state["namespace"], "s3"; got != want {
		t.Fatalf("unexpected namespace after set mutation: got %v want %v", got, want)
	}

	state["namespace"] = "changed"
	state["meta"].(map[string]any)["bucket"] = "changed"

	again, ok := hook.Get("state")
	if !ok {
		t.Fatal("expected stored map value to remain present")
	}
	restored := again.(map[string]any)
	if got, want := restored["namespace"], "s3"; got != want {
		t.Fatalf("unexpected namespace after get mutation: got %v want %v", got, want)
	}
	if got, want := restored["meta"].(map[string]any)["bucket"], "alpha"; got != want {
		t.Fatalf("unexpected nested map value after get mutation: got %v want %v", got, want)
	}
}

func TestMemoryStateHookConcurrentAccess(t *testing.T) {
	t.Helper()

	hook := &MemoryStateHook{}

	const workers = 64
	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		i := i
		go func() {
			defer wg.Done()

			key := fmt.Sprintf("service-%d", i)
			hook.Set(key, map[string]any{
				"index": i,
			})

			value, ok := hook.Get(key)
			if !ok {
				t.Errorf("expected key %q to be present", key)
				return
			}

			state := value.(map[string]any)
			if got, want := state["index"], i; got != want {
				t.Errorf("unexpected stored index for %q: got %v want %v", key, got, want)
			}
		}()
	}

	wg.Wait()

	for i := 0; i < workers; i++ {
		key := fmt.Sprintf("service-%d", i)
		value, ok := hook.Get(key)
		if !ok {
			t.Fatalf("expected key %q to remain present", key)
		}
		state := value.(map[string]any)
		if got, want := state["index"], i; got != want {
			t.Fatalf("unexpected stored index for %q: got %v want %v", key, got, want)
		}
	}
}
