package application

import (
	"context"
	"testing"
	"time"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
	"github.com/michasdev/mildstack/core/internal/application/runtime"
	deliveryhttp "github.com/michasdev/mildstack/core/internal/delivery/http"
	"github.com/michasdev/mildstack/core/internal/resources/dynamodb/domain"
)

func TestServiceMetadataRoutesAndState(t *testing.T) {
	t.Helper()

	service := New()
	if _, ok := any(service).(orchestrator.Service); !ok {
		t.Fatal("expected service to satisfy orchestrator.Service")
	}

	metadata := service.Metadata()
	if got, want := metadata.Name, "dynamodb"; got != want {
		t.Fatalf("unexpected service name: got %q want %q", got, want)
	}
	if got, want := metadata.Version, "v1"; got != want {
		t.Fatalf("unexpected service version: got %q want %q", got, want)
	}
	if got, want := metadata.Description, "MildStack DynamoDB real service"; got != want {
		t.Fatalf("unexpected service description: got %q want %q", got, want)
	}

	policy := service.Policy()
	if got, want := policy.Fidelity, orchestrator.FidelityExemplar; got != want {
		t.Fatalf("unexpected policy fidelity: got %q want %q", got, want)
	}
	if got, want := policy.ErrorPrefix, "dynamodb"; got != want {
		t.Fatalf("unexpected policy error prefix: got %q want %q", got, want)
	}
	if got, want := len(policy.Supported), 8; got != want {
		t.Fatalf("unexpected supported count: got %d want %d", got, want)
	}
	if got, want := len(policy.Unsupported), 2; got != want {
		t.Fatalf("unexpected unsupported count: got %d want %d", got, want)
	}
	policy.Supported[0] = "changed"
	policy.Unsupported[0] = "changed"
	again := service.Policy()
	if got, want := again.Supported[0], "list tables"; got != want {
		t.Fatalf("policy supported slice was not copied: got %q want %q", got, want)
	}
	if got, want := again.Unsupported[0], "global tables"; got != want {
		t.Fatalf("policy unsupported slice was not copied: got %q want %q", got, want)
	}

	expectedTags := []string{"aws", "database", "nosql", "real-service"}
	if got, want := len(metadata.Tags), len(expectedTags); got != want {
		t.Fatalf("unexpected tag count: got %d want %d", got, want)
	}
	for i, tag := range expectedTags {
		if metadata.Tags[i] != tag {
			t.Fatalf("unexpected tag at %d: got %q want %q", i, metadata.Tags[i], tag)
		}
	}

	registrar := deliveryhttp.NewRegistrar()
	if err := service.RegisterRoutes(registrar); err != nil {
		t.Fatalf("register routes: %v", err)
	}

	entry, ok := registrar.Service("dynamodb")
	if !ok {
		t.Fatal("expected dynamodb service to be registered")
	}
	if got, want := len(entry.Routes), 5; got != want {
		t.Fatalf("unexpected route count: got %d want %d", got, want)
	}
	if got, want := entry.Routes[0].Method, "DELETE"; got != want {
		t.Fatalf("unexpected first route method: got %q want %q", got, want)
	}
	if got, want := entry.Routes[0].Path, "/api/v1/runtime/services/dynamodb/tables/:table/items/:item"; got != want {
		t.Fatalf("unexpected first route path: got %q want %q", got, want)
	}
	if got, want := entry.Routes[1].Path, "/api/v1/runtime/services/dynamodb/tables"; got != want {
		t.Fatalf("unexpected second route path: got %q want %q", got, want)
	}
	if got, want := entry.Routes[4].Method, "PUT"; got != want {
		t.Fatalf("unexpected last route method: got %q want %q", got, want)
	}
	if got, want := entry.Routes[4].Path, "/api/v1/runtime/services/dynamodb/tables/:table/items/:item"; got != want {
		t.Fatalf("unexpected last route path: got %q want %q", got, want)
	}

	hook := runtime.NewStateHook()
	if err := service.AttachState(hook); err != nil {
		t.Fatalf("attach state: %v", err)
	}

	value, ok := hook.Get(domain.StateKey)
	if !ok {
		t.Fatalf("expected state for %q to be present", domain.StateKey)
	}
	state := value.(map[string]any)
	if got, want := state["service"], "dynamodb"; got != want {
		t.Fatalf("unexpected service state name: got %v want %v", got, want)
	}

	tables := state["tables"].([]any)
	if got, want := len(tables), 1; got != want {
		t.Fatalf("unexpected table count: got %d want %d", got, want)
	}

	items := state["items"].([]any)
	if got, want := len(items), 1; got != want {
		t.Fatalf("unexpected item count: got %d want %d", got, want)
	}
}

func TestServicePersistsAcrossRestart(t *testing.T) {
	t.Helper()

	baseDir := t.TempDir()
	config := StorageConfig{BaseDir: baseDir, InstanceID: "instance-a"}

	service, err := NewWithPersistence(config)
	if err != nil {
		t.Fatalf("new with persistence: %v", err)
	}

	if _, err := service.CreateTable("mildstack-archive", "pk", "sk", "PAY_PER_REQUEST"); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if _, err := service.PutItem("mildstack-archive", "item#1", map[string]domain.AttributeValue{
		"id":       domain.StringValue("item#1"),
		"title":    domain.StringValue("archive item"),
		"version":  domain.NumberValue("1"),
		"obsolete": domain.StringValue("remove me"),
	}); err != nil {
		t.Fatalf("put item: %v", err)
	}
	if _, err := service.UpdateItem("mildstack-archive", "item#1", "SET title = :title ADD version :inc REMOVE obsolete", "", nil, map[string]domain.AttributeValue{
		":title": domain.StringValue("updated archive item"),
		":inc":   domain.NumberValue("1"),
	}); err != nil {
		t.Fatalf("update item: %v", err)
	}
	if err := service.Stop(context.Background()); err != nil {
		t.Fatalf("stop: %v", err)
	}

	restarted, err := NewWithPersistence(config)
	if err != nil {
		t.Fatalf("restart with persistence: %v", err)
	}
	defer func() {
		if err := restarted.Stop(context.Background()); err != nil {
			t.Fatalf("stop restarted service: %v", err)
		}
	}()

	tables := restarted.ListTables()
	if got, want := len(tables), 1; got != want {
		t.Fatalf("unexpected table count after restart: got %d want %d", got, want)
	}

	fetched, err := restarted.GetItem("mildstack-archive", "item#1")
	if err != nil {
		t.Fatalf("get item after restart: %v", err)
	}
	if got, want := fetched.Attributes["title"].Any(), "updated archive item"; got != want {
		t.Fatalf("unexpected item attribute after restart: got %q want %q", got, want)
	}
	if got, want := fetched.Attributes["version"].Any(), "2"; got != want {
		t.Fatalf("unexpected numeric item attribute after restart: got %q want %q", got, want)
	}
	if _, ok := fetched.Attributes["obsolete"]; ok {
		t.Fatal("expected removed attribute to stay removed after restart")
	}

	hook := runtime.NewStateHook()
	if err := restarted.AttachState(hook); err != nil {
		t.Fatalf("attach state after restart: %v", err)
	}
	value, ok := hook.Get(domain.StateKey)
	if !ok {
		t.Fatalf("expected restart snapshot for %q to be present", domain.StateKey)
	}
	state := value.(map[string]any)
	items := state["items"].([]any)
	if got, want := len(items), 1; got != want {
		t.Fatalf("unexpected snapshot item count after restart: got %d want %d", got, want)
	}
}

func TestServiceTableLifecycleTransitions(t *testing.T) {
	t.Helper()

	service := New()
	current := time.Date(2026, time.April, 18, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time {
		return current
	}

	created, err := service.CreateTable("mildstack-archive", "pk", "sk", "PAY_PER_REQUEST")
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	if got, want := created.Status, domain.TableStatusCreating; got != want {
		t.Fatalf("unexpected create status: got %q want %q", got, want)
	}

	described, err := service.DescribeTable("mildstack-archive")
	if err != nil {
		t.Fatalf("describe creating table: %v", err)
	}
	if got, want := described.Status, domain.TableStatusCreating; got != want {
		t.Fatalf("unexpected creating status: got %q want %q", got, want)
	}

	deletedWhileCreating, err := service.DeleteTable("mildstack-archive")
	if err != nil {
		t.Fatalf("delete creating table: %v", err)
	}
	if got, want := deletedWhileCreating.Status, domain.TableStatusDeleting; got != want {
		t.Fatalf("unexpected deleting status: got %q want %q", got, want)
	}

	current = current.Add(250 * time.Millisecond)
	if _, err := service.DescribeTable("mildstack-archive"); err == nil {
		t.Fatal("expected describe on deleted table to fail")
	}

	tables := service.ListTables()
	for _, table := range tables {
		if table.Name == "mildstack-archive" {
			t.Fatalf("expected deleted table to be hidden from ListTables, got %+v", table)
		}
	}

	if _, err := service.DeleteTable("mildstack-archive"); err != nil {
		t.Fatalf("repeat delete should be idempotent: %v", err)
	}
}

func TestServiceRejectsDuplicateTableCreation(t *testing.T) {
	t.Helper()

	service := New()

	if _, err := service.CreateTable("mildstack-archive", "pk", "sk", "PAY_PER_REQUEST"); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if _, err := service.CreateTable("mildstack-archive", "pk", "sk", "PAY_PER_REQUEST"); err == nil {
		t.Fatal("expected duplicate table creation to fail")
	}
}

func TestServiceRejectsInvalidAndMissingRequests(t *testing.T) {
	t.Helper()

	service := New()

	if _, err := service.CreateTable("", "", "", ""); err == nil {
		t.Fatal("expected empty table name to fail")
	}
	if _, err := service.GetItem("missing", "item#1"); err == nil {
		t.Fatal("expected missing table lookup to fail")
	}
	if _, err := service.PutItem("missing", "item#1", map[string]domain.AttributeValue{"id": domain.StringValue("item#1")}); err == nil {
		t.Fatal("expected put on missing table to fail")
	}
	if err := service.DeleteItem("mildstack-records", "missing"); err == nil {
		t.Fatal("expected delete on missing item to fail")
	}
	if _, err := service.UpdateItem("mildstack-records", "missing", "SET title = :title", "attribute_exists(id)", nil, map[string]domain.AttributeValue{
		":title": domain.StringValue("updated"),
	}); err == nil {
		t.Fatal("expected failing condition to return an error")
	}
	if _, err := service.UpdateItem("mildstack-records", "example#1", "SET nested.path = :title", "", nil, map[string]domain.AttributeValue{
		":title": domain.StringValue("updated"),
	}); err == nil {
		t.Fatal("expected nested update paths to fail")
	}
}

func TestServiceStartAndStopCloseRepository(t *testing.T) {
	t.Helper()

	service, err := NewWithPersistence(StorageConfig{BaseDir: t.TempDir(), InstanceID: "instance-a"})
	if err != nil {
		t.Fatalf("new with persistence: %v", err)
	}

	if err := service.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := service.Stop(context.Background()); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if err := service.Stop(context.Background()); err != nil {
		t.Fatalf("stop should be idempotent: %v", err)
	}
}
