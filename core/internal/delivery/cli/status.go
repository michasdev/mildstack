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
			fmt.Fprint(cmd.OutOrStdout(), RenderStatus(DefaultTheme(), NewPresenter(snapshot)))
			return nil
		},
	}

	return cmd
}
