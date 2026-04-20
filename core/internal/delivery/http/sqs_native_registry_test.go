package http

import (
	"testing"

	"github.com/michasdev/mildstack/core/internal/resources/sqs/contracts"
)

func TestSQSNativeRegistryDerivesSpecsFromCatalog(t *testing.T) {
	t.Helper()

	registry := NewSQSRegistry()
	entries := registry.Entries()
	catalog := contracts.Catalog()

	if got, want := len(entries), len(catalog); got != want {
		t.Fatalf("unexpected registry entry count: got %d want %d", got, want)
	}

	for i, spec := range entries {
		if spec.Action != catalog[i].Action {
			t.Fatalf("unexpected action at %d: got %q want %q", i, spec.Action, catalog[i].Action)
		}
		if !spec.Supported {
			t.Fatalf("expected action %s to be transport-supported", spec.Action)
		}
		if isQueueLifecycleAction(spec.Action) {
			if spec.DomainDeferred {
				t.Fatalf("expected action %s to be routed to the service, not deferred", spec.Action)
			}
			if spec.MessageSurface {
				t.Fatalf("did not expect lifecycle action %s to be marked as message surface", spec.Action)
			}
			continue
		}
		if spec.MessageSurface {
			if spec.DomainDeferred {
				t.Fatalf("expected message action %s to be routed to the service seam, not deferred", spec.Action)
			}
			continue
		}
		if !spec.DomainDeferred {
			t.Fatalf("expected action %s to remain domain deferred", spec.Action)
		}
		if spec.Scope != catalog[i].Scope {
			t.Fatalf("unexpected scope for %s: got %q want %q", spec.Action, spec.Scope, catalog[i].Scope)
		}
	}
}

func TestSQSNativeRegistryRecognizesQueueLifecycleActions(t *testing.T) {
	t.Helper()

	for _, action := range []string{"CreateQueue", "DeleteQueue", "GetQueueAttributes", "GetQueueUrl", "ListQueues", "PurgeQueue", "SetQueueAttributes"} {
		if !isQueueLifecycleAction(action) {
			t.Fatalf("expected %s to be recognized as a queue lifecycle action", action)
		}
	}
	if isQueueLifecycleAction("SendMessage") {
		t.Fatal("did not expect SendMessage to be treated as a queue lifecycle action")
	}
}

func TestSQSNativeRegistryMarksPhase39MessageSurfaceActions(t *testing.T) {
	t.Helper()

	registry := NewSQSRegistry()
	for _, action := range []string{"ChangeMessageVisibility", "ChangeMessageVisibilityBatch", "DeleteMessage", "DeleteMessageBatch", "ReceiveMessage", "SendMessage", "SendMessageBatch"} {
		spec, ok := registry.Lookup(action)
		if !ok {
			t.Fatalf("expected registry action %s", action)
		}
		if !spec.MessageSurface {
			t.Fatalf("expected %s to be marked as message surface", action)
		}
		if spec.DomainDeferred {
			t.Fatalf("expected %s to be routed away from domain deferral", action)
		}
	}
}

func TestSQSNativeRegistryScopeMismatchesMapToExplicitErrors(t *testing.T) {
	t.Helper()

	registry := NewSQSRegistry()

	rootCtx := SQSRequestContext{Action: "ListQueues", Version: sqsQueryVersion, Kind: SQSRequestKindRoot}
	if _, err := registry.Resolve(rootCtx); err != nil {
		t.Fatalf("resolve matching root action: %v", err)
	}

	queueCtx := SQSRequestContext{Action: "SendMessage", Version: sqsQueryVersion, Kind: SQSRequestKindQueue}
	if _, err := registry.Resolve(queueCtx); err != nil {
		t.Fatalf("resolve matching queue action: %v", err)
	}

	queueMismatch := SQSRequestContext{Action: "CreateQueue", Version: sqsQueryVersion, Kind: SQSRequestKindQueue}
	if _, err := registry.Resolve(queueMismatch); err != ErrSQSQueuePathMismatch {
		t.Fatalf("unexpected queue mismatch error: got %v want %v", err, ErrSQSQueuePathMismatch)
	}

	rootMismatch := SQSRequestContext{Action: "SendMessage", Version: sqsQueryVersion, Kind: SQSRequestKindRoot}
	if _, err := registry.Resolve(rootMismatch); err != ErrSQSQueuePathMismatch {
		t.Fatalf("unexpected root mismatch error: got %v want %v", err, ErrSQSQueuePathMismatch)
	}
}

func TestSQSNativeRegistryRejectsUnknownAction(t *testing.T) {
	t.Helper()

	registry := NewSQSRegistry()
	if _, err := registry.Resolve(SQSRequestContext{Action: "NotARealAction", Version: sqsQueryVersion, Kind: SQSRequestKindRoot}); err != ErrSQSInvalidAction {
		t.Fatalf("unexpected unknown action error: got %v want %v", err, ErrSQSInvalidAction)
	}
}
