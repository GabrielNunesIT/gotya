---
phase: 06-v1-gap-closure
plan: "01"
subsystem: compiler
tags: [compiler, validator, sentinels, errors, tdd]
dependency_graph:
  requires: []
  provides: [per-category-sentinel-errors-from-validator]
  affects: [compiler/validator.go, compiler/compiler.go]
tech_stack:
  added: []
  patterns: [errors.Is-compatible sentinel wrapping, addError delegation]
key_files:
  created:
    - compiler/validator_sentinels_test.go
  modified:
    - compiler/validator.go
    - compiler/compiler.go
decisions:
  - "Validator refactored to call c.addError(sentinel, msg) directly — each error category carries correct sentinel for errors.Is"
  - "Validate() changed to return nothing — errors flow directly into compiler.errors via addError"
  - "strings import retained in validator.go — strings.Split and strings.TrimSpace still used"
metrics:
  duration: "2min"
  completed: "2026-03-15"
  tasks_completed: 1
  files_modified: 3
---

# Phase 6 Plan 01: Validator Per-Category Sentinel Refactor Summary

**One-liner:** Refactored Validator to emit per-category sentinels (ErrIdentityrefBase, ErrXPathSyntax, ErrMandatoryDefault, ErrConfigBoundary, ErrListMissingKey, ErrInvalidDefault) via direct c.compiler.addError calls, eliminating the v.errors accumulator and ErrInvalidDefault wrapper.

## Tasks Completed

| Task | Description | Commit | Files |
|------|-------------|--------|-------|
| 1 (RED) | Add failing sentinel tests for per-category validator errors | 4024136 | compiler/validator_sentinels_test.go |
| 1 (GREEN) | Refactor Validator to emit per-category sentinels via addError | 2fcd599 | compiler/validator.go, compiler/compiler.go |

## Changes Made

### compiler/validator.go
- Removed `errors []string` field from `Validator` struct
- Removed `errors: make([]string, 0)` from `NewValidator`
- Changed `Validate()` signature from `func (v *Validator) Validate() error` to `func (v *Validator) Validate()`
- Removed the `if len(v.errors) > 0 { return fmt.Errorf(...) }` block and `strings.Join` call
- All `v.errors = append(v.errors, fmt.Sprintf(...))` replaced with `v.compiler.addError(sentinel, msg)` using per-category sentinels:
  - `validateConfigBoundaries`: `ErrConfigBoundary`
  - `validateListKeys`: `ErrListMissingKey`
  - `validateMandatoryAndDefault`: `ErrMandatoryDefault`
  - `checkNodeIdentities` / `resolveIdentityBase`: `ErrIdentityrefBase`
  - `checkTypeMatch`: `ErrInvalidDefault`
  - `checkXPathSyntax`: `ErrXPathSyntax`

### compiler/compiler.go
- Updated call site from `if err := validator.Validate(); err != nil { c.addError(ErrInvalidDefault, err.Error()) }` to `validator.Validate()`

## Verification

```
go build ./compiler/...  -- PASS
go test ./compiler/... -count=1  -- PASS (all tests including 3 new sentinel tests)
grep -n "v\.errors" compiler/validator.go  -- no results (PASS)
grep -n "ErrInvalidDefault" compiler/validator.go  -- only in checkTypeMatch (PASS)
```

## Deviations from Plan

None - plan executed exactly as written.

## Self-Check: PASSED

All files exist and commits verified (4024136, 2fcd599).

## Impact on TEST-03 Migration

The 8 Validator-path error sites can now be migrated to `assert.ErrorIs` in Plan 03:
- `validateConfigBoundaries` errors: `errors.Is(err, compiler.ErrConfigBoundary)`
- `validateListKeys` errors: `errors.Is(err, compiler.ErrListMissingKey)`
- `validateMandatoryAndDefault` errors: `errors.Is(err, compiler.ErrMandatoryDefault)`
- `checkNodeIdentities`/`resolveIdentityBase` errors: `errors.Is(err, compiler.ErrIdentityrefBase)`
- `checkTypeMatch` errors: `errors.Is(err, compiler.ErrInvalidDefault)`
- `checkXPathSyntax` errors: `errors.Is(err, compiler.ErrXPathSyntax)`
