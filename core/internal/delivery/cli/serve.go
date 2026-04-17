package cli

import (
	"context"
	"fmt"
	"net"

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
	var factory HTTPServerFactory
	if len(factories) > 0 {
		factory = factories[0]
	}

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the shared HTTP runtime",
		RunE: func(cmd *cobra.Command, _ []string) error {
			resolvedPort := port
			if !cmd.Flags().Changed("port") {
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
				return server.Start(cmd.Context())
			}
			return manager.Serve(cmd.Context(), resolvedPort)
		},
	}
	cmd.Flags().IntVar(&port, "port", defaultServePort, "API port")

	return cmd
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
