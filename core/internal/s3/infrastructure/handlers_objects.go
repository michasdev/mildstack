package infrastructure

import "github.com/michasdev/mildstack/core/internal/s3/domain"

func (h Handlers) ListObjects(request ListObjectsRequest) (ListObjectsResponse, error) {
	objects, err := h.service.ListObjects(request.Bucket)
	if err != nil {
		return ListObjectsResponse{}, err
	}

	response := ListObjectsResponse{
		Objects: make([]ObjectPayload, len(objects)),
	}
	for i, object := range objects {
		response.Objects[i] = objectPayloadFromDomain(object, false)
	}
	return response, nil
}

func (h Handlers) GetObject(request GetObjectRequest) (GetObjectResponse, error) {
	object, err := h.service.GetObject(request.Bucket, request.Key)
	if err != nil {
		return GetObjectResponse{}, err
	}
	return GetObjectResponse{
		Object: objectPayloadFromDomain(object, true),
	}, nil
}

func (h Handlers) PutObject(request PutObjectRequest) (PutObjectResponse, error) {
	object, err := h.service.PutObject(request.Bucket, request.Key, request.Body, request.ContentType)
	if err != nil {
		return PutObjectResponse{}, err
	}
	return PutObjectResponse{
		Object: objectPayloadFromDomain(object, true),
	}, nil
}

func (h Handlers) DeleteObject(request DeleteObjectRequest) (DeleteObjectResponse, error) {
	if err := h.service.DeleteObject(request.Bucket, request.Key); err != nil {
		return DeleteObjectResponse{}, err
	}
	return DeleteObjectResponse{Deleted: true}, nil
}

func (h Handlers) HeadObject(request HeadObjectRequest) (HeadObjectResponse, error) {
	object, err := h.service.HeadObject(request.Bucket, request.Key)
	if err != nil {
		return HeadObjectResponse{}, err
	}
	return HeadObjectResponse{
		Object: objectPayloadFromDomain(object, false),
	}, nil
}

func (h Handlers) CopyObject(request CopyObjectRequest) (CopyObjectResponse, error) {
	object, err := h.service.CopyObject(request.Bucket, request.Key, request.SourceBucket, request.SourceObjectKey)
	if err != nil {
		return CopyObjectResponse{}, err
	}
	return CopyObjectResponse{
		Object: objectPayloadFromDomain(object, true),
	}, nil
}

func objectPayloadFromDomain(object domain.Object, includeBody bool) ObjectPayload {
	payload := ObjectPayload{
		Bucket:       object.Bucket,
		Key:          object.Key,
		Size:         object.Size,
		ContentType:  object.ContentType,
		ETag:         object.ETag,
		LastModified: object.LastModified,
	}
	if includeBody {
		payload.Body = append([]byte(nil), object.Body...)
	}
	return payload
}
