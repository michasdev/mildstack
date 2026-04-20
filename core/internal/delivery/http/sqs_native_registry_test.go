package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
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

	supportedActions := map[string]struct{}{
		"ChangeMessageVisibility":      {},
		"ChangeMessageVisibilityBatch": {},
		"CreateQueue":                  {},
		"DeleteMessage":                {},
		"DeleteMessageBatch":           {},
		"DeleteQueue":                  {},
		"GetQueueAttributes":           {},
		"GetQueueUrl":                  {},
		"ListQueues":                   {},
		"PurgeQueue":                   {},
		"ReceiveMessage":               {},
		"SendMessage":                  {},
		"SendMessageBatch":             {},
		"SetQueueAttributes":           {},
	}
	deferredActions := map[string]struct{}{
		"AddPermission":              {},
		"CancelMessageMoveTask":      {},
		"ListDeadLetterSourceQueues": {},
		"ListMessageMoveTasks":       {},
		"ListQueueTags":              {},
		"RemovePermission":           {},
		"StartMessageMoveTask":       {},
		"TagQueue":                   {},
		"UntagQueue":                 {},
	}

	for i, spec := range entries {
		if spec.Action != catalog[i].Action {
			t.Fatalf("unexpected action at %d: got %q want %q", i, spec.Action, catalog[i].Action)
		}

		_, supported := supportedActions[spec.Action]
		_, deferred := deferredActions[spec.Action]
		if supported && deferred {
			t.Fatalf("action %s cannot be both supported and deferred", spec.Action)
		}
		if supported {
			if !spec.Supported {
				t.Fatalf("expected action %s to be transport-supported", spec.Action)
			}
			if spec.DomainDeferred {
				t.Fatalf("expected action %s to be routed to the service, not deferred", spec.Action)
			}
		}
		if deferred {
			if spec.Supported {
				t.Fatalf("expected action %s to remain deferred", spec.Action)
			}
			if !spec.DomainDeferred {
				t.Fatalf("expected action %s to be deferred", spec.Action)
			}
		}
		if isQueueLifecycleAction(spec.Action) && spec.MessageSurface {
			t.Fatalf("did not expect lifecycle action %s to be marked as message surface", spec.Action)
		}
		if !isQueueLifecycleAction(spec.Action) && !spec.MessageSurface && !spec.DomainDeferred {
			t.Fatalf("expected action %s to remain deferred", spec.Action)
		}
		if spec.Scope != catalog[i].Scope {
			t.Fatalf("unexpected scope for %s: got %q want %q", spec.Action, spec.Scope, catalog[i].Scope)
		}
	}
}

func TestSQSNativeRegistrySeparatesSupportedAndDeferredActions(t *testing.T) {
	t.Helper()

	registry := NewSQSRegistry()

	if got, want := registry.SupportedActions(), []string{
		"ChangeMessageVisibility",
		"ChangeMessageVisibilityBatch",
		"CreateQueue",
		"DeleteMessage",
		"DeleteMessageBatch",
		"DeleteQueue",
		"GetQueueAttributes",
		"GetQueueUrl",
		"ListQueues",
		"PurgeQueue",
		"ReceiveMessage",
		"SendMessage",
		"SendMessageBatch",
		"SetQueueAttributes",
	}; !equalStringSlicesSQS(got, want) {
		t.Fatalf("unexpected supported actions: got %v want %v", got, want)
	}

	if got, want := registry.UnsupportedActions(), []string{
		"AddPermission",
		"CancelMessageMoveTask",
		"ListDeadLetterSourceQueues",
		"ListMessageMoveTasks",
		"ListQueueTags",
		"RemovePermission",
		"StartMessageMoveTask",
		"TagQueue",
		"UntagQueue",
	}; !equalStringSlicesSQS(got, want) {
		t.Fatalf("unexpected unsupported actions: got %v want %v", got, want)
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

func TestSQSNativeRegistryRoutesMessageActionsEvenWhenPolicyOmitsThem(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	service := &registryPolicyTrimmedService{}
	router := gin.New()
	RegisterSQSNativeRoutes(router, service)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/123456789012/orders/", strings.NewReader("Action=SendMessage&Version=2012-11-05&MessageBody=hello"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(recorder, request)

	if got, want := recorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected send message status: got %d want %d", got, want)
	}
	if got, want := service.sendMessageQueueName, "orders"; got != want {
		t.Fatalf("unexpected queue name captured by service: got %q want %q", got, want)
	}
	if !strings.Contains(recorder.Body.String(), "SendMessageResponse") {
		t.Fatalf("expected send message response, got %q", recorder.Body.String())
	}
}

func equalStringSlicesSQS(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

type registryPolicyTrimmedService struct {
	stubSQSNativeService
}

func (s *registryPolicyTrimmedService) Policy() orchestrator.EmulationPolicy {
	return orchestrator.NewEmulationPolicy(orchestrator.FidelityExemplar, []string{"ListQueues"}, nil, "sqs")
}
