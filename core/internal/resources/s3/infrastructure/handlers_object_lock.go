package infrastructure

type ObjectLockConfigurationPayload struct {
	Bucket string `json:"bucket"`
	Body   []byte `json:"body,omitempty"`
}

type ObjectRetentionPayload struct {
	Bucket string `json:"bucket"`
	Key    string `json:"key"`
	Body   []byte `json:"body,omitempty"`
}

type ObjectLegalHoldPayload struct {
	Bucket string `json:"bucket"`
	Key    string `json:"key"`
	Body   []byte `json:"body,omitempty"`
}

type GetObjectLockConfigurationRequest struct {
	Bucket string
}

type GetObjectLockConfigurationResponse struct {
	ObjectLock ObjectLockConfigurationPayload `json:"object_lock"`
}

type PutObjectLockConfigurationRequest struct {
	Bucket string
	Body   []byte
}

type PutObjectLockConfigurationResponse struct {
	ObjectLock ObjectLockConfigurationPayload `json:"object_lock"`
}

type GetObjectRetentionRequest struct {
	Bucket string
	Key    string
}

type GetObjectRetentionResponse struct {
	Retention ObjectRetentionPayload `json:"retention"`
}

type PutObjectRetentionRequest struct {
	Bucket string
	Key    string
	Body   []byte
}

type PutObjectRetentionResponse struct {
	Retention ObjectRetentionPayload `json:"retention"`
}

type GetObjectLegalHoldRequest struct {
	Bucket string
	Key    string
}

type GetObjectLegalHoldResponse struct {
	LegalHold ObjectLegalHoldPayload `json:"legal_hold"`
}

type PutObjectLegalHoldRequest struct {
	Bucket string
	Key    string
	Body   []byte
}

type PutObjectLegalHoldResponse struct {
	LegalHold ObjectLegalHoldPayload `json:"legal_hold"`
}

func (h Handlers) GetObjectLockConfiguration(request GetObjectLockConfigurationRequest) (GetObjectLockConfigurationResponse, error) {
	body, err := h.service.GetObjectLockConfiguration(request.Bucket)
	if err != nil {
		return GetObjectLockConfigurationResponse{}, err
	}
	return GetObjectLockConfigurationResponse{
		ObjectLock: ObjectLockConfigurationPayload{
			Bucket: request.Bucket,
			Body:   body,
		},
	}, nil
}

func (h Handlers) PutObjectLockConfiguration(request PutObjectLockConfigurationRequest) (PutObjectLockConfigurationResponse, error) {
	body, err := h.service.PutObjectLockConfiguration(request.Bucket, request.Body)
	if err != nil {
		return PutObjectLockConfigurationResponse{}, err
	}
	return PutObjectLockConfigurationResponse{
		ObjectLock: ObjectLockConfigurationPayload{
			Bucket: request.Bucket,
			Body:   body,
		},
	}, nil
}

func (h Handlers) GetObjectRetention(request GetObjectRetentionRequest) (GetObjectRetentionResponse, error) {
	body, err := h.service.GetObjectRetention(request.Bucket, request.Key)
	if err != nil {
		return GetObjectRetentionResponse{}, err
	}
	return GetObjectRetentionResponse{
		Retention: ObjectRetentionPayload{
			Bucket: request.Bucket,
			Key:    request.Key,
			Body:   body,
		},
	}, nil
}

func (h Handlers) PutObjectRetention(request PutObjectRetentionRequest) (PutObjectRetentionResponse, error) {
	body, err := h.service.PutObjectRetention(request.Bucket, request.Key, request.Body)
	if err != nil {
		return PutObjectRetentionResponse{}, err
	}
	return PutObjectRetentionResponse{
		Retention: ObjectRetentionPayload{
			Bucket: request.Bucket,
			Key:    request.Key,
			Body:   body,
		},
	}, nil
}

func (h Handlers) GetObjectLegalHold(request GetObjectLegalHoldRequest) (GetObjectLegalHoldResponse, error) {
	body, err := h.service.GetObjectLegalHold(request.Bucket, request.Key)
	if err != nil {
		return GetObjectLegalHoldResponse{}, err
	}
	return GetObjectLegalHoldResponse{
		LegalHold: ObjectLegalHoldPayload{
			Bucket: request.Bucket,
			Key:    request.Key,
			Body:   body,
		},
	}, nil
}

func (h Handlers) PutObjectLegalHold(request PutObjectLegalHoldRequest) (PutObjectLegalHoldResponse, error) {
	body, err := h.service.PutObjectLegalHold(request.Bucket, request.Key, request.Body)
	if err != nil {
		return PutObjectLegalHoldResponse{}, err
	}
	return PutObjectLegalHoldResponse{
		LegalHold: ObjectLegalHoldPayload{
			Bucket: request.Bucket,
			Key:    request.Key,
			Body:   body,
		},
	}, nil
}
