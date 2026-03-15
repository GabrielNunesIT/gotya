# T01: 01-compiler-correctness 01

**Slice:** S01 — **Milestone:** M001

## Description

Write failing test stubs for all 8 behaviors Phase 1 will implement.

Purpose: Nyquist compliance — every subsequent implementation task must have an automated verify command that runs against a pre-existing test. Stubs fail now (RED) and turn green after each fix is applied.
Output: Two test files containing 8 test functions that compile, run, and fail with t.Fatal("not yet implemented") or equivalent.

## Must-Haves

- [ ] "Running any failing-stub test produces a test failure (not a compile error or panic)"
- [ ] "All 8 test function names match exactly what subsequent plans' verify commands reference"
- [ ] "cmd/gotya/loader_test.go exists and compiles successfully"
- [ ] "Existing tests still pass after stubs are added"

## Files

- `compiler/compiler_test.go`
- `cmd/gotya/loader_test.go`
