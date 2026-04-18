package application

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestServicePutObjectStreamsPayloadContract(t *testing.T) {
	t.Helper()

	service := New()
	bucket, err := service.CreateBucket("streaming-contract", "us-east-1")
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}

	body := bytes.NewReader([]byte("streamed payload"))
	object, err := service.PutObject(bucket.Name, "archive.txt", body, "text/plain")
	if err != nil {
		t.Fatalf("put object: %v", err)
	}
	if got, want := object.Body, []byte("streamed payload"); string(got) != string(want) {
		t.Fatalf("unexpected returned body: got %q want %q", string(got), string(want))
	}
	if object.PayloadRef == "" {
		t.Fatal("expected object to receive a payload reference")
	}

	stored, ok := service.state.Object(bucket.Name, "archive.txt")
	if !ok {
		t.Fatal("expected stored object to exist")
	}
	if stored.PayloadRef == "" {
		t.Fatal("expected stored object to point at an external payload")
	}
	if len(stored.Body) != 0 {
		t.Fatalf("expected stored object body to stay out of state, got %d bytes", len(stored.Body))
	}

	fetched, err := service.GetObject(bucket.Name, "archive.txt")
	if err != nil {
		t.Fatalf("get object: %v", err)
	}
	if got, want := string(fetched.Body), "streamed payload"; got != want {
		t.Fatalf("unexpected fetched body: got %q want %q", got, want)
	}
}

func TestServicePutObjectStreamsPayload(t *testing.T) {
	t.Helper()
	TestServicePutObjectStreamsPayloadContract(t)
}

func TestConcurrentPutObjectUploads(t *testing.T) {
	t.Helper()

	baseDir := t.TempDir()
	service, err := NewWithPersistence(StorageConfig{
		BaseDir:    baseDir,
		InstanceID: "streaming-concurrency",
	})
	if err != nil {
		t.Fatalf("new with persistence: %v", err)
	}

	bucket, err := service.CreateBucket("concurrent-uploads", "us-east-1")
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}

	const uploads = 8
	const payloadSize = 256 * 1024

	var wg sync.WaitGroup
	errCh := make(chan error, uploads)
	for i := 0; i < uploads; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			key := fmt.Sprintf("object-%02d.bin", i)
			payload := bytes.Repeat([]byte{byte('a' + i)}, payloadSize)
			if _, err := service.PutObject(bucket.Name, key, bytes.NewReader(payload), "application/octet-stream"); err != nil {
				errCh <- fmt.Errorf("put %s: %w", key, err)
			}
		}(i)
	}
	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatal(err)
		}
	}

	storagePath, err := ResolveStoragePath(StorageConfig{
		BaseDir:    baseDir,
		InstanceID: "streaming-concurrency",
	})
	if err != nil {
		t.Fatalf("resolve storage path: %v", err)
	}
	statePath := filepath.Join(storagePath, stateFileName)
	info, err := os.Stat(statePath)
	if err != nil {
		t.Fatalf("stat state file: %v", err)
	}
	if info.Size() > 64*1024 {
		t.Fatalf("expected compact state file, got %d bytes", info.Size())
	}

	reloaded, err := NewWithPersistence(StorageConfig{
		BaseDir:    baseDir,
		InstanceID: "streaming-concurrency",
	})
	if err != nil {
		t.Fatalf("reload service: %v", err)
	}
	objects, err := reloaded.ListObjects(bucket.Name)
	if err != nil {
		t.Fatalf("list objects after reload: %v", err)
	}
	if got, want := len(objects), uploads; got != want {
		t.Fatalf("unexpected object count after reload: got %d want %d", got, want)
	}
	for i := 0; i < uploads; i++ {
		key := fmt.Sprintf("object-%02d.bin", i)
		object, err := reloaded.GetObject(bucket.Name, key)
		if err != nil {
			t.Fatalf("get %s after reload: %v", key, err)
		}
		expected := strings.Repeat(string(byte('a'+i)), 32)
		if len(object.Body) < 32 {
			t.Fatalf("unexpected short body for %s: %d bytes", key, len(object.Body))
		}
		if got, want := string(object.Body[:32]), expected; got != want {
			t.Fatalf("unexpected body prefix for %s: got %q want %q", key, got, want)
		}
	}
}
