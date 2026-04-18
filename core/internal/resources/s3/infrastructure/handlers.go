package infrastructure

import (
	"io"
	"time"

	"github.com/michasdev/mildstack/core/internal/resources/s3/domain"
)

type Service interface {
	ListBuckets() []domain.Bucket
	CreateBucket(name, region string) (domain.Bucket, error)
	HeadBucket(name string) (domain.Bucket, error)
	GetBucketLocation(name string) (string, error)
	DeleteBucket(name string) error
	GetBucketVersioning(bucket string) (domain.BucketVersioning, error)
	PutBucketVersioning(bucket, status string) (domain.BucketVersioning, error)
	ListObjectVersions(bucket string) (domain.ListObjectVersionsResult, error)
	GetBucketPolicy(bucket string) ([]byte, error)
	PutBucketPolicy(bucket string, body []byte) ([]byte, error)
	DeleteBucketPolicy(bucket string) error
	GetBucketEncryption(bucket string) ([]byte, error)
	PutBucketEncryption(bucket string, body []byte) ([]byte, error)
	DeleteBucketEncryption(bucket string) error
	GetBucketLifecycle(bucket string) ([]byte, error)
	PutBucketLifecycle(bucket string, body []byte) ([]byte, error)
	DeleteBucketLifecycle(bucket string) error
	GetBucketCORS(bucket string) ([]byte, error)
	PutBucketCORS(bucket string, body []byte) ([]byte, error)
	DeleteBucketCORS(bucket string) error
	GetBucketACL(bucket string) ([]byte, error)
	PutBucketACL(bucket string, body []byte) ([]byte, error)
	GetBucketTagging(bucket string) ([]byte, error)
	PutBucketTagging(bucket string, body []byte) ([]byte, error)
	DeleteBucketTagging(bucket string) error
	GetBucketOwnershipControls(bucket string) ([]byte, error)
	PutBucketOwnershipControls(bucket string, body []byte) ([]byte, error)
	DeleteBucketOwnershipControls(bucket string) error
	GetPublicAccessBlock(bucket string) ([]byte, error)
	PutPublicAccessBlock(bucket string, body []byte) ([]byte, error)
	DeletePublicAccessBlock(bucket string) error
	GetBucketNotification(bucket string) ([]byte, error)
	PutBucketNotification(bucket string, body []byte) ([]byte, error)
	GetBucketLogging(bucket string) ([]byte, error)
	PutBucketLogging(bucket string, body []byte) ([]byte, error)
	GetBucketReplication(bucket string) ([]byte, error)
	PutBucketReplication(bucket string, body []byte) ([]byte, error)
	DeleteBucketReplication(bucket string) error
	GetObjectLockConfiguration(bucket string) ([]byte, error)
	PutObjectLockConfiguration(bucket string, body []byte) ([]byte, error)
	GetObjectAcl(bucket, key string) ([]byte, error)
	PutObjectAcl(bucket, key string, body []byte) ([]byte, error)
	GetObjectTagging(bucket, key string) ([]byte, error)
	PutObjectTagging(bucket, key string, body []byte) ([]byte, error)
	DeleteObjectTagging(bucket, key string) error
	GetObjectRetention(bucket, key string) ([]byte, error)
	PutObjectRetention(bucket, key string, body []byte) ([]byte, error)
	GetObjectLegalHold(bucket, key string) ([]byte, error)
	PutObjectLegalHold(bucket, key string, body []byte) ([]byte, error)
	ListObjects(bucket string) ([]domain.Object, error)
	ListObjectsV1(request domain.ListObjectsV1Request) (domain.ListObjectsV1Result, error)
	ListObjectsV2(request domain.ListObjectsV2Request) (domain.ListObjectsV2Result, error)
	GetObject(bucket, key string) (domain.Object, error)
	HeadObject(bucket, key string) (domain.Object, error)
	PutObject(bucket, key string, body io.Reader, contentType string) (domain.Object, error)
	CopyObject(bucket, key, sourceBucket, sourceKey string) (domain.Object, error)
	DeleteObject(bucket, key string) error
	DeleteObjects(request domain.DeleteObjectsRequest) (domain.DeleteObjectsResult, error)
	CreateMultipartUpload(bucket, key, contentType string, metadata, preservedHeaders map[string]string) (domain.MultipartUpload, error)
	ListMultipartUploads(bucket string) (domain.ListMultipartUploadsResult, error)
	UploadPart(uploadID string, partNumber int, body []byte) (domain.MultipartPart, error)
	ListParts(bucket, uploadID string) (domain.ListPartsResult, error)
	CompleteMultipartUpload(uploadID string) (domain.Object, error)
	AbortMultipartUpload(uploadID string) error
}

type Handlers struct {
	service Service
}

type BucketPayload struct {
	Name      string    `json:"name"`
	Region    string    `json:"region"`
	CreatedAt time.Time `json:"created_at"`
}

type ObjectPayload struct {
	Bucket       string    `json:"bucket"`
	Key          string    `json:"key"`
	Body         []byte    `json:"body,omitempty"`
	Size         int64     `json:"size"`
	ContentType  string    `json:"content_type"`
	ETag         string    `json:"etag,omitempty"`
	LastModified time.Time `json:"last_modified,omitempty"`
}

type ListBucketsResponse struct {
	Buckets []BucketPayload `json:"buckets"`
}

type CreateBucketRequest struct {
	Name   string
	Region string
}

type CreateBucketResponse struct {
	Bucket BucketPayload `json:"bucket"`
}

type HeadBucketRequest struct {
	Name string
}

type HeadBucketResponse struct {
	Bucket BucketPayload `json:"bucket"`
}

type DeleteBucketRequest struct {
	Name string
}

type DeleteBucketResponse struct {
	Deleted bool `json:"deleted"`
}

type GetBucketLocationRequest struct {
	Bucket string
}

type GetBucketLocationResponse struct {
	Location BucketLocationPayload `json:"location"`
}

type BucketVersioningPayload struct {
	Bucket string `json:"bucket"`
	Status string `json:"status"`
}

type VersionPayload struct {
	Bucket         string    `json:"bucket"`
	Key            string    `json:"key"`
	VersionID      string    `json:"version_id"`
	Sequence       int64     `json:"sequence"`
	IsDeleteMarker bool      `json:"is_delete_marker,omitempty"`
	IsLatest       bool      `json:"is_latest,omitempty"`
	Size           int64     `json:"size"`
	ContentType    string    `json:"content_type,omitempty"`
	ETag           string    `json:"etag,omitempty"`
	LastModified   time.Time `json:"last_modified,omitempty"`
}

type BucketBodyPayload struct {
	Bucket string `json:"bucket"`
	Body   []byte `json:"body,omitempty"`
}

type BucketLocationPayload struct {
	Bucket             string `json:"bucket"`
	LocationConstraint string `json:"location_constraint,omitempty"`
}

type GetBucketVersioningRequest struct {
	Bucket string
}

type GetBucketVersioningResponse struct {
	Versioning BucketVersioningPayload `json:"versioning"`
}

type PutBucketVersioningRequest struct {
	Bucket string
	Status string
}

type PutBucketVersioningResponse struct {
	Versioning BucketVersioningPayload `json:"versioning"`
}

type GetBucketPolicyRequest struct {
	Bucket string
}

type GetBucketPolicyResponse struct {
	Policy BucketBodyPayload `json:"policy"`
}

type PutBucketPolicyRequest struct {
	Bucket string
	Body   []byte
}

type PutBucketPolicyResponse struct {
	Policy BucketBodyPayload `json:"policy"`
}

type DeleteBucketPolicyRequest struct {
	Bucket string
}

type DeleteBucketPolicyResponse struct {
	Deleted bool `json:"deleted"`
}

type GetBucketEncryptionRequest struct {
	Bucket string
}

type GetBucketEncryptionResponse struct {
	Encryption BucketBodyPayload `json:"encryption"`
}

type PutBucketEncryptionRequest struct {
	Bucket string
	Body   []byte
}

type PutBucketEncryptionResponse struct {
	Encryption BucketBodyPayload `json:"encryption"`
}

type DeleteBucketEncryptionRequest struct {
	Bucket string
}

type DeleteBucketEncryptionResponse struct {
	Deleted bool `json:"deleted"`
}

type GetBucketLifecycleRequest struct {
	Bucket string
}

type GetBucketLifecycleResponse struct {
	Lifecycle BucketBodyPayload `json:"lifecycle"`
}

type PutBucketLifecycleRequest struct {
	Bucket string
	Body   []byte
}

type PutBucketLifecycleResponse struct {
	Lifecycle BucketBodyPayload `json:"lifecycle"`
}

type DeleteBucketLifecycleRequest struct {
	Bucket string
}

type DeleteBucketLifecycleResponse struct {
	Deleted bool `json:"deleted"`
}

type GetBucketCORSRequest struct {
	Bucket string
}

type GetBucketCORSResponse struct {
	CORS BucketBodyPayload `json:"cors"`
}

type PutBucketCORSRequest struct {
	Bucket string
	Body   []byte
}

type PutBucketCORSResponse struct {
	CORS BucketBodyPayload `json:"cors"`
}

type DeleteBucketCORSRequest struct {
	Bucket string
}

type DeleteBucketCORSResponse struct {
	Deleted bool `json:"deleted"`
}

type GetBucketACLRequest struct {
	Bucket string
}

type GetBucketACLResponse struct {
	ACL BucketBodyPayload `json:"acl"`
}

type PutBucketACLRequest struct {
	Bucket string
	Body   []byte
}

type PutBucketACLResponse struct {
	ACL BucketBodyPayload `json:"acl"`
}

type GetBucketTaggingRequest struct {
	Bucket string
}

type GetBucketTaggingResponse struct {
	Tagging BucketBodyPayload `json:"tagging"`
}

type PutBucketTaggingRequest struct {
	Bucket string
	Body   []byte
}

type PutBucketTaggingResponse struct {
	Tagging BucketBodyPayload `json:"tagging"`
}

type DeleteBucketTaggingRequest struct {
	Bucket string
}

type DeleteBucketTaggingResponse struct {
	Deleted bool `json:"deleted"`
}

type ListObjectVersionsRequest struct {
	Bucket string
}

type ListObjectVersionsResponse struct {
	Bucket   string           `json:"bucket"`
	Versions []VersionPayload `json:"versions"`
}

type MultipartUploadPayload struct {
	UploadID    string `json:"upload_id"`
	Bucket      string `json:"bucket"`
	Key         string `json:"key"`
	ContentType string `json:"content_type"`
}

type MultipartPartPayload struct {
	PartNumber int    `json:"part_number"`
	ETag       string `json:"etag,omitempty"`
	Size       int64  `json:"size"`
}

type MultipartUploadSummaryPayload struct {
	UploadID    string    `json:"upload_id"`
	Bucket      string    `json:"bucket"`
	Key         string    `json:"key"`
	ContentType string    `json:"content_type"`
	PartCount   int       `json:"part_count"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
}

type MultipartPartSummaryPayload struct {
	PartNumber int       `json:"part_number"`
	ETag       string    `json:"etag,omitempty"`
	Size       int64     `json:"size"`
	CreatedAt  time.Time `json:"created_at,omitempty"`
}

type CreateMultipartUploadRequest struct {
	Bucket           string
	Key              string
	ContentType      string
	Metadata         map[string]string
	PreservedHeaders map[string]string
}

type CreateMultipartUploadResponse struct {
	Upload MultipartUploadPayload `json:"upload"`
}

type ListMultipartUploadsRequest struct {
	Bucket string
}

type ListMultipartUploadsResponse struct {
	Bucket  string                          `json:"bucket"`
	Uploads []MultipartUploadSummaryPayload `json:"uploads"`
}

type UploadPartRequest struct {
	UploadID   string
	PartNumber int
	Body       []byte
}

type UploadPartResponse struct {
	Part MultipartPartPayload `json:"part"`
}

type ListPartsRequest struct {
	Bucket   string
	UploadID string
}

type ListPartsResponse struct {
	Bucket   string                        `json:"bucket"`
	UploadID string                        `json:"upload_id"`
	Key      string                        `json:"key"`
	Parts    []MultipartPartSummaryPayload `json:"parts"`
}

type CompleteMultipartUploadRequest struct {
	UploadID string
}

type CompleteMultipartUploadResponse struct {
	Object ObjectPayload `json:"object"`
}

type AbortMultipartUploadRequest struct {
	UploadID string
}

type AbortMultipartUploadResponse struct {
	Aborted bool `json:"aborted"`
}

type ListObjectsRequest struct {
	Bucket string
}

type ListObjectsResponse struct {
	Objects []ObjectPayload `json:"objects"`
}

type ListObjectsV1Request struct {
	Bucket    string
	Prefix    string
	Delimiter string
	Marker    string
	MaxKeys   int
}

type ListObjectsV1Response struct {
	Bucket         string          `json:"bucket"`
	Prefix         string          `json:"prefix,omitempty"`
	Marker         string          `json:"marker,omitempty"`
	Delimiter      string          `json:"delimiter,omitempty"`
	MaxKeys        int             `json:"max_keys"`
	IsTruncated    bool            `json:"is_truncated"`
	NextMarker     string          `json:"next_marker,omitempty"`
	Objects        []ObjectPayload `json:"objects"`
	CommonPrefixes []string        `json:"common_prefixes,omitempty"`
}

type ListObjectsV2Request struct {
	Bucket            string
	Prefix            string
	Delimiter         string
	ContinuationToken string
	StartAfter        string
	MaxKeys           int
}

type ListObjectsV2Response struct {
	Bucket                string          `json:"bucket"`
	Prefix                string          `json:"prefix,omitempty"`
	Delimiter             string          `json:"delimiter,omitempty"`
	ContinuationToken     string          `json:"continuation_token,omitempty"`
	StartAfter            string          `json:"start_after,omitempty"`
	MaxKeys               int             `json:"max_keys"`
	KeyCount              int             `json:"key_count"`
	IsTruncated           bool            `json:"is_truncated"`
	NextContinuationToken string          `json:"next_continuation_token,omitempty"`
	Objects               []ObjectPayload `json:"objects"`
	CommonPrefixes        []string        `json:"common_prefixes,omitempty"`
}

type GetObjectRequest struct {
	Bucket string
	Key    string
}

type GetObjectResponse struct {
	Object ObjectPayload `json:"object"`
}

type PutObjectRequest struct {
	Bucket      string
	Key         string
	Body        []byte
	ContentType string
}

type PutObjectResponse struct {
	Object ObjectPayload `json:"object"`
}

type DeleteObjectRequest struct {
	Bucket string
	Key    string
}

type DeleteObjectResponse struct {
	Deleted bool `json:"deleted"`
}

type DeleteObjectsRequest struct {
	Bucket string
	Keys   []string
	Quiet  bool
}

type DeletedObjectPayload struct {
	Key string `json:"key"`
}

type DeleteObjectsErrorPayload struct {
	Key     string `json:"key"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type DeleteObjectsResponse struct {
	Deleted []DeletedObjectPayload      `json:"deleted,omitempty"`
	Errors  []DeleteObjectsErrorPayload `json:"errors,omitempty"`
}

type HeadObjectRequest struct {
	Bucket string
	Key    string
}

type HeadObjectResponse struct {
	Object ObjectPayload `json:"object"`
}

type CopyObjectRequest struct {
	Bucket          string
	Key             string
	SourceBucket    string
	SourceObjectKey string
}

type CopyObjectResponse struct {
	CopyResult CopyObjectResultPayload `json:"copy_result"`
}

type CopyObjectResultPayload struct {
	LastModified time.Time `json:"last_modified,omitempty"`
	ETag         string    `json:"etag,omitempty"`
}

func NewHandlers(service Service) Handlers {
	return Handlers{service: service}
}

func bucketPayloadFromDomain(bucket domain.Bucket) BucketPayload {
	return BucketPayload{
		Name:      bucket.Name,
		Region:    bucket.Region,
		CreatedAt: bucket.CreatedAt,
	}
}

func bucketPayloadsFromDomain(buckets []domain.Bucket) []BucketPayload {
	payloads := make([]BucketPayload, len(buckets))
	for i, bucket := range buckets {
		payloads[i] = bucketPayloadFromDomain(bucket)
	}
	return payloads
}

func bucketLocationPayloadFromRegion(bucket, region string) BucketLocationPayload {
	return BucketLocationPayload{
		Bucket:             bucket,
		LocationConstraint: region,
	}
}
