package main

import (
	"context"
	"os"

	"github.com/michasdev/mildstack/core/internal/delivery/cli"
)

func main() {
	if err := cli.Execute(context.Background(), os.Stdout, os.Stderr, cli.Commands{}); err != nil {
		os.Exit(1)
	}
}
