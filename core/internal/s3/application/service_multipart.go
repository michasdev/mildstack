package application

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/michasdev/mildstack/core/internal/s3/domain"
)

func (s *Service) CreateMultipartUpload(bucket, key, contentType string, metadata, preservedHeaders map[string]string) (domain.MultipartUpload, error) {
	bucket = strings.TrimSpace(bucket)
	key = strings.TrimSpace(key)
	contentType = strings.TrimSpace(contentType)
	if bucket == "" {
		return domain.MultipartUpload{}, fmt.Errorf("s3: bucket name is required")
	}
	if key == "" {
		return domain.MultipartUpload{}, fmt.Errorf("s3: object key is required")
	}
	if !s.state.HasBucket(bucket) {
		return domain.MultipartUpload{}, fmt.Errorf("s3: NoSuchBucket: bucket %q not found", bucket)
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	uploadID := s.nextMultipartUploadID()
	upload := domain.MultipartUpload{
		UploadID:         uploadID,
		Bucket:           bucket,
		Key:              key,
		ContentType:      contentType,
		Metadata:         cloneObjectStringMap(metadata),
		PreservedHeaders: cloneObjectStringMap(preservedHeaders),
		CreatedAt:        time.Now().UTC(),
	}
	s.multipartUploads[uploadID] = domain.CloneMultipartUpload(upload)
	return domain.CloneMultipartUpload(upload), nil
}

func (s *Service) UploadPart(uploadID string, partNumber int, body []byte) (domain.MultipartPart, error) {
	uploadID = strings.TrimSpace(uploadID)
	if uploadID == "" {
		return domain.MultipartPart{}, fmt.Errorf("s3: upload id is required")
	}
	if partNumber < 1 {
		return domain.MultipartPart{}, fmt.Errorf("s3: invalid part number %d", partNumber)
	}

	upload, ok := s.multipartUploads[uploadID]
	if !ok {
		return domain.MultipartPart{}, fmt.Errorf("s3: NoSuchUpload: multipart upload %q not found", uploadID)
	}

	part := domain.MultipartPart{
		PartNumber: partNumber,
		Body:       append([]byte(nil), body...),
		Size:       int64(len(body)),
		ETag:       multipartETag(body),
		CreatedAt:  time.Now().UTC(),
	}
	upload.Parts = upsertMultipartPart(upload.Parts, part)
	s.multipartUploads[uploadID] = domain.CloneMultipartUpload(upload)
	return domain.CloneMultipartPart(part), nil
}

func (s *Service) CompleteMultipartUpload(uploadID string) (domain.Object, error) {
	uploadID = strings.TrimSpace(uploadID)
	if uploadID == "" {
		return domain.Object{}, fmt.Errorf("s3: upload id is required")
	}

	upload, ok := s.multipartUploads[uploadID]
	if !ok {
		return domain.Object{}, fmt.Errorf("s3: NoSuchUpload: multipart upload %q not found", uploadID)
	}
	parts := sortedMultipartParts(upload.Parts)
	if len(parts) == 0 {
		return domain.Object{}, fmt.Errorf("s3: multipart upload %q has no parts", uploadID)
	}
	if !s.state.HasBucket(upload.Bucket) {
		return domain.Object{}, fmt.Errorf("s3: NoSuchBucket: bucket %q not found", upload.Bucket)
	}
	if s.state.HasObject(upload.Bucket, upload.Key) {
		if err := s.objectMutationBlocked(upload.Bucket, upload.Key); err != nil {
			return domain.Object{}, err
		}
	}

	assembled := make([]byte, 0)
	md5Digests := make([]byte, 0, len(parts)*md5.Size)
	for _, part := range parts {
		digest := md5.Sum(part.Body)
		md5Digests = append(md5Digests, digest[:]...)
		assembled = append(assembled, part.Body...)
	}
	etagDigest := md5.Sum(md5Digests)
	finalETag := fmt.Sprintf("\"%s-%d\"", hex.EncodeToString(etagDigest[:]), len(parts))

	object, err := s.storeObject(domain.Object{
		Bucket:           upload.Bucket,
		Key:              upload.Key,
		Body:             assembled,
		Size:             int64(len(assembled)),
		ContentType:      upload.ContentType,
		ETag:             finalETag,
		Metadata:         cloneObjectStringMap(upload.Metadata),
		PreservedHeaders: cloneObjectStringMap(upload.PreservedHeaders),
	})
	if err != nil {
		return domain.Object{}, err
	}
	s.clearObjectProtection(upload.Bucket, upload.Key)
	s.applyDefaultRetention(upload.Bucket, upload.Key)
	if err := s.persist(); err != nil {
		return domain.Object{}, err
	}

	delete(s.multipartUploads, uploadID)
	return object, nil
}

func (s *Service) AbortMultipartUpload(uploadID string) error {
	uploadID = strings.TrimSpace(uploadID)
	if uploadID == "" {
		return fmt.Errorf("s3: upload id is required")
	}
	if _, ok := s.multipartUploads[uploadID]; !ok {
		return fmt.Errorf("s3: NoSuchUpload: multipart upload %q not found", uploadID)
	}
	delete(s.multipartUploads, uploadID)
	return nil
}

func (s *Service) removeMultipartUploads(bucket string) {
	for uploadID, upload := range s.multipartUploads {
		if upload.Bucket == bucket {
			delete(s.multipartUploads, uploadID)
		}
	}
}

func (s *Service) nextMultipartUploadID() string {
	return fmt.Sprintf("upload-%d-%d", time.Now().UTC().UnixNano(), len(s.multipartUploads)+1)
}

func upsertMultipartPart(parts []domain.MultipartPart, part domain.MultipartPart) []domain.MultipartPart {
	cloned := make([]domain.MultipartPart, 0, len(parts)+1)
	replaced := false
	for _, existing := range parts {
		if existing.PartNumber == part.PartNumber {
			cloned = append(cloned, domain.CloneMultipartPart(part))
			replaced = true
			continue
		}
		cloned = append(cloned, domain.CloneMultipartPart(existing))
	}
	if !replaced {
		cloned = append(cloned, domain.CloneMultipartPart(part))
	}
	return cloned
}

func sortedMultipartParts(parts []domain.MultipartPart) []domain.MultipartPart {
	cloned := domain.CloneMultipartParts(parts)
	sort.SliceStable(cloned, func(i, j int) bool {
		return cloned[i].PartNumber < cloned[j].PartNumber
	})
	return cloned
}

func multipartETag(body []byte) string {
	sum := md5.Sum(body)
	return `"` + hex.EncodeToString(sum[:]) + `"`
}
