package infrastructure

import "github.com/michasdev/mildstack/core/internal/application/orchestrator"

func bucketAccessRoutes() []orchestrator.Route {
	return []orchestrator.Route{
		{
			Method: "GET",
			Path:   "/s3/buckets/:bucket/ownership-controls",
			Name:   "s3.buckets.ownership-controls.show",
		},
		{
			Method: "PUT",
			Path:   "/s3/buckets/:bucket/ownership-controls",
			Name:   "s3.buckets.ownership-controls.update",
		},
		{
			Method: "DELETE",
			Path:   "/s3/buckets/:bucket/ownership-controls",
			Name:   "s3.buckets.ownership-controls.delete",
		},
		{
			Method: "GET",
			Path:   "/s3/buckets/:bucket/public-access-block",
			Name:   "s3.buckets.public-access-block.show",
		},
		{
			Method: "PUT",
			Path:   "/s3/buckets/:bucket/public-access-block",
			Name:   "s3.buckets.public-access-block.update",
		},
		{
			Method: "DELETE",
			Path:   "/s3/buckets/:bucket/public-access-block",
			Name:   "s3.buckets.public-access-block.delete",
		},
	}
}
