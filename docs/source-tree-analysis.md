# mildstack - Source Tree Analysis

**Date:** 2026-04-25

## Overview

This repository is a monorepo with one high-complexity runtime (`core/`) and two lighter companion applications (`apps/desktop`, `apps/web/www`).

## Multi-Part Structure

- **core** (`core/`): Go runtime and AWS emulation engine
- **desktop** (`apps/desktop/`): Electron app for instance/service management
- **web** (`apps/web/www/`): Website

## Complete Directory Structure (Annotated)

```text
mildstack/
├── core/
│   ├── cmd/mildstack/                 # CLI executable entrypoint
│   └── internal/
│       ├── application/
│       │   ├── orchestrator/          # Service contract, emulation policy, route/state contracts
│       │   └── runtime/               # Runtime snapshot, state hook, global path resolver
│       ├── architecture/              # Internal architecture contract docs
│       ├── composition/               # Service wiring (S3 + DynamoDB + SQS) per instance
│       ├── delivery/
│       │   ├── cli/                   # Cobra commands, rendering, storage orchestration
│       │   └── http/                  # Gin router, runtime endpoints, native AWS adapters
│       ├── resources/
│       │   ├── awscontext/            # AWS identity defaults and ARN helpers
│       │   ├── instancepath/          # Instance-scoped storage path contract
│       │   ├── s3/                    # S3 service (domain/application/infrastructure)
│       │   ├── dynamodb/              # DynamoDB service (domain/application/infrastructure)
│       │   └── sqs/                   # SQS service (domain/application/infrastructure)
│       ├── domain/                    # Shared domain boundary docs
│       └── infrastructure/            # Shared infra boundary docs
├── apps/
│   ├── desktop/
│   │   ├── src/main/                  # Electron main process, IPC handlers
│   │   ├── src/preload/               # Typed bridge exposed to renderer
│   │   └── src/renderer/src/          # React UI (instances, S3, DynamoDB, SQS browsers)
│   └── web/www/
│       └── src/                       # Website components + feature sections
└── docs/                              # Generated project documentation for AI-assisted work
```

## Critical Directories

### `core/internal/application/orchestrator`

- **Purpose:** Canonical service contract and emulation policy.
- **Contains:** `Service`, `RouteRegistrar`, `StateHook`, `EmulationPolicy`, fidelity model.

### `core/internal/application/runtime`

- **Purpose:** Runtime snapshots, instance registry logic, storage base path resolution.
- **Contains:** `Manager`, snapshot model, copy-safe in-memory state hook, `~/.mildstack` path conventions.

### `core/internal/composition`

- **Purpose:** Runtime assembly root.
- **Contains:** Instance-aware service wiring (`DefaultRoot`) and bootstrapping of S3, DynamoDB, SQS.

### `core/internal/delivery/http`

- **Purpose:** Transport boundary and protocol adapters.
- **Contains:** Runtime health/info/services endpoints plus native adapters for S3, DynamoDB, SQS.

### `core/internal/resources/{s3,dynamodb,sqs}`

- **Purpose:** Service-local architecture units.
- **Contains:** Domain models, application logic, persistence repositories, route catalogs, transport handlers.

### `apps/desktop/src/main`

- **Purpose:** Desktop orchestration boundary.
- **Contains:** IPC handlers for S3/DynamoDB/SQS and instance lifecycle commands.

### `apps/desktop/src/renderer/src/features`

- **Purpose:** UI feature slices (instances, S3 browser, DynamoDB browser, SQS browser).

### `apps/web/www/src/features/home`

- **Purpose:** Public website landing content and sections.

## Entry Points

- `core/cmd/mildstack/main.go`
- `apps/desktop/src/main/index.ts`
- `apps/desktop/src/renderer/src/main.tsx`
- `apps/web/www/src/main.tsx`

## File Organization Patterns

- Core uses feature-oriented internal architecture and strict boundary responsibilities.
- Each AWS service package follows `domain` / `application` / `infrastructure`.
- Desktop uses process separation (`main` / `preload` / `renderer`).
- Website uses `components/` + `features/` composition.

## Key Configuration Files

- `go.mod` (root): core dependencies and Go version
- `apps/desktop/package.json`: desktop commands and build matrix
- `apps/web/www/package.json`: website scripts
- `.github/workflows/core-go-ci.yml`
- `.github/workflows/desktop-build.yml`
- `.github/workflows/mildstack-release.yml`
- `.github/workflows/deploy-mildstack-dev.yml`

