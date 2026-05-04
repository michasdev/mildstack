package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestNewRootCommandRegistersSubcommandsInFixedOrder(t *testing.T) {
	t.Helper()

	cmd := NewRootCommand(&bytes.Buffer{}, &bytes.Buffer{}, Commands{
		Serve:     &cobra.Command{Use: "start"},
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

	for i, want := range []string{"start", "instances", "stop", "delete"} {
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

func TestExecuteRendersPrettyErrorWithoutUsage(t *testing.T) {
	t.Helper()

	originalArgs := os.Args
	t.Cleanup(func() { os.Args = originalArgs })
	os.Args = []string{"mildstack", "start"}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := Execute(context.Background(), stdout, stderr, Commands{
		Serve: &cobra.Command{
			Use: "start",
			RunE: func(*cobra.Command, []string) error {
				return errors.New("start: unable to find an available port starting at 4566")
			},
		},
	})
	if err == nil {
		t.Fatal("expected execute to return error")
	}
	if got, want := stripANSI(stderr.String()), "✗ start: unable to find an available port starting at 4566\n"; got != want {
		t.Fatalf("unexpected stderr output:\n got %q\nwant %q", got, want)
	}
	if strings.Contains(stripANSI(stderr.String()), "Usage:") {
		t.Fatalf("stderr should not include usage help, got %q", stripANSI(stderr.String()))
	}
	if got := stdout.String(); got != "" {
		t.Fatalf("expected empty stdout, got %q", got)
	}
}

func TestNewRootCommandRegistersStatusAlias(t *testing.T) {
	t.Helper()

	cmd := NewRootCommand(&bytes.Buffer{}, &bytes.Buffer{}, Commands{
		Serve:     &cobra.Command{Use: "start"},
		Instances: &cobra.Command{Use: "instances"},
		Stop:      &cobra.Command{Use: "stop"},
		Delete:    &cobra.Command{Use: "delete"},
		Status:    &cobra.Command{Use: "status"},
	})

	var found bool
	for _, sub := range cmd.Commands() {
		if sub.Use == "status" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected 'status' alias command to be registered in root")
	}
}
