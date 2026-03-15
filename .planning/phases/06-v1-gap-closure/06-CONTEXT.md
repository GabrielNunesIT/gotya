# Phase 6: v1.0 Gap Closure - Context

**Gathered:** 2026-03-15
**Status:** Ready for planning

<domain>
## Phase Boundary

Close two gaps identified by the v1.0 milestone audit:
1. **TEST-03** — Migrate `compiler/compiler_test.go` error assertions from `assert.Contains(t, err.Error(), ...)` to `assert.ErrorIs(t, err, compiler.ErrXxx)` sentinel-based assertions.
2. **API-02** — Fix `gotya.Compile()` to accumulate errors across all modules instead of stopping at the first failing module.

Adding new capabilities, new sentinel types, or expanding the compiler beyond these two targeted fixes is out of scope.

</domain>

<decisions>
## Implementation Decisions

### API-02: Aggregate error type
- Use `errors.Join(errs...)` to combine per-module errors — stdlib, no new types, `errors.Is` works against any sentinel in the joined chain
- Keep one shared `compiler.Compiler` instance across all modules (preserve existing cross-module state sharing)
- Keep per-module error wrapping: `fmt.Errorf("compile module %s: %w", name, err)` — callers can tell which module failed
- Return `nil, errors.Join(errs...)` when any module fails — clean contract, no partial results alongside errors

### API-02: Test coverage
- Add a `TestCompile_MultiModuleErrors` test in `gotya_test.go` (package `gotya_test`)
- Pass 2 modules each with a distinct sentinel error; assert `errors.Is` for both sentinels in the joined result
- Proves the accumulation behavior doesn't silently regress

### TEST-03: Scope of migration
- Only migrate `assert.Contains(t, err.Error(), ...)` calls — non-error `assert.Contains` (e.g., `assert.Contains(t, schemaMod.Nodes, ...)`) are unrelated and stay unchanged
- 21 target call sites in `compiler/compiler_test.go`

### TEST-03: Table-driven test
- The identityref table-driven test (line ~939) uses `tt.errorMsg string` — replace with `tt.sentinel error`
- All 3 table cases map to `compiler.ErrIdentityrefBase`
- Assertion becomes `assert.ErrorIs(t, err, tt.sentinel)`

### TEST-03: Multi-error cases
- Some tests assert 3+ errors from a single compile call (e.g., `ErrTypeRestriction` ×3, `ErrListMissingKey` ×2)
- Check each expected sentinel individually with `assert.ErrorIs` — verifies accumulation behavior
- For repeated same-sentinel cases, one `assert.ErrorIs` check is sufficient (sentinel matches on first hit)

### Claude's Discretion
- Exact order of error collection and loop structure in `Compile()` — only constraint is `errors.Join` at end
- How to handle the `assert.Contains` calls that check for both sentinels and specific field values (line ~348: 3 type restriction calls) — planner decides whether to assert once per unique sentinel or once per call site

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- `compiler.ErrXxx` sentinels: 12 already defined in `compiler/compiler.go` lines 16–27 — all error conditions in `compiler_test.go` map to these
- `addError(sentinel, msg)` in `compiler.go`: already wraps with `%w`, so `errors.Is` works on any returned error
- `errors.Join`: stdlib since Go 1.20 — no import changes needed beyond adding it to `gotya.go`
- `gotya_test.go` (package `gotya_test`): existing test file; new multi-module test goes here

### Established Patterns
- `fmt.Errorf("%w: %s", sentinel, msg)` — `addError` wrapping pattern; `errors.Is` traverses this correctly
- `assert.ErrorIs(t, err, sentinel)` — already used in Phase 2 tests; consistent pattern
- Test files use `package gotya_test` (external) — callers cannot import `compiler` directly; must use `compiler.ErrXxx` via `compiler` package import

### Integration Points
- `gotya.go Compile()`: loop at line ~127–140 needs error collection instead of early return
- `compiler/compiler_test.go`: 21 `assert.Contains(t, err.Error(), ...)` call sites to migrate
- No changes to `compiler.go` or sentinel definitions needed — all sentinels already exist

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches within the decisions above.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 06-v1-gap-closure*
*Context gathered: 2026-03-15*
