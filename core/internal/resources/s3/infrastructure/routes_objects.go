package infrastructure

import "github.com/michasdev/mildstack/core/internal/application/orchestrator"

func objectRoutes() []orchestrator.Route {
	return []orchestrator.Route{
		{
			Method: "GET",
			Path:   "/:bucket",
			Name:   "s3.objects.list-v1",
		},
		{
			Method: "GET",
			Path:   "/:bucket?list-type=2",
			Name:   "s3.objects.list-v2",
		},
		{
			Method: "POST",
			Path:   "/:bucket?delete",
			Name:   "s3.objects.delete-batch",
		},
		{
			Method: "GET",
			Path:   "/:bucket/:object",
			Name:   "s3.objects.show",
		},
		{
			Method: "HEAD",
			Path:   "/:bucket/:object",
			Name:   "s3.objects.head",
		},
		{
			Method: "PUT",
			Path:   "/:bucket/:object",
			Name:   "s3.objects.update",
		},
		{
			Method: "DELETE",
			Path:   "/:bucket/:object",
			Name:   "s3.objects.delete",
		},
	}
}
