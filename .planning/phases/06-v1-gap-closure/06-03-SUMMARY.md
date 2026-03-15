---
phase: 06-v1-gap-closure
plan: "03"
subsystem: compiler
tags: [compiler, testing, sentinels, errors, assert.ErrorIs, TEST-03]
dependency_graph:
  requires: [06-01]
  provides: [structural-error-assertions-in-compiler-tests]
  affects: [compiler/compiler_test.go]
tech_stack:
  added: []
  patterns: [assert.ErrorIs sentinel assertions, table-driven sentinel field]
key_files:
  created: []
  modified:
    - compiler/compiler_test.go
decisions:
  - "assert.ErrorIs x3 for ErrTypeRestriction documents 3 distinct accumulation errors — repetition is intentional"
  - "assert.ErrorIs x2 for ErrListMissingKey documents 2 distinct list key errors — repetition is intentional"
  - "identityref table struct field renamed from errorMsg string to sentinel error — type-safe, refactoring-safe"
  - "Valid Local Base case retains nil sentinel — sentinel field zero value is nil, expectError=false skips ErrorIs check"
metrics:
  duration: "2min"
  completed: "2026-03-15"
  tasks_completed: 2
  files_modified: 1
---

# Phase 6 Plan 03: Migrate compiler_test.go to assert.ErrorIs Summary

**One-liner:** Migrated all 21 assert.Contains(t, err.Error(), ...) call sites in compiler/compiler_test.go to assert.ErrorIs(t, err, compiler.ErrXxx) sentinel assertions, eliminating fragile substring matching.

## Tasks Completed

| Task | Description | Commit | Files |
|------|-------------|--------|-------|
| 1 | Migrate 13 direct-path assert.Contains sites to assert.ErrorIs | f33e7ed | compiler/compiler_test.go |
| 2 | Migrate 8 Validator-path sites and update identityref table test | 0ca5c96 | compiler/compiler_test.go |

## Changes Made

### compiler/compiler_test.go

**Task 1 — 13 direct-path sites:**
- `TestCompiler_Validation` (line ~210): `ErrListMissingKey`
- `TestCompiler_IdentifierUniqueness` (line ~240): `ErrDuplicateIdent`
- `TestCompiler_ConfigBoundary` (line ~269): `ErrConfigBoundary`
- `TestCompiler_InvalidTypeRestrictions` (lines ~347-349): `ErrTypeRestriction` x3
- `TestCompiler_ListKeyValidation` (lines ~380-381): `ErrListMissingKey` x2
- `TestCompiler_CircularUses` (line ~414): `ErrCircularUses`
- `TestCompiler_MandatoryDefaultValidation` (line ~441): `ErrMandatoryDefault`
- `TestCompiler_CircularTypedef` (line ~1227): `ErrCircularTypedef`
- `TestCompiler_UnresolvableAugment` (line ~1242): `ErrAugmentNotFound`
- `TestCompiler_MaxErrors` (line ~1324): `ErrMaxErrors`

**Task 2 — 8 Validator-path sites and table test:**
- `TestCompiler_IdentityrefValidation`: renamed `errorMsg string` field to `sentinel error`; all 3 error case entries set to `compiler.ErrIdentityrefBase`; loop assertion changed from `assert.Contains(t, err.Error(), tt.errorMsg)` to `assert.ErrorIs(t, err, tt.sentinel)`
- `TestCompiler_DefaultValues` (lines ~1139-1142): `ErrInvalidDefault` x4
- `TestCompiler_XPathSyntax` (lines ~1179-1181): `ErrXPathSyntax` x3

## Verification

```
grep -n 'assert\.Contains(t, err\.Error()' compiler/compiler_test.go  -- no results (PASS)
go test ./compiler/... -count=1  -- PASS (all tests)
go test ./... -count=1  -- PASS (full suite green)
```

## Deviations from Plan

None - plan executed exactly as written.

## Self-Check: PASSED

- compiler/compiler_test.go: exists and modified
- Commit f33e7ed: Task 1 (13 direct-path sites)
- Commit 0ca5c96: Task 2 (8 Validator-path sites + identityref table)
- Zero assert.Contains(t, err.Error()) calls remain
- go test ./... passes
