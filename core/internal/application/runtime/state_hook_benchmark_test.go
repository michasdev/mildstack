package runtime

import "testing"

var benchmarkStateSink any

func BenchmarkMemoryStateHook(b *testing.B) {
	payload := map[string]any{
		"service": "s3",
		"nested": map[string]any{
			"bucket": "alpha",
		},
		"tags": []any{"core", "example"},
	}

	b.Run("Set", func(b *testing.B) {
		hook := NewStateHook()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			hook.Set("services/s3", payload)
		}
	})

	b.Run("Get", func(b *testing.B) {
		hook := NewStateHook()
		hook.Set("services/s3", payload)
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			benchmarkStateSink, _ = hook.Get("services/s3")
		}
	})
}
