# Core Internal Layout

This repository uses a feature-oriented tree under `core/internal/`. New services should start from this layout:

- `core/internal/<feature>/domain/`
- `core/internal/<feature>/application/`
- `core/internal/<feature>/infrastructure/`
- `core/internal/delivery/http/`
- `core/internal/delivery/cli/`
- `core/internal/application/orchestrator/`
- `core/internal/application/runtime/`
- `core/internal/composition/`

Current feature trees live under `core/internal/s3/` and `core/internal/dynamodb/`. Their `domain` and `application` packages stay framework-free, and the architecture guard tests that boundary directly.

## Boundary Rules

The emulation policy lives with the orchestrator contract in `core/internal/application/orchestrator/`. That keeps `Policy()`, `Fidelity`, supported behavior, unsupported behavior, and error-prefix conventions attached to the service interface instead of being inferred from delivery code or runtime internals.

Runtime owns shared state and snapshots. Delivery owns transport-specific presentation. Composition wires the pieces together. That split keeps route registration explicit, keeps runtime state out of request handlers, and keeps the policy surface close to the service implementation.

- `core/internal/application/runtime/` stores the shared state hook and copy-safe runtime snapshots
- `core/internal/delivery/` formats HTTP and CLI output without owning state or emulation policy
- `core/internal/composition/` assembles services, hooks, and registrars during bootstrap

## Shared Runtime Invariants

Runtime snapshots are copy-on-read and sorted for stable presentation. Shared service state is mutex-backed and cloned so presentation layers and services cannot alias internal runtime storage.

The shared state hook expects service-owned keys to stay under `services/<name>`. That namespacing keeps service state isolated from framework/runtime keys and makes the source of data obvious when operators inspect the runtime ledger.

Services should treat copied snapshots as read-only views. If a caller mutates a returned snapshot, the live runtime state must remain unchanged.

## Layout Guidance

Keep the current feature-oriented tree intact. The architecture here is scoped to `core/` and the shared runtime boundary. Avoid introducing desktop-app guidance into this milestone.

For the practical service checklist and test matrix, start with [service onboarding](./service_onboarding.md).
