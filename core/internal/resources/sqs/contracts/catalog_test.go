package contracts

import "testing"

func TestCatalogIncludesRequestedActions(t *testing.T) {
	t.Helper()

	specs := Catalog()
	if got, want := len(specs), 23; got != want {
		t.Fatalf("unexpected action count: got %d want %d", got, want)
	}

	expected := []string{
		"AddPermission",
		"CancelMessageMoveTask",
		"ChangeMessageVisibility",
		"ChangeMessageVisibilityBatch",
		"CreateQueue",
		"DeleteMessage",
		"DeleteMessageBatch",
		"DeleteQueue",
		"GetQueueAttributes",
		"GetQueueUrl",
		"ListDeadLetterSourceQueues",
		"ListMessageMoveTasks",
		"ListQueues",
		"ListQueueTags",
		"PurgeQueue",
		"ReceiveMessage",
		"RemovePermission",
		"SendMessage",
		"SendMessageBatch",
		"SetQueueAttributes",
		"StartMessageMoveTask",
		"TagQueue",
		"UntagQueue",
	}
	for i, spec := range specs {
		if spec.Action != expected[i] {
			t.Fatalf("unexpected action at %d: got %q want %q", i, spec.Action, expected[i])
		}
		if spec.Version != "2012-11-05" {
			t.Fatalf("unexpected version for %s: got %q want %q", spec.Action, spec.Version, "2012-11-05")
		}
	}
}

func TestCatalogHasTransportScopeMetadata(t *testing.T) {
	t.Helper()

	specs := Catalog()
	byAction := make(map[string]ActionSpec, len(specs))
	for _, spec := range specs {
		byAction[spec.Action] = spec
	}

	rootActions := []string{"CreateQueue", "GetQueueUrl", "ListQueues"}
	for _, action := range rootActions {
		spec, ok := byAction[action]
		if !ok {
			t.Fatalf("expected action %q in catalog", action)
		}
		if spec.Scope != ScopeRoot {
			t.Fatalf("unexpected scope for %s: got %q want %q", action, spec.Scope, ScopeRoot)
		}
	}

	queueActions := []string{"SendMessage", "DeleteMessage", "SetQueueAttributes", "TagQueue"}
	for _, action := range queueActions {
		spec, ok := byAction[action]
		if !ok {
			t.Fatalf("expected action %q in catalog", action)
		}
		if spec.Scope != ScopeQueue {
			t.Fatalf("unexpected scope for %s: got %q want %q", action, spec.Scope, ScopeQueue)
		}
		if !spec.UsesQueueContext {
			t.Fatalf("expected queue context for %s", action)
		}
	}

	if !byAction["CreateQueue"].ReturnsQueueURL {
		t.Fatal("expected CreateQueue to surface QueueUrl")
	}
	if !byAction["GetQueueUrl"].ReturnsQueueURL {
		t.Fatal("expected GetQueueUrl to surface QueueUrl")
	}
	if !byAction["ListQueues"].ReturnsQueueURL {
		t.Fatal("expected ListQueues to surface QueueUrl")
	}
	if !byAction["ListDeadLetterSourceQueues"].ReturnsQueueURL {
		t.Fatal("expected ListDeadLetterSourceQueues to surface QueueUrl")
	}
}

func TestCatalogMarksPhase39MessageSurfaceActions(t *testing.T) {
	t.Helper()

	byAction := make(map[string]ActionSpec)
	for _, spec := range Catalog() {
		byAction[spec.Action] = spec
	}

	messageActions := []string{
		"ChangeMessageVisibility",
		"ChangeMessageVisibilityBatch",
		"DeleteMessage",
		"DeleteMessageBatch",
		"ReceiveMessage",
		"SendMessage",
		"SendMessageBatch",
	}
	for _, action := range messageActions {
		spec, ok := byAction[action]
		if !ok {
			t.Fatalf("expected action %q in catalog", action)
		}
		if !spec.MessageSurface {
			t.Fatalf("expected action %s to be marked as message surface", action)
		}
	}

	for _, action := range []string{"AddPermission", "CreateQueue", "ListQueues", "TagQueue"} {
		spec, ok := byAction[action]
		if !ok {
			t.Fatalf("expected action %q in catalog", action)
		}
		if spec.MessageSurface {
			t.Fatalf("did not expect action %s to be marked as message surface", action)
		}
	}
}

func TestCatalogReturnsCopies(t *testing.T) {
	t.Helper()

	first := Catalog()
	first[0].Action = "Changed"

	second := Catalog()
	if second[0].Action == "Changed" {
		t.Fatal("expected catalog to return a copy")
	}
}
