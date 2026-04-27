# Core Architecture (Go Runtime)

**Part ID:** `core`  
**Path:** `core/`  
**Primary Language:** Go  
**Role:** Local AWS emulation runtime

## 1. Intent and Scope

The core runtime is designed to emulate selected AWS service behavior locally with predictable, persistent, instance-scoped state.  
It is the critical execution engine behind MildStack's goal of becoming a lightweight replacement for LocalStack in common development workflows.

Current high-value services:

- S3
- DynamoDB
- SQS

## 2. Architectural Style

Core uses a feature-oriented clean architecture under `core/internal/`:

- Shared orchestration contracts in `application/orchestrator`
- Runtime state and lifecycle support in `application/runtime`
- Explicit service wiring in `composition`
- Transport boundaries in `delivery/http` and `delivery/cli`
- Service-local modules in `resources/<service>/{domain,application,infrastructure}`

This keeps transport and persistence concerns out of service domain mutation logic.

## 3. Bootstrap and Runtime Flow

Main executable: `core/cmd/mildstack/main.go`

### Boot sequence

1. Resolve runtime paths (`~/.mildstack/...`) with migration fallback.
2. Load active ports from storage.
3. Create `runtime.Manager` with known ports.
4. Build CLI command set (`start`, `instances`, `status`, `stop`, `delete`).
5. On `start`:
   - Resolve `instanceId` for port
   - Build `composition.DefaultRoot(instanceId)`
   - Register service routes in runtime registrar
   - Attach native AWS protocol adapters (S3/DynamoDB/SQS)
   - Start Gin-backed HTTP server

### Runtime manager responsibilities

- Registered ports (running instances)
- Service metadata snapshot
- Stable snapshot projection for delivery layer

## 4. Core Contracts

`orchestrator.Service` is the canonical contract:

- `Start(context.Context) error`
- `Stop(context.Context) error`
- `Metadata() Metadata`
- `Policy() EmulationPolicy`
- `RegisterRoutes(RouteRegistrar) error`
- `AttachState(StateHook) error`

This contract ensures every service declares:

- lifecycle hooks
- discoverable route inventory
- explicit emulation fidelity and unsupported behavior
- state publication into shared runtime ledger

## 5. Service Composition

`composition.DefaultRoot(instanceID)` creates service instances and attaches a shared state hook:

- `s3.NewWithStorage(...)`
- `dynamodb.NewWithStorage(...)`
- `sqs.NewWithStorage(...)`

All three services are initialized with instance-scoped storage.  
If any initialization or state-hook attachment fails, composition fails fast.

## 6. Delivery Boundaries

## HTTP Delivery

`delivery/http` provides:

- Runtime endpoints:
  - `/api/v1/runtime/health`
  - `/api/v1/runtime/ready`
  - `/api/v1/runtime/info`
  - `/api/v1/runtime/services`
  - `/api/v1/runtime/services/:service`
- Native protocol adapters:
  - S3 AWS-style path/query surface
  - DynamoDB `X-Amz-Target` JSON surface
  - SQS query-style and target-style surfaces

The runtime route `Registrar` validates and catalogs service routes with duplicate protection.

## CLI Delivery

`delivery/cli` handles operator workflows and machine output:

- command composition
- lifecycle operations
- storage-backed instance metadata
- detached start mode readiness signaling

## 7. State and Persistence Model

## Global runtime storage

Resolved under `~/.mildstack/`:

- `config/`
- `instances/`
- `logs/`
- `cache/`

Instance records split into:

- `instances/active/`
- `instances/saved/`

## Instance-scoped service data

Local service resources live under:

- `instances/<instanceId>/<service>/...`

via shared helper `resources/instancepath`.

## Service persistence backends

- S3: filesystem repository (`state.json` + payload references)
- DynamoDB: SQLite repository (`state.db`)
- SQS: SQLite repository (`state.db`)

## 8. Service-Specific Architectural Notes

## S3

- Broadest current feature surface.
- Supports bucket/object control-plane + multipart + object lock + replication + versioning + governance subresources.
- Uses filesystem-backed persisted state and payload indirection.
- Includes AWS-style defaults for selected control-plane responses.

## DynamoDB

- Implements table + item surfaces plus query/scan and batch/transaction-like operations in app layer.
- Native adapter resolves action by `X-Amz-Target`.
- Persists tables/items and index metadata in SQLite.

## SQS

- Rich action catalog (queue lifecycle, governance, redrive, message operations).
- Registry marks supported vs deferred actions.
- Supports query and target request styles.
- Runs background worker behavior for delivery/visibility semantics and persists queue/message/governance data in SQLite.

## 9. AWS Identity and Protocol Context

`resources/awscontext` centralizes default local identity:

- account id
- region
- partition
- local endpoint
- ARN generation helpers

This keeps AWS identity semantics separate from local storage layout (`instancepath`).

## 10. Testing Strategy (Observed)

Core has comprehensive test coverage distributed across:

- service application tests
- domain state tests
- route/handler tests
- runtime manager/state hook tests
- contract tests for protocol handling

The project uses CI workflow `core-go-ci.yml` running:

- `go test ./...`
- `go build ./...`

## 11. Operational and Design Invariants

- Service-owned runtime state must be namespaced (`services/<name>`).
- Runtime snapshots must be copy-safe.
- Route registration is explicit and validated.
- Transport layers should remain thin adapters over application/service methods.
- Instance identity is mandatory for persisted AWS-backed service data.

## 12. Risks and Deep-Dive Follow-Ups

- S3 surface breadth is high; regression risk is concentrated in query-param dispatch and XML contracts.
- DynamoDB and SQS native protocol adapters require strict contract compatibility to avoid SDK drift issues.
- Cross-service parity with AWS edge behavior remains an ongoing, intentional scope boundary.

