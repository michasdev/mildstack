# Core Data Models and Persistence

**Part ID:** `core`

## 1. Persistence Topology

Core persistence is instance-scoped and service-specific:

- Global runtime folders rooted at `~/.mildstack/`
- Service data under `~/.mildstack/instances/<instanceId>/<service>/...`

Helper authority:

- `core/internal/resources/instancepath`

## 2. Runtime/Instance Registry Models (CLI Storage)

CLI storage keeps instance records in JSON files under:

- `instances/active/`
- `instances/saved/`

Record shape (simplified):

- `instanceId`
- `port`
- `pid`
- `status` (`running`, `not_started`, `errored`)
- `error` (optional)

## 3. S3 Data Model

S3 state domain (`core/internal/resources/s3/domain/state.go`) includes:

- Buckets
- Objects
- Versioning state
- Version history
- Bucket control-plane payloads (policy/encryption/lifecycle/CORS/ACL/tagging/ownership/public-access/notification/logging/replication/object-lock)
- Object governance payloads (ACL, tagging, retention, legal hold)

S3 object shape (important fields):

- `bucket`, `key`
- `body` or `payload_ref`
- `size`, `content_type`, `etag`, `last_modified`
- `metadata`, `preserved_headers`

Persistence backend:

- Filesystem repository
- Canonical state file: `state.json`
- Payload indirection and validation for large object bodies

Key invariant examples:

- object must reference existing bucket
- object must have content type
- object must have body or payload reference
- versioning and object-lock structures must be internally consistent

## 4. DynamoDB Data Model

DynamoDB domain (`core/internal/resources/dynamodb/domain/state.go`) includes:

- `State.Service`
- `Tables[]`
- `Items[]`

Table model includes:

- key schema (`partition_key`, optional `sort_key`)
- billing mode
- attribute definitions
- global/local secondary indexes
- lifecycle status (`CREATING`, `ACTIVE`, `DELETING`)
- lifecycle timestamps

Item model includes:

- `table`, `key`
- attribute map with typed `AttributeValue` (string/number/bool/null/map/list)

Persistence backend:

- SQLite (`state.db`)
- Main tables:
  - `dynamodb_meta`
  - `dynamodb_tables`
  - `dynamodb_items`

Repository behavior:

- schema bootstrap and migration-safe column checks
- full-state load/save with normalized state validation

## 5. SQS Data Model

SQS domain (`core/internal/resources/sqs/domain/state.go`) includes:

- `Queues[]`
- `Messages[]`
- `RecoveryMetadata`
- `QueueTags`
- `QueuePermissions`
- `MoveTasks`

Queue model includes:

- `name`, `url`
- `attributes`
- lifecycle timestamps (`created`, `updated`, `deleted`, `purged`)
- dead-letter recovery policy metadata

Message model includes:

- delivery identity (`message_id`, queue, receipt keys)
- content (`body`, attributes, metadata, tags)
- ordering/batch metadata (`message_group_id`, sequence, batch fields)
- redrive/dead-letter metadata
- timing (`sent`, `available`, `received`)
- recovery attempts/details

Persistence backend:

- SQLite (`state.db`)
- Main tables:
  - `sqs_meta`
  - `sqs_queues`
  - `sqs_messages`
  - `sqs_recovery_metadata`
  - `sqs_queue_governance`

## 6. Shared In-Memory Runtime State

`runtime.MemoryStateHook` maintains service snapshots:

- mutex-protected map
- deep-ish clone semantics through reflection for maps/slices/pointers/arrays/interfaces

Namespacing convention:

- service snapshots live under `services/<name>`

## 7. Data Integrity and Operational Invariants

- State published to runtime must be copy-safe.
- Persistence is instance-isolated; cross-instance state bleed is a bug.
- Service repositories are authoritative for durable state.
- Delivery layers should transform, not own, persistent state.

