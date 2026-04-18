# AWS-Compatible S3 Surface

Phase 21 migrates the public S3 transport boundary to AWS-style addressing while keeping the supported surface intentionally narrow. The exported contract now uses the canonical root, bucket, and object forms:

- `GET /` and `POST /` for bucket collection operations
- `/:bucket` for bucket-scoped operations
- `/:bucket/:object` for object-scoped operations
- query-driven bucket and object subresources for control-plane calls such as `?location`, `?versioning`, `?acl`, `?tagging`, `?uploads`, `?uploadId=...`, and `?partNumber=...`

The goal is not to emulate every AWS S3 family. It is to keep the current supported behavior reviewable while making the transport surface look like AWS or LocalStack instead of the old MildStack runtime namespace.

## Supported Surface

### Bucket operations

| Surface | Status | Notes |
|--------|--------|-------|
| `list buckets` | Supported | `GET /` returns the exemplar bucket inventory. |
| `create bucket` | Supported | `POST /` creates a bucket with the requested region. |
| `head bucket` | Supported | `HEAD /:bucket` returns the bucket metadata snapshot. |
| `delete bucket` | Supported | `DELETE /:bucket` deletes an empty bucket. |
| `get bucket location` | Supported | `GET /:bucket?location` returns the region as `LocationConstraint`. |
| `bucket policy` | Supported | Stored as raw XML and returned copy-safely. |
| `bucket encryption` | Supported | Stored as raw XML and returned copy-safely. |
| `bucket lifecycle` | Supported | Query-driven subresource on `/:bucket?lifecycle`. |
| `bucket CORS` | Supported | Stored as raw XML and returned copy-safely. |
| `bucket ACL` | Supported | Stored as raw XML; this is not an authorization engine. |
| `bucket tagging` | Supported | Stored as raw XML and returned copy-safely. |
| `bucket ownership controls` | Supported | Default response is `BucketOwnerEnforced` when no XML has been stored. |
| `public access block` | Supported | Default response blocks all public access booleans when no XML has been stored. |
| `bucket notification` | Supported | Query-driven subresource on `/:bucket?notification`. |
| `bucket logging` | Supported | Stored as raw XML and returned copy-safely. |
| `bucket replication` | Supported | Stored as structured XML and gated by versioning. |
| `bucket versioning` | Supported | Required for object lock and replication flows that depend on versioned buckets. |
| `object lock` | Supported | Bucket-level configuration is exposed through `/:bucket?object-lock`. |

### Object operations

| Surface | Status | Notes |
|--------|--------|-------|
| `list objects` | Supported | `GET /:bucket` returns the exemplar object listing. |
| `list objects v2` | Supported | `GET /:bucket?list-type=2` returns the V2 listing shape. |
| `get object` | Supported | `GET /:bucket/:object` returns the stored object body. |
| `head object` | Supported | `HEAD /:bucket/:object` returns object metadata without a body. |
| `put object` | Supported | `PUT /:bucket/:object` stores the object body and content type. |
| `copy object` | Supported | Uses `x-amz-copy-source` semantics and returns `CopyObjectResult`. |
| `delete object` | Supported | `DELETE /:bucket/:object` removes one object. |
| `delete objects` | Supported | `POST /:bucket?delete` removes a batch of keys. |
| `object retention` | Supported | `/:bucket/:object?retention` stores per-object retention metadata. |
| `object legal hold` | Supported | `/:bucket/:object?legal-hold` stores per-object legal hold metadata. |
| `object ACL` | Supported | `/:bucket/:object?acl` stores raw ACL XML per object. |
| `object tagging` | Supported | `/:bucket/:object?tagging` stores raw tag XML per object. |

### Multipart uploads

| Surface | Status | Notes |
|--------|--------|-------|
| `list multipart uploads` | Supported | `GET /:bucket?uploads` lists active uploads. |
| `create multipart upload` | Supported | `POST /:bucket/:object?uploads` starts an upload. |
| `upload part` | Supported | `PUT /:bucket/:object?partNumber=...&uploadId=...` stores a part. |
| `list parts` | Supported | `GET /:bucket/:object?uploadId=...` returns part metadata. |
| `complete multipart upload` | Supported | `POST /:bucket/:object?uploadId=...` assembles the final object. |
| `abort multipart upload` | Supported | `DELETE /:bucket/:object?uploadId=...` discards an in-progress upload. |

## Default Responses

When a supported surface has no stored XML yet, MildStack returns exemplar defaults instead of failing:

- `bucket ownership controls` -> `BucketOwnerEnforced`
- `public access block` -> all public access booleans set to `true`
- `object ACL` -> the exemplar `AccessControlPolicy` document
- `object tagging` -> empty `Tagging` / `TagSet`

These defaults are part of the contract and are pinned by tests so later phases cannot silently change them.

## Deferred Surface

The following AWS S3 families remain intentionally deferred and must stay out of the supported route catalog:

- Directory buckets / S3 Express, including `CreateSession`, `ListDirectoryBuckets`, `RenameObject`, and the metadata configuration/table flows
- Reporting and admin surfaces such as analytics, intelligent-tiering, inventory, and metrics
- Lower-value management features such as website hosting, accelerate, request payment, ABAC, and policy status
- Specialized data-plane / Object Lambda actions such as `GetObjectAttributes`, `GetObjectTorrent`, `RestoreObject`, `SelectObjectContent`, `UpdateObjectEncryption`, `UploadPartCopy`, and `WriteGetObjectResponse`

If future phases add any of those families, they should appear in a new support matrix instead of being implied by this one.
