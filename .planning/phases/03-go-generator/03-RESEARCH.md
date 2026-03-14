# Phase 3: Go Generator - Research

**Researched:** 2026-03-14
**Domain:** Go code generation from compiled YANG schema (generator/golang)
**Confidence:** HIGH

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| GOGEN-01 | Identityref leaf types generate a typed Go `const` block for the identity hierarchy, not `*string` — cross-module base resolution supported | `schema.Module.Identities map[string]*Identity` is populated by the compiler. `Identity.Bases []string` holds base names. Generator must iterate this map, resolve cross-module prefixes via `mod.Imports`, emit a typed const block + `String()` method, and fix `mapYANGTypeToGo("identityref")` which currently falls through to `"*string"`. |
| GOGEN-02 | `anydata` and `anyxml` nodes emit a valid Go field (e.g., `interface{}` or `json.RawMessage`) — no silent node drops | `hasValidNodes` already returns `true` for `*schema.AnyXML` and `*schema.AnyData`, but `generateNode` and the top-level module struct loop have no case for them — they fall through the switch and emit nothing. Adding a case to both `generateNode` and the `GenerateDevice` module-struct loop closes the gap. |
| GOGEN-03 | `rpc`, `action`, and `notification` statements generate typed Go request/response structs — input and output containers emitted as nested structs | `schema.RPC`, `schema.Action`, `schema.Notification` exist with `*BaseNode` (children map). `schema.Input` and `schema.Output` are separate child types. Generator never calls `generateNode` for these types — they are silently skipped. Design decision: standalone `XxxInput` / `XxxOutput` structs (not Go methods) is the lower-API-risk approach for v1. |
| GOGEN-04 | Deviation statements are applied as a pre-pass before code generation — `deviate not-supported` removes nodes, `deviate replace` updates type/constraints, `deviate add/delete` updates properties | `schema.Module.Deviations []*Deviation` is populated. `Deviation.Deviates map[string][]ast.Statement` holds the deviate sub-statements keyed by kind (`"not-supported"`, `"replace"`, `"add"`, `"delete"`). A pre-pass function must walk `mod.Deviations`, resolve each target path against `mod.Nodes` (same path-walk logic as augment resolution), and mutate or remove the target node before `generateNode` runs. Must execute after augment resolution. |
</phase_requirements>

---

## Summary

Phase 3 fills four generator gaps in `generator/golang/generator.go`. The schema layer already compiles all four constructs correctly — `schema.Module.Identities`, `schema.Module.Deviations`, `schema.RPC`/`Action`/`Notification`, and `schema.AnyData`/`schema.AnyXML` are all populated. The generator simply never visits them. Each gap is therefore a contained addition to the generator, not a compiler change.

The most architecturally significant work is GOGEN-01 (identityref typed consts) because it requires cross-module prefix resolution: an identityref leaf's `Bases` values may carry a module prefix (e.g., `"oc-if:interface-type"`) that must be resolved through `mod.Imports` to find the target module's identity map. This is the only gap that requires multi-module coordination.

GOGEN-04 (deviation pre-pass) is the most correctness-sensitive because RFC 7950 §7.12 requires deviations to be applied after augment resolution. The compiler's augment convergence loop already runs before generators are invoked, so a pre-pass function that runs at the start of `Generate()` / `GenerateDevice()` respects that ordering constraint automatically.

**Primary recommendation:** Implement in order GOGEN-02 (smallest), GOGEN-04 (pre-pass, no API surface), GOGEN-01 (typed consts), GOGEN-03 (RPC structs, needs design decision locked first).

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `go/format` (stdlib) | Go 1.25.5 | Format and syntax-validate generated Go | Already in use in `GenerateDevice`; zero cost |
| `text/template` (stdlib) | Go 1.25.5 | Code emission templating | Already in use; consistent with existing generator |
| `encoding/json` (stdlib) | Go 1.25.5 | `json.RawMessage` type for anydata fields | Already imported in generated file header |
| `stretchr/testify` | v1.11.1 | Test assertions | Already in use |

### No New Dependencies
The "no new dependencies" project constraint (recorded in STATE.md decisions) holds for Phase 3. All four gaps are closed with changes to existing generator code and schema types already in the codebase.

---

## Architecture Patterns

### Where Each Gap Lives

```
generator/golang/generator.go
├── GenerateDevice()
│   ├── module-struct loop (lines ~213-237)  ← GOGEN-02: add AnyData/AnyXML case here
│   └── pre-pass hook (new, before generate loops) ← GOGEN-04: applyDeviations()
├── generateNode()                            ← GOGEN-02: add AnyData/AnyXML case
│                                             ← GOGEN-03: add RPC/Action/Notification case
├── mapYANGTypeToGo()                         ← GOGEN-01: add "identityref" case
└── generateIdentityConsts() (new)            ← GOGEN-01: emit typed const block
```

### Pattern 1: Identityref Typed Const Block (GOGEN-01)

**What:** For each leaf/leaf-list whose `Type.Name == "identityref"`, instead of emitting `*string`, emit a named Go type (`type XxxIdentity string`) with a `const` block listing all identity values that derive from the base, plus a `String()` method.

**Cross-module base resolution:** The `TypeDefinition.Bases []string` field holds base names as they appear in the YANG source — potentially prefixed (e.g., `"ot:optical-channel"`). The prefix must be resolved using `mod.Imports map[string]string` (prefix → module name) to locate the correct identity set in the target module's `Identities` map. If the base is unprefixed, it is local to the current module.

**Example shape of generated output:**
```go
// Source: schema.Module.Identities + TypeDefinition.Bases
type InterfaceTypeIdentity string

const (
    InterfaceTypeEthernetCsmacd InterfaceTypeIdentity = "ethernetCsmacd"
    InterfaceTypeSoftwareLoopback InterfaceTypeIdentity = "softwareLoopback"
    // ... one const per derived identity
)

func (v InterfaceTypeIdentity) String() string { return string(v) }
```

**Identity hierarchy traversal:** An identity can derive from another derived identity (multi-level hierarchy). The generator must do a transitive closure: starting from the root base identity, collect all identities in `mod.Identities` (and imported modules' identities) whose `Bases` slice contains the root. This is a BFS/DFS over the identity graph.

**mapYANGTypeToGo fix:** Add `case "identityref": return typeName + "Identity"` — but since the type name depends on context (the leaf name and module prefix), the actual fix is: in `generateField`, when `leaf.Type.Name == "identityref"`, compute the identity type name from the leaf name and call a new `generateIdentityConsts()` helper, rather than delegating to `mapYANGTypeToGo`.

### Pattern 2: AnyData/AnyXML Field Emission (GOGEN-02)

**What:** Emit `interface{}` or `json.RawMessage` for `*schema.AnyData` and `*schema.AnyXML` nodes. `json.RawMessage` is preferred — it preserves raw JSON bytes and is directly marshalable/unmarshalable without reflection.

**Two locations need patching:**

1. The module-struct field loop in `GenerateDevice` (around line 220) currently has `case *schema.Leaf, *schema.LeafList, *schema.Container, *schema.List` — add `case *schema.AnyData, *schema.AnyXML` emitting `json.RawMessage`.

2. `generateNode()` switch — add `case *schema.AnyData, *schema.AnyXML` that emits nothing (these are leaf-like terminals; the field was already emitted by the struct loop).

**Generated field shape:**
```go
// Source: RFC 7950 §7.10 (anydata), §7.11 (anyxml)
Config json.RawMessage `json:"config,omitempty" xml:"... config,omitempty"`
```

`json.RawMessage` is already imported in the generated file header (`"encoding/json"` is in the blanket import block). No new imports needed.

### Pattern 3: Deviation Pre-Pass (GOGEN-04)

**What:** Before any node generation, walk `mod.Deviations`, resolve each deviation's target path, and mutate or remove the target node in place.

**RFC 7950 §7.12 ordering constraint:** Deviations MUST be applied after augment resolution. The compiler's augment convergence loop runs at compile time, before any generator is invoked. Therefore a pre-pass at the start of `Generate()` / `GenerateDevice()` automatically satisfies this ordering — no special sequencing logic needed in the generator.

**Deviation target path resolution:** The `Deviation.NodeName` field holds the schema node path (e.g., `"/oc-if:interfaces/oc-if:interface/oc-if:config/oc-if:type"`). This is the same absolute-path format as augment target paths. The existing augment target resolution logic in `compiler.go` should be extracted or replicated in the generator's pre-pass.

**Four deviate kinds and their generator-level effects:**

| Deviate Kind | Schema Effect | Generator Action |
|---|---|---|
| `not-supported` | Remove node from parent | Delete node from `mod.Nodes` (or parent's `Children`) before generation |
| `replace` | Update type, default, mandatory, min/max-elements, etc. | Modify the target node's `TypeDefinition` or constraint fields in place |
| `add` | Add must, unique, default, mandatory, min/max-elements, units | Append to the target node's constraint fields |
| `delete` | Remove must, unique, default, units | Remove matching entries from the target node's constraint fields |

**Pre-pass function signature:**
```go
// applyDeviations mutates mod's node tree in place.
// Must be called before any generateNode invocation.
func applyDeviations(mod *schema.Module) error
```

**`deviate replace` type mutation:** The `Deviates["replace"]` slice contains `ast.Statement` nodes. The relevant sub-statements are `type`, `default`, `mandatory`, `min-elements`, `max-elements`, `units`. For `type`, the generator must re-resolve the new type name against the module's typedefs — the same `getType()` logic used by the compiler. This is the most complex deviate kind.

### Pattern 4: RPC/Action/Notification Structs (GOGEN-03)

**What:** Emit typed Go request/response structs for `*schema.RPC`, `*schema.Action`, and `*schema.Notification` nodes found in `mod.Nodes`.

**Design decision — standalone structs (not Go methods):** CONFIRMED for v1.
- `type XxxInput struct { ... }` and `type XxxOutput struct { ... }` as top-level named types
- No Go interface or method on any parent struct
- Rationale: methods bind the struct to a specific receiver type, creating API surface that is hard to change post-v1; standalone structs are more composable and match the Proto service block model that Phase 4 will need
- ygot uses a similar approach: typed structs with no receiver methods for RPC input/output

**Schema structure:** `schema.RPC`/`schema.Action`/`schema.Notification` all embed `*BaseNode` with a `Children map[string]Node`. The children will include `*schema.Input` and `*schema.Output` nodes, each of which in turn has their own `Children` (the actual leaf/container fields).

**Node location:** RPCs are stored as top-level children of the module — they appear in `mod.Nodes` alongside containers and lists. The `generateNode` switch must handle `*schema.RPC`, `*schema.Action`, `*schema.Notification`.

**Generated struct shape:**
```go
// Source: RFC 7950 §7.13 (rpc), §7.15 (action), §7.16 (notification)
// XxxInput holds the input parameters for the xxx RPC.
type XxxInput struct {
    Field1 *string `json:"field-1,omitempty" xml:"... field-1,omitempty"`
    // ... one field per child of the input node
}

// XxxOutput holds the output parameters for the xxx RPC.
type XxxOutput struct {
    Result *string `json:"result,omitempty" xml:"... result,omitempty"`
}

// XxxNotification holds the fields of the xxx notification.
type XxxNotification struct {
    // ... one field per child of the notification node
}
```

**Notification difference:** Notifications have no Input/Output children — their children are data nodes directly. Emit as a single flat struct.

**hasValidNodes extension:** The `hasValidNodes` function must be extended to return `true` for `*schema.RPC`, `*schema.Action`, `*schema.Notification` nodes so the module-level skip logic does not filter them out.

### Anti-Patterns to Avoid

- **Mutating the schema.Module during generation without a copy:** The deviation pre-pass mutates `mod.Nodes` in place. This is acceptable because `GenerateDevice` is the terminal consumer of the schema — but if the same module is ever passed to multiple generators, the mutation will be visible to the second generator. Document this behavior; if needed, deep-copy before mutation.
- **Resolving identity hierarchy lazily per-leaf:** The identity const block should be emitted once per identity base type, not once per leaf that references it. Use the existing `visited` map to skip re-emission.
- **Using `interface{}` instead of `json.RawMessage` for anydata:** `interface{}` round-trips through `encoding/json` as `map[string]interface{}`, losing type fidelity. `json.RawMessage` preserves the raw bytes and is the correct Go idiom for "arbitrary JSON content".
- **Emitting RPC structs in the Config/State loop:** The `GenerateDevice` function loops twice (Config, then State). RPCs are neither config nor state — they must be emitted outside those loops, once per module, not twice.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Go identifier sanitization | Custom regex replacer | Existing `toCamelCaseTitle()` in generator.go | Already handles YANG hyphenated names; extend for keyword collision |
| Identity BFS traversal | Custom graph library | Simple iterative BFS with `map[string]bool` visited set | Identity graphs are small; stdlib is sufficient |
| AST statement field extraction | Custom parser | Access `ast.Statement.Argument()` and `ast.Statement.SubStatements()` | Already used throughout the compiler for deviate sub-statement parsing |
| Type re-resolution for `deviate replace` | New type resolver | Refactor/call compiler's `getType()` logic | The logic already exists; duplication is the wrong choice |

---

## Common Pitfalls

### Pitfall 1: Identityref cross-module prefix not resolved
**What goes wrong:** An identity base value like `"oc-types:IANA-IF-TYPE"` uses the module prefix `oc-types`. If the generator looks up `"oc-types:IANA-IF-TYPE"` directly in the current module's `Identities` map it finds nothing and silently emits `*string`.
**Why it happens:** `mod.Imports` maps prefix → module name. The target module's `Identities` must be accessed through the loaded schema modules slice.
**How to avoid:** In `generateIdentityConsts`, split on `:`, resolve prefix via `mod.Imports`, then locate the target module in the `[]*schema.Module` slice passed to `GenerateDevice`.
**Warning signs:** Generated file has `*string` fields for leaf types that reference cross-module identities.

### Pitfall 2: Deviation path resolution using YANG prefixes
**What goes wrong:** Deviation target paths use module prefixes (e.g., `/oc-if:interfaces/oc-if:interface`). Direct string lookup in `mod.Nodes` by path segment fails because `mod.Nodes` keys are unprefixed node names.
**Why it happens:** The deviation's `NodeName` stores the raw path from the YANG source, including prefixes.
**How to avoid:** Strip the prefix from each path segment before looking up in `Children`. Same logic as augment target resolution in the compiler.

### Pitfall 3: RPC structs emitted twice (Config + State loop)
**What goes wrong:** If RPC/Action/Notification handling is added inside the `for _, treeType := range []string{"Config", "State"}` loop in `GenerateDevice`, duplicate struct declarations fail `go/format.Source()`.
**Why it happens:** The double-pass loop exists for Config/State tree splitting; RPCs are stateless.
**How to avoid:** Add a separate post-loop pass over `mod.Nodes` for RPC/Action/Notification, guarded by the `visited` map.

### Pitfall 4: AnyData field missing from `hasValidNodes`
**What goes wrong:** `hasValidNodes` already has `case *schema.AnyXML, *schema.AnyData: return true` — this is correct. But the module-struct field emission loop in `GenerateDevice` (lines ~213-237) only switches on `*schema.Leaf`, `*schema.LeafList`, `*schema.Container`, `*schema.List`. If the AnyData/AnyXML case is added to `generateNode` but not to this struct field loop, fields are omitted from the parent struct while still not being emitted as nested types.
**How to avoid:** Both sites need the new case: the struct field loop AND `generateNode`.

### Pitfall 5: Identity const block name collision across modules
**What goes wrong:** Two modules can define identities with the same name. Emitting const blocks without a module prefix in the type name causes a compile error.
**How to avoid:** Prefix the identity type name with the CamelCase module name: `type <ModuleName><IdentityName>Identity string`.

### Pitfall 6: `deviate not-supported` on a node with active children
**What goes wrong:** Removing a container from `mod.Nodes` that has child nodes still registered in the `visited` map can produce orphaned struct declarations — structs that are declared but never referenced.
**How to avoid:** When applying `deviate not-supported` to a container/list, recursively collect all descendant node names and remove them from the `visited` map (or track which nodes are deactivated before visiting).

---

## Code Examples

### Deviation Deviates map structure
```go
// Source: schema/schema.go Deviation struct (confirmed by code audit)
type Deviation struct {
    *BaseNode
    Deviates map[string][]ast.Statement
    // Keys: "not-supported", "replace", "add", "delete"
    // Values: slice of ast.Statement sub-statements for that deviate kind
}
```

### Identity struct in schema
```go
// Source: schema/schema.go (confirmed by code audit)
type Identity struct {
    *BaseNode
    Bases []string  // base identity names (may be prefixed: "prefix:name")
}
// Accessed via: mod.Identities map[string]*Identity
```

### RPC/Action/Notification in schema
```go
// Source: schema/schema.go (confirmed by code audit)
// All three embed *BaseNode — children contain *schema.Input and *schema.Output
type RPC struct{ *BaseNode }
type Action struct{ *BaseNode }
type Notification struct{ *BaseNode }
type Input struct{ *BaseNode }   // child key: "input"
type Output struct{ *BaseNode }  // child key: "output"
```

### mapYANGTypeToGo default fallthrough (the bug)
```go
// Source: generator/golang/generator.go:605 (confirmed by code audit)
// "identityref" hits the default case and returns "*string"
func mapYANGTypeToGo(yangType string) string {
    switch yangType {
    // ... typed cases ...
    default:
        return "*string"  // identityref, leafref, and unknown types all land here
    }
}
```

### AnyData/AnyXML recognition in hasValidNodes (already correct)
```go
// Source: generator/golang/generator.go:81 (confirmed by code audit)
case *schema.Leaf, *schema.LeafList, *schema.AnyXML, *schema.AnyData:
    return true
// The node IS recognised as valid — it just never gets emitted
```

---

## State of the Art

| Old Approach | Current Approach | Impact for Phase 3 |
|--------------|------------------|---------------------|
| identityref → `*string` | identityref → typed const + `String()` method | GOGEN-01: breaks existing golden files; new golden files required |
| AnyData/AnyXML silently dropped | AnyData/AnyXML → `json.RawMessage` field | GOGEN-02: corpus test modules with anydata now produce non-empty output |
| RPC/Action/Notification not emitted | Standalone typed Input/Output structs | GOGEN-03: new top-level types; no existing output to break |
| Deviations parsed but not applied | Deviation pre-pass before generation | GOGEN-04: generated output changes for any module with deviations |

---

## Open Questions

1. **Identity hierarchy depth: how many levels in practice?**
   - What we know: RFC 7950 allows arbitrarily deep identity derivation; OpenConfig uses 2-3 levels typically
   - What's unclear: Whether the corpus in `test/assets/yangs/` has any identities deeper than 2 levels
   - Recommendation: Implement full BFS transitive closure; it's not significantly harder than single-level and is correct by construction

2. **`deviate replace` type re-resolution: refactor compiler or duplicate?**
   - What we know: The compiler's `getType()` function resolves typedef chains; the deviation pre-pass needs the same logic to apply a new type
   - What's unclear: Whether `getType()` is accessible from the generator package without creating a circular dependency
   - Recommendation: Extract a `ResolveType(td *ast.Statement, typedefs map[string]ast.Statement) (*schema.TypeDefinition, error)` function to a shared internal package, or accept that `deviate replace type` is MEDIUM complexity and deserves its own planner wave

3. **RPC nodes in `mod.Nodes` vs. attached to container children?**
   - What we know: `schema.RPC` embeds `*BaseNode` and is compiled into the module's node tree; the compiler calls `mod.AddNode()` for top-level RPCs
   - What's unclear: Whether actions (which attach to container nodes per RFC 7950 §7.15) end up in the container's `Children` map or in `mod.Nodes`
   - Recommendation: Before writing the generator case, confirm in compiler.go where `schema.Action` nodes are added (likely `parent.AddChild()` not `mod.AddNode()`)

---

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | `stretchr/testify` v1.11.1 + stdlib `go/format.Source` |
| Config file | `go.mod` (no separate test config) |
| Quick run command | `go test ./generator/golang/... -run TestGo` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| GOGEN-01 | Identityref leaf emits typed const block, not `*string` | golden | `go test ./generator/golang/... -run TestIdentityref` | ❌ Wave 0 |
| GOGEN-01 | Cross-module base resolution finds identity in imported module | unit | `go test ./generator/golang/... -run TestIdentityrefCrossModule` | ❌ Wave 0 |
| GOGEN-02 | AnyData node emits `json.RawMessage` field | golden | `go test ./generator/golang/... -run TestAnyData` | ❌ Wave 0 |
| GOGEN-02 | AnyXML node emits `json.RawMessage` field | golden | `go test ./generator/golang/... -run TestAnyXML` | ❌ Wave 0 |
| GOGEN-03 | RPC emits Input and Output structs | golden | `go test ./generator/golang/... -run TestRPC` | ❌ Wave 0 |
| GOGEN-03 | Action emits Input and Output structs | golden | `go test ./generator/golang/... -run TestAction` | ❌ Wave 0 |
| GOGEN-03 | Notification emits flat data struct | golden | `go test ./generator/golang/... -run TestNotification` | ❌ Wave 0 |
| GOGEN-04 | `deviate not-supported` removes node from generated output | golden | `go test ./generator/golang/... -run TestDeviationNotSupported` | ❌ Wave 0 |
| GOGEN-04 | `deviate replace` updates field type | golden | `go test ./generator/golang/... -run TestDeviationReplace` | ❌ Wave 0 |
| GOGEN-04 | `deviate add/delete` updates constraints | golden | `go test ./generator/golang/... -run TestDeviationAddDelete` | ❌ Wave 0 |
| GOGEN-04 | Deviation runs after augment resolution (ordering) | integration | `go test ./... -run TestCorpusDeviationOrdering` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./generator/golang/... -run TestGo`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `generator/golang/generator_identityref_test.go` — covers GOGEN-01
- [ ] `generator/golang/generator_anydata_test.go` — covers GOGEN-02
- [ ] `generator/golang/generator_rpc_test.go` — covers GOGEN-03
- [ ] `generator/golang/generator_deviation_test.go` — covers GOGEN-04
- [ ] `testdata/go/identityref/` — golden files directory for GOGEN-01
- [ ] `testdata/go/anydata/` — golden files directory for GOGEN-02
- [ ] `testdata/go/rpc/` — golden files directory for GOGEN-03
- [ ] `testdata/go/deviation/` — golden files directory for GOGEN-04

---

## Sources

### Primary (HIGH confidence)
- Direct code audit: `/home/user/repos/gotya/generator/golang/generator.go` — confirmed `mapYANGTypeToGo` default fallthrough, `hasValidNodes` AnyData recognition, missing generateNode cases (2026-03-14)
- Direct code audit: `/home/user/repos/gotya/schema/schema.go` — confirmed `Deviation.Deviates`, `Identity.Bases`, `RPC`/`Action`/`Notification`/`Input`/`Output` struct shapes, `Module.Identities`/`Module.Deviations` fields (2026-03-14)
- RFC 7950 §7.10 (anydata), §7.11 (anyxml), §7.12 (deviation), §7.13 (rpc), §7.15 (action), §7.16 (notification), §7.17 (identity), §7.3 (identityref): https://www.rfc-editor.org/rfc/rfc7950
- `.planning/research/FEATURES.md` — feature gap analysis (HIGH confidence, 2026-03-14)
- `.planning/research/SUMMARY.md` — architectural findings and known flags (HIGH confidence, 2026-03-14)

### Secondary (MEDIUM confidence)
- `.planning/STATE.md` — accumulated decisions, blockers, no-new-dependencies constraint
- openconfig/ygot identityref approach (standalone typed enum, not method): https://github.com/openconfig/ygot

---

## Metadata

**Confidence breakdown:**
- GOGEN-02 (anydata): HIGH — gap and fix confirmed by direct code audit; two-line addition
- GOGEN-04 (deviation pre-pass): HIGH — schema structures confirmed; RFC ordering constraint clear; deviate-replace type re-resolution is the complexity point (MEDIUM sub-confidence)
- GOGEN-01 (identityref): HIGH for single-module case; MEDIUM for cross-module base resolution (design not previously specified; now specified above)
- GOGEN-03 (RPC structs): HIGH — schema structures confirmed; design decision (standalone structs) resolved; action attachment point (mod.Nodes vs container children) flagged as open question requiring compiler.go verification before implementation

**Research date:** 2026-03-14
**Valid until:** 2026-04-14 (stable domain; schema types are fixed)
