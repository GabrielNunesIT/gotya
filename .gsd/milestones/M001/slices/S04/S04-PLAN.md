# S04: Protobuf Generator

**Goal:** Write failing test stubs (TDD RED) for all three PBGEN implementation gaps — anydata/anyxml, rpc/action/notification service blocks, and CEL path validation — plus the golden file test harness.
**Demo:** Write failing test stubs (TDD RED) for all three PBGEN implementation gaps — anydata/anyxml, rpc/action/notification service blocks, and CEL path validation — plus the golden file test harness.

## Must-Haves


## Tasks

- [x] **T01: 04-protobuf-generator 01** `est:2min`
  - Write failing test stubs (TDD RED) for all three PBGEN implementation gaps — anydata/anyxml, rpc/action/notification service blocks, and CEL path validation — plus the golden file test harness.

Purpose: Establish test contracts before implementation so every subsequent plan implements to a clear, pre-agreed specification.
Output: Four new test files; go test ./generator/protobuf/... fails with t.Fatal("not yet implemented").
- [x] **T02: 04-protobuf-generator 02** `est:3min`
  - Implement PBGEN-02: anydata and anyxml YANG nodes emit `google.protobuf.Any` proto fields, with the `google/protobuf/any.proto` import added conditionally only when needed.

Purpose: Closes the silent-node-drop gap for anydata/anyxml — consumers of the proto generator get a valid field for every YANG node.
Output: Modified generator/protobuf/generator.go; all 3 TestAnyData* tests pass GREEN.
- [x] **T03: 04-protobuf-generator 03** `est:4min`
  - Implement PBGEN-03: rpc and action YANG statements emit Protobuf service blocks (one per module) with typed Input/Output messages; notification statements emit standalone messages.

Purpose: Enables gRPC service stub generation from YANG RPC definitions — the primary use case for proto output in network management.
Output: Modified generator/protobuf/generator.go; all 3 TestRPC*/TestNotification* tests pass GREEN.
- [x] **T04: 04-protobuf-generator 04** `est:2min`
  - Implement PBGEN-04: CEL annotation path validation — when GenerateCELValidation=true, verify that every leaf field receiving a buf.validate annotation corresponds to a real schema node; return all invalid paths as an error before writing output.

Purpose: Catches invalid CEL-annotated paths at generation time, not silently in downstream proto compilation or at runtime.
Output: Modified generator/protobuf/validations.go + a call site in GenerateDevice; all 3 TestCELPath* tests pass GREEN.
- [x] **T05: 04-protobuf-generator 05** `est:1min`
  - Create the RFC 7950 coverage document (PBGEN-01) and generate golden proto files from the corpus to lock in TestGoldenProto GREEN (PBGEN-02, PBGEN-03, PBGEN-04 regression).

Purpose: Delivers the consumer-facing coverage document that documents exactly what the proto generator supports, and establishes the golden file regression baseline so future changes are caught immediately.
Output: docs/proto-coverage.md with all 25 RFC 7950 statements; testdata/proto/ populated; TestGoldenProto GREEN.

## Files Likely Touched

- `generator/protobuf/generator_anydata_test.go`
- `generator/protobuf/generator_rpc_test.go`
- `generator/protobuf/generator_cel_validation_test.go`
- `test/proto_golden_test.go`
- `generator/protobuf/generator.go`
- `generator/protobuf/generator.go`
- `generator/protobuf/validations.go`
- `docs/proto-coverage.md`
- `testdata/proto/`
