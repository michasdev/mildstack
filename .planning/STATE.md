---
gsd_state_version: 1.0
milestone: v1.4
milestone_name: dynamodb aws-compatible crud foundation
current_phase: 32
current_phase_name: CLI Instance Identity and Port Alias Migration
status: in_progress
stopped_at: Phase 32 Plan 01 complete
last_updated: "2026-04-19T03:44:47Z"
last_activity: 2026-04-19
progress:
  total_phases: 33
  completed_phases: 13
  total_plans: 49
  completed_plans: 43
  percent: 88
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-04-18)

**Core value:** Developers can run a fast, lightweight, and architecturally consistent local AWS emulator without paying the complexity and resource overhead of a full cloud platform clone.
**Current focus:** Milestone v1.4 - Phase 32 Plan 01 complete

## Current Position

Phase: Phase 32 in progress
Current Phase: 32
Current Phase Name: CLI Instance Identity and Port Alias Migration
Plan: 01 complete
Status: Plan complete
Last activity: 2026-04-19
Last Activity Description: Phase 32 Plan 01 completed - instanceId promoted as canonical CLI lifecycle identity, status alias of instances, JSON payload extended

Progress: [██████████] 100%

## Performance Metrics

**Velocity:**

- Total plans completed: 9
- Average duration: -
- Total execution time: 0.0 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 12 | 0 | - | - |
| 13 | 3 | - | - |
| 14 | 0 | - | - |
| 15 | 3 | - | - |
| 16 | 0 | - | - |
| 17 | 0 | - | - |
| 18 | 0 | - | - |
| 19 | 0 | - | - |
| 20 | 0 | - | - |
| 21 | 0 | - | - |
| 22 | 0 | - | - |
| 23 | 0 | - | - |
| 24 | 0 | - | - |
| 25 | 0 | - | - |
| 26 | 0 | - | - |
| 27 | 0 | - | - |
| 28 | 0 | - | - |
| 29 | 3 | - | - |
| 30 | 0 | - | - |

**Recent Trend:**

- Last 5 plans: 23-01, 23-02, 23-03
- Trend: Phase 29 completed

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- [Phase 1]: Focus the initialization milestone exclusively on `core/`
- [Phase 1]: Use one Go binary for CLI and API runtime modes
- [Phase 1]: Prove the architecture with minimal service coverage before expanding breadth
- [Phase 2]: Keep CLI delivery thin and route runtime state through the application manager
- [Phase 2]: Keep runtime state in-memory and copy-safe for deterministic CLI output
- [Phase 03]: Kept Gin isolated inside core/internal/delivery/http and exposed only plain constructor outputs to the rest of the binary.
- [Phase 03]: Used an injected HTTP server factory in the CLI so serve starts the delivery adapter explicitly instead of owning HTTP lifecycle details.
- [Phase 03]: Kept liveness static (ok) and derived readiness solely from the application-layer runtime snapshot so the HTTP edge does not own authoritative runtime state.
- [Phase 03]: Copied services and ports into transport DTOs before rendering JSON to keep runtime metadata response shapes copy-safe.
- [Phase 03]: Derived service identity from the first path segment so orchestrator.Route stays framework-agnostic while still yielding a stable catalog namespace.
- [Phase 03]: Sorted service summaries by name and route declarations by method/path/name so catalog output stays deterministic in tests and future service onboarding.
- [Milestone v1.1]: Keep desktop app work out of scope while the `core/` surface is expanded.
- [Milestone v1.1]: Use `runtime.Snapshot` as the canonical input for CLI presentation and runtime inspection.
- [Milestone v1.1]: Define service fidelity, state, and error conventions before broadening service coverage.
- [Milestone v1.2]: Treat the Python S3 example as the behavioral reference while keeping the Go implementation modular and performance-minded.
- [Milestone v1.2]: Persist S3 data under `~/.ministack` with instance-scoped tables.
- [Milestone v1.3]: Use the official AWS S3 API docs as the scope and contract source for action support decisions.
- [Milestone v1.3]: Prioritize the most-used general-purpose S3 actions and explicitly defer specialized directory-bucket, Object Lambda, and reporting-heavy actions.
- [Milestone v1.3]: Run each contract-validation phase in three waves: map gaps, plan corrections, implement and verify.
- [Milestone v1.4]: Make AWS SDK compatibility the top-level acceptance bar for DynamoDB instead of preserving the current custom REST surface.
- [Milestone v1.4]: Persist DynamoDB state in SQLite under `~/.mildstack` with explicit connection lifecycle ownership.
- [Milestone v1.4]: Require each phase to extract request, response, and error details from the official DynamoDB API Reference before implementation.
- [Phase 32]: SetInstanceID() is called once at bootstrap; snapshot embeds it into all instances rather than computing it per-Serve call.
- [Phase 32]: instancesToRuntime() uses the live snapshot as a fallback index for instanceId when storage records are legacy port-keyed.
- [Phase 32]: status command is a thin Cobra alias (cloned command with different Use/Short) to guarantee rendering parity without a second code path.

### Roadmap Evolution

- Milestone v1.2 added: S3 emulation with `~/.ministack` persistence, bucket/object core operations, versioning, multipart uploads, governance subresources, and performance-focused modularization
- Milestone v1.3 added: contract-driven validation of the supported S3 surface, action triage from the AWS index, and wave-based correction phases for core, versioning/multipart, and governance flows
- Milestone v1.3 extended: add a final AWS-compatible S3 route-surface migration phase to remove the legacy MildStack JSON API shape
- Phase 22 added: S3 File-based
- Milestone v1.4 added: DynamoDB AWS-compatible CRUD foundation with SQLite persistence, target-based transport, table lifecycle, item CRUD, query/scan, batch APIs, transactions, and SDK verification
- Phase 30 added: CLI instance management and detached runtime UX, including the `ports` to `instances` rename, richer status output, detached serve controls, and stop/delete lifecycle commands
- Phase 31 added: instance-scoped AWS resources and future resource guardrails
- Phase 32 added: CLI instance identity and port alias migration so the CLI can use `instanceId` internally while keeping `port` as the user-facing identifier
- Phase 33 added: AWS account identity and global AWS context wrapper for shared account ID, region, partition, and ARN helpers

### Pending Todos

None yet.

### Blockers/Concerns

- The current S3 implementation still exposes MildStack-shaped routes and payloads in several places, so contract alignment may reveal broader adapter changes than the existing milestone assumed.
- Some AWS actions appear in duplicate or legacy/canonical pairs (`Lifecycle` vs `LifecycleConfiguration`, `Notification` vs `NotificationConfiguration`), so alias policy must be made explicit instead of implied.
- Specialized S3 surfaces such as directory buckets, Object Lambda, and low-demand management/reporting actions must stay deferred or this milestone will lose focus.
- The current DynamoDB service is still an in-memory exemplar with custom REST endpoints and string-only item attributes, so the jump to AWS-compatible DynamoDB will require transport, persistence, and domain-model rewrites rather than incremental handler tweaks.
- `UpdateItem`, `Query`, and transactional operations are the highest implementation-risk endpoints because they introduce expression parsing, sort-key semantics, and atomic multi-operation behavior on top of SQLite.
- SDK compatibility depends on exact error names, table status transitions, and target dispatch details, so "almost compatible" responses are likely to break real client flows.

## Deferred Items

| Category | Item | Status | Deferred At |
|----------|------|--------|-------------|
| Desktop | Electron app implementation | Deferred | 2026-04-16 |
| Service breadth | Broad AWS service coverage beyond S3 and DynamoDB | Deferred | 2026-04-16 |
| Specialized S3 | Directory buckets / S3 Express, Object Lambda, and reporting-heavy S3 actions | Deferred | 2026-04-17 |

## Session Continuity

Last session: 2026-04-19
Stopped at: Phase 32 Plan 01 complete - instanceId canonical identity, status alias, JSON payload extended
Resume file: None
