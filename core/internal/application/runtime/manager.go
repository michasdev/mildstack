package runtime

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
)

type Snapshot struct {
	Services  []orchestrator.Metadata
	Ports     []int
	Instances []Instance
}

type Instance struct {
	InstanceID string
	Port       int
	PID        int
	Status     string
	Error      string
}

type Manager struct {
	mu         sync.Mutex
	services   []orchestrator.Metadata
	ports      []int
	instanceID string
}

func New(services []orchestrator.Service) *Manager {
	return NewWithPorts(services, nil)
}

func NewWithPorts(services []orchestrator.Service, ports []int) *Manager {
	copied := make([]orchestrator.Service, len(services))
	copy(copied, services)

	metadata := make([]orchestrator.Metadata, 0, len(copied))
	for _, service := range copied {
		if service == nil {
			continue
		}
		metadata = append(metadata, cloneMetadata(service.Metadata()))
	}

	return &Manager{
		services: metadata,
		ports:    sortedPorts(ports),
	}
}

func (m *Manager) Serve(ctx context.Context, port int) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, existing := range m.ports {
		if existing == port {
			return fmt.Errorf("runtime: port %d is already registered", port)
		}
	}

	m.ports = append(m.ports, port)
	return nil
}

func (m *Manager) Snapshot(ctx context.Context) Snapshot {
	if ctx != nil {
		_ = ctx.Err()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	return Snapshot{
		Services:  cloneMetadataSlice(m.services),
		Ports:     sortedPorts(m.ports),
		Instances: runningInstances(m.instanceID, m.ports),
	}
}

func (m *Manager) Ports(ctx context.Context) []int {
	if ctx != nil {
		_ = ctx.Err()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	return sortedPorts(m.ports)
}

func (m *Manager) RemovePort(port int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	filtered := m.ports[:0]
	for _, existing := range m.ports {
		if existing != port {
			filtered = append(filtered, existing)
		}
	}
	m.ports = append([]int(nil), filtered...)
}

// SetInstanceID sets the canonical instance identity that will be embedded in
// runtime snapshots. It is safe to call concurrently and must be called before
// the first Snapshot() call if callers need a stable non-empty InstanceID.
func (m *Manager) SetInstanceID(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.instanceID = id
}

func cloneMetadata(metadata orchestrator.Metadata) orchestrator.Metadata {
	copied := orchestrator.Metadata{
		Name:        metadata.Name,
		Description: metadata.Description,
		Version:     metadata.Version,
		Tags:        append([]string(nil), metadata.Tags...),
	}
	return copied
}

func cloneMetadataSlice(metadata []orchestrator.Metadata) []orchestrator.Metadata {
	copied := make([]orchestrator.Metadata, len(metadata))
	for i, item := range metadata {
		copied[i] = cloneMetadata(item)
	}
	return copied
}

func sortedPorts(ports []int) []int {
	copied := append([]int(nil), ports...)
	sort.Ints(copied)
	return copied
}

func runningInstances(instanceID string, ports []int) []Instance {
	copiedPorts := sortedPorts(ports)
	instances := make([]Instance, len(copiedPorts))
	for i, port := range copiedPorts {
		instances[i] = Instance{
			InstanceID: instanceID,
			Port:       port,
			Status:     "running",
		}
	}
	return instances
}
