# T04: 04-protobuf-generator 04

**Slice:** S04 — **Milestone:** M001

## Description

Implement PBGEN-04: CEL annotation path validation — when GenerateCELValidation=true, verify that every leaf field receiving a buf.validate annotation corresponds to a real schema node; return all invalid paths as an error before writing output.

Purpose: Catches invalid CEL-annotated paths at generation time, not silently in downstream proto compilation or at runtime.
Output: Modified generator/protobuf/validations.go + a call site in GenerateDevice; all 3 TestCELPath* tests pass GREEN.

## Must-Haves

- [ ] "When GenerateCELValidation=true and all annotated leaf paths correspond to real schema nodes, GenerateDevice returns nil"
- [ ] "When GenerateCELValidation=true and an annotated leaf path has no corresponding schema node, GenerateDevice returns an error containing 'CEL path validation failed' before writing any output"
- [ ] "When multiple invalid CEL paths exist, all are collected and returned in one error — not just the first"
- [ ] "TestCELPathValidationValid, TestCELPathValidationInvalid, TestCELPathValidationAllErrors all pass GREEN"

## Files

- `generator/protobuf/validations.go`
