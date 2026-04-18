package infrastructure

import "github.com/michasdev/mildstack/core/internal/application/orchestrator"

func bucketRoutes() []orchestrator.Route {
	return []orchestrator.Route{
		{
			Method: "GET",
			Path:   "/",
			Name:   "s3.buckets.index",
		},
		{
			Method: "POST",
			Path:   "/",
			Name:   "s3.buckets.create",
		},
		{
			Method: "HEAD",
			Path:   "/:bucket",
			Name:   "s3.buckets.head",
		},
		{
			Method: "DELETE",
			Path:   "/:bucket",
			Name:   "s3.buckets.delete",
		},
	}
}
