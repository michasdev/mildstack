# mildstack - Project Overview

**Date:** 2026-04-25  
**Type:** Monorepo (core + desktop + web)  
**Architecture:** Feature-oriented clean architecture with service-local domains

## Executive Summary

MildStack is a local AWS emulator focused on being a lightweight, developer-first alternative to LocalStack.  
The repository is centered on a Go core runtime (`core/`) that emulates AWS-compatible APIs and stateful behavior for S3, DynamoDB, and SQS.  
Two companion projects in `apps/` consume or package this core:

- `apps/desktop`: Electron app that controls and inspects local MildStack instances
- `apps/web/www`: Marketing/documentation website deployed to GitHub Pages

The core is the strategic center of the platform and contains most of the domain complexity, protocol compatibility, persistence logic, and runtime orchestration.

## Project Classification

- **Repository Type:** Monorepo
- **Primary Part:** `core` (Go backend + CLI runtime)
- **Secondary Parts:** `desktop` (Electron + React), `web` (Vite + React)
- **Primary Languages:** Go, TypeScript
- **Primary Runtime Pattern:** Service contract + per-service domain/application/infrastructure stacks

## High-Level Parts

### core

- **Location:** `core/`
- **Purpose:** Emulate AWS surfaces locally with persistent instance-scoped state
- **Stack:** Go 1.26.2, Gin, Cobra, SQLite (`modernc.org/sqlite`)
- **Services:** S3, DynamoDB, SQS

### desktop

- **Location:** `apps/desktop/`
- **Purpose:** GUI control plane for local core instances
- **Stack:** Electron, React, TypeScript, Electron Vite
- **Integration:** Uses IPC in main process, then calls core-compatible endpoints

### web

- **Location:** `apps/web/www/`
- **Purpose:** Public website and product messaging
- **Stack:** Vite, React, TypeScript, Tailwind CSS
- **Integration:** No direct runtime orchestration logic

## Technology Stack Summary

| Category | Technology | Version |
|---|---|---|
| Core language | Go | 1.26.2 |
| Core HTTP | Gin | v1.12.0 |
| Core CLI | Cobra | v1.10.1 |
| Core DB | SQLite (pure Go) | v1.49.1 |
| AWS SDK (core dependencies) | aws-sdk-go-v2 | v1.41.6 |
| Desktop shell | Electron | ^39.2.6 |
| Desktop/web UI | React | ^19.x |
| Desktop/web language | TypeScript | ^5.9.x |
| Website build | Vite | ^7.x |

## Key Features (Current Core Scope)

- Local instance lifecycle (`start`, `status`, `instances`, `stop`, `delete`)
- Instance-scoped persistent storage rooted in `~/.mildstack/instances/<instanceId>/...`
- S3 AWS-compatible HTTP surface with rich subresources (versioning, object lock, replication, multipart, tagging, ACL, bucket-level controls)
- DynamoDB compatible target-based API (`X-Amz-Target`) plus internal runtime routes
- SQS support for both query-style and target-style requests, with lifecycle/governance/message surfaces
- Runtime service catalog and readiness endpoints under `/api/v1/runtime/...`

## Architecture Highlights

- Service contract is standardized via `orchestrator.Service`
- Each service owns:
  - `domain` state and invariants
  - `application` mutation/query workflows
  - `infrastructure` route catalogs and thin adapters
- Shared runtime state is published through a copy-safe `StateHook`
- Composition bootstrap wires all services per instance ID
- Desktop app does not embed core logic; it communicates through API/IPC boundaries

## Development Entry Points

- Core CLI entrypoint: `core/cmd/mildstack/main.go`
- Desktop entrypoint: `apps/desktop/src/main/index.ts` + `apps/desktop/src/renderer/src/main.tsx`
- Website entrypoint: `apps/web/www/src/main.tsx`

## Documentation Map

- [index.md](./index.md)
- [architecture-core.md](./architecture-core.md)
- [api-contracts-core.md](./api-contracts-core.md)
- [data-models-core.md](./data-models-core.md)
- [integration-architecture.md](./integration-architecture.md)
- [source-tree-analysis.md](./source-tree-analysis.md)

