package infrastructure

import (
	"time"

	"github.com/michasdev/mildstack/core/internal/s3/domain"
)

type Service interface {
	ListBuckets() []domain.Bucket
	CreateBucket(name, region string) (domain.Bucket, error)
	HeadBucket(name string) (domain.Bucket, error)
	DeleteBucket(name string) error
	ListObjects(bucket string) ([]domain.Object, error)
	GetObject(bucket, key string) (domain.Object, error)
	PutObject(bucket, key string, size int64, contentType string) (domain.Object, error)
	DeleteObject(bucket, key string) error
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
	Bucket      string `json:"bucket"`
	Key         string `json:"key"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
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

type ListObjectsRequest struct {
	Bucket string
}

type ListObjectsResponse struct {
	Objects []ObjectPayload `json:"objects"`
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
	Size        int64
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

func NewHandlers(service Service) Handlers {
	return Handlers{service: service}
}
