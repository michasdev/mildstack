package infrastructure

import "testing"

func TestRoutesUseS3ServiceSegment(t *testing.T) {
	t.Helper()

	routes := Routes()
	if got, want := len(routes), 62; got != want {
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
		{method: "GET", path: "/s3/buckets/:bucket/location", name: "s3.buckets.location.show"},
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
		{method: "GET", path: "/s3/buckets/:bucket/ownership-controls", name: "s3.buckets.ownership-controls.show"},
		{method: "PUT", path: "/s3/buckets/:bucket/ownership-controls", name: "s3.buckets.ownership-controls.update"},
		{method: "DELETE", path: "/s3/buckets/:bucket/ownership-controls", name: "s3.buckets.ownership-controls.delete"},
		{method: "GET", path: "/s3/buckets/:bucket/public-access-block", name: "s3.buckets.public-access-block.show"},
		{method: "PUT", path: "/s3/buckets/:bucket/public-access-block", name: "s3.buckets.public-access-block.update"},
		{method: "DELETE", path: "/s3/buckets/:bucket/public-access-block", name: "s3.buckets.public-access-block.delete"},
		{method: "GET", path: "/s3/buckets/:bucket/notification", name: "s3.buckets.notification.show"},
		{method: "PUT", path: "/s3/buckets/:bucket/notification", name: "s3.buckets.notification.update"},
		{method: "GET", path: "/s3/buckets/:bucket/logging", name: "s3.buckets.logging.show"},
		{method: "PUT", path: "/s3/buckets/:bucket/logging", name: "s3.buckets.logging.update"},
		{method: "GET", path: "/s3/buckets/:bucket/replication", name: "s3.buckets.replication.show"},
		{method: "PUT", path: "/s3/buckets/:bucket/replication", name: "s3.buckets.replication.update"},
		{method: "DELETE", path: "/s3/buckets/:bucket/replication", name: "s3.buckets.replication.delete"},
		{method: "GET", path: "/s3/buckets/:bucket/versioning", name: "s3.buckets.versioning.show"},
		{method: "PUT", path: "/s3/buckets/:bucket/versioning", name: "s3.buckets.versioning.update"},
		{method: "GET", path: "/s3/buckets/:bucket/objects/versions", name: "s3.objects.versions"},
		{method: "GET", path: "/s3/buckets/:bucket/object-lock", name: "s3.buckets.object-lock.show"},
		{method: "PUT", path: "/s3/buckets/:bucket/object-lock", name: "s3.buckets.object-lock.update"},
		{method: "GET", path: "/s3/buckets/:bucket/objects/:object/retention", name: "s3.objects.retention.show"},
		{method: "PUT", path: "/s3/buckets/:bucket/objects/:object/retention", name: "s3.objects.retention.update"},
		{method: "GET", path: "/s3/buckets/:bucket/objects/:object/legal-hold", name: "s3.objects.legal-hold.show"},
		{method: "PUT", path: "/s3/buckets/:bucket/objects/:object/legal-hold", name: "s3.objects.legal-hold.update"},
		{method: "GET", path: "/s3/buckets/:bucket/objects/:object/acl", name: "s3.objects.acl.show"},
		{method: "PUT", path: "/s3/buckets/:bucket/objects/:object/acl", name: "s3.objects.acl.update"},
		{method: "GET", path: "/s3/buckets/:bucket/objects/:object/tagging", name: "s3.objects.tagging.show"},
		{method: "PUT", path: "/s3/buckets/:bucket/objects/:object/tagging", name: "s3.objects.tagging.update"},
		{method: "DELETE", path: "/s3/buckets/:bucket/objects/:object/tagging", name: "s3.objects.tagging.delete"},
		{method: "GET", path: "/s3/buckets/:bucket/objects", name: "s3.objects.list-v1"},
		{method: "GET", path: "/s3/buckets/:bucket/objects/v2", name: "s3.objects.list-v2"},
		{method: "POST", path: "/s3/buckets/:bucket/objects/delete", name: "s3.objects.delete-batch"},
		{method: "GET", path: "/s3/buckets/:bucket/objects/:object", name: "s3.objects.show"},
		{method: "HEAD", path: "/s3/buckets/:bucket/objects/:object", name: "s3.objects.head"},
		{method: "PUT", path: "/s3/buckets/:bucket/objects/:object", name: "s3.objects.update"},
		{method: "DELETE", path: "/s3/buckets/:bucket/objects/:object", name: "s3.objects.delete"},
		{method: "GET", path: "/s3/buckets/:bucket/uploads", name: "s3.multipart.uploads.index"},
		{method: "GET", path: "/s3/buckets/:bucket/uploads/:upload/parts", name: "s3.multipart.uploads.parts.index"},
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

	expectedNames := map[string]bool{
		"s3.multipart.uploads.index":       false,
		"s3.multipart.uploads.parts.index": false,
	}
	for _, route := range routes {
		if _, ok := expectedNames[route.Name]; ok {
			expectedNames[route.Name] = true
		}
	}
	for name, seen := range expectedNames {
		if !seen {
			t.Fatalf("expected multipart listing route to be registered: %s", name)
		}
	}
}
