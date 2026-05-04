package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/michasdev/mildstack/core/internal/application/runtime"
	"github.com/spf13/cobra"
)

func init() {
	cobra.EnableCommandSorting = false
}

type Commands struct {
	Serve     *cobra.Command
	Instances *cobra.Command
	Status    *cobra.Command
	Stop      *cobra.Command
	Delete    *cobra.Command
}

func NewRootCommand(out, err io.Writer, commands Commands) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mildstack",
		Short: "Shared CLI binary for the MildStack core runtime",
		Long:  "mildstack is the shared binary entrypoint for the MildStack core runtime.",
	}
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetOut(out)
	cmd.SetErr(err)
	cmd.PersistentFlags().Bool("json", false, "Render machine-readable JSON output")
	cmd.CompletionOptions.DisableDefaultCmd = true

	subcommands := []*cobra.Command{commands.Serve}
	if commands.Instances != nil {
		subcommands = append(subcommands, commands.Instances)
	}
	if commands.Status != nil {
		subcommands = append(subcommands, commands.Status)
	}
	if commands.Stop != nil {
		subcommands = append(subcommands, commands.Stop)
	}
	if commands.Delete != nil {
		subcommands = append(subcommands, commands.Delete)
	}

	for _, subcommand := range subcommands {
		if subcommand != nil {
			cmd.AddCommand(subcommand)
		}
	}

	return cmd
}

func Execute(ctx context.Context, out, err io.Writer, commands Commands) error {
	cmd := NewRootCommand(out, err, commands)
	cmd.SetContext(ctx)
	if executeErr := cmd.Execute(); executeErr != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), RenderCommandError(executeErr))
		return executeErr
	}
	return nil
}

func NewStopCommand(manager *runtime.Manager, storage Storage) *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "stop [port]",
		Short: "Stop a running instance",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targets, err := resolveLifecycleTargets(storage, lifecycleSelection{
				args:    args,
				all:     all,
				running: true,
			})
			if err != nil {
				return err
			}
			return stopInstances(manager, storage, targets)
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "Stop every running instance")

	return cmd
}

func NewDeleteCommand(manager *runtime.Manager, storage Storage) *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "delete [port]",
		Short: "Delete an instance",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targets, err := resolveLifecycleTargets(storage, lifecycleSelection{
				args: args,
				all:  all,
			})
			if err != nil {
				return err
			}
			return deleteInstances(manager, storage, targets)
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "Delete every instance and its resources")

	return cmd
}

type lifecycleSelection struct {
	args    []string
	all     bool
	running bool
}

func resolveLifecycleTargets(storage Storage, selection lifecycleSelection) ([]instanceSummary, error) {
	if selection.all && len(selection.args) > 0 {
		if selection.running {
			return nil, fmt.Errorf("stop: --all cannot be combined with a port")
		}
		return nil, fmt.Errorf("delete: --all cannot be combined with a port")
	}

	if selection.all {
		instances, err := storage.LoadInstances()
		if err != nil {
			return nil, err
		}
		if selection.running {
			return filterRunningInstances(instances), nil
		}
		return instances, nil
	}

	if len(selection.args) == 0 {
		if selection.running {
			return nil, fmt.Errorf("stop requires a port or --all")
		}
		return nil, fmt.Errorf("delete requires a port or --all")
	}

	port, err := strconv.Atoi(selection.args[0])
	if err != nil {
		if selection.running {
			return nil, fmt.Errorf("stop: invalid instance port %q", selection.args[0])
		}
		return nil, fmt.Errorf("delete: invalid instance port %q", selection.args[0])
	}

	instance, ok, err := lookupInstance(storage, port)
	if err != nil {
		return nil, err
	}
	if !ok {
		if selection.running {
			return nil, fmt.Errorf("stop: instance on port %d not found", port)
		}
		return nil, fmt.Errorf("delete: instance on port %d not found", port)
	}
	if selection.running && instance.Status != "running" {
		return nil, fmt.Errorf("stop: instance on port %d is not running", port)
	}

	return []instanceSummary{instance}, nil
}

func filterRunningInstances(instances []instanceSummary) []instanceSummary {
	targets := make([]instanceSummary, 0, len(instances))
	for _, instance := range instances {
		if instance.Status == "running" {
			targets = append(targets, instance)
		}
	}
	return targets
}

func stopInstances(manager *runtime.Manager, storage Storage, targets []instanceSummary) error {
	var errs []error
	for _, instance := range targets {
		if err := terminateProcessFn(instance.PID); err != nil {
			errs = append(errs, fmt.Errorf("stop: terminate instance on port %d: %w", instance.Port, err))
			continue
		}

		manager.RemovePort(instance.Port)
		if err := storage.DeleteActiveInstance(instance); err != nil {
			errs = append(errs, fmt.Errorf("stop: delete active instance on port %d: %w", instance.Port, err))
		}
	}

	return errors.Join(errs...)
}

func deleteInstances(manager *runtime.Manager, storage Storage, targets []instanceSummary) error {
	var errs []error
	for _, instance := range targets {
		if instance.Status == "running" {
			if err := terminateProcessFn(instance.PID); err != nil {
				errs = append(errs, fmt.Errorf("delete: terminate instance on port %d: %w", instance.Port, err))
				continue
			}
		}

		manager.RemovePort(instance.Port)
		if err := storage.DeleteActiveInstance(instance); err != nil {
			errs = append(errs, fmt.Errorf("delete: delete active instance on port %d: %w", instance.Port, err))
		}
		if err := storage.DeleteSavedInstance(instance); err != nil {
			errs = append(errs, fmt.Errorf("delete: delete saved instance on port %d: %w", instance.Port, err))
		}
		if err := storage.DeleteInstanceResources(instance.InstanceID); err != nil {
			errs = append(errs, fmt.Errorf("delete: delete resources for instance %s: %w", instance.InstanceID, err))
		}
	}

	return errors.Join(errs...)
}

func lookupInstance(storage Storage, port int) (instanceSummary, bool, error) {
	instances, err := storage.LoadInstances()
	if err != nil {
		return instanceSummary{}, false, err
	}
	for _, instance := range instances {
		if instance.Port == port {
			return instance, true, nil
		}
	}
	return instanceSummary{}, false, nil
}
