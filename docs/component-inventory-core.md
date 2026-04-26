# Core Component Inventory

**Part ID:** `core`

## Domain and Runtime Components

- `orchestrator.Service` contract
- `orchestrator.EmulationPolicy` and fidelity model
- `runtime.Manager` (ports + service metadata snapshot)
- `runtime.MemoryStateHook` (shared service state hook)
- `composition.DefaultRoot` (instance-scoped service assembly)

## Delivery Components

### HTTP

- `Router` with runtime endpoints and service catalog
- `Registrar` (route normalization, duplicate prevention, service-indexing)
- Native protocol adapters:
  - S3 native handler
  - DynamoDB native handler
  - SQS native handler + registry/contract parser

### CLI

- Root command and subcommands (`serve`, `instances`, `status`, `stop`, `delete`)
- Storage abstraction for active/saved instance records
- Detached serve launcher and readiness signaling

## Service Components

### S3

- Domain state and object/version models
- Application service modules (buckets, objects, lifecycle, versioning, governance, multipart, events/access)
- Filesystem repository with validation and payload handling
- Infrastructure route catalogs and handler adapters

### DynamoDB

- Domain tables/items/attribute models
- Application service modules (table/item/query/scan/update/batch/transaction)
- SQLite repository
- Infrastructure route and handler adapters

### SQS

- Domain queues/messages/recovery/governance/move-task models
- Application service modules (queue lifecycle, governance, redrive, message handling)
- Worker for message lifecycle behavior
- SQLite repository
- Infrastructure route catalog

