# Phase 5: Public API Stabilization - Context

**Gathered:** 2026-03-15
**Status:** Ready for planning

<domain>
## Phase Boundary

Clean up `gotya.go` so callers can depend on it without anticipating a breaking change before v1 is tagged. This phase covers API-01 through API-04 only — no new generator features, no new public functions beyond what's needed to stabilize what exists.

</domain>

<decisions>
## Implementation Decisions

### ParseError type shape (API-01, API-02)
- Define `type ParseError struct` in `gotya.go` (not in an internal package) with an `Errors []Diagnostic` field
- `type Diagnostic struct` has `File string`, `Line int`, `Column int`, `Message string` fields
- `ParseError` implements the `error` interface — `Error()` returns a human-readable summary (e.g., "3 parse errors: line 4: unexpected token ...")
- `Parse()` and `ParseFile()` return `*ParseError` (not `os.ErrInvalid`) when parsing fails
- `ParseFile()` fills `Diagnostic.File` from the path argument; `Parse()` leaves `File` as empty string (Claude's discretion on exact behavior)
- Callers use `errors.As(err, &parseErr)` to access the full `Errors []Diagnostic` slice programmatically

### Internal type leakage (API-03)
- Define `type Module struct` in `gotya.go` as an opaque wrapper around `*schema.Module`
- `Compile()` returns `([]*Module, error)` — callers never need to import the `schema` package
- How generator callers access `*schema.Module` from `*gotya.Module` is Claude's discretion (e.g., an unexported accessor or a package-level helper)
- `ParseFile()` return type changed from `(*ast.Module, error)` to `(*ASTModule, error)` for consistency with `Parse()`

### ASTModule type (API-03)
- Change `type ASTModule = ast.Module` (alias) to an opaque defined type: `type ASTModule struct { mod *ast.Module }`
- Expose only `Name() string` on `ASTModule` — no other ast.Module methods visible to callers
- `Compile()` takes `[]*ASTModule` as input (consistent with the new opaque type)

### Comment and doc style (API-04)
- Replace all `// Why:` style comments with standard godoc format: function comments begin with the function name (e.g., `// Parse parses a YANG module from source text...`)
- Remove all inline `//` comments from `gotya.go` — only godoc comments on exported symbols remain
- Remove all placeholder language: "for facade demonstration", "Simplistic error", `// NOTE:` blocks, `// TODO`

### Claude's Discretion
- Exact wording of godoc comments
- Whether `gotya.Module` exposes a `Schema() *schema.Module` accessor or uses another mechanism for generator interop
- File field behavior when `Parse()` is called without a filename

</decisions>

<specifics>
## Specific Ideas

- Current `Parse()` comment says "Simplistic error for facade demonstration" — this is the primary placeholder to remove
- The `// NOTE: We could gather parser errors here` comment is also a placeholder — remove and implement actual error collection
- Standard godoc rendered on pkg.go.dev: function comments start with `// FunctionName ...` — this is the target style

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets
- `gotya.go`: Current public facade — 74 lines. `Parse()`, `ParseFile()`, `Compile()` are the three exported functions; `ASTModule` is the one exported type
- `parser/parser.go`: Returns `*ast.Module`; has `p.errors []string` — check if it already accumulates errors or only captures first
- `compiler/compiler.go`: Has `addError()` and `MaxErrors` (COMP-07) — error accumulation is already in place for compiler errors; parser error accumulation may need similar work

### Established Patterns
- `errors.Is()` / `errors.As()` pattern established in Phase 2 (TEST-03) for compiler error sentinels
- No new dependencies — stdlib-only patterns preferred (prior decision from pre-phase)
- `compiler.Options` struct pattern (struct with fields, nil = defaults) — same pattern could apply to any future gotya-level options

### Integration Points
- `parser.New(l).ParseModule()` returns `*ast.Module` — `Parse()` wraps this; needs to expose parser diagnostics
- `ASTModule` is passed to `Compile()` — making it opaque requires updating the `ast.Module` extraction inside `Compile()`
- Generator packages (`generator/golang`, `generator/protobuf`) use `*schema.Module` directly — the `gotya.Module` wrapper must not break these; Claude decides the interop mechanism

</code_context>

<deferred>
## Deferred Ideas

- None — discussion stayed within phase scope

</deferred>

---

*Phase: 05-public-api-stabilization*
*Context gathered: 2026-03-15*
