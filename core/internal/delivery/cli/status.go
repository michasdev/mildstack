package cli

import (
	"context"
	"fmt"

	"github.com/michasdev/mildstack/core/internal/application/runtime"
	"github.com/spf13/cobra"
)

func NewInstancesCommand(manager *runtime.Manager, storage Storage) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "instances",
		Short: "Show the runtime snapshot",
		RunE: func(cmd *cobra.Command, _ []string) error {
			snapshot := manager.Snapshot(context.Background())
			instances, err := storage.LoadInstances()
			if err != nil {
				return err
			}
			snapshot.Instances = instancesToRuntime(instances)
			presenter := NewPresenter(snapshot)
			if resolveOutputMode(cmd) == OutputModeJSON {
				fmt.Fprint(cmd.OutOrStdout(), RenderStatusJSON(presenter))
				return nil
			}
			fmt.Fprint(cmd.OutOrStdout(), RenderStatus(DefaultTheme(), presenter))
			return nil
		},
	}

	return cmd
}

func instancesToRuntime(instances []instanceSummary) []runtime.Instance {
	copied := make([]runtime.Instance, len(instances))
	for i, instance := range instances {
		copied[i] = runtime.Instance{
			InstanceID: instance.InstanceID,
			Port:       instance.Port,
			PID:        instance.PID,
			Status:     instance.Status,
			Error:      instance.Error,
		}
	}
	return copied
}
