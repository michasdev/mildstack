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
			presenter := NewPresenter(runtime.Snapshot{Ports: manager.Ports(context.Background())})
			if resolveOutputMode(cmd) == OutputModeJSON {
				fmt.Fprint(cmd.OutOrStdout(), RenderPortsJSON(presenter))
				return nil
			}
			fmt.Fprint(cmd.OutOrStdout(), RenderPorts(DefaultTheme(), presenter))
			return nil
		},
	}

	return cmd
}
