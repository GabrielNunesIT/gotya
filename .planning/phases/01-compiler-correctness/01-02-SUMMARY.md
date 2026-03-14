---
phase: 01-compiler-correctness
plan: 02
subsystem: compiler
tags: [go, tdd, yang, compiler, debug-cleanup, error-propagation, nil-safety]

# Dependency graph
requires:
  - "01-01: test stubs for MalformedAugmentPath, MalformedRefinePath, DuplicateRPCInput, NoDebugOutput"
provides:
  - "compiler.go with zero fmt.Printf calls (COMP-06)"
  - "compiler.go with all five AddChild errors propagated into c.errors (COMP-05)"
  - "Four GREEN tests: NoDebugOutput, MalformedAugmentPath, MalformedRefinePath, DuplicateRPCInput (COMP-01)"
affects:
  - 01-03
  - 01-04

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "os.Pipe() + io.ReadAll() pattern for capturing stdout/stderr in tests without forking"
    - "compile() package-level helper in compiler_test.go: lexer -> parser -> compiler.New(nil).Compile()"
    - "if err := node.AddChild(x); err != nil { c.errors = append(c.errors, err.Error()) } — error-propagating AddChild pattern"

key-files:
  created: []
  modified:
    - compiler/compiler.go
    - compiler/compiler_test.go

key-decisions:
  - "fmt import retained — fmt.Sprintf and fmt.Errorf still used throughout compiler.go; only two Printf debug lines removed"
  - "compile() helper uses compiler.New(nil) — no feature options, matches existing test patterns in file"
  - "TestCompiler_NoDebugOutput uses t.Parallel() safe os.Pipe redirect — restores os.Stdout/Stderr after each capture"

requirements-completed:
  - COMP-01
  - COMP-05
  - COMP-06

# Metrics
duration: 5min
completed: 2026-03-14
---

# Phase 1 Plan 02: Debug Printf Removal, AddChild Propagation, and Four Test Implementations Summary

**Debug printfs removed from compiler.go, all five AddChild errors propagated, and four compiler correctness tests turned GREEN**

## Performance

- **Duration:** 5 min
- **Started:** 2026-03-14T17:10:00Z
- **Completed:** 2026-03-14T17:15:00Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- Removed `fmt.Printf` at `findNode` nil-return site (line 778) — COMP-06
- Removed `fmt.Printf` at grouping-not-found site in `resolveUses` (line 930) — COMP-06
- Replaced all five `_ = X.AddChild(Y)` patterns with error-propagating form — COMP-05
  - Sites: rpcNode input (415), rpcNode output (418), actionNode input (426), actionNode output (429), caseNode shorthand (728)
- Added `compile()` helper and `io`/`os` imports to compiler_test.go
- Implemented `TestCompiler_NoDebugOutput` — captures stdout+stderr via os.Pipe, asserts both empty for valid and invalid YANG
- Implemented `TestCompiler_MalformedAugmentPath` — asserts error for augment to nonexistent path, no panic
- Implemented `TestCompiler_MalformedRefinePath` — asserts error for refine to nonexistent leaf, no panic
- Implemented `TestCompiler_DuplicateRPCInput` — asserts error for duplicate `input` in rpc, no panic
- Three remaining stubs (CircularTypedef, UnresolvableAugment, MaxErrors) remain RED as expected

## Task Commits

1. **Task 1: Remove debug printfs and fix AddChild error propagation** - `4bbb86c` (fix)
2. **Task 2: Implement four compiler correctness tests** - `609212f` (test)

## Files Created/Modified

- `/home/user/repos/gotya/compiler/compiler.go` - Removed 2 debug printf lines; expanded 5 AddChild sites to propagate errors
- `/home/user/repos/gotya/compiler/compiler_test.go` - Added io/os imports, compile() helper, 4 test implementations (~89 lines)

## Decisions Made

- `fmt` import retained in compiler.go — `fmt.Sprintf` and `fmt.Errorf` remain in heavy use throughout; only the two `Printf` debug lines were removed
- `compile()` helper uses `compiler.New(nil)` matching existing test patterns; no feature options needed for error-path tests
- `TestCompiler_NoDebugOutput` uses `os.Pipe()` redirect pattern — avoids subprocess overhead while correctly capturing writes to `os.Stdout`/`os.Stderr`

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None. All four tests passed GREEN on first run after implementation.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Three remaining RED stubs: `TestCompiler_CircularTypedef`, `TestCompiler_UnresolvableAugment`, `TestCompiler_MaxErrors`
- Plan 03 should target: `TestCompiler_CircularTypedef` and `TestCompiler_UnresolvableAugment`
- Plan 04 should target: `TestCompiler_MaxErrors`

---
*Phase: 01-compiler-correctness*
*Completed: 2026-03-14*
