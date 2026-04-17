package infrastructure

import "github.com/michasdev/mildstack/core/internal/s3/domain"

type Service interface {
	ListBuckets() []domain.Bucket
	CreateBucket(name, region string) (domain.Bucket, error)
	ListObjects(bucket string) ([]domain.Object, error)
	GetObject(bucket, key string) (domain.Object, error)
	PutObject(bucket, key string, size int64, contentType string) (domain.Object, error)
	DeleteObject(bucket, key string) error
}

type Handlers struct {
	service Service
}

type BucketPayload struct {
	Name   string `json:"name"`
	Region string `json:"region"`
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

func (h Handlers) ListBuckets() ListBucketsResponse {
	buckets := h.service.ListBuckets()
	response := ListBucketsResponse{
		Buckets: make([]BucketPayload, len(buckets)),
	}
	for i, bucket := range buckets {
		response.Buckets[i] = BucketPayload{
			Name:   bucket.Name,
			Region: bucket.Region,
		}
	}
	return response
}

func (h Handlers) CreateBucket(request CreateBucketRequest) (CreateBucketResponse, error) {
	bucket, err := h.service.CreateBucket(request.Name, request.Region)
	if err != nil {
		return CreateBucketResponse{}, err
	}
	return CreateBucketResponse{
		Bucket: BucketPayload{
			Name:   bucket.Name,
			Region: bucket.Region,
		},
	}, nil
}

func (h Handlers) ListObjects(request ListObjectsRequest) (ListObjectsResponse, error) {
	objects, err := h.service.ListObjects(request.Bucket)
	if err != nil {
		return ListObjectsResponse{}, err
	}

	response := ListObjectsResponse{
		Objects: make([]ObjectPayload, len(objects)),
	}
	for i, object := range objects {
		response.Objects[i] = ObjectPayload{
			Bucket:      object.Bucket,
			Key:         object.Key,
			Size:        object.Size,
			ContentType: object.ContentType,
		}
	}
	return response, nil
}

func (h Handlers) GetObject(request GetObjectRequest) (GetObjectResponse, error) {
	object, err := h.service.GetObject(request.Bucket, request.Key)
	if err != nil {
		return GetObjectResponse{}, err
	}
	return GetObjectResponse{
		Object: ObjectPayload{
			Bucket:      object.Bucket,
			Key:         object.Key,
			Size:        object.Size,
			ContentType: object.ContentType,
		},
	}, nil
}

func (h Handlers) PutObject(request PutObjectRequest) (PutObjectResponse, error) {
	object, err := h.service.PutObject(request.Bucket, request.Key, request.Size, request.ContentType)
	if err != nil {
		return PutObjectResponse{}, err
	}
	return PutObjectResponse{
		Object: ObjectPayload{
			Bucket:      object.Bucket,
			Key:         object.Key,
			Size:        object.Size,
			ContentType: object.ContentType,
		},
	}, nil
}

func (h Handlers) DeleteObject(request DeleteObjectRequest) (DeleteObjectResponse, error) {
	if err := h.service.DeleteObject(request.Bucket, request.Key); err != nil {
		return DeleteObjectResponse{}, err
	}
	return DeleteObjectResponse{Deleted: true}, nil
}
