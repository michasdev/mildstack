package main

import (
	"context"
	"os"

	"github.com/michasdev/mildstack/core/internal/application/runtime"
	"github.com/michasdev/mildstack/core/internal/composition"
	"github.com/michasdev/mildstack/core/internal/delivery/cli"
	deliveryhttp "github.com/michasdev/mildstack/core/internal/delivery/http"
)

func main() {
	root := composition.Assemble(nil)
	manager := runtime.New(root)
	httpServerFactory := func(port int) cli.HTTPServer {
		router := deliveryhttp.NewRouter(deliveryhttp.DefaultConfig(), manager)
		return deliveryhttp.NewServer(manager, router, port)
	}
	commands := cli.Commands{
		Serve:  cli.NewServeCommand(manager, httpServerFactory),
		Status: cli.NewStatusCommand(manager),
		Ports:  cli.NewPortsCommand(manager),
	}

	if err := cli.Execute(context.Background(), os.Stdout, os.Stderr, commands); err != nil {
		os.Exit(1)
	}
}
