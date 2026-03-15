# Phase 4: Protobuf Generator - Context

**Gathered:** 2026-03-15
**Status:** Ready for planning

<domain>
## Phase Boundary

Produce correct, complete Protobuf output for all YANG constructs relevant to a network management gRPC service, with all coverage gaps explicitly documented. This phase covers PBGEN-01 through PBGEN-04 only — no new generator features beyond what's listed in requirements.

</domain>

<decisions>
## Implementation Decisions

### AnyData/AnyXML field type (PBGEN-02)
- Both `anydata` and `anyxml` YANG nodes emit `google.protobuf.Any` — same type for both, no differentiation
- `import "google/protobuf/any.proto";` added to generated output only when the module contains anydata or anyxml nodes (not unconditionally)

### Service block structure (PBGEN-03)
- Claude's discretion — user did not specify. Prefer one `service` block per YANG module (named after the module) so RPCs from different modules don't collide in a single service definition.

### Golden file tests
- Add golden file tests for the Proto generator in this phase (deferred from Phase 2)
- Use the same corpus as the Go generator: `test/assets/yangs/` (205 BBF/IETF/IEEE YANG files)
- Run the full `GenerateDevice` pipeline (parse → compile → GenerateDevice), mirroring real usage
- `-update` flag pattern (go test -run TestGoldenProto ./... -update regenerates golden files)
- Golden files stored in `testdata/` at repo root (consistent with Phase 2 decision)

### CEL path validation (PBGEN-04)
- When `GenerateCELValidation` is enabled and a CEL annotation path doesn't correspond to a real schema node: return an error and do not write the output file (same philosophy as go/format.Source() in the Go generator)
- Collect all invalid paths before returning — one error pass shows all problems, not just the first
- Implementation approach: Claude's discretion — validate that each leaf field that receives a CEL annotation has a corresponding node resolvable in the compiled schema

### Claude's Discretion
- Exact CEL path resolution mechanism (how to walk the schema to verify a field path)
- Coverage document format — a Markdown table with columns: YANG statement, Proto mapping, status (supported/unsupported/out-of-scope), notes
- Whether to validate CEL paths only when GenerateCELValidation=true or always

</decisions>

<specifics>
## Specific Ideas

- Proto generator already has `buildValidateOptions` in `validations.go` — CEL path validation is a post-generation check, not integrated into `buildValidateOptions` itself
- The Go generator pattern for format.Source validation (error returned before writing) is the model for CEL path validation error behavior

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets
- `generator/protobuf/generator.go`: Full proto generator — handles Container, List, Leaf, LeafList, Choice/Case, enums, CEL validation options. Missing: AnyData/AnyXML cases and RPC/Action/Notification cases in `generateField`/`generateNode`
- `generator/protobuf/validations.go`: `buildValidateOptions` — CEL/protovalidate constraint builder. CEL path validation is a new separate concern from this
- `generator/protobuf/generator_test.go` + `validations_test.go` + `device_test.go`: Existing test files — new tests follow same package/style
- `test/assets/yangs/`: 205 real-world YANG files ready for golden file testing
- `test/generate.go` (`//go:build ignore`): Reference for full pipeline corpus test pattern

### Established Patterns
- `generateField` switch handles node types — add `*schema.AnyData, *schema.AnyXML` case here and in `generateNode`
- `generateNode` currently handles only Container and List — add RPC/Action/Notification handling here
- `shouldSkip` pattern used consistently for deprecated/obsolete filtering
- Generator tests use `testify/assert` co-located in `_test.go` files
- TDD pattern from Phase 3: write RED stubs first, then implement to GREEN

### Integration Points
- `GenerateDevice` header section (where CEL import is added) — add `google/protobuf/any.proto` import detection here
- `generateField` switch — AnyData/AnyXML and RPC/Action/Notification fall-through cases need explicit handling
- Post-generation CEL path validation runs after `GenerateDevice` writes output (or as a pre-write validation pass)

</code_context>

<deferred>
## Deferred Ideas

- None — discussion stayed within phase scope

</deferred>

---

*Phase: 04-protobuf-generator*
*Context gathered: 2026-03-15*
