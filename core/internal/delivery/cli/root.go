package cli

import (
	"context"
	"io"

	"github.com/spf13/cobra"
)

func init() {
	cobra.EnableCommandSorting = false
}

type Commands struct {
	Serve     *cobra.Command
	Instances *cobra.Command
	Status    *cobra.Command
	Ports     *cobra.Command
	UI        *cobra.Command
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

	subcommands := []*cobra.Command{commands.Serve}
	switch {
	case commands.Instances != nil:
		subcommands = append(subcommands, commands.Instances)
		if commands.Ports != nil {
			subcommands = append(subcommands, commands.Ports)
		}
	case commands.Status != nil:
		subcommands = append(subcommands, commands.Status)
		if commands.Ports != nil {
			subcommands = append(subcommands, commands.Ports)
		}
	case commands.Ports != nil:
		subcommands = append(subcommands, commands.Ports)
	}
	subcommands = append(subcommands, commands.UI)

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
