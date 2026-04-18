package application

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/michasdev/mildstack/core/internal/resources/dynamodb/domain"
)

func TestResolveStoragePath(t *testing.T) {
	t.Helper()

	baseDir := t.TempDir()
	path, err := ResolveStoragePath(StorageConfig{
		BaseDir:    baseDir,
		InstanceID: "instance-a",
	})
	if err != nil {
		t.Fatalf("resolve storage path: %v", err)
	}

	want := filepath.Join(baseDir, "instances", "instance-a", "dynamodb")
	if got, want := path, want; got != want {
		t.Fatalf("unexpected storage path: got %q want %q", got, want)
	}
}

func TestSQLiteRepositoryBootstrapAndPersistAcrossRestart(t *testing.T) {
	t.Helper()

	baseDir := t.TempDir()
	repo := mustOpenSQLiteRepositoryAt(t, baseDir, "instance-a")
	defer func() {
		if err := repo.Close(); err != nil {
			t.Fatalf("close repo: %v", err)
		}
	}()

	initial, err := repo.Load()
	if err != nil {
		t.Fatalf("load initial state: %v", err)
	}
	if got, want := len(initial.Tables), 1; got != want {
		t.Fatalf("unexpected bootstrap table count: got %d want %d", got, want)
	}
	if got, want := initial.Tables[0].Name, "mildstack-records"; got != want {
		t.Fatalf("unexpected bootstrap table name: got %q want %q", got, want)
	}

	state := domain.NewState()
	state.UpsertTable(domain.Table{
		Name:         "mildstack-archive",
		PartitionKey: "pk",
		SortKey:      "sk",
		BillingMode:  "PAY_PER_REQUEST",
		Status:       domain.TableStatusCreating,
		CreatedAt:    state.Tables[0].CreatedAt,
	})
	state.UpsertItem(domain.Item{
		Table: "mildstack-archive",
		Key:   "item#1",
		Attributes: map[string]domain.AttributeValue{
			"id":    domain.StringValue("item#1"),
			"title": domain.StringValue("archive item"),
		},
	})
	if err := repo.Save(state); err != nil {
		t.Fatalf("save mutated state: %v", err)
	}
	if err := repo.Close(); err != nil {
		t.Fatalf("close repo after save: %v", err)
	}

	reopened := mustOpenSQLiteRepositoryAt(t, baseDir, "instance-a")
	defer func() {
		if err := reopened.Close(); err != nil {
			t.Fatalf("close reopened repo: %v", err)
		}
	}()

	loaded, err := reopened.Load()
	if err != nil {
		t.Fatalf("load restarted state: %v", err)
	}
	if got, want := len(loaded.Tables), 2; got != want {
		t.Fatalf("unexpected table count after restart: got %d want %d", got, want)
	}
	archive, ok := loaded.Table("mildstack-archive")
	if !ok {
		t.Fatal("expected archive table to survive restart")
	}
	if got, want := archive.Status, domain.TableStatusCreating; got != want {
		t.Fatalf("unexpected archive table status after restart: got %q want %q", got, want)
	}
	fetched, ok := loaded.Item("mildstack-archive", "item#1")
	if !ok {
		t.Fatal("expected item to survive restart")
	}
	if got, want := fetched.Attributes["title"].Any(), "archive item"; got != want {
		t.Fatalf("unexpected item title after restart: got %q want %q", got, want)
	}

	statePath := filepath.Join(repo.storageDir, sqliteFileName)
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("expected sqlite database to exist: %v", err)
	}
}

func TestSQLiteRepositoryIsolatesInstances(t *testing.T) {
	t.Helper()

	baseDir := t.TempDir()
	repoA := mustOpenSQLiteRepositoryAt(t, baseDir, "instance-a")
	defer func() {
		if err := repoA.Close(); err != nil {
			t.Fatalf("close repoA: %v", err)
		}
	}()
	repoB := mustOpenSQLiteRepositoryAt(t, baseDir, "instance-b")
	defer func() {
		if err := repoB.Close(); err != nil {
			t.Fatalf("close repoB: %v", err)
		}
	}()

	state := domain.NewState()
	state.UpsertTable(domain.Table{
		Name:         "mildstack-archive",
		PartitionKey: "pk",
		SortKey:      "sk",
		BillingMode:  "PAY_PER_REQUEST",
		Status:       domain.TableStatusCreating,
	})
	if err := repoA.Save(state); err != nil {
		t.Fatalf("save repoA state: %v", err)
	}

	loadedA, err := repoA.Load()
	if err != nil {
		t.Fatalf("load repoA state: %v", err)
	}
	if !loadedA.HasTable("mildstack-archive") {
		t.Fatal("expected repoA to contain custom table")
	}

	loadedB, err := repoB.Load()
	if err != nil {
		t.Fatalf("load repoB state: %v", err)
	}
	if loadedB.HasTable("mildstack-archive") {
		t.Fatal("expected repoB to stay isolated from repoA state")
	}
}

func TestSQLiteRepositoryPersistsLifecycleMetadata(t *testing.T) {
	t.Helper()

	repo := mustOpenSQLiteRepository(t, "instance-a")
	defer func() {
		if err := repo.Close(); err != nil {
			t.Fatalf("close repo: %v", err)
		}
	}()

	createdAt := time.Date(2026, time.April, 18, 12, 0, 0, 0, time.UTC)
	state := domain.State{
		Service: "dynamodb",
		Tables: []domain.Table{
			{
				Name:         "mildstack-archive",
				PartitionKey: "pk",
				SortKey:      "sk",
				BillingMode:  "PAY_PER_REQUEST",
				Status:       domain.TableStatusCreating,
				CreatedAt:    createdAt,
				ActivationAt: createdAt.Add(50 * time.Millisecond),
			},
			{
				Name:         "mildstack-deleted",
				PartitionKey: "pk",
				BillingMode:  "PAY_PER_REQUEST",
				Status:       domain.TableStatusDeleting,
				CreatedAt:    createdAt,
				DeletedAt:    createdAt.Add(time.Second),
			},
		},
	}

	if err := repo.Save(state); err != nil {
		t.Fatalf("save lifecycle state: %v", err)
	}

	loaded, err := repo.Load()
	if err != nil {
		t.Fatalf("reload lifecycle state: %v", err)
	}

	archive, ok := loaded.Table("mildstack-archive")
	if !ok {
		t.Fatal("expected archive table to be present")
	}
	if got, want := archive.Status, domain.TableStatusCreating; got != want {
		t.Fatalf("unexpected archive status: got %q want %q", got, want)
	}
	if got, want := archive.CreatedAt, createdAt; !got.Equal(want) {
		t.Fatalf("unexpected archive created_at: got %v want %v", got, want)
	}
	if got, want := archive.ActivationAt, createdAt.Add(50*time.Millisecond); !got.Equal(want) {
		t.Fatalf("unexpected archive activation_at: got %v want %v", got, want)
	}

	deleted, ok := loaded.Table("mildstack-deleted")
	if !ok {
		t.Fatal("expected deleted table to be present")
	}
	if got, want := deleted.Status, domain.TableStatusDeleting; got != want {
		t.Fatalf("unexpected deleted table status: got %q want %q", got, want)
	}
	if got, want := deleted.DeletedAt, createdAt.Add(time.Second); !got.Equal(want) {
		t.Fatalf("unexpected deleted_at: got %v want %v", got, want)
	}
}

func TestSQLiteRepositoryClosesCleanly(t *testing.T) {
	t.Helper()

	repo := mustOpenSQLiteRepository(t, "instance-a")
	if err := repo.Close(); err != nil {
		t.Fatalf("close repo: %v", err)
	}
	if err := repo.Close(); err != nil {
		t.Fatalf("second close should be safe: %v", err)
	}
	if _, err := repo.Load(); err == nil {
		t.Fatal("expected load on closed repo to fail")
	}
}

func TestSQLiteRepositoryLoadsLegacyItemEncoding(t *testing.T) {
	t.Helper()

	repo := mustOpenSQLiteRepository(t, "instance-legacy")
	defer func() {
		if err := repo.Close(); err != nil {
			t.Fatalf("close repo: %v", err)
		}
	}()

	if _, err := repo.db.ExecContext(context.Background(), `
		INSERT INTO dynamodb_tables(name, partition_key, sort_key, billing_mode, status, created_at_ns, activation_at_ns, deleted_at_ns)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, "mildstack-archive", "id", "", "PAY_PER_REQUEST", domain.TableStatusActive, 0, 0, 0); err != nil {
		t.Fatalf("insert table: %v", err)
	}
	if _, err := repo.db.ExecContext(context.Background(), `
		INSERT INTO dynamodb_items(table_name, item_key, attributes_json)
		VALUES (?, ?, ?)
	`, "mildstack-archive", "item#legacy", `{"id":"item#legacy","title":"legacy item","version":"1"}`); err != nil {
		t.Fatalf("insert legacy item: %v", err)
	}

	loaded, err := repo.Load()
	if err != nil {
		t.Fatalf("load legacy state: %v", err)
	}

	item, ok := loaded.Item("mildstack-archive", "item#legacy")
	if !ok {
		t.Fatal("expected legacy item to load")
	}
	if got, want := item.Attributes["title"].Any(), "legacy item"; got != want {
		t.Fatalf("unexpected legacy title: got %q want %q", got, want)
	}
	if got, want := item.Attributes["version"].Any(), "1"; got != want {
		t.Fatalf("unexpected legacy version: got %q want %q", got, want)
	}
}

func mustOpenSQLiteRepository(t *testing.T, instanceID string) *SQLiteRepository {
	t.Helper()

	return mustOpenSQLiteRepositoryAt(t, t.TempDir(), instanceID)
}

func mustOpenSQLiteRepositoryAt(t *testing.T, baseDir, instanceID string) *SQLiteRepository {
	t.Helper()

	storagePath, err := ResolveStoragePath(StorageConfig{
		BaseDir:    baseDir,
		InstanceID: instanceID,
	})
	if err != nil {
		t.Fatalf("resolve storage path: %v", err)
	}

	repo, err := NewSQLiteRepository(storagePath)
	if err != nil {
		t.Fatalf("open sqlite repository: %v", err)
	}
	return repo
}
