package runtime

import (
	"reflect"
	"sync"
)

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
	if value == nil {
		return nil
	}

	cloned := cloneReflectValue(reflect.ValueOf(value))
	if !cloned.IsValid() {
		return nil
	}

	return cloned.Interface()
}

func cloneReflectValue(value reflect.Value) reflect.Value {
	if !value.IsValid() {
		return reflect.Value{}
	}

	switch value.Kind() {
	case reflect.Interface:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		cloned := cloneReflectValue(value.Elem())
		if !cloned.IsValid() {
			return reflect.Zero(value.Type())
		}
		out := reflect.New(value.Type()).Elem()
		out.Set(cloned)
		return out
	case reflect.Map:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		out := reflect.MakeMapWithSize(value.Type(), value.Len())
		for _, key := range value.MapKeys() {
			out.SetMapIndex(key, cloneReflectValue(value.MapIndex(key)))
		}
		return out
	case reflect.Slice:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		out := reflect.MakeSlice(value.Type(), value.Len(), value.Len())
		for i := 0; i < value.Len(); i++ {
			out.Index(i).Set(cloneReflectValue(value.Index(i)))
		}
		return out
	case reflect.Array:
		out := reflect.New(value.Type()).Elem()
		for i := 0; i < value.Len(); i++ {
			out.Index(i).Set(cloneReflectValue(value.Index(i)))
		}
		return out
	case reflect.Ptr:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		out := reflect.New(value.Elem().Type())
		out.Elem().Set(cloneReflectValue(value.Elem()))
		return out
	default:
		return value
	}
}
