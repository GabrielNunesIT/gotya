# T01: 04-protobuf-generator 01

**Slice:** S04 — **Milestone:** M001

## Description

Write failing test stubs (TDD RED) for all three PBGEN implementation gaps — anydata/anyxml, rpc/action/notification service blocks, and CEL path validation — plus the golden file test harness.

Purpose: Establish test contracts before implementation so every subsequent plan implements to a clear, pre-agreed specification.
Output: Four new test files; go test ./generator/protobuf/... fails with t.Fatal("not yet implemented").

## Must-Haves

- [ ] "All PBGEN-02 unit tests exist and fail (RED) — anydata/anyxml → google.protobuf.Any not yet implemented"
- [ ] "All PBGEN-03 unit tests exist and fail (RED) — rpc/action service blocks and notification messages not yet implemented"
- [ ] "All PBGEN-04 unit tests exist and fail (RED) — CEL path validation not yet implemented"
- [ ] "TestGoldenProto test harness exists in test/proto_golden_test.go and compiles"
- [ ] "go test ./generator/protobuf/... fails with t.Fatal('not yet implemented') in all new stub tests"

## Files

- `generator/protobuf/generator_anydata_test.go`
- `generator/protobuf/generator_rpc_test.go`
- `generator/protobuf/generator_cel_validation_test.go`
- `test/proto_golden_test.go`
