# T01: 05-public-api-stabilization 01

**Slice:** S05 — **Milestone:** M001

## Description

Write failing test stubs for all Phase 5 public API behaviors.

Purpose: Establish the RED baseline that Phase 5 plans 02 and 03 will turn GREEN. Having all test stubs in place before implementation ensures the implementation is driven by the observable contract, not inferred from the code.
Output: gotya_test.go with 6 test functions (package gotya_test), all failing.

## Must-Haves

- [ ] "All API-01/02/03 test functions exist in gotya_test.go and fail RED (compilation or t.Fatal)"
- [ ] "gotya_test.go imports only the gotya package — no ast, schema, compiler imports"
- [ ] "Test stubs cover: ParseError type assertion, multiple diagnostics, opaque ASTModule, opaque Module return from Compile"

## Files

- `gotya_test.go`
