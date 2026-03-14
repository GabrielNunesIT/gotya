---
phase: 02-testing-infrastructure
plan: 01
subsystem: compiler
tags: [errors, sentinels, testing, refactor]
dependency_graph:
  requires: []
  provides: [compiler.ErrCircularTypedef, compiler.ErrCircularUses, compiler.ErrAugmentNotFound, compiler.ErrDuplicateIdent, compiler.ErrListMissingKey, compiler.ErrConfigBoundary, compiler.ErrInvalidDefault, compiler.ErrMandatoryDefault, compiler.ErrTypeRestriction, compiler.ErrIdentityrefBase, compiler.ErrXPathSyntax, compiler.ErrMaxErrors]
  affects: [compiler/compiler.go, compiler/compiler_test.go, compiler/validator.go]
tech_stack:
  added: []
  patterns: [errors.Is / errors.Join, sentinel error vars, fmt.Errorf %w wrapping]
key_files:
  created: []
  modified:
    - compiler/compiler.go
    - compiler/compiler_test.go
    - compiler/validator.go
decisions:
  - "Validator refactored to call c.addError(sentinel, msg) directly rather than accumulating []string and returning a joined error — ensures each error category carries the correct sentinel for errors.Is"
  - "ErrMaxErrors check uses errors.Is(c.errors[last], ErrMaxErrors) instead of strings.HasPrefix — consistent with new typed error approach"
  - "validator.Validate() now always returns nil; all errors propagate via c.addError"
metrics:
  duration: 8min
  completed: 2026-03-14
  tasks_completed: 2
  files_modified: 3
---

# Phase 2 Plan 01: Compiler Error Sentinels Summary

**One-liner:** Typed sentinel errors (Err* vars + errors.Join) replacing string accumulation, with all test assertions migrated to errors.Is.

## What Was Built

Introduced 12 exported sentinel error variables in `compiler/compiler.go` and migrated the entire compiler error infrastructure from `[]string` to `[]error`. The `addError` method now takes `(sentinel error, msg string)` and wraps each error as `fmt.Errorf("%w: %s", sentinel, msg)`. The final `Compile()` return uses `errors.Join(c.errors...)` instead of `strings.Join`. The `Validator` was refactored to call `c.addError` directly with per-category sentinels instead of accumulating strings.

All `assert.Contains(t, err.Error(), ...)` assertions in `compiler_test.go` were replaced with `assert.ErrorIs(t, err, compiler.ErrXxx)`. The `TestCompiler_IdentityrefValidation` table-driven test had its `errorMsg string` field replaced with `wantSentinel error`.

## Tasks

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Declare sentinel vars and migrate compiler internals | 4014a1a | compiler/compiler.go |
| 2 | Migrate compiler_test.go string assertions to errors.Is | bc15a40 | compiler/compiler_test.go, compiler/validator.go, compiler/compiler.go |

## Verification Results

- `go test ./compiler/... -race -count=1`: PASS
- `grep -c "assert.Contains.*err.Error" compiler/compiler_test.go`: 0
- `grep -c "ErrCircularTypedef\|ErrAugmentNotFound\|ErrDuplicateIdent" compiler/compiler.go`: 21

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Validator accumulated errors as []string returning a single joined string**
- **Found during:** Task 2
- **Issue:** Validator returned all errors via a single `fmt.Errorf("validation failed...\n%s", strings.Join(...))` which could only be wrapped under one sentinel in `c.addError`. XPath, identityref, and default errors would all be wrapped with `ErrInvalidDefault`, causing `errors.Is(err, compiler.ErrXPathSyntax)` to fail.
- **Fix:** Refactored `Validator` to call `c.addError(sentinel, msg)` directly with the correct per-category sentinel (ErrXPathSyntax, ErrIdentityrefBase, ErrInvalidDefault, ErrMandatoryDefault, ErrListMissingKey, ErrConfigBoundary). `Validate()` now returns nil always.
- **Files modified:** compiler/validator.go, compiler/compiler.go
- **Commit:** bc15a40

## Self-Check

- [x] compiler/compiler.go — modified (sentinels + []error + addError signature)
- [x] compiler/compiler_test.go — migrated to assert.ErrorIs
- [x] compiler/validator.go — refactored to use c.addError with sentinels
- [x] Commit 4014a1a exists
- [x] Commit bc15a40 exists
- [x] go test ./compiler/... -race passes
- [x] Zero assert.Contains(err.Error()) patterns remain
