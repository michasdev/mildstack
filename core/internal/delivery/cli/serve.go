package cli

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/michasdev/mildstack/core/internal/application/runtime"
	"github.com/spf13/cobra"
)

type HTTPServer interface {
	Start(context.Context) error
}

type HTTPServerFactory func(port int) HTTPServer

const defaultServePort = 4566
const detachedReadyFileEnv = "MILDSTACK_DETACHED_READY_FILE"

var listenTCP = net.Listen
var startDetachedServe = defaultStartDetachedServe

func NewServeCommand(manager *runtime.Manager, factories ...HTTPServerFactory) *cobra.Command {
	var port int
	var detach bool
	var factory HTTPServerFactory
	if len(factories) > 0 {
		factory = factories[0]
	}

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the shared HTTP runtime",
		RunE: func(cmd *cobra.Command, _ []string) error {
			resolvedPort := port
			if cmd.Flags().Changed("port") {
				if err := ensureServePortAvailable(resolvedPort); err != nil {
					return err
				}
			} else {
				var err error
				resolvedPort, err = pickServePort(defaultServePort)
				if err != nil {
					return err
				}
			}

			if factory != nil {
				server := factory(resolvedPort)
				if server == nil {
					return nil
				}
				start := func(ctx context.Context) error {
					return server.Start(ctx)
				}
				if detach {
					return startDetachedServe(cmd.Context(), resolvedPort, start)
				}
				return start(cmd.Context())
			}
			return manager.Serve(cmd.Context(), resolvedPort)
		},
	}
	cmd.Flags().IntVar(&port, "port", defaultServePort, "API port")
	cmd.Flags().BoolVar(&detach, "detach", false, "Return after the instance has started")
	cmd.Flags().BoolVar(&detach, "d", false, "Alias for --detach")

	return cmd
}

func defaultStartDetachedServe(ctx context.Context, port int, _ func(context.Context) error) error {
	if ctx == nil {
		ctx = context.Background()
	}

	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("serve: resolve executable for detached mode: %w", err)
	}

	readyFile, err := os.CreateTemp("", "mildstack-detached-*.ready")
	if err != nil {
		return fmt.Errorf("serve: create detached readiness file: %w", err)
	}
	readyPath := readyFile.Name()
	if err := readyFile.Close(); err != nil {
		_ = os.Remove(readyPath)
		return fmt.Errorf("serve: close detached readiness file: %w", err)
	}
	defer os.Remove(readyPath)

	cmd := exec.CommandContext(ctx, executable, "serve", "--port", strconv.Itoa(port))
	cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", detachedReadyFileEnv, readyPath))

	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("serve: open null device for detached mode: %w", err)
	}
	defer devNull.Close()
	cmd.Stdin = devNull
	cmd.Stdout = devNull
	cmd.Stderr = devNull

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("serve: start detached process: %w", err)
	}

	waitErr := make(chan error, 1)
	go func() {
		waitErr <- cmd.Wait()
	}()

	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()

	for {
		if _, err := os.Stat(readyPath); err == nil {
			return nil
		}

		select {
		case err := <-waitErr:
			if err == nil {
				return fmt.Errorf("serve: detached process exited before signaling readiness")
			}
			return fmt.Errorf("serve: detached process exited before signaling readiness: %w", err)
		case <-ticker.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func ensureServePortAvailable(port int) error {
	listener, err := listenTCP("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("serve: port %d is already in use: %w", port, err)
	}
	return listener.Close()
}

func pickServePort(startPort int) (int, error) {
	var lastErr error
	for candidate := startPort; candidate < startPort+3; candidate++ {
		listener, err := listenTCP("tcp", fmt.Sprintf(":%d", candidate))
		if err != nil {
			lastErr = err
			continue
		}

		_ = listener.Close()
		return candidate, nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no available ports starting at %d", startPort)
	}
	return 0, fmt.Errorf("serve: unable to find an available port starting at %d: %w", startPort, lastErr)
}
