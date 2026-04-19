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
	paths := runtime.ResolvePaths()
	homeDir, _ := os.UserHomeDir()
	configDir, _ := os.UserConfigDir()
	storage := cli.NewStorage(paths, runtime.LegacyBaseDirFrom(homeDir, configDir))
	activePorts, err := storage.LoadActivePorts()
	if err != nil {
		panic(err)
	}
	// Manager starts with no services; services are wired per-serve call once
	// the port (and therefore the instanceId) is known.
	manager := runtime.NewWithPorts(nil, activePorts)

	httpServerFactory := func(port int) cli.HTTPServer {
		instanceID, err := storage.ResolveInstanceIDForPort(port)
		if err != nil {
			return recordingHTTPServer{server: failedHTTPServer{err: fmt.Errorf("resolve instance id: %w", err)}, storage: storage, port: port}
		}
		root := composition.DefaultRoot(instanceID)
		manager.SetInstanceID(instanceID)
		manager.SetServices(root.Services)

		router := deliveryhttp.NewRouter(deliveryhttp.DefaultConfig(), manager)
		if err := registerServiceRoutes(router.Registrar(), root.Services); err != nil {
			return recordingHTTPServer{server: failedHTTPServer{err: err}, storage: storage, port: port, instanceID: instanceID}
		}
		if err := registerNativeDynamoDBRoutes(router, root.Services); err != nil {
			return recordingHTTPServer{server: failedHTTPServer{err: err}, storage: storage, port: port, instanceID: instanceID}
		}
		if err := registerNativeS3Routes(router, root.Services); err != nil {
			return recordingHTTPServer{server: failedHTTPServer{err: err}, storage: storage, port: port, instanceID: instanceID}
		}
		registrar := instanceRegistrar{manager: manager, storage: storage, instanceID: instanceID}
		return recordingHTTPServer{server: deliveryhttp.NewServer(registrar, router, port), storage: storage, port: port, instanceID: instanceID}
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

type instanceRegistrar struct {
	manager    *runtime.Manager
	storage    cli.Storage
	instanceID string
}

func (r instanceRegistrar) Serve(ctx context.Context, port int) error {
	if !containsPort(r.manager.Ports(ctx), port) {
		if err := r.manager.Serve(ctx, port); err != nil {
			return err
		}
	}
	if err := r.storage.SaveSavedInstanceWithID(r.instanceID, port); err != nil {
		return err
	}
	if err := r.storage.SaveActiveInstanceWithID(r.instanceID, port); err != nil {
		return err
	}
	if err := signalDetachedReady(port); err != nil {
		return err
	}
	return nil
}

func (r instanceRegistrar) Release(_ context.Context, port int) error {
	instanceID := strings.TrimSpace(r.instanceID)
	if instanceID == "" {
		var err error
		instanceID, err = r.storage.ResolveInstanceIDForPort(port)
		if err != nil {
			return err
		}
	}

	r.manager.RemovePort(port)
	return r.storage.DeleteActiveInstanceByIdentity(instanceID, port)
}

type failedHTTPServer struct {
	err error
}

func (s failedHTTPServer) Start(context.Context) error {
	return s.err
}

type recordingHTTPServer struct {
	server     cli.HTTPServer
	storage    cli.Storage
	port       int
	instanceID string
}

func (s recordingHTTPServer) Start(ctx context.Context) error {
	err := s.server.Start(ctx)
	if err != nil {
		if recordErr := s.storage.SaveErroredInstanceWithID(s.instanceID, s.port, err); recordErr != nil {
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
