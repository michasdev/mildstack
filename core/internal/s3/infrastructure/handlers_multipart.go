package infrastructure

func (h Handlers) CreateMultipartUpload(request CreateMultipartUploadRequest) (CreateMultipartUploadResponse, error) {
	upload, err := h.service.CreateMultipartUpload(request.Bucket, request.Key, request.ContentType, request.Metadata, request.PreservedHeaders)
	if err != nil {
		return CreateMultipartUploadResponse{}, err
	}
	return CreateMultipartUploadResponse{
		Upload: MultipartUploadPayload{
			UploadID:    upload.UploadID,
			Bucket:      upload.Bucket,
			Key:         upload.Key,
			ContentType: upload.ContentType,
		},
	}, nil
}

func (h Handlers) UploadPart(request UploadPartRequest) (UploadPartResponse, error) {
	part, err := h.service.UploadPart(request.UploadID, request.PartNumber, request.Body)
	if err != nil {
		return UploadPartResponse{}, err
	}
	return UploadPartResponse{
		Part: MultipartPartPayload{
			PartNumber: part.PartNumber,
			ETag:       part.ETag,
			Size:       part.Size,
		},
	}, nil
}

func (h Handlers) CompleteMultipartUpload(request CompleteMultipartUploadRequest) (CompleteMultipartUploadResponse, error) {
	object, err := h.service.CompleteMultipartUpload(request.UploadID)
	if err != nil {
		return CompleteMultipartUploadResponse{}, err
	}
	return CompleteMultipartUploadResponse{
		Object: objectPayloadFromDomain(object, true),
	}, nil
}

func (h Handlers) AbortMultipartUpload(request AbortMultipartUploadRequest) (AbortMultipartUploadResponse, error) {
	if err := h.service.AbortMultipartUpload(request.UploadID); err != nil {
		return AbortMultipartUploadResponse{}, err
	}
	return AbortMultipartUploadResponse{Aborted: true}, nil
}
