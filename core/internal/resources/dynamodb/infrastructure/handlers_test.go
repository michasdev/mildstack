package infrastructure_test

import (
	"testing"

	"github.com/michasdev/mildstack/core/internal/resources/dynamodb/application"
	"github.com/michasdev/mildstack/core/internal/resources/dynamodb/infrastructure"
)

func TestHandlersDriveRealServiceAndReturnCopies(t *testing.T) {
	t.Helper()

	service := application.New()
	handlers := infrastructure.NewHandlers(service)

	tables := handlers.ListTables()
	if got, want := len(tables.Tables), 1; got != want {
		t.Fatalf("unexpected initial table count: got %d want %d", got, want)
	}
	tables.Tables[0].Name = "mutated"
	again := handlers.ListTables()
	if got, want := again.Tables[0].Name, "mildstack-records"; got != want {
		t.Fatalf("table payload was not copied: got %q want %q", got, want)
	}

	createResp, err := handlers.CreateTable(infrastructure.CreateTableRequest{
		Name:         "mildstack-archive",
		PartitionKey: "pk",
		SortKey:      "sk",
		BillingMode:  "PAY_PER_REQUEST",
	})
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	if got, want := createResp.Table.Name, "mildstack-archive"; got != want {
		t.Fatalf("unexpected table name: got %q want %q", got, want)
	}

	putResp, err := handlers.PutItem(infrastructure.PutItemRequest{
		Table: createResp.Table.Name,
		Key:   "item#1",
		Attributes: map[string]string{
			"id":    "item#1",
			"title": "archive item",
		},
	})
	if err != nil {
		t.Fatalf("put item: %v", err)
	}
	if got, want := putResp.Item.Key, "item#1"; got != want {
		t.Fatalf("unexpected item key: got %q want %q", got, want)
	}

	getResp, err := handlers.GetItem(infrastructure.GetItemRequest{
		Table: createResp.Table.Name,
		Key:   putResp.Item.Key,
	})
	if err != nil {
		t.Fatalf("get item: %v", err)
	}
	if got, want := getResp.Item.Attributes["title"], "archive item"; got != want {
		t.Fatalf("unexpected item title: got %q want %q", got, want)
	}

	getResp.Item.Attributes["title"] = "mutated"
	againItem, err := handlers.GetItem(infrastructure.GetItemRequest{
		Table: createResp.Table.Name,
		Key:   putResp.Item.Key,
	})
	if err != nil {
		t.Fatalf("get item: %v", err)
	}
	if got, want := againItem.Item.Attributes["title"], "archive item"; got != want {
		t.Fatalf("item payload was not copied: got %q want %q", got, want)
	}

	deleteResp, err := handlers.DeleteItem(infrastructure.DeleteItemRequest{
		Table: createResp.Table.Name,
		Key:   putResp.Item.Key,
	})
	if err != nil {
		t.Fatalf("delete item: %v", err)
	}
	if !deleteResp.Deleted {
		t.Fatal("expected delete response to report success")
	}
	if _, err := handlers.GetItem(infrastructure.GetItemRequest{
		Table: createResp.Table.Name,
		Key:   putResp.Item.Key,
	}); err == nil {
		t.Fatal("expected deleted item lookup to fail")
	}
}

func TestHandlersSurfaceServiceErrors(t *testing.T) {
	t.Helper()

	handlers := infrastructure.NewHandlers(application.New())

	if _, err := handlers.CreateTable(infrastructure.CreateTableRequest{}); err == nil {
		t.Fatal("expected empty table creation to fail")
	}
	if _, err := handlers.GetItem(infrastructure.GetItemRequest{Table: "missing", Key: "item#1"}); err == nil {
		t.Fatal("expected missing table lookup to fail")
	}
	if _, err := handlers.PutItem(infrastructure.PutItemRequest{Table: "missing", Key: "item#1"}); err == nil {
		t.Fatal("expected put on missing table to fail")
	}
	if _, err := handlers.DeleteItem(infrastructure.DeleteItemRequest{Table: "mildstack-records", Key: "missing"}); err == nil {
		t.Fatal("expected delete on missing item to fail")
	}
}
