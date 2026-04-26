# Desktop Architecture

**Part ID:** `desktop`  
**Path:** `apps/desktop/`  
**Stack:** Electron + React + TypeScript

## Summary

The desktop app is a visual control plane around the core runtime.  
It does not implement AWS emulation logic itself; it orchestrates and consumes core-compatible operations via Electron IPC.

## Process Layout

- `src/main/`: privileged process, owns IPC handlers and runtime process integration
- `src/preload/`: typed bridge exposed to renderer (`window.api`)
- `src/renderer/src/`: React UI

## Key Feature Areas

- Instance management
- S3 browser
- DynamoDB browser
- SQS browser

## Integration with Core

- Renderer invokes `window.api.*`
- Main process maps those calls into:
  - local endpoint resolution
  - service commands
  - validation that an instance is running before operations

## Routing and State

- Renderer routing uses hash router (`createHashRouter`)
- Local UI state uses React hooks and Zustand where needed

## Build and Packaging

- Dev/build tooling: `electron-vite`, `vite`, TypeScript
- Packaging: `electron-builder`
- Includes scripts to compile core CLI binaries for multiple OS/arch targets

