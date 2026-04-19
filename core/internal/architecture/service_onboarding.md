# Service Onboarding

Use this guide together with [Core Internal Layout](./layout.md) when adding a new feature service. Start here when you need to understand how a service declares fidelity, what it supports, and how it participates in the shared runtime. The current S3 and DynamoDB services prove the real-service template: domain-owned state, application-layer mutation, and a thin feature-local transport adapter.

## Package Layout

New services should follow the same shape as the existing S3 and DynamoDB real-service template:

- `core/internal/domain/` for the shared domain boundary
- `core/internal/application/orchestrator/` for the service contract, fidelity policy, and route/state hooks
- `core/internal/application/runtime/` for copy-safe snapshots and the shared state hook
- `core/internal/composition/` for shared wiring and bootstrap assembly
- `core/internal/infrastructure/` for the shared infrastructure boundary
- `core/internal/delivery/http/` for HTTP presentation
- `core/internal/delivery/cli/` and `core/internal/delivery/cli/ui/` for CLI presentation
- `core/internal/resources/instancepath/` for the shared instance-scoped storage helper used by AWS-backed services
- `core/internal/<feature>/domain/` for framework-free business rules
- `core/internal/<feature>/application/` for the service implementation
- `core/internal/<feature>/infrastructure/` for route catalogs and thin request/response adapters

## Emulation Contract

Every service should satisfy `orchestrator.Service`, including the new `Policy()` contract:

- `Policy()` returns an `orchestrator.EmulationPolicy`
- `Metadata()` returns a stable name, description, version, and tags
- `RegisterRoutes()` attaches transport routes explicitly during bootstrap
- `AttachState()` seeds namespaced runtime state for the shared ledger
- `Start()` and `Stop()` remain no-ops unless the service truly owns runtime lifecycle work

The policy is the first-class way to describe how complete a service is. The contract fields are:

- `Fidelity` records the service level:
  - `exemplar` means the service is a reference-quality implementation for the current milestone
  - `partial` means the service is intentionally incomplete but still exposes supported behavior
  - `unsupported` means the service exists only as a placeholder and does not claim real behavior
- `Supported` lists the operations or behaviors the service does implement
- `Unsupported` lists the operations or behaviors that must fail closed
- `ErrorPrefix` namespaces unsupported errors so callers can identify the service that rejected the request

When a service rejects an operation, use the shared helper so errors stay consistent:

- `orchestrator.UnsupportedError(policy, operation)` formats the failure with the policy prefix
- unsupported responses should describe the operation as unsupported rather than inventing service-specific wording

Service authors should keep `Supported` and `Unsupported` aligned with the actual surface. If a service is partial, the guide for that service should make it obvious which operations are safe to call and which ones intentionally fail. For real services, `Supported` should mirror the concrete request/response endpoints the feature actually exposes.

## Shared Runtime Rules

Runtime state is shared through the bootstrap hook, and the stored values are copy-safe. Services should treat returned snapshots as read-only and should not rely on aliasing internal runtime storage. Domain state helpers should clone nested slices and maps before returning data to the runtime or to transport adapters.

Service state keys must stay namespaced under `services/<name>`. That convention keeps service-owned state isolated from framework/runtime keys and makes the source of a value obvious when operators inspect the ledger.

Keep route registration and state attachment in the bootstrap path instead of moving them into request-time helpers or core runtime abstractions. `composition.DefaultRoot()` creates the shared state hook, attaches each service once, and panics if attachment fails. That failure mode is deliberate so a bad service wiring cannot silently start a broken binary.

## CLI Contract

The shared CLI is operator-first, but it also has a stable machine-readable contract now:

- `status` renders human-friendly output by default and supports `--json` for stable automation output
- `ports` must always print something useful in human mode, including an explicit empty-state message when no ports are registered
- JSON output should be shaped at the CLI boundary from copy-safe runtime snapshots, not from mutable service internals
- terminal styling belongs in delivery packages only, and the light-green accent should stay restrained so the command output remains readable

## Global Storage Layout

User-scoped CLI files should go through the resolver in `core/internal/application/runtime/` instead of ad hoc path joins. The new layout uses `~/.mildstack/` on macOS and Linux, with explicit `config/`, `instances/`, `logs/`, and `cache/` subdirectories. During the transition period, read paths may still honor legacy locations, but new writes should land in the home-scoped layout.

Instance records are further split under `instances/` so the CLI can tell active ports apart from saved instance metadata:

- `instances/active/` stores the currently active ports the CLI should surface in `status` and `ports`
- `instances/saved/` keeps the saved instance records that let a user rerun the same instance later

AWS-backed services should use `MILDSTACK_INSTANCE_ID` as the required bootstrap identity seed and must resolve storage under `instances/<instanceID>/<service>`. The helper in `core/internal/resources/instancepath/` owns the path assembly rule so new services do not duplicate it. If a service cannot prove that layout in tests, it is not ready to be wired into the root composition.

## Test Matrix

Future services should add the same regression shape as the current S3 and DynamoDB template:

- `core/internal/<feature>/application/service_test.go`
- `core/internal/<feature>/domain/state_test.go`
- `core/internal/<feature>/infrastructure/routes_test.go`
- `core/internal/<feature>/infrastructure/handlers_test.go`
- `core/internal/resources/s3/application/service_test.go`
- `core/internal/resources/s3/domain/state_test.go`
- `core/internal/resources/s3/infrastructure/routes_test.go`
- `core/internal/resources/s3/infrastructure/handlers_test.go`
- `core/internal/resources/dynamodb/application/service_test.go`
- `core/internal/resources/dynamodb/domain/state_test.go`
- `core/internal/resources/dynamodb/infrastructure/routes_test.go`
- `core/internal/resources/dynamodb/infrastructure/handlers_test.go`
- `TestServiceMetadataRoutesAndState`
- `TestServiceRealOperationsMutateState`
- `TestServiceRejectsInvalidAndMissingRequests`
- `TestServiceStartAndStopAreNoOps`
- `MILDSTACK_INSTANCE_ID`
- `instances/<instanceID>/<service>`

The service test should verify metadata, `Policy()`, route registration, state attachment, and lifecycle no-ops. The domain test should prove snapshots and helpers copy nested state. The infrastructure test should verify the request/response adapter copies payloads and fails closed. If a service exposes extra behavior, add focused tests near the service package instead of broad integration helpers.

## Existing Template

Treat the current S3 and DynamoDB tests as the canonical template. They show the expected contract shape, the route-registration pattern, the emulation policy fields, the way state should be namespaced through the runtime hook, and the boundary between application logic and package-local transport adapters.
