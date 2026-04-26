# Desktop Development Guide

**Part ID:** `desktop`

## Prerequisites

- Node.js (project workflows currently use Node 22.x in CI)
- npm

## Install

```bash
cd /Users/michel.freitas/Documents/projects/others/mildstack/apps/desktop
npm ci
```

## Development

```bash
npm run dev
```

## Typecheck and Test

```bash
npm run typecheck
npm run test
```

## Build

```bash
npm run build
```

## Core Integration Notes

- Renderer must consume `window.api` from preload.
- Main process IPC handlers are the right place for service calls.
- Keep instance validation in the middleware path before S3/DynamoDB/SQS operations.

