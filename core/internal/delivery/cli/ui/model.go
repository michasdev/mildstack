package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/application/runtime"
)

type focus int

const (
	focusServices focus = iota
	focusPorts
)

type model struct {
	snapshot     runtime.Snapshot
	focus        focus
	serviceIndex int
	portIndex    int
	detail       string
	quitting     bool
}

func NewModel(snapshot runtime.Snapshot) model {
	return model{
		snapshot: cloneSnapshot(snapshot),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "tab":
			m.focus = toggleFocus(m.focus)
			m.detail = ""
		case "shift+tab":
			m.focus = toggleFocus(m.focus)
			m.detail = ""
		case "up", "k":
			m.stepSelection(-1)
			m.detail = ""
		case "down", "j":
			m.stepSelection(1)
			m.detail = ""
		case "enter":
			m.detail = m.selectedDetail()
		case "esc", "backspace":
			if m.detail != "" {
				m.detail = ""
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m model) View() string {
	sections := []string{
		titleStyle.Render("MildStack UI"),
		helperStyle.Render("Tab switches focus, Enter inspects, Esc goes back, q quits"),
		renderPane("Services", m.focus == focusServices, m.renderServices()),
		renderPane("Ports", m.focus == focusPorts, m.renderPorts()),
	}

	if m.detail != "" {
		sections = append(sections, detailStyle.Render(m.detail))
	}

	return strings.Join(sections, "\n\n") + "\n"
}

func (m *model) stepSelection(delta int) {
	switch m.focus {
	case focusServices:
		if len(m.snapshot.Services) == 0 {
			return
		}
		m.serviceIndex = clampIndex(m.serviceIndex+delta, len(m.snapshot.Services))
	case focusPorts:
		if len(m.snapshot.Ports) == 0 {
			return
		}
		m.portIndex = clampIndex(m.portIndex+delta, len(m.snapshot.Ports))
	}
}

func (m model) selectedDetail() string {
	switch m.focus {
	case focusServices:
		if len(m.snapshot.Services) == 0 {
			return ""
		}
		service := m.snapshot.Services[m.serviceIndex]
		lines := []string{
			fmt.Sprintf("Service: %s", service.Name),
			fmt.Sprintf("Version: %s", blankFallback(service.Version)),
		}
		if service.Description != "" {
			lines = append(lines, fmt.Sprintf("Description: %s", service.Description))
		}
		if len(service.Tags) > 0 {
			lines = append(lines, fmt.Sprintf("Tags: %s", strings.Join(service.Tags, ", ")))
		}
		return strings.Join(lines, "\n")
	case focusPorts:
		if len(m.snapshot.Ports) == 0 {
			return ""
		}
		port := m.snapshot.Ports[m.portIndex]
		return fmt.Sprintf("Port: %d\nInspection: active runtime port", port)
	default:
		return ""
	}
}

func (m model) renderServices() []string {
	if len(m.snapshot.Services) == 0 {
		return []string{"(none)"}
	}

	lines := make([]string, len(m.snapshot.Services))
	for i, service := range m.snapshot.Services {
		line := service.Name
		if service.Version != "" {
			line = fmt.Sprintf("%s %s", service.Name, service.Version)
		}
		if i == m.serviceIndex && m.focus == focusServices {
			lines[i] = selectedStyle.Render("> " + line)
			continue
		}
		lines[i] = "  " + line
	}
	return lines
}

func (m model) renderPorts() []string {
	if len(m.snapshot.Ports) == 0 {
		return []string{"(none)"}
	}

	lines := make([]string, len(m.snapshot.Ports))
	for i, port := range m.snapshot.Ports {
		line := fmt.Sprintf("%d", port)
		if i == m.portIndex && m.focus == focusPorts {
			lines[i] = selectedStyle.Render("> " + line)
			continue
		}
		lines[i] = "  " + line
	}
	return lines
}

func cloneSnapshot(snapshot runtime.Snapshot) runtime.Snapshot {
	return runtime.Snapshot{
		Services: cloneServices(snapshot.Services),
		Ports:    append([]int(nil), snapshot.Ports...),
	}
}

func cloneServices(services []orchestrator.Metadata) []orchestrator.Metadata {
	copied := make([]orchestrator.Metadata, len(services))
	for i, service := range services {
		copied[i] = orchestrator.Metadata{
			Name:        service.Name,
			Description: service.Description,
			Version:     service.Version,
			Tags:        append([]string(nil), service.Tags...),
		}
	}
	return copied
}

func toggleFocus(current focus) focus {
	if current == focusServices {
		return focusPorts
	}
	return focusServices
}

func clampIndex(index, size int) int {
	switch {
	case size <= 0:
		return 0
	case index < 0:
		return size - 1
	case index >= size:
		return 0
	default:
		return index
	}
}

func blankFallback(value string) string {
	if value == "" {
		return "(none)"
	}
	return value
}
