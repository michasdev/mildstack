package infrastructure

import "github.com/michasdev/mildstack/core/internal/application/orchestrator"

func bucketSubresourceRoutes() []orchestrator.Route {
	return []orchestrator.Route{
		{
			Method: "GET",
			Path:   "/s3/buckets/:bucket/policy",
			Name:   "s3.buckets.policy.show",
		},
		{
			Method: "PUT",
			Path:   "/s3/buckets/:bucket/policy",
			Name:   "s3.buckets.policy.update",
		},
		{
			Method: "DELETE",
			Path:   "/s3/buckets/:bucket/policy",
			Name:   "s3.buckets.policy.delete",
		},
		{
			Method: "GET",
			Path:   "/s3/buckets/:bucket/encryption",
			Name:   "s3.buckets.encryption.show",
		},
		{
			Method: "PUT",
			Path:   "/s3/buckets/:bucket/encryption",
			Name:   "s3.buckets.encryption.update",
		},
		{
			Method: "DELETE",
			Path:   "/s3/buckets/:bucket/encryption",
			Name:   "s3.buckets.encryption.delete",
		},
		{
			Method: "GET",
			Path:   "/s3/buckets/:bucket/lifecycle",
			Name:   "s3.buckets.lifecycle.show",
		},
		{
			Method: "PUT",
			Path:   "/s3/buckets/:bucket/lifecycle",
			Name:   "s3.buckets.lifecycle.update",
		},
		{
			Method: "DELETE",
			Path:   "/s3/buckets/:bucket/lifecycle",
			Name:   "s3.buckets.lifecycle.delete",
		},
		{
			Method: "GET",
			Path:   "/s3/buckets/:bucket/cors",
			Name:   "s3.buckets.cors.show",
		},
		{
			Method: "PUT",
			Path:   "/s3/buckets/:bucket/cors",
			Name:   "s3.buckets.cors.update",
		},
		{
			Method: "DELETE",
			Path:   "/s3/buckets/:bucket/cors",
			Name:   "s3.buckets.cors.delete",
		},
		{
			Method: "GET",
			Path:   "/s3/buckets/:bucket/acl",
			Name:   "s3.buckets.acl.show",
		},
		{
			Method: "PUT",
			Path:   "/s3/buckets/:bucket/acl",
			Name:   "s3.buckets.acl.update",
		},
		{
			Method: "GET",
			Path:   "/s3/buckets/:bucket/tagging",
			Name:   "s3.buckets.tagging.show",
		},
		{
			Method: "PUT",
			Path:   "/s3/buckets/:bucket/tagging",
			Name:   "s3.buckets.tagging.update",
		},
		{
			Method: "DELETE",
			Path:   "/s3/buckets/:bucket/tagging",
			Name:   "s3.buckets.tagging.delete",
		},
	}
}
