package sns_test

import (
	"testing"

	"github.com/michasdev/mildstack/core/internal/resources/sns/domain"
	"github.com/michasdev/mildstack/core/internal/resources/sns/infrastructure"
)

func TestSNSIsolationByInstanceAccountAndRegion(t *testing.T) {
	t.Helper()

	baseDir := t.TempDir()
	firstInstancePath, err := infrastructure.ResolveStatePath(baseDir, "instance-a")
	if err != nil {
		t.Fatalf("resolve first instance path: %v", err)
	}
	secondInstancePath, err := infrastructure.ResolveStatePath(baseDir, "instance-b")
	if err != nil {
		t.Fatalf("resolve second instance path: %v", err)
	}
	if firstInstancePath == secondInstancePath {
		t.Fatalf("expected distinct instance paths, got %q", firstInstancePath)
	}

	storeA, err := infrastructure.NewSQLiteStore(firstInstancePath)
	if err != nil {
		t.Fatalf("new store A: %v", err)
	}
	t.Cleanup(func() { _ = storeA.Close() })

	storeB, err := infrastructure.NewSQLiteStore(secondInstancePath)
	if err != nil {
		t.Fatalf("new store B: %v", err)
	}
	t.Cleanup(func() { _ = storeB.Close() })

	tenantA := domain.NewTenant("111122223333", "us-east-1", "aws")
	tenantB := domain.NewTenant("111122223333", "eu-west-1", "aws")
	tenantC := domain.NewTenant("999988887777", "us-east-1", "aws")

	if err := storeA.UpsertTopic(tenantA.Key(), tenantA.TopicARN("orders"), "orders"); err != nil {
		t.Fatalf("upsert tenant A topic: %v", err)
	}
	if err := storeA.UpsertTopic(tenantB.Key(), tenantB.TopicARN("orders"), "orders"); err != nil {
		t.Fatalf("upsert tenant B topic: %v", err)
	}
	if err := storeB.UpsertTopic(tenantC.Key(), tenantC.TopicARN("orders"), "orders"); err != nil {
		t.Fatalf("upsert tenant C topic in instance B: %v", err)
	}

	arnsTenantA, err := storeA.ListTopicARNsByTenant(tenantA.Key())
	if err != nil {
		t.Fatalf("list tenant A topics: %v", err)
	}
	if got, want := len(arnsTenantA), 1; got != want {
		t.Fatalf("unexpected tenant A topic count: got %d want %d", got, want)
	}
	if got, want := arnsTenantA[0], tenantA.TopicARN("orders"); got != want {
		t.Fatalf("unexpected tenant A topic arn: got %q want %q", got, want)
	}

	arnsTenantB, err := storeA.ListTopicARNsByTenant(tenantB.Key())
	if err != nil {
		t.Fatalf("list tenant B topics: %v", err)
	}
	if got, want := len(arnsTenantB), 1; got != want {
		t.Fatalf("unexpected tenant B topic count: got %d want %d", got, want)
	}
	if got, want := arnsTenantB[0], tenantB.TopicARN("orders"); got != want {
		t.Fatalf("unexpected tenant B topic arn: got %q want %q", got, want)
	}

	arnsTenantAInstanceB, err := storeB.ListTopicARNsByTenant(tenantA.Key())
	if err != nil {
		t.Fatalf("list tenant A in instance B: %v", err)
	}
	if got, want := len(arnsTenantAInstanceB), 0; got != want {
		t.Fatalf("expected no tenant A topics in instance B, got %d", got)
	}

	arnsTenantCInstanceB, err := storeB.ListTopicARNsByTenant(tenantC.Key())
	if err != nil {
		t.Fatalf("list tenant C in instance B: %v", err)
	}
	if got, want := len(arnsTenantCInstanceB), 1; got != want {
		t.Fatalf("unexpected tenant C topic count in instance B: got %d want %d", got, want)
	}
}
