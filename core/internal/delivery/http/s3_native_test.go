package http

import (
	"bytes"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/michasdev/mildstack/core/internal/resources/awscontext"
	s3domain "github.com/michasdev/mildstack/core/internal/resources/s3/domain"
)

type countingBody struct {
	payload []byte
	reads   int
	closed  bool
	offset  int
}

func (b *countingBody) Read(p []byte) (int, error) {
	b.reads++
	if b.offset >= len(b.payload) {
		return 0, io.EOF
	}
	n := copy(p, b.payload[b.offset:])
	b.offset += n
	return n, nil
}

func (b *countingBody) Close() error {
	b.closed = true
	return nil
}

type spyS3NativeService struct {
	putObjectBody io.Reader
	putObjectETag string
}

func (s *spyS3NativeService) ListBuckets() []s3domain.Bucket { return nil }
func (s *spyS3NativeService) ListObjects(bucket string) ([]s3domain.Object, error) {
	return nil, nil
}
func (s *spyS3NativeService) CreateBucket(name, region string) (s3domain.Bucket, error) {
	return s3domain.Bucket{}, nil
}
func (s *spyS3NativeService) HeadBucket(name string) (s3domain.Bucket, error) {
	return s3domain.Bucket{}, nil
}
func (s *spyS3NativeService) DeleteBucket(name string) error { return nil }
func (s *spyS3NativeService) GetObject(bucket, key string) (s3domain.Object, error) {
	return s3domain.Object{}, nil
}
func (s *spyS3NativeService) HeadObject(bucket, key string) (s3domain.Object, error) {
	return s3domain.Object{}, nil
}
func (s *spyS3NativeService) PutObject(bucket, key string, body io.Reader, contentType string) (s3domain.Object, error) {
	s.putObjectBody = body
	return s3domain.Object{ETag: s.putObjectETag}, nil
}
func (s *spyS3NativeService) DeleteObject(bucket, key string) error { return nil }

func TestS3NativePutObjectStreamsBody(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	spy := &spyS3NativeService{putObjectETag: `"etag"`}
	engine := gin.New()
	RegisterS3NativeRoutes(engine, spy)

	body := &countingBody{payload: []byte("streamed-body")}
	request := httptest.NewRequest(http.MethodPut, "/streaming-bucket/archive.txt", nil)
	request.Body = body
	request.Header.Set("Content-Type", "text/plain")
	recorder := httptest.NewRecorder()

	engine.ServeHTTP(recorder, request)

	if got, want := recorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected status: got %d want %d", got, want)
	}
	if spy.putObjectBody != body {
		t.Fatal("expected handler to pass the original request body through unchanged")
	}
	if body.reads != 0 {
		t.Fatalf("expected handler to defer body reads to the service, got %d reads", body.reads)
	}
}

func TestS3NativePutObjectDoesNotBufferWholeRequest(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	spy := &spyS3NativeService{putObjectETag: `"etag"`}
	engine := gin.New()
	RegisterS3NativeRoutes(engine, spy)

	body := &countingBody{payload: bytes.Repeat([]byte("x"), 1024)}
	request := httptest.NewRequest(http.MethodPut, "/streaming-bucket/large.bin", nil)
	request.Body = body
	recorder := httptest.NewRecorder()

	engine.ServeHTTP(recorder, request)

	if got, want := recorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected status: got %d want %d", got, want)
	}
	if body.reads != 0 {
		t.Fatalf("expected zero pre-buffering reads, got %d", body.reads)
	}
}

func TestS3NativeListBucketsUsesSharedAWSAccountID(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	spy := &spyS3NativeService{}
	engine := gin.New()
	RegisterS3NativeRoutes(engine, spy)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	engine.ServeHTTP(recorder, request)

	if got, want := recorder.Code, http.StatusOK; got != want {
		t.Fatalf("unexpected status: got %d want %d", got, want)
	}
	var payload struct {
		XMLName xml.Name `xml:"ListAllMyBucketsResult"`
		Owner   struct {
			ID string `xml:"ID"`
		} `xml:"Owner"`
	}
	if err := xml.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode list buckets response: %v", err)
	}
	if got, want := payload.Owner.ID, awscontext.Default().AccountID; got != want {
		t.Fatalf("unexpected owner id: got %q want %q", got, want)
	}
}
