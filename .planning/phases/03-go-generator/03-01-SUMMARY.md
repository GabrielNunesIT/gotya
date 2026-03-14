---
phase: 03-go-generator
plan: 01
subsystem: testing
tags: [golang, generator, tdd, anydata, anyxml, identityref, rpc, action, notification, deviation]

# Dependency graph
requires:
  - phase: 02-testing-infrastructure
    provides: GenerateDevice format.Source integration; golang_test package pattern established
provides:
  - 10 failing test stubs (RED) across 4 test files covering all Go generator gaps
  - Precise -run filter targets matching VALIDATION.md sampling map for plans 02-05
  - Stub contracts for GOGEN-01 through GOGEN-04
affects:
  - 03-go-generator plans 02, 03, 04, 05 (each must GREEN the stubs created here)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Each stub test is self-contained (no shared helpers) — lexer.New → parser.New → p.ParseModule() → compiler.New(nil).Compile() → GenerateDevice → t.Fatal"
    - "Cross-module or cross-deviation stubs use manually constructed schema.Module structs when compiler lacks loader-based resolution"

key-files:
  created:
    - generator/golang/generator_anydata_test.go
    - generator/golang/generator_identityref_test.go
    - generator/golang/generator_rpc_test.go
    - generator/golang/generator_deviation_test.go
  modified: []

key-decisions:
  - "TestIdentityrefCrossModule uses manually constructed schema.Module — compiler cannot resolve prefixed bases (bt:base-identity) without a loader at stub stage; acceptable per plan"
  - "Deviation tests (all three) use manually constructed schema.Module — cross-module deviation application not supported by compiler; represents post-deviation schema state directly"
  - "t.Fatal(not yet implemented) is the only stub body — all assertions deferred to implementation plans 02-05"

patterns-established:
  - "Stub-only pattern: each test runs the full parse→compile→GenerateDevice pipeline then t.Fatal immediately — validates infrastructure works before asserting behavior"
  - "Manual schema construction for cross-module scenarios: avoids import resolution complexity, acceptable at stub stage"

requirements-completed: [GOGEN-01, GOGEN-02, GOGEN-03, GOGEN-04]

# Metrics
duration: 12min
completed: 2026-03-14
---

# Phase 3 Plan 01: Go Generator Failing Test Stubs Summary

**10 RED test stubs across anydata/anyxml, identityref, rpc/action/notification, and deviation covering all Go generator gaps (GOGEN-01 through GOGEN-04)**

## Performance

- **Duration:** 12 min
- **Started:** 2026-03-14T21:49:07Z
- **Completed:** 2026-03-14T22:01:00Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Created 4 test files under generator/golang/ with 10 failing stubs total
- All stubs compile cleanly and fail with t.Fatal("not yet implemented") — unambiguous RED signal
- All pre-existing generator tests continue to pass (TestGoGenerator_Generate, TestGoGenerator_FormatValidation, TestGenerateDeviceFromTestAssets)
- Test names match VALIDATION.md sampling map exactly for precise -run filter targeting in plans 02-05

## Task Commits

Each task was committed atomically:

1. **Task 1: Write failing stubs for GOGEN-02 and GOGEN-01** - `71067cd` (test)
2. **Task 2: Write failing stubs for GOGEN-03 and GOGEN-04** - `d7746de` (test)

## Files Created/Modified
- `generator/golang/generator_anydata_test.go` - TestAnyData and TestAnyXML stubs (GOGEN-02)
- `generator/golang/generator_identityref_test.go` - TestIdentityref and TestIdentityrefCrossModule stubs (GOGEN-01)
- `generator/golang/generator_rpc_test.go` - TestRPC, TestAction, TestNotification stubs (GOGEN-03)
- `generator/golang/generator_deviation_test.go` - TestDeviationNotSupported, TestDeviationReplace, TestDeviationAddDelete stubs (GOGEN-04)

## Decisions Made
- TestIdentityrefCrossModule uses manually constructed schema.Module because the compiler's identityref resolution requires a loader when the base identity is in an imported module (prefixed as bt:base-identity). The plan explicitly permits this approach at stub stage.
- All three deviation tests use manually constructed schema.Module objects representing the post-deviation schema state. The compiler's applyDeviations() works within a single module; cross-module deviation application would require a loader-aware multi-pass compile. Manual construction is acceptable for stub stage.

## Deviations from Plan

None — plan executed exactly as written. The plan explicitly anticipated the cross-module compiler limitation and authorized manual schema construction as the stub-stage approach.

## Issues Encountered
- First attempt at TestIdentityrefCrossModule compiled two modules with inline YANG using prefixed identityref base (bt:base-identity). The compiler returned "unknown prefix bt" because import resolution requires a ModuleLoader. Resolved by switching to manually constructed schema.Module per plan's explicit authorization.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All 10 stub tests are RED and ready for implementation in plans 02-05
- Plans 02-05 can use exact -run filters: TestAnyData|TestAnyXML, TestIdentityref|TestIdentityrefCrossModule, TestRPC|TestAction|TestNotification, TestDeviationNotSupported|TestDeviationReplace|TestDeviationAddDelete
- Blocker noted in STATE.md: deviation application ordering and cross-module identityref base resolution design needed before plans 04 and 02 respectively

---
*Phase: 03-go-generator*
*Completed: 2026-03-14*
