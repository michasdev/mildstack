package cli

import (
	"context"
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
	Stop      *cobra.Command
	Delete    *cobra.Command
}

func NewRootCommand(out, err io.Writer, commands Commands) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mildstack",
		Short: "Shared CLI binary for the MildStack core runtime",
		Long:  "mildstack is the shared binary entrypoint for the MildStack core runtime.",
	}
	cmd.SetOut(out)
	cmd.SetErr(err)
	cmd.PersistentFlags().Bool("json", false, "Render machine-readable JSON output")
	cmd.CompletionOptions.DisableDefaultCmd = true

	subcommands := []*cobra.Command{commands.Serve}
	if commands.Instances != nil {
		subcommands = append(subcommands, commands.Instances)
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
	return cmd.Execute()
}

func NewStopCommand(manager *runtime.Manager, storage Storage) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop a running instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			port, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("stop: invalid instance port %q", args[0])
			}

			instance, ok, err := lookupInstance(storage, port)
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("stop: instance on port %d not found", port)
			}
			if instance.Status != "running" {
				return fmt.Errorf("stop: instance on port %d is not running", port)
			}

			manager.RemovePort(port)
			return storage.DeleteActiveInstance(port)
		},
	}

	return cmd
}

func NewDeleteCommand(manager *runtime.Manager, storage Storage) *cobra.Command {
	var port int

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete an instance",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !cmd.Flags().Changed("port") {
				return fmt.Errorf("delete requires --port")
			}

			if _, ok, err := lookupInstance(storage, port); err != nil {
				return err
			} else if !ok {
				return fmt.Errorf("delete: instance on port %d not found", port)
			}

			manager.RemovePort(port)
			if err := storage.DeleteActiveInstance(port); err != nil {
				return err
			}
			return storage.DeleteSavedInstance(port)
		},
	}
	cmd.Flags().IntVar(&port, "port", 0, "instance port")

	return cmd
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
