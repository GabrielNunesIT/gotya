# Project Research Summary

**Project:** gotya — YANG parsing and Go code generation library
**Domain:** YANG-to-Go/Proto codegen library (Go)
**Researched:** 2026-03-14
**Confidence:** HIGH

## Executive Summary

gotya is a YANG 1.1 parser, compiler, and code generator implemented entirely in Go. It takes `.yang` source files through a multi-pass compilation pipeline — lexer, parser, compiler (AST to semantic schema), and then one or more generators that emit either Go structs or Protobuf messages. The core architecture is sound and largely already implemented: the data flow is strictly downward (parser feeds compiler feeds generator), the schema layer is immutable at generator time, and the public API is a clean facade over the pipeline. The recommended approach is to harden existing correctness first, then complete the feature surface that real-world YANG models require, and finally stabilize the public API for a v1 release.

The most important insight from the combined research is that gotya is closer to v1 than it may appear, but has a set of critical bugs that would produce crashes or silent data loss in production: nil-pointer panics on malformed augment paths, a stack overflow on circular typedefs, an infinite loop on circular imports, and an unbounded augment resolution loop that silently succeeds without applying all augments. These must be fixed before any generator audit or feature work, because they corrupt the schema tree that generators depend on. Once the foundation is clean, the remaining work is well-defined: a structured codegen audit against the real-world YANG corpus already in `test/assets/yangs/`, followed by filling confirmed gaps (identityref typed constants, RPC/Action/Notification structs, deviation application, AnyData/AnyXML fields), and finally a focused API stabilization pass before tagging v1.

Key risks are scoped and manageable. The YANG feature surface is large but the checklist for what is and is not implemented is now explicit (see FEATURES.md). The ecosystem peers — goyang and ygot — confirm that augment chaining, typedef chains, and identityref resolution are the hardest problems in this domain; gotya has encountered all three and they are all tractable fixes. The "no new dependencies" constraint from PROJECT.md is not a limitation here: golden file testing, `go/format.Source()` validation, and corpus smoke tests are all stdlib patterns that add significant quality assurance at zero dependency cost.

---

## Key Findings

### Recommended Stack

gotya's stack is intentionally minimal and should stay that way. The existing tools — Go 1.25.5, stretchr/testify v1.11.1, mockery v3.x, and golangci-lint v2 — are all current and appropriate. No new runtime dependencies are warranted. The one recommended addition is using `go/format.Source()` (stdlib, already available) in the Go generator's output path to catch syntax errors in generated code at generation time rather than at user compile time. Golden file testing should be implemented with 20 lines of stdlib (`os.ReadFile`, `bytes.Equal`, `-update` flag) rather than importing goldie or any other library.

**Core technologies:**
- Go 1.25.5: only language; no framework, no database, no service clients
- stretchr/testify v1.11.1: test assertions — already in use, correct version
- golangci-lint v2: multi-linter CI enforcement — already configured; run `migrate` if `.golangci.yml` was written for v1
- go/format (stdlib): add to Go generator output path — validates generated code syntax, ensures gofmt-canonical output, zero cost
- text/template (stdlib): generator templating — already in use, no reason to switch to jennifer or any code-gen library

### Expected Features

Research confirmed gotya is competitive with ygot on data-plane constructs. The gaps are concentrated in control-plane constructs and schema-modifying constructs.

**Must have (table stakes — implemented):**
- Container/List/Leaf/LeafList/Choice/Case/Union/Bits/Enum Go struct generation
- RFC 7951 JSON codec for generated structs
- Grouping/uses/refine expansion, augment resolution
- Config/State split, fakeroot/device aggregation
- CEL/buf-validate annotations in Protobuf generator
- Getters, setters, validation methods, default population

**Must have (table stakes — NOT YET IMPLEMENTED, v1 blockers):**
- Identityref as typed Go const block — currently silently falls through to `*string`; the single most glaring gap versus ygot; every real-world OpenConfig module uses identityref
- AnyData/AnyXML field emission — currently produces no output, silently drops nodes; any model using anydata produces wrong Go structs
- Deviation application pre-pass — standard vendor-specific adjustment mechanism; deviations are parsed and stored but never applied to generated output
- RPC/Action/Notification Go struct generation — parsed and stored but neither generator emits code; blocked on AddChild() bug fix
- Presence-container distinction — incorrect nil semantics; easy fix, high correctness impact

**Should have (differentiators, v1.x):**
- Leafref validation in Validate() methods
- RPC/Action/Notification Proto service blocks
- if-feature full parenthesised expression parser
- ordered-by user preservation for leaf-list

**Defer (v2+):**
- Runtime XPath / must / when evaluation — explicitly deferred; requires a full XPath engine
- JSON Schema / OpenAPI generator — separate output format via pluggable interface
- Cross-module type deduplication, streaming/lazy compilation for 100k+ node schemas

### Architecture Approach

The architecture is a classic multi-pass compilation pipeline with strict downward data flow. The five major stages (lexer/parser, compiler, schema layer, generators, codec) have clean boundaries and the key invariant — generators never call back into the compiler — is correctly established. The compiler's multi-pass design with a convergence-guarded augment resolution loop is the correct pattern for this problem domain. The primary structural improvement needed is upgrading the test strategy from substring assertions to golden file tests and adding a corpus smoke test that runs all ~170 YANG assets in `test/assets/yangs/` through the full pipeline.

**Major components:**
1. `token` / `ast` / `parser` — foundational; pure data, no deps; do not modify
2. `compiler` — multi-pass AST-to-schema transform; the source of most bugs; fix here first
3. `schema` — immutable semantic tree shared between compiler output and generators
4. `generator/golang` + `generator/protobuf` — emit source code from schema; accept `io.Writer`
5. `codec/rfc7951` — runtime JSON encode/decode; already implemented and tested
6. `gotya.go` / `cmd/gotya` — public API facade and CLI; stabilize before v1 tag

### Critical Pitfalls

1. **Nil pointer panic on malformed augment/refine paths** — `findNode()` returns nil and callers at compiler.go:774 and compiler.go:992 dereference without nil checks; add nil guards and propagate errors; remove debug `fmt.Printf` placeholders at the same sites; this is P0 before any audit work

2. **Circular typedef causes stack overflow (not panic — process termination)** — `getType()` recurses without a depth limit or visited set; `typedef A { type B; } typedef B { type A; }` crashes the process; add `visited map[string]struct{}` parameter; this cannot be recovered from in production

3. **Augment chaining silently produces incomplete schemas** — the convergence loop has no max-iteration cap and no assertion that all augments resolved; augment B targeting a node added by augment A may silently be skipped; add a cap and a post-loop assertion that `pendingAugments` is empty; confirmed as a known failure mode in openconfig/goyang issue #265

4. **Circular import causes infinite recursion** — the module loader caches but does not distinguish "loading" from "loaded"; two mutually-importing modules crash; add an in-progress set; this is documented in CONCERNS.md

5. **Public API frozen with placeholder errors** — `Parse()` returns `os.ErrInvalid` (a sentinel from the standard library, not a domain error); parser errors are discarded; the comment `// Simplistic error for facade demonstration` is in `gotya.go`; changing post-v1 requires a major version bump; fix before tagging v1

---

## Implications for Roadmap

Based on combined research, the correct sequencing is: fix foundation bugs first (so the audit corpus runs cleanly), then add testing infrastructure (so gaps discovered during audit are captured as tests), then audit and fill generator gaps, then stabilize the public API.

### Phase 1: Compiler Correctness Hardening

**Rationale:** Every subsequent phase depends on a compiler that does not panic, does not infinitely loop, and produces correct schema trees. Running the codegen audit against a panicking compiler produces misleading results. This is the bottleneck for all downstream work.

**Delivers:** A compiler that handles malformed and pathological YANG inputs gracefully; no crashes or infinite loops; clean stdout/stderr from library code.

**Addresses:**
- Remove debug `fmt.Printf` statements (compiler.go:778, compiler.go:930)
- Nil-check guards on all `findNode()` call sites
- Max-iteration cap and post-loop assertion for augment resolution
- Visited-set guard in `getType()` for circular typedef detection
- In-progress set in module loader for circular import detection
- Bounded error accumulation with configurable `MaxErrors` in `compiler.Options`
- Propagate all `AddChild()` errors (remove `_ =` call sites)

**Avoids:** Pitfalls 1, 2, 3, 4, 5, 6, 7 from PITFALLS.md; this phase eliminates all crash-class and silent-corruption-class bugs.

**Research flag:** Standard patterns — no additional research needed; all fixes are mechanical and well-specified.

---

### Phase 2: Testing Infrastructure

**Rationale:** The architecture research establishes that the current substring-assertion test strategy misses entire classes of regressions. Adding golden file tests and a corpus smoke test before the audit ensures that every gap found during audit is captured as a failing test (not a document entry), and that every fix is locked in by a passing test.

**Delivers:**
- Golden file test harness for both Go and Protobuf generators (`testdata/go/` and `testdata/proto/` directories, `-update` flag)
- `go/format.Source()` validation in the Go generator's output path
- Corpus smoke test in `test/integration_test.go` that iterates all `test/assets/yangs/` files, runs parse+compile+generate, and asserts no panic and syntactically valid Go output
- Error sentinel values replacing string-matched errors in compiler tests

**Avoids:** Pitfall 8 (codegen audit scope creep — incomplete audit shipped as complete) by establishing the "audit done when every statement type has a passing test" criterion before the audit starts.

**Research flag:** Standard patterns — golden file testing and `go/format.Source()` are fully documented stdlib patterns; no additional research needed.

---

### Phase 3: Go Generator Audit and Gap Filling

**Rationale:** With a panic-free compiler and a golden file test harness in place, the real-world YANG corpus can be run through the Go generator systematically. The audit produces a signed-off RFC 7950 statement coverage matrix. Confirmed gaps become work items ordered by user-facing impact.

**Delivers:**
- Complete RFC 7950 statement coverage matrix for the Go generator
- Identityref → typed Go const block (P1 — highest-value unimplemented feature)
- AnyData/AnyXML field emission in Go generator (P1 — silent node drops)
- Presence-container distinction in generated Go (P1 — incorrect nil semantics)
- Deviation application pre-pass (P1 — generated code does not match device reality)
- Go keyword collision handling in identifier sanitization (Pitfall 10)
- JSON struct tags using original YANG identifiers for RFC 7951 correctness

**Avoids:** Pitfall 8 (incomplete audit), Pitfall 10 (Go keyword collisions in generated code).

**Research flag:** May need phase research for deviation application ordering (must run after augment resolution) and identityref cross-module base resolution. All other gaps are mechanical fills.

---

### Phase 4: RPC/Action/Notification Go Generation

**Rationale:** Separated from Phase 3 because RPC generation is blocked on the AddChild() bug fix (Phase 1) and requires a separate design decision about whether RPCs become Go methods or standalone request/response structs. Doing this in isolation also allows Proto RPC service blocks to be designed consistently.

**Delivers:**
- Typed Go request/response structs for `rpc`, `action`, and `notification` nodes
- AddChild() error propagation verified (prerequisite from Phase 1)
- Input/Output child nodes correctly emitted as nested structs

**Research flag:** Needs phase research — the design choice (method vs. standalone struct) has API implications for v1; ygot's approach should be examined as a reference point.

---

### Phase 5: Protobuf Generator Audit and Gap Filling

**Rationale:** Follows the Go generator audit using the same methodology. Proto generator gaps are expected to overlap (AnyData, RPC service blocks) but the output format requires separate attention. Proto service block generation for RPC depends on Go RPC structs being settled first.

**Delivers:**
- Complete RFC 7950 statement coverage matrix for the Protobuf generator
- AnyData/AnyXML as `google.protobuf.Any` or `bytes` fields
- RPC/Action/Notification as Proto service blocks
- CEL annotation path validation (post-generation schema-path check)

**Research flag:** Standard patterns for the data-plane constructs; needs phase research for Proto service block design and CEL annotation path validation.

---

### Phase 6: Public API Stabilization

**Rationale:** Must be the final phase before v1 tag. API decisions made here are permanent under Go module semver semantics. The research identified concrete problems: `os.ErrInvalid` as parse error sentinel, parser errors discarded in `Parse()`, and placeholder comments in `gotya.go`. These are the kind of mistakes that require a v2 major version bump to fix post-release.

**Delivers:**
- `gotya.ParseError` structured type (or equivalent) replacing `os.ErrInvalid`
- Parser diagnostics (file, line, column, message) propagated through `Parse()`
- All `// Simplistic`, `// placeholder`, `// TODO`, and `// for demonstration` comments removed from `gotya.go`
- Audit of all exported symbols: no internal package types leaking through public API
- Documentation of the `ASTModule = ast.Module` alias decision

**Avoids:** Pitfall 9 (API frozen with placeholder errors forcing a v2 major version bump).

**Research flag:** Standard patterns — Go module API stability practices are fully documented; no additional research needed.

---

### Phase Ordering Rationale

- **Phase 1 before all others:** Compiler panics corrupt every result downstream. The audit corpus, the golden file tests, and the feature gap analysis are all unreliable until the compiler is crash-free.
- **Phase 2 before Phase 3:** The test harness must exist before the audit so gaps are captured as failing tests, not as documents. Without this ordering, the audit is declared done when the document looks complete rather than when the tests pass.
- **Phase 3 before Phase 4:** RPC generation depends on the base generator being correct for containers and leaves (RPC input/output are containers). Running the full Go generator audit first surfaces any remaining container/leaf issues that would otherwise be confused with RPC-specific bugs.
- **Phase 4 before Phase 5:** Proto RPC service blocks reference Go-level type decisions. Locking in the Go RPC struct design first prevents the Proto generator from diverging.
- **Phase 6 last:** Cannot be done until the feature surface is known (Phases 3–5). Stabilizing the API before knowing what RPC types or error types will look like is premature.

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 4 (RPC/Action/Notification Go):** Design choice between Go methods vs. standalone structs has v1 API implications; ygot's approach and proto-gen-go's RPC pattern should be examined as references before committing to a design.
- **Phase 5 (Protobuf generator audit):** Proto service block design and CEL annotation path validation are underspecified; needs research into buf/validate CEL expression scoping rules.

Phases with standard patterns (skip research-phase):
- **Phase 1 (Compiler hardening):** All fixes are specified down to the function and line number in PITFALLS.md and ARCHITECTURE.md; no research needed.
- **Phase 2 (Testing infrastructure):** Golden file testing and `go/format.Source()` are stdlib patterns with canonical implementations; no research needed.
- **Phase 3 (Go generator audit):** RFC 7950 Section 7 is the complete checklist; the audit methodology is specified in ARCHITECTURE.md; the only research trigger is if a gap is found that has no obvious implementation path.
- **Phase 6 (API stabilization):** Go module release workflow is fully documented; no research needed.

---

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All technologies verified against official sources; no new dependencies justified; single borderline decision (goldie) clearly resolved in favor of stdlib |
| Features | HIGH | Based on direct codebase audit cross-referenced against RFC 7950 and ygot; gaps confirmed by reading actual generator code, not inferred |
| Architecture | HIGH | Current codebase examined directly; patterns verified against goyang/ygot; the multi-pass compilation pattern is well-established in this domain |
| Pitfalls | HIGH | Grounded in actual codebase (CONCERNS.md), confirmed against goyang/pyang/libyang issue trackers, and cross-referenced with RFC 7950; specific line numbers provided for bugs |

**Overall confidence: HIGH**

### Gaps to Address

- **Deviation application ordering:** ARCHITECTURE.md specifies deviation must run after augment resolution (RFC 7950 §7.12), but the interaction between deviation application and the existing augment convergence loop is not fully specified. Validate the implementation approach before Phase 3.
- **Identityref cross-module base resolution:** The identity hierarchy is compiled into `schema.Module.Identities`, but the generator must resolve cross-module prefix references to find base identities in imported modules. The resolution mechanism for this is not specified in the research; needs to be designed before Phase 3 work on identityref begins.
- **Union member ordering in RFC 7951 codec:** FEATURES.md flags that union member ordering and JSON decode ambiguity need audit, but the correct behavior is not specified. The RFC 7951 codec behavior for union types should be validated against RFC 7951 §4 during Phase 3.
- **RPC struct design (method vs. standalone):** This is the primary open design question for Phase 4; it must be resolved before any RPC generation code is written to avoid an API change post-v1.

---

## Sources

### Primary (HIGH confidence)
- RFC 7950 — The YANG 1.1 Data Modeling Language (official spec): https://www.rfc-editor.org/rfc/rfc7950
- RFC 7951 — JSON Encoding of Data Modeled with YANG: https://www.rfc-editor.org/rfc/rfc7951
- openconfig/goyang v1.6.3 (July 2025) on GitHub: https://github.com/openconfig/goyang — augment chaining failure mode confirmed in issue #265
- openconfig/ygot v0.34.0 (September 2025) on GitHub: https://github.com/openconfig/ygot — golden file testing pattern, `go/format.Source()` usage confirmed
- pkg.go.dev — go/format stdlib package: https://pkg.go.dev/go/format
- golangci-lint v2 official docs: https://golangci-lint.run/docs/product/changelog/
- Go compiler error limit (10 errors default): https://github.com/golang/go/issues/5142
- gotya CONCERNS.md — direct codebase audit (2026-03-14)
- gotya PROJECT.md — requirements and known gaps (2026-03-14)
- Direct code audit: generator/golang/generator.go, generator/protobuf/generator.go, schema/schema.go, compiler/compiler.go

### Secondary (MEDIUM confidence)
- ygot YANG-to-Protobuf transformations spec: https://github.com/openconfig/ygot/blob/master/docs/yang-to-protobuf-transformations-spec.md
- ygot golden file pattern confirmed in ygen/genstate_test.go (updateGolden flag): https://github.com/openconfig/ygot
- mockery v3.6.0 announcement: https://topofmind.dev/blog/2025/04/08/announcing-mockery-v3/
- goldie v2.8.0: https://pkg.go.dev/github.com/sebdah/goldie/v2 — evaluated and rejected in favor of stdlib approach
- pyang issue #183 — circular dependency detection: https://github.com/mbj4668/pyang/issues/183
- CESNET/libyang issue #285 — augment resolution in imported modules: https://github.com/CESNET/libyang/issues/285

### Tertiary (supporting, issue trackers)
- facebookincubator/ent issue #275 — reserved keyword collision in codegen (confirms Go keyword collision is a real-world problem class)
- golang/protobuf issue #1206 — Go name conflicts from non-style-compliant proto names
- YangModels/yang GitHub corpus — reference for real-world YANG module patterns: https://github.com/YangModels/yang

---
*Research completed: 2026-03-14*
*Ready for roadmap: yes*

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

# Stack Research

**Domain:** YANG parsing and code generation library (Go)
**Researched:** 2026-03-14
**Confidence:** HIGH for core stack (all verified against official sources); MEDIUM for ecosystem peers (WebSearch + pkg.go.dev)

---

## Current Stack (What gotya Already Uses)

| Technology | Version | Purpose |
|------------|---------|---------|
| Go | 1.25.5 | Only language; library and generated code are both Go |
| github.com/stretchr/testify | v1.11.1 | assert/require/mock in tests |
| vektra/mockery | v3.x | Interface mock generation (dev tool, not a dep) |
| golangci-lint | v2.x | Multi-linter aggregator (dev tool, not a dep) |

No framework, no database, no service clients. The current dep graph is intentionally minimal and should stay that way (PROJECT.md explicitly states: "No new dependencies: Prefer stdlib solutions").

---

## Ecosystem Peers — What They Do and What to Learn From Them

### openconfig/goyang — v1.6.3 (July 2025)

**What it is:** The canonical Go YANG parser. Converts `.yang` files to an in-memory AST (`pkg/yang`) or a more fully-resolved Entry tree. It is explicitly "a work in progress" and does not attempt code generation.

**What ygot does with it:** ygot uses goyang as its parsing backend, then ygen does code generation on top of the Entry tree goyang produces.

**Confidence:** HIGH — verified on pkg.go.dev and GitHub (July 2025 release).

**What to learn:**
- goyang exposes a two-layer model: raw AST and a resolved `Entry` tree. gotya has the same structure (AST → compiled schema). This layering is correct.
- goyang's Entry tree normalizes `augment`, `grouping`, `uses`, and `typedef` resolution before any generator sees the schema. gotya's compiler handles this too — but the active bugs in augment/refine resolution and circular typedef resolution are exactly the class of problems goyang has battle-tested solutions for. Reading goyang's resolution code is worthwhile during the bug-fix phase.
- goyang tests its parser with a corpus of real `.yang` files. gotya already has a `test/assets/yangs/` directory with ~160 real BBF/IETF/IEEE models. That corpus should be wired into integration tests, not just sitting on disk unused.

**Do NOT adopt goyang as a dependency.** gotya has its own lexer/parser/AST — importing goyang would create a parallel runtime representation with no benefit, and goyang itself says to use ygot if you want code gen.

---

### openconfig/ygot — v0.34.0 (September 2025)

**What it is:** The production-grade YANG-to-Go code generator used in production OpenConfig network stacks. It generates Go structs, validation methods, and JSON (RFC 7951) marshaling. Uses goyang for parsing, ygen for code generation, ygot for runtime helpers, ytypes for schema-driven validation.

**Confidence:** HIGH — verified on GitHub (September 2025 release).

**What to learn:**

#### Testing pattern: golden files over inline string assertions

ygot's code generation tests (`ygen/`) store expected output in `testdata/structs/*.formatted-txt` files and compare actual generator output byte-for-byte against those files. Tests accept a `-update` flag to regenerate golden files when output intentionally changes.

gotya's current generator tests use `assert.Contains(t, out, "expected snippet")` — this catches regressions on specific snippets but completely misses unexpected output, wrong ordering, extra fields, or structural issues. For a code generator, golden-file testing is the correct pattern: the whole output is the contract, not a selection of substrings.

**Recommended action:** Add golden file testing for both Go and Protobuf generators, with a `-update` flag to regenerate. No new dependency required — this is plain `os.ReadFile` / `bytes.Equal` / flag logic.

#### Testing pattern: YANG corpus integration tests

ygot runs its generator against the full OpenConfig YANG model set as part of CI. gotya has ~160 real BBF/IETF/IEEE models in `test/assets/yangs/` but no test that feeds them through the full pipeline and checks the output compiles. A "smoke" integration test that parses all assets, compiles them, and generates Go/Proto output (checking only that no panic occurs and the output passes `go/format.Source()` syntax validation) would catch the augment/refine bugs that are currently only known from manual testing.

#### Generation quality: `go/format.Source()` round-trip

ygot passes all generated Go source through `go/format.Source()` before writing output. This serves two purposes: it catches syntax errors in generated code immediately (instead of silently writing invalid Go), and it ensures the output is always gofmt-canonical. gotya's Go generator should do the same.

`go/format` is stdlib — no dependency required. Use `go/format.Source([]byte)` which returns `([]byte, error)`; an error means the generator produced syntactically invalid Go.

---

## Recommended Stack Changes (What to Add)

The project constraint is "no new dependencies without justification." Only one addition is justified, and it is borderline optional:

### Option A: No new dependencies (recommended)

Implement golden file testing with plain stdlib:

```go
// In generator_test.go
var update = flag.Bool("update", false, "update golden files")

func checkGolden(t *testing.T, name string, got []byte) {
    t.Helper()
    path := filepath.Join("testdata", name+".golden")
    if *update {
        require.NoError(t, os.WriteFile(path, got, 0o644))
        return
    }
    want, err := os.ReadFile(path)
    require.NoError(t, err)
    assert.Equal(t, string(want), string(got))
}
```

This pattern is used by the Go standard library itself (`go/format`, `go/doc`, etc.) and requires zero new dependencies.

### Option B: Add goldie v2 (optional, low value here)

goldie v2.8.0 (October 2025) provides the golden file pattern with added conveniences: `go test -update` flag integration, colored diffs, JSON/XML pretty-printing. It does not integrate with testify's `assert` directly but works alongside it.

**Do NOT add goldie.** The stdlib approach is 20 lines and gotya's PROJECT.md explicitly resists new deps. goldie adds value in projects with dozens of test fixtures needing templating; gotya's generator tests are straightforward byte comparisons.

---

## Development Tools (Keep and Upgrade)

### golangci-lint — upgrade to v2

The codebase already uses golangci-lint v2 (configured in `.golangci.yml`). Current stable is v2.1.2+ (April 2025). Note that v2 changed configuration schema significantly:

- `enable-all` / `disable-all` replaced with `linters.default: [all|standard|none|fast]`
- Requires `version: "2"` at top of `.golangci.yml`
- New `golangci-lint migrate` command assists with config migration

**Confidence:** HIGH — verified on golangci-lint.run official docs.

If the existing `.golangci.yml` was written for v1, run `golangci-lint migrate` before the v1.0 release.

### mockery — already at v3

mockery v3.6.0 (2025) is current. The codebase already uses mockery. No action needed unless `.mockery.yaml` was written for v2 config schema (check the [v3 migration guide](https://vektra.github.io/mockery/v3.0/v3/)).

### go/format (stdlib) — add to generators

Use `go/format.Source([]byte) ([]byte, error)` in the Go generator's output path. Not a new dependency — it is in the Go standard library at `go/format`. This validates generated code before writing and formats it canonically.

---

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| openconfig/goyang as dependency | gotya has its own parser/AST; importing goyang creates a parallel in-memory representation with no benefit; goyang itself is "not complete" | gotya's own lexer/parser/compiler |
| openconfig/ygot as dependency | ygot is a competing implementation, not a utility library; its runtime types are incompatible with gotya's generated types | gotya's own generators |
| goldie or other golden-file libraries | 20-line stdlib alternative exists; PROJECT.md restricts new deps | os.ReadFile + bytes.Equal + -update flag |
| jennifer (code-gen library) | gotya already has a working text/template-based generator; jennifer would require a full generator rewrite for no correctness gain | text/template (already in use) |
| stretchr/testify v2 (go-openapi fork) | The go-openapi fork is a separate project; stretchr/testify v1.11.1 is the correct canonical version; no v2 exists at stretchr/testify | stretchr/testify v1.11.1 |

---

## Patterns the Ecosystem Uses That gotya Should Adopt

These are patterns, not dependencies. Zero new deps required.

### 1. Golden file testing for generators

**Pattern:** Store complete expected generator output in `testdata/*.golden` files. Compare full output, not substrings. Accept `-update` flag to regenerate.

**Why:** `assert.Contains` catches regressions on specific snippets but misses: extra unexpected fields, wrong struct ordering, malformed imports, missing methods. A golden file test is the full contract.

**Where to apply:** `generator/golang/generator_test.go` and `generator/protobuf/generator_test.go`.

### 2. `go/format.Source()` validation in generators

**Pattern:** After building the output buffer, pass it through `go/format.Source()`. Return an error if it fails. Write the formatted bytes, not the raw buffer.

**Why:** Catches generator bugs at the point of generation, not when the user tries to compile. Also eliminates the need to manually maintain whitespace in templates.

**Where to apply:** `generator/golang/generator.go` output path.

### 3. Corpus smoke tests against real YANG assets

**Pattern:** A single test that iterates all files in `test/assets/yangs/`, runs parse→compile→generate on each, and asserts no panic and (for Go generator) no `go/format.Source()` error.

**Why:** gotya already has 160+ real BBF/IETF/IEEE models. Running them through the pipeline is the cheapest way to find augment resolution crashes, typedef panics, and nil dereferences before users do. ygot uses the OpenConfig public model set for the same purpose.

**Where to apply:** New `test/integration_test.go` (or alongside existing `test/test.go`).

### 4. Error sentinel values over string matching in tests

**Pattern:** Define exported `var Err... = errors.New(...)` sentinels for expected compiler errors. In tests, use `errors.Is()` or `require.ErrorIs()` rather than `assert.Contains(t, err.Error(), "some string")`.

**Why:** String-matched error messages are fragile; wording changes break unrelated tests. Compiler error types also make it easier for library users to handle errors programmatically.

**Where to apply:** compiler package, wherever `errors.New()` or `fmt.Errorf()` produce errors that tests currently match by string.

---

## Stack Variants

**If adding RFC 7951 codec test coverage:**
- Use the existing `codec/rfc7951/` package directly in tests
- Test against the same YANG corpus as the generator tests
- No new dependencies

**If building generation audit tooling (for the "audit Go/Proto generators" work item):**
- Parse `test/assets/yangs/` in a standalone main program or test binary
- Use `go/format.Source()` to validate Go output syntax
- Diff actual output against expected via standard `diff` semantics (bytes.Equal)
- No new dependencies

---

## Version Compatibility

| Package | Compatible With | Notes |
|---------|-----------------|-------|
| stretchr/testify v1.11.1 | Go 1.21+ | v1.11.1 is the current latest at stretchr; not the go-openapi fork |
| golangci-lint v2 | Go 1.22+ | v2 config schema differs from v1; run `migrate` if upgrading |
| mockery v3.x | Go 1.21+ | v3 config schema differs from v2; see migration guide |
| go/format (stdlib) | all supported Go versions | Part of Go toolchain; no separate installation |

---

## Sources

- [openconfig/goyang on GitHub](https://github.com/openconfig/goyang) — v1.6.3 (July 2025), confirmed active; HIGH confidence
- [openconfig/ygot on GitHub](https://github.com/openconfig/ygot) — v0.34.0 (September 2025), golden file testing pattern; HIGH confidence
- [pkg.go.dev — go/format](https://pkg.go.dev/go/format) — Source() and Node() API; HIGH confidence (stdlib)
- [pkg.go.dev — stretchr/testify](https://pkg.go.dev/github.com/stretchr/testify) — v1.11.1 current; HIGH confidence
- [goldie v2 on pkg.go.dev](https://pkg.go.dev/github.com/sebdah/goldie/v2) — v2.8.0 (October 2025); MEDIUM confidence (WebSearch + pkg.go.dev)
- [golangci-lint v2 official docs](https://golangci-lint.run/docs/product/changelog/) — v2 config migration; HIGH confidence
- [mockery v3 announcement](https://topofmind.dev/blog/2025/04/08/announcing-mockery-v3/) — v3.6.0 current; MEDIUM confidence (WebSearch)
- [YangModels/yang on GitHub](https://github.com/YangModels/yang) — reference YANG module corpus for integration testing; HIGH confidence

---

*Stack research for: gotya — YANG parsing and Go code generation library*
*Researched: 2026-03-14*

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

# Pitfalls Research

**Domain:** YANG parsing/compilation/codegen library (Go)
**Researched:** 2026-03-14
**Confidence:** HIGH — findings are grounded in the actual codebase (CONCERNS.md), confirmed against goyang/pyang/libyang issue trackers, and cross-referenced with RFC 7950.

---

## Critical Pitfalls

### Pitfall 1: Augment Chaining Silently Produces Incomplete Schemas

**What goes wrong:**
When augment A creates a node that augment B then targets, a single-pass or naive iterative resolver may silently skip augment B because its target did not yet exist at resolution time. The schema compiles without error, but the nodes from augment B are absent. Users get a silently truncated schema.

The gotya compiler uses an iterative loop (`compiler.go:172-197`) that terminates when no progress is made, but has no max-iteration guard and no explicit check that all augments were resolved. A convergence failure produces zero diagnostic output.

Real-world confirmation: openconfig/goyang issue #265 documents exactly this failure mode — "augment not found" when a second augment targets a node that was itself created by a prior augment in the same compilation run.

**Why it happens:**
Augment resolution order is determined by module load order, not dependency order. When the graph is processed linearly, a later augment may see a schema tree that does not yet include nodes from an earlier augment that has not been resolved yet.

**How to avoid:**
1. Add a max-iteration cap (e.g. 1000 iterations) to the convergence loop with a hard error if the cap is reached.
2. After the loop exits, assert that `pendingAugments` is empty. If not, emit errors naming each unresolved augment path — never silently succeed.
3. Add a test: module A augments `/base/c1`, module B augments `/base/c1/c2` where `c2` was added by A's augment. Verify both `c1` and `c2` appear in the compiled output.

**Warning signs:**
- Augment resolution loop exits but `pendingAugments` is non-empty with no error emitted.
- Tests only cover single-level augments on the same module's nodes.
- No test exercises cross-module augment chaining.

**Phase to address:** Bug fix phase (correctness hardening). Must be complete before codegen audit, since the audit may feed real-world YANG files with exactly this pattern.

---

### Pitfall 2: Nil Pointer Panic on Malformed Augment/Refine Paths

**What goes wrong:**
`findNode()` returns `nil` for any path segment that does not exist. The callers at `compiler.go:774` and `compiler.go:992` do not check for nil before calling methods on the returned node. On any YANG file with a partially-valid augment or refine path, the compiler panics rather than returning an error.

This is a library-level correctness issue: a panic in a library function is always a bug from the consumer's perspective, regardless of how malformed the input is.

**Why it happens:**
The debug `fmt.Printf` at line 778 fires when nil is encountered, indicating this code path was observed during development but never hardened — the printf was left as a placeholder for what should be error propagation.

**How to avoid:**
1. In `findNode()` callers, check `if node == nil { c.addError(...); return }` before dereferencing.
2. Remove both `fmt.Printf` debug statements (`compiler.go:778` and `compiler.go:930`) and route through the error list.
3. Add tests: augment with a path segment that refers to a non-existent container; refine with a path that partially matches then diverges. Both must return errors, not panics.

**Warning signs:**
- `fmt.Printf("DEBUG findNode` still present in compiler.go.
- No test that feeds a malformed augment path and asserts an error (not a panic).
- `go test -fuzz` or manual corpus testing triggers a panic.

**Phase to address:** Bug fix phase, first priority. Panics in library code are a hard blocker for v1.

---

### Pitfall 3: Circular Typedef Resolution Causes Stack Overflow

**What goes wrong:**
`getType()` resolves typedefs by recursively calling itself (`compiler.go:796`) with no depth limit. If a YANG file defines `typedef A { type B; }` and `typedef B { type A; }`, the compiler recurses until the goroutine stack overflows — an unrecoverable runtime crash, not a caught error.

Unlike a panic that can be recovered, a stack overflow in Go terminates the entire process.

**Why it happens:**
The circular typedef case is rare in well-formed YANG, so it was not defended against during initial implementation. The `visited` pattern used for grouping resolution was not applied to typedef resolution.

**How to avoid:**
1. Pass a `visited map[string]struct{}` (or a depth counter) into `getType()` and error immediately on re-entry with the same typedef name.
2. A depth counter of ~50 is sufficient — legitimate YANG typedef chains are at most a few levels deep.
3. Add a test: `typedef A { type B; } typedef B { type A; }` must produce an error, not crash.

**Warning signs:**
- No circular typedef test in `compiler_test.go`.
- `getType()` signature does not accept a `visited` parameter or depth counter.

**Phase to address:** Bug fix phase. Circular typedef is uncommon in production YANG but any public library that crashes on legal-but-pathological input is not v1-ready.

---

### Pitfall 4: Silent Schema Corruption from Ignored AddChild Errors

**What goes wrong:**
Five call sites discard `AddChild()` errors with `_ =` (`compiler.go:415, 418, 426, 429, 728`). If `AddChild()` returns a duplicate-identifier error, the node is either not added or added in a corrupt state, and compilation continues silently. The resulting schema may be missing RPC input/output nodes or may have an incorrect choice/case structure with no diagnostic.

**Why it happens:**
The auto-generated RPC/Action input and output nodes have predictable names, so developers assumed they could not collide. But this assumption is an assertion that was never verified in code. The shorthand case site (`728`) is similar — generated case names were assumed to be unique.

**How to avoid:**
1. Replace every `_ = node.AddChild(...)` with an explicit check: if the error is not a "duplicate identifier" error, propagate it; if it is a duplicate, emit a compiler diagnostic indicating a schema invariant was violated.
2. Alternatively, if these nodes genuinely cannot collide (because they are auto-generated singletons), document that constraint with a comment and add a test that confirms `AddChild()` never returns an error for these cases.
3. Do not silently swallow errors at any call site in a library that users depend on for correctness.

**Warning signs:**
- `_ = *.AddChild(` pattern present in compiler.go.
- No test that exercises duplicate RPC input/output creation and verifies the error is surfaced.

**Phase to address:** Bug fix phase. Silent schema corruption is a correctness blocker.

---

### Pitfall 5: Debug Output Leaking into Library Stdout

**What goes wrong:**
Two `fmt.Printf` calls remain in production compiler code (`compiler.go:778` and `compiler.go:930`). Any program that pipes or captures the output of gotya will receive unexpected diagnostic noise. Scripts and CI pipelines that diff generated output will see spurious diffs. This is a hard disqualifier for a published library.

**Why it happens:**
Debug printfs are added during development and forgotten. There is no lint rule preventing them, and since the paths that trigger them are error paths, they may not appear in ordinary test runs.

**How to avoid:**
1. Remove both `fmt.Printf` calls immediately. Replace with entries in `c.errors`.
2. Add a `go vet` or static analysis check (`forbidigo` linter, or a simple `grep` in CI) that fails on `fmt.Printf` in non-test files under `compiler/`.
3. Run integration tests that capture stdout and assert it is empty during normal compilation.

**Warning signs:**
- `fmt.Printf("DEBUG` strings present in `compiler.go`.
- No CI check that asserts clean stdout during compilation of valid YANG.

**Phase to address:** Code quality phase (first, quick wins). This is the highest-visibility indicator of an unfinished library.

---

### Pitfall 6: Circular Import Not Detected — Infinite Recursion Risk

**What goes wrong:**
The module loader (`cmd/gotya/loader.go`) caches loaded modules, but there is no cycle detection at the import resolution level. If module A imports B and B imports A, the loader recurses until the call stack exhausts — a runtime crash. The `gotya.Compile()` public API exposes this risk directly to library consumers.

**Why it happens:**
The caching approach (`loader.go:87-112`) assumes that cached modules act as a natural cycle breaker, but caching a module that is still in the process of being loaded (i.e., "currently on the import stack") is different from caching a module that has finished loading. If the in-progress module is cached before resolution completes, the cycle is hidden; if it is cached after, the recursion still occurs.

**How to avoid:**
1. Maintain an "in-progress" set (separate from the "completed" cache) during module loading.
2. If a module name appears in the in-progress set when a new `Load()` call is made for it, return a `circular import` error immediately.
3. Add a test: two YANG files that import each other. Must produce a clear error, not a crash.

**Warning signs:**
- No circular import test in the test suite.
- Loader code does not distinguish between "loading" and "loaded" states.

**Phase to address:** Bug fix phase. Circular imports are a documented gap in `CONCERNS.md` and a known risk path.

---

### Pitfall 7: Unbounded Error Accumulation as a Denial-of-Service Vector

**What goes wrong:**
`c.errors` (`compiler.go:22`) grows without limit. A YANG file crafted (or accidentally generated) to produce thousands of validation errors will cause the compiler to consume unbounded memory before returning. For a library used in build pipelines or CI, this is a practical availability issue.

**Why it happens:**
Error accumulation was chosen over fail-fast for better diagnostics — a good design decision. The implementation omitted an upper bound, which is a common oversight when the "many errors" case is never tested.

**How to avoid:**
1. Cap at a configurable limit (default 100 errors). When the cap is reached, add a final synthetic error: `"too many errors; compilation stopped after N errors"` and return immediately.
2. Expose the cap as a field in `compiler.Options` so callers can raise it if needed.
3. Add a test that feeds a module generating 200+ errors and verifies the error count is bounded and the truncation message appears.

**Warning signs:**
- No `MaxErrors` field in `compiler.Options`.
- No test for high-error-count input.
- `c.errors` slice has no capacity cap.

**Phase to address:** Bug fix phase. Pair with the existing "bound compiler error accumulation" task in PROJECT.md.

---

### Pitfall 8: Codegen Audit Scope Creep — Shipping an Incomplete Audit as a Complete One

**What goes wrong:**
The PROJECT.md explicitly flags that neither Go nor Proto generator output has been "systematically checked against the full YANG feature surface." The risk is: the audit is started, some gaps are found and fixed, and then the audit is declared done before the full feature surface is covered. v1 ships with silent codegen gaps.

YANG has many constructs that are easy to miss: `anydata`, `anyxml`, `action` (vs. `rpc`), `notification`, `deviation`, `feature`/`if-feature` pruning effect on generated types, union types with multiple member types, `leafref` with `require-instance false`, bit types, and identity/identityref resolution.

Known parallel risk from the goyang ecosystem: openconfig/goyang has 31 open issues, many in categories like "missing statement support" and "typedef resolution across module boundaries" — indicating these are the areas most likely to have gaps.

**Why it happens:**
Feature surface for YANG codegen is large and lacks a natural checklist. Without a structured test matrix, developers fix what they encounter and leave the rest invisible.

**How to avoid:**
1. Before running the audit, create an explicit checklist of all YANG statement types and their expected codegen behavior. Use RFC 7950 Section 7 as the authoritative list.
2. For each statement type, write a minimal YANG snippet and assert the generated Go/Proto output contains the expected construct.
3. Treat the audit as a test-writing exercise, not a manual inspection exercise. Each gap found becomes a failing test; fixes make the tests pass.
4. The audit is done when every statement type in the checklist has a passing test.

**Warning signs:**
- Audit produces a list of findings in a document rather than a list of failing tests.
- No structured RFC 7950 statement coverage matrix exists.
- "Audit complete" is declared before `anydata`, `action`, `notification`, and union codegen have explicit test cases.

**Phase to address:** Codegen audit phase. The structure of the audit (test-matrix approach vs. ad-hoc inspection) determines whether v1 ships with confidence or with hidden gaps.

---

### Pitfall 9: Public API Freezing Too Early or Too Late

**What goes wrong:**
Two failure modes exist:
- Freeze too early: `gotya.go` exports `os.ErrInvalid` as the parse error and uses a `// Simplistic error for facade demonstration` comment. If this ships as v1, callers will write `if errors.Is(err, os.ErrInvalid)` and those callers will break if the error is later made descriptive.
- Freeze too late: Internal types leak through the public API (e.g., `type ASTModule = ast.Module` exposes the `ast` package's type directly). If `ast.Module` gains or changes methods, the API contract changes without a major version bump.

**Why it happens:**
API design under time pressure defers "clean this up later." Once downstream code exists, cleaning up is a breaking change.

**How to avoid:**
1. Before tagging v1, audit every exported symbol in `gotya.go`. For each: is the error type a sentinel or a structured type? Is the return type an internal package type or a stable public type?
2. The `os.ErrInvalid` usage in `Parse()` must be replaced with either a domain-specific error type (`gotya.ParseError`) or at minimum `fmt.Errorf("parse: %w", ...)` with a message that does not commit to a sentinel.
3. The `// Simplistic error for facade demonstration` comment is a warning sign — any comment of this form in `gotya.go` before v1 is a bug.
4. The `ASTModule = ast.Module` alias is acceptable only if `ast.Module` is considered a stable type. Document this decision explicitly.

**Warning signs:**
- `os.ErrInvalid` returned from `Parse()`.
- `// Simplistic`, `// placeholder`, `// TODO`, or `// for demonstration` comments in `gotya.go`.
- Parser errors are not propagated to the caller (currently the parser's `Errors()` slice is discarded in `Parse()`).

**Phase to address:** API stabilization phase. Must precede v1 tag. Cannot be retrofitted after v1 without a major version bump.

---

### Pitfall 10: Generated Code Name Collisions with Go Keywords or Existing Identifiers

**What goes wrong:**
YANG identifiers use hyphens and allow names that are valid Go keywords (`type`, `range`, `default`, `interface`, etc.) or names that collide with generated method names. Without sanitization, the Go generator produces uncompilable output for any YANG module containing a leaf named `type` or a container named `interface`.

Parallel evidence: the Ent framework (issue #275) and Pulumi (issue #5136) both encountered exactly this failure mode. protoc-gen-go acknowledges a "remote possibility of name collision" in its CamelCase rewrite.

**Why it happens:**
YANG namespaces are separate from Go namespaces, and the mapping is not always bijective. Early codegen implementations often handle the common cases (alphanumeric names) and discover edge cases only when users file bugs against real-world YANG models.

**How to avoid:**
1. Maintain an explicit list of Go reserved keywords and common collision candidates (`String`, `Error`, `Reset`, `ProtoMessage`, `GetXxx` for any leaf named `Xxx`).
2. When a YANG identifier maps to a collision, apply a deterministic renaming strategy (e.g., append `_` or a domain-specific suffix) and document the strategy.
3. The codegen audit (Pitfall 8) must include YANG modules with identifier names that are Go keywords.
4. Add a test: YANG module with a leaf named `type` and a container named `range`. Generated Go must compile.

**Warning signs:**
- No keyword collision check in the Go or Proto generator.
- No test for YANG identifiers that are Go reserved words.
- `CONCERNS.md` notes "No Input Validation on YANG Identifiers" as an open issue.

**Phase to address:** Codegen audit and improvement phase.

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| `_ = node.AddChild(...)` | Avoids handling errors for "impossible" cases | Silent schema corruption; no diagnostic when the "impossible" case occurs | Never — comment with rationale and add an explicit panic or log instead |
| `fmt.Printf("DEBUG ...")` in compiler | Fast dev feedback during implementation | Pollutes stdout in library consumers; breaks scripting | Never in production code — use a logger or remove |
| `os.ErrInvalid` as parse error | Quick placeholder | Callers check for this sentinel; changing it post-v1 is a breaking change | Never in a stable public API |
| No circular-import guard | Simpler loader code | Infinite recursion crash on pathological (but valid) input | Never in a published library |
| Error accumulation without a cap | Collects all diagnostics | Unbounded memory growth on malformed input | Acceptable in dev/diagnostic mode; must be bounded with a configurable cap before v1 |
| Test only the happy path for augments | Tests pass quickly | Core feature surface (augment chaining, cross-module) is untested; regressions invisible | Only acceptable during initial prototyping, not before v1 |

---

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| Module loader + import paths | Silently skipping unreadable directories makes "module not found" errors appear as if the module does not exist, when the real cause is a permissions or path error | Collect unreadable-directory errors and include them in the "module not found" diagnostic, or fail explicitly on unreadable configured paths |
| Multi-module compilation via `gotya.Compile()` | Compiling each module independently means cross-module augments are not resolved; augments from module A targeting module B's nodes are silently skipped | The public `Compile()` function must ensure all modules share a single compiler instance so cross-module augment resolution can proceed |
| Generated Go structs with RFC 7951 JSON codec | YANG leaf names with hyphens are valid and common; Go struct fields cannot have hyphens; if the JSON tag is not set to the original YANG name, RFC 7951 serialization produces wrong output | The Go generator must emit `json:"yang-leaf-name"` tags using the original YANG identifier, not the Go-sanitized field name |
| Proto generation with CEL validation | CEL expressions reference YANG paths; if the generator does not validate that referenced paths exist in the schema, generated proto files will have invalid CEL annotations | Run a post-generation schema-path check against generated CEL annotations before writing output |

---

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| O(N*K) augment resolution loop | Compilation time grows super-linearly with number of augments and augment chain depth | Build a dependency graph of augments; process topologically (or sort by path depth) | Noticeable with 50+ augments across 5+ modules; acute with IETF/OpenConfig-scale models |
| String path splitting on every `findNode()` call | `findNode()` CPU time visible in profiles for large schemas | Pre-compute and cache split path segments at augment parse time | Large YANG schemas with deep augment chains (100+ node paths) |
| Unbounded `c.errors` growth | OOM on heavily malformed input | Cap at configurable N (default 100), report truncation | Any input that generates 10,000+ errors |
| No cross-module type deduplication | Redundant schema nodes across modules waste memory; RFC 7951 codec may generate duplicate logic | Shared compiler state across a multi-module compile | Large multi-module systems (OpenConfig BGP + routing + interfaces combined) |

---

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| YANG identifier used directly as a Go identifier in generated code without sanitization | Generated code with unsanitized identifiers may compile with unexpected behavior, or may include user-controlled strings in positions that affect compilation | Validate that all YANG identifiers match the YANG identifier regex (`[a-zA-Z_][a-zA-Z0-9_\-.]*`) before use in codegen |
| Module name used as filename component without restriction to configured paths | `filepath.Clean()` reduces but does not eliminate path traversal; a module named `../../../etc/passwd` could read outside the search root | Validate module names against YANG identifier rules before constructing file paths; restrict file lookups to configured search directories |
| Cross-module grouping resolution without module ownership validation | A malicious or malformed YANG file could reference groupings from modules it did not import | When resolving `externalGroupStack` references, verify the grouping's owning module matches a declared import prefix |

---

## DX Pitfalls (Library Consumer Experience)

| Pitfall | Consumer Impact | Better Approach |
|---------|-----------------|-----------------|
| Parser errors discarded in `Parse()` | Callers get `os.ErrInvalid` with no location, no message; impossible to show useful errors to end users | Return parser errors as a structured type with file, line, column, and message |
| Unreadable directory silently skipped | User configures a typo'd search path; library silently finds nothing; user spends hours debugging the wrong thing | Return a diagnostic warning for each unreadable path in the search list |
| No way for callers to bound error count | A caller building a language server that compiles on every keystroke can OOM if the schema is temporarily invalid | Expose `MaxErrors int` in `compiler.Options` |
| Debug printf on stderr/stdout | CI pipelines fail diff assertions; editor extensions display garbage in their output panes | Zero-output contract: a library must produce no stdout/stderr output during normal operation |

---

## "Looks Done But Isn't" Checklist

- [ ] **Augment resolution:** Compiles successfully with single-level augments — verify multi-level chained augments across two modules also produce correct output.
- [ ] **Circular typedef:** `getType()` exists and resolves typedefs — verify a circular typedef definition produces an error, not a crash.
- [ ] **Circular import:** Module loader uses a cache — verify two files that import each other produce an error, not infinite recursion.
- [ ] **Error propagation:** Compiler collects errors — verify errors from `AddChild()` failures are included, not silently discarded.
- [ ] **Clean output:** Library compiles YANG — verify stdout/stderr is empty during compilation; no debug printfs.
- [ ] **Public API errors:** `Parse()` returns an error — verify the error is descriptive (not `os.ErrInvalid`) and carries parser diagnostics.
- [ ] **Codegen completeness:** Go generator produces output for the test YANG files — verify it also handles `anydata`, `action`, `notification`, union with multiple members, and identityref.
- [ ] **Keyword safety:** Go generator produces valid Go for common YANG field names — verify a leaf named `type` and a container named `range` produce compilable Go output.
- [ ] **Convergence guarantee:** Augment resolution loop terminates — verify it also terminates with an explicit error for a circular augment dependency pattern, not silently.

---

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Nil pointer panic shipped to users | HIGH | Issue a patch release; add nil-check guard; announce in changelog as a bug fix |
| Stack overflow from circular typedef shipped to users | HIGH | Patch release; add visited-set guard; likely requires emergency semver patch |
| Debug printf discovered post-v1 | LOW | Patch release; remove printfs; no API changes required |
| Silent augment resolution failure discovered via user bug report | MEDIUM | Add convergence assertion; write regression test from user's YANG files; patch release |
| Unbounded error accumulation reported as OOM | MEDIUM | Add cap with `MaxErrors` in Options; patch release; default of 100 is backward compatible |
| API error type change post-v1 (e.g. fixing `os.ErrInvalid`) | HIGH | Requires v2 major version or compatibility shim; cannot be fixed in a patch |

---

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Augment chaining produces incomplete schema | Bug fix phase — add convergence assertion and test | Test: two-module augment chain; assert both augmented nodes appear |
| Nil pointer panic in node traversal | Bug fix phase — add nil checks, remove debug printfs | Test: malformed augment path returns error, not panic |
| Circular typedef stack overflow | Bug fix phase — add visited set in `getType()` | Test: `typedef A { type B; } typedef B { type A; }` returns error |
| Silent AddChild error swallowing | Bug fix phase — handle or document all `_ =` sites | Test: duplicate RPC input creation produces a diagnostic |
| Debug printf in production | Code quality phase (first, quick wins) | CI: stdout is empty for any valid YANG compilation |
| Circular import crash | Bug fix phase — add in-progress set to loader | Test: two mutually-importing modules produce a clear error |
| Unbounded error accumulation | Bug fix phase — add MaxErrors cap | Test: 200-error input produces bounded output with truncation message |
| Codegen gaps not discovered until user reports | Codegen audit phase — structured RFC 7950 statement matrix | Audit checklist signed off; every statement type has a passing codegen test |
| API frozen with placeholder errors | API stabilization phase — audit gotya.go before v1 tag | No `os.ErrInvalid`, no `// Simplistic` comments; parser errors propagate |
| Go keyword collision in generated code | Codegen audit and improvement phase | Test: YANG with leaf named `type`; generated Go compiles |

---

## Sources

- openconfig/goyang issue tracker — confirmed augment chaining failure (issue #265), namespace collision in deviate+augment (issue #146), typedef resolution across submodules (issue #239): https://github.com/openconfig/goyang/issues
- pyang issue #183 — false-positive circular dependency detection from statement ordering: https://github.com/mbj4668/pyang/issues/183
- CESNET/libyang issue #285 — augment targets not resolved in imported modules: https://github.com/CESNET/libyang/issues/285
- golang/protobuf issue #1206 — Go name conflicts from non-style-compliant proto names: https://github.com/golang/protobuf/issues/1206
- facebookincubator/ent issue #275 — reserved keyword collision in codegen: https://github.com/facebookincubator/ent/issues/275
- RFC 7950 Section 7.17 (augment), Section 7.16 (uses/refine), Section 7.3 (typedef): https://www.rfc-editor.org/rfc/rfc7950.html
- Go module release and versioning workflow (API stability commitments): https://go.dev/doc/modules/release-workflow
- gotya CONCERNS.md — direct codebase audit (2026-03-14)
- gotya PROJECT.md — requirements and known gaps (2026-03-14)

---
*Pitfalls research for: YANG parsing/compilation/codegen library (Go)*
*Researched: 2026-03-14*