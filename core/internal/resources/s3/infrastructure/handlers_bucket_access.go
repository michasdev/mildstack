package infrastructure

type GetBucketOwnershipControlsRequest struct {
	Bucket string
}

type GetBucketOwnershipControlsResponse struct {
	OwnershipControls BucketBodyPayload `json:"ownership_controls"`
}

type PutBucketOwnershipControlsRequest struct {
	Bucket string
	Body   []byte
}

type PutBucketOwnershipControlsResponse struct {
	OwnershipControls BucketBodyPayload `json:"ownership_controls"`
}

type DeleteBucketOwnershipControlsRequest struct {
	Bucket string
}

type DeleteBucketOwnershipControlsResponse struct {
	Deleted bool `json:"deleted"`
}

type GetPublicAccessBlockRequest struct {
	Bucket string
}

type GetPublicAccessBlockResponse struct {
	PublicAccessBlock BucketBodyPayload `json:"public_access_block"`
}

type PutPublicAccessBlockRequest struct {
	Bucket string
	Body   []byte
}

type PutPublicAccessBlockResponse struct {
	PublicAccessBlock BucketBodyPayload `json:"public_access_block"`
}

type DeletePublicAccessBlockRequest struct {
	Bucket string
}

type DeletePublicAccessBlockResponse struct {
	Deleted bool `json:"deleted"`
}

func (h Handlers) GetBucketOwnershipControls(request GetBucketOwnershipControlsRequest) (GetBucketOwnershipControlsResponse, error) {
	body, err := h.service.GetBucketOwnershipControls(request.Bucket)
	if err != nil {
		return GetBucketOwnershipControlsResponse{}, err
	}
	return GetBucketOwnershipControlsResponse{
		OwnershipControls: BucketBodyPayload{
			Bucket: request.Bucket,
			Body:   body,
		},
	}, nil
}

func (h Handlers) PutBucketOwnershipControls(request PutBucketOwnershipControlsRequest) (PutBucketOwnershipControlsResponse, error) {
	body, err := h.service.PutBucketOwnershipControls(request.Bucket, request.Body)
	if err != nil {
		return PutBucketOwnershipControlsResponse{}, err
	}
	return PutBucketOwnershipControlsResponse{
		OwnershipControls: BucketBodyPayload{
			Bucket: request.Bucket,
			Body:   body,
		},
	}, nil
}

func (h Handlers) DeleteBucketOwnershipControls(request DeleteBucketOwnershipControlsRequest) (DeleteBucketOwnershipControlsResponse, error) {
	if err := h.service.DeleteBucketOwnershipControls(request.Bucket); err != nil {
		return DeleteBucketOwnershipControlsResponse{}, err
	}
	return DeleteBucketOwnershipControlsResponse{Deleted: true}, nil
}

func (h Handlers) GetPublicAccessBlock(request GetPublicAccessBlockRequest) (GetPublicAccessBlockResponse, error) {
	body, err := h.service.GetPublicAccessBlock(request.Bucket)
	if err != nil {
		return GetPublicAccessBlockResponse{}, err
	}
	return GetPublicAccessBlockResponse{
		PublicAccessBlock: BucketBodyPayload{
			Bucket: request.Bucket,
			Body:   body,
		},
	}, nil
}

func (h Handlers) PutPublicAccessBlock(request PutPublicAccessBlockRequest) (PutPublicAccessBlockResponse, error) {
	body, err := h.service.PutPublicAccessBlock(request.Bucket, request.Body)
	if err != nil {
		return PutPublicAccessBlockResponse{}, err
	}
	return PutPublicAccessBlockResponse{
		PublicAccessBlock: BucketBodyPayload{
			Bucket: request.Bucket,
			Body:   body,
		},
	}, nil
}

func (h Handlers) DeletePublicAccessBlock(request DeletePublicAccessBlockRequest) (DeletePublicAccessBlockResponse, error) {
	if err := h.service.DeletePublicAccessBlock(request.Bucket); err != nil {
		return DeletePublicAccessBlockResponse{}, err
	}
	return DeletePublicAccessBlockResponse{Deleted: true}, nil
}
