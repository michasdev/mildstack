# mildstack Documentation Index

**Type:** Monorepo with 3 parts  
**Primary Language:** Go  
**Architecture:** Feature-oriented core runtime + supporting desktop/web apps  
**Last Updated:** 2026-04-25

## Project Overview

MildStack is a local AWS emulator aimed at becoming a lightweight substitute for LocalStack for day-to-day development flows.  
The repository is centered around the Go `core/` runtime, with companion `desktop` and `web` projects in `apps/`.

## Project Structure

### Core Runtime (`core`)

- **Type:** backend
- **Tech Stack:** Go, Gin, Cobra, SQLite
- **Entry Point:** `core/cmd/mildstack/main.go`

### Desktop (`apps/desktop`)

- **Type:** desktop
- **Tech Stack:** Electron, React, TypeScript
- **Entry Points:** main process + renderer entrypoint

### Web (`apps/web/www`)

- **Type:** web
- **Tech Stack:** Vite, React, TypeScript, Tailwind
- **Entry Point:** `apps/web/www/src/main.tsx`

## Generated Documentation

### Core Documentation

- [Project Overview](./project-overview.md)
- [Source Tree Analysis](./source-tree-analysis.md)
- [Integration Architecture](./integration-architecture.md)
- [Deployment Guide](./deployment-guide.md)
- [Project Parts Metadata](./project-parts.json)

### Core Runtime (Deep Focus)

- [Architecture - Core](./architecture-core.md)
- [API Contracts - Core](./api-contracts-core.md)
- [Data Models - Core](./data-models-core.md)
- [Component Inventory - Core](./component-inventory-core.md)
- [Development Guide - Core](./development-guide-core.md)

### Desktop (Light Scan)

- [Architecture - Desktop](./architecture-desktop.md)
- [API Contracts - Desktop](./api-contracts-desktop.md)
- [Component Inventory - Desktop](./component-inventory-desktop.md)
- [Development Guide - Desktop](./development-guide-desktop.md)

### Web (Light Scan)

- [Architecture - Web](./architecture-web.md)
- [API Contracts - Web](./api-contracts-web.md)
- [Data Models - Web](./data-models-web.md)
- [Component Inventory - Web](./component-inventory-web.md)
- [Development Guide - Web](./development-guide-web.md)

## Existing Repository Documentation

- [Root README](../README.md)
- [Desktop README](../apps/desktop/README.md)
- [Web README](../apps/web/www/README.md)
- [Core Internal Layout](../core/internal/architecture/layout.md)
- [Core Service Onboarding](../core/internal/architecture/service_onboarding.md)
- [S3 AWS-Compatible Surface Notes](../core/internal/resources/s3/README.md)

## Getting Started by Task

- **Core feature work:** start with `architecture-core.md`, then `api-contracts-core.md`, then `data-models-core.md`.
- **Desktop integration work:** start with `architecture-desktop.md` and `api-contracts-desktop.md`.
- **Website updates:** start with `architecture-web.md` and `development-guide-web.md`.

