# T04: 01-compiler-correctness 04

**Slice:** S01 — **Milestone:** M001

## Description

Implement the three remaining compiler fixes: circular typedef detection in getType(), augment loop iteration cap, and bounded error accumulation via MaxErrors. Turn the final three failing test stubs GREEN.

Purpose: COMP-02 prevents stack overflow from user-crafted circular typedefs; COMP-04 prevents the augment loop from spinning indefinitely on pathological input; COMP-07 bounds memory use and provides predictable behavior when a YANG file contains hundreds of validation errors.
Output: compiler.go with all three fixes applied; all 8 Phase 1 tests GREEN.

## Must-Haves

- [ ] "A YANG module with a circular typedef (A uses B uses A) returns an error, not a stack overflow"
- [ ] "A YANG module with unresolvable augments returns an error listing the augment paths"
- [ ] "Compilation stops after MaxErrors (default 100) and appends a truncation message"
- [ ] "All pre-existing tests still pass"

## Files

- `compiler/compiler.go`
- `compiler/compiler_test.go`
