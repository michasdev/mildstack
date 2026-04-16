# Core Internal Layout

The architecture establishes a feature-oriented tree under `core/internal/`, where each feature owns its own layer subpackages:

- `core/internal/<feature>/domain/`
- `core/internal/<feature>/application/`
- `core/internal/<feature>/infrastructure/`
- `core/internal/delivery/http/`
- `core/internal/delivery/cli/`
- `core/internal/application/orchestrator/`
- `core/internal/composition/`

`core/internal/<feature>/domain` and `core/internal/<feature>/application` stay framework-free. Feature packages own their own subpackages inside each layer, so future AWS services can grow without inventing new top-level structure.

`core/internal/api` and `core/internal/cli` are not part of the phase-1 structure. Delivery is split by transport under `core/internal/delivery/http/` and `core/internal/delivery/cli/`, while shared assembly belongs in `core/internal/composition/`.
