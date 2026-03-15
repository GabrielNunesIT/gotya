# T02: 03-go-generator 02

**Slice:** S03 — **Milestone:** M001

## Description

Implement GOGEN-02: emit json.RawMessage fields for anydata and anyxml nodes in the Go generator.
Turn TestAnyData and TestAnyXML from RED to GREEN.

Purpose: No YANG nodes silently dropped from generated output. anydata/anyxml are valid data
nodes that callers need to read and write.

Output:
- generator/golang/generator.go — two-site patch:
  1. Module-struct field emission loop in GenerateDevice adds case for AnyData/AnyXML
  2. generateNode adds case for AnyData/AnyXML (returns nil — no nested struct needed)
- generator/golang/generator_anydata_test.go — replace t.Fatal stubs with real assertions

## Must-Haves

- [ ] "A YANG module with an anydata node generates a Go struct field typed json.RawMessage — no node is silently dropped"
- [ ] "A YANG module with an anyxml node generates a Go struct field typed json.RawMessage"
- [ ] "go/format.Source succeeds on all generated output — no syntax errors introduced"
- [ ] "TestAnyData and TestAnyXML pass (GREEN)"

## Files

- `generator/golang/generator.go`
- `generator/golang/generator_anydata_test.go`
