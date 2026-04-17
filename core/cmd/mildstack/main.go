package main

import (
	"context"
	"os"
	"strings"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/application/runtime"
	"github.com/michasdev/mildstack/core/internal/composition"
	"github.com/michasdev/mildstack/core/internal/delivery/cli"
	cliui "github.com/michasdev/mildstack/core/internal/delivery/cli/ui"
	deliveryhttp "github.com/michasdev/mildstack/core/internal/delivery/http"
)

func main() {
	root := composition.DefaultRoot(resolveInstanceID())
	paths := runtime.ResolvePaths()
	homeDir, _ := os.UserHomeDir()
	configDir, _ := os.UserConfigDir()
	storage := cli.NewStorage(paths, runtime.LegacyBaseDirFrom(homeDir, configDir))
	activePorts, err := storage.LoadActivePorts()
	if err != nil {
		panic(err)
	}
	manager := runtime.NewWithPorts(root.Services, activePorts)
	httpServerFactory := func(port int) cli.HTTPServer {
		router := deliveryhttp.NewRouter(deliveryhttp.DefaultConfig(), manager)
		if err := registerServiceRoutes(router.Registrar(), root.Services); err != nil {
			return failedHTTPServer{err: err}
		}
		return deliveryhttp.NewServer(instanceRegistrar{manager: manager, storage: storage}, router, port)
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

const defaultInstanceID = "default"

func resolveInstanceID() string {
	instanceID := strings.TrimSpace(os.Getenv("MILDSTACK_INSTANCE_ID"))
	if instanceID == "" {
		return defaultInstanceID
	}
	return instanceID
}

type instanceRegistrar struct {
	manager *runtime.Manager
	storage cli.Storage
}

func (r instanceRegistrar) Serve(ctx context.Context, port int) error {
	if err := r.manager.Serve(ctx, port); err != nil {
		return err
	}
	if err := r.storage.SaveSavedInstance(port); err != nil {
		return err
	}
	if err := r.storage.SaveActiveInstance(port); err != nil {
		return err
	}
	return nil
}

func (r instanceRegistrar) Release(_ context.Context, port int) error {
	return r.storage.DeleteActiveInstance(port)
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
