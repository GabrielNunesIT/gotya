---
phase: 01-compiler-correctness
plan: 04
subsystem: compiler
tags: [go, tdd, yang, compiler, circular-typedef, augment-loop-cap, max-errors, error-bounding]

# Dependency graph
requires:
  - "01-02: addError sites, getType(), augment loop in compiler.go"
  - "01-03: TestLoader_CircularImport stub"
provides:
  - "getType() with visited map[string]bool cycle detection (COMP-02)"
  - "augment loop with maxIter cap preventing infinite spin (COMP-04)"
  - "Options.MaxErrors + addError() helper bounding error accumulation (COMP-07)"
  - "All 8 Phase 1 tests GREEN"
affects:
  - 02-01

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "visited map[string]bool passed through recursive getType() calls — initialized lazily inside function on first use; defer delete cleanup"
    - "maxIter = len(augments)+1 computed before loop; iter counter checked at top of each iteration"
    - "addError() helper: check-before-append with truncation sentinel; all error append sites use helper"

key-files:
  created: []
  modified:
    - compiler/compiler.go
    - compiler/compiler_test.go

key-decisions:
  - "addError() internal body retains direct c.errors = append() — only the two lines inside the helper body; all external sites use c.addError()"
  - "union member recursion in getType passes nil (not visited) — union type members are not typedef chains and should not participate in cycle detection"
  - "defer delete(visited, td.Name) used for cleanup — ensures visited entry is cleared on all return paths"

requirements-completed:
  - COMP-02
  - COMP-04
  - COMP-07

# Metrics
duration: 3min
completed: 2026-03-14
---

# Phase 1 Plan 04: Circular Typedef Detection, Augment Loop Cap, and MaxErrors Summary

**getType() visited-set cycle detection, augment loop maxIter cap, and addError() error bounding implemented; all 8 Phase 1 tests GREEN**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-14T17:10:38Z
- **Completed:** 2026-03-14T17:13:26Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- Added `maxErrors int` field to `Compiler` struct, initialized to 100 in `New()`
- Added `MaxErrors int` field to `Options` struct (0 = use default 100)
- Added `addError(msg string)` helper method enforcing `maxErrors` limit with truncation sentinel message
- Replaced all ~26 external `c.errors = append(c.errors, ...)` sites with `c.addError(...)` — COMP-07
- Changed `getType()` signature to accept `visited map[string]bool` as second parameter
- Added cycle detection inside `getType()` at typedef resolution branch — COMP-02
- Updated all external `getType()` call sites (2 leaf/leaf-list sites) to pass `nil`
- Union member recursive call passes `nil` (not visited) — correct per plan
- Added `maxIter := len(c.augments) + 1` and `iter` counter guard to augment convergence loop — COMP-04
- Implemented `TestCompiler_CircularTypedef` — typedef A→B→A returns "circular typedef" error
- Implemented `TestCompiler_UnresolvableAugment` — augment to nonexistent path returns "augment target not found"
- Implemented `TestCompiler_MaxErrors` — 110-key list generates >100 errors; output bounded with "compilation stopped" truncation
- All 8 Phase 1 tests GREEN; full test suite green (zero regressions)

## Task Commits

1. **Task 1: Add MaxErrors, addError helper, getType visited set, augment loop cap** - `4925855` (feat)
2. **Task 2: Implement CircularTypedef, UnresolvableAugment, MaxErrors tests** - `6a0a61a` (test)

## Files Created/Modified

- `/home/user/repos/gotya/compiler/compiler.go` - Added maxErrors field, addError() helper, visited-set in getType(), augment loop cap; all error append sites migrated to addError()
- `/home/user/repos/gotya/compiler/compiler_test.go` - Added fmt/strings imports; replaced 3 t.Fatal stubs with real test implementations (~40 lines)

## Decisions Made

- `addError()` internal body retains two direct `c.errors = append()` calls — only the helper itself may bypass the helper to avoid infinite recursion
- Union member recursion passes `nil` to `getType()` — union members are independent types, not typedef chains; visited state must not carry over
- `defer delete(visited, td.Name)` used for cleanup — ensures entry is cleared on any return path

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None. All three tests passed GREEN on first run after implementation.

## User Setup Required

None.

## Next Phase Readiness

- All 8 Phase 1 compiler correctness requirements complete: COMP-01 through COMP-07
- Phase 2 can begin: code generation (Go structs, Protobuf definitions)

---
*Phase: 01-compiler-correctness*
*Completed: 2026-03-14*
