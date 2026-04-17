package infrastructure

import "github.com/michasdev/mildstack/core/internal/application/orchestrator"

func bucketRoutes() []orchestrator.Route {
	return []orchestrator.Route{
		{
			Method: "GET",
			Path:   "/s3/buckets",
			Name:   "s3.buckets.index",
		},
		{
			Method: "POST",
			Path:   "/s3/buckets",
			Name:   "s3.buckets.create",
		},
		{
			Method: "HEAD",
			Path:   "/s3/buckets/:bucket",
			Name:   "s3.buckets.head",
		},
		{
			Method: "DELETE",
			Path:   "/s3/buckets/:bucket",
			Name:   "s3.buckets.delete",
		},
	}
}
