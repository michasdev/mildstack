package cli

import (
	"os"
	"path/filepath"
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

	activePath := filepath.Join(paths.InstancesDir, "active", "8080.json")
	savedPath := filepath.Join(paths.InstancesDir, "saved", "8080.json")
	if _, err := os.Stat(activePath); err != nil {
		t.Fatalf("expected active instance file: %v", err)
	}
	if _, err := os.Stat(savedPath); err != nil {
		t.Fatalf("expected saved instance file: %v", err)
	}

	if err := storage.DeleteActiveInstance(8080); err != nil {
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
