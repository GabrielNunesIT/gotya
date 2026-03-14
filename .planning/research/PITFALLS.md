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
