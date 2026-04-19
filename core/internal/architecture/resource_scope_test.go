package architecture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResourceScopeGuardrailRequiresSharedInstancePathHelper(t *testing.T) {
	t.Helper()

	matches, err := filepath.Glob("../resources/*/application/storage_paths.go")
	if err != nil {
		t.Fatalf("glob storage paths: %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("expected at least one storage path resolver to validate")
	}

	for _, path := range matches {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if !strings.Contains(string(content), "instancepath.Resolve(") {
			t.Fatalf("%s must use the shared instancepath helper", path)
		}
	}
}

func TestResourceScopeGuardrailRequiresSharedAWSContextHelper(t *testing.T) {
	t.Helper()

	type fileCheck struct {
		path           string
		mustContain    []string
		mustNotContain []string
	}

	checks := []fileCheck{
		{
			path:           "../resources/s3/domain/state.go",
			mustContain:    []string{"awscontext.Default()"},
			mustNotContain: []string{"us-east-1"},
		},
		{
			path:           "../resources/s3/application/repository_fs.go",
			mustContain:    []string{"awscontext.Default().Region"},
			mustNotContain: []string{"defaultRegion"},
		},
		{
			path:           "../resources/s3/application/service_buckets.go",
			mustContain:    []string{"awscontext.Default().Region"},
			mustNotContain: []string{"defaultRegion"},
		},
		{
			path:           "../resources/s3/application/service_bucket_access.go",
			mustContain:    []string{"awscontext.Default()"},
			mustNotContain: []string{"owner-id", "123456789012"},
		},
		{
			path:           "../resources/s3/application/service_subresources.go",
			mustContain:    []string{"awscontext.Default()"},
			mustNotContain: []string{"owner-id", "123456789012"},
		},
		{
			path:           "../delivery/http/s3_native.go",
			mustContain:    []string{"awscontext.Default()"},
			mustNotContain: []string{"us-east-1"},
		},
		{
			path:           "../delivery/http/dynamodb_native.go",
			mustContain:    []string{"awscontext.Default()", "TableArn"},
			mustNotContain: []string{"123456789012", "us-east-1"},
		},
	}

	for _, check := range checks {
		content, err := os.ReadFile(check.path)
		if err != nil {
			t.Fatalf("read %s: %v", check.path, err)
		}
		text := string(content)
		for _, want := range check.mustContain {
			if !strings.Contains(text, want) {
				t.Fatalf("%s must contain %q", check.path, want)
			}
		}
		for _, forbidden := range check.mustNotContain {
			if strings.Contains(text, forbidden) {
				t.Fatalf("%s must not contain %q", check.path, forbidden)
			}
		}
	}
}
