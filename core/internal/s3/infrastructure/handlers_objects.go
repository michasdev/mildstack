package infrastructure

import (
	"bytes"
	"github.com/michasdev/mildstack/core/internal/s3/domain"
)

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

func (h Handlers) ListObjectsV1(request ListObjectsV1Request) (ListObjectsV1Response, error) {
	result, err := h.service.ListObjectsV1(domain.ListObjectsV1Request{
		Bucket:    request.Bucket,
		Prefix:    request.Prefix,
		Delimiter: request.Delimiter,
		Marker:    request.Marker,
		MaxKeys:   request.MaxKeys,
	})
	if err != nil {
		return ListObjectsV1Response{}, err
	}

	return ListObjectsV1Response{
		Bucket:         result.Bucket,
		Prefix:         result.Prefix,
		Marker:         result.Marker,
		Delimiter:      result.Delimiter,
		MaxKeys:        result.MaxKeys,
		IsTruncated:    result.IsTruncated,
		NextMarker:     result.NextMarker,
		Objects:        objectPayloadsFromDomain(result.Objects, false),
		CommonPrefixes: append([]string(nil), result.CommonPrefixes...),
	}, nil
}

func (h Handlers) ListObjectsV2(request ListObjectsV2Request) (ListObjectsV2Response, error) {
	result, err := h.service.ListObjectsV2(domain.ListObjectsV2Request{
		Bucket:            request.Bucket,
		Prefix:            request.Prefix,
		Delimiter:         request.Delimiter,
		ContinuationToken: request.ContinuationToken,
		StartAfter:        request.StartAfter,
		MaxKeys:           request.MaxKeys,
	})
	if err != nil {
		return ListObjectsV2Response{}, err
	}

	return ListObjectsV2Response{
		Bucket:                result.Bucket,
		Prefix:                result.Prefix,
		Delimiter:             result.Delimiter,
		ContinuationToken:     result.ContinuationToken,
		StartAfter:            result.StartAfter,
		MaxKeys:               result.MaxKeys,
		KeyCount:              result.KeyCount,
		IsTruncated:           result.IsTruncated,
		NextContinuationToken: result.NextContinuationToken,
		Objects:               objectPayloadsFromDomain(result.Objects, false),
		CommonPrefixes:        append([]string(nil), result.CommonPrefixes...),
	}, nil
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
	object, err := h.service.PutObject(request.Bucket, request.Key, bytes.NewReader(request.Body), request.ContentType)
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

func (h Handlers) DeleteObjects(request DeleteObjectsRequest) (DeleteObjectsResponse, error) {
	result, err := h.service.DeleteObjects(domain.DeleteObjectsRequest{
		Bucket: request.Bucket,
		Keys:   append([]string(nil), request.Keys...),
		Quiet:  request.Quiet,
	})
	if err != nil {
		return DeleteObjectsResponse{}, err
	}

	response := DeleteObjectsResponse{
		Deleted: make([]DeletedObjectPayload, len(result.Deleted)),
		Errors:  make([]DeleteObjectsErrorPayload, len(result.Errors)),
	}
	for i, deleted := range result.Deleted {
		response.Deleted[i] = DeletedObjectPayload{Key: deleted.Key}
	}
	for i, deleteErr := range result.Errors {
		response.Errors[i] = DeleteObjectsErrorPayload{
			Key:     deleteErr.Key,
			Code:    deleteErr.Code,
			Message: deleteErr.Message,
		}
	}
	return response, nil
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
		CopyResult: copyObjectResultFromDomain(object),
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

func objectPayloadsFromDomain(objects []domain.Object, includeBody bool) []ObjectPayload {
	payloads := make([]ObjectPayload, len(objects))
	for i, object := range objects {
		payloads[i] = objectPayloadFromDomain(object, includeBody)
	}
	return payloads
}

func copyObjectResultFromDomain(object domain.Object) CopyObjectResultPayload {
	return CopyObjectResultPayload{
		LastModified: object.LastModified,
		ETag:         object.ETag,
	}
}
