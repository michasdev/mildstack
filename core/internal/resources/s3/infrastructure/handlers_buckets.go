package infrastructure

func (h Handlers) ListBuckets() ListBucketsResponse {
	buckets := h.service.ListBuckets()
	response := ListBucketsResponse{
		Buckets: bucketPayloadsFromDomain(buckets),
	}
	return response
}

func (h Handlers) CreateBucket(request CreateBucketRequest) (CreateBucketResponse, error) {
	bucket, err := h.service.CreateBucket(request.Name, request.Region)
	if err != nil {
		return CreateBucketResponse{}, err
	}
	return CreateBucketResponse{
		Bucket: bucketPayloadFromDomain(bucket),
	}, nil
}

func (h Handlers) HeadBucket(request HeadBucketRequest) (HeadBucketResponse, error) {
	bucket, err := h.service.HeadBucket(request.Name)
	if err != nil {
		return HeadBucketResponse{}, err
	}
	return HeadBucketResponse{
		Bucket: bucketPayloadFromDomain(bucket),
	}, nil
}

func (h Handlers) GetBucketLocation(request GetBucketLocationRequest) (GetBucketLocationResponse, error) {
	location, err := h.service.GetBucketLocation(request.Bucket)
	if err != nil {
		return GetBucketLocationResponse{}, err
	}
	return GetBucketLocationResponse{
		Location: bucketLocationPayloadFromRegion(request.Bucket, location),
	}, nil
}

func (h Handlers) DeleteBucket(request DeleteBucketRequest) (DeleteBucketResponse, error) {
	if err := h.service.DeleteBucket(request.Name); err != nil {
		return DeleteBucketResponse{}, err
	}
	return DeleteBucketResponse{Deleted: true}, nil
}
