# MildStack S3 Phase 19 Support Matrix

Phase 19 locks the governance, ACL, and object-governance contract surface for the S3 emulator. The goal is to make the shipped behavior explicit, keep the transport naming policy reviewable, and distinguish real support from deferred AWS surface area.

## Supported Surface

### Bucket governance

| Surface | Status | Notes |
|--------|--------|-------|
| `bucket policy` | Supported | Stored as raw XML and returned copy-safely. |
| `bucket encryption` | Supported | Stored as raw XML and returned copy-safely. |
| `bucket lifecycle` | Supported | MildStack keeps the short `/lifecycle` transport name; see alias policy below. |
| `bucket cors` | Supported | Stored as raw XML and returned copy-safely. |
| `bucket acl` | Supported | Stored as raw XML; this is not an authorization engine. |
| `bucket tagging` | Supported | Stored as raw XML and returned copy-safely. |
| `bucket ownership controls` | Supported | Default response is `BucketOwnerEnforced` when no XML has been stored. |
| `public access block` | Supported | Default response blocks all public access booleans (`true`/`true`/`true`/`true`) when no XML has been stored. |
| `bucket notification` | Supported | MildStack keeps the short `/notification` transport name; see alias policy below. |
| `bucket logging` | Supported | Stored as raw XML and returned copy-safely. |
| `bucket replication` | Supported | Stored as structured XML and gated by versioning. |
| `bucket versioning` | Supported | Required for object lock and replication flows that depend on versioned buckets. |

### Object governance

| Surface | Status | Notes |
|--------|--------|-------|
| `object locking` | Supported | Persists object-lock configuration and enforces versioning and protection checks. |
| `object retention` | Supported | Stored per object and cleared when the object is deleted. |
| `object legal hold` | Supported | Stored per object and cleared when the object is deleted. |
| `object acl` | Supported | Stored as raw XML per object; default lookup returns the exemplar `AccessControlPolicy` body. |
| `object tagging` | Supported | Stored as raw XML per object; default lookup returns empty `Tagging/TagSet` XML. |

### Core object and bucket operations

The phase also keeps the existing bucket and object CRUD surface, object listing, multipart upload flow, and `get bucket location` behavior intact. These capabilities remain part of the S3 service policy and are pinned by the regression suite.

## Compatibility Aliases

MildStack intentionally keeps the short transport names for the two historical subresource families below:

| Canonical AWS name | MildStack transport name | Policy |
|--------------------|--------------------------|--------|
| `GetBucketLifecycleConfiguration` / `PutBucketLifecycleConfiguration` | `/lifecycle` | Keep the short name as the current contract and document it as the compatibility form. |
| `GetBucketNotificationConfiguration` / `PutBucketNotificationConfiguration` | `/notification` | Keep the short name as the current contract and document it as the compatibility form. |

The rest of the Phase 19 governance surfaces use explicit transport names that match the feature family directly, such as `/ownership-controls`, `/public-access-block`, `/acl`, and `/tagging`.

## Default Responses

When a surface has no stored XML yet, MildStack returns the exemplar defaults instead of failing:

- `bucket ownership controls` -> `BucketOwnerEnforced`
- `public access block` -> all public access booleans set to `true`
- `object acl` -> the exemplar `AccessControlPolicy` document
- `object tagging` -> empty `Tagging` / `TagSet`

These defaults are part of the contract and are pinned by tests so later phases cannot silently change them.

## Deferred Surface

Phase 19 does not expand into the broader AWS S3 surface. The following families remain intentionally deferred:

- Directory buckets / S3 Express
- Object Lambda
- Reporting-heavy and admin/reporting surfaces
- Any broader IAM-style authorization engine beyond the raw ACL documents already supported here

If future phases add any of those families, they should appear in a new support matrix instead of being implied by this one.
