---
id: T01
parent: S04
milestone: M001
provides:
  - "RED stub tests for PBGEN-02 (anydata/anyxml → google.protobuf.Any)"
  - "RED stub tests for PBGEN-03 (rpc/action service blocks, notification standalone message)"
  - "RED stub tests for PBGEN-04 (CEL path validation — collect all errors pre-write)"
  - "TestGoldenProto harness with -update-proto flag in test/proto_golden_test.go"
requires: []
affects: []
key_files: []
key_decisions: []
patterns_established: []
observability_surfaces: []
drill_down_paths: []
duration: 2min
verification_result: passed
completed_at: 2026-03-15
blocker_discovered: false
---
# T01: 04-protobuf-generator 01

**# Phase 4 Plan 01: RED Stubs for PBGEN-02, PBGEN-03, PBGEN-04 Summary**

## What Happened

# Phase 4 Plan 01: RED Stubs for PBGEN-02, PBGEN-03, PBGEN-04 Summary

**9 failing TDD RED stub tests (anydata/anyxml, rpc/action/notification, CEL path validation) plus proto golden file harness — all fail with t.Fatal("not yet implemented"), no compile errors**

## Performance

- **Duration:** 2 min
- **Started:** 2026-03-15T12:21:50Z
- **Completed:** 2026-03-15T12:23:53Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- 6 RED stub tests for PBGEN-02 (TestAnyDataField, TestAnyXMLField, TestAnyDataConditionalImport) and PBGEN-03 (TestRPCServiceBlock, TestActionServiceBlock, TestNotificationMessage)
- 3 RED stub tests for PBGEN-04 (TestCELPathValidationValid, TestCELPathValidationInvalid, TestCELPathValidationAllErrors)
- TestGoldenProto real harness with -update-proto flag, corpusLoader reuse, and graceful missing-file error message
- All pre-existing tests in generator/protobuf/... continue to pass

## Task Commits

Each task was committed atomically:

1. **Task 1: RED stubs for PBGEN-02 and PBGEN-03** - `0b2ad53` (test)
2. **Task 2: RED stubs for PBGEN-04 and golden harness** - `dd1bf02` (test)

**Plan metadata:** *(docs commit below)*

_Note: TDD plan — all commits are test-only RED stubs; implementation commits come in Plans 02-05_

## Files Created/Modified
- `generator/protobuf/generator_anydata_test.go` - RED stubs for TestAnyDataField, TestAnyXMLField, TestAnyDataConditionalImport (PBGEN-02)
- `generator/protobuf/generator_rpc_test.go` - RED stubs for TestRPCServiceBlock, TestActionServiceBlock, TestNotificationMessage (PBGEN-03)
- `generator/protobuf/generator_cel_validation_test.go` - RED stubs for TestCELPathValidationValid, TestCELPathValidationInvalid, TestCELPathValidationAllErrors (PBGEN-04)
- `test/proto_golden_test.go` - TestGoldenProto golden file harness with -update-proto flag

## Decisions Made
- Used `package protobuf_test` (external test package) for all new test files — consistent with `generator_test.go` and `device_test.go` already in the same directory, and the public API (`New()`, `Options{}`) is all that's needed
- Proto golden flag named `-update-proto` not `-update` — avoids potential conflict with other flag declarations in the same `test` package
- TestGoldenProto is a real harness, not a stub with `t.Fatal` — it fails gracefully with "golden file missing — run with -update-proto to create" which is informative and expected behavior before Plan 05
- CEL validation stubs use `schema.NewLeaf()` + `schema.TypeDefinition{Length/Range}` directly (no YANG source + compiler) since the schema nodes themselves are what matters for the validation behavior

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- Minor: initial draft of `test/proto_golden_test.go` used `astModuleCache` (undefined type) instead of `*ast.Module` — caught by `go test -c` compile check and fixed immediately before commit. No impact on task commits.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All 9 RED stubs are ready; Plans 02, 03, 04 can implement in any order to turn stubs GREEN
- TestGoldenProto harness is ready; Plan 05 runs with -update-proto to generate `testdata/proto/` golden files
- No blockers — all stub files compile, all fail with "not yet implemented"

---
*Phase: 04-protobuf-generator*
*Completed: 2026-03-15*
