# Service Onboarding

Use this guide together with [Core Internal Layout](./layout.md) when adding a new feature service. Start here when you need to understand how a service declares fidelity, what it supports, and how it participates in the shared runtime.

## Package Layout

New services should follow the same shape as the existing S3 and DynamoDB exemplars:

- `core/internal/<feature>/domain/` for framework-free business rules
- `core/internal/<feature>/application/` for the service implementation
- `core/internal/<feature>/infrastructure/` for adapters that talk outward
- `core/internal/delivery/http/` and `core/internal/delivery/cli/` for transport-specific presentation
- `core/internal/composition/` for shared wiring and bootstrap assembly

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

Service authors should keep `Supported` and `Unsupported` aligned with the actual surface. If a service is partial, the guide for that service should make it obvious which operations are safe to call and which ones intentionally fail.

## Shared Runtime Rules

Runtime state is shared through the bootstrap hook, and the stored values are copy-safe. Services should treat returned snapshots as read-only and should not rely on aliasing internal runtime storage.

Service state keys must stay namespaced under `services/<name>`. That convention keeps service-owned state isolated from framework/runtime keys and makes the source of a value obvious when operators inspect the ledger.

Keep route registration and state attachment in the bootstrap path instead of moving them into request-time helpers or core runtime abstractions. `composition.DefaultRoot()` creates the shared state hook, attaches each service once, and panics if attachment fails. That failure mode is deliberate so a bad service wiring cannot silently start a broken binary.

## Test Matrix

Future services should add the same regression shape as the existing exemplars:

- `core/internal/<feature>/application/service_test.go`
- `TestServiceMetadataRoutesAndState`
- `TestServiceStartAndStopAreNoOps`

The service test should verify metadata, `Policy()`, route registration, state attachment, and lifecycle no-ops. If a service exposes extra behavior, add focused tests near the service package instead of broad integration helpers.

## Existing Exemplars

Treat the current `core/internal/s3/application/service_test.go` and `core/internal/dynamodb/application/service_test.go` as templates. They show the expected contract shape, the route-registration pattern, the emulation policy fields, and the way state should be namespaced through the runtime hook.
