package infrastructure

func (h Handlers) GetBucketNotification(request GetBucketNotificationRequest) (GetBucketNotificationResponse, error) {
	body, err := h.service.GetBucketNotification(request.Bucket)
	if err != nil {
		return GetBucketNotificationResponse{}, err
	}
	return GetBucketNotificationResponse{
		Notification: BucketBodyPayload{
			Bucket: request.Bucket,
			Body:   body,
		},
	}, nil
}

func (h Handlers) PutBucketNotification(request PutBucketNotificationRequest) (PutBucketNotificationResponse, error) {
	body, err := h.service.PutBucketNotification(request.Bucket, request.Body)
	if err != nil {
		return PutBucketNotificationResponse{}, err
	}
	return PutBucketNotificationResponse{
		Notification: BucketBodyPayload{
			Bucket: request.Bucket,
			Body:   body,
		},
	}, nil
}

func (h Handlers) GetBucketLogging(request GetBucketLoggingRequest) (GetBucketLoggingResponse, error) {
	body, err := h.service.GetBucketLogging(request.Bucket)
	if err != nil {
		return GetBucketLoggingResponse{}, err
	}
	return GetBucketLoggingResponse{
		Logging: BucketBodyPayload{
			Bucket: request.Bucket,
			Body:   body,
		},
	}, nil
}

func (h Handlers) PutBucketLogging(request PutBucketLoggingRequest) (PutBucketLoggingResponse, error) {
	body, err := h.service.PutBucketLogging(request.Bucket, request.Body)
	if err != nil {
		return PutBucketLoggingResponse{}, err
	}
	return PutBucketLoggingResponse{
		Logging: BucketBodyPayload{
			Bucket: request.Bucket,
			Body:   body,
		},
	}, nil
}

func (h Handlers) GetBucketReplication(request GetBucketReplicationRequest) (GetBucketReplicationResponse, error) {
	body, err := h.service.GetBucketReplication(request.Bucket)
	if err != nil {
		return GetBucketReplicationResponse{}, err
	}
	return GetBucketReplicationResponse{
		Replication: BucketBodyPayload{
			Bucket: request.Bucket,
			Body:   body,
		},
	}, nil
}

func (h Handlers) PutBucketReplication(request PutBucketReplicationRequest) (PutBucketReplicationResponse, error) {
	body, err := h.service.PutBucketReplication(request.Bucket, request.Body)
	if err != nil {
		return PutBucketReplicationResponse{}, err
	}
	return PutBucketReplicationResponse{
		Replication: BucketBodyPayload{
			Bucket: request.Bucket,
			Body:   body,
		},
	}, nil
}

func (h Handlers) DeleteBucketReplication(request DeleteBucketReplicationRequest) (DeleteBucketReplicationResponse, error) {
	if err := h.service.DeleteBucketReplication(request.Bucket); err != nil {
		return DeleteBucketReplicationResponse{}, err
	}
	return DeleteBucketReplicationResponse{Deleted: true}, nil
}

type GetBucketNotificationRequest struct {
	Bucket string
}

type GetBucketNotificationResponse struct {
	Notification BucketBodyPayload `json:"notification"`
}

type PutBucketNotificationRequest struct {
	Bucket string
	Body   []byte
}

type PutBucketNotificationResponse struct {
	Notification BucketBodyPayload `json:"notification"`
}

type GetBucketLoggingRequest struct {
	Bucket string
}

type GetBucketLoggingResponse struct {
	Logging BucketBodyPayload `json:"logging"`
}

type PutBucketLoggingRequest struct {
	Bucket string
	Body   []byte
}

type PutBucketLoggingResponse struct {
	Logging BucketBodyPayload `json:"logging"`
}

type GetBucketReplicationRequest struct {
	Bucket string
}

type GetBucketReplicationResponse struct {
	Replication BucketBodyPayload `json:"replication"`
}

type PutBucketReplicationRequest struct {
	Bucket string
	Body   []byte
}

type PutBucketReplicationResponse struct {
	Replication BucketBodyPayload `json:"replication"`
}

type DeleteBucketReplicationRequest struct {
	Bucket string
}

type DeleteBucketReplicationResponse struct {
	Deleted bool `json:"deleted"`
}
