# Phase 6: v1.0 Gap Closure - Research

**Researched:** 2026-03-15
**Domain:** Go error handling, testify assertion migration, multi-module error accumulation
**Confidence:** HIGH

## Summary

Phase 6 closes two v1.0 audit gaps in a Go library (go 1.25.5, testify v1.11.1). Both gaps are surgical: no new types, no new dependencies, no architectural changes.

TEST-03 migrates 21 `assert.Contains(t, err.Error(), ...)` calls in `compiler/compiler_test.go` to `assert.ErrorIs(t, err, compiler.ErrXxx)` assertions. All 12 sentinel variables already exist in `compiler/compiler.go`. The mapping from string-checked messages to sentinels is fully determined from reading the source. One subtlety: the `Validator` struct in `compiler/validator.go` routes its errors through a single `c.addError(ErrInvalidDefault, err.Error())` call, meaning identityref, default-value, and XPath errors currently appear under `ErrInvalidDefault`. Migrating those tests requires the Validator to call `c.addError` with the correct per-category sentinel directly — a pattern already used in Phase 2 (see STATE.md accumulated decisions).

API-02 changes `gotya.Compile()` in `gotya.go` (lines 131-138) from early-return on first module error to error accumulation across all modules, joined with `errors.Join(errs...)` at the end. A `TestCompile_MultiModuleErrors` test in `gotya_test.go` proves accumulation.

**Primary recommendation:** Implement TEST-03 first (validator refactoring + test migration), then API-02 (loop restructure + new test). Each is a standalone, independently verifiable change.

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**API-02: Aggregate error type**
- Use `errors.Join(errs...)` to combine per-module errors — stdlib, no new types, `errors.Is` works against any sentinel in the joined chain
- Keep one shared `compiler.Compiler` instance across all modules (preserve existing cross-module state sharing)
- Keep per-module error wrapping: `fmt.Errorf("compile module %s: %w", name, err)` — callers can tell which module failed
- Return `nil, errors.Join(errs...)` when any module fails — clean contract, no partial results alongside errors

**API-02: Test coverage**
- Add a `TestCompile_MultiModuleErrors` test in `gotya_test.go` (package `gotya_test`)
- Pass 2 modules each with a distinct sentinel error; assert `errors.Is` for both sentinels in the joined result
- Proves the accumulation behavior doesn't silently regress

**TEST-03: Scope of migration**
- Only migrate `assert.Contains(t, err.Error(), ...)` calls — non-error `assert.Contains` (e.g., `assert.Contains(t, schemaMod.Nodes, ...)`) are unrelated and stay unchanged
- 21 target call sites in `compiler/compiler_test.go`

**TEST-03: Table-driven test**
- The identityref table-driven test (line ~939) uses `tt.errorMsg string` — replace with `tt.sentinel error`
- All 3 table cases map to `compiler.ErrIdentityrefBase`
- Assertion becomes `assert.ErrorIs(t, err, tt.sentinel)`

**TEST-03: Multi-error cases**
- Some tests assert 3+ errors from a single compile call (e.g., `ErrTypeRestriction` x3, `ErrListMissingKey` x2)
- Check each expected sentinel individually with `assert.ErrorIs` — verifies accumulation behavior
- For repeated same-sentinel cases, one `assert.ErrorIs` check is sufficient (sentinel matches on first hit)

### Claude's Discretion
- Exact order of error collection and loop structure in `Compile()` — only constraint is `errors.Join` at end
- How to handle the `assert.Contains` calls that check for both sentinels and specific field values (line ~348: 3 type restriction calls) — planner decides whether to assert once per unique sentinel or once per call site

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope.
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| TEST-03 | Compiler error conditions use typed error sentinels (not string-matched messages) so tests assert on structured values, not fragile substrings | All 21 target call sites identified and mapped to sentinels; Validator refactoring path identified |
| API-02 | Parser diagnostics (all parse errors, not just the first) are propagated through `gotya.Compile()` — callers can access the full error list | Current early-return loop identified in `gotya.go` lines 131-138; `errors.Join` stdlib pattern confirmed; test design documented |
</phase_requirements>

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `errors` (stdlib) | Go 1.25.5 | `errors.Is`, `errors.Join`, sentinel vars | Zero-cost, universally traverses wrapped chains |
| `fmt` (stdlib) | Go 1.25.5 | `fmt.Errorf("%w: %s", sentinel, msg)` | Already the wrapping pattern in `addError()` |
| `github.com/stretchr/testify` | v1.11.1 | `assert.ErrorIs`, `assert.Error`, `assert.NoError` | Already the project test assertion library |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `testing` (stdlib) | Go 1.25.5 | `t.Helper()`, `t.Parallel()`, `t.Run()` | All test functions |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `errors.Join` | custom aggregate type | `errors.Join` is stdlib since Go 1.20; already present in `compiler.go` line 269; no new type needed |
| `assert.ErrorIs` | `require.ErrorIs` | `assert` continues test after failure; `require` halts; `assert` preferred for multi-error tests that want to report all failures |

**Installation:** No new dependencies required.

## Architecture Patterns

### Recommended Project Structure
No structural changes. Phase touches three files only:
```
compiler/
├── compiler_test.go   # TEST-03: 21 assert.Contains -> assert.ErrorIs migrations
├── validator.go       # TEST-03: call c.addError with correct sentinel per category
gotya.go               # API-02: Compile() loop restructure
gotya_test.go          # API-02: TestCompile_MultiModuleErrors
```

### Pattern 1: Sentinel Error with errors.Join

**What:** Multiple errors accumulated in a slice, joined at the end so callers can `errors.Is` against any sentinel in the chain.

**When to use:** Any function that accumulates errors across iterations.

**Example (existing pattern in `compiler/compiler.go` line 269):**
```go
// Source: compiler/compiler.go lines 268-271
if len(c.errors) > 0 {
    return mod, fmt.Errorf("compilation failed with %d errors:\n%w", len(c.errors), errors.Join(c.errors...))
}
```

And each error in the slice was added via:
```go
// Source: compiler/compiler.go lines 83-91
func (c *Compiler) addError(sentinel error, msg string) {
    // ...
    c.errors = append(c.errors, fmt.Errorf("%w: %s", sentinel, msg))
    // ...
}
```

`errors.Is` traverses the `errors.Join` tree and finds any matching sentinel.

### Pattern 2: Validator Calls addError Directly

**What:** Instead of returning an aggregate string error from `Validator.Validate()` and wrapping it under a single sentinel, each validation method calls `c.addError(correctSentinel, msg)` directly.

**When to use:** When validation categories need distinct sentinels.

**Current (broken for TEST-03):**
```go
// compiler/compiler.go line 253-255
validator := NewValidator(c, mod)
if err := validator.Validate(); err != nil {
    c.addError(ErrInvalidDefault, err.Error())  // loses sentinel granularity
}
```

**Required pattern (established in Phase 2):**
```go
// From STATE.md: "Validator refactored to call c.addError(sentinel, msg) directly"
// validator.go methods call c.compiler.addError(sentinel, msg) per category
// No single aggregate wrapper — each error carries the right sentinel
```

### Pattern 3: Error Accumulation Loop (API-02)

**What:** Collect errors per module, join at end instead of early return.

**Current (early-return in `gotya.go` lines 131-138):**
```go
for _, m := range astModules {
    schemaMod, err := comp.Compile(m.mod)
    if err != nil {
        return nil, fmt.Errorf("compile module %s: %w", m.mod.Argument(), err)
    }
    // ...
}
```

**Required pattern:**
```go
var errs []error
for _, m := range astModules {
    schemaMod, err := comp.Compile(m.mod)
    if err != nil {
        errs = append(errs, fmt.Errorf("compile module %s: %w", m.mod.Argument(), err))
        continue
    }
    // append to compiled only on success
}
if len(errs) > 0 {
    return nil, errors.Join(errs...)
}
return compiled, nil
```

### Anti-Patterns to Avoid

- **Mixing partial results with errors:** When any module fails, return `nil, errors.Join(errs...)` — not `compiled, errors.Join(errs...)`. Locked decision.
- **Single `errors.Is` for multi-error tests:** `errors.Is` stops at first match. When the same sentinel appears 3 times (e.g., ErrTypeRestriction), one `assert.ErrorIs` is sufficient — it confirms the sentinel is present in the chain.
- **Asserting message substrings as a fallback:** After migration, no `assert.Contains(t, err.Error(), ...)` should remain for error checks. The entire point of TEST-03 is structural — string checks are inherently fragile.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Joining multiple errors | Custom aggregate error type | `errors.Join` (stdlib Go 1.20+) | Traversable by `errors.Is`/`errors.As`; already in use in `compiler.go` |
| Sentinel error traversal | Manual string scanning | `errors.Is` | Handles wrapped chains; `errors.Join` tree is fully traversable |

**Key insight:** `errors.Join` creates a tree of wrapped errors. `errors.Is` performs a depth-first traversal of this tree. This means `errors.Is(joinedErr, ErrListMissingKey)` will find `ErrListMissingKey` even if it appears deep in the chain, wrapped as `fmt.Errorf("%w: list key ...", ErrListMissingKey)`.

## Common Pitfalls

### Pitfall 1: Validator Errors Routed Through Wrong Sentinel

**What goes wrong:** Tests for identityref, default values, and XPath syntax use `assert.Contains(t, err.Error(), tt.errorMsg)` / `assert.Contains(t, err.Error(), "invalid default ...")` etc. If only the test assertions are changed to `assert.ErrorIs(t, err, ErrIdentityrefBase)` without also fixing the Validator, all these tests will FAIL because the Validator currently wraps everything under `ErrInvalidDefault`.

**Why it happens:** `compiler.go` line 254: `c.addError(ErrInvalidDefault, err.Error())` — a single call wraps ALL validator errors under one sentinel regardless of category.

**How to avoid:** The Validator methods (`checkNodeIdentities`, `validateDefaultValues`, `validateXPathSyntax`, `validateConfigBoundaries`, `validateListKeys`, `validateMandatoryAndDefault`) must each call `c.compiler.addError(correctSentinel, msg)` directly instead of appending to `v.errors`. The `Validate()` method then removes its string-joining return value.

**Warning signs:** `assert.ErrorIs(t, err, ErrIdentityrefBase)` returns false on an error that was produced by identityref validation.

### Pitfall 2: assert.Contains on Non-Error Context Accidentally Removed

**What goes wrong:** The file has many `assert.Contains` calls — not just the 21 error ones. For example:
- `assert.Contains(t, schemaMod.Nodes, "interfaces")` (line 65)
- `assert.Contains(t, cont.GetChildren(), "id")` (line 132)

These are NOT migration targets. Only `assert.Contains(t, err.Error(), ...)` calls need changing.

**How to avoid:** Filter precisely: grep for `assert.Contains(t, err.Error()` — not just `assert.Contains`.

**Warning signs:** Compilation errors after migration ("type schema.Node is not a string" etc.) or test failures asserting map membership.

### Pitfall 3: Table-Driven Test Struct Requires Field Rename

**What goes wrong:** `TestCompiler_IdentityrefValidation` uses `tt.errorMsg string` in the struct. Simply changing `assert.Contains(t, err.Error(), tt.errorMsg)` to `assert.ErrorIs(t, err, tt.errorMsg)` is a type mismatch (string is not error).

**How to avoid:** Rename the struct field from `errorMsg string` to `sentinel error`. Update all 3 table entries to use `compiler.ErrIdentityrefBase` instead of the string message. Update the assertion to `assert.ErrorIs(t, err, tt.sentinel)`.

**Warning signs:** Compilation error: `cannot use tt.errorMsg (type string) as type error`.

### Pitfall 4: errors.Join Import Missing in gotya.go

**What goes wrong:** `gotya.go` currently imports `fmt` and `errors` but may not use `errors.Join` yet. Adding the accumulation loop requires verifying `errors` is imported.

**How to avoid:** Check the current import block. `errors` is already imported in `gotya.go` (line 5 not present — actually it's NOT in the current imports). The file uses `fmt` but not `errors` directly. `errors.Join` will require adding `"errors"` to the import block.

**Warning signs:** Compilation error: `errors.Join undefined`.

### Pitfall 5: One Compiler Instance — State Accumulation Across Modules

**What goes wrong:** `gotya.Compile()` creates a single `comp := compiler.New(compOpts)` and reuses it across all modules (locked decision). The compiler has stateful fields like `c.errors []error`. After module A fails, `c.errors` is non-empty when module B is compiled. The compiler's `Compile()` method appends to `c.errors` but only clears module-specific state (groupings, typedefs, imports, augments) at the start of each call — it does NOT clear `c.errors`.

**How to avoid:** The per-module `err` returned by `comp.Compile(m.mod)` already encapsulates that module's error count. The `gotya.Compile()` accumulation collects these per-module errors into `errs []error`, wraps each with the module name. The compiler's internal `c.errors` slice growing across modules is expected — it won't cause duplicate reporting since each `comp.Compile()` returns a fresh per-module error based on errors accumulated since last call.

**Warning signs:** Tests showing the same error reported multiple times (once per module).

## Code Examples

### TEST-03: Standard Single-Sentinel Migration

```go
// BEFORE (fragile string match):
assert.Contains(t, err.Error(), "list invalid-list must have at least one key")

// AFTER (structural sentinel check):
assert.ErrorIs(t, err, compiler.ErrListMissingKey)
```

### TEST-03: Multi-Sentinel Migration (multiple errors from one compile)

```go
// BEFORE:
assert.Contains(t, err.Error(), "type string for node invalid-string cannot have range")
assert.Contains(t, err.Error(), "type int32 for node invalid-int cannot have length")
assert.Contains(t, err.Error(), "type int32 for node invalid-int cannot have pattern")

// AFTER (one ErrorIs per unique sentinel is sufficient — errors.Is finds first match):
assert.ErrorIs(t, err, compiler.ErrTypeRestriction)
// Optional: keep additional ErrorIs calls to document that MULTIPLE errors expected
// But for same sentinel repeated N times, one assert.ErrorIs is definitive.
```

### TEST-03: Table-Driven Struct Migration

```go
// BEFORE:
tests := []struct {
    name        string
    input       string
    expectError bool
    errorMsg    string
}{
    {name: "Missing Local Base", ..., errorMsg: "invalid identityref base 'unknown-base'..."},
    // ...
}
// In loop:
assert.Contains(t, err.Error(), tt.errorMsg)

// AFTER:
tests := []struct {
    name        string
    input       string
    expectError bool
    sentinel    error
}{
    {name: "Missing Local Base", ..., sentinel: compiler.ErrIdentityrefBase},
    {name: "No Base Statement", ..., sentinel: compiler.ErrIdentityrefBase},
    {name: "Unknown Prefix",    ..., sentinel: compiler.ErrIdentityrefBase},
}
// In loop:
assert.ErrorIs(t, err, tt.sentinel)
```

### API-02: Compile() Accumulation Loop

```go
// gotya.go — replace lines 131-138
func Compile(astModules []*ASTModule, opts *CompileOptions) ([]*Module, error) {
    compOpts := &compiler.Options{}
    if opts != nil {
        compOpts.MaxErrors = opts.MaxErrors
    }
    comp := compiler.New(compOpts)

    var compiled []*Module
    var errs []error
    for _, m := range astModules {
        schemaMod, err := comp.Compile(m.mod)
        if err != nil {
            errs = append(errs, fmt.Errorf("compile module %s: %w", m.mod.Argument(), err))
            continue
        }
        if schemaMod != nil {
            compiled = append(compiled, &Module{schema: schemaMod})
        }
    }
    if len(errs) > 0 {
        return nil, errors.Join(errs...)
    }
    return compiled, nil
}
```

### API-02: TestCompile_MultiModuleErrors

```go
// gotya_test.go
func TestCompile_MultiModuleErrors(t *testing.T) {
    // Module A: missing key on config list -> ErrListMissingKey
    modA, err := gotya.Parse(`
        module mod-a {
            namespace "urn:a"; prefix "a";
            list bad { leaf x { type string; } }
        }`)
    require.NoError(t, err)

    // Module B: type range restriction on string -> ErrTypeRestriction
    modB, err := gotya.Parse(`
        module mod-b {
            namespace "urn:b"; prefix "b";
            leaf y { type string { range "1..10"; } }
        }`)
    require.NoError(t, err)

    _, compileErr := gotya.Compile([]*gotya.ASTModule{modA, modB}, nil)
    require.Error(t, compileErr)

    // Both modules' sentinels must be findable in the joined error
    // Note: sentinels from compiler package are not directly accessible in gotya_test
    // Use errors.As on *gotya.ParseError pattern, OR verify error string contains both
    // module names. The test in gotya_test cannot import compiler directly.
    require.Contains(t, compileErr.Error(), "mod-a")
    require.Contains(t, compileErr.Error(), "mod-b")
}
```

**IMPORTANT NOTE on TestCompile_MultiModuleErrors:** The `gotya_test` package (external) cannot import `compiler` directly. The locked decision says "assert `errors.Is` for both sentinels" — this is achievable only if `compiler.ErrXxx` sentinels are re-exported from the `gotya` package, OR if the test imports `compiler` explicitly. Looking at the existing `gotya_test.go`, it imports only `github.com/gotya/gotya`. The CONTEXT.md says to assert `errors.Is` for both sentinels. The planner must decide: either import `compiler` in the test (valid since `compiler` is an internal package of the same module), or use a different approach. The test file already has `"errors"` in its imports.

## Complete 21-Site Migration Map

| Line | Current String | Sentinel | Notes |
|------|---------------|----------|-------|
| 210 | `"list invalid-list must have at least one key"` | `ErrListMissingKey` | Direct c.validate() path |
| 240 | `"duplicate identifier 'id'"` | `ErrDuplicateIdent` | Direct c.parseChildren() path |
| 269 | `"has 'config true' but its parent base has 'config false'"` | `ErrConfigBoundary` | Direct c.validate() path |
| 347 | `"type string for node invalid-string cannot have range"` | `ErrTypeRestriction` | Direct c.validateType() path |
| 348 | `"type int32 for node invalid-int cannot have length"` | `ErrTypeRestriction` | Same sentinel as 347 |
| 349 | `"type int32 for node invalid-int cannot have pattern"` | `ErrTypeRestriction` | Same sentinel as 347, 348 |
| 380 | `"key 'not-here' not found in list 'bad-list-nonexistent'"` | `ErrListMissingKey` | Direct c.validate() path |
| 381 | `"key 'wrong-type' in list 'bad-list-wrong-type' must be a leaf"` | `ErrListMissingKey` | Same sentinel as 380 |
| 414 | `"circular dependency detected in uses"` | `ErrCircularUses` | Direct c.resolveUses() path |
| 441 | `"leaf 'invalid-leaf' is mandatory and cannot have a default value"` | `ErrMandatoryDefault` | Direct c.validate() path |
| 939 | `tt.errorMsg` (table-driven, 3 cases) | `ErrIdentityrefBase` | **Validator path — requires Validator fix** |
| 1139 | `"invalid default 'True' for boolean leaf bad-bool"` | `ErrInvalidDefault` | **Validator path — requires Validator fix** |
| 1140 | `"invalid default '12.5' for integer leaf bad-int"` | `ErrInvalidDefault` | **Validator path — requires Validator fix** |
| 1141 | `"invalid default '-5' for unsigned integer leaf bad-uint"` | `ErrInvalidDefault` | **Validator path — requires Validator fix** |
| 1142 | `"invalid default 'green' for enum leaf bad-enum"` | `ErrInvalidDefault` | **Validator path — requires Validator fix** |
| 1179 | `"mismatched quotes in when expression"` | `ErrXPathSyntax` | **Validator path — requires Validator fix** |
| 1180 | `"mismatched brackets or parentheses in must expression"` | `ErrXPathSyntax` | **Validator path — requires Validator fix** |
| 1181 | `"mismatched brackets or parentheses in path expression"` | `ErrXPathSyntax` | **Validator path — requires Validator fix** |
| 1227 | `"circular typedef"` | `ErrCircularTypedef` | Direct c.getType() path |
| 1242 | `"augment target not found"` | `ErrAugmentNotFound` | Direct augment loop path |
| 1324 | `"compilation stopped"` | `ErrMaxErrors` | Direct addError() path |

**Rows 939, 1139-1142, 1179-1181:** These 8 sites require Validator refactoring. The Validator currently collects `v.errors []string` and returns a combined string that `compiler.go` wraps as `ErrInvalidDefault`. After refactoring, the Validator must call `c.compiler.addError(sentinel, msg)` directly per category, removing the string-based return path for these categories.

**Rows 210, 240, 269, 347-349, 380-381, 414, 441, 1227, 1242, 1324:** These 13 sites route through `c.addError()` directly and already carry the correct sentinel. Only the test assertions need changing — no production code changes required for these.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| String-matched error assertions | Sentinel-based `errors.Is` | Go 1.13 (wrapped errors), Go 1.20 (errors.Join) | Refactoring-safe; messages can change without breaking tests |
| Early-return on first module error | Accumulate all module errors | API-02 (Phase 6) | Callers see full failure picture |

**Deprecated/outdated:**
- `assert.Contains(t, err.Error(), substring)` for error type verification: replaced by `assert.ErrorIs` — substring checks are fragile, break on message rewording

## Open Questions

1. **TestCompile_MultiModuleErrors: can gotya_test import compiler?**
   - What we know: `gotya_test.go` uses `package gotya_test` (external); it currently imports only `github.com/gotya/gotya`. The `compiler` package is within the same module (`github.com/gotya/gotya/compiler`).
   - What's unclear: CONTEXT.md says "assert `errors.Is` for both sentinels" — this requires access to `compiler.ErrXxx` constants from the test, which means either importing `compiler` in `gotya_test.go` or re-exporting the sentinels via `gotya` package.
   - Recommendation: Import `compiler` directly in `gotya_test.go` for the new test. It is a same-module package. The existing precedent (`compiler_test.go` imports `compiler`) shows this is the project pattern. The planner should clarify import structure for the new test.

2. **Validator refactoring: does removing v.errors break anything?**
   - What we know: `Validator.Validate()` currently accumulates `v.errors []string` and returns a combined `fmt.Errorf` string. After refactoring, the Validator calls `c.compiler.addError()` directly, making `Validate()` return nil always (or not at all).
   - What's unclear: Whether any existing tests call `Validator` directly (they don't — all tests go through `compile()` helper or `compiler.New(nil).Compile()`).
   - Recommendation: Refactor is safe. Change `Validate()` to call `c.addError()` for each category, remove `v.errors`, have `Validate()` return nothing (or always nil). The `compiler.go` call site at line 253 becomes `validator.Validate()` with no error check.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | testify v1.11.1 (assert + require) |
| Config file | none (standard `go test`) |
| Quick run command | `go test ./compiler/... -run TestCompiler_ -v` |
| Full suite command | `go test ./...` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| TEST-03 | All error assertions in compiler_test.go use assert.ErrorIs against sentinel | unit | `go test ./compiler/... -run TestCompiler_ -v` | Yes (migrated) |
| API-02 | Compile() with multi-module input returns errors from ALL failing modules | unit | `go test ./... -run TestCompile_MultiModuleErrors -v` | No — Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./compiler/... -run TestCompiler_ -count=1`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `gotya_test.go` needs `TestCompile_MultiModuleErrors` function added — covers API-02
- [ ] `compiler/validator.go` needs per-category `c.addError()` calls — precondition for TEST-03 Validator sites

*(Existing test infrastructure covers the framework; only new test function and production code refactoring are needed)*

## Sources

### Primary (HIGH confidence)
- `/home/user/repos/gotya/compiler/compiler.go` — All 12 sentinel definitions (lines 16-27), `addError()` (lines 83-91), `Compile()` (lines 94-272), `validate()` (lines 527-558), `validateType()` (lines 560-578)
- `/home/user/repos/gotya/compiler/compiler_test.go` — All 21 `assert.Contains(t, err.Error(), ...)` sites confirmed by grep; table-driven test structure at line 854-944
- `/home/user/repos/gotya/compiler/validator.go` — Validator error routing: `v.errors []string` + single `c.addError(ErrInvalidDefault, err.Error())` at compiler.go line 254
- `/home/user/repos/gotya/gotya.go` — Current early-return loop at lines 131-138; existing `errors.Join` import absent; `fmt` import present
- `/home/user/repos/gotya/gotya_test.go` — Existing test structure; package `gotya_test`; imports `errors` stdlib
- `/home/user/repos/gotya/go.mod` — Go 1.25.5 (errors.Join available since 1.20); testify v1.11.1
- `.planning/STATE.md` — Phase 2 decision: "Validator refactored to call c.addError(sentinel, msg) directly"

### Secondary (MEDIUM confidence)
- `.planning/phases/06-v1-gap-closure/06-CONTEXT.md` — Locked implementation decisions, verified against source

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- 21-site sentinel mapping: HIGH — derived directly from source code grep + compiler.go addError call sites
- Validator refactoring necessity: HIGH — derived from reading compiler.go line 254 and validator.go error path
- API-02 loop restructure: HIGH — current loop at gotya.go lines 131-138 is unambiguous
- TestCompile_MultiModuleErrors sentinel access: MEDIUM — import path for compiler in gotya_test needs planner decision

**Research date:** 2026-03-15
**Valid until:** 2026-04-15 (stable codebase; no external dependencies changing)