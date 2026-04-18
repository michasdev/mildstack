package infrastructure

func (h Handlers) GetBucketPolicy(request GetBucketPolicyRequest) (GetBucketPolicyResponse, error) {
	body, err := h.service.GetBucketPolicy(request.Bucket)
	if err != nil {
		return GetBucketPolicyResponse{}, err
	}
	return GetBucketPolicyResponse{
		Policy: BucketBodyPayload{
			Bucket: request.Bucket,
			Body:   body,
		},
	}, nil
}

func (h Handlers) PutBucketPolicy(request PutBucketPolicyRequest) (PutBucketPolicyResponse, error) {
	body, err := h.service.PutBucketPolicy(request.Bucket, request.Body)
	if err != nil {
		return PutBucketPolicyResponse{}, err
	}
	return PutBucketPolicyResponse{
		Policy: BucketBodyPayload{
			Bucket: request.Bucket,
			Body:   body,
		},
	}, nil
}

func (h Handlers) DeleteBucketPolicy(request DeleteBucketPolicyRequest) (DeleteBucketPolicyResponse, error) {
	if err := h.service.DeleteBucketPolicy(request.Bucket); err != nil {
		return DeleteBucketPolicyResponse{}, err
	}
	return DeleteBucketPolicyResponse{Deleted: true}, nil
}

func (h Handlers) GetBucketEncryption(request GetBucketEncryptionRequest) (GetBucketEncryptionResponse, error) {
	body, err := h.service.GetBucketEncryption(request.Bucket)
	if err != nil {
		return GetBucketEncryptionResponse{}, err
	}
	return GetBucketEncryptionResponse{
		Encryption: BucketBodyPayload{
			Bucket: request.Bucket,
			Body:   body,
		},
	}, nil
}

func (h Handlers) PutBucketEncryption(request PutBucketEncryptionRequest) (PutBucketEncryptionResponse, error) {
	body, err := h.service.PutBucketEncryption(request.Bucket, request.Body)
	if err != nil {
		return PutBucketEncryptionResponse{}, err
	}
	return PutBucketEncryptionResponse{
		Encryption: BucketBodyPayload{
			Bucket: request.Bucket,
			Body:   body,
		},
	}, nil
}

func (h Handlers) DeleteBucketEncryption(request DeleteBucketEncryptionRequest) (DeleteBucketEncryptionResponse, error) {
	if err := h.service.DeleteBucketEncryption(request.Bucket); err != nil {
		return DeleteBucketEncryptionResponse{}, err
	}
	return DeleteBucketEncryptionResponse{Deleted: true}, nil
}

func (h Handlers) GetBucketLifecycle(request GetBucketLifecycleRequest) (GetBucketLifecycleResponse, error) {
	body, err := h.service.GetBucketLifecycle(request.Bucket)
	if err != nil {
		return GetBucketLifecycleResponse{}, err
	}
	return GetBucketLifecycleResponse{
		Lifecycle: BucketBodyPayload{
			Bucket: request.Bucket,
			Body:   body,
		},
	}, nil
}

func (h Handlers) PutBucketLifecycle(request PutBucketLifecycleRequest) (PutBucketLifecycleResponse, error) {
	body, err := h.service.PutBucketLifecycle(request.Bucket, request.Body)
	if err != nil {
		return PutBucketLifecycleResponse{}, err
	}
	return PutBucketLifecycleResponse{
		Lifecycle: BucketBodyPayload{
			Bucket: request.Bucket,
			Body:   body,
		},
	}, nil
}

func (h Handlers) DeleteBucketLifecycle(request DeleteBucketLifecycleRequest) (DeleteBucketLifecycleResponse, error) {
	if err := h.service.DeleteBucketLifecycle(request.Bucket); err != nil {
		return DeleteBucketLifecycleResponse{}, err
	}
	return DeleteBucketLifecycleResponse{Deleted: true}, nil
}

func (h Handlers) GetBucketCORS(request GetBucketCORSRequest) (GetBucketCORSResponse, error) {
	body, err := h.service.GetBucketCORS(request.Bucket)
	if err != nil {
		return GetBucketCORSResponse{}, err
	}
	return GetBucketCORSResponse{
		CORS: BucketBodyPayload{
			Bucket: request.Bucket,
			Body:   body,
		},
	}, nil
}

func (h Handlers) PutBucketCORS(request PutBucketCORSRequest) (PutBucketCORSResponse, error) {
	body, err := h.service.PutBucketCORS(request.Bucket, request.Body)
	if err != nil {
		return PutBucketCORSResponse{}, err
	}
	return PutBucketCORSResponse{
		CORS: BucketBodyPayload{
			Bucket: request.Bucket,
			Body:   body,
		},
	}, nil
}

func (h Handlers) DeleteBucketCORS(request DeleteBucketCORSRequest) (DeleteBucketCORSResponse, error) {
	if err := h.service.DeleteBucketCORS(request.Bucket); err != nil {
		return DeleteBucketCORSResponse{}, err
	}
	return DeleteBucketCORSResponse{Deleted: true}, nil
}

func (h Handlers) GetBucketACL(request GetBucketACLRequest) (GetBucketACLResponse, error) {
	body, err := h.service.GetBucketACL(request.Bucket)
	if err != nil {
		return GetBucketACLResponse{}, err
	}
	return GetBucketACLResponse{
		ACL: BucketBodyPayload{
			Bucket: request.Bucket,
			Body:   body,
		},
	}, nil
}

func (h Handlers) PutBucketACL(request PutBucketACLRequest) (PutBucketACLResponse, error) {
	body, err := h.service.PutBucketACL(request.Bucket, request.Body)
	if err != nil {
		return PutBucketACLResponse{}, err
	}
	return PutBucketACLResponse{
		ACL: BucketBodyPayload{
			Bucket: request.Bucket,
			Body:   body,
		},
	}, nil
}

func (h Handlers) GetBucketTagging(request GetBucketTaggingRequest) (GetBucketTaggingResponse, error) {
	body, err := h.service.GetBucketTagging(request.Bucket)
	if err != nil {
		return GetBucketTaggingResponse{}, err
	}
	return GetBucketTaggingResponse{
		Tagging: BucketBodyPayload{
			Bucket: request.Bucket,
			Body:   body,
		},
	}, nil
}

func (h Handlers) PutBucketTagging(request PutBucketTaggingRequest) (PutBucketTaggingResponse, error) {
	body, err := h.service.PutBucketTagging(request.Bucket, request.Body)
	if err != nil {
		return PutBucketTaggingResponse{}, err
	}
	return PutBucketTaggingResponse{
		Tagging: BucketBodyPayload{
			Bucket: request.Bucket,
			Body:   body,
		},
	}, nil
}

func (h Handlers) DeleteBucketTagging(request DeleteBucketTaggingRequest) (DeleteBucketTaggingResponse, error) {
	if err := h.service.DeleteBucketTagging(request.Bucket); err != nil {
		return DeleteBucketTaggingResponse{}, err
	}
	return DeleteBucketTaggingResponse{Deleted: true}, nil
}
