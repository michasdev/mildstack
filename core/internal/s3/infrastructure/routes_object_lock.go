package infrastructure

import "github.com/michasdev/mildstack/core/internal/application/orchestrator"

func objectLockRoutes() []orchestrator.Route {
	return []orchestrator.Route{
		{
			Method: "GET",
			Path:   "/s3/buckets/:bucket/object-lock",
			Name:   "s3.buckets.object-lock.show",
		},
		{
			Method: "PUT",
			Path:   "/s3/buckets/:bucket/object-lock",
			Name:   "s3.buckets.object-lock.update",
		},
		{
			Method: "GET",
			Path:   "/s3/buckets/:bucket/objects/:object/retention",
			Name:   "s3.objects.retention.show",
		},
		{
			Method: "PUT",
			Path:   "/s3/buckets/:bucket/objects/:object/retention",
			Name:   "s3.objects.retention.update",
		},
		{
			Method: "GET",
			Path:   "/s3/buckets/:bucket/objects/:object/legal-hold",
			Name:   "s3.objects.legal-hold.show",
		},
		{
			Method: "PUT",
			Path:   "/s3/buckets/:bucket/objects/:object/legal-hold",
			Name:   "s3.objects.legal-hold.update",
		},
	}
}
