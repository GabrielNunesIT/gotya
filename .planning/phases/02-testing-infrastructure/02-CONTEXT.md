# Phase 2: Testing Infrastructure - Context

**Gathered:** 2026-03-14
**Status:** Ready for planning

<domain>
## Phase Boundary

Add corpus smoke test, golden file harness, and typed error sentinels so every future change is captured by tests. This phase does NOT add generator features — it only adds testing infrastructure that makes Phase 3 and 4 audits reliable.

</domain>

<decisions>
## Implementation Decisions

### Golden File Scope

- **No golden files this phase** — skip golden file tests for now; focus on corpus smoke test and error sentinels
- Golden file infrastructure is deferred to Phase 3 (Go generator audit) and Phase 4 (Proto generator audit)

### Golden File Design (when implemented in Phase 3/4)

- **Source YANG fixtures**: Use real YANG files from the BBF, IANA, and IETF directories under `https://github.com/YangModels/yang/tree/main/standard` — not hand-crafted fixtures
- **Golden file location**: Top-level `testdata/` directory at repo root
- **Update mechanism**: `-update` flag — `go test -run TestGolden ./... -update` rewrites all golden files (standard Go pattern)

### Corpus Smoke Test

- **Existing infrastructure**: `test/generate.go` exists (`//go:build ignore`) and drives the full pipeline over `test/assets/yangs/` — but output is discarded and it's not part of `go test`
- **Goal**: Convert this into a proper `go test` test that asserts no panic and valid Go output for each file in the corpus

### Error Sentinels

- Compiler error conditions should use typed sentinels (exported vars or types in compiler package)
- Tests should use `errors.Is()` / `errors.As()` instead of `assert.Contains(t, err.Error(), "some substring")`
- Existing string-matching assertions in `compiler/compiler_test.go` should be migrated

### go/format.Source() Integration

- Go generator should validate output with `go/format.Source()` before writing
- Invalid generated Go = generation error surfaced at generation time (not at user compile time)

### Claude's Discretion

- Exact corpus test structure (single test with subtests vs. table-driven)
- How failures are reported for corpus test (fail fast vs. collect all failures)
- Exact sentinel type design (var vs. type, granularity per condition)
- Internal plumbing for `-update` flag detection

</decisions>

<specifics>
## Specific Ideas

- Real-world YANG files from `YangModels/yang` standard directory — BBF, IANA, IETF focus. These are the canonical reference models that real network devices implement. Using them as fixtures ensures gotya generates correct output for real-world inputs.
- The `-update` flag pattern is the Go community standard (used by ygot, Go stdlib tests, etc.) — regenerates golden files intentionally.

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets

- `test/generate.go`: Full pipeline driver (parse → compile → generate Go + Proto) over `test/assets/yangs/`. Already has `fileLoader` implementation. Refactor into `_test.go` to make it part of `go test`.
- `test/assets/yangs/`: 205 real-world YANG files (BBF, IETF, IEEE) — ready corpus
- `compiler/compiler_test.go`: 17+ test functions, all using `assert.Contains(t, err.Error(), ...)` for error assertions — migration targets for typed sentinels

### Established Patterns

- Tests are co-located in `_test.go` files using `testify/assert`
- Table-driven tests used in `compiler_test.go` for multi-case scenarios
- `t.Parallel()` used in compiler tests
- All test packages use external test package style (`package xxx_test`)

### Integration Points

- `go/format.Source()` integrates into `generator/golang/generator.go` output path — called before writing to `io.Writer`
- Corpus smoke test integrates into `test/` package — converts `generate.go` logic into proper test
- Error sentinels live in `compiler` package as exported `var Err...` or exported error types

</code_context>

<deferred>
## Deferred Ideas

- Golden file tests for Go generator — Phase 3 (deferred from this phase by user decision)
- Golden file tests for Proto generator — Phase 4
- Coverage enforcement (no coverage threshold was requested)
- Benchmark tests for compiler performance

</deferred>

---

*Phase: 02-testing-infrastructure*
*Context gathered: 2026-03-14*
