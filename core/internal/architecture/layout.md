# Core Internal Layout

This repository uses a feature-oriented tree under `core/internal/`. The current layout is the one future contributors should mirror:

- `core/internal/domain/`
- `core/internal/application/orchestrator/`
- `core/internal/application/runtime/`
- `core/internal/composition/`
- `core/internal/infrastructure/`
- `core/internal/delivery/http/`
- `core/internal/delivery/cli/`
- `core/internal/delivery/cli/ui/`
- `core/internal/resources/s3/`
- `core/internal/resources/dynamodb/`
- `core/internal/<feature>/domain/`
- `core/internal/<feature>/application/`
- `core/internal/<feature>/infrastructure/`

The shared roots above stay visible because they are part of the architecture contract, not incidental helpers. The current feature trees live under `core/internal/resources/s3/` and `core/internal/resources/dynamodb/`. Their `domain`, `application`, and `infrastructure` packages stay feature-local, while the shared roots handle orchestration, runtime snapshots, composition, and transport presentation.

These two services define the reusable real-service template for the milestone: domain-owned mutation, application-layer request/response methods, and a feature-local `infrastructure/` package for route catalogs and thin adapters.

## Boundary Rules

The emulation policy lives with the orchestrator contract in `core/internal/application/orchestrator/`. That keeps `Policy()`, `Fidelity`, supported behavior, unsupported behavior, and error-prefix conventions attached to the service interface instead of being inferred from delivery code or runtime internals.

Runtime owns shared state and snapshots. Delivery owns transport-specific presentation. Composition wires the pieces together. That split keeps route registration explicit, keeps runtime state out of request handlers, and keeps the policy surface close to the service implementation. Package-local infrastructure handlers may shape requests and responses, but they should still delegate to application methods instead of owning authoritative state.

The CLI delivery layer follows the same separation. Human-readable terminal output stays in `core/internal/delivery/cli/`, the interactive Charm view stays in `core/internal/delivery/cli/ui/`, and machine-readable `--json` output is shaped at the delivery boundary without leaking styling or formatting concerns into runtime code. The `ports` command should remain useful even when no ports are registered, so empty states must render explicitly rather than returning silence.

Global CLI data now resolves through a user-scoped home directory layout. The base directory lives at `~/.mildstack/` on macOS and Linux, with a Windows-safe home equivalent chosen automatically when the platform requires it. The resolver exposes explicit `config/`, `instances/`, `logs/`, and `cache/` subdirectories, and the CLI storage layer prefers the new layout while still reading legacy paths during the transition window.

- `core/internal/domain/` documents the shared domain boundary used by feature packages
- `core/internal/application/orchestrator/` owns the service contract, fidelity policy, and route/state hooks
- `core/internal/application/runtime/` stores the shared state hook and copy-safe runtime snapshots
- `core/internal/composition/` assembles services, hooks, and registrars during bootstrap
- `core/internal/infrastructure/` documents the shared infrastructure boundary used by feature packages
- `core/internal/delivery/` formats HTTP and CLI output without owning state or emulation policy

## Shared Runtime Invariants

Runtime snapshots are copy-on-read and sorted for stable presentation. Shared service state is mutex-backed and cloned so presentation layers and services cannot alias internal runtime storage. Nested state should also be cloned before snapshots are attached to the runtime hook so copied payloads stay safe to mutate in tests and transport adapters.

The shared state hook expects service-owned keys to stay under `services/<name>`. That namespacing keeps service state isolated from framework/runtime keys and makes the source of data obvious when operators inspect the runtime ledger.

Services should treat copied snapshots as read-only views. If a caller mutates a returned snapshot, the live runtime state must remain unchanged.

## Layout Guidance

Keep the current feature-oriented tree intact. The architecture here is scoped to `core/` and the shared runtime boundary. Avoid introducing desktop-app guidance into this milestone.

For the practical service checklist and test matrix, start with [service onboarding](./service_onboarding.md).
