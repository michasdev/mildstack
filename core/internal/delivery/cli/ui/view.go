package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true)
	helperStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	sectionStyle  = lipgloss.NewStyle().Bold(true)
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	detailStyle   = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(0, 1)
)

func renderPane(title string, focused bool, lines []string) string {
	label := sectionStyle.Render(title)
	if focused {
		label = selectedStyle.Render(title)
	}

	return strings.Join(append([]string{label}, lines...), "\n")
}
