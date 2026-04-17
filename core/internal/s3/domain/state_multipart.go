package domain

import "time"

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
