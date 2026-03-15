# T02: 06-v1-gap-closure 02

**Slice:** S06 — **Milestone:** M001

## Description

Fix gotya.Compile() to accumulate errors across all modules instead of returning on the first failure, and prove the behavior with TestCompile_MultiModuleErrors in gotya_test.go.

Purpose: Callers with multiple-module inputs see the full failure picture, not just the first module's error.
Output: Updated gotya.go Compile() loop; new TestCompile_MultiModuleErrors test function in gotya_test.go.

## Must-Haves

- [ ] "Calling Compile() with two modules that each have errors returns a single error containing diagnostics from both"
- [ ] "The returned error message includes the name of every failing module"
- [ ] "When all modules compile cleanly, Compile() still returns the compiled slice with no error"
- [ ] "go test ./... -run TestCompile_MultiModuleErrors passes"

## Files

- `gotya.go`
- `gotya_test.go`
