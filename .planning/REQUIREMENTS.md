# Requirements: gotya

**Defined:** 2026-03-14
**Core Value:** Parse YANG, compile it to a validated schema, and generate correct, usable Go or Protobuf code from it — reliably enough to ship as a library others depend on.

## v1 Requirements

### Compiler Correctness

- [x] **COMP-01**: Library never panics or crashes on malformed YANG input — nil pointer dereference on `findNode()` return values guarded with proper error propagation
- [x] **COMP-02**: Circular typedef definitions (A uses B uses A) produce a compile error, not a stack overflow — `getType()` has a visited set and depth limit
- [x] **COMP-03**: Circular module imports produce a compile error, not infinite recursion — loader has an in-progress set for cycle detection
- [x] **COMP-04**: Augment resolution verifies all augments applied — loop has a max-iteration cap and a post-loop assertion that `pendingAugments` is empty
- [x] **COMP-05**: All `AddChild()` failures propagate as errors — no `_ = AddChild(...)` call sites remain; RPC, Action, and Case nodes report duplicate identifier errors
- [x] **COMP-06**: Library emits nothing to stdout or stderr during normal operation — all `fmt.Printf("DEBUG ...`)` calls removed from production code paths
- [x] **COMP-07**: Compiler error accumulation is bounded — `compiler.Options` has a `MaxErrors` field (default 100); compilation stops after limit with truncation message

### Testing Infrastructure

- [x] **TEST-01**: Corpus smoke test runs all YANG files in `test/assets/yangs/` through the full parse→compile→generate pipeline and asserts no panic and syntactically valid Go output
- [x] **TEST-02**: Go generator validates output with `go/format.Source()` before writing — invalid generated Go surfaces at generation time, not at user compile time
- [x] **TEST-03**: Compiler error conditions use typed error sentinels (not string-matched messages) so tests assert on structured values, not fragile substrings

### Go Generator

- [ ] **GOGEN-01**: Identityref leaf types generate a typed Go `const` block for the identity hierarchy, not `*string` — cross-module base resolution supported
- [ ] **GOGEN-02**: `anydata` and `anyxml` nodes emit a valid Go field (e.g., `interface{}` or `json.RawMessage`) — no silent node drops
- [ ] **GOGEN-03**: `rpc`, `action`, and `notification` statements generate typed Go request/response structs — input and output containers emitted as nested structs
- [ ] **GOGEN-04**: Deviation statements are applied as a pre-pass before code generation — `deviate not-supported` removes nodes, `deviate replace` updates type/constraints, `deviate add/delete` updates properties

### Protobuf Generator

- [ ] **PBGEN-01**: RFC 7950 statement coverage matrix exists — every YANG statement type is explicitly marked as supported, unsupported, or out-of-scope in a `docs/proto-coverage.md` document
- [ ] **PBGEN-02**: `anydata` and `anyxml` nodes emit a valid Protobuf field (`google.protobuf.Any` or `bytes`) — no silent node drops
- [ ] **PBGEN-03**: `rpc` and `action` statements emit Protobuf `service` block definitions with `rpc` methods referencing typed request/response messages
- [ ] **PBGEN-04**: Generated CEL annotation paths are validated against the compiled schema after generation — invalid paths produce a generation error, not silent incorrect annotations

### Public API

- [ ] **API-01**: `gotya.Parse()` returns a domain-specific error type (`*gotya.ParseError` or equivalent) — not `os.ErrInvalid`; error includes file, line, column, and message fields
- [ ] **API-02**: Parser diagnostics (all parse errors, not just the first) are propagated through `gotya.Parse()` — callers can access the full error list
- [ ] **API-03**: All exported symbols in `gotya.go` are intentional public API — no internal package types leak through function signatures; documented as stable
- [ ] **API-04**: All placeholder, demonstration, and TODO comments removed from `gotya.go` — public API file documents actual behavior only

## v2 Requirements

### Go Generator Extensions

- **GOGEN-V2-01**: Leafref constraints generate validation method checks at runtime
- **GOGEN-V2-02**: Full `if-feature` expression parser with parenthesized expressions and mixed `and`/`or`/`not` operators
- **GOGEN-V2-03**: `ordered-by user` leaf-lists preserved as slices in generated Go (instead of being sorted)

### Protobuf Generator Extensions

- **PBGEN-V2-01**: Proto field options emit YANG source metadata annotations (module, path, description)
- **PBGEN-V2-02**: Cross-module type deduplication in Proto output

### Runtime

- **RUNTIME-01**: XPath / `must` / `when` expression evaluation at runtime — requires full XPath engine
- **RUNTIME-02**: Streaming/lazy compilation for large schemas (100k+ nodes)

## Out of Scope

| Feature | Reason |
|---------|--------|
| Security hardening (path traversal, identifier sanitization) | Not blocking v1; risk is low in trusted-input library context; deferred |
| JSON Schema / OpenAPI generator | New output format is a separate milestone; pluggable interface already exists |
| Runtime XPath evaluation | Requires a full XPath engine; significant scope; separate milestone |
| Mobile/web tooling | Not applicable to a Go library |
| Graphical schema browser | Out of scope for a library project |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| COMP-01 | Phase 1 | Complete |
| COMP-02 | Phase 1 | Complete |
| COMP-03 | Phase 1 | Complete |
| COMP-04 | Phase 1 | Complete |
| COMP-05 | Phase 1 | Complete |
| COMP-06 | Phase 1 | Complete |
| COMP-07 | Phase 1 | Complete |
| TEST-01 | Phase 2 | Complete |
| TEST-02 | Phase 2 | Complete |
| TEST-03 | Phase 2 | Complete |
| GOGEN-01 | Phase 3 | Pending |
| GOGEN-02 | Phase 3 | Pending |
| GOGEN-03 | Phase 3 | Pending |
| GOGEN-04 | Phase 3 | Pending |
| PBGEN-01 | Phase 4 | Pending |
| PBGEN-02 | Phase 4 | Pending |
| PBGEN-03 | Phase 4 | Pending |
| PBGEN-04 | Phase 4 | Pending |
| API-01 | Phase 5 | Pending |
| API-02 | Phase 5 | Pending |
| API-03 | Phase 5 | Pending |
| API-04 | Phase 5 | Pending |

**Coverage:**
- v1 requirements: 22 total
- Mapped to phases: 22
- Unmapped: 0 ✓

---
*Requirements defined: 2026-03-14*
*Last updated: 2026-03-14 after roadmap creation*
