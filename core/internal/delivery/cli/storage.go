package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/michasdev/mildstack/core/internal/application/runtime"
)

type Storage struct {
	paths         runtime.Paths
	legacyBaseDir string
}

type instanceRecord struct {
	InstanceID string `json:"instanceId,omitempty"`
	Port       int    `json:"port"`
	PID        int    `json:"pid,omitempty"`
	Status     string `json:"status,omitempty"`
	Error      string `json:"error,omitempty"`
}

type instanceSummary struct {
	InstanceID string `json:"instanceId"`
	Port       int    `json:"port"`
	PID        int    `json:"pid,omitempty"`
	Status     string `json:"status"`
	Error      string `json:"error,omitempty"`
}

func NewStorage(paths runtime.Paths, legacyBaseDir string) Storage {
	return Storage{
		paths:         paths,
		legacyBaseDir: legacyBaseDir,
	}
}

func (s Storage) ConfigPath(name string) string {
	return filepath.Join(s.paths.ConfigDir, name)
}

func (s Storage) InstancePath(name string) string {
	return filepath.Join(s.paths.InstancesDir, name)
}

func (s Storage) LogPath(name string) string {
	return filepath.Join(s.paths.LogsDir, name)
}

func (s Storage) CachePath(name string) string {
	return filepath.Join(s.paths.CacheDir, name)
}

func (s Storage) SavedInstancesDir() string {
	return filepath.Join(s.paths.InstancesDir, "saved")
}

func (s Storage) ActiveInstancesDir() string {
	return filepath.Join(s.paths.InstancesDir, "active")
}

func (s Storage) ReadConfig(name string) ([]byte, error) {
	return s.readFile(s.ConfigPath(name), s.legacyCategoryPath("config", name))
}

func (s Storage) WriteConfig(name string, data []byte) error {
	return writeFile(s.ConfigPath(name), data)
}

func (s Storage) ReadInstance(name string) ([]byte, error) {
	return s.readFile(s.InstancePath(name), s.legacyCategoryPath("instances", name))
}

func (s Storage) WriteInstance(name string, data []byte) error {
	return writeFile(s.InstancePath(name), data)
}

func (s Storage) MigrateConfig(name string) (bool, error) {
	return s.migrateFile(s.ConfigPath(name), s.legacyCategoryPath("config", name))
}

func (s Storage) MigrateInstance(name string) (bool, error) {
	return s.migrateFile(s.InstancePath(name), s.legacyCategoryPath("instances", name))
}

func (s Storage) SaveSavedInstance(port int) error {
	return s.saveInstanceWithID("", s.SavedInstancesDir(), port, "not_started", "")
}

func (s Storage) SaveActiveInstance(port int) error {
	return s.saveInstanceWithID("", s.ActiveInstancesDir(), port, "running", "")
}

// SaveSavedInstanceWithID persists a saved instance record with the canonical instanceId.
func (s Storage) SaveSavedInstanceWithID(instanceID string, port int) error {
	return s.saveInstanceWithID(instanceID, s.SavedInstancesDir(), port, "not_started", "")
}

// SaveActiveInstanceWithID persists an active instance record with the canonical instanceId.
func (s Storage) SaveActiveInstanceWithID(instanceID string, port int) error {
	return s.saveInstanceWithID(instanceID, s.ActiveInstancesDir(), port, "running", "")
}

func (s Storage) SaveErroredInstance(port int, err error) error {
	message := ""
	if err != nil {
		message = strings.TrimSpace(err.Error())
	}
	return s.saveInstanceWithID("", s.ActiveInstancesDir(), port, "errored", message)
}

func (s Storage) DeleteActiveInstance(port int) error {
	path := s.instanceRecordPath(s.ActiveInstancesDir(), port)
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

func (s Storage) DeleteSavedInstance(port int) error {
	path := s.instanceRecordPath(s.SavedInstancesDir(), port)
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

func (s Storage) LoadActivePorts() ([]int, error) {
	instances, err := s.LoadInstances()
	if err != nil {
		return nil, err
	}

	ports := make([]int, 0, len(instances))
	for _, instance := range instances {
		if instance.Status == "running" {
			ports = append(ports, instance.Port)
		}
	}

	sort.Ints(ports)
	return ports, nil
}

func (s Storage) LoadInstances() ([]instanceSummary, error) {
	active, err := s.loadInstanceRecords(s.ActiveInstancesDir())
	if err != nil {
		return nil, err
	}
	saved, err := s.loadInstanceRecords(s.SavedInstancesDir())
	if err != nil {
		return nil, err
	}

	instances := make(map[int]instanceSummary, len(saved)+len(active))
	for _, record := range saved {
		if record.Port <= 0 {
			continue
		}
		id := record.InstanceID
		if id == "" {
			id = instanceIDFromPort(record.Port)
		}
		instances[record.Port] = instanceSummary{
			InstanceID: id,
			Port:       record.Port,
			PID:        record.PID,
			Status:     "not_started",
			Error:      strings.TrimSpace(record.Error),
		}
	}
	for _, record := range active {
		if record.Port <= 0 {
			continue
		}
		status := strings.TrimSpace(record.Status)
		errorMessage := strings.TrimSpace(record.Error)
		alive := record.PID > 0 && processAlive(record.PID)
		switch {
		case errorMessage != "":
			status = "errored"
		case status == "running" && alive:
			status = "running"
		default:
			status = "not_started"
		}
		// active record wins on instanceId; fall back to saved, then derive from port
		instanceID := record.InstanceID
		if instanceID == "" {
			if prev, ok := instances[record.Port]; ok && prev.InstanceID != "" {
				instanceID = prev.InstanceID
			}
		}
		if instanceID == "" {
			instanceID = instanceIDFromPort(record.Port)
		}
		instance := instanceSummary{
			InstanceID: instanceID,
			Port:       record.Port,
			PID:        record.PID,
			Status:     status,
			Error:      errorMessage,
		}
		if instance.Status == "running" {
			instance.Error = ""
		}
		instances[record.Port] = instance
	}

	result := make([]instanceSummary, 0, len(instances))
	for _, instance := range instances {
		result = append(result, instance)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Port < result[j].Port
	})
	return result, nil
}

func (s Storage) loadInstanceRecords(dir string) ([]instanceRecord, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	records := make([]instanceRecord, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		var record instanceRecord
		if err := json.Unmarshal(data, &record); err != nil {
			return nil, err
		}
		if record.Port > 0 {
			records = append(records, record)
		}
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].Port < records[j].Port
	})
	return records, nil
}

func (s Storage) readFile(primary string, legacy string) ([]byte, error) {
	if data, err := os.ReadFile(primary); err == nil {
		return data, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	if legacy == "" {
		return nil, fs.ErrNotExist
	}

	data, err := os.ReadFile(legacy)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (s Storage) migrateFile(primary string, legacy string) (bool, error) {
	if _, err := os.Stat(primary); err == nil {
		return false, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return false, err
	}

	if legacy == "" {
		return false, fs.ErrNotExist
	}

	data, err := os.ReadFile(legacy)
	if err != nil {
		return false, err
	}

	if err := writeFile(primary, data); err != nil {
		return false, err
	}

	return true, nil
}

func (s Storage) legacyCategoryPath(category, name string) string {
	if s.legacyBaseDir == "" {
		return ""
	}

	return filepath.Join(s.legacyBaseDir, category, name)
}

func (s Storage) saveInstanceWithID(instanceID string, dir string, port int, status string, message string) error {
	record := instanceRecord{
		InstanceID: instanceID,
		Port:       port,
		PID:        os.Getpid(),
		Status:     status,
		Error:      strings.TrimSpace(message),
	}
	if record.Status == "not_started" {
		record.PID = 0
	}
	if record.Status == "errored" {
		record.PID = 0
	}

	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}

	path := s.instanceRecordPath(dir, port)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	return process.Signal(syscall.Signal(0)) == nil
}

func (s Storage) instanceRecordPath(dir string, port int) string {
	return filepath.Join(dir, strconv.Itoa(port)+".json")
}

func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// instanceIDFromPort derives a stable, human-readable instance identity from
// a port number. The format matches the derivation in main.go so that legacy
// records without an explicit instanceId are always resolved consistently.
func instanceIDFromPort(port int) string {
	return fmt.Sprintf("mildstack-%d", port)
}
