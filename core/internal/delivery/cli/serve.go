package cli

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/michasdev/mildstack/core/internal/application/runtime"
	"github.com/spf13/cobra"
)

type HTTPServer interface {
	Start(context.Context) error
}

type HTTPServerFactory func(port int) HTTPServer

const defaultServePort = 4566

var listenTCP = net.Listen

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
				start := func() error {
					return server.Start(cmd.Context())
				}
				if detach {
					started := make(chan error, 1)
					go func() {
						started <- start()
					}()
					return waitForDetachedServe(cmd.Context(), manager, resolvedPort, started)
				}
				return start()
			}
			return manager.Serve(cmd.Context(), resolvedPort)
		},
	}
	cmd.Flags().IntVar(&port, "port", defaultServePort, "API port")
	cmd.Flags().BoolVar(&detach, "detach", false, "Return after the instance has started")
	cmd.Flags().BoolVar(&detach, "d", false, "Alias for --detach")

	return cmd
}

func ensureServePortAvailable(port int) error {
	listener, err := listenTCP("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("serve: port %d is already in use: %w", port, err)
	}
	return listener.Close()
}

func waitForDetachedServe(ctx context.Context, manager *runtime.Manager, port int, result <-chan error) error {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		if containsPort(manager.Ports(ctx), port) {
			return nil
		}

		select {
		case err := <-result:
			return err
		case <-ticker.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
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

func containsPort(ports []int, port int) bool {
	for _, existing := range ports {
		if existing == port {
			return true
		}
	}
	return false
}
