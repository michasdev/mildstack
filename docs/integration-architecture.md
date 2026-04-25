# Integration Architecture

## 1. Core Integration Topology

The repository follows a hub-and-spoke topology:

- `core` is the execution hub (AWS emulation runtime)
- `desktop` is an operator-facing client over local runtime behavior
- `web` is informational and deployment-facing, not runtime-coupled

## 2. Integration Points

### desktop -> core

- **Type:** Local process + API integration
- **Mechanism:** Electron main-process IPC handlers invoke core-compatible operations.
- **Scope:** Instance lifecycle + service operations for S3, DynamoDB, SQS.
- **Critical guard:** Instance validation middleware ensures operations target a running instance.

### web -> core

- **Type:** Product-level integration (indirect)
- **Mechanism:** Website communicates product and distribution, but does not call core runtime APIs in this repository.

### CI/CD integration

- Core CI validates Go build/test on PR changes in `core/**`.
- Desktop CI and release workflows compile core CLI binaries for desktop packaging.
- Web workflow deploys static site to GitHub Pages.

## 3. Data and Control Flow (desktop + core)

1. User action in renderer UI.
2. Renderer calls `window.api`.
3. Preload forwards to main process channel.
4. Main process validates instance state and resolves endpoint.
5. Main process performs service operation (SDK/native request semantics).
6. Result returns through IPC to renderer.

## 4. Contract Boundaries

- Core remains the sole owner of AWS emulation semantics.
- Desktop owns visualization and local UX orchestration.
- Web owns communication/marketing surface.

