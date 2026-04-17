---
phase: 15
plan: 02
subsystem: s3
tags: [s3, bucket-governance, replication, object-lock]
dependency_graph:
  requires: [15-01]
  provides: [S3-08 bucket event governance slice]
  affects:
    - core/internal/s3/domain/state.go
    - core/internal/s3/application/repository_fs.go
    - core/internal/s3/application/service.go
    - core/internal/s3/application/service_bucket_events.go
    - core/internal/s3/infrastructure/routes.go
    - core/internal/s3/infrastructure/routes_bucket_events.go
    - core/internal/s3/infrastructure/handlers.go
    - core/internal/s3/infrastructure/handlers_bucket_events.go
tech_stack:
  added: [encoding/xml]
  patterns:
    - bucket-scoped raw config storage for notification and logging
    - structured replication config with a versioning gate
    - thin route and handler adapters over application services
key_files:
  created:
    - core/internal/s3/application/service_bucket_events.go
    - core/internal/s3/infrastructure/routes_bucket_events.go
    - core/internal/s3/infrastructure/handlers_bucket_events.go
  modified:
    - core/internal/s3/domain/state.go
    - core/internal/s3/domain/state_test.go
    - core/internal/s3/application/repository_fs.go
    - core/internal/s3/application/repository_fs_test.go
    - core/internal/s3/application/service.go
    - core/internal/s3/application/service_test.go
    - core/internal/s3/infrastructure/routes.go
    - core/internal/s3/infrastructure/handlers.go
    - core/internal/s3/infrastructure/routes_test.go
    - core/internal/s3/infrastructure/handlers_test.go
decisions:
  - Store notification and logging as raw XML blobs and return bucket-scoped default XML when no config exists.
  - Store replication as structured state and require bucket versioning to be enabled before a put is accepted.
  - Keep route registration explicit with a dedicated bucket-event slice so the surface stays testable and bounded.
metrics:
  duration: "15m"
  completed_date: "2026-04-17T14:59:52Z"
---

# Phase 15 Plan 02: Bucket Event Governance Summary

Bucket notification, logging, and replication support now persists through the S3 snapshot, returns bucket-scoped defaults for notification and logging, and gates replication behind enabled bucket versioning.

## What Changed

- Added notification, logging, and replication state to the S3 domain model and repository normalization path.
- Added service methods for get/put/delete bucket-event subresources with raw XML echo for notification and logging and structured replication handling.
- Added dedicated route and handler slices and extended the route catalog to include the new endpoints exactly once.
- Expanded tests to cover persistence round-trips, replication versioning gating, bucket-delete cleanup, and route registration counts.

## Verification

- `go test ./core/internal/s3/domain ./core/internal/s3/application ./core/internal/s3/infrastructure -count=1`
- Route catalog count now resolves to 42 entries with the new bucket-event routes present once.
- Repository round-trips now preserve notification, logging, and replication configs.
- Replication puts fail until bucket versioning is enabled on the source bucket.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking issue] Resolved a `BucketLoggingConfig` field and method name collision**
- **Found during:** implementation and initial test run
- **Issue:** the new logging storage field collided with the existing `BucketLoggingConfig(...)` accessor name in `domain.State`
- **Fix:** renamed the storage field to `BucketLogging` and kept the accessor name intact
- **Files modified:** `core/internal/s3/domain/state.go`, `core/internal/s3/application/repository_fs.go`
- **Commit:** `0b92354e`

## Self-Check: PASSED
