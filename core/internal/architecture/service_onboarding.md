# Service Onboarding

Use this guide together with [Core Internal Layout](./layout.md) when adding a new feature service.

## Package Layout

New services should follow the same shape as the existing S3 and DynamoDB exemplars:

- `core/internal/<feature>/domain/` for framework-free business rules
- `core/internal/<feature>/application/` for the service implementation
- `core/internal/<feature>/infrastructure/` for adapters that talk outward
- `core/internal/delivery/http/` and `core/internal/delivery/cli/` for transport-specific presentation
- `core/internal/composition/` for shared wiring and bootstrap assembly

## Contract Expectations

Every service should satisfy `orchestrator.Service`:

- `Metadata()` returns a stable name, description, version, and tags
- `RegisterRoutes()` attaches transport routes explicitly during bootstrap
- `AttachState()` seeds namespaced runtime state for the shared ledger
- `Start()` and `Stop()` remain no-ops unless the service truly owns runtime lifecycle work

## Bootstrap Model

Bootstrap is intentionally explicit. `composition.DefaultRoot()` creates the shared state hook, attaches each service once, and panics if attachment fails. That failure mode is deliberate so a bad service wiring cannot silently start a broken binary.

Keep route registration and state attachment in the bootstrap path instead of moving them into request-time helpers or core runtime abstractions.

## Test Matrix

Future services should add the same regression shape as the existing exemplars:

- `core/internal/<feature>/application/service_test.go`
- `TestServiceMetadataRoutesAndState`
- `TestServiceStartAndStopAreNoops`

The service test should verify metadata, route registration, state attachment, and lifecycle no-ops. If a service exposes extra behavior, add focused tests near the service package instead of broad integration helpers.

## Existing Exemplars

Treat the current `core/internal/s3/application/service_test.go` and `core/internal/dynamodb/application/service_test.go` as templates. They show the expected contract shape, the route-registration pattern, and the way state should be namespaced through the runtime hook.
