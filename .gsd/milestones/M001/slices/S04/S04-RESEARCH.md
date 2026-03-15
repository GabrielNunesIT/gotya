# Phase 4: Protobuf Generator - Research

**Researched:** 2026-03-15
**Domain:** Protobuf v3 generator, YANG RFC 7950 statement types, CEL validation, golden file testing
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- Both `anydata` and `anyxml` YANG nodes emit `google.protobuf.Any` — same type for both, no differentiation
- `import "google/protobuf/any.proto";` added to generated output only when the module contains anydata or anyxml nodes (not unconditionally)
- Add golden file tests for the Proto generator in this phase (deferred from Phase 2)
- Use the same corpus as the Go generator: `test/assets/yangs/` (205 BBF/IETF/IEEE YANG files)
- Run the full `GenerateDevice` pipeline (parse → compile → GenerateDevice), mirroring real usage
- `-update` flag pattern (go test -run TestGoldenProto ./... -update regenerates golden files)
- Golden files stored in `testdata/` at repo root (consistent with Phase 2 decision)
- When `GenerateCELValidation` is enabled and a CEL annotation path doesn't correspond to a real schema node: return an error and do not write the output file (same philosophy as go/format.Source() in the Go generator)
- Collect all invalid paths before returning — one error pass shows all problems, not just the first
- Prefer one `service` block per YANG module (named after the module) so RPCs from different modules don't collide in a single service definition

### Claude's Discretion

- Exact CEL path resolution mechanism (how to walk the schema to verify a field path)
- Coverage document format — a Markdown table with columns: YANG statement, Proto mapping, status (supported/unsupported/out-of-scope), notes
- Whether to validate CEL paths only when GenerateCELValidation=true or always

### Deferred Ideas (OUT OF SCOPE)

- None — discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| PBGEN-01 | RFC 7950 statement coverage matrix exists — every YANG statement type is explicitly marked as supported, unsupported, or out-of-scope in a `docs/proto-coverage.md` document | RFC 7950 statement list (sections 7.1–7.21) enumerated below; all 22 top-level statements classified |
| PBGEN-02 | `anydata` and `anyxml` nodes emit a valid Protobuf field (`google.protobuf.Any`) — no silent node drops | `*schema.AnyData` and `*schema.AnyXML` types confirmed in schema.go; `generateField` switch missing these cases; conditional import logic pattern established |
| PBGEN-03 | `rpc` and `action` statements emit Protobuf `service` block definitions with `rpc` methods referencing typed request/response messages | `*schema.RPC`, `*schema.Action` have `input`/`output` children in `GetBase().Children`; `generateNode` switch missing these cases; service block naming decision: one service per module |
| PBGEN-04 | Generated CEL annotation paths validated against compiled schema after generation — invalid paths produce a generation error before the file is written | `buildValidateOptions` in `validations.go` builds CEL annotation strings; post-generation walk needed to verify each CEL path field reference exists in schema; error collection before write mirrors Go generator `format.Source` pattern |
</phase_requirements>

---

## Summary

Phase 4 fills four specific gaps in the existing Protobuf generator. The generator already handles `Container`, `List`, `Leaf`, `LeafList`, `Choice`/`Case`, enums, CEL validation options, and `GeneratePopulateDefault`. Three new node-handling gaps must be closed (anydata/anyxml, rpc/action/notification), one new validation pass must be added (CEL path verification), a coverage document must be written, and golden file tests must be added to lock in correctness.

All four YANG schema types involved (`*schema.AnyData`, `*schema.AnyXML`, `*schema.RPC`, `*schema.Action`) are already emitted by the compiler and present in `schema.go` — no schema changes are needed. RPC/Action nodes always have `"input"` and `"output"` children in `GetBase().Children` (compiler guarantees this by adding empty nodes when absent). Notification nodes have only direct children (no input/output wrapper).

The golden file pattern does not exist yet for the proto generator. The `testdata/` directory at repo root does not exist — it must be created as part of Wave 0. The existing `test/corpus_test.go` provides an exact structural model (sequential load + parallel subtests, corpusLoader, `-update` flag) that the proto golden file test replicates.

**Primary recommendation:** Implement in TDD order — stubs RED → anydata/anyxml → rpc/action/notification → CEL path validation → golden files → coverage doc. Each task is independent and unblocks the next.

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/gotya/gotya/schema` | (project-internal) | Schema node types for all YANG constructs | The only schema representation in the project |
| `github.com/stretchr/testify` | v1.11.1 | Assertions in tests | Already used in all test files |
| stdlib `strings`, `fmt`, `io`, `sort` | Go stdlib | Output formatting and I/O | Already in use in generator.go |
| stdlib `flag` | Go stdlib | `-update` flag for golden file regeneration | Standard Go testing pattern; no new dep |
| stdlib `os`, `path/filepath` | Go stdlib | Golden file read/write | Already used in device_test.go |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `google/protobuf/any.proto` | proto import (not Go dep) | Type for anydata/anyxml fields | When module contains anydata or anyxml nodes |
| `protoc` v3.21.12 | System binary | Proto syntax validation | Optional: can use for golden file validation if desired; not a build dep |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `google.protobuf.Any` for anydata/anyxml | `bytes` | Decision is locked: use `google.protobuf.Any` |
| One service per module | One global service | Decision is locked: one per module prevents name collisions |
| Collect all CEL errors before returning | Fail-fast on first error | Decision is locked: collect all, mirrors Phase 2 error accumulation |

**Installation:** No new Go dependencies needed. All tools are in stdlib or already in go.mod.

---

## Architecture Patterns

### Recommended Project Structure
```
generator/protobuf/
├── generator.go          # Primary file — add anydata/anyxml and rpc/action/notification cases
├── validations.go        # CEL constraint builder — add post-generation path validation here
├── generator_test.go     # Existing unit tests — add PBGEN-02, PBGEN-03, PBGEN-04 stubs
├── validations_test.go   # Existing validation tests — add CEL path validation tests
└── device_test.go        # Existing corpus test — keep as-is (not golden file test)

testdata/                 # Create at repo root (does not exist yet)
└── proto/                # Golden .proto files, one per corpus YANG module
    ├── alb-co-alarm-types.proto
    └── ...

test/
└── proto_golden_test.go  # New golden file test for proto generator

docs/
└── proto-coverage.md     # New coverage matrix document (PBGEN-01)
```

### Pattern 1: Conditional Import for google.protobuf.Any

**What:** Scan module nodes for anydata/anyxml before emitting the header; add `import "google/protobuf/any.proto";` only when needed.
**When to use:** In `Generate()` and `GenerateDevice()` before the header import block.
**Example:**
```go
// In GenerateDevice — scan all modules before writing header
func containsAnyNode(modules []*schema.Module) bool {
    for _, mod := range modules {
        if moduleHasAny(mod.Nodes) {
            return true
        }
    }
    return false
}

func moduleHasAny(nodes map[string]schema.Node) bool {
    for _, node := range nodes {
        switch node.(type) {
        case *schema.AnyData, *schema.AnyXML:
            return true
        }
        if len(node.GetChildren()) > 0 {
            if moduleHasAny(node.GetChildren()) {
                return true
            }
        }
    }
    return false
}
```

### Pattern 2: AnyData/AnyXML in generateField Switch

**What:** Add a case for `*schema.AnyData, *schema.AnyXML` in the `generateField` switch.
**When to use:** Alongside the existing Leaf/LeafList/Container/List/Choice cases.
**Example:**
```go
case *schema.AnyData, *schema.AnyXML:
    if _, err := fmt.Fprintf(w, "\tgoogle.protobuf.Any %s = %d;\n", fieldName, *fieldIndex); err != nil {
        return fmt.Errorf("write err: %w", err)
    }
    *fieldIndex++
```

### Pattern 3: RPC/Action in generateNode — Service Block

**What:** Add `*schema.RPC` and `*schema.Action` cases to `generateNode` to emit typed `message` blocks for input/output, and a `service` block per module in `GenerateDevice`.
**When to use:** `generateNode` handles individual RPC/Action message generation; `GenerateDevice` collects and emits the `service` block after all messages.

**Service block naming:** Module name converted to CamelCase, e.g., `ietf-interfaces` → `IetfInterfacesService`.

**Example — message generation:**
```go
case *schema.RPC, *schema.Action:
    rpcPrefix := toCamelCaseTitle(n.Name())
    if inputNode, ok := n.GetBase().Children["input"]; ok {
        inputMsgName := rpcPrefix + "Input"
        if !visited[inputMsgName] {
            visited[inputMsgName] = true
            if inp, ok2 := inputNode.(*schema.Input); ok2 {
                if err := g.generateMessage(inputMsgName, nil, inp.Children, w, visited); err != nil {
                    return err
                }
            }
        }
    }
    if outputNode, ok := n.GetBase().Children["output"]; ok {
        outputMsgName := rpcPrefix + "Output"
        if !visited[outputMsgName] {
            visited[outputMsgName] = true
            if out, ok2 := outputNode.(*schema.Output); ok2 {
                if err := g.generateMessage(outputMsgName, nil, out.Children, w, visited); err != nil {
                    return err
                }
            }
        }
    }
    return nil
```

**Example — service block emission (in GenerateDevice, post-message loop):**
```go
// Emit one service block per module if the module contains RPCs or actions
for _, mod := range modules {
    var rpcMethods []string
    for _, name := range getSortedChildNames(mod.Nodes) {
        node := mod.Nodes[name]
        if n, ok := node.(*schema.RPC); ok {
            rpcMethods = append(rpcMethods, fmt.Sprintf(
                "\trpc %s(%sInput) returns (%sOutput);\n",
                toCamelCaseTitle(n.Name()), toCamelCaseTitle(n.Name()), toCamelCaseTitle(n.Name()),
            ))
        }
        // same for *schema.Action
    }
    if len(rpcMethods) > 0 {
        svcName := toCamelCaseTitle(mod.Name) + "Service"
        fmt.Fprintf(w, "service %s {\n", svcName)
        for _, m := range rpcMethods {
            fmt.Fprint(w, m)
        }
        fmt.Fprintf(w, "}\n\n")
    }
}
```

### Pattern 4: Notification in generateNode

**What:** Notification has no input/output — its direct children are the message fields.
**Example:**
```go
case *schema.Notification:
    notifMsgName := toCamelCaseTitle(n.Name()) + "Notification"
    if !visited[notifMsgName] {
        visited[notifMsgName] = true
        if err := g.generateMessage(notifMsgName, n.Description, n.GetBase().Children, w, visited); err != nil {
            return err
        }
    }
    return nil
```

### Pattern 5: CEL Path Validation (PBGEN-04)

**What:** After building CEL annotation strings in `buildValidateOptions`, a separate post-generation walk verifies each leaf field path referenced in CEL annotations exists in the compiled schema. This is a pre-write pass, not integrated into `buildValidateOptions` itself.

**Implementation approach:** The simplest correct mechanism is to track during generation which field names received CEL annotations, then walk the compiled schema to verify each field path corresponds to a real leaf. The `GenerateDevice` implementation already has access to the `[]*schema.Module` — a helper that recursively walks `mod.Nodes` looking up each path segment suffices for v1 scope.

**Error collection pattern (mirrors Phase 1/2):**
```go
// collectCELPathErrors returns all invalid CEL annotation paths across modules.
// Returns nil if all paths are valid.
func collectCELPathErrors(modules []*schema.Module) []string {
    var errs []string
    for _, mod := range modules {
        for path, found := range collectAnnotatedPaths(mod) {
            if !found {
                errs = append(errs, fmt.Sprintf("module %s: CEL path %q has no corresponding schema node", mod.Name, path))
            }
        }
    }
    sort.Strings(errs) // deterministic order
    return errs
}
```

**Error return (pre-write, all at once):**
```go
if g.Options.GenerateCELValidation {
    if errs := collectCELPathErrors(modules); len(errs) > 0 {
        return fmt.Errorf("CEL path validation failed:\n%s", strings.Join(errs, "\n"))
    }
}
```

### Pattern 6: Golden File Tests

**What:** Mirror `test/corpus_test.go` structure for proto output.
**Location:** `test/proto_golden_test.go` (same package `test`).
**Golden file location:** `testdata/proto/` at repo root.

**Key mechanics:**
- `-update` flag: `var update = flag.Bool("update", false, "regenerate golden files")`
- Sequential load via corpusLoader (already available in `test/` package)
- Per-module subtest writes to `testdata/proto/<modname>.proto`
- On normal run: compare `GenerateDevice` output to stored golden file
- On `-update`: write new golden file

**Example structure:**
```go
// In test/proto_golden_test.go
var update = flag.Bool("update", false, "regenerate golden .proto files")

func TestGoldenProto(t *testing.T) {
    // ... load modules via corpusLoader (reuse from corpus_test.go)
    goldenDir := filepath.Join("..", "testdata", "proto")
    if *update {
        _ = os.MkdirAll(goldenDir, 0750)
    }
    for _, nm := range modules {
        nm := nm
        t.Run(nm.name, func(t *testing.T) {
            t.Parallel()
            var buf bytes.Buffer
            gen := protobuf.New(&protobuf.Options{
                PackageName: "corpus",
                RootName:    "Device",
            })
            if err := gen.GenerateDevice([]*schema.Module{nm.mod}, &buf); err != nil {
                t.Errorf("GenerateDevice(%s): %v", nm.name, err)
                return
            }
            goldenPath := filepath.Join(goldenDir, nm.name+".proto")
            if *update {
                if err := os.WriteFile(goldenPath, buf.Bytes(), 0600); err != nil {
                    t.Fatalf("write golden: %v", err)
                }
                return
            }
            golden, err := os.ReadFile(goldenPath)
            if os.IsNotExist(err) {
                t.Errorf("golden file missing for %s — run with -update to create", nm.name)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, string(golden), buf.String())
        })
    }
}
```

**Note:** The corpusLoader is already defined in `test/corpus_test.go` within the `test` package. The proto golden test in the same package can reuse it directly without duplication.

### Anti-Patterns to Avoid

- **Silent node drops:** Every `generateField` call must handle every node type — unmatched types in the switch fall through without emitting a field, silently dropping nodes. Add explicit cases or a default that logs/skips with a comment.
- **Unconditional `google/protobuf/any.proto` import:** Only add it when the module actually contains anydata or anyxml. Unconditional import causes proto lint errors on modules without Any fields.
- **Single-error CEL validation:** Returning on the first invalid path hides subsequent problems. Collect all errors, then return.
- **Service block duplication:** Use the `visited` map to guard service block emission; a module name appearing in multiple passes must not produce duplicate `service` blocks.
- **Notification treated as RPC:** Notification has no input/output; emitting `rpc NotifName(NotifNameInput) returns (NotifNameOutput)` is wrong. Notification maps to a standalone message only, not to a service rpc entry.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Proto syntax validation of generated output | Custom regex checker | `protoc` binary (already at `/usr/bin/protoc`) | protoc catches all syntax errors including field number conflicts, import resolution, reserved names |
| CamelCase/snake_case conversion | New helpers | `toCamelCaseTitle`, `convertToSnakeCase` (already in generator.go) | Both helpers are tested and handle YANG hyphenated names correctly |
| Schema node traversal | Custom walk | `node.GetChildren()` + type switch | BaseNode.GetChildren() is the canonical traversal API |
| Test infrastructure for golden files | Custom diffing | `assert.Equal` from testify + `os.ReadFile`/`os.WriteFile` | Already used throughout; no new dep needed |

**Key insight:** The generator already has all needed helpers. The gaps are purely in the `generateField` and `generateNode` switch statements, plus the new service block and CEL validation passes.

---

## Common Pitfalls

### Pitfall 1: Missing `google.protobuf.Any` Import Causes Proto Syntax Error
**What goes wrong:** Output contains `google.protobuf.Any SomeField = 1;` but no `import "google/protobuf/any.proto";` — protoc rejects the file.
**Why it happens:** Import is only needed conditionally; easy to add the field type without adding the import.
**How to avoid:** Scan for anydata/anyxml before writing header. Write import before any message blocks.
**Warning signs:** `protoc` validation failure on generated output mentioning undefined type `google.protobuf.Any`.

### Pitfall 2: RPC/Action Nodes Appearing as Module-Level Fields
**What goes wrong:** `generateField` is called for RPC/Action nodes at module level in `GenerateDevice`, emitting them as proto message fields instead of service rpc methods.
**Why it happens:** `GenerateDevice` iterates `mod.Nodes` and calls `generateField` on each child — RPC/Action must be excluded from this field loop (same as Go generator's `continue` for these types).
**How to avoid:** Add explicit `continue` or skip check for `*schema.RPC`, `*schema.Action`, `*schema.Notification` in the field emission loop, parallel to Go generator pattern.
**Warning signs:** Generated output contains `TestModule test_module = 2;` for an RPC named `test`.

### Pitfall 3: Empty Input/Output Messages Cause Proto Field Number Issues
**What goes wrong:** An RPC with no input/output leaf children emits `message ResetInput {}` which is valid proto, but the service block reference `rpc Reset(ResetInput) returns (ResetOutput)` requires both messages to exist.
**Why it happens:** Compiler guarantees input and output children exist (adds empty ones if absent), so this is actually safe — but the message must still be emitted even if it has no fields.
**How to avoid:** Always emit the Input/Output message even when children map is empty.
**Warning signs:** `protoc` error "message X not defined" on service rpc method.

### Pitfall 4: CEL Path Resolution Scope
**What goes wrong:** CEL annotation paths use proto field names (snake_case), but schema lookup uses YANG names (hyphenated). Walking `mod.Nodes["some-field"]` when the CEL path says `"some_field"` finds nothing.
**Why it happens:** `buildValidateOptions` produces CEL annotations referencing proto field names, but schema nodes are keyed by YANG name.
**How to avoid:** The CEL path validation must reverse the snake_case conversion: `strings.ReplaceAll(protoFieldName, "_", "-")` before looking up in schema. Alternatively, build a proto-name→schema-node map during generation.
**Warning signs:** All CEL paths report as invalid even for nodes that exist in the schema.

### Pitfall 5: Golden File Test Race Condition
**What goes wrong:** Parallel subtests all try to write to `testdata/proto/` simultaneously, causing intermittent write failures.
**Why it happens:** `os.MkdirAll` and `os.WriteFile` in parallel subtests race on directory creation.
**How to avoid:** Create `testdata/proto/` before launching subtests (in the parent test function, not inside the subtest). Check existing `corpus_test.go` pattern — `os.MkdirAll` is called once before the parallel subtest loop.
**Warning signs:** Flaky test failures on `-update` runs with "no such file or directory".

### Pitfall 6: Service Block Emitted Inside Message Block
**What goes wrong:** `service IetfInterfacesService { ... }` appears inside `message IetfInterfaces { ... }` because service emission is inserted mid-loop.
**Why it happens:** If service blocks are emitted in the same loop as message blocks, they appear at the wrong nesting level.
**How to avoid:** Emit all service blocks in a separate post-loop pass in `GenerateDevice`, after all messages are complete. Pattern matches Go generator's post-loop RPC/Action struct emission.
**Warning signs:** `protoc` parse error "nested services are not allowed in proto3".

---

## Code Examples

Verified patterns from existing codebase:

### Existing generateField Switch (Addition Points)
```go
// Source: generator/protobuf/generator.go lines 327-451
func (g *ProtoGenerator) generateField(node schema.Node, w io.Writer, fieldIndex *int) error {
    // ...
    switch n := node.(type) {
    case *schema.Leaf:         // handled
    case *schema.LeafList:     // handled
    case *schema.Container:    // handled
    case *schema.List:         // handled
    case *schema.Choice:       // handled
    // MISSING: *schema.AnyData, *schema.AnyXML  ← add here (PBGEN-02)
    // MISSING: *schema.RPC, *schema.Action, *schema.Notification ← add skip here
    }
}
```

### Existing generateNode Switch (Addition Points)
```go
// Source: generator/protobuf/generator.go lines 194-202
func (g *ProtoGenerator) generateNode(node schema.Node, w io.Writer, visited map[string]bool) error {
    switch n := node.(type) {
    case *schema.Container:    // handled
    case *schema.List:         // handled
    // MISSING: *schema.RPC, *schema.Action  ← add here (PBGEN-03)
    // MISSING: *schema.Notification         ← add here (PBGEN-03)
    // *schema.AnyData, *schema.AnyXML are terminal — no children to recurse into
    }
    return nil
}
```

### How RPC Input/Output Children Are Accessed
```go
// Source: generator/golang/generator.go lines 425-451
// Pattern confirmed: input/output accessed as GetBase().Children["input"/"output"]
if inputNode, ok := n.GetBase().Children["input"]; ok {
    if inp, ok2 := inputNode.(*schema.Input); ok2 {
        // inp.Children contains the input leaf fields
    }
}
```

### Compiler Guarantees Empty Input/Output Nodes
```go
// Source: compiler/compiler.go lines 457-466
// Compiler always adds input/output children if absent:
if rpcNode.GetChildren()["input"] == nil {
    if err := rpcNode.AddChild(schema.NewInput()); err != nil { ... }
}
// So n.GetBase().Children["input"] is ALWAYS non-nil for *schema.RPC and *schema.Action
```

### Go Generator Error Collection Pattern (CEL validation model)
```go
// Source: generator/golang/generator.go lines 319-325
// format.Source is run pre-write; errors abort before any output is written
formatted, err := format.Source(buf.Bytes())
if err != nil {
    return fmt.Errorf("generated Go has invalid syntax: %w", err)
}
if _, err := w.Write(formatted); err != nil { ... }
```

### Corpus Test Load Pattern (golden file test base)
```go
// Source: test/corpus_test.go lines 109-170
// Sequential load + parallel subtests pattern to avoid data race on corpusLoader cache
var modules []namedModule
for _, entry := range entries {
    schemaMod, loadErr := loader.Load(modName)
    if loadErr != nil {
        t.Logf(...)  // log, not fail — cross-module errors are expected in corpus
    }
    if schemaMod != nil {
        modules = append(modules, namedModule{...})
    }
}
for _, nm := range modules {
    nm := nm
    t.Run(nm.name, func(t *testing.T) {
        t.Parallel()
        // generation and assertion
    })
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| anydata/anyxml → silently dropped | anydata/anyxml → `google.protobuf.Any` | This phase | No silent drops; any.proto import conditional |
| rpc/action → silently ignored | rpc/action → `service` block + request/response messages | This phase | gRPC service stubs become generatable |
| CEL paths → unvalidated | CEL paths → validated post-generation, error before write | This phase | Invalid paths caught at generation time |
| No proto golden tests | Golden file regression suite in `testdata/proto/` | This phase | Regeneration regressions caught in CI |

**No deprecated approaches to replace** — this phase adds capabilities that don't currently exist.

---

## RFC 7950 Statement Coverage Matrix (for PBGEN-01)

The coverage document `docs/proto-coverage.md` must enumerate every RFC 7950 statement section. Based on sections 7.1–7.21 of the RFC (confirmed from `docs/rfc7950.txt`):

| YANG Statement | RFC Section | Proto Mapping | Status | Notes |
|----------------|-------------|---------------|--------|-------|
| module | 7.1 | package + top-level message | supported | Module name → proto package + CamelCase message |
| submodule | 7.2 | (merged by compiler) | out-of-scope | Compiler resolves includes; generator sees flat module |
| typedef | 7.3 | resolved to primitive | supported | Typedef expansion happens at compiler level |
| type | 7.4 | scalar proto type | supported | mapYANGTypeToProto() handles string/int/bool/bytes etc. |
| container | 7.5 | message | supported | Recursive message generation |
| leaf | 7.6 | scalar field | supported | CEL constraints via buildValidateOptions |
| leaf-list | 7.7 | repeated scalar field | supported | `repeated` prefix |
| list | 7.8 | repeated message field | supported | `repeated MessageType field` |
| choice | 7.9 | oneof | supported | Case children as CaseMessage types |
| anydata | 7.10 | google.protobuf.Any | supported (this phase) | PBGEN-02; conditional import |
| anyxml | 7.11 | google.protobuf.Any | supported (this phase) | PBGEN-02; same as anydata |
| grouping | 7.12 | (resolved by compiler) | out-of-scope | Uses expansion at compiler level |
| uses | 7.13 | (resolved by compiler) | out-of-scope | Compiler expands groupings before generator |
| rpc | 7.14 | service rpc + Input/Output messages | supported (this phase) | PBGEN-03 |
| action | 7.15 | service rpc + Input/Output messages | supported (this phase) | PBGEN-03 |
| notification | 7.16 | standalone message (not in service) | supported (this phase) | PBGEN-03; no input/output wrapper |
| augment | 7.17 | (resolved by compiler) | out-of-scope | Augment application happens at compiler level |
| identity | 7.18 | string field (identityref base) | unsupported | Identityref leaf emits `string`; no typed enum for identity hierarchy |
| extension | 7.19 | ignored | out-of-scope | Extension statements not represented in schema |
| feature / if-feature | 7.20.1, 7.20.2 | ignored | out-of-scope | Feature guards not evaluated; nodes always emitted |
| deviation | 7.20.3 | (pre-pass, not in Go generator yet) | out-of-scope | Deviation pre-pass not applied in proto generator (v1 scope) |
| config | 7.21.1 | ignored | out-of-scope | Config/state split not emitted in proto |
| status | 7.21.2 | SkipDeprecated/SkipObsolete options | supported | shouldSkip() filters deprecated/obsolete |
| description | 7.21.3 | // comment (AddAnnotations option) | supported | AddAnnotations=true emits description as comment |
| reference | 7.21.4 | ignored | out-of-scope | Reference statement not emitted |
| when | 7.21.5 | ignored | out-of-scope | XPath conditions not evaluatable in proto |

---

## Open Questions

1. **Notification in service block**
   - What we know: PBGEN-03 spec says "rpc and action statements emit service block definitions" — notification is listed separately in the requirement description as "typed structs"
   - What's unclear: The PBGEN-03 requirement text says "rpc and action statements emit Protobuf service block definitions" — notification is NOT listed in PBGEN-03. The standalone message approach is consistent with how notification works (it's a push, not a request/response pair).
   - Recommendation: Emit Notification as a standalone proto message (not a service rpc method). This matches RFC 7950 semantics and the Go generator pattern.

2. **CEL path validation scope — what constitutes a "path"**
   - What we know: `buildValidateOptions` in validations.go produces annotation strings for length/range/pattern constraints; these are CEL expressions, not field path references
   - What's unclear: The PBGEN-04 requirement says "CEL annotation paths are validated against the compiled schema" — the current CEL annotations use expressions like `this.size()`, `this.matches()`, `this >= 1`, which reference `this` (current field), not dotted schema paths
   - Recommendation: Interpret "CEL path validation" as validating that each leaf field that receives a `(buf.validate.field)` annotation is itself a real schema node (not a phantom field). The CONTEXT.md confirms: "validate that each leaf field that receives a CEL annotation has a corresponding node resolvable in the compiled schema". Implementation: during generation, track `{protoFieldName: yangNodeName}` for annotated fields; post-generation, verify each yangNodeName exists in module.Nodes (or nested children).

---

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) + testify v1.11.1 |
| Config file | none — standard `go test` |
| Quick run command | `go test ./generator/protobuf/... -run TestProto -v` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| PBGEN-01 | `docs/proto-coverage.md` exists and contains all 22 statement rows | smoke | `go test ./generator/protobuf/... -run TestCoverageDocExists` | ❌ Wave 0 |
| PBGEN-02 | anydata/anyxml nodes emit `google.protobuf.Any` field; no silent drop | unit | `go test ./generator/protobuf/... -run TestAnyData` | ❌ Wave 0 |
| PBGEN-02 | conditional `google/protobuf/any.proto` import added when anydata present | unit | `go test ./generator/protobuf/... -run TestAnyDataImport` | ❌ Wave 0 |
| PBGEN-03 | rpc statement emits service block + Input/Output messages | unit | `go test ./generator/protobuf/... -run TestRPCService` | ❌ Wave 0 |
| PBGEN-03 | action statement emits service rpc method in module service block | unit | `go test ./generator/protobuf/... -run TestActionService` | ❌ Wave 0 |
| PBGEN-03 | notification emits standalone message (not in service) | unit | `go test ./generator/protobuf/... -run TestNotificationMessage` | ❌ Wave 0 |
| PBGEN-04 | invalid CEL annotation path returns error, no file written | unit | `go test ./generator/protobuf/... -run TestCELPathValidation` | ❌ Wave 0 |
| PBGEN-04 | all invalid paths collected before returning (not fail-fast) | unit | `go test ./generator/protobuf/... -run TestCELPathValidationMultiple` | ❌ Wave 0 |
| PBGEN-02,03 | golden files match GenerateDevice output for corpus | regression | `go test ./test/... -run TestGoldenProto` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./generator/protobuf/... -run TestProto`
- **Per wave merge:** `go test ./...`
- **Phase gate:** `go test ./...` green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `generator/protobuf/generator_test.go` — add stub tests for PBGEN-02 (TestAnyDataField, TestAnyXMLField, TestAnyDataConditionalImport) — these are new test functions in the existing file
- [ ] `generator/protobuf/generator_test.go` — add stub tests for PBGEN-03 (TestRPCServiceBlock, TestActionServiceBlock, TestNotificationMessage)
- [ ] `generator/protobuf/validations_test.go` — add stub tests for PBGEN-04 (TestCELPathValidationValid, TestCELPathValidationInvalid, TestCELPathValidationAllErrors)
- [ ] `test/proto_golden_test.go` — new file with TestGoldenProto
- [ ] `testdata/proto/` — new directory (created by `-update` run after implementation)
- [ ] `docs/proto-coverage.md` — new file (PBGEN-01)
- [ ] Framework install: none needed — stdlib + testify already in go.mod

---

## Sources

### Primary (HIGH confidence)
- `/home/user/repos/gotya/generator/protobuf/generator.go` — full generator source, confirmed missing cases
- `/home/user/repos/gotya/generator/protobuf/validations.go` — CEL constraint builder, confirmed scope
- `/home/user/repos/gotya/schema/schema.go` — all schema node types confirmed: AnyData, AnyXML, RPC, Action, Notification, Input, Output
- `/home/user/repos/gotya/compiler/compiler.go` lines 453–497 — confirmed compiler guarantees input/output children always present
- `/home/user/repos/gotya/generator/golang/generator.go` — RPC/Action/Notification/AnyData patterns confirmed (lines 304–461)
- `/home/user/repos/gotya/test/corpus_test.go` — golden file test base pattern confirmed
- `/home/user/repos/gotya/docs/rfc7950.txt` — RFC 7950 statement sections 7.1–7.21 enumerated directly from source

### Secondary (MEDIUM confidence)
- `generator/protobuf/generator_test.go`, `device_test.go`, `validations_test.go` — test style and conventions confirmed
- `go.mod` — dependency inventory confirmed (only testify; no protoc-related Go deps)
- `protoc --version` output — protoc 3.21.12 available at `/usr/bin/protoc`

### Tertiary (LOW confidence)
- None

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all code inspected directly, no external dependencies needed
- Architecture: HIGH — patterns derived from existing Go generator and existing proto generator; no speculation
- Pitfalls: HIGH — derived from direct code inspection of the existing switch statements and compiler guarantees
- RFC 7950 coverage matrix: HIGH — statement list extracted directly from `docs/rfc7950.txt` which is present in the repo

**Research date:** 2026-03-15
**Valid until:** 2026-04-15 (stable domain — no external library changes affect this)