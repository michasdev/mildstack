package infrastructure

import "github.com/michasdev/mildstack/core/internal/application/orchestrator"

func bucketEventRoutes() []orchestrator.Route {
	return []orchestrator.Route{
		{
			Method: "GET",
			Path:   "/s3/buckets/:bucket/notification",
			Name:   "s3.buckets.notification.show",
		},
		{
			Method: "PUT",
			Path:   "/s3/buckets/:bucket/notification",
			Name:   "s3.buckets.notification.update",
		},
		{
			Method: "GET",
			Path:   "/s3/buckets/:bucket/logging",
			Name:   "s3.buckets.logging.show",
		},
		{
			Method: "PUT",
			Path:   "/s3/buckets/:bucket/logging",
			Name:   "s3.buckets.logging.update",
		},
		{
			Method: "GET",
			Path:   "/s3/buckets/:bucket/replication",
			Name:   "s3.buckets.replication.show",
		},
		{
			Method: "PUT",
			Path:   "/s3/buckets/:bucket/replication",
			Name:   "s3.buckets.replication.update",
		},
		{
			Method: "DELETE",
			Path:   "/s3/buckets/:bucket/replication",
			Name:   "s3.buckets.replication.delete",
		},
	}
}
