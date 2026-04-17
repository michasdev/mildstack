package infrastructure

import "github.com/michasdev/mildstack/core/internal/application/orchestrator"

func objectRoutes() []orchestrator.Route {
	return []orchestrator.Route{
		{
			Method: "GET",
			Path:   "/s3/buckets/:bucket/objects",
			Name:   "s3.objects.index",
		},
		{
			Method: "GET",
			Path:   "/s3/buckets/:bucket/objects/:object",
			Name:   "s3.objects.show",
		},
		{
			Method: "HEAD",
			Path:   "/s3/buckets/:bucket/objects/:object",
			Name:   "s3.objects.head",
		},
		{
			Method: "PUT",
			Path:   "/s3/buckets/:bucket/objects/:object",
			Name:   "s3.objects.update",
		},
		{
			Method: "DELETE",
			Path:   "/s3/buckets/:bucket/objects/:object",
			Name:   "s3.objects.delete",
		},
	}
}
