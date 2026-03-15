# S05: Public Api Stabilization

**Goal:** Write failing test stubs for all Phase 5 public API behaviors.
**Demo:** Write failing test stubs for all Phase 5 public API behaviors.

## Must-Haves


## Tasks

- [x] **T01: 05-public-api-stabilization 01** `est:3min`
  - Write failing test stubs for all Phase 5 public API behaviors.

Purpose: Establish the RED baseline that Phase 5 plans 02 and 03 will turn GREEN. Having all test stubs in place before implementation ensures the implementation is driven by the observable contract, not inferred from the code.
Output: gotya_test.go with 6 test functions (package gotya_test), all failing.
- [x] **T02: 05-public-api-stabilization 02** `est:10min`
  - Add structured parser diagnostics and replace os.ErrInvalid with *ParseError.

Purpose: Satisfies API-01 and API-02 — callers can now type-assert parse errors to get structured file/line/column/message data, and all parse errors are propagated (not just the first).
Output: Diagnostic and ParseError types in gotya.go; Diagnostics() method added to parser.Parser; Parse() and ParseFile() rewritten to use them.
- [x] **T03: 05-public-api-stabilization 03** `est:2min`
  - Convert ASTModule from alias to opaque struct, wrap schema.Module in gotya.Module, fix cmd/gotya call sites, and clean all godoc comments.

Purpose: Satisfies API-03 (no internal type leakage through exported signatures) and API-04 (no placeholder comments). After this plan, gotya.go presents a stable, well-documented public API — ready to tag v1.
Output: Opaque ASTModule and Module types in gotya.go; updated cmd/gotya/main.go; clean godoc throughout gotya.go.

## Files Likely Touched

- `gotya_test.go`
- `parser/parser.go`
- `gotya.go`
