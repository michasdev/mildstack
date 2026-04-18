package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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
	if got, want := len(entry.Routes), 62; got != want {
		t.Fatalf("unexpected route count: got %d want %d", got, want)
	}
	found := false
	for _, route := range entry.Routes {
		if route.Method == "GET" && route.Path == "/api/v1/runtime/services/s3/:bucket" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected list objects route to be registered")
	}
	found = false
	for _, route := range entry.Routes {
		if route.Method == "GET" && route.Path == "/api/v1/runtime/services/s3/:bucket?object-lock" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected object lock route to be registered")
	}
}

func TestRegisterNativeS3RoutesExposesAwsCompatibleSmokeSurface(t *testing.T) {
	t.Helper()

	root := composition.DefaultRoot("test-instance")
	manager := runtime.New(root.Services)
	router := deliveryhttp.NewRouter(deliveryhttp.DefaultConfig(), manager)

	if err := registerNativeS3Routes(router, root.Services); err != nil {
		t.Fatalf("register native s3 routes: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	router.Engine().ServeHTTP(recorder, request)

	if got, want := recorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected list buckets status: got %d want %d", got, want)
	}
	if !strings.Contains(recorder.Body.String(), "ListAllMyBucketsResult") {
		t.Fatalf("expected list buckets XML, got %q", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "mildstack-assets") {
		t.Fatalf("expected seeded bucket in list buckets response, got %q", recorder.Body.String())
	}

	createRecorder := httptest.NewRecorder()
	createRequest := httptest.NewRequest(http.MethodPut, "/native-smoke-bucket", nil)
	router.Engine().ServeHTTP(createRecorder, createRequest)
	if got, want := createRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected create bucket status: got %d want %d", got, want)
	}

	putRecorder := httptest.NewRecorder()
	putRequest := httptest.NewRequest(http.MethodPut, "/native-smoke-bucket/native.txt", strings.NewReader("native-mode smoke payload"))
	putRequest.Header.Set("Content-Type", "text/plain")
	router.Engine().ServeHTTP(putRecorder, putRequest)
	if got, want := putRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected put object status: got %d want %d", got, want)
	}

	getRecorder := httptest.NewRecorder()
	getRequest := httptest.NewRequest(http.MethodGet, "/native-smoke-bucket/native.txt", nil)
	router.Engine().ServeHTTP(getRecorder, getRequest)
	if got, want := getRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected get object status: got %d want %d", got, want)
	}
	if got, want := getRecorder.Body.String(), "native-mode smoke payload"; got != want {
		t.Fatalf("unexpected object body: got %q want %q", got, want)
	}

	deleteObjectRecorder := httptest.NewRecorder()
	deleteObjectRequest := httptest.NewRequest(http.MethodDelete, "/native-smoke-bucket/native.txt", nil)
	router.Engine().ServeHTTP(deleteObjectRecorder, deleteObjectRequest)
	if got, want := deleteObjectRecorder.Code, http.StatusNoContent; got != want {
		t.Fatalf("unexpected delete object status: got %d want %d", got, want)
	}

	deleteBucketRecorder := httptest.NewRecorder()
	deleteBucketRequest := httptest.NewRequest(http.MethodDelete, "/native-smoke-bucket", nil)
	router.Engine().ServeHTTP(deleteBucketRecorder, deleteBucketRequest)
	if got, want := deleteBucketRecorder.Code, http.StatusNoContent; got != want {
		t.Fatalf("unexpected delete bucket status: got %d want %d", got, want)
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

func TestInstanceRegistrarServeSkipsDuplicateLoadedPort(t *testing.T) {
	t.Helper()

	homeDir := t.TempDir()
	configDir := t.TempDir()
	paths := runtime.ResolvePathsFrom(homeDir, configDir)
	storage := cli.NewStorage(paths, runtime.LegacyBaseDirFrom(homeDir, configDir))
	manager := runtime.NewWithPorts(nil, []int{9090})
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
}
