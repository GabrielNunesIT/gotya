# gotya

## What This Is

gotya is a Go library and CLI tool for working with YANG data models (RFC 7950). It provides a complete, production-ready pipeline: parse YANG source files → compile to a validated semantic schema → generate idiomatic Go structs or Protobuf definitions. It targets Go developers building network management tooling who need to consume YANG-modeled data. The library is now v1-stable: public API is intentional and documented, the compiler handles any input without panicking, and generators cover all major YANG statement types.

## Core Value

Parse YANG, compile it to a validated schema, and generate correct, usable Go or Protobuf code from it — reliably enough to ship as a library others depend on.

## Requirements

### Validated

- ✓ YANG lexer and recursive descent parser (produces `*ast.Module`) — existing (pre-v1.0)
- ✓ Compiler: AST → semantic schema with type resolution and reference linking — existing (pre-v1.0)
- ✓ Go code generator: structs, getters, setters, fakeroot, ordered maps — existing (pre-v1.0)
- ✓ Protobuf code generator with CEL validation support — existing (pre-v1.0)
- ✓ RFC 7951 JSON codec for generated Go types — existing (pre-v1.0)
- ✓ CLI tool with configurable flags for all generation options — existing (pre-v1.0)
- ✓ Directory-based module loader for resolving YANG imports — existing (pre-v1.0)
- ✓ Library never panics or crashes on malformed YANG input — v1.0 (COMP-01)
- ✓ Circular typedef definitions produce a compile error, not a stack overflow — v1.0 (COMP-02)
- ✓ Circular module imports produce a compile error, not infinite recursion — v1.0 (COMP-03)
- ✓ Augment resolution loop has max-iteration cap and convergence assertion — v1.0 (COMP-04)
- ✓ All `AddChild()` failures propagate as errors — v1.0 (COMP-05)
- ✓ Library emits nothing to stdout/stderr during normal operation — v1.0 (COMP-06)
- ✓ Compiler error accumulation bounded by `Options.MaxErrors` (default 100) — v1.0 (COMP-07)
- ✓ Corpus smoke test: all yangs in `test/assets/yangs/` through full pipeline — v1.0 (TEST-01)
- ✓ Go generator validates output with `go/format.Source()` before writing — v1.0 (TEST-02)
- ✓ Compiler error conditions use typed error sentinels, not string-matched messages — v1.0 (TEST-03)
- ✓ Identityref leaf types generate typed Go `const` blocks — v1.0 (GOGEN-01)
- ✓ `anydata`/`anyxml` nodes emit valid Go fields (no silent drops) — v1.0 (GOGEN-02)
- ✓ `rpc`/`action`/`notification` generate typed Go request/response structs — v1.0 (GOGEN-03)
- ✓ Deviation statements applied as pre-pass before code generation — v1.0 (GOGEN-04)
- ✓ RFC 7950 statement coverage matrix in `docs/proto-coverage.md` — v1.0 (PBGEN-01)
- ✓ `anydata`/`anyxml` emit `google.protobuf.Any` in Protobuf output — v1.0 (PBGEN-02)
- ✓ `rpc`/`action` emit Protobuf `service` blocks with typed methods — v1.0 (PBGEN-03)
- ✓ CEL annotation paths validated against compiled schema post-generation — v1.0 (PBGEN-04)
- ✓ `gotya.Parse()` returns `*ParseError` with file, line, column, message — v1.0 (API-01)
- ✓ `gotya.Parse()` returns all parse diagnostics, not just the first — v1.0 (API-02)
- ✓ All exported symbols in `gotya.go` are intentional public API — v1.0 (API-03)
- ✓ No placeholder/TODO comments in `gotya.go` — v1.0 (API-04)

### Active

- [ ] Leafref constraint validation methods at runtime (GOGEN-V2-01)
- [ ] Full `if-feature` expression parser with parenthesized expressions (GOGEN-V2-02)
- [ ] `ordered-by user` leaf-lists preserved as slices in generated Go (GOGEN-V2-03)
- [ ] Proto field options emit YANG source metadata annotations (PBGEN-V2-01)
- [ ] Cross-module type deduplication in Proto output (PBGEN-V2-02)

### Out of Scope

- Security hardening (path traversal, identifier sanitization) — not blocking; low risk in trusted-input library; deferred
- JSON Schema / OpenAPI generator — new output format is a separate milestone; pluggable interface exists
- Runtime XPath / `must`/`when` evaluation — requires full XPath engine; significant scope; separate milestone (RUNTIME-01)
- Streaming/lazy compilation for large schemas (100k+ nodes) — separate milestone (RUNTIME-02)
- Mobile/web tooling — not applicable to a Go library
- Graphical schema browser — out of scope for a library project

## Context

**Current state (v1.0):** Production-ready library. 71,200 lines Go. 22/22 v1 requirements satisfied. Full pipeline tested: parse → compile → Go/Proto generation. Typed error sentinels, corpus tests, structured API diagnostics.

**Tech stack:** Go 1.25+, stdlib only (no external runtime deps). `go/format` for output validation. `google.protobuf.Any` for anydata in Proto output.

**Known tech debt:**
- `Module.Schema()` returns `*schema.Module` directly — callers storing the result must import `schema` package. Explicit design decision (clean accessor); full opaqueness not the goal.
- `generator/golang/generator.go` lines 243, 658: `goType := "*string" // Default placeholder` fallback in switch. All relevant YANG type cases are handled above it; fallback is intentional.

## Constraints

- **Language**: Go only — library and generated code are all Go
- **Compatibility**: Must work with Go 1.25+ module system
- **API stability**: Public API in `gotya.go` is stable at v1; internal packages can change
- **No new dependencies**: Prefer stdlib solutions; any new dep requires justification

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Multi-stage compiler (lexer→parser→AST→schema) | Clean separation, testable at each layer | ✓ Good |
| Pluggable Generator interface | Supports Go and Proto without forking core | ✓ Good |
| Error accumulation over fail-fast | Useful diagnostics for malformed YANG | ✓ Good — bounded by MaxErrors |
| RFC 7951 codec included in library | Common need for users of generated structs | ✓ Good |
| Typed error sentinels (`ErrCircularTypedef`, etc.) | Tests verify structured contracts, not fragile strings | ✓ Good |
| `go/format.Source()` post-generation validation | Catches generated-code bugs at generation time | ✓ Good |
| Corpus smoke test against `test/assets/yangs/` | Regression gate for real-world YANG files | ✓ Good |
| Schema accessor pattern (`Module.Schema()`) | Clean API for generator callers without fully opaque types | ✓ Good — design-intentional |
| Deviation pre-pass before code generation | Correct vendor schema modeling | ✓ Good |
| `ParseError`/`Diagnostic` types for public API | Structured diagnostics; all errors propagated | ✓ Good |
| TDD (failing tests first) throughout all phases | Drove correct behavior, prevented scope creep | ✓ Good |
| Decimal phase numbering for gap closure (Phase 6) | Clear insertion semantics for post-audit work | ✓ Good |

---
*Last updated: 2026-03-15 after v1.0 milestone*
