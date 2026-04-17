---
phase: 15-bucket-governance-and-object-lock
plan: 01
subsystem: s3
tags: [go, s3, governance, persistence, routing]
depends_on: [14-03]
requires: [S3-08]
provides: [bucket-governance-state, governance-service-surface, governance-route-catalog]
affects: [core/internal/s3/domain, core/internal/s3/application, core/internal/s3/infrastructure]
tech-stack:
  added: [none]
  patterns: [bucket-scoped state maps, thin transport adapters, atomic JSON persistence]
key-files:
  created:
    - core/internal/s3/application/service_subresources.go
    - core/internal/s3/infrastructure/handlers_subresources.go
    - core/internal/s3/infrastructure/routes_subresources.go
  modified:
    - core/internal/s3/domain/state.go
    - core/internal/s3/domain/state_test.go
    - core/internal/s3/application/repository_fs.go
    - core/internal/s3/application/repository_fs_test.go
    - core/internal/s3/application/service.go
    - core/internal/s3/application/service_test.go
    - core/internal/s3/infrastructure/handlers.go
    - core/internal/s3/infrastructure/handlers_test.go
    - core/internal/s3/infrastructure/routes.go
    - core/internal/s3/infrastructure/routes_test.go
decisions:
  - Keep bucket governance as bucket-scoped raw-body stores with ACL falling back to a canonical default response.
  - Clear all governance maps during bucket deletion so no orphaned subresource state survives.
  - Expose the new governance surface through thin service, route, and handler slices instead of expanding the hot object path.
metrics:
  duration: "about 1 hour"
  completed_date: "2026-04-17"
---
# Phase 15 Plan 01: Bucket Governance and Object Lock Summary
Bucket governance subresources now persist as bucket-scoped state with copy-safe get/put/delete handlers and ACL defaults.

## Completed Work

| Task | Name | Commit | Files |
| ---- | ---- | ------ | ----- |
| 1 | Add durable bucket governance storage and the first S3-08 subresource surface | `7c3d7a73` | `core/internal/s3/domain/state.go`, `core/internal/s3/application/repository_fs.go`, `core/internal/s3/application/service.go`, `core/internal/s3/application/service_subresources.go`, `core/internal/s3/infrastructure/handlers.go`, `core/internal/s3/infrastructure/handlers_subresources.go`, `core/internal/s3/infrastructure/routes.go`, `core/internal/s3/infrastructure/routes_subresources.go`, `core/internal/s3/domain/state_test.go`, `core/internal/s3/application/repository_fs_test.go`, `core/internal/s3/application/service_test.go`, `core/internal/s3/infrastructure/handlers_test.go`, `core/internal/s3/infrastructure/routes_test.go` |

## Deviations from Plan

None. The governance slice stayed inside the existing S3 service boundary and the object CRUD path remained unchanged.

## Verification

- `go test ./core/internal/s3/domain ./core/internal/s3/application -count=1`
- `go test ./core/internal/s3/infrastructure -count=1`
- `go test ./core/internal/s3/... -count=1`

## Self-Check: PASSED

- `core/internal/s3/application/service_subresources.go` exists.
- `git log --oneline --all` contains `7c3d7a73`.
