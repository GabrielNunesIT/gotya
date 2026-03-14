# Feature Research

**Domain:** YANG-to-Go/Proto codegen library
**Researched:** 2026-03-14
**Confidence:** HIGH (based on direct codebase audit + RFC 7950 spec + ygot comparison)

---

## Executive Note

This analysis is grounded in direct inspection of the existing generators
(`generator/golang/generator.go`, `generator/protobuf/generator.go`, `schema/schema.go`)
cross-referenced against RFC 7950 and the ygot/goyang ecosystem. Confidence in
gap identification is HIGH because the gaps were confirmed by reading actual code,
not inferred.

---

## Feature Landscape

### Table Stakes (Users Expect These)

Features that Go developers consuming YANG-modeled data assume work correctly.
Missing or broken = library is not usable on real-world YANG files.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Container → Go struct | Core data modelling unit in YANG | LOW | Implemented. Audit needed for edge cases. |
| List → Go slice-of-struct with key fields | Every NETCONF/RESTCONF model uses lists | LOW | Implemented. Ordered-map variant also present. |
| Leaf scalar type mapping (int8–uint64, bool, string, decimal64, binary, empty) | Required for any leaf to be usable | LOW | Implemented in `mapYANGTypeToGo`. |
| Enumeration → named Go iota type with String/MarshalJSON | Idiomatic Go for YANG enumerations | MEDIUM | Implemented. |
| Typedef → named Go type alias | Common in real YANG modules | MEDIUM | Implemented for scalar typedefs. |
| Union → tagged-field struct | RFC 7950 §7.4 union requires multiple-type holder | HIGH | Implemented via union struct with one active field. |
| Bits → uint64 with named bit constants | Used in capability flags, interface features | MEDIUM | Implemented with Set/Clear/String/MarshalJSON helpers. |
| Choice/Case → oneof-style interface + case structs | Standard YANG for optional sub-trees | HIGH | Implemented with Go interface + custom MarshalJSON/UnmarshalJSON. |
| Grouping expansion (uses/refine) | Mandatory for modular YANG models | HIGH | Compiler resolves; generator inherits. Edge cases fragile (see PITFALLS). |
| Augment resolution | Core to OpenConfig and vendor extension patterns | HIGH | Compiler resolves multi-level augments. Has convergence and nil-panic bugs (tracked). |
| if-feature guarding | Required so feature-gated nodes are conditionally emitted | MEDIUM | Compiler evaluates; expression parser only handles simple and/or/not — parenthesised expressions not supported. |
| RFC 7951 JSON codec for generated structs | Users need to marshal/unmarshal NETCONF/RESTCONF payloads | HIGH | Codec package exists and is tested. |
| Config/State split in generated Go | OpenConfig and many vendor models separate config/state trees | MEDIUM | Implemented via `GenerateFakeroot` with dual Config/State struct generation. |
| Validation methods on generated structs (range, length, pattern, mandatory) | Users need correctness guarantees at Go level | HIGH | Implemented via `Validate()` methods. Pattern validation uses regexp but pattern correctness not pre-validated. |
| Default value population helper | Many YANG leaves have defaults; needed for correct no-op configs | MEDIUM | Implemented via `GeneratePopulateDefault` option. |
| Getter/setter generation | Standard ergonomic API for generated structs | LOW | Implemented via `GenerateGetters` / `GenerateSetters` options. |
| Namespace/module annotation on struct fields | Required for RFC 7951 JSON encoding with prefixes | MEDIUM | Implemented via `yang:` struct tag on fields. |
| Fakeroot / Device aggregation | Multi-module generation needs a single entry struct | MEDIUM | Implemented via `GenerateFakeroot`. |
| Protobuf message generation for containers/lists | Proto output target for gRPC/gNMI consumers | MEDIUM | Implemented. Enum blocks, oneof for choice, repeated for leaf-list. |
| CEL validation annotations on Proto fields | Used in buf/validate ecosystem | HIGH | Implemented via `GenerateCELValidation` option. |
| Description-as-comment annotation | Users expect doc comments in generated code | LOW | Implemented via `AddAnnotations` option. |
| Submodule inclusion (`include`) | YANG modules routinely use submodules | MEDIUM | Compiler handles submodule inclusion. No dedicated test coverage. |

### Differentiators (Competitive Advantage)

Features that set gotya apart from goyang (parser-only) and ygot (OpenConfig-centric, heavyweight).

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Dual Go + Proto output from single pipeline | Ygot separates gen_go and proto_generator binaries; gotya unifies | LOW | Already implemented. Needs consistent coverage parity between backends. |
| CEL/buf-validate Proto annotations | Ygot's proto output has no built-in CEL validation support | MEDIUM | Already implemented. Differentiator only if the Go generator also validates patterns — currently it uses regexp, which is weaker. |
| RFC 7951 JSON codec included as library | goyang does not provide a codec; ygot's codec is tightly coupled to ygot structs | HIGH | Already implemented. Key advantage for adoption. |
| Identityref as typed Go constant set | Ygot generates Go enums for identityref; gotya does not yet — opportunity to be cleaner | HIGH | NOT YET IMPLEMENTED. Identityref leaves currently fall through to `*string`. This is the most glaring gap vs ygot. |
| Deviation application to generated output | Applied deviations change the actual generated schema to match what a device actually supports | HIGH | NOT YET IMPLEMENTED. `schema.Deviation` is parsed and stored but generators never apply deviations. |
| RPC/Action as typed Go request/response structs | Network tooling needs to issue RPCs, not just read state | HIGH | NOT YET IMPLEMENTED. `schema.RPC`, `schema.Action`, `schema.Notification` are parsed but neither generator emits code for them. |
| Notification as typed Go struct | Event-driven telemetry consumers need typed notifications | HIGH | NOT YET IMPLEMENTED. Same gap as RPC. |
| Presence-container distinction | RFC 7950 §7.5.1 presence containers have semantic meaning; nil vs empty differ | MEDIUM | `schema.Container.Presence` field exists but generators treat presence containers identically to non-presence ones. |
| AnyData/AnyXML as typed placeholder field | Users need a field to hold arbitrary nested data | LOW | AnyData/AnyXML recognised in `hasValidNodes` but `generateField` never switches on them; they silently produce no output. |
| ordered-by user support for leaf-list | Some YANG models require user-ordered leaf-lists; insertion order must be preserved | MEDIUM | `LeafList.OrderedBy` field exists in schema; generator does not distinguish `ordered-by user` from `ordered-by system`. |
| Leafref validation (cross-node reference check) | Tools that validate instances need leafref reachability checks | HIGH | `TypeDefinition.Path` field exists. Neither compiler nor generators validate leafref targets. |

### Anti-Features (Commonly Requested, Often Problematic)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Runtime XPath / `must` / `when` evaluation | Users want full constraint checking | Requires a full XPath engine; substantial scope; adds a mandatory dependency; most Go code just needs struct validation, not XPath runtime | Emit `must`/`when` expressions as comments or doc tags. Expose them in an API for callers to evaluate with their own XPath library. Tracked as future milestone in PROJECT.md. |
| Full NETCONF/RESTCONF client built-in | Seems natural alongside the codec | Conflates code generation with protocol; doubles the dependency surface; breaks the single-responsibility model | Keep gotya as pure codegen + codec. Users combine with ncclient, gnmic, or other protocol libraries. |
| JSON Schema / OpenAPI generation | Useful for REST API tooling | Separate problem domain; YANG types do not map 1:1 to JSON Schema; creates maintenance burden for each new output format | Implement as a separate generator via the pluggable Generator interface in a future milestone. |
| Automatic regeneration / watch-mode CLI | Developer ergonomics | Out of scope for a library; better handled by make/go generate integration | Document recommended `go generate` pattern in README. |
| Per-node feature flag evaluation at runtime | Some callers want to filter nodes by feature support at runtime | State explosion; tightly couples generated code to deployment; hard to test | `SupportedFeatures` option at compile time already covers the primary use case. |
| Producing a single merged flat namespace output | Some tools want all nodes in one flat package | Destroys module boundaries required for RFC 7951 qualified names; breaks cross-module leafrefs | Keep per-module namespace structure; allow multi-module via GenerateDevice/fakeroot. |

---

## Feature Dependencies

```
Identityref as typed constant set
    └──requires──> Identity hierarchy in schema.Module.Identities (already compiled)
    └──requires──> Generator reads Identities map and emits Go const block

Deviation application
    └──requires──> Deviation stored in schema.Module.Deviations (already compiled)
    └──requires──> Generator applies deviate-not-supported / deviate-replace before emit

RPC/Action/Notification generation (Go)
    └──requires──> schema.RPC / schema.Action / schema.Notification nodes in tree
    └──requires──> Input/Output child nodes emitted as request/response structs
    └──requires──> AddChild() error propagation fixed (tracked bug — silent RPC/Action failures)

RPC/Action/Notification generation (Proto)
    └──requires──> RPC/Action Go structs defined first (or in parallel)
    └──requires──> Proto service blocks (not yet modelled)

AnyData/AnyXML field output
    └──requires──> generateField() switch case for *schema.AnyData, *schema.AnyXML
    └──enhances──> RFC 7951 codec (can represent as json.RawMessage)

Presence-container distinction
    └──requires──> generateField() inspect Container.Presence != nil

Leafref validation
    └──requires──> Schema-level path resolution (findNode path logic already exists)
    └──requires──> Validate() method emits cross-node reference check

Pattern correctness pre-validation
    └──requires──> Compiler validates TypeDefinition.Pattern as valid Go regexp at compile time
    └──enhances──> Validate() methods (no panic from bad regexp at runtime)
```

### Dependency Notes

- **Identityref typed output requires Identity map**: The compiler already collects `schema.Module.Identities`. The generator only needs to iterate that map and emit a `const` block plus a `String()` method. Medium complexity.
- **Deviation application requires Deviations slice**: `schema.Module.Deviations` is populated by the compiler. The generator needs a pre-pass that mutates the schema node tree before emit — or filters during emit. Mutating the tree is cleaner.
- **RPC output requires AddChild() bug fix**: RPCs with Input/Output nodes silently fail to add children today (tracked in CONCERNS.md). Generating RPC structs before that bug is fixed will produce incomplete output. Fix the bug first.
- **AnyData/AnyXML output conflicts with RFC 7951 codec assumptions**: The codec currently assumes all leaf values are typed Go fields. Adding `json.RawMessage` fields for anydata will require codec changes in the same milestone.

---

## MVP Definition

### Launch With (v1)

These are required before the library can be described as covering the YANG 1.1 feature surface.

- [ ] **Identityref → typed Go const block** — the single most visible gap vs ygot; every real-world OpenConfig module uses identityref; currently silently maps to `*string`
- [ ] **AnyData/AnyXML field emission** — currently produces no output, silently drops nodes; any real YANG model that uses anydata produces wrong Go structs
- [ ] **Presence-container distinction** — incorrect nil semantics for presence containers; easy fix, high correctness impact
- [ ] **Deviation application pre-pass** — deviations are the standard mechanism for vendor-specific adjustments; without application, generated code does not match what a real device serves
- [ ] **RPC/Action/Notification generation (Go)** — typed request/response structs; blocked on AddChild() bug fix
- [ ] **AnyData/AnyXML in Proto generator** — emit `google.protobuf.Any` or `bytes` field; currently these nodes are silently skipped
- [ ] **Fix identityref fallthrough in type mapper** — `mapYANGTypeToGo` returns `*string` for identityref; must be fixed in conjunction with typed const block
- [ ] **Audit and fix Go generator on real OpenConfig models** — PROJECT.md explicitly calls for this; it is the discovery step for any remaining table-stakes gaps

### Add After Validation (v1.x)

- [ ] **Leafref validation in Validate() methods** — useful but requires cross-struct path resolution; add once core is stable
- [ ] **RPC/Action/Notification generation (Proto)** — adds `service` blocks to proto output; depends on Go RPC structs being correct first
- [ ] **if-feature parenthesised expression support** — PROJECT.md tracks this; affects models with complex feature expressions
- [ ] **ordered-by user preservation in leaf-list** — ordering semantics differ; relevant for ordered configuration lists

### Future Consideration (v2+)

- [ ] **Runtime XPath / must / when evaluation** — explicitly deferred in PROJECT.md; separate milestone
- [ ] **JSON Schema / OpenAPI generator** — new output format via pluggable interface; separate milestone
- [ ] **Cross-module type deduplication** — performance optimisation for large multi-module schemas; CONCERNS.md notes the gap
- [ ] **Streaming / lazy compilation for very large schemas** — CONCERNS.md notes 100k+ node limit

---

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Identityref → typed const block | HIGH | MEDIUM | P1 |
| AnyData/AnyXML field emission (Go) | HIGH | LOW | P1 |
| Deviation application pre-pass | HIGH | HIGH | P1 |
| RPC/Action/Notification (Go) | HIGH | MEDIUM | P1 |
| Presence-container distinction | MEDIUM | LOW | P1 |
| Fix AddChild() silent errors (prerequisite) | HIGH | LOW | P1 |
| Audit Go generator on real models | HIGH | MEDIUM | P1 |
| AnyData/AnyXML in Proto | MEDIUM | LOW | P2 |
| Leafref validation in Validate() | MEDIUM | HIGH | P2 |
| RPC/Action/Notification (Proto service) | MEDIUM | HIGH | P2 |
| if-feature full expression parser | LOW | MEDIUM | P2 |
| ordered-by user for leaf-list | LOW | MEDIUM | P3 |
| Cross-module type deduplication | LOW | HIGH | P3 |

**Priority key:**
- P1: Must have for v1 completeness — real-world YANG models will fail without these
- P2: Should have, add when core is working
- P3: Nice to have, future consideration

---

## Competitor Feature Analysis

| Feature | goyang (parser only) | ygot (openconfig) | gotya |
|---------|---------------------|-------------------|-------|
| YANG parsing | YES | YES (via goyang) | YES (own lexer/parser) |
| Go struct generation | NO (parser only) | YES | YES |
| Protobuf generation | NO | YES (separate binary) | YES (unified) |
| RFC 7951 JSON codec | NO | YES | YES |
| CEL/buf-validate annotations | NO | NO | YES |
| Identityref → typed enum | NO | YES | NO (gap — *string fallback) |
| Union types | NO | YES | YES |
| Bits types | NO | YES | YES |
| Choice/Case | NO | YES | YES |
| RPC/Action/Notification | NO | YES (partial) | NO (gap) |
| Deviation application | NO | YES (partial) | NO (gap) |
| AnyData/AnyXML output | NO | Partial | NO (gap) |
| OpenConfig-specific transforms | NO | YES | NO (by design) |
| XPath evaluation | NO | NO | NO (by design) |

**Key insight:** gotya is already competitive with ygot on data-plane constructs (containers, lists, leaves, enums, unions, bits, choice). The remaining gaps are in the control-plane constructs (RPC, notification, action) and schema-modifying constructs (deviation, identityref). These are the P1 targets for the next milestone.

---

## YANG Constructs That Cause Codegen Pain

These are the constructs most commonly implemented incorrectly or incompletely in YANG generators. Included here to inform test planning and audit scope.

| Construct | Why It's Hard | Current gotya Status |
|-----------|---------------|----------------------|
| Union | Multiple valid types for one leaf; Go's type system needs a tagged union wrapper; JSON marshal must try members in order | Implemented but union member ordering and JSON decode ambiguity need audit |
| Choice/Case with shorthand case | RFC 7950 allows a single child node in a choice without an explicit case statement; the "shorthand case" must be synthesised as a Case node | Compiler synthesises shorthand case; AddChild() bug can silently drop it (tracked) |
| Augment targeting augmented nodes | If augment A adds a node, augment B can target that added node; resolution order matters | Convergence loop handles this but no test coverage; infinite loop possible |
| Typedef chain (typedef of typedef) | Multi-level typedef resolution is recursive; circular typedefs cause stack overflow | Depth limit missing (tracked in CONCERNS.md) |
| Identityref with cross-module bases | An identity can derive from a base in another module; the generator must resolve the prefix to find the base | Not implemented; identityref silently becomes *string |
| Leafref with relative path | Leafref paths can be relative using `../`; path walking must respect the schema tree | Stored as string, not resolved |
| Deviation on augmented nodes | A deviation can target a node that was added by an augment in a different module; requires deviation to be applied after augment resolution | Not implemented |
| Empty type | RFC 7950 §9.11: empty type means the leaf exists or doesn't; Go represents as `*bool` but semantics differ from `bool` leaf with true/false values | Mapped to `*bool`, which is correct for presence checking but may confuse users expecting JSON `[null]` encoding per RFC 7951 §6.9 |
| Bits with no members | Degenerate case; should be an error but generators must handle gracefully | Falls through to `*string` with a comment — acceptable fallback |

---

## Sources

- Direct code audit: `/home/user/repos/gotya/generator/golang/generator.go`, `/home/user/repos/gotya/generator/protobuf/generator.go`, `/home/user/repos/gotya/schema/schema.go`, `/home/user/repos/gotya/compiler/compiler.go`
- Project context: `.planning/PROJECT.md`, `.planning/codebase/CONCERNS.md`
- [RFC 7950 — The YANG 1.1 Data Modeling Language](https://www.rfc-editor.org/rfc/rfc7950) (official spec — HIGH confidence)
- [openconfig/ygot — YANG Go toolkit](https://github.com/openconfig/ygot) (ecosystem comparison — MEDIUM confidence, based on search results)
- [openconfig/goyang — YANG parser](https://github.com/openconfig/goyang) (ecosystem context — MEDIUM confidence)
- [ygot YANG-to-Protobuf transformations spec](https://github.com/openconfig/ygot/blob/master/docs/yang-to-protobuf-transformations-spec.md) (mapping reference — MEDIUM confidence)

---

*Feature research for: YANG-to-Go/Proto codegen library (gotya)*
*Researched: 2026-03-14*
