package infrastructure

import "github.com/michasdev/mildstack/core/internal/application/orchestrator"

func multipartRoutes() []orchestrator.Route {
	return []orchestrator.Route{
		{
			Method: "GET",
			Path:   "/:bucket?uploads",
			Name:   "s3.multipart.uploads.index",
		},
		{
			Method: "GET",
			Path:   "/:bucket/:object?uploadId=:upload",
			Name:   "s3.multipart.uploads.parts.index",
		},
		{
			Method: "POST",
			Path:   "/:bucket/:object?uploads",
			Name:   "s3.multipart.uploads.create",
		},
		{
			Method: "PUT",
			Path:   "/:bucket/:object?partNumber=:part&uploadId=:upload",
			Name:   "s3.multipart.uploads.part",
		},
		{
			Method: "POST",
			Path:   "/:bucket/:object?uploadId=:upload",
			Name:   "s3.multipart.uploads.complete",
		},
		{
			Method: "DELETE",
			Path:   "/:bucket/:object?uploadId=:upload",
			Name:   "s3.multipart.uploads.abort",
		},
	}
}
