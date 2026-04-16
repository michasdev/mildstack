package main

import (
	"context"
	"os"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/application/runtime"
	"github.com/michasdev/mildstack/core/internal/composition"
	"github.com/michasdev/mildstack/core/internal/delivery/cli"
	cliui "github.com/michasdev/mildstack/core/internal/delivery/cli/ui"
	deliveryhttp "github.com/michasdev/mildstack/core/internal/delivery/http"
)

func main() {
	root := composition.DefaultRoot()
	manager := runtime.New(root.Services)
	httpServerFactory := func(port int) cli.HTTPServer {
		router := deliveryhttp.NewRouter(deliveryhttp.DefaultConfig(), manager)
		if err := registerServiceRoutes(router.Registrar(), root.Services); err != nil {
			return failedHTTPServer{err: err}
		}
		return deliveryhttp.NewServer(manager, router, port)
	}
	commands := cli.Commands{
		Serve:  cli.NewServeCommand(manager, httpServerFactory),
		Status: cli.NewStatusCommand(manager),
		Ports:  cli.NewPortsCommand(manager),
		UI:     cliui.NewUICommand(manager),
	}

	if err := cli.Execute(context.Background(), os.Stdout, os.Stderr, commands); err != nil {
		os.Exit(1)
	}
}

type failedHTTPServer struct {
	err error
}

func (s failedHTTPServer) Start(context.Context) error {
	return s.err
}

func registerServiceRoutes(registrar orchestrator.RouteRegistrar, services []orchestrator.Service) error {
	for _, service := range services {
		if service == nil {
			continue
		}
		if err := service.RegisterRoutes(registrar); err != nil {
			return err
		}
	}
	return nil
}
