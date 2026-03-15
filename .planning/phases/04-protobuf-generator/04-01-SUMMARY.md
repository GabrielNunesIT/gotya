---
phase: 04-protobuf-generator
plan: "01"
subsystem: testing
tags: [protobuf, tdd, red-stubs, anydata, anyxml, rpc, action, notification, cel-validation, golden-files]

# Dependency graph
requires:
  - phase: 03-go-generator
    provides: "schema.AnyData/AnyXML/RPC/Action/Notification node types and manually-constructed module test pattern"
provides:
  - "RED stub tests for PBGEN-02 (anydata/anyxml → google.protobuf.Any)"
  - "RED stub tests for PBGEN-03 (rpc/action service blocks, notification standalone message)"
  - "RED stub tests for PBGEN-04 (CEL path validation — collect all errors pre-write)"
  - "TestGoldenProto harness with -update-proto flag in test/proto_golden_test.go"
affects:
  - 04-02 (implements PBGEN-02 anydata/anyxml to make stubs GREEN)
  - 04-03 (implements PBGEN-03 rpc/action/notification to make stubs GREEN)
  - 04-04 (implements PBGEN-04 CEL path validation to make stubs GREEN)
  - 04-05 (runs TestGoldenProto -update-proto to generate golden files)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "TDD RED stub pattern: t.Fatal('not yet implemented') at top of test body, implementation asserts below — avoids needing to delete stub body later"
    - "proto golden test harness: -update-proto flag (separate from -update used by Go golden tests)"

key-files:
  created:
    - generator/protobuf/generator_anydata_test.go
    - generator/protobuf/generator_rpc_test.go
    - generator/protobuf/generator_cel_validation_test.go
    - test/proto_golden_test.go
  modified: []

key-decisions:
  - "Used package protobuf_test (external) for new test files — consistent with generator_test.go and device_test.go in same directory"
  - "Proto golden flag named -update-proto (not -update) to avoid conflict with any future -update flag in the same test package"
  - "Golden test reuses corpusLoader and namedModule directly from corpus_test.go (same package) — no duplication"
  - "CEL path validation stubs use manually-constructed schema.Module with NewLeaf nodes — compiler invocation not needed for RED stubs"
  - "TestGoldenProto is a real harness (not a t.Fatal stub) — fails gracefully with descriptive message when golden files are missing"

patterns-established:
  - "Pattern: proto RED stubs mirror Go generator test structure (YANG source → lexer → parser → compiler → GenerateDevice)"
  - "Pattern: TestGoldenProto uses -update-proto flag; os.MkdirAll called in parent test before parallel subtests to avoid race on mkdir"

requirements-completed:
  - PBGEN-02
  - PBGEN-03
  - PBGEN-04

# Metrics
duration: 2min
completed: 2026-03-15
---

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
