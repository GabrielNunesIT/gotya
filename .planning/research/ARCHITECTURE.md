# Architecture Research

**Domain:** YANG compiler / code generation library in Go
**Researched:** 2026-03-14
**Confidence:** HIGH (current codebase examined directly; patterns verified against goyang/ygot open-source references)

## Standard Architecture

### System Overview

```
┌──────────────────────────────────────────────────────────────────────┐
│                         Input Layer                                   │
│   ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────┐  │
│   │  YANG string │  │  YANG file   │  │  Directory of .yang files│  │
│   └──────┬───────┘  └──────┬───────┘  └────────────┬─────────────┘  │
└──────────┼─────────────────┼───────────────────────┼────────────────┘
           │                 │                        │
┌──────────▼─────────────────▼────────────────────── ▼────────────────┐
│                         Lexer/Parser Layer                            │
│   ┌──────────────────────────────────────────────────────────────┐  │
│   │  Lexer (token stream) → Parser (recursive descent) → AST    │  │
│   │  Output: *ast.Module  |  Errors: []string (parse errors)    │  │
│   └──────────────────────────────────────┬───────────────────────┘  │
└─────────────────────────────────────────┼──────────────────────────┘
                                          │
┌─────────────────────────────────────────▼──────────────────────────┐
│                       Compiler Layer                                  │
│   ┌──────────────────────────────────────────────────────────────┐  │
│   │  Pass 1: Module metadata  (namespace, prefix, features)      │  │
│   │  Pass 2: Typedefs + identities (transitive resolution)       │  │
│   │  Pass 3: Groupings expansion (uses/refine, depth-guarded)    │  │
│   │  Pass 4: Data node tree build (container/list/leaf/...)      │  │
│   │  Pass 5: Augment application (convergence-guarded loop)      │  │
│   │  Pass 6: Deviation application                               │  │
│   │  Pass 7: Feature pruning (if-feature evaluation)            │  │
│   │  Pass 8: Cross-cutting validation (keys, types, config)      │  │
│   │  Output: *schema.Module | Errors: bounded []string           │  │
│   └──────────────────────────────────────┬───────────────────────┘  │
│                  ModuleLoader interface ──┘  (resolves imports)       │
└─────────────────────────────────────────┼──────────────────────────┘
                                          │
┌─────────────────────────────────────────▼──────────────────────────┐
│                        Schema Layer                                   │
│   Node interface, BaseNode, Container, List, Leaf, LeafList,         │
│   Choice, Case, RPC, Action, Notification, AnyXML, AnyData           │
│   (read-only tree; shared between compiler output and generators)     │
└─────────────────────────┬──────────────────────────────────────────┘
                          │
           ┌──────────────┴──────────────┐
           │                             │
┌──────────▼─────────┐       ┌───────────▼─────────┐
│  Go Generator       │       │  Protobuf Generator  │
│  generator/golang   │       │  generator/protobuf  │
│  → io.Writer        │       │  → io.Writer         │
└─────────────────────┘       └─────────────────────┘
           │
           ▼
┌──────────────────────┐
│  codec/rfc7951       │
│  (uses generated     │
│  types at runtime)   │
└──────────────────────┘
```

### Component Responsibilities

| Component | Responsibility | Boundary Rule |
|-----------|----------------|---------------|
| `token` | Token type constants, Position | No dependencies; foundational |
| `ast` | AST node types (Module, Statement, BaseNode) | No dependencies; pure data |
| `parser/lexer` | Tokenize YANG source | Depends on `token` only |
| `parser` | Build AST from token stream | Depends on `lexer`, `token`, `ast` |
| `schema` | Semantic node types for compiled output | Depends on `ast` (stores raw groupings/typedefs) |
| `compiler` | Transform AST → validated schema tree | Depends on `ast`, `schema`; uses `ModuleLoader` interface |
| `generator` | Generator interface definition | Depends on `schema` only |
| `generator/golang` | Emit Go source from schema | Depends on `schema`, `generator` |
| `generator/protobuf` | Emit .proto from schema | Depends on `schema`, `generator` |
| `codec/rfc7951` | RFC 7951 JSON encode/decode | Depends on reflection + schema knowledge |
| `gotya.go` | Public API facade | Depends on `parser`, `compiler` |
| `cmd/gotya` | CLI tool + DirectoryLoader | Depends on public API + generators |

**Invariant:** Data flows strictly downward — generators never call back into the compiler, and the compiler never calls generators.

## Recommended Project Structure

```
gotya/
├── token/              # Token types (no deps — foundational)
├── ast/                # AST node types (no deps — pure data)
├── parser/
│   ├── lexer/          # Tokenizer
│   └── parser.go       # Recursive descent parser
├── schema/             # Compiled semantic tree (no compiler dep)
├── compiler/
│   ├── compiler.go     # Orchestration, multi-pass logic
│   └── validator.go    # Cross-cutting RFC 7950 validation rules
├── generator/
│   ├── generator.go    # Generator interface
│   ├── golang/         # Go struct emitter
│   │   ├── generator.go
│   │   └── device.go
│   └── protobuf/       # Proto file emitter
│       ├── generator.go
│       ├── device.go
│       └── validations.go
├── codec/
│   └── rfc7951/        # Runtime JSON codec
├── test/
│   ├── testdata/       # Golden files (.golden extension)
│   │   ├── go/         # Expected Go generator output per fixture
│   │   └── proto/      # Expected Proto generator output per fixture
│   ├── assets/yangs/   # Real-world YANG fixture files
│   └── generate.go     # go:generate entrypoint (build tag ignore)
├── gotya.go            # Public API
└── cmd/gotya/          # CLI
    ├── main.go
    └── loader.go
```

### Structure Rationale

- **`test/testdata/`:** Go convention for test fixtures; ignored by `go build` and `go vet`. Golden files live here, separated by generator type, one file per YANG construct test case. This is the standard pattern used by goyang, ygot, and the Go standard library itself.
- **`compiler/validator.go` separate from `compiler.go`:** Keeps orchestration logic (pass ordering, state threading) distinct from the RFC 7950 rule implementations. Easier to add new validation rules without touching the pass orchestrator.
- **`schema/` depends on `ast/` but not on `compiler/`:** Generators only import `schema`; they must never import `compiler`. This boundary prevents accidentally coupling generation logic to compilation state.

## Architectural Patterns

### Pattern 1: Multi-Pass Compilation with Convergence Guard

**What:** The compiler runs semantically distinct passes in a fixed order. Augment resolution is the one pass that requires an iterative loop (some augment targets may not yet exist when the augment is first evaluated, because the target is produced by another augment applied earlier). A convergence guard — a max-iteration cap and a "did anything change?" check per iteration — prevents infinite loops.

**When to use:** Any time a resolution step has forward-reference dependencies that cannot be resolved in a single linear walk (augments, some cross-module typedef chains).

**Trade-offs:** Multi-pass is easier to reason about and test than a single pass with deferred callbacks, but requires careful state management between passes. The convergence guard adds a small overhead but is the only safe way to handle augment chains.

**Correct shape for augment loop:**

```go
const maxAugmentPasses = 10 // RFC 7950 does not bound depth, but 10 covers all real YANG models

func (c *Compiler) applyAugments(mod *schema.Module, augments []pendingAugment) error {
    for pass := 0; pass < maxAugmentPasses; pass++ {
        applied := 0
        remaining := augments[:0]
        for _, a := range augments {
            if err := c.tryApplyAugment(mod, a); err == nil {
                applied++
            } else {
                remaining = append(remaining, a)
            }
        }
        augments = remaining
        if len(augments) == 0 {
            return nil
        }
        if applied == 0 {
            // No progress — remaining augments target non-existent paths
            return fmt.Errorf("augments could not be resolved: %v", remaining)
        }
    }
    return fmt.Errorf("augment resolution did not converge after %d passes", maxAugmentPasses)
}
```

### Pattern 2: Golden-File Testing for Generators

**What:** Generator tests run the full pipeline (parse → compile → generate) on a curated YANG input, capture the output as a string, and compare it byte-for-byte against a committed reference file in `testdata/`. An `-update` flag regenerates the reference files when generation behavior changes intentionally.

**When to use:** Every generator test. String assertions (`assert.Contains`) are acceptable for targeted feature spot-checks, but the full output for any non-trivial YANG module must be in a golden file to detect regressions.

**Trade-offs:** Golden files are verbose but catch silent regressions that `Contains` checks miss. The `-update` workflow keeps maintenance tractable — run `go test ./... -update` after an intentional change.

**Implementation shape:**

```go
var updateGolden = flag.Bool("update", false, "update golden files")

func runGoldenTest(t *testing.T, yangSource string, goldenPath string, gen generator.Generator) {
    t.Helper()
    // parse + compile omitted for brevity
    var buf bytes.Buffer
    if err := gen.Generate(schemaMod, &buf); err != nil {
        t.Fatalf("generate: %v", err)
    }
    got := buf.Bytes()
    if *updateGolden {
        if err := os.WriteFile(goldenPath, got, 0644); err != nil {
            t.Fatalf("writing golden file: %v", err)
        }
        return
    }
    want, err := os.ReadFile(goldenPath)
    if err != nil {
        t.Fatalf("reading golden file %s: %v", goldenPath, err)
    }
    if !bytes.Equal(got, want) {
        t.Errorf("generator output mismatch for %s\ndiff:\n%s", goldenPath, diffStrings(string(want), string(got)))
    }
}
```

### Pattern 3: Table-Driven Error Tests for the Compiler

**What:** Compiler correctness tests use a `[]struct{ name, input string; wantErr bool; errContains string }` table. Each row is one YANG fragment that exercises a single rule. Tests run in parallel with `t.Run`.

**When to use:** Every RFC 7950 validation rule should have at least one "valid input, no error" row and one "invalid input, specific error message" row in a table.

**Trade-offs:** Tables are more compact than separate test functions and make coverage gaps visible at a glance. The cost is that each row must be self-contained YANG — no shared state between rows.

**Why this matters here:** The existing compiler tests already use this pattern partially (see `TestCompiler_IdentityrefValidation`). The pattern should be applied uniformly to cover augment chaining and circular import detection, which are currently untested.

### Pattern 4: Bounded Error Accumulation

**What:** The compiler accumulates errors across a pass (rather than returning on the first error) so callers see all problems in a single compile run. But accumulation is bounded: after N errors, the compiler records a truncation notice and stops collecting new errors. The Go compiler uses N=10 by default; a YANG compiler can reasonably use N=20.

**When to use:** All compiler passes. The current implementation accumulates without bound, which is a correctness risk (malformed YANG with thousands of errors can exhaust memory).

**Implementation shape:**

```go
const maxErrors = 20

type errorCollector struct {
    errs      []string
    truncated bool
}

func (e *errorCollector) add(msg string) {
    if e.truncated {
        return
    }
    if len(e.errs) >= maxErrors {
        e.errs = append(e.errs, fmt.Sprintf("... (too many errors, first %d shown)", maxErrors))
        e.truncated = true
        return
    }
    e.errs = append(e.errs, msg)
}
```

### Pattern 5: Schema Auditing via Real-World Fixtures

**What:** To find generator coverage gaps, compile and generate output from a curated set of real-world YANG modules (the `test/assets/yangs/` corpus already contains ~170 BBF/IETF/IEEE modules). Audit the output for: missing node types, incorrect field tags, malformed identifiers, and files that fail `go build` or `protoc`.

**When to use:** Once as a structured audit exercise (see Generation Audit section), then as a regression suite by promoting representative audit fixtures into golden file tests.

**The audit signal hierarchy:**
1. Panic during generation → critical bug
2. Generated file fails `go build` / `protoc` → blocking bug
3. Generated file compiles but a YANG feature is silently dropped → coverage gap
4. Generated file is technically correct but incorrect idiomatically → quality issue

## Data Flow

### Compilation Pipeline (canonical)

```
YANG source string
    │
    ▼
lexer.New(src) → token stream
    │
    ▼
parser.New(lex).ParseModule() → *ast.Module  [nil on parse failure]
    │                            parser.Errors() []string
    ▼
compiler.New(opts).Compile(astMod) → *schema.Module, error
    │  [opts.Loader resolves imports/includes on demand]
    │  [opts.SupportedFeatures drives if-feature pruning]
    ▼
schema.Module  (immutable after Compile returns)
    │
    ├─► golang.GoGenerator.Generate(mod, w) → io.Writer
    │
    └─► protobuf.ProtobufGenerator.Generate(mod, w) → io.Writer
```

### Module Loader Resolution (on-demand)

```
compiler encounters "import foo"
    │
    ▼
opts.Loader.LoadAST("foo") → *ast.Module (parsed, cached)
    │
    ▼
opts.Loader.Load("foo") → *schema.Module (compiled, cached)
    │
    ▼
compiler uses schema.Module to resolve cross-module references
```

**Key invariant:** The loader is the only place that triggers recursive compilation. The compiler itself never calls Compile recursively — it requests already-compiled schema modules through the loader interface.

### Error Flow

```
Parser errors    → parser.Errors() []string       (caller checks after ParseModule)
Compiler errors  → error return from Compile()    (wrapped multi-error string)
Generator errors → error return from Generate()   (wrapped single error)
```

There is no error type — errors are strings throughout the current implementation. This is acceptable for v1; richer error types (with source position, rule code) would be a v2 concern.

## Fix and Improvement Order

This order minimizes the risk of regressions by fixing the foundation before building on it.

**Order A — Compiler correctness first, then testing, then generators:**

| Order | Item | Why First |
|-------|------|-----------|
| 1 | Nil-pointer panics in augment/refine path resolution | Panics are silent data loss; they break the audit corpus too |
| 2 | Augment resolution loop: add max-iteration guard | Required before any augment-heavy module can be audited |
| 3 | Circular typedef depth limit | Stack overflow risk in production; blocks reliable testing |
| 4 | Bounded error accumulation | Must be done before adding many new error paths |
| 5 | Silent `AddChild()` errors in RPC/Action/Case | Silent failures produce incorrect schemas |
| 6 | Remove debug `fmt.Printf` | Embarrassing in a library; do it before any external review |
| 7 | Module loader error reporting | Needed for audit script to produce clean output |
| 8 | Add missing compiler tests (augment chain, circular import, AddChild) | Tests for the bugs just fixed; lock in correctness |
| 9 | Go generator audit against real YANG corpus | Now safe to run without panics |
| 10 | Fix Go generator coverage gaps from audit | Evidence-driven; don't guess |
| 11 | Protobuf generator audit | Same approach |
| 12 | Fix Protobuf generator coverage gaps | Evidence-driven |

**Rationale for this order:** Panics in the compiler corrupt audit results — you cannot trust generator output if the schema tree was built from a panic-recovered partial state. The error accumulation bound must be added before the new test cases, because the new tests check specific error messages and an unbounded accumulator can swamp a message you're looking for. Generators come last because they depend on a correct schema tree.

## Generation Audit Structure

The `test/assets/yangs/` directory contains ~170 real-world BBF, IETF, and IEEE YANG modules — a ready-made audit corpus. The existing `test/generate.go` already drives generation from this corpus but discards coverage information. A structured audit adds:

**Phase 1 — Crash audit (run existing generate.go, fix panics)**

```bash
cd test && go run generate.go 2>&1 | grep -E "panic|nil pointer|goroutine"
```

Any panic is a P0 bug. Fix all before proceeding.

**Phase 2 — Compile audit (verify generated files build)**

```bash
# For Go
go build ./test/out/device.go
# For Proto
protoc --proto_path=test/out test/out/device.proto
```

Compilation failures are P1 bugs. Capture the error, trace to the schema node type, fix the generator.

**Phase 3 — Coverage gap audit (enumerate YANG features, check presence in output)**

Build a checklist from RFC 7950 §7 (statement coverage). For each statement type, check whether the generator produces correct output:

| YANG Feature | Expected Go output | Expected Proto output | Status |
|--------------|--------------------|-----------------------|--------|
| `leaf` basic types | typed pointer field | proto scalar | ? |
| `leaf-list` | `[]*T` slice field | repeated field | ? |
| `container` | struct type | message type | ? |
| `list` with key | `[]*T` + ordered-map methods | repeated message | ? |
| `choice`/`case` | flattened fields on parent struct | oneof | ? |
| `typedef` resolution | resolved Go type | resolved proto type | ? |
| `grouping`/`uses` | expanded inline | expanded inline | ? |
| `augment` applied | fields appear on target struct | fields appear on target | ? |
| `rpc` | (skipped or method?) | service method | ? |
| `notification` | struct type | message type | ? |
| `action` | (skipped or method?) | (skipped?) | ? |
| `anyxml`/`anydata` | `interface{}` or `[]byte` field | bytes field | ? |
| `enumeration` | const block + typed enum | enum message | ? |
| `union` | interface or oneof struct | oneof | ? |
| `identityref` | string or typed const | string | ? |
| `leafref` | resolved target type | resolved target type | ? |
| `bits` | uint64 or bitfield struct | uint64 | ? |
| `decimal64` | string or float64 | string | ? |
| Annotations (`json`, `xml`, `yang` tags) | all three present | N/A | ? |
| `description` as Go doc comment | `// description text` | comment | ? |

Fill the Status column (OK / MISSING / WRONG) during the audit. MISSING and WRONG items become work items ordered by user-facing impact.

**Phase 4 — Promote fixtures to golden files**

For each YANG module that exercised a coverage gap and was fixed, add a golden file test in `test/testdata/`. This prevents silent regression.

## Anti-Patterns

### Anti-Pattern 1: String Assertion Coverage Masking

**What people do:** Use `assert.Contains(t, output, "type Foo struct {")` to verify generator output. The current generator tests do this extensively.

**Why it's wrong:** `Contains` checks only verify that a token is somewhere in the output. They do not catch: wrong field order, missing struct tags, incorrect types for adjacent fields, extra fields that should not exist. A generator with 10 Contains checks can silently omit half of the expected output.

**Do this instead:** Add golden file coverage for all non-trivial YANG inputs. Keep `Contains` checks only for targeted feature-flag variations (e.g., "with ordered maps enabled, expect IsOrdered method").

### Anti-Pattern 2: Augment Resolution Without a Convergence Guard

**What people do:** Loop over pending augments until the list is empty, without tracking whether any progress was made per iteration.

**Why it's wrong:** If two augments target each other or a target that never appears, the loop runs forever. This is a live bug in the current codebase.

**Do this instead:** Track `applied` count per iteration. If `applied == 0` and `len(remaining) > 0`, the loop cannot make progress — return an error immediately instead of spinning.

### Anti-Pattern 3: Generating Directly into a File

**What people do:** Open a file in the generator and write directly to it.

**Why it's wrong:** Makes the generator untestable without filesystem setup. All current generators correctly accept `io.Writer` — this anti-pattern must not be introduced.

**Do this instead:** Accept `io.Writer`. The CLI handles file creation. Tests pass `bytes.Buffer`.

### Anti-Pattern 4: Silent AddChild Failures

**What people do:** Ignore the error return from `AddChild()` in RPC, Action, and Case node handling.

**Why it's wrong:** Duplicate identifier collisions are silently swallowed, producing a schema with fewer nodes than the YANG model defines. Downstream generators produce incorrect code.

**Do this instead:** Propagate `AddChild()` errors into the error accumulator. Tests for this pattern: compile a YANG module with a duplicate identifier inside an RPC input and assert the error is reported.

### Anti-Pattern 5: Inline YANG in Production Generator Tests

**What people do:** Embed multi-hundred-line YANG strings in generator test functions.

**Why it's wrong:** Inline YANG obscures test intent, is hard to diff when it changes, and cannot be reused across tests. The YANG syntax is also validated implicitly, making it unclear whether a test is testing the generator or the parser.

**Do this instead:** For anything beyond a 20-line spot-check, put the YANG in `testdata/` as a `.yang` file and load it with `os.ReadFile`. This also lets you validate the fixture independently with pyang.

## Integration Points

### Internal Boundaries

| Boundary | Communication | Invariant |
|----------|---------------|-----------|
| parser → compiler | `*ast.Module` value | Compiler must not modify AST nodes |
| compiler → generators | `*schema.Module` value | Generators must not modify schema nodes |
| compiler → loader | `ModuleLoader` interface | Loader is injected; compiler has no knowledge of filesystem |
| generators → io.Writer | byte stream | Generator never manages file lifecycle |

### RFC 7950 Compliance Reference Points

The authoritative source for what the compiler must accept and reject is RFC 7950. Key sections affecting implementation:

- §7.1–7.3: module/submodule/imports — affects loader resolution
- §7.5: augment — affects convergence-guarded loop
- §7.8: grouping/uses/refine — affects depth limit for circular detection
- §7.12: deviation — affects deviation pass ordering (must run after augments)
- §7.13: if-feature — affects pruning pass ordering (must run after deviations)
- §9: type system — affects typedef chain resolution and type restriction validation

## Sources

- goyang pkg/yang package docs: https://pkg.go.dev/github.com/openconfig/goyang/pkg/yang (MEDIUM confidence — pkg.go.dev summary; source examined indirectly)
- ygot architecture: https://github.com/openconfig/ygot/blob/master/docs/design.md (attempted fetch; 429, synthesized from README and genstate_test.go search results — MEDIUM confidence)
- ygot golden file pattern: https://github.com/openconfig/ygot/blob/master/ygen/genstate_test.go (confirmed existence of `updateGolden` flag pattern — HIGH confidence)
- Go golden file testing: https://ieftimov.com/posts/testing-in-go-golden-files/ (MEDIUM confidence)
- Go compiler error limit (10 errors default): https://github.com/golang/go/issues/5142 (HIGH confidence — official Go issue tracker)
- RFC 7950 YANG 1.1: https://www.rfc-editor.org/rfc/rfc7950.txt (HIGH confidence — authoritative)
- Current codebase (compiler_test.go, generator/golang/generator_test.go, test/generate.go, test/assets/yangs/): directly read (HIGH confidence)

---
*Architecture research for: YANG Go codegen library (gotya)*
*Researched: 2026-03-14*
