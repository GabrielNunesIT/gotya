# Phase 2: Testing Infrastructure - Research

**Researched:** 2026-03-14
**Domain:** Go testing patterns — corpus smoke tests, go/format.Source() validation, typed error sentinels
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **No golden files this phase** — skip golden file tests for now; focus on corpus smoke test and error sentinels
- **Corpus test**: convert `test/generate.go` (`//go:build ignore`) into a proper go test that asserts no panic and valid Go output for each file in the corpus
- **Error sentinels**: migrate `compiler/compiler_test.go` string assertions (`assert.Contains(t, err.Error(), ...)`) to `errors.Is` / `errors.As`
- **go/format.Source()**: integrate into Go generator output path — invalid generated Go surfaces at generation time

### Claude's Discretion

- Exact corpus test structure (single test with subtests vs. table-driven)
- How failures are reported for corpus test (fail fast vs. collect all failures)
- Exact sentinel type design (var vs. type, granularity per condition)
- Internal plumbing for `-update` flag detection (deferred to Phase 3 / 4)

### Deferred Ideas (OUT OF SCOPE)

- Golden file tests for Go generator — Phase 3
- Golden file tests for Proto generator — Phase 4
- Coverage enforcement (no coverage threshold was requested)
- Benchmark tests for compiler performance
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| TEST-01 | Corpus smoke test runs all YANG files in `test/assets/yangs/` through the full parse→compile→generate pipeline and asserts no panic and syntactically valid Go output | Covered by: corpus test pattern (§Architecture Patterns), fileLoader reuse (§Code Examples), go/format.Source() (§Standard Stack) |
| TEST-02 | Go generator validates output with `go/format.Source()` before writing — invalid generated Go surfaces at generation time, not at user compile time | Covered by: go/format.Source() integration pattern (§Architecture Patterns), wrap-and-forward io.Writer approach (§Code Examples) |
| TEST-03 | Compiler error conditions use typed error sentinels (not string-matched messages) so tests assert on structured values, not fragile substrings | Covered by: sentinel design (§Architecture Patterns), migration map (§Code Examples), errors.Is/As patterns (§Standard Stack) |
</phase_requirements>

---

## Summary

Phase 2 adds testing infrastructure with no new runtime features. Three tasks map directly to three requirements: (1) convert the `//go:build ignore` corpus driver into a proper `go test` test, (2) integrate `go/format.Source()` into the Go generator output path so bad generated code is caught immediately, and (3) introduce exported error sentinel vars in the `compiler` package and migrate the ~20 `assert.Contains(t, err.Error(), ...)` assertions in `compiler_test.go` to `errors.Is` / `errors.As`.

All three tasks are mechanical with no ambiguity about the right approach. The patterns are established Go stdlib idioms — no new dependencies are needed. The codebase already has the raw material: `test/generate.go` contains the `fileLoader` and the corpus iteration loop that just needs to be converted to a `_test.go` file; `generator/golang/generator.go` writes into a `bytes.Buffer` internally before forwarding to the final `io.Writer`, which is the natural injection point for `go/format.Source()`; and the compiler's `addError()` calls all use plain string messages that can be replaced by sentinel-based wrapping.

Key codebase discovery: `generator/golang/device_test.go` already contains a `TestGenerateDeviceFromTestAssets` test that partially addresses TEST-01 — but it uses `fmt.Printf` for per-module logging, does not call `go/format.Source()` on output, and its `fileLoader` is a local duplicate of the one in `test/generate.go`. TEST-01 wants the canonical corpus test to live in `test/` (converting `generate.go`), not in `generator/golang/`. The two can coexist: the device_test.go test stays as a generator-level integration test; the new `test/corpus_test.go` is the official regression gate.

**Primary recommendation:** Three independent tasks in sequence — sentinels first (compiler package, no external impact), then go/format.Source() (generator package, no external impact), then corpus test (test package, depends on the other two being stable).

---

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `go/format` (stdlib) | Go 1.25.5 | Validates and formats Go source bytes | Zero-cost syntax validation; same formatter as `gofmt`; authoritative check for generated code correctness |
| `testing` (stdlib) | Go 1.25.5 | Test harness, subtests, t.Run, t.Helper | Already in use; no alternative |
| `errors` (stdlib) | Go 1.25.5 | `errors.Is`, `errors.As`, `errors.New`, `fmt.Errorf("%w", ...)` | Standard error wrapping chain; no alternative |
| `github.com/stretchr/testify/assert` | v1.11.1 | Assertion library | Already in use across all test files |
| `github.com/stretchr/testify/require` | v1.11.1 | Fatal assertions (stops test on failure) | Already imported in `device_test.go`; use `require.NoError` for setup steps that must succeed |

### No New Dependencies

This phase adds zero new `go.mod` dependencies. All patterns use stdlib only.

**Existing go.mod:**
```
module github.com/gotya/gotya
go 1.25.5
require github.com/stretchr/testify v1.11.1
```

---

## Architecture Patterns

### Recommended Project Structure (additions this phase)

```
compiler/
├── compiler.go           # add exported Err* sentinel vars + wrap in addError
├── compiler_test.go      # migrate assert.Contains -> errors.Is/errors.As
test/
├── assets/yangs/         # 205 YANG corpus files (unchanged)
├── generate.go           # keep as-is (//go:build ignore, go:generate target)
├── test.go               # keep as-is
└── corpus_test.go        # NEW: proper go test corpus smoke test
generator/golang/
├── generator.go          # add go/format.Source() call in GenerateDevice
└── device_test.go        # keep as-is (already tests corpus through generator)
```

### Pattern 1: Typed Error Sentinels

**What:** Export named error vars from the `compiler` package. Wrap them into error messages using `fmt.Errorf("%w", ErrXxx)`. Tests use `errors.Is(err, compiler.ErrXxx)` instead of `strings.Contains(err.Error(), "some substring")`.

**When to use:** Any compiler error condition where a test needs to assert on the specific error kind, not the message text.

**Sentinel var design (recommended — keep it coarse-grained):**

Coarse sentinels reduce the number of exports and are easier to keep stable. Fine-grained sentinels (one per message variant) are overkill for compiler diagnostics.

```go
// Source: compiler/compiler.go — add after package declaration
// Exported error sentinels for structured error assertions.
var (
    ErrCircularTypedef  = errors.New("circular typedef")
    ErrCircularUses     = errors.New("circular dependency in uses")
    ErrAugmentNotFound  = errors.New("augment target not found")
    ErrDuplicateIdent   = errors.New("duplicate identifier")
    ErrListMissingKey   = errors.New("list missing key")
    ErrConfigBoundary   = errors.New("config boundary violation")
    ErrInvalidDefault   = errors.New("invalid default value")
    ErrMandatoryDefault = errors.New("mandatory leaf with default")
    ErrTypeRestriction  = errors.New("invalid type restriction")
    ErrListKeyInvalid   = errors.New("list key invalid")
    ErrIdentityrefBase  = errors.New("invalid identityref base")
    ErrXPathSyntax      = errors.New("xpath syntax error")
    ErrMaxErrors        = errors.New("compilation stopped: too many errors")
)
```

**addError wrapping — how to inject the sentinel:**

The compiler accumulates errors as plain strings in `[]string`. The returned error is built by joining them with `\n`. To support `errors.Is`, the compiler's final returned error must be a type that implements `Is()` or each individual error must be wrapped. The practical approach is:

Option A (recommended): Keep `[]string` accumulation but return a custom type that implements `Unwrap() []error` (Go 1.20+ multi-error join). Each call to `addError` that corresponds to a sentinel also stores a parallel `[]error` slice. Final returned error is `errors.Join(...)` of all structured errors plus the full message string.

Option B (simpler): Return the first structured error via `fmt.Errorf("%w: %s", ErrXxx, detail)` when the compiler has exactly one error class. Does not work for multi-error accumulation.

Option C (pragmatic for this phase): Each `addError` call wraps the sentinel using `fmt.Errorf("%w: %s", ErrXxx, msg)`. Store both in `[]error` (not `[]string`). Final join is `errors.Join(errs...)`. Tests call `errors.Is(err, compiler.ErrCircularTypedef)` — this works because `errors.Join` result implements `Unwrap() []error` and `errors.Is` recursively searches.

**Recommendation: Option C.** Change `c.errors []string` to `c.errors []error`. Change `addError(msg string)` to `addError(sentinel error, msg string)` or keep the string form and wrap internally. The returned error from `Compile()` uses `errors.Join(c.errors...)`. Tests migrate from:

```go
assert.Contains(t, err.Error(), "circular typedef")
```

to:

```go
assert.ErrorIs(t, err, compiler.ErrCircularTypedef)
```

### Pattern 2: go/format.Source() in Generator

**What:** The Go generator writes to a `bytes.Buffer` before writing to the caller's `io.Writer`. After all generation is done, call `go/format.Source(buf.Bytes())` before forwarding. If the result is invalid Go syntax, return a generation error — never write malformed output.

**When to use:** In `GenerateDevice` (and optionally `Generate`), at the end of generation, before writing to the external `io.Writer`.

**Example:**

```go
// Source: go/format stdlib — pkg.go.dev/go/format
import "go/format"

func (g *GoGenerator) GenerateDevice(modules []*schema.Module, w io.Writer) error {
    var buf bytes.Buffer
    // ... all existing generation writes to buf ...

    formatted, err := format.Source(buf.Bytes())
    if err != nil {
        return fmt.Errorf("generated Go is not valid syntax: %w", err)
    }
    _, err = w.Write(formatted)
    return err
}
```

**Key detail:** The generator already uses `bytes.Buffer` internally (confirmed in generator.go — `var buf bytes.Buffer` exists before the final `w.Write`). The injection point is just before `w.Write(buf.Bytes())`. No architectural change needed.

**Benefit of using format.Source vs format.Node:** `format.Source()` takes `[]byte`, is simpler, and also applies gofmt formatting. The output is canonical. `format.Node` requires an AST — unnecessary complexity.

### Pattern 3: Corpus Smoke Test

**What:** A `_test.go` file in the `test/` package that iterates all `.yang` files in `test/assets/yangs/`, runs each through the full pipeline, and asserts:
1. No panic (Go's test framework catches panics and converts them to test failures)
2. `go/format.Source()` succeeds on the generated output (syntactically valid Go)

**Subtest structure (recommended):** `t.Run(modName, ...)` for each module. This gives a clear per-module pass/fail line in `go test -v` output. Failures do not stop other modules (collect all failures, not fail-fast), which is the correct behavior for a regression gate.

**Reporting strategy:** Collect all failures. A corpus regression gate is most useful when it shows ALL broken modules, not just the first. Use `t.Errorf` (not `t.Fatalf`) inside subtests so all modules are attempted.

**Example structure:**

```go
// Source: standard Go table-driven test with t.Run subtests
// File: test/corpus_test.go
package test

import (
    "bytes"
    "go/format"
    "os"
    "path/filepath"
    "strings"
    "testing"

    "github.com/gotya/gotya/ast"
    "github.com/gotya/gotya/compiler"
    golanggenerator "github.com/gotya/gotya/generator/golang"
    "github.com/gotya/gotya/parser"
    "github.com/gotya/gotya/parser/lexer"
    "github.com/gotya/gotya/schema"
)

func TestCorpus(t *testing.T) {
    yangsDir := filepath.Join("assets", "yangs")
    loader := &corpusLoader{
        dir:         yangsDir,
        astCache:    make(map[string]*ast.Module),
        schemaCache: make(map[string]*schema.Module),
    }

    files, err := os.ReadDir(yangsDir)
    if err != nil {
        t.Fatalf("read corpus dir: %v", err)
    }

    var modules []*schema.Module
    for _, f := range files {
        // ... populate modules, skip submodules ...
    }

    gen := golanggenerator.New(&golanggenerator.Options{
        PackageName: "corpus",
        RootName:    "Device",
    })

    for _, mod := range modules {
        mod := mod // capture
        t.Run(mod.Name, func(t *testing.T) {
            t.Parallel()
            var buf bytes.Buffer
            if err := gen.Generate(mod, &buf); err != nil {
                t.Errorf("generation failed: %v", err)
                return
            }
            if _, err := format.Source(buf.Bytes()); err != nil {
                t.Errorf("generated Go is not valid syntax: %v", err)
            }
        })
    }
}
```

**fileLoader reuse:** `test/generate.go` has a `fileLoader` implementation with `LoadAST` and `Load`. The corpus test must have its own copy (or the loader can be extracted to a shared `test/loader_test.go` file). Since `generate.go` is `//go:build ignore`, it cannot be imported. The simplest approach: copy the loader into `corpus_test.go` (it is ~70 lines) and name it `corpusLoader`. This is a pragmatic duplication; DRY concerns are secondary to keeping the corpus test self-contained.

**Alternatively**, since `device_test.go` in `generator/golang/` already has a `testLoader` with the same logic, the corpus test in `test/` can have its own copy. Three copies of the same 70-line pattern is acceptable given the constraint that `generate.go` can't be imported.

**Compile errors in corpus:** The corpus contains real-world YANG files that may produce compiler errors (e.g., missing imports, unresolved augments against modules not in the corpus). The corpus test should NOT fail on compiler errors — only on panics and invalid Go syntax. Log compiler errors with `t.Logf` so they are visible in `-v` output without failing the test. This matches the behavior of `generate.go` today (`log.Printf("Module %s compiled with errors: %v", ...)`) and the existing `device_test.go` (`fmt.Printf("Module %s compiled with errors: %v\n", ...)`).

**Package name for corpus_test.go:** `package test` (internal test package style, matching `test.go`). This avoids needing an external package import for the `fileLoader` type.

### Anti-Patterns to Avoid

- **Don't use `assert.Contains(t, err.Error(), "substring")` for new tests** — this is the pattern being eliminated; new error assertions must use `errors.Is` / `errors.As`
- **Don't return a bare `errors.New("some message")` from addError** — sentinels must be declared once as package-level vars; inline `errors.New` in addError calls creates unequal errors that `errors.Is` cannot match
- **Don't use `t.Fatalf` in corpus subtest loop** — use `t.Errorf` to collect all failures; `t.Fatalf` would stop the subtest but it's already inside `t.Run` so the difference is minimal, but `Errorf` is the idiomatic choice for non-setup assertions
- **Don't call `go/format.Source()` on partial output** — only call it on the complete final buffer, never on individual nodes or fragments; partial Go source will always fail the syntax check
- **Don't validate with `go/parser.ParseFile`** — `go/format.Source()` is strictly better: it validates AND formats in one call; no need for two calls

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Syntax validation of generated Go | Custom token scanner / string heuristics | `go/format.Source()` | The only authoritative check; handles all edge cases including import blocks, generics, and multi-line strings |
| Error wrapping for multi-error join | Custom `MultiError` struct | `errors.Join` (Go 1.20+) | stdlib; implements `Unwrap() []error`; `errors.Is` traversal is automatic |
| Error identity comparison | String matching on `.Error()` | `errors.Is` / `errors.As` | Sentinel vars are equality-comparable; wrapping chain is traversed automatically |
| Test assertion helper for error sentinels | Custom `assertErrorIs` function | `assert.ErrorIs(t, err, sentinel)` | testify v1.11.1 includes `assert.ErrorIs`; no custom helper needed |

**Key insight:** The Go standard library covers all three problems in this phase. No library research was needed beyond confirming what is already in stdlib.

---

## Common Pitfalls

### Pitfall 1: errors.Join Requires Go 1.20+

**What goes wrong:** `errors.Join` was added in Go 1.20. If the go.mod `go` directive is less than 1.20, `errors.Join` is unavailable.

**Why it happens:** Legacy go.mod files with old minimum versions.

**How to avoid:** The project's go.mod already declares `go 1.25.5` — this is well past 1.20. `errors.Join` is fully available. No issue.

**Warning signs:** Compiler error `errors.Join undefined`.

### Pitfall 2: errors.Is Does Not Work With fmt.Errorf Without %w

**What goes wrong:** `fmt.Errorf("circular typedef: %s", name)` — without `%w` — creates an error that does NOT wrap the sentinel. `errors.Is(err, ErrCircularTypedef)` returns `false`.

**Why it happens:** `%w` is required to create a wrapping relationship. `%s` converts the sentinel to its string representation but discards the identity.

**How to avoid:** Always use `fmt.Errorf("%w: %s", ErrCircularTypedef, detail)` when the sentinel must be detectable. When `addError` wraps: `c.errors = append(c.errors, fmt.Errorf("%w: %s", sentinel, msg))`.

**Warning signs:** Tests pass with `assert.Contains` but fail with `assert.ErrorIs` after migration.

### Pitfall 3: go/format.Source() Fails on Incomplete Package

**What goes wrong:** `Generate()` (single-module, no package header) writes `type Foo struct {}` without a package declaration. `go/format.Source()` requires a complete Go source file starting with `package`.

**Why it happens:** `format.Source` parses the bytes as a complete Go source file. Missing package clause is a parse error.

**How to avoid:** Only apply `format.Source()` to the output of `GenerateDevice()` (which writes the `package` header) and to `Generate()` (which also writes `// Code generated...\npackage %s\n\n`). Both entry points include the package header, so both are safe to validate. Confirm by reading the first few lines of each function — verified: both write `package <name>` as the first non-comment line.

**Warning signs:** `format.Source()` returns `expected 'package', found '...'` errors on valid generation runs.

### Pitfall 4: Corpus Test Flakiness From t.Parallel() + Shared Loader

**What goes wrong:** The `corpusLoader` caches AST and schema by module name. If multiple subtests running in parallel all call `loader.Load()` concurrently, the cache map has a data race.

**Why it happens:** Go's `map` is not goroutine-safe. Race detector will catch this.

**How to avoid:** Two options:
1. Build the modules slice serially before the parallel `t.Run` loop (recommended: sequential load, parallel assertion)
2. Protect the loader with a `sync.Mutex`

Option 1 is simpler and correct: load all modules sequentially into a `[]*schema.Module` slice, then iterate the slice with `t.Run` + `t.Parallel()` to run the generation+format.Source assertions in parallel. The generation step uses independent `bytes.Buffer` per subtest and does not touch the loader.

**Warning signs:** `go test -race` reports a data race in `corpusLoader.astCache` or `schemaCache`.

### Pitfall 5: Sentinel Granularity vs. Stability

**What goes wrong:** Exporting a separate sentinel for every distinct error message (e.g., `ErrListKeyNotFound`, `ErrListKeyNotLeaf`, `ErrListNoKey`) creates a large exported API surface that is hard to evolve. Each sentinel is effectively a public API commitment under semver.

**Why it happens:** Over-engineering sentinels to match the old one-sentinel-per-assert-Contains pattern.

**How to avoid:** Group by error category, not by message. `ErrListMissingKey` covers "list must have at least one key", "key not found in list", and "key must be a leaf" — all are list key problems. Tests that previously distinguished by substring can use `errors.Is(err, ErrListMissingKey)` without further specificity. If a caller needs message text for display, they get it from `err.Error()`.

---

## Code Examples

### Sentinel Declaration and Wrapping

```go
// Source: compiler/compiler.go — package-level sentinel vars
var (
    ErrCircularTypedef  = errors.New("circular typedef")
    ErrCircularUses     = errors.New("circular dependency in uses")
    ErrAugmentNotFound  = errors.New("augment target not found")
    ErrDuplicateIdent   = errors.New("duplicate identifier")
    ErrListMissingKey   = errors.New("list key error")
    ErrConfigBoundary   = errors.New("config boundary violation")
    ErrInvalidDefault   = errors.New("invalid default value")
    ErrMandatoryDefault = errors.New("mandatory leaf with default")
    ErrTypeRestriction  = errors.New("invalid type restriction")
    ErrIdentityrefBase  = errors.New("invalid identityref base")
    ErrXPathSyntax      = errors.New("xpath syntax error")
    ErrMaxErrors        = errors.New("compilation stopped: too many errors")
)

// addError updated signature — wraps sentinel into stored error
func (c *Compiler) addError(sentinel error, msg string) {
    if len(c.errors) > 0 && errors.Is(c.errors[len(c.errors)-1], ErrMaxErrors) {
        return
    }
    c.errors = append(c.errors, fmt.Errorf("%w: %s", sentinel, msg))
    if len(c.errors) >= c.maxErrors {
        c.errors = append(c.errors, fmt.Errorf("%w (limit %d)", ErrMaxErrors, c.maxErrors))
    }
}

// Compile return — join errors
func (c *Compiler) buildError() error {
    if len(c.errors) == 0 {
        return nil
    }
    return errors.Join(c.errors...)
}
```

### Migration: assert.Contains -> assert.ErrorIs

```go
// Before (string matching — fragile)
assert.Contains(t, err.Error(), "circular typedef")

// After (sentinel matching — typed and stable)
assert.ErrorIs(t, err, compiler.ErrCircularTypedef)

// For table-driven tests with errorMsg field:
// Before:
errorMsg: "invalid identityref base 'unknown-base' in id: identity not found locally"
// ...
assert.Contains(t, err.Error(), tt.errorMsg)

// After:
wantSentinel: compiler.ErrIdentityrefBase
// ...
assert.ErrorIs(t, err, tt.wantSentinel)
```

### go/format.Source() Integration

```go
// Source: go/format stdlib — pkg.go.dev/go/format#Source
import "go/format"

// In GenerateDevice, replace the final w.Write with:
formatted, err := format.Source(buf.Bytes())
if err != nil {
    // err contains the syntax error with line/column information
    return fmt.Errorf("generated Go has invalid syntax: %w", err)
}
if _, err := w.Write(formatted); err != nil {
    return fmt.Errorf("write formatted output: %w", err)
}
return nil
```

### Corpus Smoke Test — collect-all-failures pattern

```go
// Source: standard Go subtest pattern
// test/corpus_test.go — package test

// Load all modules sequentially (avoids concurrent map access)
var modules []namedModule
for _, f := range files {
    // ... skip dirs, non-.yang, submodules ...
    schemaMod, err := loader.Load(modName)
    if err != nil {
        t.Logf("module %s compiled with errors (not a test failure): %v", modName, err)
    }
    if schemaMod != nil {
        modules = append(modules, namedModule{name: modName, mod: schemaMod})
    }
}

if len(modules) == 0 {
    t.Fatal("corpus loaded zero modules — corpus directory may be missing or empty")
}

gen := golanggenerator.New(&golanggenerator.Options{
    PackageName: "corpus",
    RootName:    "Device",
})

// Run assertions in parallel subtests
for _, nm := range modules {
    nm := nm
    t.Run(nm.name, func(t *testing.T) {
        t.Parallel()
        var buf bytes.Buffer
        if genErr := gen.Generate(nm.mod, &buf); genErr != nil {
            t.Errorf("Generate(%s): %v", nm.name, genErr)
            return
        }
        if _, fmtErr := format.Source(buf.Bytes()); fmtErr != nil {
            t.Errorf("Generate(%s) produced invalid Go syntax: %v", nm.name, fmtErr)
        }
    })
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `assert.Contains(t, err.Error(), "...")` | `assert.ErrorIs(t, err, sentinel)` | Go 1.13 + errors package | Tests are no longer coupled to error message wording; refactoring messages without breaking tests |
| Manual error joining with `strings.Join` | `errors.Join(errs...)` | Go 1.20 | Multi-error wrapping with `errors.Is` traversal works automatically |
| `gofmt` as external process to validate | `go/format.Source()` in-process | Go 1.0+ (always available) | No subprocess required; synchronous validation in the generator output path |

**Deprecated/outdated:**
- `ioutil.ReadFile`: replaced by `os.ReadFile` since Go 1.16 — `generate.go` uses `os.ReadFile` correctly; no issue
- `// +build ignore` build tag: replaced by `//go:build ignore` since Go 1.17 — `generate.go` has both forms; the new `//go:build` form is authoritative

---

## Open Questions

1. **Should `Generate()` (single-module path) also call `go/format.Source()`?**
   - What we know: TEST-02 requires it on the "Go generator output path." `GenerateDevice` is the main production path. `Generate` is the per-module path used in tests.
   - What's unclear: Whether TEST-02 intends to cover `Generate()` as well, or only `GenerateDevice()`.
   - Recommendation: Apply it to `GenerateDevice()` only (the production entry point). The test `Generate()` path can be left without it for now; it is tested via corpus subtests that call `format.Source` explicitly. This avoids double-formatting in the device_test.go flow.

2. **Should `addError` signature change to `addError(sentinel error, msg string)` or keep `addError(msg string)` and match sentinel by string internally?**
   - What we know: There are ~25 call sites in `compiler.go` for `addError`. Changing the signature touches all of them. Matching by string internally is brittle and replicates the exact anti-pattern being eliminated.
   - What's unclear: Whether changing 25 call sites is acceptable scope for one plan.
   - Recommendation: Change the signature. The 25 call sites are mechanical changes. The alternative (internal string matching) is worse than the problem it avoids.

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go standard `testing` + testify v1.11.1 |
| Config file | none — standard `go test` invocation |
| Quick run command | `go test ./compiler/... ./generator/golang/... ./test/...` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| TEST-01 | All corpus YANG files run through pipeline without panic, output passes `format.Source()` | integration | `go test ./test/... -run TestCorpus -v` | No — Wave 0 |
| TEST-02 | `GenerateDevice` returns error when output is not valid Go syntax | unit | `go test ./generator/golang/... -run TestGoGenerator_FormatValidation` | No — Wave 0 |
| TEST-03 | Compiler error conditions return typed sentinels matchable by `errors.Is` | unit | `go test ./compiler/... -run TestCompiler_` | Yes (compiler_test.go exists; needs migration) |

### Sampling Rate

- **Per task commit:** `go test ./compiler/... ./generator/golang/... ./test/...`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `test/corpus_test.go` — covers TEST-01; needs `corpusLoader` implementation and `TestCorpus` function
- [ ] `generator/golang/generator_test.go` — add `TestGoGenerator_FormatValidation` asserting that generation error is returned for a module that would produce syntactically invalid Go (covers TEST-02)
- [ ] Framework install: none — `testing` and `go/format` are stdlib; testify already in go.mod

---

## Sources

### Primary (HIGH confidence)

- `go/format` stdlib docs — `format.Source()` signature, behavior, error format: https://pkg.go.dev/go/format#Source
- `errors` stdlib docs — `errors.Join`, `errors.Is`, `errors.New`, `%w` wrapping: https://pkg.go.dev/errors
- Direct codebase audit: `compiler/compiler.go`, `compiler/compiler_test.go`, `generator/golang/generator.go`, `generator/golang/device_test.go`, `test/generate.go` — all read in full (2026-03-14)
- `go.mod` — confirmed Go 1.25.5, testify v1.11.1, no other deps
- `test/assets/yangs/` — confirmed 205 YANG corpus files

### Secondary (MEDIUM confidence)

- Phase 1 research SUMMARY.md — confirms `go/format.Source()` is the right pattern; confirms no new dependencies; confirms testify v1.11.1 is in use
- CONTEXT.md for this phase — locked decisions used directly to scope research

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — go/format, errors, testing are stdlib; testify already in use; zero new dependencies
- Architecture: HIGH — all patterns derived from direct reading of the actual source files in the repo; no guesswork
- Pitfalls: HIGH — derived from direct code reading (race condition on shared loader, %w requirement for errors.Is, format.Source requiring complete package)

**Research date:** 2026-03-14
**Valid until:** 2026-09-14 (stable stdlib patterns; no fast-moving dependencies)