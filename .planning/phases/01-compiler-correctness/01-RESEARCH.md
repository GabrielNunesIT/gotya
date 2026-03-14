# Phase 1: Compiler Correctness - Research

**Researched:** 2026-03-14
**Domain:** Go compiler internals — nil-safety, cycle detection, bounded error accumulation
**Confidence:** HIGH

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| COMP-01 | Library never panics or crashes on malformed YANG input — nil pointer dereference on `findNode()` return values guarded with proper error propagation | Two confirmed nil-deref sites at compiler.go:778 and compiler.go:930; fix pattern: nil-check before dereference, append to `c.errors`, return early |
| COMP-02 | Circular typedef definitions (A uses B uses A) produce a compile error, not a stack overflow — `getType()` has a visited set and depth limit | `getType()` at compiler.go:791–826 recurses without a visited set; the `resolveUses()` visited-map pattern at compiler.go:876–881 is the exact template to follow |
| COMP-03 | Circular module imports produce a compile error, not infinite recursion — loader has an in-progress set for cycle detection | `DirectoryLoader.Load()` caches completed modules but writes to the cache AFTER compilation; a module that triggers its own import during compilation re-enters `Load()` with nothing in the cache, causing infinite recursion; fix: add an `inProgress map[string]bool` checked before calling `compiler.New().Compile()` |
| COMP-04 | Augment resolution verifies all augments applied — loop has a max-iteration cap and a post-loop assertion that `pendingAugments` is empty | Augment loop at compiler.go:171–197 terminates on no-progress but has no iteration cap and no post-loop check; augments that cannot resolve silently reach the error append at compiler.go:199 — the error append already exists but no cap guards pathological input from looping indefinitely before reaching it |
| COMP-05 | All `AddChild()` failures propagate as errors — no `_ = AddChild(...)` call sites remain; RPC, Action, and Case nodes report duplicate identifier errors | Five confirmed `_ =` sites: compiler.go:415, 418, 426, 429, 728; pattern: replace `_ = node.AddChild(x)` with `if err := node.AddChild(x); err != nil { c.errors = append(c.errors, err.Error()) }` |
| COMP-06 | Library emits nothing to stdout or stderr during normal operation — all `fmt.Printf("DEBUG ...`)` calls removed from production code paths | Two confirmed debug printf sites: compiler.go:778 (findNode) and compiler.go:930 (resolveUses); these are in error paths where errors are already being returned — removal is mechanical |
| COMP-07 | Compiler error accumulation is bounded — `compiler.Options` has a `MaxErrors` field (default 100); compilation stops after limit with truncation message | `c.errors []string` at compiler.go:22 is unbounded; `Options` struct at compiler.go:34–37 needs a `MaxErrors int` field; every `c.errors = append(c.errors, ...)` site needs a len-check guard |

</phase_requirements>

---

## Summary

Phase 1 is a targeted hardening pass on the compiler package and the CLI `DirectoryLoader`. The domain is well-understood: nil-safety, cycle detection, and bounded resource consumption. All seven requirements map to specific, located bugs in the codebase — there are no design ambiguities and no new architecture decisions required.

The current test suite passes entirely (all packages green as of 2026-03-14). However, none of the existing tests exercise the failure modes this phase addresses: malformed augment paths, circular typedefs, circular imports, unbounded error accumulation, or the debug printf paths. Each requirement therefore demands at least one new test that was previously absent.

The most important ordering constraint within the phase is that COMP-06 (remove debug printfs) must be done alongside COMP-01 (nil-guard findNode), because both sites are co-located: the printf at compiler.go:778 fires inside the nil-return branch of `findNode()`, and the printf at compiler.go:930 fires inside the grouping-not-found branch of `resolveUses()`. Removing the printfs and adding error propagation are a single atomic change at each site.

**Primary recommendation:** Address each requirement as a standalone, independently testable commit. The order COMP-06/COMP-01 together, then COMP-05, then COMP-02, then COMP-03, then COMP-04, then COMP-07 minimizes the surface area of each change and makes bisection straightforward if a regression appears.

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib | 1.25.5 | All fixes use only the stdlib | No new dependencies; all patterns are standard Go idioms |
| stretchr/testify | v1.11.1 | `assert` and `require` in new tests | Already in use across all test files; consistent assertion style |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `errors` (stdlib) | — | `errors.New()` for new error values | Any new error sentinel needed for test assertions |
| `fmt` (stdlib) | — | `fmt.Errorf()` for error wrapping | Error context at call sites |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `map[string]bool` visited set | `map[string]struct{}` | `struct{}` uses zero memory per entry; either works; `bool` is marginally more readable for the existing codebase style |
| Depth counter for COMP-02 | Visited set only | A visited set is sufficient and consistent with the `resolveUses()` pattern already in the file; a depth counter adds no safety benefit when a visited set is present |

**Installation:** No new packages required.

---

## Architecture Patterns

### Where the Fixes Live

```
compiler/
├── compiler.go     # All 7 requirements have fix sites here
│                   # + Options.MaxErrors (COMP-07)
│                   # + getType() visited set (COMP-02)
│                   # + findNode() nil-guard + printf removal (COMP-01, COMP-06)
│                   # + resolveUses() printf removal (COMP-06)
│                   # + _ = AddChild() sites (COMP-05)
│                   # + augment loop cap (COMP-04)
└── compiler_test.go # New tests for each requirement
cmd/gotya/
└── loader.go       # DirectoryLoader.Load() in-progress set (COMP-03)
```

### Pattern 1: Visited-Set Cycle Detection (for COMP-02)

`resolveUses()` at compiler.go:869 already implements the correct pattern. `getType()` needs the same treatment.

**What:** Pass a `visited map[string]bool` parameter through `getType()`. Before recursing into a typedef, check `visited[typedefName]`; if true, append an error and return a zero `TypeDefinition`. Set `visited[typedefName] = true` before recursing; use `defer delete(visited, typedefName)` after.

**When to use:** Any recursive function that follows user-controlled references in the YANG tree.

**Example (resolveUses pattern to replicate):**
```go
// Source: compiler/compiler.go:869-881 — existing pattern
func (c *Compiler) resolveUses(stmt ast.Statement, ..., visited map[string]bool, ...) {
    groupName := stmt.Argument()
    if visited == nil {
        visited = make(map[string]bool)
    }
    if visited[groupName] {
        c.errors = append(c.errors, "circular dependency detected in uses: "+groupName)
        return
    }
    visited[groupName] = true
    defer delete(visited, groupName)
    // ... recursion proceeds safely
}
```

Apply to `getType()`:
```go
// New signature
func (c *Compiler) getType(stmts []ast.Statement, visited map[string]bool) schema.TypeDefinition {
    // ...
    if typedefAST, ok := c.typedefs[td.Name]; ok {
        if visited == nil {
            visited = make(map[string]bool)
        }
        if visited[td.Name] {
            c.errors = append(c.errors, "circular typedef detected: "+td.Name)
            return td
        }
        visited[td.Name] = true
        defer delete(visited, td.Name)
        resolved := c.getType(typedefAST.SubStatements(), visited)
        // ... merge resolved
    }
}
```

Note: `getType()` is called from multiple sites in `compiler.go`. All call sites must be updated to pass `nil` as the initial visited map (the function initializes it on first use).

### Pattern 2: In-Progress Guard for Loader (for COMP-03)

`DirectoryLoader.Load()` at loader.go:87–112 caches results after compilation completes. A circular import `A imports B imports A` causes `Load("A")` to call `Load("B")` which calls `Load("A")` again — the schemaCache has no entry for `A` yet because compilation has not finished.

**What:** Add an `inProgress map[string]bool` field to `DirectoryLoader`. Check it at the top of `Load()` before looking up `schemaCache`. If `inProgress[name]` is true, return a sentinel error. Set `inProgress[name] = true` before calling `Compile()`; set it to false (or delete) after.

```go
// Source pattern: loader.go:87 — add in-progress guard
func (l *DirectoryLoader) Load(name string) (*schema.Module, error) {
    if m, ok := l.schemaCache[name]; ok {
        return m, nil
    }
    if l.inProgress[name] {
        return nil, fmt.Errorf("circular import detected: module %s is already being compiled", name)
    }
    l.inProgress[name] = true
    defer func() { delete(l.inProgress, name) }()
    // ... rest of existing Load() body unchanged
}
```

### Pattern 3: Augment Loop Cap (for COMP-04)

The convergence loop at compiler.go:171–197 already terminates on no-progress. The gap is that pathological input (N augments with no resolvable targets) loops N times before making no progress — this is bounded but can be slow and should be capped explicitly.

**What:** Add a `maxIter` constant (value: `len(c.augments) + 1` computed once before the loop, or a fixed cap of 1000) and an iteration counter. Break with an error if the cap is reached before convergence.

```go
// Source pattern: compiler.go:171 — add cap
maxIter := len(c.augments) + 1
iter := 0
progress := true
for progress {
    if iter >= maxIter {
        // append error for each remaining augment and break
        for _, aug := range c.augments {
            c.errors = append(c.errors, "augment target not found (loop cap reached): "+aug.Argument())
        }
        c.augments = nil
        break
    }
    iter++
    progress = false
    // ... existing loop body unchanged
}
// The post-loop check at compiler.go:199 remains as the definitive assertion
```

The post-loop error append at compiler.go:199–201 already reports unresolved augments. The cap just prevents an unbounded spin before reaching that check.

### Pattern 4: AddChild Error Propagation (for COMP-05)

All five `_ =` sites follow the same pattern. The RPC/Action sites (compiler.go:415, 418, 426, 429) are guarded by a nil-check so the AddChild call only runs when the child does not already exist — a duplicate error from these sites would be unexpected but should still propagate. The Case site (compiler.go:728) is the shorthand-case creation path.

```go
// Before (5 sites)
_ = rpcNode.AddChild(schema.NewInput())

// After (same pattern at all 5 sites)
if err := rpcNode.AddChild(schema.NewInput()); err != nil {
    c.errors = append(c.errors, err.Error())
}
```

### Pattern 5: MaxErrors Bound (for COMP-07)

```go
// Updated Options struct
type Options struct {
    Loader            ModuleLoader
    SupportedFeatures []string
    MaxErrors         int // 0 means use default of 100
}

// Helper used at every c.errors append site
func (c *Compiler) addError(msg string) bool {
    c.errors = append(c.errors, msg)
    max := c.maxErrors
    if max <= 0 {
        max = 100
    }
    return len(c.errors) >= max
}
```

Or alternatively: wrap all `c.errors = append(...)` call sites with an inline guard. The helper function approach is cleaner since there are approximately 20 error-append sites in compiler.go.

When the cap is reached, the compiler should append a final message — `"compilation stopped: too many errors (limit N)"` — and return immediately from `Compile()`.

### Anti-Patterns to Avoid

- **Panic recovery as a substitute for nil guards:** Using `recover()` to catch nil-pointer panics is not an acceptable fix for COMP-01. The requirement is proper nil-checking with error propagation, not a deferred recover.
- **Changing the `getType()` signature without updating all call sites:** `getType()` is called from `compileDataNode()` and the inline union-member recursion at compiler.go:856. All call sites must pass `nil` as the initial visited map.
- **Setting inProgress to false in a non-deferred path:** If the Compile call panics or returns early, `inProgress[name]` must still be cleared to avoid permanently blocking re-compilation of a module in a subsequent call.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Cycle detection | Custom DFS graph traversal | `map[string]bool` visited set passed as parameter | The problem is a simple reachability check, not a full topological sort; a visited set is 5 lines; a graph library is 500 |
| Error limiting | Ring buffer, priority queue | Slice with length check | Errors are appended once and read once at the end; slice with a cap check is the correct data structure |
| In-progress detection | Separate goroutine, mutex, channel | `map[string]bool` field on the loader | `DirectoryLoader` is not concurrent; a simple bool map is correct and zero-overhead |

**Key insight:** All seven requirements are solved by standard Go idioms that are already present elsewhere in the same file. The fixes are application of existing patterns to new sites, not new algorithms.

---

## Common Pitfalls

### Pitfall 1: getType() Signature Change Breaks Internal Recursion
**What goes wrong:** `getType()` calls itself recursively for union member types at compiler.go:856 (`member := c.getType([]ast.Statement{sub})`). If the visited-map parameter is added to the signature but this internal recursive call is not updated, it fails to compile.
**Why it happens:** There are two recursive call sites — the typedef resolution path AND the union member path. The union member path does not need cycle detection (union members are not typedef references) but must still be updated to pass the visited parameter (can pass nil or the current visited map).
**How to avoid:** Search for all `c.getType(` occurrences before writing the fix. There are at least three: the typedef resolution path (needs the visited map), the union member recursion (pass nil), and any call from `compileDataNode()` (pass nil as initial value).
**Warning signs:** Compile error `not enough arguments in call to c.getType` — means an internal call site was missed.

### Pitfall 2: Loader inProgress Map Not Initialized
**What goes wrong:** `DirectoryLoader` is constructed in `NewDirectoryLoader()`. Adding an `inProgress` field without initializing it in `NewDirectoryLoader()` causes a nil map panic on the first `inProgress[name] = true` assignment.
**Why it happens:** Go nil maps are readable (return zero value) but panic on write.
**How to avoid:** Add `inProgress: make(map[string]bool)` to the `NewDirectoryLoader()` return literal at loader.go:29–34.
**Warning signs:** `panic: assignment to entry in nil map` on the first circular import test.

### Pitfall 3: Augment Loop Cap Set Too Low
**What goes wrong:** The cap `len(c.augments) + 1` assumes augments resolve in at most N passes. Valid augment chains of depth K require K passes. If `maxIter` is set to a fixed constant smaller than the maximum valid chain depth in the test corpus, valid YANG files start failing.
**Why it happens:** The correct cap for a convergence loop is `len(pending) + 1` (computed once before the loop starts), not a small fixed constant. Each pass must resolve at least one augment for progress to continue; after N augments are resolved in N passes, the (N+1)th pass will make no progress and the loop exits normally.
**How to avoid:** Compute `maxIter = len(c.augments) + 1` before the loop, not a hardcoded constant. The `test/assets/yangs/` corpus should be run after the fix to verify no regressions.
**Warning signs:** Test failures on valid YANG files with deep augment chains (augment B targeting a node added by augment A).

### Pitfall 4: Debug Printf Removal Breaks Error Visibility
**What goes wrong:** The printf at compiler.go:778 fires in `findNode()` when a path component does not resolve. After removing it, the caller may silently return nil with no error recorded if the nil-guard is not also added. The printf was serving as a debug breadcrumb that is now the only indication something went wrong.
**Why it happens:** The existing code returns nil from `findNode()` (line 779) but does not append to `c.errors`. The caller (`resolveAugments` loop) handles a nil return by adding the augment to `pendingAugments` — correct behavior — but the grouping-not-found case at line 930 returns early without any error being propagated if the printf is removed without adding error recording.
**How to avoid:** For each printf removal, verify the surrounding code already records an error (or add one). At compiler.go:930 the printf fires inside `if grpAST == nil` — the `c.errors = append(...)` at line 929 already records the error before the printf; removing the printf is safe at that site. At compiler.go:778 (inside `findNode()`), the function returns nil; the nil propagates to the caller's nil-check at compiler.go:179; the caller already handles it. The printf at 778 is safe to remove without any other change.
**Warning signs:** A test that previously saw a debug message now produces no output and no error — means an error was lost.

### Pitfall 5: MaxErrors Check Missing at Every Append Site
**What goes wrong:** If `addError()` is introduced as a helper but some `c.errors = append(c.errors, ...)` sites are missed (call sites that don't use the helper), the error bound is not actually enforced.
**Why it happens:** There are approximately 20 error-append sites in compiler.go. A search-and-replace pass is required.
**How to avoid:** After adding the helper, grep for all `c.errors = append(c.errors` occurrences and replace each one with `c.addError(...)`. A CI linter rule checking for direct slice append to `c.errors` is ideal but out of scope for Phase 1.
**Warning signs:** A test that feeds a YANG file with 200 validation errors does not stop at 100.

---

## Code Examples

Verified patterns from the existing codebase:

### Existing Visited-Set Pattern (resolveUses — compiler.go:869)
```go
// Source: compiler/compiler.go:869-881
func (c *Compiler) resolveUses(stmt ast.Statement, localGroupings map[string]ast.Statement,
    visited map[string]bool, mod *schema.Module, parent schema.Node) {
    groupName := stmt.Argument()
    if visited == nil {
        visited = make(map[string]bool)
    }
    if visited[groupName] {
        c.errors = append(c.errors, "circular dependency detected in uses: "+groupName)
        return
    }
    visited[groupName] = true
    defer delete(visited, groupName)
    // ... safe to recurse
}
```

### Existing AddChild Error Propagation (already correct — compiler.go:733)
```go
// Source: compiler/compiler.go:733-735 — the correct pattern
if err := parent.AddChild(child); err != nil {
    c.errors = append(c.errors, err.Error())
}
```

The five `_ = AddChild(...)` sites at lines 415, 418, 426, 429, 728 must be brought to match this already-correct pattern.

### Existing Loader Cache Pattern (loader.go:87-112)
```go
// Source: cmd/gotya/loader.go:87-112
func (l *DirectoryLoader) Load(name string) (*schema.Module, error) {
    if m, ok := l.schemaCache[name]; ok {
        return m, nil  // cache hit — safe, no recursion
    }
    // BUG: nothing prevents re-entry here for in-progress modules
    astMod, err := l.LoadAST(name)
    // ...
    comp := compiler.New(&compiler.Options{Loader: l})
    schemaMod, err := comp.Compile(astMod)  // this can call Load() again
    if schemaMod != nil {
        l.schemaCache[name] = schemaMod  // cache written AFTER compile
    }
    // ...
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Unbounded recursion in getType() | Visited-set guard (to be added) | Phase 1 | Circular typedefs return error instead of crashing the process |
| `_ = AddChild(...)` at RPC/Action/Case sites | Propagated errors (to be added) | Phase 1 | Duplicate identifier errors surface instead of silently corrupting schema |
| Unbounded `c.errors` slice | MaxErrors-bounded slice (to be added) | Phase 1 | Memory-safe on adversarial input |
| Debug printf in production code | Error-only output (to be added) | Phase 1 | Library can be used in scripted pipelines without stdout contamination |

**Deprecated/outdated after Phase 1:**
- `fmt.Printf("DEBUG ...")` in compiler.go — replaced by proper error propagation
- `_ = AddChild(...)` pattern — replaced by `if err := ...; err != nil` pattern

---

## Open Questions

1. **Should `compiler.Options.MaxErrors` default to 0 (disabled) or 100 (enabled)?**
   - What we know: The Go compiler defaults to 10 errors before stopping (https://github.com/golang/go/issues/5142). Most linters use 50–100. COMP-07 specifies default 100.
   - What's unclear: Whether existing tests rely on receiving more than 100 errors from a single Compile() call. One test (`TestCompiler_InvalidTypeRestrictions`) expects three specific error messages from a single file — well under the limit.
   - Recommendation: Default to 100 as specified in COMP-07. Existing tests are unaffected; no test input generates more than 100 errors.

2. **Does the `getType()` visited-set fix need to handle cross-typedef chains (A → B → C → A) or only direct cycles (A → A)?**
   - What we know: The visited set pattern from `resolveUses()` handles chains of any depth — it accumulates visited names across the chain and detects a cycle when any name is visited twice.
   - What's unclear: Nothing — the visited-set approach handles both direct and indirect cycles identically.
   - Recommendation: Use the visited set; no additional logic needed.

---

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing stdlib + stretchr/testify v1.11.1 |
| Config file | none (standard `go test`) |
| Quick run command | `/usr/local/go/bin/go test ./compiler/ ./cmd/gotya/ -count=1` |
| Full suite command | `/usr/local/go/bin/go test ./... -count=1` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| COMP-01 | Malformed augment path (nil intermediate node) does not panic | unit | `/usr/local/go/bin/go test ./compiler/ -run TestCompiler_MalformedAugmentPath -count=1` | ❌ Wave 0 |
| COMP-01 | Malformed refine path (nil intermediate node) does not panic | unit | `/usr/local/go/bin/go test ./compiler/ -run TestCompiler_MalformedRefinePath -count=1` | ❌ Wave 0 |
| COMP-02 | Circular typedef (A → B → A) returns error, not stack overflow | unit | `/usr/local/go/bin/go test ./compiler/ -run TestCompiler_CircularTypedef -count=1` | ❌ Wave 0 |
| COMP-03 | Circular module imports return error, not infinite recursion | unit | `/usr/local/go/bin/go test ./cmd/gotya/ -run TestLoader_CircularImport -count=1` | ❌ Wave 0 |
| COMP-04 | Unresolvable augment path produces error listing the path | unit | `/usr/local/go/bin/go test ./compiler/ -run TestCompiler_UnresolvableAugment -count=1` | ❌ Wave 0 |
| COMP-05 | Duplicate RPC input node produces error | unit | `/usr/local/go/bin/go test ./compiler/ -run TestCompiler_DuplicateRPCInput -count=1` | ❌ Wave 0 |
| COMP-06 | Compiler produces no stdout/stderr on normal and error input | unit | `/usr/local/go/bin/go test ./compiler/ -run TestCompiler_NoDebugOutput -count=1` | ❌ Wave 0 |
| COMP-07 | Compile stops after MaxErrors with truncation message | unit | `/usr/local/go/bin/go test ./compiler/ -run TestCompiler_MaxErrors -count=1` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `/usr/local/go/bin/go test ./compiler/ ./cmd/gotya/ -count=1`
- **Per wave merge:** `/usr/local/go/bin/go test ./... -count=1`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `compiler/compiler_test.go` — add `TestCompiler_MalformedAugmentPath`, `TestCompiler_MalformedRefinePath`, `TestCompiler_CircularTypedef`, `TestCompiler_UnresolvableAugment`, `TestCompiler_DuplicateRPCInput`, `TestCompiler_NoDebugOutput`, `TestCompiler_MaxErrors`
- [ ] `cmd/gotya/loader_test.go` — new file; add `TestLoader_CircularImport` (requires a pair of temp YANG files that mutually import each other)

Note: `cmd/gotya` currently has no test files (`[no test files]` confirmed by running the test suite). The loader test will require creating `cmd/gotya/loader_test.go` as part of Wave 0 setup.

---

## Sources

### Primary (HIGH confidence)
- Direct code audit — `compiler/compiler.go` (all line numbers cited are verified by reading the file)
- Direct code audit — `cmd/gotya/loader.go` (circular import bug confirmed by reading Load() body)
- Direct code audit — `compiler/compiler_test.go` (all cited gaps confirmed by absence in the test file)
- `.planning/codebase/CONCERNS.md` (2026-03-14) — all bug locations cross-referenced and confirmed
- `.planning/research/SUMMARY.md` (2026-03-14) — phase rationale and specific line numbers

### Secondary (MEDIUM confidence)
- Go compiler error limit design: https://github.com/golang/go/issues/5142 — supports the 100-error default
- openconfig/goyang issue #265 — augment chaining failure mode confirms the augment loop is a known class of bug in YANG compilers

### Tertiary (LOW confidence — not needed for this phase)
- None. All fixes are mechanical and fully specified from direct code reading.

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies; all tools verified by running the test suite
- Architecture: HIGH — all fix sites identified by reading the source; patterns derived from existing code in the same file
- Pitfalls: HIGH — derived from direct code reading and cross-checked against CONCERNS.md; specific line numbers provided and verified
- Test gaps: HIGH — confirmed by running `go test ./...` and observing `[no test files]` for `cmd/gotya` and absence of listed test functions in `compiler_test.go`

**Research date:** 2026-03-14
**Valid until:** 90 days — compiler.go is not expected to change until Phase 1 work begins; the fix patterns are stable Go idioms
