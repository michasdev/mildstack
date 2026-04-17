package cli

import (
	"fmt"
	"strings"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
)

func RenderStatus(theme Theme, presenter Presenter) string {
	var buf strings.Builder

	buf.WriteString(theme.TitleStyle.Render(theme.TitleLabel))
	buf.WriteString("\n")
	fmt.Fprintf(&buf, "%s: %s\n\n", theme.LabelStyle.Render(theme.StateLabel), theme.AccentStyle.Render(presenter.PresentReadiness()))

	renderSection(&buf, theme.SectionStyle.Render(theme.ServicesLabel), theme.Indent, theme.EmptyStyle.Render(theme.EmptyLabel), renderServiceLines(presenter.Services()))
	buf.WriteString("\n")
	renderSection(&buf, theme.SectionStyle.Render(theme.PortsLabel), theme.Indent, theme.EmptyStyle.Render(theme.EmptyLabel), renderPortLines(presenter.Ports()))

	return buf.String()
}

func RenderStatusJSON(presenter Presenter) string {
	return renderJSON(presenter.StatusPayload())
}

func RenderPorts(theme Theme, presenter Presenter) string {
	ports := presenter.Ports()
	if len(ports) == 0 {
		return theme.EmptyStyle.Render("No ports registered") + "\n"
	}

	var buf strings.Builder
	for _, port := range ports {
		fmt.Fprintf(&buf, "%s\n", theme.AccentStyle.Render(fmt.Sprintf("%d", port)))
	}
	return buf.String()
}

func RenderPortsJSON(presenter Presenter) string {
	return renderJSON(presenter.PortsPayload())
}

func RenderReadiness(theme Theme, presenter Presenter) string {
	return fmt.Sprintf("%s: %s", theme.LabelStyle.Render(theme.StateLabel), theme.AccentStyle.Render(presenter.PresentReadiness()))
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
