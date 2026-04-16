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
	Serve  *cobra.Command
	Status *cobra.Command
	Ports  *cobra.Command
}

func NewRootCommand(out, err io.Writer, commands Commands) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mildstack",
		Short: "Shared CLI binary for the MildStack core runtime",
		Long:  "mildstack is the shared binary entrypoint for the MildStack core runtime.",
	}
	cmd.SetOut(out)
	cmd.SetErr(err)

	for _, subcommand := range []*cobra.Command{commands.Serve, commands.Status, commands.Ports} {
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
