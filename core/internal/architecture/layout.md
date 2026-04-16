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

Delivery is transport-specific. HTTP and CLI formatting stay under `core/internal/delivery/`, while shared wiring belongs in `core/internal/composition/`. Bootstrap is explicit and one-time: composition attaches service state before assembling the root, and services expose routes through the orchestrator contract rather than by mutating core runtime code.

Runtime snapshots are copy-on-read and sorted for stable presentation. Shared service state is mutex-backed and cloned so presentation layers and services cannot alias internal runtime storage.

For the practical service checklist and test matrix, start with [service onboarding](./service_onboarding.md).
