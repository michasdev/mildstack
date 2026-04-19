package ui

import (
	"strings"

	charmgloss "charm.land/lipgloss/v2"
)

var (
	violet        = charmgloss.Color("#8b5cf6")
	softViolet    = charmgloss.Color("#a78bfa")
	muted         = charmgloss.Color("#9ca3af")
	titleStyle    = charmgloss.NewStyle().Bold(true).Foreground(violet)
	helperStyle   = charmgloss.NewStyle().Foreground(muted)
	sectionStyle  = charmgloss.NewStyle().Bold(true).Foreground(softViolet)
	selectedStyle = charmgloss.NewStyle().Bold(true).Foreground(violet)
	detailStyle   = charmgloss.NewStyle().Border(charmgloss.NormalBorder()).BorderForeground(violet).Padding(0, 1)
)

func renderPane(title string, focused bool, lines []string) string {
	label := sectionStyle.Render(title)
	if focused {
		label = selectedStyle.Render(title)
	}

	return strings.Join(append([]string{label}, lines...), "\n")
}
