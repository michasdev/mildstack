package cli

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

var commandErrorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444")).Bold(true)

func RenderCommandError(err error) string {
	if err == nil {
		return ""
	}

	message := strings.TrimSpace(err.Error())
	if message == "" {
		message = "unexpected error"
	}

	return commandErrorStyle.Render(fmt.Sprintf("✗ %s", message))
}
