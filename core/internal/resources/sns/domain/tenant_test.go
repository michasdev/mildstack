package domain

import "testing"

func TestTenantKeyAndTopicARN(t *testing.T) {
	t.Helper()

	tenant := NewTenant("111122223333", "eu-west-1", "aws")
	if got, want := tenant.Key(), "111122223333:eu-west-1"; got != want {
		t.Fatalf("unexpected tenant key: got %q want %q", got, want)
	}
	if got, want := tenant.TopicARN("orders"), "arn:aws:sns:eu-west-1:111122223333:orders"; got != want {
		t.Fatalf("unexpected topic arn: got %q want %q", got, want)
	}
}
