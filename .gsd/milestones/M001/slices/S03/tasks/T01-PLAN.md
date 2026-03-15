# T01: 03-go-generator 01

**Slice:** S03 — **Milestone:** M001

## Description

Write failing test stubs (RED phase) for all four Go generator gaps. Each stub compiles,
fails immediately via t.Fatal("not yet implemented"), and targets the exact function name
referenced in VALIDATION.md so subsequent plans can run a precise -run filter.

Purpose: Nyquist compliance — every implementation task in plans 02-05 must have a pre-existing
test that turns GREEN. Stubs are the contract; implementation fills them.

Output:
- 4 new test files under generator/golang/
- 9 failing tests total (2 + 2 + 3 + 3 = 10, minus TestNotification which is GOGEN-03 but
  listed under generator_rpc_test.go)

## Must-Haves

- [ ] "go test ./generator/golang/... -run 'TestAnyData|TestAnyXML|TestIdentityref|TestRPC|TestAction|TestNotification|TestDeviation' fails (RED) before any implementation"
- [ ] "Each test file compiles cleanly — t.Fatal('not yet implemented') is the only body"
- [ ] "Test names match the VALIDATION.md sampling map exactly so later plans can target them precisely"

## Files

- `generator/golang/generator_anydata_test.go`
- `generator/golang/generator_identityref_test.go`
- `generator/golang/generator_rpc_test.go`
- `generator/golang/generator_deviation_test.go`
