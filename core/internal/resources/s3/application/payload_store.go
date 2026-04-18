package application

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type PayloadStore interface {
	StorePayload(hint string, body io.Reader) (ref string, size int64, etag string, err error)
	OpenPayload(ref string) (io.ReadCloser, error)
	DeletePayload(ref string) error
}

type memoryPayloadStore struct {
	mu      sync.RWMutex
	payload map[string][]byte
}

func newMemoryPayloadStore() *memoryPayloadStore {
	return &memoryPayloadStore{
		payload: make(map[string][]byte),
	}
}

func (s *memoryPayloadStore) StorePayload(hint string, body io.Reader) (string, int64, string, error) {
	if body == nil {
		body = bytes.NewReader(nil)
	}

	buf := &bytes.Buffer{}
	hash := md5.New()
	size, err := io.Copy(io.MultiWriter(buf, hash), body)
	if err != nil {
		return "", 0, "", err
	}

	ref := payloadRefFor(hint, hash.Sum(nil))
	s.mu.Lock()
	s.payload[ref] = append([]byte(nil), buf.Bytes()...)
	s.mu.Unlock()

	return ref, size, `"` + hex.EncodeToString(hash.Sum(nil)) + `"`, nil
}

func (s *memoryPayloadStore) OpenPayload(ref string) (io.ReadCloser, error) {
	src, err := s.readPayload(ref)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(src)), nil
}

func (s *memoryPayloadStore) DeletePayload(ref string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.payload, ref)
	return nil
}

func (s *memoryPayloadStore) readPayload(ref string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	body, ok := s.payload[ref]
	if !ok {
		return nil, fmt.Errorf("s3: payload %q not found", ref)
	}
	return append([]byte(nil), body...), nil
}

type filePayloadStore struct {
	payloadPath string
}

func newFilePayloadStore(storagePath string) *filePayloadStore {
	return &filePayloadStore{
		payloadPath: filepath.Join(storagePath, "payloads"),
	}
}

func (s *filePayloadStore) StorePayload(hint string, body io.Reader) (string, int64, string, error) {
	if body == nil {
		body = bytes.NewReader(nil)
	}
	if err := os.MkdirAll(s.payloadPath, 0o755); err != nil {
		return "", 0, "", err
	}

	tempFile, err := os.CreateTemp(s.payloadPath, "payload-*.tmp")
	if err != nil {
		return "", 0, "", err
	}
	tempPath := tempFile.Name()

	hash := md5.New()
	size, copyErr := io.Copy(io.MultiWriter(tempFile, hash), body)
	if copyErr != nil {
		tempFile.Close()
		_ = os.Remove(tempPath)
		return "", 0, "", copyErr
	}
	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tempPath)
		return "", 0, "", err
	}
	ref := payloadRefFor(hint, hash.Sum(nil))
	finalPath := s.payloadFilePath(ref)
	if err := os.Rename(tempPath, finalPath); err != nil {
		_ = os.Remove(tempPath)
		return "", 0, "", err
	}

	return ref, size, `"` + hex.EncodeToString(hash.Sum(nil)) + `"`, nil
}

func (s *filePayloadStore) OpenPayload(ref string) (io.ReadCloser, error) {
	return os.Open(s.payloadFilePath(ref))
}

func (s *filePayloadStore) DeletePayload(ref string) error {
	if ref == "" {
		return nil
	}
	if err := os.Remove(s.payloadFilePath(ref)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s *filePayloadStore) payloadFilePath(ref string) string {
	return filepath.Join(s.payloadPath, ref)
}

func payloadRefFor(hint string, bodyHash []byte) string {
	hintHash := md5.Sum([]byte(strings.TrimSpace(hint)))
	return fmt.Sprintf("payload-%s-%s", hex.EncodeToString(hintHash[:]), hex.EncodeToString(bodyHash))
}
