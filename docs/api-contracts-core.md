# Core API Contracts

**Part ID:** `core`  
**Audience:** Core/backend contributors and integration clients  
**Base Runtime Prefix:** `/api/v1`

## 1. Runtime and Service Catalog Endpoints

These endpoints are served by `core/internal/delivery/http/router.go`.

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/v1/runtime/health` | Liveness probe (`status=ok`) |
| GET | `/api/v1/runtime/ready` | Readiness probe (`ready` when at least one port is active) |
| GET | `/api/v1/runtime/info` | Service metadata + active port snapshot |
| GET | `/api/v1/runtime/services` | Service catalog with route counts |
| GET | `/api/v1/runtime/services/:service` | Detailed route list for one registered service |

## 2. Internal Service Route Catalog (`/api/v1/runtime/services/...`)

Service routes are registered through `orchestrator.RouteRegistrar` and normalized under:

- `/api/v1/runtime/services/<service>/...`

## S3 route catalog (registered)

Representative entries:

- `GET /api/v1/runtime/services/s3/...` bucket index and object list
- `POST /api/v1/runtime/services/s3/...` bucket/object/multipart create operations
- `PUT /api/v1/runtime/services/s3/...` update operations (object write, controls, lock, tagging, etc.)
- `DELETE /api/v1/runtime/services/s3/...` delete operations
- `HEAD /api/v1/runtime/services/s3/...` metadata checks

Catalog source is split across:

- `routes_buckets.go`
- `routes_subresources.go`
- `routes_bucket_access.go`
- `routes_bucket_events.go`
- `routes_versions.go`
- `routes_object_lock.go`
- `routes_object_governance.go`
- `routes_objects.go`
- `routes_multipart.go`

## DynamoDB route catalog (registered)

- `GET /api/v1/runtime/services/dynamodb/tables`
- `POST /api/v1/runtime/services/dynamodb/tables`
- `GET /api/v1/runtime/services/dynamodb/tables/:table/items/:item`
- `PUT /api/v1/runtime/services/dynamodb/tables/:table/items/:item`
- `DELETE /api/v1/runtime/services/dynamodb/tables/:table/items/:item`

## SQS route catalog (registered)

- `GET /api/v1/runtime/services/sqs/queues`
- `POST /api/v1/runtime/services/sqs/queues`
- `GET /api/v1/runtime/services/sqs/queues/:queue`
- `DELETE /api/v1/runtime/services/sqs/queues/:queue`
- `GET /api/v1/runtime/services/sqs/queues/:queue/messages`
- `POST /api/v1/runtime/services/sqs/queues/:queue/messages`
- `DELETE /api/v1/runtime/services/sqs/queues/:queue/messages/:receiptHandle`

## 3. Native S3-Compatible Surface

S3 native adapter owns non-`/api` paths and dispatches by method + path + query.

## Bucket-level surfaces

- `GET /` list buckets
- `PUT /:bucket` create bucket
- `HEAD /:bucket` head bucket
- `DELETE /:bucket` delete bucket
- `GET /:bucket?location`
- `GET/PUT/DELETE /:bucket?policy`
- `GET/PUT/DELETE /:bucket?encryption`
- `GET/PUT/DELETE /:bucket?lifecycle`
- `GET/PUT/DELETE /:bucket?cors`
- `GET/PUT /:bucket?acl`
- `GET/PUT/DELETE /:bucket?tagging`
- `GET/PUT /:bucket?notification`
- `GET/PUT /:bucket?logging`
- `GET/PUT/DELETE /:bucket?replication`
- `GET/PUT /:bucket?versioning`
- `GET /:bucket?versions`
- `GET/PUT/DELETE /:bucket?ownershipControls`
- `GET/PUT/DELETE /:bucket?publicAccessBlock`
- `GET/PUT /:bucket?object-lock`

## Object-level surfaces

- `GET /:bucket/:object`
- `HEAD /:bucket/:object`
- `PUT /:bucket/:object`
- `DELETE /:bucket/:object`
- `POST /:bucket?delete` (batch delete)
- `PUT /:bucket/:object` with `x-amz-copy-source` (copy object)
- `GET/PUT /:bucket/:object?retention`
- `GET/PUT /:bucket/:object?legal-hold`
- `GET/PUT /:bucket/:object?acl`
- `GET/PUT/DELETE /:bucket/:object?tagging`

## Multipart surfaces

- `GET /:bucket?uploads`
- `POST /:bucket/:object?uploads`
- `PUT /:bucket/:object?partNumber=:part&uploadId=:upload`
- `GET /:bucket/:object?uploadId=:upload` (list parts)
- `POST /:bucket/:object?uploadId=:upload` (complete)
- `DELETE /:bucket/:object?uploadId=:upload` (abort)

## 4. Native DynamoDB-Compatible Surface

DynamoDB native adapter accepts AWS JSON API style:

- `POST /`
- `Content-Type: application/x-amz-json-1.0`
- `X-Amz-Target: DynamoDB_20120810.<Action>`

Supported target actions include:

- `ListTables`
- `CreateTable`
- `DescribeTable`
- `DeleteTable`
- `GetItem`
- `PutItem`
- `UpdateItem`
- `Query`
- `Scan`
- `DeleteItem`
- `BatchGetItem`
- `BatchWriteItem`
- `TransactGetItems`
- `TransactWriteItems`
- `UpdateTimeToLive`
- `DescribeTimeToLive`

Explicitly marked unsupported/deferred in registry:

- `UpdateTable`

## 5. Native SQS-Compatible Surface

SQS adapter supports:

- Query-style actions (`Action=...&Version=2012-11-05`)
- Target-style actions (`X-Amz-Target`) when applicable
- Root scope (`/`) and queue scope (`/<accountId>/<queueName>`)

Supported action families:

- Queue lifecycle (`CreateQueue`, `DeleteQueue`, `GetQueueUrl`, `ListQueues`, `PurgeQueue`, attributes)
- Queue governance (`TagQueue`, `UntagQueue`, `AddPermission`, `RemovePermission`, `ListQueueTags`)
- Redrive (`ListDeadLetterSourceQueues`, `StartMessageMoveTask`, `CancelMessageMoveTask`, `ListMessageMoveTasks`)
- Message surface (`SendMessage`, `ReceiveMessage`, `DeleteMessage`, visibility and batch variants)

Action catalog source:

- `core/internal/resources/sqs/contracts/catalog.go`

Compatibility routing source:

- `core/internal/delivery/http/sqs_native_contract.go`
- `core/internal/delivery/http/sqs_native_registry.go`

## 6. Error and Compatibility Behavior

- Native adapters intentionally fail closed when request ownership is unclear or action is unsupported.
- Error prefixing follows service policy (`s3`, `dynamodb`, `sqs`) via orchestrator conventions.
- Runtime catalog endpoints can return server errors when service routes are not registered.

## 7. Integration Notes for Desktop

Desktop IPC handlers call local endpoints and expect:

- standard HTTP status semantics
- DynamoDB target-based operation responses
- S3 XML compatibility for AWS SDK expectations
- SQS action semantics consistent with AWS query/JSON conventions

