package cli

import (
	"context"

	"github.com/michasdev/mildstack/core/internal/application/runtime"
	"github.com/spf13/cobra"
)

func NewServeCommand(manager *runtime.Manager) *cobra.Command {
	var port int

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Register an API instance in the shared runtime",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return manager.Serve(context.Background(), port)
		},
	}
	cmd.Flags().IntVar(&port, "port", 8080, "API port")

	return cmd
}
