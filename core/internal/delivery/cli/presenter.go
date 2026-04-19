package cli

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/application/runtime"
)

type Presenter struct {
	services  []orchestrator.Metadata
	ports     []int
	instances []runtime.Instance
}

func NewPresenter(snapshot runtime.Snapshot) Presenter {
	return Presenter{
		services:  cloneMetadata(snapshot.Services),
		ports:     append([]int(nil), snapshot.Ports...),
		instances: cloneInstances(snapshot.Instances),
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

	fmt.Fprintf(&buf, "State: %s\n\n", p.PresentReadiness())
	buf.WriteString("Services:\n")
	if len(p.services) == 0 {
		buf.WriteString("  (none)\n")
	} else {
		for _, service := range p.services {
			fmt.Fprintf(&buf, "- %s %s\n", service.Name, service.Version)
		}
	}

	buf.WriteString("\nInstances:\n")
	if len(p.instancesForDisplay()) == 0 {
		buf.WriteString("  (none)\n")
	} else {
		for _, instance := range p.instancesForDisplay() {
			fmt.Fprintf(&buf, "- %d %s\n", instance.Port, instanceStatusLine(instance))
		}
	}

	buf.WriteString("Ports:\n")
	runningPorts := p.runningPorts()
	if len(runningPorts) == 0 {
		buf.WriteString("  (none)\n")
	} else {
		for _, port := range runningPorts {
			fmt.Fprintf(&buf, "- %d\n", port)
		}
	}

	return buf.String()
}

func (p Presenter) Services() []orchestrator.Metadata {
	return cloneMetadata(p.services)
}

func (p Presenter) Ports() []int {
	return append([]int(nil), p.runningPorts()...)
}

func (p Presenter) PresentPorts() string {
	ports := p.runningPorts()
	if len(ports) == 0 {
		return "No ports registered\n"
	}

	var buf bytes.Buffer
	for _, port := range ports {
		fmt.Fprintf(&buf, "%d\n", port)
	}
	return buf.String()
}

func (p Presenter) PresentReadiness() string {
	instances := p.instancesForDisplay()
	if len(instances) == 0 {
		return "not_started"
	}

	running := false
	errored := false
	for _, instance := range instances {
		switch instance.Status {
		case "running":
			running = true
		case "errored":
			errored = true
		}
	}

	switch {
	case running:
		return "running"
	case errored:
		return "errored"
	default:
		return "not_started"
	}
}

func (p Presenter) StatusPayload() statusPayload {
	return statusPayload{
		State:     p.PresentReadiness(),
		Services:  cloneServices(p.services),
		Instances: cloneInstancesPayload(p.instancesForDisplay()),
		Ports:     append([]int(nil), p.runningPorts()...),
	}
}

func (p Presenter) PortsPayload() portsPayload {
	return portsPayload{
		Ports: append([]int(nil), p.runningPorts()...),
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

func cloneInstances(instances []runtime.Instance) []runtime.Instance {
	copied := make([]runtime.Instance, len(instances))
	for i, instance := range instances {
		copied[i] = runtime.Instance{
			InstanceID: instance.InstanceID,
			Port:       instance.Port,
			PID:        instance.PID,
			Status:     instance.Status,
			Error:      instance.Error,
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

func cloneInstancesPayload(instances []runtime.Instance) []instancePayload {
	copied := make([]instancePayload, len(instances))
	for i, instance := range instances {
		copied[i] = instancePayload{
			InstanceID: instance.InstanceID,
			Port:       instance.Port,
			PID:        instance.PID,
			Status:     instance.Status,
			Error:      instance.Error,
		}
	}
	return copied
}

func (p Presenter) instancesForDisplay() []runtime.Instance {
	if len(p.instances) > 0 {
		return cloneInstances(p.instances)
	}
	if len(p.ports) == 0 {
		return nil
	}
	instances := make([]runtime.Instance, len(p.ports))
	for i, port := range p.ports {
		instances[i] = runtime.Instance{
			Port:   port,
			Status: "running",
		}
	}
	return instances
}

func (p Presenter) runningPorts() []int {
	instances := p.instancesForDisplay()
	if len(instances) == 0 {
		return append([]int(nil), p.ports...)
	}

	ports := make([]int, 0, len(instances))
	for _, instance := range instances {
		if instance.Status == "running" {
			ports = append(ports, instance.Port)
		}
	}
	return ports
}

func instanceStatusLine(instance runtime.Instance) string {
	if instance.Status == "" {
		return "running"
	}
	if instance.Error == "" {
		return instance.Status
	}
	return instance.Status + ": " + instance.Error
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
