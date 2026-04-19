---
phase: 32-cli-instance-identity-and-port-alias-migration
plan: 01
subsystem: cli
tags: [go, cobra, runtime, lifecycle, identity, migration]

# Dependency graph
requires:
  - phase: 31-instance-scoped-aws-resources-and-future-resource-guardrails
    provides: instanceId as canonical bootstrap identity for AWS-backed resources

provides:
  - InstanceID field in runtime.Instance and Manager.SetInstanceID() method
  - SaveActiveInstanceWithID and SaveSavedInstanceWithID storage helpers
  - Legacy port-keyed records still load and InstanceID falls back from live snapshot
  - NewStatusCommand as a thin alias delegating to NewInstancesCommand
  - instancesToRuntime() that merges storage summaries with live snapshot identity
  - instancePayload JSON schema extended with instanceId field
  - Full regression coverage for alias parity and canonical identity in JSON output

affects: [33-aws-account-identity, desktop-app-instances-view]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Manager.SetInstanceID(): caller sets canonical identity once at bootstrap, snapshot embeds it into every Instance"
    - "instancesToRuntime() fallback: storage summaries without instanceId inherit from live snapshot by port"
    - "Thin alias pattern: NewStatusCommand wraps NewInstancesCommand with different Use/Short, no separate rendering path"
    - "saveInstanceWithID(): single internal helper accepts instanceID string, empty string preserves compatibility"

key-files:
  created: []
  modified:
    - core/internal/application/runtime/manager.go
    - core/internal/application/runtime/manager_test.go
    - core/internal/delivery/cli/storage.go
    - core/internal/delivery/cli/storage_test.go
    - core/internal/delivery/cli/status.go
    - core/internal/delivery/cli/presenter.go
    - core/internal/delivery/cli/output.go
    - core/internal/delivery/cli/root.go
    - core/internal/delivery/cli/root_test.go
    - core/internal/delivery/cli/presenter_test.go
    - core/internal/delivery/cli/commands_test.go
    - core/cmd/mildstack/main.go

key-decisions:
  - "InstanceID is set once on Manager via SetInstanceID() at bootstrap; snapshot embeds it into all instances rather than computing it per-Serve call"
  - "instancesToRuntime() uses live snapshot as fallback index for instanceId when storage records are legacy port-keyed"
  - "status command is a thin Cobra alias (cloned command with different Use) to guarantee rendering parity without a second code path"
  - "saveInstanceWithID() is the single internal helper; SaveActiveInstance/SaveSavedInstance pass empty string to preserve backward compatibility"
  - "instanceId is omitempty in JSON so legacy records without an id produce valid output during the migration window"

patterns-established:
  - "canonical identity is seeded at bootstrap and flows down through snapshot copies, not computed at presentation time"
  - "alias commands are created by cloning the primary command struct and changing Use/Short, not by wrapping the handler"
  - "storage summaries fall back to live snapshot identity by port when the on-disk record predates the instanceId field"

requirements-completed: []

# Metrics
duration: 6min
completed: 2026-04-19
---

# Phase 32 Plan 01: CLI Instance Identity and Port Alias Migration Summary

**instanceId promoted as canonical CLI lifecycle identity via Manager.SetInstanceID(), migration-safe storage helpers, a status alias delegating to instances, and JSON payload extended with instanceId field**

## Performance

- **Duration:** 6 min
- **Started:** 2026-04-19T03:38:47Z
- **Completed:** 2026-04-19T03:44:47Z
- **Tasks:** 3
- **Files modified:** 12

## Accomplishments

- `runtime.Instance` and `Manager` gained `InstanceID` field and `SetInstanceID()` method; snapshots now embed canonical identity into every instance copy
- CLI storage extended with `SaveActiveInstanceWithID` / `SaveSavedInstanceWithID`; `instanceRecord` and `instanceSummary` carry `instanceId,omitempty`; legacy port-keyed records without the field still load and fall back to live snapshot identity via port index
- `NewStatusCommand` implemented as a thin Cobra alias of `NewInstancesCommand` (same handler, different `Use`/`Short`), registered in `Commands` struct and wired in `main.go`
- `instancePayload` JSON schema extended with `instanceId` field; presenter payload clone helpers propagate it
- `instancesToRuntime()` updated to accept live snapshot instances as fallback source for `instanceId` when storage records are legacy
- Regression suite covers alias output parity (human and JSON), canonical identity in JSON payload, and copy-safe snapshot behavior after identity migration

## Task Commits

1. **RED (Task 1): add failing tests for instanceId in runtime snapshots and storage** - `fad62a27` (test)
2. **Task 1: Promote instanceId into runtime snapshots and storage migration helpers** - `000eecae` (feat)
3. **RED (Task 2): add failing tests for status alias and instanceId in presenter payload** - `64e5fab2` (test)
4. **Task 2: Make status a thin alias of instances and extend payload schema** - `44cc7db4` (feat)
5. **Task 3: Extend regression coverage for lifecycle commands and binary wiring** - `7c8873d8` (feat)

## Files Created/Modified

- `core/internal/application/runtime/manager.go` - Added `InstanceID` to `Instance`, `instanceID` field to `Manager`, `SetInstanceID()` method, updated `runningInstances()` signature
- `core/internal/application/runtime/manager_test.go` - Added `TestManagerInstanceCarriesInstanceID` and `TestNewWithPortsSeedsInstanceIDFromRegisteredIdentity`
- `core/internal/delivery/cli/storage.go` - Added `InstanceID` to `instanceRecord`/`instanceSummary`, `saveInstanceWithID()` helper, `SaveActiveInstanceWithID`, `SaveSavedInstanceWithID`; updated `LoadInstances()` to propagate `InstanceID` with active-record-wins merge
- `core/internal/delivery/cli/storage_test.go` - Added `TestStorageInstanceSummaryCarriesInstanceID` and `TestStorageLegacyPortKeyedRecordLoadsAndPresentsWithInstanceID`
- `core/internal/delivery/cli/status.go` - Added `NewStatusCommand()` alias; updated `instancesToRuntime()` to accept live snapshot instances for `instanceId` fallback
- `core/internal/delivery/cli/presenter.go` - Updated `cloneInstances()` and `cloneInstancesPayload()` to copy `InstanceID`
- `core/internal/delivery/cli/output.go` - Added `InstanceID string` with `json:"instanceId,omitempty"` to `instancePayload`
- `core/internal/delivery/cli/root.go` - Added `Status *cobra.Command` to `Commands` struct; registered in `NewRootCommand`
- `core/internal/delivery/cli/root_test.go` - Added `TestNewRootCommandRegistersStatusAlias`
- `core/internal/delivery/cli/presenter_test.go` - Added `TestPresenterStatusPayloadIncludesInstanceID` with copy-safety assertion
- `core/internal/delivery/cli/commands_test.go` - Added `TestStatusAliasMatchesInstancesOutput`, `TestStatusAliasJSONMatchesInstancesJSON`, `TestCommandsServeInstancesJSONIncludesInstanceID`; wired `Status` into `newTestCommand()`; updated `TestCommandsServeInstancesJSON` to assert `instanceId` non-empty
- `core/cmd/mildstack/main.go` - Added `manager.SetInstanceID(instanceID)` after manager creation; added `Status: cli.NewStatusCommand(manager, storage)` to commands

## Decisions Made

- `SetInstanceID()` is called once at bootstrap in `main.go` rather than threading the id through every `Serve` call — keeps the identity model simple and the `Manager` interface stable
- `instancesToRuntime()` uses the live snapshot as a fallback index by port — avoids requiring every operator to immediately upgrade their storage records while still surfacing the canonical id when the manager knows it
- `status` alias is a Cobra command clone (copy of the `instances` command with `Use`/`Short` overwritten) rather than a `cobra.Command.AddCommand` alias — this guarantees identical flag parsing, output, and empty-state handling without a second implementation branch

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed TestManagerInstanceCarriesInstanceID to require SetInstanceID before snapshot**
- **Found during:** Task 1 (GREEN phase)
- **Issue:** Test created manager without calling `SetInstanceID` but expected non-empty `InstanceID` — design requires explicit seeding before snapshot
- **Fix:** Updated test to call `manager.SetInstanceID("test-instance-abc")` before `Serve` and use exact string assertion instead of non-empty check
- **Files modified:** `core/internal/application/runtime/manager_test.go`
- **Committed in:** `000eecae` (Task 1 feat commit)

**2. [Rule 1 - Bug] Fixed instancesToRuntime to fall back to live snapshot for instanceId**
- **Found during:** Task 3 (TestCommandsServeInstancesJSON and TestCommandsServeInstancesJSONIncludesInstanceID failing)
- **Issue:** `commandServerStub.Start()` calls `SaveActiveInstance(port)` without an `instanceId`, so storage summaries loaded by `NewInstancesCommand` have empty `InstanceID`. The manager snapshot carries the id but it was discarded when the snapshot instances were overwritten with storage data
- **Fix:** Updated `instancesToRuntime()` to accept live snapshot instances and build a `port -> InstanceID` fallback index; storage summaries without `instanceId` inherit from the live snapshot
- **Files modified:** `core/internal/delivery/cli/status.go`
- **Committed in:** `7c8873d8` (Task 3 feat commit)

---

**Total deviations:** 2 auto-fixed (both Rule 1 - Bug)
**Impact on plan:** Both fixes were necessary for correctness. No scope creep — fixes remained within Task 1 and Task 3 boundaries.

## Issues Encountered

None beyond the two auto-fixed bugs documented above.

## Known Stubs

None - all instanceId fields are wired from real bootstrap identity.

## Threat Flags

No new network endpoints, auth paths, file access patterns, or schema changes at trust boundaries introduced beyond what the plan's threat model covers (T-32-01 through T-32-03).

## Next Phase Readiness

- `instanceId` is canonical in runtime snapshots and CLI storage records; Phase 33 (AWS account identity) can read it from snapshots without additional plumbing
- `status` alias is registered and covered by parity tests; no alias drift risk
- `port` remains in all human-facing and JSON surfaces as compatibility locator throughout the migration window
- Legacy port-keyed records load and fall back safely; no forced migration required before Phase 33

---
*Phase: 32-cli-instance-identity-and-port-alias-migration*
*Completed: 2026-04-19*
