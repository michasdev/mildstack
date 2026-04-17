package application

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/michasdev/mildstack/core/internal/s3/domain"
)

const stateFileName = "state.json"

type FSRepository struct {
	storagePath string
	statePath   string
}

func NewFSRepository(storagePath string) *FSRepository {
	return &FSRepository{
		storagePath: storagePath,
		statePath:   filepath.Join(storagePath, stateFileName),
	}
}

func (r *FSRepository) Load() (domain.State, error) {
	data, err := os.ReadFile(r.statePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return domain.NewState(), nil
		}
		return domain.State{}, err
	}

	var state domain.State
	if err := json.Unmarshal(data, &state); err != nil {
		return domain.State{}, fmt.Errorf("decode %s: %w", stateFileName, err)
	}
	if err := validateState(state); err != nil {
		return domain.State{}, err
	}

	return normalizeState(state), nil
}

func (r *FSRepository) Save(state domain.State) error {
	normalized := normalizeState(state)
	if err := validateState(normalized); err != nil {
		return err
	}

	data, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	if err := os.MkdirAll(r.storagePath, 0o755); err != nil {
		return err
	}

	tempFile, err := os.CreateTemp(r.storagePath, "state-*.tmp")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()

	if _, err := tempFile.Write(data); err != nil {
		tempFile.Close()
		_ = os.Remove(tempPath)
		return err
	}
	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	if err := os.Rename(tempPath, r.statePath); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	return nil
}

func validateState(state domain.State) error {
	if state.Service != "s3" {
		return fmt.Errorf("invalid service %q", state.Service)
	}

	buckets := make(map[string]struct{}, len(state.Buckets))
	for _, bucket := range state.Buckets {
		if bucket.Name == "" {
			return fmt.Errorf("invalid bucket: empty name")
		}
		if bucket.Region == "" {
			return fmt.Errorf("invalid bucket %q: empty region", bucket.Name)
		}
		buckets[bucket.Name] = struct{}{}
	}

	for _, object := range state.Objects {
		if object.Bucket == "" || object.Key == "" {
			return fmt.Errorf("invalid object key %q for bucket %q", object.Key, object.Bucket)
		}
		if _, ok := buckets[object.Bucket]; !ok {
			return fmt.Errorf("invalid object %s/%s: bucket not found", object.Bucket, object.Key)
		}
		if object.ContentType == "" {
			return fmt.Errorf("invalid object %s/%s: empty content type", object.Bucket, object.Key)
		}
	}

	return nil
}

func normalizeState(state domain.State) domain.State {
	normalized := domain.State{
		Service: state.Service,
		Buckets: make([]domain.Bucket, len(state.Buckets)),
		Objects: make([]domain.Object, len(state.Objects)),
	}
	copy(normalized.Buckets, state.Buckets)
	for i := range state.Objects {
		normalized.Objects[i] = state.Objects[i]
		normalized.Objects[i].Body = append([]byte(nil), state.Objects[i].Body...)
		if len(state.Objects[i].Metadata) > 0 {
			normalized.Objects[i].Metadata = make(map[string]string, len(state.Objects[i].Metadata))
			for key, value := range state.Objects[i].Metadata {
				normalized.Objects[i].Metadata[key] = value
			}
		}
		if len(state.Objects[i].PreservedHeaders) > 0 {
			normalized.Objects[i].PreservedHeaders = make(map[string]string, len(state.Objects[i].PreservedHeaders))
			for key, value := range state.Objects[i].PreservedHeaders {
				normalized.Objects[i].PreservedHeaders[key] = value
			}
		}
	}

	for i := range normalized.Buckets {
		if normalized.Buckets[i].Region == "" {
			normalized.Buckets[i].Region = defaultRegion
		}
		if normalized.Buckets[i].CreatedAt.IsZero() {
			normalized.Buckets[i].CreatedAt = fallbackBucketCreatedAt(normalized.Buckets[i].Name)
		}
	}

	sort.SliceStable(normalized.Buckets, func(i, j int) bool {
		return normalized.Buckets[i].Name < normalized.Buckets[j].Name
	})
	sort.SliceStable(normalized.Objects, func(i, j int) bool {
		if normalized.Objects[i].Bucket == normalized.Objects[j].Bucket {
			return normalized.Objects[i].Key < normalized.Objects[j].Key
		}
		return normalized.Objects[i].Bucket < normalized.Objects[j].Bucket
	})

	for i := range normalized.Objects {
		if len(normalized.Objects[i].Body) == 0 && normalized.Objects[i].Bucket == "mildstack-assets" && normalized.Objects[i].Key == "bootstrap.txt" {
			normalized.Objects[i].Body = []byte("MildStack asset v1")
		}
		if normalized.Objects[i].ContentType == "" {
			normalized.Objects[i].ContentType = "application/octet-stream"
		}
		if normalized.Objects[i].Size == 0 {
			normalized.Objects[i].Size = int64(len(normalized.Objects[i].Body))
		}
		if normalized.Objects[i].ETag == "" {
			normalized.Objects[i].ETag = computeETag(normalized.Objects[i].Body)
		}
		if normalized.Objects[i].LastModified.IsZero() {
			normalized.Objects[i].LastModified = fallbackBucketCreatedAt(normalized.Objects[i].Bucket)
		}
	}

	return normalized
}

func computeETag(body []byte) string {
	sum := md5.Sum(body)
	return `"` + hex.EncodeToString(sum[:]) + `"`
}

func fallbackBucketCreatedAt(name string) time.Time {
	if name == "mildstack-assets" {
		return time.Date(2026, time.April, 16, 0, 0, 0, 0, time.UTC)
	}

	return time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
}
