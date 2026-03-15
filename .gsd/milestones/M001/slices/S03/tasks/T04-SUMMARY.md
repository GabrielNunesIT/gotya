---
id: T04
parent: S03
milestone: M001
provides:
  - "applyDeviations(mod) pre-pass function in generator.go — mutates node tree before code generation"
  - "resolveDeviationPath(mod, path) helper — walks prefix:name paths in mod.Nodes"
  - "GenerateDevice deviation pre-pass loop — called before Config+State generation loops"
  - "TestDeviationNotSupported, TestDeviationReplace, TestDeviationAddDelete all GREEN"
requires: []
affects: []
key_files: []
key_decisions: []
patterns_established: []
observability_surfaces: []
drill_down_paths: []
duration: 2min
verification_result: passed
completed_at: 2026-03-14
blocker_discovered: false
---
# T04: 03-go-generator 04

**# Phase 03 Plan 04: Deviation Pre-Pass Summary**

## What Happened

# Phase 03 Plan 04: Deviation Pre-Pass Summary

**applyDeviations pre-pass in GenerateDevice that removes or modifies schema nodes before Go struct generation, driven by YANG deviation statements (RFC 7950 §7.12)**

## Performance

- **Duration:** 2 min
- **Started:** 2026-03-14T22:03:52Z
- **Completed:** 2026-03-14T22:05:52Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- `applyDeviations(mod *schema.Module)` function iterates `mod.Deviations`, resolves paths, and mutates node tree in place
- `resolveDeviationPath` walks prefix-stripped path segments through `mod.Nodes` and `Children` maps
- `GenerateDevice` calls `applyDeviations` for each module before the `Config`/`State` generation loops
- All three deviation tests turned GREEN: `TestDeviationNotSupported`, `TestDeviationReplace`, `TestDeviationAddDelete`

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement applyDeviations pre-pass in generator.go** - `24773c0` (feat)
2. **Task 2: Update deviation test stubs to real assertions** - `be8be34` (test)

**Plan metadata:** (docs commit follows)

## Files Created/Modified

- `generator/golang/generator.go` - Added `applyDeviations` and `resolveDeviationPath` functions; added pre-pass loop in `GenerateDevice`
- `generator/golang/generator_deviation_test.go` - Replaced three `t.Fatal("not yet implemented")` stubs with real assertions using manually constructed `schema.Deviation` and `ast.BaseNode` objects

## Decisions Made

- `dev.Name()` used instead of `dev.NodeName` field — `BaseNode.Name()` is the correct method accessor for the name field
- Path resolution is lenient: unresolvable deviation paths are skipped with no error for v1 compatibility
- `deviate add` and `deviate delete` are no-ops (v1 scope) since the generator does not emit constraints as Go code
- `ast.BaseNode` struct literal used directly in tests for constructing `ast.Statement` values — no mock needed since `BaseNode` is a concrete exported type

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed `dev.NodeName()` call — should be `dev.Name()`**
- **Found during:** Task 1 (build verification)
- **Issue:** Plan specified `dev.NodeName()` but `NodeName` is a string field on `BaseNode`, not a method; `Name()` is the method that returns it
- **Fix:** Changed `dev.NodeName()` to `dev.Name()` in `applyDeviations`
- **Files modified:** `generator/golang/generator.go`
- **Verification:** `go build ./generator/golang/...` passes
- **Committed in:** `24773c0` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 - bug in plan's method call)
**Impact on plan:** Single-line fix with no scope change. All plan requirements satisfied.

## Issues Encountered

None beyond the `dev.NodeName()` vs `dev.Name()` fix above.

## Next Phase Readiness

- Deviation pre-pass is complete and tested — vendor YANG modules using `deviate not-supported` or `deviate replace` will produce correct Go
- `TestIdentityref` and `TestIdentityrefCrossModule` remain RED (pre-existing stubs from plan 03-03, not in scope for this plan)
- Corpus test failures are pre-existing and unrelated to deviation work

---
*Phase: 03-go-generator*
*Completed: 2026-03-14*
