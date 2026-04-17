package infrastructure

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
