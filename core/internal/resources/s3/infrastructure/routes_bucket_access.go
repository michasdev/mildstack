package infrastructure

import "github.com/michasdev/mildstack/core/internal/application/orchestrator"

func bucketAccessRoutes() []orchestrator.Route {
	return []orchestrator.Route{
		{
			Method: "GET",
			Path:   "/:bucket?ownershipControls",
			Name:   "s3.buckets.ownership-controls.show",
		},
		{
			Method: "PUT",
			Path:   "/:bucket?ownershipControls",
			Name:   "s3.buckets.ownership-controls.update",
		},
		{
			Method: "DELETE",
			Path:   "/:bucket?ownershipControls",
			Name:   "s3.buckets.ownership-controls.delete",
		},
		{
			Method: "GET",
			Path:   "/:bucket?publicAccessBlock",
			Name:   "s3.buckets.public-access-block.show",
		},
		{
			Method: "PUT",
			Path:   "/:bucket?publicAccessBlock",
			Name:   "s3.buckets.public-access-block.update",
		},
		{
			Method: "DELETE",
			Path:   "/:bucket?publicAccessBlock",
			Name:   "s3.buckets.public-access-block.delete",
		},
	}
}
