package cli

import (
	"errors"
	"testing"
)

func TestRenderCommandError(t *testing.T) {
	t.Helper()

	if got, want := stripANSI(RenderCommandError(errors.New("boom"))), "✗ boom"; got != want {
		t.Fatalf("unexpected command error render:\n got %q\nwant %q", got, want)
	}
	if got, want := stripANSI(RenderCommandError(errors.New("   "))), "✗ unexpected error"; got != want {
		t.Fatalf("unexpected blank command error render:\n got %q\nwant %q", got, want)
	}
	if got, want := RenderCommandError(nil), ""; got != want {
		t.Fatalf("unexpected nil command error render:\n got %q\nwant %q", got, want)
	}
}
