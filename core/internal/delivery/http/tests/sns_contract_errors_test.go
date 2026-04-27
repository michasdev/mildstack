package tests

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSNSContractErrorEnvelopesMatchGoldenFiles(t *testing.T) {
	t.Helper()

	router, _ := newSNSContractHarness(t)

	cases := []struct {
		name         string
		method       string
		pathAndQuery string
		contentType  string
		status       int
		goldenFile   string
	}{
		{
			name:         "missing action",
			method:       http.MethodGet,
			pathAndQuery: "/?Version=2010-03-31",
			status:       http.StatusBadRequest,
			goldenFile:   "missing_action.xml",
		},
		{
			name:         "missing version",
			method:       http.MethodGet,
			pathAndQuery: "/?Action=CreateTopic",
			status:       http.StatusBadRequest,
			goldenFile:   "missing_version.xml",
		},
		{
			name:         "invalid version",
			method:       http.MethodGet,
			pathAndQuery: "/?Action=CreateTopic&Version=2012-11-05",
			status:       http.StatusBadRequest,
			goldenFile:   "invalid_version.xml",
		},
		{
			name:         "invalid action",
			method:       http.MethodGet,
			pathAndQuery: "/?Action=UnknownAction&Version=2010-03-31",
			status:       http.StatusBadRequest,
			goldenFile:   "invalid_action.xml",
		},
		{
			name:         "invalid parameter",
			method:       http.MethodGet,
			pathAndQuery: "/?Action=CreateTopic&Version=2010-03-31&Name=invalid.name",
			status:       http.StatusBadRequest,
			goldenFile:   "invalid_parameter.xml",
		},
		{
			name:         "not found",
			method:       http.MethodGet,
			pathAndQuery: "/?Action=GetTopicAttributes&Version=2010-03-31&TopicArn=arn:aws:sns:us-east-1:00000000000:missing",
			status:       http.StatusNotFound,
			goldenFile:   "not_found.xml",
		},
		{
			name:         "malformed query",
			method:       http.MethodPost,
			pathAndQuery: "/",
			contentType:  "application/json",
			status:       http.StatusBadRequest,
			goldenFile:   "invalid_query_parameter.xml",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Helper()

			request := httptest.NewRequest(tc.method, tc.pathAndQuery, nil)
			if strings.TrimSpace(tc.contentType) != "" {
				request.Header.Set("Content-Type", tc.contentType)
			}

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)

			if got, want := recorder.Code, tc.status; got != want {
				t.Fatalf("unexpected status: got %d want %d", got, want)
			}

			gotBody := strings.TrimSpace(recorder.Body.String())
			wantBody := strings.TrimSpace(snsGoldenErrorBody(t, tc.goldenFile))
			if gotBody != wantBody {
				t.Fatalf("unexpected xml body for %s\n--- got ---\n%s\n--- want ---\n%s", tc.name, gotBody, wantBody)
			}
		})
	}
}

func snsGoldenErrorBody(t *testing.T, fileName string) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not resolve runtime caller for golden file lookup")
	}
	goldenPath := filepath.Join(filepath.Dir(currentFile), "..", "testdata", "sns", "errors", fileName)
	content, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden file %s: %v", goldenPath, err)
	}
	return string(content)
}
