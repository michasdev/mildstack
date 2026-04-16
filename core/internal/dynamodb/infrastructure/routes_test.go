package infrastructure

import "testing"

func TestRoutesUseDynamoDBServiceSegment(t *testing.T) {
	t.Helper()

	routes := Routes()
	if got, want := len(routes), 4; got != want {
		t.Fatalf("unexpected route count: got %d want %d", got, want)
	}

	expected := []struct {
		method string
		path   string
		name   string
	}{
		{method: "GET", path: "/dynamodb/tables", name: "dynamodb.tables.index"},
		{method: "GET", path: "/dynamodb/tables/:table", name: "dynamodb.tables.show"},
		{method: "GET", path: "/dynamodb/tables/:table/items", name: "dynamodb.items.index"},
		{method: "GET", path: "/dynamodb/tables/:table/items/:item", name: "dynamodb.items.show"},
	}

	for i, route := range routes {
		if route.Method != expected[i].method {
			t.Fatalf("unexpected method at %d: got %q want %q", i, route.Method, expected[i].method)
		}
		if route.Path != expected[i].path {
			t.Fatalf("unexpected path at %d: got %q want %q", i, route.Path, expected[i].path)
		}
		if route.Name != expected[i].name {
			t.Fatalf("unexpected name at %d: got %q want %q", i, route.Name, expected[i].name)
		}
	}
}
