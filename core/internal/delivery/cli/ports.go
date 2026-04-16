package cli

import (
	"context"
	"fmt"

	"github.com/michasdev/mildstack/core/internal/application/runtime"
	"github.com/spf13/cobra"
)

func NewPortsCommand(manager *runtime.Manager) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ports",
		Short: "List active runtime ports",
		RunE: func(cmd *cobra.Command, _ []string) error {
			for _, port := range manager.Ports(context.Background()) {
				fmt.Fprintln(cmd.OutOrStdout(), port)
			}
			return nil
		},
	}

	return cmd
}
