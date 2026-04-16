package cli

import (
	"context"

	"github.com/michasdev/mildstack/core/internal/application/runtime"
	"github.com/spf13/cobra"
)

func NewPortsCommand(manager *runtime.Manager) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ports",
		Short: "List active runtime ports",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ports := manager.Ports(context.Background())
			cmd.Print(RenderPorts(DefaultTheme(), NewPresenter(runtime.Snapshot{Ports: ports})))
			return nil
		},
	}

	return cmd
}
