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
