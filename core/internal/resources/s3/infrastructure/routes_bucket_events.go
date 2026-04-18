package infrastructure

import "github.com/michasdev/mildstack/core/internal/application/orchestrator"

func bucketEventRoutes() []orchestrator.Route {
	return []orchestrator.Route{
		{
			Method: "GET",
			Path:   "/:bucket?notification",
			Name:   "s3.buckets.notification.show",
		},
		{
			Method: "PUT",
			Path:   "/:bucket?notification",
			Name:   "s3.buckets.notification.update",
		},
		{
			Method: "GET",
			Path:   "/:bucket?logging",
			Name:   "s3.buckets.logging.show",
		},
		{
			Method: "PUT",
			Path:   "/:bucket?logging",
			Name:   "s3.buckets.logging.update",
		},
		{
			Method: "GET",
			Path:   "/:bucket?replication",
			Name:   "s3.buckets.replication.show",
		},
		{
			Method: "PUT",
			Path:   "/:bucket?replication",
			Name:   "s3.buckets.replication.update",
		},
		{
			Method: "DELETE",
			Path:   "/:bucket?replication",
			Name:   "s3.buckets.replication.delete",
		},
	}
}
