package architecture

import (
	"os"
	"strings"
	"testing"
)

func TestArchitectureDocsMatchCurrentTreeAndTemplate(t *testing.T) {
	t.Helper()

	layout := readDoc(t, "layout.md")
	onboarding := readDoc(t, "service_onboarding.md")

	requiredLayoutPaths := []string{
		"core/internal/domain/",
		"core/internal/application/orchestrator/",
		"core/internal/application/runtime/",
		"core/internal/composition/",
		"core/internal/infrastructure/",
		"core/internal/delivery/http/",
		"core/internal/delivery/cli/",
		"core/internal/delivery/cli/ui/",
		"core/internal/resources/s3/",
		"core/internal/resources/dynamodb/",
	}
	for _, want := range requiredLayoutPaths {
		if !strings.Contains(layout, want) {
			t.Fatalf("layout.md is missing required package path %q", want)
		}
	}

	requiredOnboardingPaths := []string{
		"core/internal/domain/",
		"core/internal/application/orchestrator/",
		"core/internal/application/runtime/",
		"core/internal/composition/",
		"core/internal/infrastructure/",
		"core/internal/delivery/http/",
		"core/internal/delivery/cli/",
		"core/internal/delivery/cli/ui/",
		"core/internal/resources/s3/application/service_test.go",
		"core/internal/resources/s3/domain/state_test.go",
		"core/internal/resources/s3/infrastructure/routes_test.go",
		"core/internal/resources/s3/infrastructure/handlers_test.go",
		"core/internal/resources/dynamodb/application/service_test.go",
		"core/internal/resources/dynamodb/domain/state_test.go",
		"core/internal/resources/dynamodb/infrastructure/routes_test.go",
		"core/internal/resources/dynamodb/infrastructure/handlers_test.go",
		"services/<name>",
		"TestServiceMetadataRoutesAndState",
		"TestServiceRealOperationsMutateState",
		"TestServiceRejectsInvalidAndMissingRequests",
		"TestServiceStartAndStopAreNoOps",
	}
	for _, want := range requiredOnboardingPaths {
		if !strings.Contains(onboarding, want) {
			t.Fatalf("service_onboarding.md is missing required guidance %q", want)
		}
	}

	for _, path := range []string{
		"../domain",
		"../application",
		"../application/orchestrator",
		"../application/runtime",
		"../composition",
		"../delivery/cli",
		"../delivery/cli/ui",
		"../delivery/http",
		"../infrastructure",
		"../resources/s3/domain",
		"../resources/s3/application",
		"../resources/s3/infrastructure",
		"../resources/dynamodb/domain",
		"../resources/dynamodb/application",
		"../resources/dynamodb/infrastructure",
	} {
		mustExist(t, path)
	}
}

func readDoc(t *testing.T, name string) string {
	t.Helper()

	content, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}

	return string(content)
}
