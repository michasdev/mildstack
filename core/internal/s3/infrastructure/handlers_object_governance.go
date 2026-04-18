package infrastructure

type ObjectBodyPayload struct {
	Bucket string `json:"bucket"`
	Key    string `json:"key"`
	Body   []byte `json:"body,omitempty"`
}

type GetObjectAclRequest struct {
	Bucket string
	Key    string
}

type GetObjectAclResponse struct {
	ACL ObjectBodyPayload `json:"acl"`
}

type PutObjectAclRequest struct {
	Bucket string
	Key    string
	Body   []byte
}

type PutObjectAclResponse struct {
	ACL ObjectBodyPayload `json:"acl"`
}

type GetObjectTaggingRequest struct {
	Bucket string
	Key    string
}

type GetObjectTaggingResponse struct {
	Tagging ObjectBodyPayload `json:"tagging"`
}

type PutObjectTaggingRequest struct {
	Bucket string
	Key    string
	Body   []byte
}

type PutObjectTaggingResponse struct {
	Tagging ObjectBodyPayload `json:"tagging"`
}

type DeleteObjectTaggingRequest struct {
	Bucket string
	Key    string
}

type DeleteObjectTaggingResponse struct {
	Deleted bool `json:"deleted"`
}

func (h Handlers) GetObjectAcl(request GetObjectAclRequest) (GetObjectAclResponse, error) {
	body, err := h.service.GetObjectAcl(request.Bucket, request.Key)
	if err != nil {
		return GetObjectAclResponse{}, err
	}
	return GetObjectAclResponse{
		ACL: ObjectBodyPayload{
			Bucket: request.Bucket,
			Key:    request.Key,
			Body:   body,
		},
	}, nil
}

func (h Handlers) PutObjectAcl(request PutObjectAclRequest) (PutObjectAclResponse, error) {
	body, err := h.service.PutObjectAcl(request.Bucket, request.Key, request.Body)
	if err != nil {
		return PutObjectAclResponse{}, err
	}
	return PutObjectAclResponse{
		ACL: ObjectBodyPayload{
			Bucket: request.Bucket,
			Key:    request.Key,
			Body:   body,
		},
	}, nil
}

func (h Handlers) GetObjectTagging(request GetObjectTaggingRequest) (GetObjectTaggingResponse, error) {
	body, err := h.service.GetObjectTagging(request.Bucket, request.Key)
	if err != nil {
		return GetObjectTaggingResponse{}, err
	}
	return GetObjectTaggingResponse{
		Tagging: ObjectBodyPayload{
			Bucket: request.Bucket,
			Key:    request.Key,
			Body:   body,
		},
	}, nil
}

func (h Handlers) PutObjectTagging(request PutObjectTaggingRequest) (PutObjectTaggingResponse, error) {
	body, err := h.service.PutObjectTagging(request.Bucket, request.Key, request.Body)
	if err != nil {
		return PutObjectTaggingResponse{}, err
	}
	return PutObjectTaggingResponse{
		Tagging: ObjectBodyPayload{
			Bucket: request.Bucket,
			Key:    request.Key,
			Body:   body,
		},
	}, nil
}

func (h Handlers) DeleteObjectTagging(request DeleteObjectTaggingRequest) (DeleteObjectTaggingResponse, error) {
	if err := h.service.DeleteObjectTagging(request.Bucket, request.Key); err != nil {
		return DeleteObjectTaggingResponse{}, err
	}
	return DeleteObjectTaggingResponse{Deleted: true}, nil
}
