# Phase 5: Public API Stabilization - Research

**Researched:** 2026-03-15
**Domain:** Go public API design, error types, opaque wrappers, godoc conventions
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**ParseError type shape (API-01, API-02)**
- Define `type ParseError struct` in `gotya.go` (not in an internal package) with an `Errors []Diagnostic` field
- `type Diagnostic struct` has `File string`, `Line int`, `Column int`, `Message string` fields
- `ParseError` implements the `error` interface — `Error()` returns a human-readable summary (e.g., "3 parse errors: line 4: unexpected token ...")
- `Parse()` and `ParseFile()` return `*ParseError` (not `os.ErrInvalid`) when parsing fails
- `ParseFile()` fills `Diagnostic.File` from the path argument; `Parse()` leaves `File` as empty string (Claude's discretion on exact behavior)
- Callers use `errors.As(err, &parseErr)` to access the full `Errors []Diagnostic` slice programmatically

**Internal type leakage (API-03)**
- Define `type Module struct` in `gotya.go` as an opaque wrapper around `*schema.Module`
- `Compile()` returns `([]*Module, error)` — callers never need to import the `schema` package
- How generator callers access `*schema.Module` from `*gotya.Module` is Claude's discretion (e.g., an unexported accessor or a package-level helper)
- `ParseFile()` return type changed from `(*ast.Module, error)` to `(*ASTModule, error)` for consistency with `Parse()`

**ASTModule type (API-03)**
- Change `type ASTModule = ast.Module` (alias) to an opaque defined type: `type ASTModule struct { mod *ast.Module }`
- Expose only `Name() string` on `ASTModule` — no other ast.Module methods visible to callers
- `Compile()` takes `[]*ASTModule` as input (consistent with the new opaque type)

**Comment and doc style (API-04)**
- Replace all `// Why:` style comments with standard godoc format: function comments begin with the function name (e.g., `// Parse parses a YANG module from source text...`)
- Remove all inline `//` comments from `gotya.go` — only godoc comments on exported symbols remain
- Remove all placeholder language: "for facade demonstration", "Simplistic error", `// NOTE:` blocks, `// TODO`

### Claude's Discretion
- Exact wording of godoc comments
- Whether `gotya.Module` exposes a `Schema() *schema.Module` accessor or uses another mechanism for generator interop
- File field behavior when `Parse()` is called without a filename

### Deferred Ideas (OUT OF SCOPE)
- None — discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| API-01 | `gotya.Parse()` returns a domain-specific error type (`*gotya.ParseError` or equivalent) — not `os.ErrInvalid`; error includes file, line, column, and message fields | Parser already emits `[]string` with line numbers embedded in message text; those strings need to be parsed into structured `Diagnostic` fields. The token.Position struct (Line, Column int) is the authoritative source for positional data — parser must capture it per error instead of embedding in a string. |
| API-02 | Parser diagnostics (all parse errors, not just the first) are propagated through `gotya.Parse()` — callers can access the full error list | Parser already accumulates into `p.errors []string` (does not stop at first error); `p.Errors()` is already exported. What's missing: (1) converting strings to `[]Diagnostic`, (2) `Parse()` checking `p.Errors()` even when `ParseModule()` returns non-nil. |
| API-03 | All exported symbols in `gotya.go` are intentional public API — no internal package types leak through function signatures; documented as stable | Three leaks to fix: (1) `type ASTModule = ast.Module` alias exposes `ast.Module` methods; (2) `ParseFile()` returns `*ast.Module` directly; (3) `Compile()` takes `[]*ASTModule` (currently `[]*ast.Module` alias) and returns `[]*schema.Module`. Also: `compiler.Options` appears in `Compile()` signature — must decide if this stays or is wrapped. |
| API-04 | All placeholder, demonstration, and TODO comments removed from `gotya.go` — public API file documents actual behavior only | Three placeholder items confirmed: (1) `// Simplistic error for facade demonstration` comment + `os.ErrInvalid` return; (2) `// NOTE: We could gather parser errors here...` comment block; (3) `// Why:` godoc style throughout the file. |
</phase_requirements>

## Summary

Phase 5 is a surgical rewrite of a single 74-line file (`gotya.go`) plus a small parser enhancement. The codebase is mature: all parsing, compilation, and generation machinery works correctly — this phase is purely about the public contract that callers see.

There are four concrete problems to fix. First, `Parse()` returns `os.ErrInvalid` (a generic sentinel) instead of a structured error with file/line/column information. Second, even though the parser accumulates multiple errors internally in `p.errors []string`, none of those errors escape through the public API — `Parse()` only checks whether `ParseModule()` returned nil. Third, `gotya.go` exports `ast.Module` and `schema.Module` types through function signatures, meaning callers must import those internal packages to write type assertions. Fourth, the file contains placeholder/facade comments that misrepresent the actual behavior.

**Primary recommendation:** Implement the four changes sequentially — `ParseError`/`Diagnostic` types, parser error plumbing, opaque `ASTModule` and `Module` wrappers, then godoc cleanup. Write a `gotya_test.go` file at the same time as it drives the correct API surface.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| stdlib `errors` | Go 1.25 | `errors.As` pattern for typed error unwrapping | Already used in compiler; no new dependency |
| stdlib `fmt` | Go 1.25 | `fmt.Errorf` for error wrapping in `Error()` | Already imported in gotya.go |
| stdlib `strings` | Go 1.25 | String building for `Error()` summary | May be needed for multi-line summary |

### No New Dependencies
This phase requires zero new imports beyond what is already in go.mod. The only change to imports in `gotya.go` is dropping the `ast`, `schema`, and `os` imports once the opaque wrappers are in place.

## Architecture Patterns

### Recommended Project Structure
```
gotya.go           # All changes live here (types + functions)
gotya_test.go      # New file: public API black-box tests
parser/parser.go   # Minor change: addError must capture token.Position
```

### Pattern 1: Structured Error Type with errors.As

**What:** Define `ParseError` in `gotya.go` (not in a sub-package) so it appears in the caller's import path as `gotya.ParseError`.

**When to use:** Any time a library needs to provide structured diagnostic data to callers without requiring them to import internal packages.

**Example:**
```go
// Source: stdlib errors package conventions, Go blog "Working with Errors in Go 1.13"
type Diagnostic struct {
    File    string
    Line    int
    Column  int
    Message string
}

type ParseError struct {
    Errors []Diagnostic
}

func (e *ParseError) Error() string {
    if len(e.Errors) == 1 {
        return fmt.Sprintf("parse error: %s", e.Errors[0].Message)
    }
    return fmt.Sprintf("%d parse errors: %s", len(e.Errors), e.Errors[0].Message)
}

// Caller code:
var parseErr *gotya.ParseError
if errors.As(err, &parseErr) {
    for _, d := range parseErr.Errors {
        fmt.Printf("%s:%d:%d: %s\n", d.File, d.Line, d.Column, d.Message)
    }
}
```

### Pattern 2: Opaque Wrapper Type

**What:** Replace a type alias (`type ASTModule = ast.Module`) with a defined type that wraps the internal type and exposes only the subset of methods that form the stable API.

**When to use:** When an internal type has many methods but the public API should expose only a few, or when you want the ability to change the internal representation without breaking callers.

**Example:**
```go
// Source: Go spec — defined type vs. alias
// type ASTModule = ast.Module  <-- OLD: alias leaks all ast.Module methods
type ASTModule struct {
    mod *ast.Module
}

// Name returns the module's name (the YANG module argument).
func (m *ASTModule) Name() string {
    return m.mod.Argument()
}
```

### Pattern 3: Unexported Field Accessor for Generator Interop

**What:** The `gotya.Module` wrapper holds `*schema.Module` in an unexported field. Generator packages (in the same repository but different packages) need to unwrap it. The cleanest approach is a package-level function in `gotya.go` that is NOT exported to external callers but is accessible to internal packages — however Go does not support "internal-only exported" without the `internal/` directory trick.

**Recommended approach:** Expose `Schema() *schema.Module` as a public method on `*gotya.Module`. This is honest — generators legitimately need the schema. The stable public API contract is on the top-level functions, not on the internals of `*schema.Module`. Document `Schema()` clearly so callers know it returns an internal type.

**Alternative:** Move generator invocation entirely behind the `gotya` package (add `GenerateGo`, `GenerateProto` functions to `gotya.go`). This is heavier and out of scope for this phase.

**Recommended for this phase:** Add `Schema() *schema.Module` on `*gotya.Module` — it is a clean, honest accessor that does not require restructuring the generators.

```go
type Module struct {
    schema *schema.Module
}

// Schema returns the compiled schema.Module for use with generator packages.
func (m *Module) Schema() *schema.Module {
    return m.schema
}
```

### Pattern 4: Parser Error Plumbing with Positional Data

**What:** The parser's `addError(msg string)` currently embeds line number in the message string. To produce structured `Diagnostic` values, the parser must capture position separately.

**Current state:** `parser.Parser.errors []string` — strings with embedded line numbers.

**Required change:** Change `Parser.errors` to `[]parserError` (unexported struct with position + message) or change the `Errors()` return type.

**Recommended approach:** Keep `errors []string` as-is (backward compatible for parser unit tests), but add a second slice `diagnostics []parserDiagnostic` that captures position data. Then export a `Diagnostics()` method alongside `Errors()`. Or: change `errors []string` to store structured data and regenerate the `Errors() []string` output from them.

**Simpler approach (preferred):** Change `Parser.errors` to store a struct, update `Errors() []string` to reformat from the struct, and add `Diagnostics() []ParserDiagnostic` (unexported type if kept internal, or returned as a new exported type from the parser package). Then `gotya.Parse()` calls `p.Diagnostics()` to build `[]Diagnostic` for the `ParseError`.

**Key insight:** `p.curToken.Pos` is available at every `addError` call site — the fix is minimal. Each `addError` call passes `p.curToken.Pos.Line` and `p.curToken.Pos.Column` in addition to the message string.

### Pattern 5: `compiler.Options` Leakage

**What:** `Compile(astModules []*ASTModule, opts *compiler.Options)` exposes `compiler.Options` as a parameter type. This forces callers to import `github.com/GabrielNunesIT/gotya/compiler`.

**Decision needed (Claude's discretion):** The CONTEXT.md decisions do not explicitly address `compiler.Options`. Options:
1. Define `type Options struct` in `gotya.go` with the same fields and translate to `compiler.Options` internally.
2. Leave `compiler.Options` as-is and accept that callers who need options must import the `compiler` package.
3. Accept nil-means-defaults (current pattern already works for most callers).

**Recommendation:** Option 1 — define a thin `gotya.CompileOptions` (or just use the existing nil-means-defaults path and document it). Since the `CONTEXT.md` does not lock this in either direction, a `type CompileOptions struct` mirror in `gotya.go` is the cleanest approach and completes the API encapsulation.

### Anti-Patterns to Avoid

- **Changing parser.Errors() return type from []string:** Existing tests in `parser_test.go` call `assert.Empty(t, p.Errors())` — that test must keep passing. Add new method; don't break old one.
- **Returning `*ParseError` when there are no errors:** `Parse()` must return `nil, nil` on success — not `&ParseError{Errors: nil}`.
- **Importing `schema` in gotya_test.go:** The test for API-03 is specifically that callers can use the public API without importing `schema` or `ast`. The test file must not import those packages.
- **Making `gotya.Module` embed `*schema.Module`:** Embedding exposes all `schema.Module` fields publicly (it's a struct, not an interface). Use a field: `schema *schema.Module`.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Error summary formatting | Custom string builder | `fmt.Sprintf` with `len()` check | Trivial, stdlib |
| Error wrapping / unwrapping | Custom Is/As methods | Standard `errors.As` | Go 1.13+ built-in; already used in compiler |
| Type-checking errors | `err == os.ErrInvalid` | `errors.As(err, &parseErr)` | Type-safe, gives structured data |

**Key insight:** The `errors.As` / `errors.Is` pattern is already established in this codebase (Phase 2, compiler package). Phase 5 applies the same pattern at the public API layer.

## Common Pitfalls

### Pitfall 1: ParseModule returns non-nil with errors
**What goes wrong:** The parser's `ParseModule()` can return a non-nil `*ast.Module` AND have errors in `p.errors` simultaneously. The current `gotya.Parse()` only checks `astMod == nil` and silently discards `p.errors` when parsing partially succeeds.
**Why it happens:** The parser has error-recovery (`recoverStatement`) that allows parsing to continue after errors. A syntactically valid outer structure can be returned even when interior statements failed.
**How to avoid:** Always check `len(p.Errors()) > 0` after `ParseModule()`, regardless of whether `astMod` is nil. If there are errors, return a `*ParseError` instead of the AST.
**Warning signs:** Tests that pass malformed YANG and only assert on the error value — they pass even if errors are swallowed.

### Pitfall 2: Line/column data not available in parser errors
**What goes wrong:** Current `parser.addError(msg string)` receives only a message string. Position data (from `p.curToken.Pos`) is sometimes embedded in the string ("at line 4") but is not available as a structured `int`.
**Why it happens:** The parser was designed for diagnostic printing, not programmatic error consumption.
**How to avoid:** The parser must be updated so each `addError` call captures `p.curToken.Pos.Line` and `p.curToken.Pos.Column` alongside the message. This data is available at every call site.
**Warning signs:** `Diagnostic.Line` is 0 for all errors — means position wasn't captured.

### Pitfall 3: ASTModule alias exposes all ast.Module methods
**What goes wrong:** `type ASTModule = ast.Module` is a type _alias_, not a defined type. All methods on `ast.Module` (including `BaseNode` embedded methods like `TokenLiteral()`, `SubStatements()`, etc.) are immediately visible to callers of the `gotya` package.
**Why it happens:** Aliases are transparent by design; they provide no encapsulation.
**How to avoid:** Change to `type ASTModule struct { mod *ast.Module }` (defined type). Only add methods that form the intentional public contract.
**Warning signs:** Callers importing `gotya` can call `mod.SubStatements()` without importing `ast` — that's the leak.

### Pitfall 4: Compile() signature still uses []*ASTModule after the alias change
**What goes wrong:** When `ASTModule` was an alias for `*ast.Module`, the compiler received `[]*ast.Module`. After making it an opaque struct, the compiler still needs `*ast.Module` values internally.
**Why it happens:** The change must thread through: `Compile()` in `gotya.go` must extract `m.mod` from each `*ASTModule` before passing to `compiler.New().Compile()`.
**How to avoid:** In `Compile()`, iterate the `[]*ASTModule` input and call `astMod.mod` (the unexported field, accessible because `Compile()` is in the same package as `ASTModule`).

### Pitfall 5: ParseFile return type
**What goes wrong:** Current `ParseFile` returns `(*ast.Module, error)`, not `(*ASTModule, error)`. After making `ASTModule` opaque, `ParseFile` must wrap the result.
**Why it happens:** The original code shared the `Parse()` implementation by calling it, but `Parse()` now returns `*ASTModule`.
**How to avoid:** Update `ParseFile` to call `Parse()` and return its result directly (since `Parse()` will return `*ASTModule`). The `File` field in `Diagnostic` must be filled before returning — this means `ParseFile` needs its own parse call or a helper that accepts a filename parameter.

### Pitfall 6: Generator packages break after Compile() return type changes
**What goes wrong:** Changing `Compile()` to return `[]*gotya.Module` means existing generator call sites that pass `[]*schema.Module` to `GenerateDevice` will no longer compile.
**Why it happens:** `GenerateDevice` in both generators takes `[]*schema.Module` — if callers go through `gotya.Compile()`, they now get `[]*gotya.Module` back.
**How to avoid:** The `Schema() *schema.Module` accessor on `*gotya.Module` lets callers do: `mods[i].Schema()`. Update any affected code in `cmd/gotya` or test files that call `Compile()` and then pass results to generators.

## Code Examples

Verified patterns from existing codebase:

### How parser errors are currently collected
```go
// Source: parser/parser.go
func (p *Parser) addError(msg string) {
    p.errors = append(p.errors, msg)
}

// Errors returns any parsing errors encountered.
func (p *Parser) Errors() []string {
    return p.errors
}
```

### Token position is available at every addError call site
```go
// Source: parser/parser.go — example call site
p.addError(fmt.Sprintf("expected keyword at line %d, got %s", p.curToken.Pos.Line, p.curToken.Type))
// p.curToken.Pos.Column is also available: p.curToken.Pos.Column
```

### token.Position struct (authoritative source)
```go
// Source: token/token.go
type Position struct {
    Line   int
    Column int
}
```

### How compiler uses errors.As pattern (established precedent)
```go
// Source: compiler/compiler.go
var (
    ErrCircularTypedef = errors.New("circular typedef")
    // ...
)
func (c *Compiler) addError(sentinel error, msg string) {
    c.errors = append(c.errors, fmt.Errorf("%w: %s", sentinel, msg))
}
```

### Current placeholder code to remove
```go
// Source: gotya.go lines 28-33
// NOTE: We could gather parser errors here and return them.
// For now, if the module is nil, it failed completely.
if astMod == nil {
    return nil, os.ErrInvalid // Simplistic error for facade demonstration
}
```

### Current ASTModule alias (to replace with defined type)
```go
// Source: gotya.go line 18
type ASTModule = ast.Module   // alias — exposes all ast.Module methods
```

### How cmd/gotya loader uses the public API (potential callsite to audit)
```go
// Source: cmd/gotya/ — uses compiler.Options and schema.Module directly
// After this phase, cmd/gotya callers may need to call .Schema() on *gotya.Module
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `os.ErrInvalid` as parse error | `*gotya.ParseError` with `[]Diagnostic` | This phase | Callers get structured diagnostics |
| Type alias `ASTModule = ast.Module` | Opaque `type ASTModule struct` | This phase | Encapsulates ast package |
| `Compile()` returns `[]*schema.Module` | `Compile()` returns `[]*gotya.Module` | This phase | Encapsulates schema package |
| Inline `// Why:` comments | Godoc-style `// FunctionName ...` comments | This phase | Standard pkg.go.dev rendering |

**Deprecated/outdated after this phase:**
- `os.ErrInvalid` in parse path: replaced by `*ParseError`
- Direct use of `ast` and `schema` imports in `gotya.go`: replaced by opaque types
- `// NOTE:` and `// Why:` comment style: replaced by standard godoc

## Open Questions

1. **Does `compiler.Options` stay in `Compile()` signature?**
   - What we know: CONTEXT.md decisions address `ASTModule` and `Module` wrappers but do not mention `compiler.Options`
   - What's unclear: Whether callers need to pass `compiler.Options` or whether nil-means-defaults is sufficient for v1
   - Recommendation: Add `type CompileOptions = compiler.Options` alias in `gotya.go` — this lets callers import only `gotya`, but the planner may choose to address this or leave it as-is

2. **How does `ParseFile` fill `Diagnostic.File` without duplicating parse logic?**
   - What we know: `ParseFile` currently calls `Parse(string(content))` and returns the result
   - What's unclear: After the refactor, `Parse()` returns `*ASTModule` (success) or `*ParseError` (failure); the `File` field in `ParseError.Errors[].File` must come from the caller
   - Recommendation: `ParseFile` calls a shared internal `parseWithFile(content, filename string)` helper, or post-processes the returned `*ParseError` to fill `File` before returning

3. **What happens to existing tests that test the parser directly?**
   - What we know: `parser_test.go` calls `p.Errors()` and asserts it returns `[]string`
   - What's unclear: If `Errors()` signature changes, those tests break
   - Recommendation: Keep `Errors() []string` intact; add a new `Diagnostics()` method that returns structured data consumed by `gotya.Parse()`

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | testify v1.11.1 (assert + require packages) |
| Config file | none — standard `go test ./...` |
| Quick run command | `go test ./... -run TestAPI -count=1` |
| Full suite command | `go test ./... -count=1` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| API-01 | `Parse()` on syntactically invalid YANG returns `*ParseError` with file/line/col/message | unit | `go test . -run TestParse_Error -count=1` | ❌ Wave 0 |
| API-01 | `errors.As(err, &parseErr)` succeeds on `Parse()` error | unit | `go test . -run TestParse_ErrorsAs -count=1` | ❌ Wave 0 |
| API-02 | `Parse()` on multi-error YANG returns all diagnostics, not just first | unit | `go test . -run TestParse_MultipleErrors -count=1` | ❌ Wave 0 |
| API-03 | `gotya_test.go` imports only `gotya` (no `ast`, `schema`) | compile-time | `go build ./...` | ❌ Wave 0 |
| API-03 | `Compile()` returns `[]*gotya.Module` (not `[]*schema.Module`) | unit | `go test . -run TestCompile_OpaqueReturn -count=1` | ❌ Wave 0 |
| API-03 | `ASTModule.Name()` returns module name, no other AST methods visible | unit | `go test . -run TestASTModule_Name -count=1` | ❌ Wave 0 |
| API-04 | `gotya.go` contains no `// TODO`, `// NOTE:`, `// Why:`, or "demonstration" strings | static | `grep -n 'TODO\|NOTE:\|Why:\|demonstration' gotya.go` returns empty | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test . -count=1` (public API package only)
- **Per wave merge:** `go test ./... -count=1`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `gotya_test.go` — covers API-01, API-02, API-03 (black-box tests, package `gotya_test`)
- [ ] No framework install needed — testify already in go.mod

## Sources

### Primary (HIGH confidence)
- Direct code inspection: `/home/user/repos/gotya/gotya.go` — current API surface
- Direct code inspection: `/home/user/repos/gotya/parser/parser.go` — `errors []string`, `p.Errors()`, `p.curToken.Pos`
- Direct code inspection: `/home/user/repos/gotya/token/token.go` — `Position{Line, Column int}`
- Direct code inspection: `/home/user/repos/gotya/ast/ast.go` — `ast.Module` methods
- Direct code inspection: `/home/user/repos/gotya/schema/schema.go` — `schema.Module` fields
- Direct code inspection: `/home/user/repos/gotya/generator/generator.go` — `GenerateDevice([]*schema.Module, io.Writer)`
- Direct code inspection: `/home/user/repos/gotya/compiler/compiler.go` — sentinel errors pattern, `Options` struct
- Go specification: defined types vs. type aliases (https://go.dev/ref/spec#Type_definitions)

### Secondary (MEDIUM confidence)
- Go blog "Working with Errors in Go 1.13": `errors.As` canonical pattern
- Go documentation godoc conventions: https://go.dev/blog/godoc

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies; all patterns verified in existing codebase
- Architecture: HIGH — changes are scoped to one file + minor parser extension; all touch points inspected
- Pitfalls: HIGH — identified by direct code inspection of the four affected files

**Research date:** 2026-03-15
**Valid until:** 2026-04-15 (stable domain; code is the source of truth)