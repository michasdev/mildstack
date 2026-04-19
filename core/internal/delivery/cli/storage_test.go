package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/michasdev/mildstack/core/internal/application/runtime"
)

func TestStorageReadsLegacyConfigAndMigratesToHomeScopedLayout(t *testing.T) {
	t.Helper()

	homeDir := t.TempDir()
	configDir := t.TempDir()
	paths := runtime.ResolvePathsFrom(homeDir, configDir)
	legacyBase := runtime.LegacyBaseDirFrom(homeDir, configDir)
	storage := NewStorage(paths, legacyBase)

	legacyConfigPath := filepath.Join(legacyBase, "config", "config.json")
	if err := os.MkdirAll(filepath.Dir(legacyConfigPath), 0o755); err != nil {
		t.Fatalf("create legacy dir: %v", err)
	}
	if err := os.WriteFile(legacyConfigPath, []byte(`{"version":"legacy"}`), 0o644); err != nil {
		t.Fatalf("write legacy config: %v", err)
	}

	data, err := storage.ReadConfig("config.json")
	if err != nil {
		t.Fatalf("read legacy config: %v", err)
	}
	if got, want := string(data), `{"version":"legacy"}`; got != want {
		t.Fatalf("unexpected legacy config: got %q want %q", got, want)
	}

	migrated, err := storage.MigrateConfig("config.json")
	if err != nil {
		t.Fatalf("migrate config: %v", err)
	}
	if !migrated {
		t.Fatal("expected config migration to copy legacy data")
	}

	newConfigPath := filepath.Join(paths.ConfigDir, "config.json")
	if _, err := os.Stat(newConfigPath); err != nil {
		t.Fatalf("expected migrated config in new layout: %v", err)
	}

	migrated, err = storage.MigrateConfig("config.json")
	if err != nil {
		t.Fatalf("migrate config second time: %v", err)
	}
	if migrated {
		t.Fatal("expected config migration to be idempotent")
	}
}

func TestStorageWritesToHomeScopedLayout(t *testing.T) {
	t.Helper()

	homeDir := t.TempDir()
	configDir := t.TempDir()
	paths := runtime.ResolvePathsFrom(homeDir, configDir)
	storage := NewStorage(paths, runtime.LegacyBaseDirFrom(homeDir, configDir))

	if err := storage.WriteInstance("registry.json", []byte(`{"ports":[8080]}`)); err != nil {
		t.Fatalf("write instance: %v", err)
	}

	if got, want := filepath.Join(paths.InstancesDir, "registry.json"), filepath.Join(homeDir, ".mildstack", "instances", "registry.json"); got != want {
		t.Fatalf("unexpected instance path: got %q want %q", got, want)
	}

	data, err := os.ReadFile(filepath.Join(paths.InstancesDir, "registry.json"))
	if err != nil {
		t.Fatalf("read instance: %v", err)
	}
	if got, want := string(data), `{"ports":[8080]}`; got != want {
		t.Fatalf("unexpected instance payload: got %q want %q", got, want)
	}
}

func TestStorageTracksActiveAndSavedInstances(t *testing.T) {
	t.Helper()

	homeDir := t.TempDir()
	configDir := t.TempDir()
	paths := runtime.ResolvePathsFrom(homeDir, configDir)
	storage := NewStorage(paths, runtime.LegacyBaseDirFrom(homeDir, configDir))

	if err := storage.SaveSavedInstance(8080); err != nil {
		t.Fatalf("save saved instance: %v", err)
	}
	if err := storage.SaveActiveInstance(8080); err != nil {
		t.Fatalf("save active instance: %v", err)
	}

	ports, err := storage.LoadActivePorts()
	if err != nil {
		t.Fatalf("load active ports: %v", err)
	}
	if got, want := len(ports), 1; got != want || ports[0] != 8080 {
		t.Fatalf("unexpected active ports: %#v", ports)
	}

	instances, err := storage.LoadInstances()
	if err != nil {
		t.Fatalf("load instances: %v", err)
	}
	if len(instances) != 1 {
		t.Fatalf("expected one instance, got %#v", instances)
	}
	if instances[0].InstanceID == "" {
		t.Fatal("expected generated instance id")
	}

	activePath := filepath.Join(paths.InstancesDir, "active", instances[0].InstanceID+".json")
	savedPath := filepath.Join(paths.InstancesDir, "saved", instances[0].InstanceID+".json")
	if _, err := os.Stat(activePath); err != nil {
		t.Fatalf("expected active instance file: %v", err)
	}
	if _, err := os.Stat(savedPath); err != nil {
		t.Fatalf("expected saved instance file: %v", err)
	}

	if err := storage.DeleteActiveInstance(instances[0]); err != nil {
		t.Fatalf("delete active instance: %v", err)
	}
	if _, err := os.Stat(activePath); !os.IsNotExist(err) {
		t.Fatalf("expected active file to be deleted, got err=%v", err)
	}

	ports, err = storage.LoadActivePorts()
	if err != nil {
		t.Fatalf("load active ports after delete: %v", err)
	}
	if len(ports) != 0 {
		t.Fatalf("expected no active ports after delete, got %#v", ports)
	}
}

func TestStorageTracksErroredInstancesAndHidesThemFromActivePorts(t *testing.T) {
	t.Helper()

	homeDir := t.TempDir()
	configDir := t.TempDir()
	paths := runtime.ResolvePathsFrom(homeDir, configDir)
	storage := NewStorage(paths, runtime.LegacyBaseDirFrom(homeDir, configDir))

	if err := storage.SaveSavedInstance(9090); err != nil {
		t.Fatalf("save saved instance: %v", err)
	}
	if err := storage.SaveErroredInstance(9090, os.ErrClosed); err != nil {
		t.Fatalf("save errored instance: %v", err)
	}

	instances, err := storage.LoadInstances()
	if err != nil {
		t.Fatalf("load instances: %v", err)
	}
	if len(instances) != 1 {
		t.Fatalf("expected one instance, got %#v", instances)
	}
	if got, want := instances[0].Status, "errored"; got != want {
		t.Fatalf("unexpected instance status: got %q want %q", got, want)
	}
	if got, want := instances[0].Error, "file already closed"; got != want {
		t.Fatalf("unexpected instance error: got %q want %q", got, want)
	}

	ports, err := storage.LoadActivePorts()
	if err != nil {
		t.Fatalf("load active ports: %v", err)
	}
	if len(ports) != 0 {
		t.Fatalf("expected errored instance to be excluded from active ports, got %#v", ports)
	}
}

func TestStorageTracksInactiveInstancesAsNotStarted(t *testing.T) {
	t.Helper()

	homeDir := t.TempDir()
	configDir := t.TempDir()
	paths := runtime.ResolvePathsFrom(homeDir, configDir)
	storage := NewStorage(paths, runtime.LegacyBaseDirFrom(homeDir, configDir))

	if err := storage.SaveSavedInstance(7070); err != nil {
		t.Fatalf("save saved instance: %v", err)
	}
	instances, err := storage.LoadInstances()
	if err != nil {
		t.Fatalf("load instances after save: %v", err)
	}
	if len(instances) != 1 {
		t.Fatalf("expected one instance after save, got %#v", instances)
	}
	recordPath := filepath.Join(paths.InstancesDir, "active", instances[0].InstanceID+".json")
	if err := os.MkdirAll(filepath.Dir(recordPath), 0o755); err != nil {
		t.Fatalf("create active dir: %v", err)
	}
	if err := os.WriteFile(recordPath, []byte("{\n  \"instanceId\": \""+instances[0].InstanceID+"\",\n  \"port\": 7070,\n  \"status\": \"running\"\n}\n"), 0o644); err != nil {
		t.Fatalf("write inactive active record: %v", err)
	}

	instances, err = storage.LoadInstances()
	if err != nil {
		t.Fatalf("load instances: %v", err)
	}
	if len(instances) != 1 {
		t.Fatalf("expected one instance, got %#v", instances)
	}
	if got, want := instances[0].Status, "not_started"; got != want {
		t.Fatalf("unexpected inactive instance status: got %q want %q", got, want)
	}
	if got := instances[0].Error; got != "" {
		t.Fatalf("expected inactive instance to have no error, got %q", got)
	}
}

func TestStorageInstanceSummaryCarriesInstanceID(t *testing.T) {
	t.Helper()

	homeDir := t.TempDir()
	configDir := t.TempDir()
	paths := runtime.ResolvePathsFrom(homeDir, configDir)
	storage := NewStorage(paths, runtime.LegacyBaseDirFrom(homeDir, configDir))

	if err := storage.SaveActiveInstanceWithID("inst-abc", 8080); err != nil {
		t.Fatalf("save active instance with id: %v", err)
	}

	instances, err := storage.LoadInstances()
	if err != nil {
		t.Fatalf("load instances: %v", err)
	}
	if len(instances) != 1 {
		t.Fatalf("expected one instance, got %#v", instances)
	}
	if got, want := instances[0].InstanceID, "inst-abc"; got != want {
		t.Fatalf("unexpected instance id: got %q want %q", got, want)
	}
	if got, want := instances[0].Port, 8080; got != want {
		t.Fatalf("unexpected port: got %d want %d", got, want)
	}
}

func TestStorageLegacyPortKeyedRecordLoadsAndPresentsWithInstanceID(t *testing.T) {
	t.Helper()

	homeDir := t.TempDir()
	configDir := t.TempDir()
	paths := runtime.ResolvePathsFrom(homeDir, configDir)
	storage := NewStorage(paths, runtime.LegacyBaseDirFrom(homeDir, configDir))

	// write a legacy port-keyed record without instanceId
	legacyRecord := []byte("{\n  \"port\": 9090,\n  \"pid\": 0,\n  \"status\": \"not_started\"\n}\n")
	savedDir := filepath.Join(paths.InstancesDir, "saved")
	if err := os.MkdirAll(savedDir, 0o755); err != nil {
		t.Fatalf("create saved dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(savedDir, "9090.json"), legacyRecord, 0o644); err != nil {
		t.Fatalf("write legacy record: %v", err)
	}

	instances, err := storage.LoadInstances()
	if err != nil {
		t.Fatalf("load legacy instances: %v", err)
	}
	if len(instances) != 1 {
		t.Fatalf("expected one instance from legacy record, got %#v", instances)
	}
	if got, want := instances[0].Port, 9090; got != want {
		t.Fatalf("unexpected port: got %d want %d", got, want)
	}
	if got, want := instances[0].InstanceID, legacyInstanceIDFromPort(9090); got != want {
		t.Fatalf("unexpected legacy instance id: got %q want %q", got, want)
	}
}

func TestStorageDeleteInstanceResourcesRemovesInstanceDirectory(t *testing.T) {
	t.Helper()

	homeDir := t.TempDir()
	configDir := t.TempDir()
	paths := runtime.ResolvePathsFrom(homeDir, configDir)
	storage := NewStorage(paths, runtime.LegacyBaseDirFrom(homeDir, configDir))

	instanceID := "inst-delete-me"
	instanceDir := filepath.Join(paths.InstancesDir, instanceID)
	if err := os.MkdirAll(filepath.Join(instanceDir, "s3"), 0o755); err != nil {
		t.Fatalf("create instance dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(instanceDir, "s3", "state.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("write instance payload: %v", err)
	}

	if err := storage.DeleteInstanceResources(instanceID); err != nil {
		t.Fatalf("delete instance resources: %v", err)
	}
	if _, err := os.Stat(instanceDir); !os.IsNotExist(err) {
		t.Fatalf("expected instance directory to be deleted, got err=%v", err)
	}
}

func TestNewInstanceIDReturnsRandomNonLegacyID(t *testing.T) {
	t.Helper()

	first, err := NewInstanceID()
	if err != nil {
		t.Fatalf("new instance id: %v", err)
	}
	second, err := NewInstanceID()
	if err != nil {
		t.Fatalf("new instance id second call: %v", err)
	}
	if first == "" || second == "" {
		t.Fatal("expected non-empty instance ids")
	}
	if first == second {
		t.Fatalf("expected distinct random ids, got %q", first)
	}
	if strings.HasPrefix(first, "mildstack-") || strings.HasPrefix(second, "mildstack-") {
		t.Fatalf("expected non-legacy instance ids, got %q and %q", first, second)
	}
}
