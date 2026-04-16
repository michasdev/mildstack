package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/michasdev/mildstack/core/internal/application/runtime"
	"github.com/spf13/cobra"
)

func NewUICommand(manager *runtime.Manager) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ui",
		Short: "Inspect the runtime state interactively",
		RunE: func(cmd *cobra.Command, _ []string) error {
			snapshot := manager.Snapshot(cmd.Context())
			program := tea.NewProgram(NewModel(snapshot), tea.WithInput(cmd.InOrStdin()), tea.WithOutput(cmd.OutOrStdout()))
			_, err := program.Run()
			return err
		},
	}

	return cmd
}
