package ui

import "testing"

func TestRenderPaneIsDeterministicAfterANSIRemoval(t *testing.T) {
	t.Helper()

	got := stripANSI(renderPane("Services", true, []string{"> alpha", "  beta"}))
	if got != "Services\n> alpha\n  beta" {
		t.Fatalf("unexpected pane render:\n%s", got)
	}

	got = stripANSI(renderPane("Ports", false, []string{"  8080"}))
	if got != "Ports\n  8080" {
		t.Fatalf("unexpected pane render:\n%s", got)
	}
}
