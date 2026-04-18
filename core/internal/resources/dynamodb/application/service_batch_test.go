package application

import (
	"errors"
	"fmt"
	"testing"

	"github.com/michasdev/mildstack/core/internal/resources/dynamodb/domain"
)

func TestServiceBatchWriteItemReturnsUnprocessedItemsAfterSupportedLimit(t *testing.T) {
	t.Helper()

	service := New()
	if _, err := service.CreateTable("mildstack-batch", "id", "", "PAY_PER_REQUEST"); err != nil {
		t.Fatalf("create table: %v", err)
	}

	requests := make([]BatchWriteRequestItem, 0, 26)
	for i := 1; i <= 26; i++ {
		requests = append(requests, BatchWriteRequestItem{
			PutItem: batchTestItemDocument(fmt.Sprintf("item#%02d", i), fmt.Sprintf("title-%02d", i)),
		})
	}

	result, err := service.BatchWriteItem(BatchWriteItemRequest{
		Tables: []BatchWriteTableRequest{
			{
				Table:    "mildstack-batch",
				Requests: requests,
			},
		},
	})
	if err != nil {
		t.Fatalf("batch write: %v", err)
	}
	if got, want := len(result.Unprocessed), 1; got != want {
		t.Fatalf("unexpected unprocessed table count: got %d want %d", got, want)
	}
	if got, want := len(result.Unprocessed[0].Requests), 1; got != want {
		t.Fatalf("unexpected unprocessed request count: got %d want %d", got, want)
	}
	if got, want := result.Unprocessed[0].Requests[0].PutItem["id"].Any(), "item#26"; got != want {
		t.Fatalf("unexpected unprocessed item id: got %v want %v", got, want)
	}

	if _, err := service.GetItem("mildstack-batch", "item#25"); err != nil {
		t.Fatalf("expected supported batch write item to persist: %v", err)
	}
	if _, err := service.GetItem("mildstack-batch", "item#26"); err == nil {
		t.Fatal("expected item beyond supported limit to remain unprocessed")
	}
}

func TestServiceBatchGetItemReturnsDeterministicResultsAndUnprocessedKeys(t *testing.T) {
	t.Helper()

	service := New()
	if _, err := service.CreateTable("mildstack-batch", "id", "", "PAY_PER_REQUEST"); err != nil {
		t.Fatalf("create table: %v", err)
	}
	for i := 1; i <= 3; i++ {
		if _, err := service.PutItem("mildstack-batch", fmt.Sprintf("item#%03d", i), batchTestItemDocument(fmt.Sprintf("item#%03d", i), fmt.Sprintf("title-%03d", i))); err != nil {
			t.Fatalf("seed item %d: %v", i, err)
		}
	}

	keys := make([]map[string]domain.AttributeValue, 0, 101)
	for i := 1; i <= 101; i++ {
		keys = append(keys, map[string]domain.AttributeValue{
			"id": domain.StringValue(fmt.Sprintf("item#%03d", i)),
		})
	}

	result, err := service.BatchGetItem(BatchGetItemRequest{
		Tables: []BatchGetTableRequest{
			{
				Table: "mildstack-batch",
				Keys:  keys,
			},
		},
	})
	if err != nil {
		t.Fatalf("batch get: %v", err)
	}
	if got, want := len(result.Responses), 1; got != want {
		t.Fatalf("unexpected batch get table response count: got %d want %d", got, want)
	}
	if got, want := len(result.Responses[0].Items), 3; got != want {
		t.Fatalf("unexpected batch get item count: got %d want %d", got, want)
	}
	if got, want := result.Responses[0].Items[0].Key, "item#001"; got != want {
		t.Fatalf("unexpected first batch get key: got %q want %q", got, want)
	}
	if got, want := result.Responses[0].Items[2].Key, "item#003"; got != want {
		t.Fatalf("unexpected third batch get key: got %q want %q", got, want)
	}
	if got, want := len(result.Unprocessed), 1; got != want {
		t.Fatalf("unexpected unprocessed table count: got %d want %d", got, want)
	}
	if got, want := len(result.Unprocessed[0].Keys), 1; got != want {
		t.Fatalf("unexpected unprocessed key count: got %d want %d", got, want)
	}
	if got, want := result.Unprocessed[0].Keys[0]["id"].Any(), "item#101"; got != want {
		t.Fatalf("unexpected unprocessed key value: got %v want %v", got, want)
	}
}

func TestServiceTransactItemsAreAtomicAndReturnCancellationReasons(t *testing.T) {
	t.Helper()

	service := New()
	if _, err := service.CreateTable("mildstack-transact", "id", "", "PAY_PER_REQUEST"); err != nil {
		t.Fatalf("create table: %v", err)
	}

	if err := service.TransactWriteItems(TransactWriteItemsRequest{
		Items: []TransactWriteItem{
			{
				Table:   "mildstack-transact",
				PutItem: batchTestItemDocument("item#1", "title-1"),
			},
			{
				Table:   "mildstack-transact",
				PutItem: batchTestItemDocument("item#2", "title-2"),
			},
		},
	}); err != nil {
		t.Fatalf("transaction write success: %v", err)
	}

	fetched, err := service.TransactGetItems(TransactGetItemsRequest{
		Items: []TransactGetItem{
			{
				Table: "mildstack-transact",
				Key:   map[string]domain.AttributeValue{"id": domain.StringValue("item#2")},
			},
			{
				Table: "mildstack-transact",
				Key:   map[string]domain.AttributeValue{"id": domain.StringValue("missing")},
			},
			{
				Table: "mildstack-transact",
				Key:   map[string]domain.AttributeValue{"id": domain.StringValue("item#1")},
			},
		},
	})
	if err != nil {
		t.Fatalf("transaction get: %v", err)
	}
	if got, want := len(fetched.Items), 3; got != want {
		t.Fatalf("unexpected transact get item count: got %d want %d", got, want)
	}
	if fetched.Items[1].Item != nil {
		t.Fatal("expected missing transaction get item to be omitted")
	}
	if got, want := fetched.Items[0].Item.Key, "item#2"; got != want {
		t.Fatalf("unexpected first transaction get key: got %q want %q", got, want)
	}
	if got, want := fetched.Items[2].Item.Key, "item#1"; got != want {
		t.Fatalf("unexpected third transaction get key: got %q want %q", got, want)
	}

	err = service.TransactWriteItems(TransactWriteItemsRequest{
		Items: []TransactWriteItem{
			{
				Table:   "mildstack-transact",
				PutItem: batchTestItemDocument("item#3", "title-3"),
			},
			{
				Table: "mildstack-transact",
				DeleteKey: map[string]domain.AttributeValue{
					"id": domain.StringValue("item#3"),
				},
			},
		},
	})
	if err == nil {
		t.Fatal("expected conflicting transaction to fail")
	}
	var canceled *TransactionCanceledError
	if !errors.As(err, &canceled) {
		t.Fatalf("expected transaction canceled error, got %T: %v", err, err)
	}
	if got, want := len(canceled.Reasons), 2; got != want {
		t.Fatalf("unexpected cancellation reason count: got %d want %d", got, want)
	}
	if got, want := canceled.Reasons[0].Code, "TransactionConflict"; got != want {
		t.Fatalf("unexpected cancellation reason code: got %q want %q", got, want)
	}
	if got, err := service.GetItem("mildstack-transact", "item#3"); err == nil {
		t.Fatalf("expected conflicting transaction to remain uncommitted, got item %#v", got)
	}
}

func batchTestItemDocument(id, title string) map[string]domain.AttributeValue {
	return map[string]domain.AttributeValue{
		"id":    domain.StringValue(id),
		"title": domain.StringValue(title),
	}
}
