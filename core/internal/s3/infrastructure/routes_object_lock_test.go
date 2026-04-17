package infrastructure

import "testing"

func TestObjectLockRoutesAreRegisteredOnce(t *testing.T) {
	t.Helper()

	routes := Routes()
	if got, want := len(routes), 48; got != want {
		t.Fatalf("unexpected route count: got %d want %d", got, want)
	}

	expected := []struct {
		method string
		path   string
		name   string
	}{
		{method: "GET", path: "/s3/buckets/:bucket/object-lock", name: "s3.buckets.object-lock.show"},
		{method: "PUT", path: "/s3/buckets/:bucket/object-lock", name: "s3.buckets.object-lock.update"},
		{method: "GET", path: "/s3/buckets/:bucket/objects/:object/retention", name: "s3.objects.retention.show"},
		{method: "PUT", path: "/s3/buckets/:bucket/objects/:object/retention", name: "s3.objects.retention.update"},
		{method: "GET", path: "/s3/buckets/:bucket/objects/:object/legal-hold", name: "s3.objects.legal-hold.show"},
		{method: "PUT", path: "/s3/buckets/:bucket/objects/:object/legal-hold", name: "s3.objects.legal-hold.update"},
	}

	if got, want := routes[30].Name, "s3.objects.versions"; got != want {
		t.Fatalf("unexpected versioning route at 30: got %q want %q", got, want)
	}
	if got, want := routes[31].Name, expected[0].name; got != want {
		t.Fatalf("unexpected object lock route at 31: got %q want %q", got, want)
	}
	if got, want := routes[32].Name, expected[1].name; got != want {
		t.Fatalf("unexpected object lock route at 32: got %q want %q", got, want)
	}
	if got, want := routes[33].Name, expected[2].name; got != want {
		t.Fatalf("unexpected object lock route at 33: got %q want %q", got, want)
	}
	if got, want := routes[34].Name, expected[3].name; got != want {
		t.Fatalf("unexpected object lock route at 34: got %q want %q", got, want)
	}
	if got, want := routes[35].Name, expected[4].name; got != want {
		t.Fatalf("unexpected object lock route at 35: got %q want %q", got, want)
	}
	if got, want := routes[36].Name, expected[5].name; got != want {
		t.Fatalf("unexpected object lock route at 36: got %q want %q", got, want)
	}
}
