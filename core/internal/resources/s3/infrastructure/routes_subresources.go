package infrastructure

import "github.com/michasdev/mildstack/core/internal/application/orchestrator"

func bucketSubresourceRoutes() []orchestrator.Route {
	return []orchestrator.Route{
		{
			Method: "GET",
			Path:   "/:bucket?location",
			Name:   "s3.buckets.location.show",
		},
		{
			Method: "GET",
			Path:   "/:bucket?policy",
			Name:   "s3.buckets.policy.show",
		},
		{
			Method: "PUT",
			Path:   "/:bucket?policy",
			Name:   "s3.buckets.policy.update",
		},
		{
			Method: "DELETE",
			Path:   "/:bucket?policy",
			Name:   "s3.buckets.policy.delete",
		},
		{
			Method: "GET",
			Path:   "/:bucket?encryption",
			Name:   "s3.buckets.encryption.show",
		},
		{
			Method: "PUT",
			Path:   "/:bucket?encryption",
			Name:   "s3.buckets.encryption.update",
		},
		{
			Method: "DELETE",
			Path:   "/:bucket?encryption",
			Name:   "s3.buckets.encryption.delete",
		},
		{
			Method: "GET",
			Path:   "/:bucket?lifecycle",
			Name:   "s3.buckets.lifecycle.show",
		},
		{
			Method: "PUT",
			Path:   "/:bucket?lifecycle",
			Name:   "s3.buckets.lifecycle.update",
		},
		{
			Method: "DELETE",
			Path:   "/:bucket?lifecycle",
			Name:   "s3.buckets.lifecycle.delete",
		},
		{
			Method: "GET",
			Path:   "/:bucket?cors",
			Name:   "s3.buckets.cors.show",
		},
		{
			Method: "PUT",
			Path:   "/:bucket?cors",
			Name:   "s3.buckets.cors.update",
		},
		{
			Method: "DELETE",
			Path:   "/:bucket?cors",
			Name:   "s3.buckets.cors.delete",
		},
		{
			Method: "GET",
			Path:   "/:bucket?acl",
			Name:   "s3.buckets.acl.show",
		},
		{
			Method: "PUT",
			Path:   "/:bucket?acl",
			Name:   "s3.buckets.acl.update",
		},
		{
			Method: "GET",
			Path:   "/:bucket?tagging",
			Name:   "s3.buckets.tagging.show",
		},
		{
			Method: "PUT",
			Path:   "/:bucket?tagging",
			Name:   "s3.buckets.tagging.update",
		},
		{
			Method: "DELETE",
			Path:   "/:bucket?tagging",
			Name:   "s3.buckets.tagging.delete",
		},
	}
}
