package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	VersioningEnabled   = "Enabled"
	VersioningSuspended = "Suspended"
	VersioningNull      = "null"
)

type BucketVersioning struct {
	Bucket string `json:"bucket"`
	Status string `json:"status"`
}

type VersionRecord struct {
	Bucket           string            `json:"bucket"`
	Key              string            `json:"key"`
	VersionID        string            `json:"version_id"`
	Sequence         int64             `json:"sequence"`
	IsDeleteMarker   bool              `json:"is_delete_marker,omitempty"`
	Body             []byte            `json:"body,omitempty"`
	PayloadRef       string            `json:"payload_ref,omitempty"`
	Size             int64             `json:"size"`
	ContentType      string            `json:"content_type,omitempty"`
	ETag             string            `json:"etag,omitempty"`
	LastModified     time.Time         `json:"last_modified,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	PreservedHeaders map[string]string `json:"preserved_headers,omitempty"`
}

type DeleteMarker = VersionRecord

type VersionHistory []VersionRecord

type ListObjectVersionsRequest struct {
	Bucket string
}

type ListObjectVersionsResult struct {
	Bucket   string
	Versions []VersionRecord
}

func (s State) BucketVersioningStatus(bucket string) string {
	for _, setting := range s.BucketVersioning {
		if setting.Bucket == bucket {
			return setting.Status
		}
	}
	return ""
}

func (s State) VersioningEnabled(bucket string) bool {
	switch s.BucketVersioningStatus(bucket) {
	case VersioningEnabled, VersioningSuspended:
		return true
	default:
		return false
	}
}

func (s *State) SetBucketVersioning(bucket, status string) BucketVersioning {
	bucket = strings.TrimSpace(bucket)
	status = strings.TrimSpace(status)
	if bucket == "" {
		return BucketVersioning{}
	}

	for i := range s.BucketVersioning {
		if s.BucketVersioning[i].Bucket == bucket {
			if status == "" {
				s.BucketVersioning = append(s.BucketVersioning[:i], s.BucketVersioning[i+1:]...)
				return BucketVersioning{Bucket: bucket}
			}
			s.BucketVersioning[i].Status = status
			return s.BucketVersioning[i]
		}
	}

	if status == "" {
		return BucketVersioning{Bucket: bucket}
	}

	setting := BucketVersioning{Bucket: bucket, Status: status}
	s.BucketVersioning = append(s.BucketVersioning, setting)
	return setting
}

func (s State) ListObjectVersions(bucket string) []VersionRecord {
	history := make([]VersionRecord, 0, len(s.VersionHistory))
	for _, record := range s.VersionHistory {
		if record.Bucket != bucket {
			continue
		}
		history = append(history, cloneVersionRecord(record))
	}

	seen := make(map[string]struct{}, len(history))
	for _, record := range history {
		seen[record.Key] = struct{}{}
	}

	for _, object := range s.Objects {
		if object.Bucket != bucket {
			continue
		}
		if _, ok := seen[object.Key]; ok {
			continue
		}
		history = append(history, versionRecordFromObject(object, VersioningNull, 0))
	}

	sort.SliceStable(history, func(i, j int) bool {
		if history[i].Key == history[j].Key {
			if history[i].Sequence == history[j].Sequence {
				return history[i].VersionID > history[j].VersionID
			}
			return history[i].Sequence > history[j].Sequence
		}
		return history[i].Key < history[j].Key
	})

	return history
}

func (s *State) RecordObjectVersion(object Object) VersionRecord {
	sequence := s.nextVersionSequence()
	record := versionRecordFromObject(object, s.versionIDForSequence(sequence), sequence)
	s.VersionHistory = append(s.VersionHistory, record)
	return cloneVersionRecord(record)
}

func (s *State) RecordDeleteMarker(bucket, key string) DeleteMarker {
	sequence := s.nextVersionSequence()
	record := DeleteMarker{
		Bucket:         bucket,
		Key:            key,
		VersionID:      s.versionIDForSequence(sequence),
		Sequence:       sequence,
		IsDeleteMarker: true,
		LastModified:   time.Now().UTC(),
	}
	s.VersionHistory = append(s.VersionHistory, VersionRecord(record))
	return cloneVersionRecord(VersionRecord(record))
}

func (s State) VersionCount(bucket string, key string) int {
	count := 0
	for _, record := range s.VersionHistory {
		if record.Bucket == bucket && record.Key == key {
			count++
		}
	}
	return count
}

func (h VersionHistory) sorted() []VersionRecord {
	records := make([]VersionRecord, len(h))
	for i := range h {
		records[i] = cloneVersionRecord(h[i])
	}
	sort.SliceStable(records, func(i, j int) bool {
		if records[i].Bucket == records[j].Bucket {
			if records[i].Key == records[j].Key {
				if records[i].Sequence == records[j].Sequence {
					return records[i].VersionID < records[j].VersionID
				}
				return records[i].Sequence < records[j].Sequence
			}
			return records[i].Key < records[j].Key
		}
		return records[i].Bucket < records[j].Bucket
	})
	return records
}

func (h VersionHistory) nextSequence() int64 {
	var max int64
	for _, record := range h {
		if record.Sequence > max {
			max = record.Sequence
		}
	}
	return max + 1
}

func (h VersionHistory) hasBucket(bucket string) bool {
	for _, record := range h {
		if record.Bucket == bucket {
			return true
		}
	}
	return false
}

func (s *State) nextVersionSequence() int64 {
	return s.VersionHistory.nextSequence()
}

func (s State) versionIDForSequence(sequence int64) string {
	return fmt.Sprintf("v%020d", sequence)
}

func (s *State) removeBucketVersioning(bucket string) {
	if len(s.BucketVersioning) == 0 {
		return
	}

	filtered := s.BucketVersioning[:0]
	for _, setting := range s.BucketVersioning {
		if setting.Bucket != bucket {
			filtered = append(filtered, setting)
		}
	}
	if len(filtered) == 0 {
		s.BucketVersioning = nil
		return
	}
	s.BucketVersioning = append([]BucketVersioning(nil), filtered...)
}

func hasVersionHistoryForBucket(history VersionHistory, bucket string) bool {
	return history.hasBucket(bucket)
}

func cloneVersionRecord(record VersionRecord) VersionRecord {
	record.Body = append([]byte(nil), record.Body...)
	record.Metadata = cloneStringMap(record.Metadata)
	record.PreservedHeaders = cloneStringMap(record.PreservedHeaders)
	return record
}

func versionRecordFromObject(object Object, versionID string, sequence int64) VersionRecord {
	record := VersionRecord{
		Bucket:           object.Bucket,
		Key:              object.Key,
		VersionID:        versionID,
		Sequence:         sequence,
		Body:             append([]byte(nil), object.Body...),
		PayloadRef:       object.PayloadRef,
		Size:             object.Size,
		ContentType:      object.ContentType,
		ETag:             object.ETag,
		LastModified:     object.LastModified,
		Metadata:         cloneStringMap(object.Metadata),
		PreservedHeaders: cloneStringMap(object.PreservedHeaders),
	}
	if record.Size == 0 {
		record.Size = int64(len(record.Body))
	}
	return record
}
