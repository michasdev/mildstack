package runtime

import "sync"

// MemoryStateHook is a mutex-backed in-memory implementation of the shared
// service state hook contract.
type MemoryStateHook struct {
	mu     sync.RWMutex
	values map[string]any
}

// NewStateHook returns the preferred shared in-memory hook instance.
func NewStateHook() *MemoryStateHook {
	return &MemoryStateHook{}
}

// Set stores a value for the provided key, replacing any previous value.
func (h *MemoryStateHook) Set(key string, value any) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.values == nil {
		h.values = make(map[string]any)
	}

	h.values[key] = cloneStateValue(value)
}

// Get returns a copy of the stored value and reports whether the key exists.
func (h *MemoryStateHook) Get(key string) (any, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	value, ok := h.values[key]
	if !ok {
		return nil, false
	}

	return cloneStateValue(value), true
}

func cloneStateValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		copied := make(map[string]any, len(typed))
		for key, item := range typed {
			copied[key] = cloneStateValue(item)
		}
		return copied
	case []any:
		copied := make([]any, len(typed))
		for i, item := range typed {
			copied[i] = cloneStateValue(item)
		}
		return copied
	default:
		return value
	}
}
