# T05: 03-go-generator 05

**Slice:** S03 — **Milestone:** M001

## Description

Implement GOGEN-01: emit typed Go const blocks for identityref leaf types instead of *string.
Turn TestIdentityref and TestIdentityrefCrossModule from RED to GREEN.

Purpose: Identityref is YANG's equivalent of a typed enum. Emitting *string forces callers
to use string literals in switch statements — no compile-time safety. A typed const block
gives callers autocomplete, exhaustiveness checking, and a String() method.

Output:
- generator/golang/generator.go — generateIdentityConsts function + wiring in generateField
  and generateNode Leaf case; cross-module modules slice threaded through call chain
- generator/golang/generator_identityref_test.go — stubs replaced with real assertions

## Must-Haves

- [ ] "A YANG module with an identity hierarchy and an identityref leaf generates a typed Go const block (type XxxIdentity string + const (...)) — not a *string field"
- [ ] "A YANG module that uses an identityref base from an imported module produces the same typed const block via cross-module resolution"
- [ ] "Generated identity const blocks are emitted once per base identity, not once per referencing leaf"
- [ ] "TestIdentityref and TestIdentityrefCrossModule are GREEN"
- [ ] "go/format.Source succeeds on all generated output containing identity consts"

## Files

- `generator/golang/generator.go`
- `generator/golang/generator_identityref_test.go`
