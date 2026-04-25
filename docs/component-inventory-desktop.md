# Desktop Component Inventory

**Part ID:** `desktop`

## Main Process Components

- IPC handlers:
  - `s3-ipc.ts`
  - `dynamodb-ipc.ts`
  - `sqs-ipc.ts`
  - `mildstack-ipc.ts`
- Instance and endpoint support:
  - `instance-state.ts`
  - `local-endpoint.ts`
  - `setup-cli.ts`
- Shared middleware:
  - `ipc-middleware.ts` (instance-running validation)

## Preload Components

- Typed `window.api` bridge exposing service and runtime operations to renderer.

## Renderer Feature Components

- `features/instances`
- `features/s3-browser`
- `features/dynamodb-browser`
- `features/sqs-browser`

## Shared UI Components

- `components/shared/*` for shell-level building blocks
- `components/ui/*` for reusable primitives

