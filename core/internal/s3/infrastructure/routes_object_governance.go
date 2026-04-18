package infrastructure

import "github.com/michasdev/mildstack/core/internal/application/orchestrator"

func objectGovernanceRoutes() []orchestrator.Route {
	return []orchestrator.Route{
		{
			Method: "GET",
			Path:   "/:bucket/:object?acl",
			Name:   "s3.objects.acl.show",
		},
		{
			Method: "PUT",
			Path:   "/:bucket/:object?acl",
			Name:   "s3.objects.acl.update",
		},
		{
			Method: "GET",
			Path:   "/:bucket/:object?tagging",
			Name:   "s3.objects.tagging.show",
		},
		{
			Method: "PUT",
			Path:   "/:bucket/:object?tagging",
			Name:   "s3.objects.tagging.update",
		},
		{
			Method: "DELETE",
			Path:   "/:bucket/:object?tagging",
			Name:   "s3.objects.tagging.delete",
		},
	}
}
