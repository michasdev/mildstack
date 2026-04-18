package infrastructure

import "github.com/michasdev/mildstack/core/internal/resources/s3/domain"

func (h Handlers) GetBucketVersioning(request GetBucketVersioningRequest) (GetBucketVersioningResponse, error) {
	versioning, err := h.service.GetBucketVersioning(request.Bucket)
	if err != nil {
		return GetBucketVersioningResponse{}, err
	}
	return GetBucketVersioningResponse{
		Versioning: BucketVersioningPayload{
			Bucket: versioning.Bucket,
			Status: versioning.Status,
		},
	}, nil
}

func (h Handlers) PutBucketVersioning(request PutBucketVersioningRequest) (PutBucketVersioningResponse, error) {
	versioning, err := h.service.PutBucketVersioning(request.Bucket, request.Status)
	if err != nil {
		return PutBucketVersioningResponse{}, err
	}
	return PutBucketVersioningResponse{
		Versioning: BucketVersioningPayload{
			Bucket: versioning.Bucket,
			Status: versioning.Status,
		},
	}, nil
}

func (h Handlers) ListObjectVersions(request ListObjectVersionsRequest) (ListObjectVersionsResponse, error) {
	result, err := h.service.ListObjectVersions(request.Bucket)
	if err != nil {
		return ListObjectVersionsResponse{}, err
	}

	versions := make([]VersionPayload, len(result.Versions))
	latestByKey := make(map[string]struct{}, len(result.Versions))
	for i, record := range result.Versions {
		_, isLatest := latestByKey[record.Key]
		if !isLatest {
			latestByKey[record.Key] = struct{}{}
		}
		versions[i] = VersionPayload{
			Bucket:         record.Bucket,
			Key:            record.Key,
			VersionID:      record.VersionID,
			Sequence:       record.Sequence,
			IsDeleteMarker: record.IsDeleteMarker,
			IsLatest:       !isLatest,
			Size:           record.Size,
			ContentType:    record.ContentType,
			ETag:           record.ETag,
			LastModified:   record.LastModified,
		}
	}

	return ListObjectVersionsResponse{
		Bucket:   result.Bucket,
		Versions: versions,
	}, nil
}

func objectVersionsFromDomain(records []domain.VersionRecord) []VersionPayload {
	versions := make([]VersionPayload, len(records))
	latestByKey := make(map[string]struct{}, len(records))
	for i, record := range records {
		_, seen := latestByKey[record.Key]
		if !seen {
			latestByKey[record.Key] = struct{}{}
		}
		versions[i] = VersionPayload{
			Bucket:         record.Bucket,
			Key:            record.Key,
			VersionID:      record.VersionID,
			Sequence:       record.Sequence,
			IsDeleteMarker: record.IsDeleteMarker,
			IsLatest:       !seen,
			Size:           record.Size,
			ContentType:    record.ContentType,
			ETag:           record.ETag,
			LastModified:   record.LastModified,
		}
	}
	return versions
}
