package infrastructure

import "github.com/michasdev/mildstack/core/internal/application/orchestrator"

func multipartRoutes() []orchestrator.Route {
	return []orchestrator.Route{
		{
			Method: "GET",
			Path:   "/s3/buckets/:bucket/uploads",
			Name:   "s3.multipart.uploads.index",
		},
		{
			Method: "GET",
			Path:   "/s3/buckets/:bucket/uploads/:upload/parts",
			Name:   "s3.multipart.uploads.parts.index",
		},
		{
			Method: "POST",
			Path:   "/s3/buckets/:bucket/objects/:object/uploads",
			Name:   "s3.multipart.uploads.create",
		},
		{
			Method: "PUT",
			Path:   "/s3/buckets/:bucket/objects/:object/uploads/:upload/parts/:part",
			Name:   "s3.multipart.uploads.part",
		},
		{
			Method: "POST",
			Path:   "/s3/buckets/:bucket/objects/:object/uploads/:upload/complete",
			Name:   "s3.multipart.uploads.complete",
		},
		{
			Method: "DELETE",
			Path:   "/s3/buckets/:bucket/objects/:object/uploads/:upload",
			Name:   "s3.multipart.uploads.abort",
		},
	}
}
