package infrastructure

import (
	"strings"
	"testing"
)

func TestRoutesUseAWSS3Surface(t *testing.T) {
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
		{method: "GET", path: "/", name: "s3.buckets.index"},
		{method: "POST", path: "/", name: "s3.buckets.create"},
		{method: "HEAD", path: "/:bucket", name: "s3.buckets.head"},
		{method: "DELETE", path: "/:bucket", name: "s3.buckets.delete"},
		{method: "GET", path: "/:bucket?location", name: "s3.buckets.location.show"},
		{method: "GET", path: "/:bucket?policy", name: "s3.buckets.policy.show"},
		{method: "PUT", path: "/:bucket?policy", name: "s3.buckets.policy.update"},
		{method: "DELETE", path: "/:bucket?policy", name: "s3.buckets.policy.delete"},
		{method: "GET", path: "/:bucket?encryption", name: "s3.buckets.encryption.show"},
		{method: "PUT", path: "/:bucket?encryption", name: "s3.buckets.encryption.update"},
		{method: "DELETE", path: "/:bucket?encryption", name: "s3.buckets.encryption.delete"},
		{method: "GET", path: "/:bucket?lifecycle", name: "s3.buckets.lifecycle.show"},
		{method: "PUT", path: "/:bucket?lifecycle", name: "s3.buckets.lifecycle.update"},
		{method: "DELETE", path: "/:bucket?lifecycle", name: "s3.buckets.lifecycle.delete"},
		{method: "GET", path: "/:bucket?cors", name: "s3.buckets.cors.show"},
		{method: "PUT", path: "/:bucket?cors", name: "s3.buckets.cors.update"},
		{method: "DELETE", path: "/:bucket?cors", name: "s3.buckets.cors.delete"},
		{method: "GET", path: "/:bucket?acl", name: "s3.buckets.acl.show"},
		{method: "PUT", path: "/:bucket?acl", name: "s3.buckets.acl.update"},
		{method: "GET", path: "/:bucket?tagging", name: "s3.buckets.tagging.show"},
		{method: "PUT", path: "/:bucket?tagging", name: "s3.buckets.tagging.update"},
		{method: "DELETE", path: "/:bucket?tagging", name: "s3.buckets.tagging.delete"},
		{method: "GET", path: "/:bucket?ownershipControls", name: "s3.buckets.ownership-controls.show"},
		{method: "PUT", path: "/:bucket?ownershipControls", name: "s3.buckets.ownership-controls.update"},
		{method: "DELETE", path: "/:bucket?ownershipControls", name: "s3.buckets.ownership-controls.delete"},
		{method: "GET", path: "/:bucket?publicAccessBlock", name: "s3.buckets.public-access-block.show"},
		{method: "PUT", path: "/:bucket?publicAccessBlock", name: "s3.buckets.public-access-block.update"},
		{method: "DELETE", path: "/:bucket?publicAccessBlock", name: "s3.buckets.public-access-block.delete"},
		{method: "GET", path: "/:bucket?notification", name: "s3.buckets.notification.show"},
		{method: "PUT", path: "/:bucket?notification", name: "s3.buckets.notification.update"},
		{method: "GET", path: "/:bucket?logging", name: "s3.buckets.logging.show"},
		{method: "PUT", path: "/:bucket?logging", name: "s3.buckets.logging.update"},
		{method: "GET", path: "/:bucket?replication", name: "s3.buckets.replication.show"},
		{method: "PUT", path: "/:bucket?replication", name: "s3.buckets.replication.update"},
		{method: "DELETE", path: "/:bucket?replication", name: "s3.buckets.replication.delete"},
		{method: "GET", path: "/:bucket?versioning", name: "s3.buckets.versioning.show"},
		{method: "PUT", path: "/:bucket?versioning", name: "s3.buckets.versioning.update"},
		{method: "GET", path: "/:bucket?versions", name: "s3.objects.versions"},
		{method: "GET", path: "/:bucket?object-lock", name: "s3.buckets.object-lock.show"},
		{method: "PUT", path: "/:bucket?object-lock", name: "s3.buckets.object-lock.update"},
		{method: "GET", path: "/:bucket/:object?retention", name: "s3.objects.retention.show"},
		{method: "PUT", path: "/:bucket/:object?retention", name: "s3.objects.retention.update"},
		{method: "GET", path: "/:bucket/:object?legal-hold", name: "s3.objects.legal-hold.show"},
		{method: "PUT", path: "/:bucket/:object?legal-hold", name: "s3.objects.legal-hold.update"},
		{method: "GET", path: "/:bucket/:object?acl", name: "s3.objects.acl.show"},
		{method: "PUT", path: "/:bucket/:object?acl", name: "s3.objects.acl.update"},
		{method: "GET", path: "/:bucket/:object?tagging", name: "s3.objects.tagging.show"},
		{method: "PUT", path: "/:bucket/:object?tagging", name: "s3.objects.tagging.update"},
		{method: "DELETE", path: "/:bucket/:object?tagging", name: "s3.objects.tagging.delete"},
		{method: "GET", path: "/:bucket", name: "s3.objects.list-v1"},
		{method: "GET", path: "/:bucket?list-type=2", name: "s3.objects.list-v2"},
		{method: "POST", path: "/:bucket?delete", name: "s3.objects.delete-batch"},
		{method: "GET", path: "/:bucket/:object", name: "s3.objects.show"},
		{method: "HEAD", path: "/:bucket/:object", name: "s3.objects.head"},
		{method: "PUT", path: "/:bucket/:object", name: "s3.objects.update"},
		{method: "DELETE", path: "/:bucket/:object", name: "s3.objects.delete"},
		{method: "GET", path: "/:bucket?uploads", name: "s3.multipart.uploads.index"},
		{method: "GET", path: "/:bucket/:object?uploadId=:upload", name: "s3.multipart.uploads.parts.index"},
		{method: "POST", path: "/:bucket/:object?uploads", name: "s3.multipart.uploads.create"},
		{method: "PUT", path: "/:bucket/:object?partNumber=:part&uploadId=:upload", name: "s3.multipart.uploads.part"},
		{method: "POST", path: "/:bucket/:object?uploadId=:upload", name: "s3.multipart.uploads.complete"},
		{method: "DELETE", path: "/:bucket/:object?uploadId=:upload", name: "s3.multipart.uploads.abort"},
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

	for _, forbidden := range []string{
		"lifecycle-configuration",
		"notification-configuration",
		"directory-buckets",
		"s3-express",
		"object-lambda",
		"metadata-configuration",
		"metadata-table",
		"inventory",
		"analytics",
		"accelerate",
		"request-payment",
		"website",
		"metrics",
		"select-object-content",
		"write-get-object-response",
	} {
		for _, route := range routes {
			if strings.Contains(route.Path, forbidden) || strings.Contains(route.Name, forbidden) {
				t.Fatalf("unexpected deferred route exposed: %s matched %q", route.Path, forbidden)
			}
		}
	}
}
