package infrastructure

import (
	"strings"
	"testing"
)

func TestRoutesUseS3ServiceSegment(t *testing.T) {
	t.Helper()

	routes := Routes()
	if got, want := len(routes), 35; got != want {
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
		{method: "GET", path: "/s3/buckets/:bucket/policy", name: "s3.buckets.policy.show"},
		{method: "PUT", path: "/s3/buckets/:bucket/policy", name: "s3.buckets.policy.update"},
		{method: "DELETE", path: "/s3/buckets/:bucket/policy", name: "s3.buckets.policy.delete"},
		{method: "GET", path: "/s3/buckets/:bucket/encryption", name: "s3.buckets.encryption.show"},
		{method: "PUT", path: "/s3/buckets/:bucket/encryption", name: "s3.buckets.encryption.update"},
		{method: "DELETE", path: "/s3/buckets/:bucket/encryption", name: "s3.buckets.encryption.delete"},
		{method: "GET", path: "/s3/buckets/:bucket/lifecycle", name: "s3.buckets.lifecycle.show"},
		{method: "PUT", path: "/s3/buckets/:bucket/lifecycle", name: "s3.buckets.lifecycle.update"},
		{method: "DELETE", path: "/s3/buckets/:bucket/lifecycle", name: "s3.buckets.lifecycle.delete"},
		{method: "GET", path: "/s3/buckets/:bucket/cors", name: "s3.buckets.cors.show"},
		{method: "PUT", path: "/s3/buckets/:bucket/cors", name: "s3.buckets.cors.update"},
		{method: "DELETE", path: "/s3/buckets/:bucket/cors", name: "s3.buckets.cors.delete"},
		{method: "GET", path: "/s3/buckets/:bucket/acl", name: "s3.buckets.acl.show"},
		{method: "PUT", path: "/s3/buckets/:bucket/acl", name: "s3.buckets.acl.update"},
		{method: "GET", path: "/s3/buckets/:bucket/tagging", name: "s3.buckets.tagging.show"},
		{method: "PUT", path: "/s3/buckets/:bucket/tagging", name: "s3.buckets.tagging.update"},
		{method: "DELETE", path: "/s3/buckets/:bucket/tagging", name: "s3.buckets.tagging.delete"},
		{method: "GET", path: "/s3/buckets/:bucket/versioning", name: "s3.buckets.versioning.show"},
		{method: "PUT", path: "/s3/buckets/:bucket/versioning", name: "s3.buckets.versioning.update"},
		{method: "GET", path: "/s3/buckets/:bucket/objects/versions", name: "s3.objects.versions"},
		{method: "GET", path: "/s3/buckets/:bucket/objects", name: "s3.objects.list-v1"},
		{method: "GET", path: "/s3/buckets/:bucket/objects/v2", name: "s3.objects.list-v2"},
		{method: "POST", path: "/s3/buckets/:bucket/objects/delete", name: "s3.objects.delete-batch"},
		{method: "GET", path: "/s3/buckets/:bucket/objects/:object", name: "s3.objects.show"},
		{method: "HEAD", path: "/s3/buckets/:bucket/objects/:object", name: "s3.objects.head"},
		{method: "PUT", path: "/s3/buckets/:bucket/objects/:object", name: "s3.objects.update"},
		{method: "DELETE", path: "/s3/buckets/:bucket/objects/:object", name: "s3.objects.delete"},
		{method: "POST", path: "/s3/buckets/:bucket/objects/:object/uploads", name: "s3.multipart.uploads.create"},
		{method: "PUT", path: "/s3/buckets/:bucket/objects/:object/uploads/:upload/parts/:part", name: "s3.multipart.uploads.part"},
		{method: "POST", path: "/s3/buckets/:bucket/objects/:object/uploads/:upload/complete", name: "s3.multipart.uploads.complete"},
		{method: "DELETE", path: "/s3/buckets/:bucket/objects/:object/uploads/:upload", name: "s3.multipart.uploads.abort"},
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

	for _, route := range routes {
		if strings.Contains(route.Name, "list-multipart") || strings.Contains(route.Name, "list-parts") {
			t.Fatalf("unexpected multipart listing route registered: %s", route.Name)
		}
	}
}
