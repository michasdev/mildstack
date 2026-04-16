package cli

import (
	"context"

	"github.com/michasdev/mildstack/core/internal/application/runtime"
	"github.com/spf13/cobra"
)

type HTTPServer interface {
	Start(context.Context) error
}

type HTTPServerFactory func(port int) HTTPServer

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
			if factory != nil {
				server := factory(port)
				if server == nil {
					return nil
				}
				return server.Start(cmd.Context())
			}
			return manager.Serve(cmd.Context(), port)
		},
	}
	cmd.Flags().IntVar(&port, "port", 8080, "API port")

	return cmd
}
