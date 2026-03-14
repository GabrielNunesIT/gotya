# gotya

## What This Is

gotya is a Go library and CLI tool for working with YANG data models (RFC 7950). It provides a complete pipeline: parse YANG source files → compile to a validated semantic schema → generate idiomatic Go structs or Protobuf definitions. It targets Go developers building network management tooling who need to consume YANG-modeled data.

## Core Value

Parse YANG, compile it to a validated schema, and generate correct, usable Go or Protobuf code from it — reliably enough to ship as a library others depend on.

## Requirements

### Validated

- ✓ YANG lexer and recursive descent parser (produces `*ast.Module`) — existing
- ✓ Compiler: AST → semantic schema with type resolution and reference linking — existing
- ✓ Go code generator: structs, getters, setters, fakeroot, ordered maps — existing
- ✓ Protobuf code generator with CEL validation support — existing
- ✓ RFC 7951 JSON codec for generated Go types — existing
- ✓ CLI tool with configurable flags for all generation options — existing
- ✓ Directory-based module loader for resolving YANG imports — existing

### Active

- [ ] Fix nil pointer panics in node traversal (augment/refine path resolution)
- [ ] Fix silent errors from ignored `AddChild()` failures in RPC/Action/Case nodes
- [ ] Fix augment resolution loop: add max-iteration guard and convergence check
- [ ] Fix circular typedef resolution (no depth limit — potential stack overflow)
- [ ] Remove debug `fmt.Printf` statements left in compiler production code
- [ ] Add proper error reporting for unreadable directories in module loader
- [ ] Bound compiler error accumulation (stop after N errors, report truncation)
- [ ] Add tests: augment chaining (multi-level augment dependencies)
- [ ] Add tests: circular import detection (A imports B imports A)
- [ ] Add tests: `AddChild()` duplicate identifier error propagation
- [ ] Add tests: invalid augment path syntax edge cases
- [ ] Audit Go generator output against real YANG models — identify coverage gaps
- [ ] Audit Protobuf generator output against real YANG models — identify coverage gaps
- [ ] Improve Go/Proto generation based on audit findings

### Out of Scope

- Security hardening (path traversal, identifier sanitization) — deferred; not blocking v1
- New output format generators (JSON Schema, OpenAPI, etc.) — future milestone
- Runtime XPath / `must`/`when` evaluation — significant scope, separate milestone
- Full `if-feature` expression parser with parentheses — tracked as follow-on
- Mobile/web tooling — not applicable

## Context

Brownfield project with substantial existing implementation. The core parsing and generation pipeline is functional. The main gaps before calling this v1-stable are correctness bugs that can cause panics on real-world YANG files, missing test coverage for edge cases, and code quality issues (debug output, silent failures) that would be embarrassing in a published library.

The generation improvement work starts with an audit — neither Go nor Proto output has been systematically checked against the full YANG feature surface yet.

## Constraints

- **Language**: Go only — library and generated code are all Go
- **Compatibility**: Must work with Go 1.25+ module system
- **API stability**: Public API in `gotya.go` should be stable before v1 tag; internal packages can change
- **No new dependencies**: Prefer stdlib solutions; any new dep requires justification

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Multi-stage compiler (lexer→parser→AST→schema) | Clean separation, testable at each layer | ✓ Good |
| Pluggable Generator interface | Supports Go and Proto without forking core | ✓ Good |
| Error accumulation over fail-fast | Useful diagnostics for malformed YANG | — Pending (unbounded accumulation is a bug) |
| RFC 7951 codec included in library | Common need for users of generated structs | ✓ Good |

---
*Last updated: 2026-03-14 after initialization*
