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
	"strings"
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
	normalized := normalizeState(state)
	if err := validateState(normalized); err != nil {
		return domain.State{}, err
	}

	return normalized, nil
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

	for _, setting := range state.BucketVersioning {
		if setting.Bucket == "" {
			return fmt.Errorf("invalid bucket versioning entry: empty bucket")
		}
		if _, ok := buckets[setting.Bucket]; !ok {
			return fmt.Errorf("invalid bucket versioning %q: bucket not found", setting.Bucket)
		}
		switch setting.Status {
		case "", domain.VersioningEnabled, domain.VersioningSuspended:
		default:
			return fmt.Errorf("invalid bucket versioning %q: unsupported status %q", setting.Bucket, setting.Status)
		}
	}

	for _, record := range state.VersionHistory {
		if record.Bucket == "" || record.Key == "" {
			return fmt.Errorf("invalid version record key %q for bucket %q", record.Key, record.Bucket)
		}
		if _, ok := buckets[record.Bucket]; !ok {
			return fmt.Errorf("invalid version record %s/%s: bucket not found", record.Bucket, record.Key)
		}
		if record.VersionID == "" {
			return fmt.Errorf("invalid version record %s/%s: empty version id", record.Bucket, record.Key)
		}
		if record.Sequence < 0 {
			return fmt.Errorf("invalid version record %s/%s: negative sequence", record.Bucket, record.Key)
		}
		if !record.IsDeleteMarker && record.ContentType == "" {
			return fmt.Errorf("invalid version record %s/%s: empty content type", record.Bucket, record.Key)
		}
	}

	for _, entry := range []struct {
		name  string
		value map[string][]byte
	}{
		{name: "bucket policies", value: state.BucketPolicies},
		{name: "bucket encryption", value: state.BucketEncryption},
		{name: "bucket lifecycle", value: state.BucketLifecycle},
		{name: "bucket cors", value: state.BucketCORS},
		{name: "bucket acl", value: state.BucketACL},
		{name: "bucket tagging", value: state.BucketTagging},
		{name: "bucket notifications", value: state.BucketNotifications},
		{name: "bucket logging", value: state.BucketLogging},
	} {
		for bucket := range entry.value {
			if bucket == "" {
				return fmt.Errorf("invalid %s entry: empty bucket", entry.name)
			}
			if _, ok := buckets[bucket]; !ok {
				return fmt.Errorf("invalid %s %q: bucket not found", entry.name, bucket)
			}
		}
	}

	for bucket, config := range state.BucketReplication {
		if bucket == "" {
			return fmt.Errorf("invalid bucket replication entry: empty bucket")
		}
		if _, ok := buckets[bucket]; !ok {
			return fmt.Errorf("invalid bucket replication %q: bucket not found", bucket)
		}
		if strings.TrimSpace(config.Role) == "" {
			return fmt.Errorf("invalid bucket replication %q: empty role", bucket)
		}
		if len(config.Rules) == 0 {
			return fmt.Errorf("invalid bucket replication %q: empty rules", bucket)
		}
	}

	return nil
}

func normalizeState(state domain.State) domain.State {
	normalized := domain.State{
		Service:             state.Service,
		Buckets:             make([]domain.Bucket, len(state.Buckets)),
		Objects:             make([]domain.Object, len(state.Objects)),
		BucketVersioning:    make([]domain.BucketVersioning, len(state.BucketVersioning)),
		VersionHistory:      make(domain.VersionHistory, len(state.VersionHistory)),
		BucketPolicies:      cloneBucketBodies(state.BucketPolicies),
		BucketEncryption:    cloneBucketBodies(state.BucketEncryption),
		BucketLifecycle:     cloneBucketBodies(state.BucketLifecycle),
		BucketCORS:          cloneBucketBodies(state.BucketCORS),
		BucketACL:           cloneBucketBodies(state.BucketACL),
		BucketTagging:       cloneBucketBodies(state.BucketTagging),
		BucketNotifications: cloneBucketBodies(state.BucketNotifications),
		BucketLogging:       cloneBucketBodies(state.BucketLogging),
		BucketReplication:   cloneBucketReplicationConfigs(state.BucketReplication),
	}
	copy(normalized.Buckets, state.Buckets)
	copy(normalized.BucketVersioning, state.BucketVersioning)
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
	for i := range state.VersionHistory {
		normalized.VersionHistory[i] = state.VersionHistory[i]
		normalized.VersionHistory[i].Body = append([]byte(nil), state.VersionHistory[i].Body...)
		if len(state.VersionHistory[i].Metadata) > 0 {
			normalized.VersionHistory[i].Metadata = make(map[string]string, len(state.VersionHistory[i].Metadata))
			for key, value := range state.VersionHistory[i].Metadata {
				normalized.VersionHistory[i].Metadata[key] = value
			}
		}
		if len(state.VersionHistory[i].PreservedHeaders) > 0 {
			normalized.VersionHistory[i].PreservedHeaders = make(map[string]string, len(state.VersionHistory[i].PreservedHeaders))
			for key, value := range state.VersionHistory[i].PreservedHeaders {
				normalized.VersionHistory[i].PreservedHeaders[key] = value
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
	sort.SliceStable(normalized.BucketVersioning, func(i, j int) bool {
		return normalized.BucketVersioning[i].Bucket < normalized.BucketVersioning[j].Bucket
	})
	sort.SliceStable(normalized.VersionHistory, func(i, j int) bool {
		if normalized.VersionHistory[i].Bucket == normalized.VersionHistory[j].Bucket {
			if normalized.VersionHistory[i].Key == normalized.VersionHistory[j].Key {
				if normalized.VersionHistory[i].Sequence == normalized.VersionHistory[j].Sequence {
					return normalized.VersionHistory[i].VersionID < normalized.VersionHistory[j].VersionID
				}
				return normalized.VersionHistory[i].Sequence < normalized.VersionHistory[j].Sequence
			}
			return normalized.VersionHistory[i].Key < normalized.VersionHistory[j].Key
		}
		return normalized.VersionHistory[i].Bucket < normalized.VersionHistory[j].Bucket
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
	for i := range normalized.VersionHistory {
		sequence := normalized.VersionHistory[i].Sequence
		if sequence == 0 {
			sequence = int64(i) + 1
			normalized.VersionHistory[i].Sequence = sequence
		}
		if normalized.VersionHistory[i].VersionID == "" {
			normalized.VersionHistory[i].VersionID = fmt.Sprintf("v%020d", sequence)
		}
		if normalized.VersionHistory[i].LastModified.IsZero() {
			normalized.VersionHistory[i].LastModified = fallbackBucketCreatedAt(normalized.VersionHistory[i].Bucket)
		}
		if !normalized.VersionHistory[i].IsDeleteMarker && normalized.VersionHistory[i].ContentType == "" {
			normalized.VersionHistory[i].ContentType = "application/octet-stream"
		}
		if normalized.VersionHistory[i].IsDeleteMarker {
			normalized.VersionHistory[i].ContentType = ""
			normalized.VersionHistory[i].Size = 0
			normalized.VersionHistory[i].Body = nil
		} else if normalized.VersionHistory[i].Size == 0 {
			normalized.VersionHistory[i].Size = int64(len(normalized.VersionHistory[i].Body))
		}
		if normalized.VersionHistory[i].ETag == "" && !normalized.VersionHistory[i].IsDeleteMarker {
			normalized.VersionHistory[i].ETag = computeETag(normalized.VersionHistory[i].Body)
		}
	}

	normalized.BucketPolicies = pruneOrphanedBucketBodies(normalized.BucketPolicies, normalized.Buckets)
	normalized.BucketEncryption = pruneOrphanedBucketBodies(normalized.BucketEncryption, normalized.Buckets)
	normalized.BucketLifecycle = pruneOrphanedBucketBodies(normalized.BucketLifecycle, normalized.Buckets)
	normalized.BucketCORS = pruneOrphanedBucketBodies(normalized.BucketCORS, normalized.Buckets)
	normalized.BucketACL = pruneOrphanedBucketBodies(normalized.BucketACL, normalized.Buckets)
	normalized.BucketTagging = pruneOrphanedBucketBodies(normalized.BucketTagging, normalized.Buckets)
	normalized.BucketNotifications = pruneOrphanedBucketBodies(normalized.BucketNotifications, normalized.Buckets)
	normalized.BucketLogging = pruneOrphanedBucketBodies(normalized.BucketLogging, normalized.Buckets)
	normalized.BucketReplication = pruneOrphanedBucketReplication(normalized.BucketReplication, normalized.Buckets)

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

func cloneBucketBodies(values map[string][]byte) map[string][]byte {
	if len(values) == 0 {
		return nil
	}

	cloned := make(map[string][]byte, len(values))
	for bucket, body := range values {
		cloned[bucket] = append([]byte(nil), body...)
	}
	return cloned
}

func pruneOrphanedBucketBodies(values map[string][]byte, buckets []domain.Bucket) map[string][]byte {
	if len(values) == 0 {
		return nil
	}

	known := make(map[string]struct{}, len(buckets))
	for _, bucket := range buckets {
		known[bucket.Name] = struct{}{}
	}

	pruned := make(map[string][]byte, len(values))
	for bucket, body := range values {
		if _, ok := known[bucket]; !ok {
			continue
		}
		pruned[bucket] = append([]byte(nil), body...)
	}
	if len(pruned) == 0 {
		return nil
	}
	return pruned
}

func cloneBucketReplicationConfigs(values map[string]domain.BucketReplicationConfig) map[string]domain.BucketReplicationConfig {
	if len(values) == 0 {
		return nil
	}

	cloned := make(map[string]domain.BucketReplicationConfig, len(values))
	for bucket, config := range values {
		cloned[bucket] = cloneBucketReplicationConfig(config)
	}
	return cloned
}

func pruneOrphanedBucketReplication(values map[string]domain.BucketReplicationConfig, buckets []domain.Bucket) map[string]domain.BucketReplicationConfig {
	if len(values) == 0 {
		return nil
	}

	known := make(map[string]struct{}, len(buckets))
	for _, bucket := range buckets {
		known[bucket.Name] = struct{}{}
	}

	pruned := make(map[string]domain.BucketReplicationConfig, len(values))
	for bucket, config := range values {
		if _, ok := known[bucket]; !ok {
			continue
		}
		pruned[bucket] = normalizeBucketReplicationConfig(config)
	}
	if len(pruned) == 0 {
		return nil
	}
	return pruned
}

func normalizeBucketReplicationConfig(config domain.BucketReplicationConfig) domain.BucketReplicationConfig {
	config = cloneBucketReplicationConfig(config)
	for i := range config.Rules {
		if config.Rules[i].ID == "" {
			config.Rules[i].ID = fmt.Sprintf("rule-%d", i+1)
		}
		if config.Rules[i].Status == "" {
			config.Rules[i].Status = "Enabled"
		}
	}
	return config
}

func cloneBucketReplicationConfig(config domain.BucketReplicationConfig) domain.BucketReplicationConfig {
	cloned := domain.BucketReplicationConfig{
		Role:  config.Role,
		Rules: make([]domain.BucketReplicationRule, len(config.Rules)),
	}
	copy(cloned.Rules, config.Rules)
	return cloned
}
