package infrastructure

import "testing"

func TestObjectLockRoutesAreRegisteredOnce(t *testing.T) {
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
		{method: "GET", path: "/:bucket?object-lock", name: "s3.buckets.object-lock.show"},
		{method: "PUT", path: "/:bucket?object-lock", name: "s3.buckets.object-lock.update"},
		{method: "GET", path: "/:bucket/:object?retention", name: "s3.objects.retention.show"},
		{method: "PUT", path: "/:bucket/:object?retention", name: "s3.objects.retention.update"},
		{method: "GET", path: "/:bucket/:object?legal-hold", name: "s3.objects.legal-hold.show"},
		{method: "PUT", path: "/:bucket/:object?legal-hold", name: "s3.objects.legal-hold.update"},
	}

	if got, want := routes[37].Name, "s3.objects.versions"; got != want {
		t.Fatalf("unexpected versioning route at 37: got %q want %q", got, want)
	}
	if got, want := routes[38].Name, expected[0].name; got != want {
		t.Fatalf("unexpected object lock route at 38: got %q want %q", got, want)
	}
	if got, want := routes[39].Name, expected[1].name; got != want {
		t.Fatalf("unexpected object lock route at 39: got %q want %q", got, want)
	}
	if got, want := routes[40].Name, expected[2].name; got != want {
		t.Fatalf("unexpected object lock route at 40: got %q want %q", got, want)
	}
	if got, want := routes[41].Name, expected[3].name; got != want {
		t.Fatalf("unexpected object lock route at 41: got %q want %q", got, want)
	}
	if got, want := routes[42].Name, expected[4].name; got != want {
		t.Fatalf("unexpected object lock route at 42: got %q want %q", got, want)
	}
	if got, want := routes[43].Name, expected[5].name; got != want {
		t.Fatalf("unexpected object lock route at 43: got %q want %q", got, want)
	}
}
