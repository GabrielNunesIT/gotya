# T01: 06-v1-gap-closure 01

**Slice:** S06 — **Milestone:** M001

## Description

Refactor compiler/validator.go so each validation method calls c.compiler.addError(sentinel, msg) directly with the per-category sentinel, eliminating the v.errors []string accumulator and the single ErrInvalidDefault wrapper. This is the prerequisite for Plan 03's test migration: 8 of the 21 assert.Contains sites target Validator-path errors, and those sites cannot be migrated to assert.ErrorIs until the correct sentinels are emitted.

Purpose: Unlock the TEST-03 migration for Validator-path errors by giving each error category its own sentinel at emission time.
Output: compiler/validator.go with per-category c.compiler.addError calls; Validate() returns nothing (or always nil); v.errors field removed.

## Must-Haves

- [ ] "Each validator category emits errors under its correct sentinel, not under ErrInvalidDefault"
- [ ] "errors.Is(err, compiler.ErrIdentityrefBase) returns true for identityref validation failures"
- [ ] "errors.Is(err, compiler.ErrInvalidDefault) returns true for default value validation failures"
- [ ] "errors.Is(err, compiler.ErrXPathSyntax) returns true for XPath syntax validation failures"
- [ ] "go test ./compiler/... passes with no failures"

## Files

- `compiler/validator.go`
