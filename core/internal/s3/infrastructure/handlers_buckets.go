package infrastructure

func (h Handlers) ListBuckets() ListBucketsResponse {
	buckets := h.service.ListBuckets()
	response := ListBucketsResponse{
		Buckets: make([]BucketPayload, len(buckets)),
	}
	for i, bucket := range buckets {
		response.Buckets[i] = BucketPayload{
			Name:      bucket.Name,
			Region:    bucket.Region,
			CreatedAt: bucket.CreatedAt,
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
			Name:      bucket.Name,
			Region:    bucket.Region,
			CreatedAt: bucket.CreatedAt,
		},
	}, nil
}

func (h Handlers) HeadBucket(request HeadBucketRequest) (HeadBucketResponse, error) {
	bucket, err := h.service.HeadBucket(request.Name)
	if err != nil {
		return HeadBucketResponse{}, err
	}
	return HeadBucketResponse{
		Bucket: BucketPayload{
			Name:      bucket.Name,
			Region:    bucket.Region,
			CreatedAt: bucket.CreatedAt,
		},
	}, nil
}

func (h Handlers) DeleteBucket(request DeleteBucketRequest) (DeleteBucketResponse, error) {
	if err := h.service.DeleteBucket(request.Name); err != nil {
		return DeleteBucketResponse{}, err
	}
	return DeleteBucketResponse{Deleted: true}, nil
}
