package domain

import (
	"sort"
	"time"
)

type MultipartUpload struct {
	UploadID         string            `json:"upload_id"`
	Bucket           string            `json:"bucket"`
	Key              string            `json:"key"`
	ContentType      string            `json:"content_type"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	PreservedHeaders map[string]string `json:"preserved_headers,omitempty"`
	Parts            []MultipartPart   `json:"parts,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
}

type MultipartPart struct {
	PartNumber int       `json:"part_number"`
	Body       []byte    `json:"body,omitempty"`
	ETag       string    `json:"etag,omitempty"`
	Size       int64     `json:"size"`
	CreatedAt  time.Time `json:"created_at"`
}

type MultipartUploadSummary struct {
	UploadID    string    `json:"upload_id"`
	Bucket      string    `json:"bucket"`
	Key         string    `json:"key"`
	ContentType string    `json:"content_type"`
	PartCount   int       `json:"part_count"`
	CreatedAt   time.Time `json:"created_at"`
}

type MultipartPartSummary struct {
	PartNumber int       `json:"part_number"`
	ETag       string    `json:"etag,omitempty"`
	Size       int64     `json:"size"`
	CreatedAt  time.Time `json:"created_at"`
}

type ListMultipartUploadsResult struct {
	Bucket  string                   `json:"bucket"`
	Uploads []MultipartUploadSummary `json:"uploads"`
}

type ListPartsResult struct {
	Bucket   string                 `json:"bucket"`
	UploadID string                 `json:"upload_id"`
	Key      string                 `json:"key"`
	Parts    []MultipartPartSummary `json:"parts"`
}

func CloneMultipartUpload(upload MultipartUpload) MultipartUpload {
	upload.Metadata = cloneStringMap(upload.Metadata)
	upload.PreservedHeaders = cloneStringMap(upload.PreservedHeaders)
	upload.Parts = CloneMultipartParts(upload.Parts)
	return upload
}

func CloneMultipartPart(part MultipartPart) MultipartPart {
	part.Body = append([]byte(nil), part.Body...)
	return part
}

func CloneMultipartParts(parts []MultipartPart) []MultipartPart {
	cloned := make([]MultipartPart, len(parts))
	for i := range parts {
		cloned[i] = CloneMultipartPart(parts[i])
	}
	return cloned
}

func CloneMultipartUploadSummary(summary MultipartUploadSummary) MultipartUploadSummary {
	return summary
}

func CloneMultipartPartSummary(summary MultipartPartSummary) MultipartPartSummary {
	return summary
}

func CloneMultipartUploadSummaries(summaries []MultipartUploadSummary) []MultipartUploadSummary {
	cloned := make([]MultipartUploadSummary, len(summaries))
	for i := range summaries {
		cloned[i] = CloneMultipartUploadSummary(summaries[i])
	}
	return cloned
}

func CloneMultipartPartSummaries(summaries []MultipartPartSummary) []MultipartPartSummary {
	cloned := make([]MultipartPartSummary, len(summaries))
	for i := range summaries {
		cloned[i] = CloneMultipartPartSummary(summaries[i])
	}
	return cloned
}

func SortMultipartUploadSummaries(summaries []MultipartUploadSummary) {
	sort.SliceStable(summaries, func(i, j int) bool {
		if summaries[i].Bucket != summaries[j].Bucket {
			return summaries[i].Bucket < summaries[j].Bucket
		}
		if summaries[i].Key != summaries[j].Key {
			return summaries[i].Key < summaries[j].Key
		}
		if summaries[i].UploadID != summaries[j].UploadID {
			return summaries[i].UploadID < summaries[j].UploadID
		}
		return summaries[i].CreatedAt.Before(summaries[j].CreatedAt)
	})
}

func SortMultipartPartSummaries(summaries []MultipartPartSummary) {
	sort.SliceStable(summaries, func(i, j int) bool {
		if summaries[i].PartNumber != summaries[j].PartNumber {
			return summaries[i].PartNumber < summaries[j].PartNumber
		}
		if summaries[i].CreatedAt != summaries[j].CreatedAt {
			return summaries[i].CreatedAt.Before(summaries[j].CreatedAt)
		}
		return summaries[i].ETag < summaries[j].ETag
	})
}
