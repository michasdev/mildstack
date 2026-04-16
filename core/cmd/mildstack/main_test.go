package main

import (
	"testing"

	"github.com/michasdev/mildstack/core/internal/application/runtime"
	"github.com/michasdev/mildstack/core/internal/composition"
	deliveryhttp "github.com/michasdev/mildstack/core/internal/delivery/http"
)

func TestRegisterServiceRoutesRegistersS3BeforeServing(t *testing.T) {
	t.Helper()

	root := composition.DefaultRoot()
	manager := runtime.New(root.Services)
	router := deliveryhttp.NewRouter(deliveryhttp.DefaultConfig(), manager)

	if err := registerServiceRoutes(router.Registrar(), root.Services); err != nil {
		t.Fatalf("register service routes: %v", err)
	}

	entry, ok := router.Registrar().Service("s3")
	if !ok {
		t.Fatal("expected s3 service to be registered")
	}
	if got, want := len(entry.Routes), 4; got != want {
		t.Fatalf("unexpected route count: got %d want %d", got, want)
	}
	if got, want := entry.Routes[2].Path, "/api/v1/runtime/services/s3/buckets/:bucket/objects"; got != want {
		t.Fatalf("unexpected object route path: got %q want %q", got, want)
	}
}
