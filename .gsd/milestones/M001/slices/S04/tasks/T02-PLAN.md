# T02: 04-protobuf-generator 02

**Slice:** S04 — **Milestone:** M001

## Description

Implement PBGEN-02: anydata and anyxml YANG nodes emit `google.protobuf.Any` proto fields, with the `google/protobuf/any.proto` import added conditionally only when needed.

Purpose: Closes the silent-node-drop gap for anydata/anyxml — consumers of the proto generator get a valid field for every YANG node.
Output: Modified generator/protobuf/generator.go; all 3 TestAnyData* tests pass GREEN.

## Must-Haves

- [ ] "A YANG module with anydata nodes produces a proto field 'google.protobuf.Any <fieldname> = N;'"
- [ ] "A YANG module with anyxml nodes produces the same google.protobuf.Any field"
- [ ] "The generated output includes 'import \"google/protobuf/any.proto\";' only when the module contains anydata or anyxml nodes"
- [ ] "A YANG module with no anydata/anyxml nodes does NOT include the google/protobuf/any.proto import"
- [ ] "TestAnyDataField, TestAnyXMLField, TestAnyDataConditionalImport all pass GREEN"

## Files

- `generator/protobuf/generator.go`
