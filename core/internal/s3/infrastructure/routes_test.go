package infrastructure

import "testing"

func TestRoutesUseS3ServiceSegment(t *testing.T) {
	t.Helper()

	routes := Routes()
	if got, want := len(routes), 9; got != want {
		t.Fatalf("unexpected route count: got %d want %d", got, want)
	}

	expected := []struct {
		method string
		path   string
		name   string
	}{
		{method: "GET", path: "/s3/buckets", name: "s3.buckets.index"},
		{method: "POST", path: "/s3/buckets", name: "s3.buckets.create"},
		{method: "HEAD", path: "/s3/buckets/:bucket", name: "s3.buckets.head"},
		{method: "DELETE", path: "/s3/buckets/:bucket", name: "s3.buckets.delete"},
		{method: "GET", path: "/s3/buckets/:bucket/objects", name: "s3.objects.index"},
		{method: "GET", path: "/s3/buckets/:bucket/objects/:object", name: "s3.objects.show"},
		{method: "HEAD", path: "/s3/buckets/:bucket/objects/:object", name: "s3.objects.head"},
		{method: "PUT", path: "/s3/buckets/:bucket/objects/:object", name: "s3.objects.update"},
		{method: "DELETE", path: "/s3/buckets/:bucket/objects/:object", name: "s3.objects.delete"},
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
