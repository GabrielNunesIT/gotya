# S03: Go Generator

**Goal:** Write failing test stubs (RED phase) for all four Go generator gaps.
**Demo:** Write failing test stubs (RED phase) for all four Go generator gaps.

## Must-Haves


## Tasks

- [x] **T01: 03-go-generator 01** `est:12min`
  - Write failing test stubs (RED phase) for all four Go generator gaps. Each stub compiles,
fails immediately via t.Fatal("not yet implemented"), and targets the exact function name
referenced in VALIDATION.md so subsequent plans can run a precise -run filter.

Purpose: Nyquist compliance — every implementation task in plans 02-05 must have a pre-existing
test that turns GREEN. Stubs are the contract; implementation fills them.

Output:
- 4 new test files under generator/golang/
- 9 failing tests total (2 + 2 + 3 + 3 = 10, minus TestNotification which is GOGEN-03 but
  listed under generator_rpc_test.go)
- [x] **T02: 03-go-generator 02** `est:10min`
  - Implement GOGEN-02: emit json.RawMessage fields for anydata and anyxml nodes in the Go generator.
Turn TestAnyData and TestAnyXML from RED to GREEN.

Purpose: No YANG nodes silently dropped from generated output. anydata/anyxml are valid data
nodes that callers need to read and write.

Output:
- generator/golang/generator.go — two-site patch:
  1. Module-struct field emission loop in GenerateDevice adds case for AnyData/AnyXML
  2. generateNode adds case for AnyData/AnyXML (returns nil — no nested struct needed)
- generator/golang/generator_anydata_test.go — replace t.Fatal stubs with real assertions
- [x] **T03: 03-go-generator 03** `est:5min`
  - Implement GOGEN-03: emit typed Go structs for rpc, action, and notification YANG statements.
Turn TestRPC, TestAction, TestNotification from RED to GREEN.

Purpose: Callers building gRPC or REST APIs from YANG need typed input/output types they can
instantiate and pass around — not just config/state data structs.

Output:
- generator/golang/generator.go — three additions:
  1. hasValidNodes extended to return true for RPC/Action/Notification
  2. generateNode handles *schema.RPC, *schema.Action, *schema.Notification
  3. GenerateDevice post-loop pass emits RPC/Action/Notification structs once per module (NOT
     inside the Config+State loop)
- generator/golang/generator_rpc_test.go — stubs replaced with real assertions
- [x] **T04: 03-go-generator 04** `est:2min`
  - Implement GOGEN-04: apply deviation statements as a pre-pass before code generation.
Turn TestDeviationNotSupported, TestDeviationReplace, TestDeviationAddDelete from RED to GREEN.

Purpose: Vendor YANG modules routinely use deviations to remove or modify nodes for specific
device capabilities. Generated Go that ignores deviations would describe a superset of what
the device actually supports — callers could set fields that don't exist on the device.

Output:
- generator/golang/generator.go — new applyDeviations(mod) function + pre-pass call in GenerateDevice
- generator/golang/generator_deviation_test.go — stubs replaced with real assertions
- [x] **T05: 03-go-generator 05** `est:10min`
  - Implement GOGEN-01: emit typed Go const blocks for identityref leaf types instead of *string.
Turn TestIdentityref and TestIdentityrefCrossModule from RED to GREEN.

Purpose: Identityref is YANG's equivalent of a typed enum. Emitting *string forces callers
to use string literals in switch statements — no compile-time safety. A typed const block
gives callers autocomplete, exhaustiveness checking, and a String() method.

Output:
- generator/golang/generator.go — generateIdentityConsts function + wiring in generateField
  and generateNode Leaf case; cross-module modules slice threaded through call chain
- generator/golang/generator_identityref_test.go — stubs replaced with real assertions

## Files Likely Touched

- `generator/golang/generator_anydata_test.go`
- `generator/golang/generator_identityref_test.go`
- `generator/golang/generator_rpc_test.go`
- `generator/golang/generator_deviation_test.go`
- `generator/golang/generator.go`
- `generator/golang/generator_anydata_test.go`
- `generator/golang/generator.go`
- `generator/golang/generator_rpc_test.go`
- `generator/golang/generator.go`
- `generator/golang/generator_deviation_test.go`
- `generator/golang/generator.go`
- `generator/golang/generator_identityref_test.go`
