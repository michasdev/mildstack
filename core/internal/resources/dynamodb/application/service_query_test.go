package application

import (
	"testing"

	"github.com/michasdev/mildstack/core/internal/resources/dynamodb/domain"
)

func TestServiceQuerySupportsSortKeyPredicatesAndOrdering(t *testing.T) {
	t.Helper()

	service := New()
	if _, err := service.CreateTable("mildstack-reads", "pk", "sk", "PAY_PER_REQUEST"); err != nil {
		t.Fatalf("create table: %v", err)
	}
	seedQueryItems(t, service)

	tests := []struct {
		name      string
		expr      string
		values    map[string]domain.AttributeValue
		forward   *bool
		limit     *int
		wantSKs   []string
		wantCount int
	}{
		{
			name: "equal",
			expr: "pk = :pk AND sk = :sk",
			values: map[string]domain.AttributeValue{
				":pk": domain.StringValue("series#1"),
				":sk": domain.StringValue("002"),
			},
			wantSKs:   []string{"002"},
			wantCount: 1,
		},
		{
			name: "less than",
			expr: "pk = :pk AND sk < :sk",
			values: map[string]domain.AttributeValue{
				":pk": domain.StringValue("series#1"),
				":sk": domain.StringValue("002"),
			},
			wantSKs:   []string{"001"},
			wantCount: 1,
		},
		{
			name: "less than or equal",
			expr: "pk = :pk AND sk <= :sk",
			values: map[string]domain.AttributeValue{
				":pk": domain.StringValue("series#1"),
				":sk": domain.StringValue("002"),
			},
			wantSKs:   []string{"001", "002"},
			wantCount: 2,
		},
		{
			name: "greater than",
			expr: "pk = :pk AND sk > :sk",
			values: map[string]domain.AttributeValue{
				":pk": domain.StringValue("series#1"),
				":sk": domain.StringValue("002"),
			},
			wantSKs:   []string{"003"},
			wantCount: 1,
		},
		{
			name: "greater than or equal",
			expr: "pk = :pk AND sk >= :sk",
			values: map[string]domain.AttributeValue{
				":pk": domain.StringValue("series#1"),
				":sk": domain.StringValue("002"),
			},
			wantSKs:   []string{"002", "003"},
			wantCount: 2,
		},
		{
			name: "between",
			expr: "pk = :pk AND sk BETWEEN :start AND :end",
			values: map[string]domain.AttributeValue{
				":pk":    domain.StringValue("series#1"),
				":start": domain.StringValue("002"),
				":end":   domain.StringValue("003"),
			},
			wantSKs:   []string{"002", "003"},
			wantCount: 2,
		},
		{
			name: "begins_with",
			expr: "pk = :pk AND begins_with(sk, :prefix)",
			values: map[string]domain.AttributeValue{
				":pk":     domain.StringValue("series#1"),
				":prefix": domain.StringValue("00"),
			},
			wantSKs:   []string{"001", "002", "003"},
			wantCount: 3,
		},
		{
			name: "descending limit",
			expr: "pk = :pk AND sk BETWEEN :start AND :end",
			values: map[string]domain.AttributeValue{
				":pk":    domain.StringValue("series#1"),
				":start": domain.StringValue("001"),
				":end":   domain.StringValue("003"),
			},
			forward:   boolPtr(false),
			limit:     intPtr(2),
			wantSKs:   []string{"003", "002"},
			wantCount: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := service.Query("mildstack-reads", tc.expr, "", nil, tc.values, tc.limit, nil, tc.forward)
			if err != nil {
				t.Fatalf("query: %v", err)
			}
			if got, want := result.Count, tc.wantCount; got != want {
				t.Fatalf("unexpected query count: got %d want %d", got, want)
			}
			if got, want := len(result.Items), len(tc.wantSKs); got != want {
				t.Fatalf("unexpected query item count: got %d want %d", got, want)
			}
			for i, wantSK := range tc.wantSKs {
				if got := attrString(result.Items[i].Attributes["sk"]); got != wantSK {
					t.Fatalf("unexpected query sort key at %d: got %q want %q", i, got, wantSK)
				}
			}
		})
	}
}

func TestServiceScanSupportsPaginationAndFilters(t *testing.T) {
	t.Helper()

	service := New()
	if _, err := service.CreateTable("mildstack-reads", "pk", "sk", "PAY_PER_REQUEST"); err != nil {
		t.Fatalf("create table: %v", err)
	}
	seedQueryItems(t, service)

	firstPage, err := service.Scan("mildstack-reads", "begins_with(title, :prefix)", nil, map[string]domain.AttributeValue{
		":prefix": domain.StringValue("keep"),
	}, intPtr(1), nil)
	if err != nil {
		t.Fatalf("scan first page: %v", err)
	}
	if got, want := firstPage.Count, 0; got != want {
		t.Fatalf("unexpected first page count: got %d want %d", got, want)
	}
	if got, want := firstPage.ScannedCount, 1; got != want {
		t.Fatalf("unexpected first page scanned count: got %d want %d", got, want)
	}
	if len(firstPage.Items) != 0 {
		t.Fatalf("expected first scan page to be empty, got %#v", firstPage.Items)
	}
	if got, want := attrString(firstPage.LastEvaluatedKey["sk"]), "001"; got != want {
		t.Fatalf("unexpected first page cursor: got %q want %q", got, want)
	}

	secondPage, err := service.Scan("mildstack-reads", "begins_with(title, :prefix)", nil, map[string]domain.AttributeValue{
		":prefix": domain.StringValue("keep"),
	}, intPtr(1), firstPage.LastEvaluatedKey)
	if err != nil {
		t.Fatalf("scan second page: %v", err)
	}
	if got, want := secondPage.Count, 1; got != want {
		t.Fatalf("unexpected second page count: got %d want %d", got, want)
	}
	if got, want := attrString(secondPage.Items[0].Attributes["title"]), "keep-two"; got != want {
		t.Fatalf("unexpected second page title: got %q want %q", got, want)
	}
	if got, want := attrString(secondPage.LastEvaluatedKey["sk"]), "002"; got != want {
		t.Fatalf("unexpected second page cursor: got %q want %q", got, want)
	}
}

func TestServiceReadPlannerRejectsUnsupportedExpressions(t *testing.T) {
	t.Helper()

	service := New()
	if _, err := service.CreateTable("mildstack-reads", "pk", "sk", "PAY_PER_REQUEST"); err != nil {
		t.Fatalf("create table: %v", err)
	}
	seedQueryItems(t, service)

	if _, err := service.Query("mildstack-reads", "pk = :pk AND sk = :sk", "begins_with(meta.title, :prefix)", nil, map[string]domain.AttributeValue{
		":pk":     domain.StringValue("series#1"),
		":sk":     domain.StringValue("001"),
		":prefix": domain.StringValue("keep"),
	}, nil, nil, nil); err == nil {
		t.Fatal("expected nested filter expression to fail")
	}
	if _, err := service.Query("mildstack-reads", "", "", nil, nil, nil, nil, nil); err == nil {
		t.Fatal("expected missing key condition expression to fail")
	}
	if _, err := service.Scan("mildstack-reads", "title <> :title", nil, map[string]domain.AttributeValue{
		":title": domain.StringValue("skip-one"),
	}, nil, map[string]domain.AttributeValue{
		"pk": domain.StringValue("series#1"),
	}); err == nil {
		t.Fatal("expected scan with invalid key cursor to fail")
	}
}

func seedQueryItems(t *testing.T, service *Service) {
	t.Helper()

	items := []domain.Item{
		{
			Table: "mildstack-reads",
			Key:   "row#1",
			Attributes: map[string]domain.AttributeValue{
				"pk":    domain.StringValue("series#1"),
				"sk":    domain.StringValue("001"),
				"title": domain.StringValue("skip-one"),
			},
		},
		{
			Table: "mildstack-reads",
			Key:   "row#2",
			Attributes: map[string]domain.AttributeValue{
				"pk":    domain.StringValue("series#1"),
				"sk":    domain.StringValue("002"),
				"title": domain.StringValue("keep-two"),
			},
		},
		{
			Table: "mildstack-reads",
			Key:   "row#3",
			Attributes: map[string]domain.AttributeValue{
				"pk":    domain.StringValue("series#1"),
				"sk":    domain.StringValue("003"),
				"title": domain.StringValue("keep-three"),
			},
		},
		{
			Table: "mildstack-reads",
			Key:   "row#4",
			Attributes: map[string]domain.AttributeValue{
				"pk":    domain.StringValue("series#2"),
				"sk":    domain.StringValue("001"),
				"title": domain.StringValue("other-series"),
			},
		},
	}

	for _, item := range items {
		if _, err := service.PutItem(item.Table, item.Key, item.Attributes); err != nil {
			t.Fatalf("seed item %s/%s: %v", item.Table, item.Key, err)
		}
	}
}

func attrString(value domain.AttributeValue) string {
	if value.S == nil {
		return ""
	}
	return *value.S
}

func intPtr(value int) *int {
	return &value
}

func boolPtr(value bool) *bool {
	return &value
}
