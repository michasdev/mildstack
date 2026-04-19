package cli

import (
	"context"
	"fmt"

	"github.com/michasdev/mildstack/core/internal/application/runtime"
	"github.com/spf13/cobra"
)

// NewStatusCommand returns a thin alias of the instances command so that
// operator scripts targeting "status" continue to work without a separate
// rendering path that could drift from instances.
func NewStatusCommand(manager *runtime.Manager, storage Storage) *cobra.Command {
	cmd := NewInstancesCommand(manager, storage)
	cmd.Use = "status"
	cmd.Short = "Show the runtime snapshot (alias for instances)"
	return cmd
}

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
			snapshot.Instances = instancesToRuntime(instances, snapshot.Instances)
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

// instancesToRuntime converts storage summaries into runtime instances.
// If a storage summary lacks an InstanceID, the function falls back to the
// InstanceID carried by the corresponding live snapshot instance (matched by port)
// so that the canonical identity is always present when the manager knows it.
func instancesToRuntime(instances []instanceSummary, liveInstances []runtime.Instance) []runtime.Instance {
	// build a port -> InstanceID index from the live snapshot
	liveID := make(map[int]string, len(liveInstances))
	for _, live := range liveInstances {
		if live.InstanceID != "" {
			liveID[live.Port] = live.InstanceID
		}
	}

	copied := make([]runtime.Instance, len(instances))
	for i, instance := range instances {
		id := instance.InstanceID
		if id == "" {
			id = liveID[instance.Port]
		}
		copied[i] = runtime.Instance{
			InstanceID: id,
			Port:       instance.Port,
			PID:        instance.PID,
			Status:     instance.Status,
			Error:      instance.Error,
		}
	}
	return copied
}
