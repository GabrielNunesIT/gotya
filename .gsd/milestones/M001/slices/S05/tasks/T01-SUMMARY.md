---
id: T01
parent: S05
milestone: M001
provides:
  - "RED test baseline in gotya_test.go: 6 failing test stubs covering all Phase 5 API contracts"
  - "Black-box external test package (package gotya_test) that imports only the gotya package"
requires: []
affects: []
key_files: []
key_decisions: []
patterns_established: []
observability_surfaces: []
drill_down_paths: []
duration: 3min
verification_result: passed
completed_at: 2026-03-15
blocker_discovered: false
---
# T01: 05-public-api-stabilization 01

**# Phase 5 Plan 01: Public API Test Stubs Summary**

## What Happened

# Phase 5 Plan 01: Public API Test Stubs Summary

**6 failing RED test stubs in gotya_test.go covering ParseError type, multiple diagnostics, opaque ASTModule/Module return types, and ParseFile file-filled diagnostics — zero internal package imports**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-15T16:13:28Z
- **Completed:** 2026-03-15T16:16:00Z
- **Tasks:** 1
- **Files modified:** 1

## Accomplishments
- Created `gotya_test.go` as `package gotya_test` with 6 test functions, all producing compile-time RED failures
- Tests confirm the three API contracts: `ParseError` type (API-01/02), opaque return types (API-03), and diagnostic File field (API-02)
- File imports only `github.com/gotya/gotya`, `errors`, `os`, `testing`, and `testify/require` — no `ast`, `schema`, or `compiler` packages
- Compilation fails with `undefined: gotya.ParseError` and related errors — establishing the RED baseline for plans 02 and 03

## Task Commits

1. **Task 1: RED test stubs** - `98a2237` (test)

**Plan metadata:** committed with docs commit below

## Files Created/Modified
- `/home/user/repos/gotya/gotya_test.go` - 6 black-box test stubs for all Phase 5 API behaviors

## Decisions Made
- `Schema()` method accessor chosen for `TestCompile_OpaqueReturn` — callers call `compiled[0].Schema()` without importing the schema package; this is the minimal verifiable interface at compile time
- `t.TempDir()` used for temp file in `TestParseFile_FilledDiagnostic` — idiomatic Go test cleanup, no external helpers needed
- Two-case `TestParseFile_FilledDiagnostic`: nonexistent file (I/O error, NOT ParseError) + invalid YANG file (ParseError WITH File field) — covers both sides of the API contract

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None — compilation failed precisely as designed. All 6 undefined symbols confirmed in error output.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- RED baseline established: `go test .` fails with `undefined: gotya.ParseError` and related errors
- Plan 05-02 turns GREEN: implement `ParseError`, `Diagnostic`, opaque `ASTModule`, opaque `Module` in `gotya.go`
- Plan 05-03 turns remaining: godoc cleanup (API-04)

---
*Phase: 05-public-api-stabilization*
*Completed: 2026-03-15*
