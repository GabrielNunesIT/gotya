# T04: 03-go-generator 04

**Slice:** S03 — **Milestone:** M001

## Description

Implement GOGEN-04: apply deviation statements as a pre-pass before code generation.
Turn TestDeviationNotSupported, TestDeviationReplace, TestDeviationAddDelete from RED to GREEN.

Purpose: Vendor YANG modules routinely use deviations to remove or modify nodes for specific
device capabilities. Generated Go that ignores deviations would describe a superset of what
the device actually supports — callers could set fields that don't exist on the device.

Output:
- generator/golang/generator.go — new applyDeviations(mod) function + pre-pass call in GenerateDevice
- generator/golang/generator_deviation_test.go — stubs replaced with real assertions

## Must-Haves

- [ ] "A YANG module with 'deviate not-supported' on a leaf produces generated Go that does NOT contain a field for that leaf"
- [ ] "A YANG module with 'deviate replace' updating a leaf type produces generated Go with the new type, not the original"
- [ ] "The deviation pre-pass runs before any generateNode call — RFC 7950 §7.12 ordering is satisfied automatically"
- [ ] "TestDeviationNotSupported, TestDeviationReplace, TestDeviationAddDelete are GREEN"

## Files

- `generator/golang/generator.go`
- `generator/golang/generator_deviation_test.go`
