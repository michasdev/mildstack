package cli

import (
	"context"
	"fmt"

	"github.com/michasdev/mildstack/core/internal/application/runtime"
	"github.com/spf13/cobra"
)

func NewStatusCommand(manager *runtime.Manager) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show the runtime snapshot",
		RunE: func(cmd *cobra.Command, _ []string) error {
			snapshot := manager.Snapshot(context.Background())
			out := cmd.OutOrStdout()

			fmt.Fprintln(out, "Services:")
			if len(snapshot.Services) == 0 {
				fmt.Fprintln(out, "  (none)")
			}
			for _, service := range snapshot.Services {
				fmt.Fprintf(out, "- %s %s\n", service.Name, service.Version)
			}

			fmt.Fprintln(out, "Ports:")
			if len(snapshot.Ports) == 0 {
				fmt.Fprintln(out, "  (none)")
			}
			for _, port := range snapshot.Ports {
				fmt.Fprintf(out, "- %d\n", port)
			}

			return nil
		},
	}

	return cmd
}
