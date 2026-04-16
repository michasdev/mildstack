package infrastructure

import "github.com/michasdev/mildstack/core/internal/application/orchestrator"

func Routes() []orchestrator.Route {
	return []orchestrator.Route{
		{
			Method: "GET",
			Path:   "/s3/buckets",
			Name:   "s3.buckets.index",
		},
		{
			Method: "GET",
			Path:   "/s3/buckets/:bucket",
			Name:   "s3.buckets.show",
		},
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
	}
}
