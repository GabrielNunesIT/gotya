---
phase: 03-go-generator
plan: "04"
subsystem: generator
tags: [yang, deviation, go-generator, schema, rfc7950]

# Dependency graph
requires:
  - phase: 03-go-generator/03-03
    provides: "AnyData/AnyXML, RPC/Action/Notification generation, generator_deviation_test.go stubs"
provides:
  - "applyDeviations(mod) pre-pass function in generator.go — mutates node tree before code generation"
  - "resolveDeviationPath(mod, path) helper — walks prefix:name paths in mod.Nodes"
  - "GenerateDevice deviation pre-pass loop — called before Config+State generation loops"
  - "TestDeviationNotSupported, TestDeviationReplace, TestDeviationAddDelete all GREEN"
affects:
  - 03-go-generator/03-05
  - 04-proto-generator

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Deviation pre-pass: applyDeviations runs before any generateNode call — RFC 7950 §7.12 ordering"
    - "Path resolution: strip prefix:name segments before map lookup in Children"
    - "Lenient path resolution: skip unresolvable deviation paths with no error (v1 tolerance)"
    - "ast.BaseNode used as concrete Statement impl in tests — struct literal construction"

key-files:
  created: []
  modified:
    - generator/golang/generator.go
    - generator/golang/generator_deviation_test.go

key-decisions:
  - "applyDeviations placed before toCamelCaseTitle utility functions at bottom of generator.go — consistent with existing file structure"
  - "dev.Name() used (not dev.NodeName field) — BaseNode.Name() method returns the NodeName field"
  - "Lenient path resolution: unresolvable paths skipped silently — v1 tolerance for incomplete module graphs"
  - "deviate add/delete are no-ops for v1 — constraints (min/max-elements, must, unique) not emitted as Go code"
  - "ast.BaseNode concrete struct used for Statement construction in tests — available from ast package, no mocking needed"

patterns-established:
  - "Deviation pre-pass pattern: applyDeviations(mod) called for each module in a pre-pass loop before the treeType generation loops"

requirements-completed:
  - GOGEN-04

# Metrics
duration: 2min
completed: 2026-03-14
---

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
