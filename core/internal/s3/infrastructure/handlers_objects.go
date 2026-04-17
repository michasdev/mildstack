package infrastructure

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
