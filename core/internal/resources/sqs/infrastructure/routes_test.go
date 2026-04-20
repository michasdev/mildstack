package infrastructure

import "testing"

func TestRoutesUseSQSServiceSegment(t *testing.T) {
	t.Helper()

	routes := Routes()
	if got, want := len(routes), 7; got != want {
		t.Fatalf("unexpected route count: got %d want %d", got, want)
	}

	expected := []struct {
		method string
		path   string
		name   string
	}{
		{method: "GET", path: "/sqs/queues", name: "sqs.queues.index"},
		{method: "POST", path: "/sqs/queues", name: "sqs.queues.create"},
		{method: "GET", path: "/sqs/queues/:queue", name: "sqs.queues.show"},
		{method: "DELETE", path: "/sqs/queues/:queue", name: "sqs.queues.delete"},
		{method: "GET", path: "/sqs/queues/:queue/messages", name: "sqs.messages.receive"},
		{method: "POST", path: "/sqs/queues/:queue/messages", name: "sqs.messages.send"},
		{method: "DELETE", path: "/sqs/queues/:queue/messages/:receiptHandle", name: "sqs.messages.delete"},
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
