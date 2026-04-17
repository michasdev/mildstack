package infrastructure

import "github.com/michasdev/mildstack/core/internal/application/orchestrator"

func versioningRoutes() []orchestrator.Route {
	return []orchestrator.Route{
		{
			Method: "GET",
			Path:   "/s3/buckets/:bucket/versioning",
			Name:   "s3.buckets.versioning.show",
		},
		{
			Method: "PUT",
			Path:   "/s3/buckets/:bucket/versioning",
			Name:   "s3.buckets.versioning.update",
		},
		{
			Method: "GET",
			Path:   "/s3/buckets/:bucket/objects/versions",
			Name:   "s3.objects.versions",
		},
	}
}
