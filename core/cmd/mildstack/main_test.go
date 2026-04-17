package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/michasdev/mildstack/core/internal/application/runtime"
	"github.com/michasdev/mildstack/core/internal/composition"
	"github.com/michasdev/mildstack/core/internal/delivery/cli"
	deliveryhttp "github.com/michasdev/mildstack/core/internal/delivery/http"
)

func TestRegisterServiceRoutesRegistersS3BeforeServing(t *testing.T) {
	t.Helper()

	root := composition.DefaultRoot("test-instance")
	manager := runtime.New(root.Services)
	router := deliveryhttp.NewRouter(deliveryhttp.DefaultConfig(), manager)

	if err := registerServiceRoutes(router.Registrar(), root.Services); err != nil {
		t.Fatalf("register service routes: %v", err)
	}

	entry, ok := router.Registrar().Service("s3")
	if !ok {
		t.Fatal("expected s3 service to be registered")
	}
	if got, want := len(entry.Routes), 48; got != want {
		t.Fatalf("unexpected route count: got %d want %d", got, want)
	}
	found := false
	for _, route := range entry.Routes {
		if route.Method == "GET" && route.Path == "/api/v1/runtime/services/s3/buckets/:bucket/objects" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected list objects route to be registered")
	}
	found = false
	for _, route := range entry.Routes {
		if route.Method == "GET" && route.Path == "/api/v1/runtime/services/s3/buckets/:bucket/object-lock" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected object lock route to be registered")
	}
}

func TestInstanceRegistrarPersistsAndReleasesActiveInstance(t *testing.T) {
	t.Helper()

	homeDir := t.TempDir()
	configDir := t.TempDir()
	paths := runtime.ResolvePathsFrom(homeDir, configDir)
	storage := cli.NewStorage(paths, runtime.LegacyBaseDirFrom(homeDir, configDir))
	manager := runtime.NewWithPorts(nil, nil)
	registrar := instanceRegistrar{manager: manager, storage: storage}

	if err := registrar.Serve(context.Background(), 9090); err != nil {
		t.Fatalf("serve: %v", err)
	}

	ports, err := storage.LoadActivePorts()
	if err != nil {
		t.Fatalf("load active ports: %v", err)
	}
	if len(ports) != 1 || ports[0] != 9090 {
		t.Fatalf("unexpected active ports: %#v", ports)
	}

	savedPath := filepath.Join(paths.InstancesDir, "saved", "9090.json")
	if _, err := os.Stat(savedPath); err != nil {
		t.Fatalf("expected saved instance file: %v", err)
	}

	if err := registrar.Release(context.Background(), 9090); err != nil {
		t.Fatalf("release: %v", err)
	}

	ports, err = storage.LoadActivePorts()
	if err != nil {
		t.Fatalf("load active ports after release: %v", err)
	}
	if len(ports) != 0 {
		t.Fatalf("expected no active ports after release, got %#v", ports)
	}

	if _, err := os.Stat(savedPath); err != nil {
		t.Fatalf("saved instance should remain available: %v", err)
	}
}
