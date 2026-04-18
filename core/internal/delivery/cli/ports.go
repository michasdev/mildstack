package cli

import (
	"fmt"

	"github.com/michasdev/mildstack/core/internal/application/runtime"
	"github.com/spf13/cobra"
)

func NewPortsCommand(manager *runtime.Manager, storage Storage) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ports",
		Short: "List active runtime ports",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ports, err := storage.LoadActivePorts()
			if err != nil {
				return err
			}
			presenter := NewPresenter(runtime.Snapshot{Ports: ports})
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
