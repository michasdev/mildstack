package http

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/michasdev/mildstack/core/internal/application/runtime"
	"github.com/michasdev/mildstack/core/internal/composition"
)

type registrarStub struct {
	portCh chan int
}

func (r *registrarStub) Serve(context.Context, int) error {
	return nil
}

func TestServerStartAndShutdown(t *testing.T) {
	t.Helper()

	manager := runtime.New(composition.Assemble(nil).Services)
	router := NewRouter(DefaultConfig(), manager)
	server := NewServer(manager, router, 0)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- server.Start(ctx)
	}()

	port := waitForRecordedPort(t, manager)
	if port == 0 {
		t.Fatal("expected the server to record a bound port")
	}

	waitForListener(t, port)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("server start returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not shut down in time")
	}
}

func waitForRecordedPort(t *testing.T, manager *runtime.Manager) int {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for {
		ports := manager.Ports(context.Background())
		if len(ports) > 0 {
			return ports[0]
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for recorded port")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func waitForListener(t *testing.T, port int) {
	t.Helper()

	client := &http.Client{Timeout: 50 * time.Millisecond}
	deadline := time.Now().Add(2 * time.Second)
	url := fmt.Sprintf("http://127.0.0.1:%d/api/v1/runtime", port)

	for {
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for listener on port %d: %v", port, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
