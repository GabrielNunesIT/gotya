---
id: T03
parent: S03
milestone: M001
provides:
  - RPC/Action/Notification Go struct generation (XxxInput, XxxOutput, XxxNotification)
  - generateRPCStruct helper for emitting operation data structures
  - Post-loop pass in GenerateDevice for module-level RPC/Notification emission
  - Action emission from within generateStruct for container-attached operations
requires: []
affects: []
key_files: []
key_decisions: []
patterns_established: []
observability_surfaces: []
drill_down_paths: []
duration: 5min
verification_result: passed
completed_at: 2026-03-14
blocker_discovered: false
---
# T03: 03-go-generator 03

**# Phase 03 Plan 03: RPC/Action/Notification Struct Generation Summary**

## What Happened

# Phase 03 Plan 03: RPC/Action/Notification Struct Generation Summary

**YANG rpc/action/notification statements emit typed XxxInput, XxxOutput, XxxNotification Go structs via post-loop pass and container-child Action pass — TestRPC, TestAction, TestNotification GREEN**

## Performance

- **Duration:** 5 min
- **Started:** 2026-03-14T21:57:24Z
- **Completed:** 2026-03-14T21:57:54Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- `generateRPCStruct` helper emits top-level typed structs for RPC/Action/Notification children with sorted fields
- Post-loop pass in `GenerateDevice` iterates module nodes for `*schema.RPC` and `*schema.Notification`, emitting their Input/Output/Notification structs once outside the Config+State duplication loop
- `generateStruct` has a dedicated Action pass that emits `*schema.Action` structs when processing container/list children (RFC 7950 §7.15 — actions attach to data nodes, not to module root)
- `generateField` and the module-level struct field loop skip RPC/Action/Notification to prevent bogus `*string` placeholder fields
- `hasValidNodes` returns true for RPC/Action/Notification so containers that contain only Actions are still visited and processed

## Task Commits

1. **Task 1: Patch generator.go — RPC/Action/Notification struct emission** - `ced3c77` (feat)
2. **Task 2: Update RPC test stubs to real assertions** - `91d8ec6` (feat)

**Plan metadata:** (docs commit pending)

## Files Created/Modified

- `/home/user/repos/gotya/generator/golang/generator.go` - Added hasValidNodes cases, generateNode RPC/Action/Notification cases, generateRPCStruct helper, post-loop pass in GenerateDevice, Action pass in generateStruct, generateField skip
- `/home/user/repos/gotya/generator/golang/generator_rpc_test.go` - Replaced t.Fatal stubs with real assertions for TestRPC, TestAction, TestNotification

## Decisions Made

- `hasValidNodes` returns true for RPC/Action/Notification. This is needed so that containers with only Action children are still visited (they pass the `hasValid` check). The side effect is that modules with only RPCs generate empty module-level structs (`TestRpcState {}`), which is valid Go. The alternative would require a separate `hasActionNodes` predicate.
- `generateField` and the module-level inline field loop both skip RPC/Action/Notification. Having two skip sites is necessary because module struct field generation uses inline code (not `generateField`) — a known design debt in the generator.
- Action handling uses a dedicated loop in `generateStruct` rather than relying on `generateNode` recursion through `validNonChoiceNodes`, because Actions don't appear in `getFlatDataNodes` results (they're not data tree nodes). The separate loop scans the raw `children` map.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Module-level struct emitted bogus `*string` field for RPC nodes**
- **Found during:** Task 1 (generator.go implementation)
- **Issue:** The inline field-generation loop in `GenerateDevice` had no case for RPC/Action/Notification, so they fell through to `goType := "*string"`, generating e.g. `ResetCounters *string` in the module struct
- **Fix:** Added explicit `continue` for `*schema.RPC, *schema.Action, *schema.Notification` in the module struct field loop
- **Files modified:** generator/golang/generator.go
- **Verification:** Output no longer contains bogus `ResetCounters *string` field; checked via manual run before test update
- **Committed in:** ced3c77 (Task 1 commit)

**2. [Rule 1 - Bug] Action inside container not processed when container has no other data nodes**
- **Found during:** Task 1 (testing Action generation)
- **Issue:** `hasValidNodes` originally only returned true for data nodes. A container with only an Action child returned false, causing the container (and its Action) to be skipped entirely. No `PingInput`/`PingOutput` was generated.
- **Fix:** Made `hasValidNodes` return true for `*schema.Action` (and other operation nodes) so the container is visited. Added `generateField` skip so no data field is emitted for the Action.
- **Files modified:** generator/golang/generator.go
- **Verification:** `TestAction` generates `TestActionPingInput` and `TestActionPingOutput`; both contain the substring `PingInput`/`PingOutput` that the test asserts
- **Committed in:** ced3c77 (Task 1 commit)

---

**Total deviations:** 2 auto-fixed (2 Rule 1 bugs found during implementation)
**Impact on plan:** Both fixes required for correct output. No scope creep.

## Issues Encountered

None - all issues were identified during implementation and fixed inline.

## Next Phase Readiness

- GOGEN-03 complete: RPC/Action/Notification struct generation working and tested
- Phase 03-go-generator: remaining plans (if any) can build on this foundation
- Phase 04-protobuf-generator: `*schema.RPC`, `*schema.Action`, `*schema.Notification` schema types are now proven to be accessible and usable for code generation — PBGEN-03 service/RPC proto generation can proceed

---
*Phase: 03-go-generator*
*Completed: 2026-03-14*
