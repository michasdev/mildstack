package cli

import (
	"bytes"
	"context"
	"testing"

	"github.com/spf13/cobra"
)

func TestNewRootCommandRegistersSubcommandsInFixedOrder(t *testing.T) {
	t.Helper()

	cmd := NewRootCommand(&bytes.Buffer{}, &bytes.Buffer{}, Commands{
		Serve:     &cobra.Command{Use: "serve"},
		Instances: &cobra.Command{Use: "instances"},
		Stop:      &cobra.Command{Use: "stop"},
		Delete:    &cobra.Command{Use: "delete"},
	})

	if got, want := cmd.Use, "mildstack"; got != want {
		t.Fatalf("unexpected root use: got %q want %q", got, want)
	}

	subcommands := cmd.Commands()
	if len(subcommands) != 4 {
		t.Fatalf("expected 4 subcommands, got %d", len(subcommands))
	}

	for i, want := range []string{"serve", "instances", "stop", "delete"} {
		if got := subcommands[i].Use; got != want {
			t.Fatalf("unexpected subcommand at %d: got %q want %q", i, got, want)
		}
	}

	for _, subcommand := range subcommands {
		if subcommand.Use == "completion" {
			t.Fatal("unexpected completion command")
		}
	}
}

func TestExecuteWiresContextAndRootCommand(t *testing.T) {
	t.Helper()

	if err := Execute(context.Background(), &bytes.Buffer{}, &bytes.Buffer{}, Commands{}); err != nil {
		t.Fatalf("execute: %v", err)
	}
}
