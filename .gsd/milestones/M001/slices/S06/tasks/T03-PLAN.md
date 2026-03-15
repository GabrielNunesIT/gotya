# T03: 06-v1-gap-closure 03

**Slice:** S06 — **Milestone:** M001

## Description

Migrate all 21 assert.Contains(t, err.Error(), ...) error-assertion call sites in compiler/compiler_test.go to assert.ErrorIs(t, err, compiler.ErrXxx) sentinel assertions. Plan 01's Validator refactor is a prerequisite: without it, the 8 Validator-path sites would fail because the Validator was emitting ErrInvalidDefault for all categories.

Purpose: Error assertions are structural (refactoring-safe) instead of fragile substring checks.
Output: compiler/compiler_test.go with zero assert.Contains(t, err.Error(), ...) calls; all error assertions use typed sentinels.

## Must-Haves

- [ ] "compiler/compiler_test.go contains zero assert.Contains(t, err.Error(), ...) calls"
- [ ] "All 21 former string-match sites now assert via assert.ErrorIs(t, err, compiler.ErrXxx)"
- [ ] "The identityref table-driven test uses sentinel error field, not errorMsg string field"
- [ ] "go test ./compiler/... passes with no failures"

## Files

- `compiler/compiler_test.go`
