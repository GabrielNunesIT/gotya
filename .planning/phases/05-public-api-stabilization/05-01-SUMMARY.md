---
phase: 05-public-api-stabilization
plan: 01
subsystem: api
tags: [gotya, tdd, red, public-api, ParseError, ASTModule, Module]

# Dependency graph
requires:
  - phase: 04-protobuf-generator
    provides: "Completed schema/AST/compiler infrastructure that tests exercise via gotya.go facade"
provides:
  - "RED test baseline in gotya_test.go: 6 failing test stubs covering all Phase 5 API contracts"
  - "Black-box external test package (package gotya_test) that imports only the gotya package"
affects: [05-02, 05-03]

# Tech tracking
tech-stack:
  added: []
  patterns: [tdd-red-first, external-test-package, errors-As-assertion]

key-files:
  created: [gotya_test.go]
  modified: []

key-decisions:
  - "gotya_test.go uses package gotya_test (external) — callers cannot import ast/schema/compiler directly"
  - "t.TempDir() + os.WriteFile used for ParseFile temp file fixture — no helper packages needed"
  - "Schema() accessor pattern chosen for TestCompile_OpaqueReturn — method call verifiable at compile time without schema import"
  - "Blank ParseError.File case (Parse with no path) not tested here — ParseFile fills it, Parse does not"

patterns-established:
  - "External test package (package foo_test) verifies API encapsulation — internal types cannot leak"
  - "errors.As(err, &parseErr) pattern as primary diagnostic access mechanism"

requirements-completed: [API-01, API-02, API-03, API-04]

# Metrics
duration: 3min
completed: 2026-03-15
---

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
