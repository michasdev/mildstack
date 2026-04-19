package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/application/runtime"
	"github.com/michasdev/mildstack/core/internal/composition"
	"github.com/michasdev/mildstack/core/internal/delivery/cli"
	deliveryhttp "github.com/michasdev/mildstack/core/internal/delivery/http"
)

func main() {
	instanceID := resolveInstanceID()
	root := composition.DefaultRoot(instanceID)
	paths := runtime.ResolvePaths()
	homeDir, _ := os.UserHomeDir()
	configDir, _ := os.UserConfigDir()
	storage := cli.NewStorage(paths, runtime.LegacyBaseDirFrom(homeDir, configDir))
	activePorts, err := storage.LoadActivePorts()
	if err != nil {
		panic(err)
	}
	manager := runtime.NewWithPorts(root.Services, activePorts)
	manager.SetInstanceID(instanceID)
	httpServerFactory := func(port int) cli.HTTPServer {
		router := deliveryhttp.NewRouter(deliveryhttp.DefaultConfig(), manager)
		if err := registerServiceRoutes(router.Registrar(), root.Services); err != nil {
			return recordingHTTPServer{server: failedHTTPServer{err: err}, storage: storage, port: port}
		}
		if err := registerNativeDynamoDBRoutes(router, root.Services); err != nil {
			return recordingHTTPServer{server: failedHTTPServer{err: err}, storage: storage, port: port}
		}
		if err := registerNativeS3Routes(router, root.Services); err != nil {
			return recordingHTTPServer{server: failedHTTPServer{err: err}, storage: storage, port: port}
		}
		return recordingHTTPServer{server: deliveryhttp.NewServer(instanceRegistrar{manager: manager, storage: storage}, router, port), storage: storage, port: port}
	}
	commands := cli.Commands{
		Serve:     cli.NewServeCommand(manager, httpServerFactory),
		Instances: cli.NewInstancesCommand(manager, storage),
		Status:    cli.NewStatusCommand(manager, storage),
		Stop:      cli.NewStopCommand(manager, storage),
		Delete:    cli.NewDeleteCommand(manager, storage),
	}

	if err := cli.Execute(context.Background(), os.Stdout, os.Stderr, commands); err != nil {
		os.Exit(1)
	}
}

func resolveInstanceID() string {
	return strings.TrimSpace(os.Getenv("MILDSTACK_INSTANCE_ID"))
}

type instanceRegistrar struct {
	manager *runtime.Manager
	storage cli.Storage
}

func (r instanceRegistrar) Serve(ctx context.Context, port int) error {
	if !containsPort(r.manager.Ports(ctx), port) {
		if err := r.manager.Serve(ctx, port); err != nil {
			return err
		}
	}
	if err := r.storage.SaveSavedInstance(port); err != nil {
		return err
	}
	if err := r.storage.SaveActiveInstance(port); err != nil {
		return err
	}
	if err := signalDetachedReady(port); err != nil {
		return err
	}
	return nil
}

func (r instanceRegistrar) Release(_ context.Context, port int) error {
	r.manager.RemovePort(port)
	return r.storage.DeleteActiveInstance(port)
}

type failedHTTPServer struct {
	err error
}

func (s failedHTTPServer) Start(context.Context) error {
	return s.err
}

type recordingHTTPServer struct {
	server  cli.HTTPServer
	storage cli.Storage
	port    int
}

func (s recordingHTTPServer) Start(ctx context.Context) error {
	err := s.server.Start(ctx)
	if err != nil {
		if recordErr := s.storage.SaveErroredInstance(s.port, err); recordErr != nil {
			return fmt.Errorf("%w (and failed to persist errored instance: %v)", err, recordErr)
		}
	}
	return err
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

func registerNativeS3Routes(router *deliveryhttp.Router, services []orchestrator.Service) error {
	if router == nil {
		return nil
	}

	for _, service := range services {
		if service == nil || service.Metadata().Name != "s3" {
			continue
		}

		s3Service, ok := service.(deliveryhttp.S3NativeService)
		if !ok {
			return fmt.Errorf("s3 service does not expose the native http surface")
		}
		deliveryhttp.RegisterS3NativeRoutes(router.Engine(), s3Service)
		return nil
	}

	return nil
}

func registerNativeDynamoDBRoutes(router *deliveryhttp.Router, services []orchestrator.Service) error {
	if router == nil {
		return nil
	}

	for _, service := range services {
		if service == nil || service.Metadata().Name != "dynamodb" {
			continue
		}

		dynamoDBService, ok := service.(deliveryhttp.DynamoDBNativeService)
		if !ok {
			return fmt.Errorf("dynamodb service does not expose the native http surface")
		}
		deliveryhttp.RegisterDynamoDBNativeRoutes(router.Engine(), dynamoDBService)
		return nil
	}

	return nil
}

func containsPort(ports []int, port int) bool {
	for _, existing := range ports {
		if existing == port {
			return true
		}
	}
	return false
}

func signalDetachedReady(port int) error {
	readyFile := strings.TrimSpace(os.Getenv("MILDSTACK_DETACHED_READY_FILE"))
	if readyFile == "" {
		return nil
	}

	payload := fmt.Sprintf("%d\n", port)
	return os.WriteFile(readyFile, []byte(payload), 0o600)
}
