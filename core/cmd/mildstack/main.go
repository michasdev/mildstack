package main

import (
	"context"
	"os"

	"github.com/michasdev/mildstack/core/internal/application/runtime"
	"github.com/michasdev/mildstack/core/internal/composition"
	"github.com/michasdev/mildstack/core/internal/delivery/cli"
)

func main() {
	root := composition.Assemble(nil)
	manager := runtime.New(root)
	commands := cli.Commands{
		Serve:  cli.NewServeCommand(manager),
		Status: cli.NewStatusCommand(manager),
		Ports:  cli.NewPortsCommand(manager),
	}

	if err := cli.Execute(context.Background(), os.Stdout, os.Stderr, commands); err != nil {
		os.Exit(1)
	}
}
