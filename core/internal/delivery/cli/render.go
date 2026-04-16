package cli

import (
	"fmt"
	"strings"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
)

func RenderStatus(theme Theme, presenter Presenter) string {
	var buf strings.Builder

	buf.WriteString(theme.TitleLabel)
	buf.WriteString("\n")
	fmt.Fprintf(&buf, "%s: %s\n\n", theme.StateLabel, presenter.PresentReadiness())

	renderSection(&buf, theme.ServicesLabel, theme.Indent, theme.EmptyLabel, renderServiceLines(presenter.Services()))
	buf.WriteString("\n")
	renderSection(&buf, theme.PortsLabel, theme.Indent, theme.EmptyLabel, renderPortLines(presenter.Ports()))

	return buf.String()
}

func RenderPorts(theme Theme, presenter Presenter) string {
	_ = theme
	return presenter.PresentPorts()
}

func RenderReadiness(theme Theme, presenter Presenter) string {
	return fmt.Sprintf("%s: %s", theme.StateLabel, presenter.PresentReadiness())
}

func RenderError(theme Theme, err error) string {
	_ = theme
	return PresentError(err)
}

func renderSection(buf *strings.Builder, title, indent, emptyLabel string, lines []string) {
	buf.WriteString(title)
	buf.WriteString("\n")
	if len(lines) == 0 {
		fmt.Fprintf(buf, "%s%s\n", indent, emptyLabel)
		return
	}

	for _, line := range lines {
		fmt.Fprintf(buf, "%s%s\n", indent, line)
	}
}

func renderServiceLines(services []orchestrator.Metadata) []string {
	if len(services) == 0 {
		return nil
	}

	lines := make([]string, len(services))
	for i, service := range services {
		if service.Version == "" {
			lines[i] = service.Name
			continue
		}
		lines[i] = fmt.Sprintf("%s %s", service.Name, service.Version)
	}
	return lines
}

func renderPortLines(ports []int) []string {
	if len(ports) == 0 {
		return nil
	}

	lines := make([]string, len(ports))
	for i, port := range ports {
		lines[i] = fmt.Sprintf("%d", port)
	}
	return lines
}
