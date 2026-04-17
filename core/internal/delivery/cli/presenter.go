package cli

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/application/runtime"
)

type Presenter struct {
	services []orchestrator.Metadata
	ports    []int
}

func NewPresenter(snapshot runtime.Snapshot) Presenter {
	return Presenter{
		services: cloneMetadata(snapshot.Services),
		ports:    append([]int(nil), snapshot.Ports...),
	}
}

func PresentStatus(snapshot runtime.Snapshot) string {
	return NewPresenter(snapshot).PresentStatus()
}

func PresentPorts(ports []int) string {
	return NewPresenter(runtime.Snapshot{Ports: append([]int(nil), ports...)}).PresentPorts()
}

func PresentReadiness(snapshot runtime.Snapshot) string {
	return NewPresenter(snapshot).PresentReadiness()
}

func PresentError(err error) string {
	return renderError(err)
}

func (p Presenter) PresentStatus() string {
	var buf bytes.Buffer

	buf.WriteString("Services:\n")
	if len(p.services) == 0 {
		buf.WriteString("  (none)\n")
	} else {
		for _, service := range p.services {
			fmt.Fprintf(&buf, "- %s %s\n", service.Name, service.Version)
		}
	}

	buf.WriteString("Ports:\n")
	if len(p.ports) == 0 {
		buf.WriteString("  (none)\n")
	} else {
		for _, port := range p.ports {
			fmt.Fprintf(&buf, "- %d\n", port)
		}
	}

	return buf.String()
}

func (p Presenter) Services() []orchestrator.Metadata {
	return cloneMetadata(p.services)
}

func (p Presenter) Ports() []int {
	return append([]int(nil), p.ports...)
}

func (p Presenter) PresentPorts() string {
	if len(p.ports) == 0 {
		return "No ports registered\n"
	}

	var buf bytes.Buffer
	for _, port := range p.ports {
		fmt.Fprintf(&buf, "%d\n", port)
	}
	return buf.String()
}

func (p Presenter) PresentReadiness() string {
	if len(p.ports) > 0 {
		return "ready"
	}
	return "not_ready"
}

func (p Presenter) StatusPayload() statusPayload {
	return statusPayload{
		State:    p.PresentReadiness(),
		Services: cloneServices(p.services),
		Ports:    append([]int(nil), p.ports...),
	}
}

func (p Presenter) PortsPayload() portsPayload {
	return portsPayload{
		Ports: append([]int(nil), p.ports...),
	}
}

func cloneMetadata(metadata []orchestrator.Metadata) []orchestrator.Metadata {
	copied := make([]orchestrator.Metadata, len(metadata))
	for i, item := range metadata {
		copied[i] = orchestrator.Metadata{
			Name:        item.Name,
			Description: item.Description,
			Version:     item.Version,
			Tags:        append([]string(nil), item.Tags...),
		}
	}
	return copied
}

func cloneServices(services []orchestrator.Metadata) []servicePayload {
	copied := make([]servicePayload, len(services))
	for i, item := range services {
		copied[i] = servicePayload{
			Name:        item.Name,
			Description: item.Description,
			Version:     item.Version,
			Tags:        append([]string(nil), item.Tags...),
		}
	}
	return copied
}

func renderError(err error) string {
	if err == nil {
		return ""
	}

	message := strings.TrimSpace(err.Error())
	if message == "" {
		return "error"
	}

	return fmt.Sprintf("error: %s", message)
}
