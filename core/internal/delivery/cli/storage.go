package cli

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/michasdev/mildstack/core/internal/application/runtime"
)

type Storage struct {
	paths         runtime.Paths
	legacyBaseDir string
}

type instanceRecord struct {
	Port int `json:"port"`
	PID  int `json:"pid"`
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
	return s.saveInstance(s.SavedInstancesDir(), port)
}

func (s Storage) SaveActiveInstance(port int) error {
	return s.saveInstance(s.ActiveInstancesDir(), port)
}

func (s Storage) DeleteActiveInstance(port int) error {
	path := s.instanceRecordPath(s.ActiveInstancesDir(), port)
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

func (s Storage) LoadActivePorts() ([]int, error) {
	return s.loadInstancePorts(s.ActiveInstancesDir())
}

func (s Storage) loadInstancePorts(dir string) ([]int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	ports := make([]int, 0, len(entries))
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
			ports = append(ports, record.Port)
		}
	}

	sort.Ints(ports)
	return ports, nil
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

func (s Storage) saveInstance(dir string, port int) error {
	record := instanceRecord{
		Port: port,
		PID:  os.Getpid(),
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

func (s Storage) instanceRecordPath(dir string, port int) string {
	return filepath.Join(dir, strconv.Itoa(port)+".json")
}

func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
