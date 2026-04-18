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

func (h Handlers) ListMultipartUploads(request ListMultipartUploadsRequest) (ListMultipartUploadsResponse, error) {
	result, err := h.service.ListMultipartUploads(request.Bucket)
	if err != nil {
		return ListMultipartUploadsResponse{}, err
	}

	uploads := make([]MultipartUploadSummaryPayload, len(result.Uploads))
	for i, upload := range result.Uploads {
		uploads[i] = MultipartUploadSummaryPayload{
			UploadID:    upload.UploadID,
			Bucket:      upload.Bucket,
			Key:         upload.Key,
			ContentType: upload.ContentType,
			PartCount:   upload.PartCount,
			CreatedAt:   upload.CreatedAt,
		}
	}

	return ListMultipartUploadsResponse{
		Bucket:  result.Bucket,
		Uploads: uploads,
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

func (h Handlers) ListParts(request ListPartsRequest) (ListPartsResponse, error) {
	result, err := h.service.ListParts(request.Bucket, request.UploadID)
	if err != nil {
		return ListPartsResponse{}, err
	}

	parts := make([]MultipartPartSummaryPayload, len(result.Parts))
	for i, part := range result.Parts {
		parts[i] = MultipartPartSummaryPayload{
			PartNumber: part.PartNumber,
			ETag:       part.ETag,
			Size:       part.Size,
			CreatedAt:  part.CreatedAt,
		}
	}

	return ListPartsResponse{
		Bucket:   result.Bucket,
		UploadID: result.UploadID,
		Key:      result.Key,
		Parts:    parts,
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
