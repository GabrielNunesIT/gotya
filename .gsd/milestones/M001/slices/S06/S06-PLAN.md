# S06: V1 Gap Closure

**Goal:** Refactor compiler/validator.
**Demo:** Refactor compiler/validator.

## Must-Haves


## Tasks

- [x] **T01: 06-v1-gap-closure 01**
  - Refactor compiler/validator.go so each validation method calls c.compiler.addError(sentinel, msg) directly with the per-category sentinel, eliminating the v.errors []string accumulator and the single ErrInvalidDefault wrapper. This is the prerequisite for Plan 03's test migration: 8 of the 21 assert.Contains sites target Validator-path errors, and those sites cannot be migrated to assert.ErrorIs until the correct sentinels are emitted.

Purpose: Unlock the TEST-03 migration for Validator-path errors by giving each error category its own sentinel at emission time.
Output: compiler/validator.go with per-category c.compiler.addError calls; Validate() returns nothing (or always nil); v.errors field removed.
- [x] **T02: 06-v1-gap-closure 02** `est:10min`
  - Fix gotya.Compile() to accumulate errors across all modules instead of returning on the first failure, and prove the behavior with TestCompile_MultiModuleErrors in gotya_test.go.

Purpose: Callers with multiple-module inputs see the full failure picture, not just the first module's error.
Output: Updated gotya.go Compile() loop; new TestCompile_MultiModuleErrors test function in gotya_test.go.
- [x] **T03: 06-v1-gap-closure 03**
  - Migrate all 21 assert.Contains(t, err.Error(), ...) error-assertion call sites in compiler/compiler_test.go to assert.ErrorIs(t, err, compiler.ErrXxx) sentinel assertions. Plan 01's Validator refactor is a prerequisite: without it, the 8 Validator-path sites would fail because the Validator was emitting ErrInvalidDefault for all categories.

Purpose: Error assertions are structural (refactoring-safe) instead of fragile substring checks.
Output: compiler/compiler_test.go with zero assert.Contains(t, err.Error(), ...) calls; all error assertions use typed sentinels.

## Files Likely Touched

- `compiler/validator.go`
- `gotya.go`
- `gotya_test.go`
- `compiler/compiler_test.go`
