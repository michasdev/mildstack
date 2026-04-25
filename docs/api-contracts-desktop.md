# Desktop API Contracts

**Part ID:** `desktop`  
**Type:** Internal IPC contracts (not public HTTP API)

## Summary

The desktop application communicates through Electron IPC channels exposed by `preload` to renderer code.

Key channel groups:

- `s3:*` (bucket/object browsing and mutations)
- `dynamodb:*` (table/item operations)
- `sqs:*` (queue/message operations)
- `mildstack:*` (instance lifecycle and validation)
- `instance:*` (selected instance state)

## Contract Shape

- Renderer -> `window.api.<service>.<method>(...)`
- Main process -> `ipcMain.handle(<channel>, ...)`
- Main process validates instance state before service actions (`ipc-middleware`)

## Notes

- This is an internal app contract.
- External clients should target the core native/runtime HTTP surfaces, not desktop IPC.

