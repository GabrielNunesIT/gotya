# Stack Research

**Domain:** YANG parsing and code generation library (Go)
**Researched:** 2026-03-14
**Confidence:** HIGH for core stack (all verified against official sources); MEDIUM for ecosystem peers (WebSearch + pkg.go.dev)

---

## Current Stack (What gotya Already Uses)

| Technology | Version | Purpose |
|------------|---------|---------|
| Go | 1.25.5 | Only language; library and generated code are both Go |
| github.com/stretchr/testify | v1.11.1 | assert/require/mock in tests |
| vektra/mockery | v3.x | Interface mock generation (dev tool, not a dep) |
| golangci-lint | v2.x | Multi-linter aggregator (dev tool, not a dep) |

No framework, no database, no service clients. The current dep graph is intentionally minimal and should stay that way (PROJECT.md explicitly states: "No new dependencies: Prefer stdlib solutions").

---

## Ecosystem Peers — What They Do and What to Learn From Them

### openconfig/goyang — v1.6.3 (July 2025)

**What it is:** The canonical Go YANG parser. Converts `.yang` files to an in-memory AST (`pkg/yang`) or a more fully-resolved Entry tree. It is explicitly "a work in progress" and does not attempt code generation.

**What ygot does with it:** ygot uses goyang as its parsing backend, then ygen does code generation on top of the Entry tree goyang produces.

**Confidence:** HIGH — verified on pkg.go.dev and GitHub (July 2025 release).

**What to learn:**
- goyang exposes a two-layer model: raw AST and a resolved `Entry` tree. gotya has the same structure (AST → compiled schema). This layering is correct.
- goyang's Entry tree normalizes `augment`, `grouping`, `uses`, and `typedef` resolution before any generator sees the schema. gotya's compiler handles this too — but the active bugs in augment/refine resolution and circular typedef resolution are exactly the class of problems goyang has battle-tested solutions for. Reading goyang's resolution code is worthwhile during the bug-fix phase.
- goyang tests its parser with a corpus of real `.yang` files. gotya already has a `test/assets/yangs/` directory with ~160 real BBF/IETF/IEEE models. That corpus should be wired into integration tests, not just sitting on disk unused.

**Do NOT adopt goyang as a dependency.** gotya has its own lexer/parser/AST — importing goyang would create a parallel runtime representation with no benefit, and goyang itself says to use ygot if you want code gen.

---

### openconfig/ygot — v0.34.0 (September 2025)

**What it is:** The production-grade YANG-to-Go code generator used in production OpenConfig network stacks. It generates Go structs, validation methods, and JSON (RFC 7951) marshaling. Uses goyang for parsing, ygen for code generation, ygot for runtime helpers, ytypes for schema-driven validation.

**Confidence:** HIGH — verified on GitHub (September 2025 release).

**What to learn:**

#### Testing pattern: golden files over inline string assertions

ygot's code generation tests (`ygen/`) store expected output in `testdata/structs/*.formatted-txt` files and compare actual generator output byte-for-byte against those files. Tests accept a `-update` flag to regenerate golden files when output intentionally changes.

gotya's current generator tests use `assert.Contains(t, out, "expected snippet")` — this catches regressions on specific snippets but completely misses unexpected output, wrong ordering, extra fields, or structural issues. For a code generator, golden-file testing is the correct pattern: the whole output is the contract, not a selection of substrings.

**Recommended action:** Add golden file testing for both Go and Protobuf generators, with a `-update` flag to regenerate. No new dependency required — this is plain `os.ReadFile` / `bytes.Equal` / flag logic.

#### Testing pattern: YANG corpus integration tests

ygot runs its generator against the full OpenConfig YANG model set as part of CI. gotya has ~160 real BBF/IETF/IEEE models in `test/assets/yangs/` but no test that feeds them through the full pipeline and checks the output compiles. A "smoke" integration test that parses all assets, compiles them, and generates Go/Proto output (checking only that no panic occurs and the output passes `go/format.Source()` syntax validation) would catch the augment/refine bugs that are currently only known from manual testing.

#### Generation quality: `go/format.Source()` round-trip

ygot passes all generated Go source through `go/format.Source()` before writing output. This serves two purposes: it catches syntax errors in generated code immediately (instead of silently writing invalid Go), and it ensures the output is always gofmt-canonical. gotya's Go generator should do the same.

`go/format` is stdlib — no dependency required. Use `go/format.Source([]byte)` which returns `([]byte, error)`; an error means the generator produced syntactically invalid Go.

---

## Recommended Stack Changes (What to Add)

The project constraint is "no new dependencies without justification." Only one addition is justified, and it is borderline optional:

### Option A: No new dependencies (recommended)

Implement golden file testing with plain stdlib:

```go
// In generator_test.go
var update = flag.Bool("update", false, "update golden files")

func checkGolden(t *testing.T, name string, got []byte) {
    t.Helper()
    path := filepath.Join("testdata", name+".golden")
    if *update {
        require.NoError(t, os.WriteFile(path, got, 0o644))
        return
    }
    want, err := os.ReadFile(path)
    require.NoError(t, err)
    assert.Equal(t, string(want), string(got))
}
```

This pattern is used by the Go standard library itself (`go/format`, `go/doc`, etc.) and requires zero new dependencies.

### Option B: Add goldie v2 (optional, low value here)

goldie v2.8.0 (October 2025) provides the golden file pattern with added conveniences: `go test -update` flag integration, colored diffs, JSON/XML pretty-printing. It does not integrate with testify's `assert` directly but works alongside it.

**Do NOT add goldie.** The stdlib approach is 20 lines and gotya's PROJECT.md explicitly resists new deps. goldie adds value in projects with dozens of test fixtures needing templating; gotya's generator tests are straightforward byte comparisons.

---

## Development Tools (Keep and Upgrade)

### golangci-lint — upgrade to v2

The codebase already uses golangci-lint v2 (configured in `.golangci.yml`). Current stable is v2.1.2+ (April 2025). Note that v2 changed configuration schema significantly:

- `enable-all` / `disable-all` replaced with `linters.default: [all|standard|none|fast]`
- Requires `version: "2"` at top of `.golangci.yml`
- New `golangci-lint migrate` command assists with config migration

**Confidence:** HIGH — verified on golangci-lint.run official docs.

If the existing `.golangci.yml` was written for v1, run `golangci-lint migrate` before the v1.0 release.

### mockery — already at v3

mockery v3.6.0 (2025) is current. The codebase already uses mockery. No action needed unless `.mockery.yaml` was written for v2 config schema (check the [v3 migration guide](https://vektra.github.io/mockery/v3.0/v3/)).

### go/format (stdlib) — add to generators

Use `go/format.Source([]byte) ([]byte, error)` in the Go generator's output path. Not a new dependency — it is in the Go standard library at `go/format`. This validates generated code before writing and formats it canonically.

---

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| openconfig/goyang as dependency | gotya has its own parser/AST; importing goyang creates a parallel in-memory representation with no benefit; goyang itself is "not complete" | gotya's own lexer/parser/compiler |
| openconfig/ygot as dependency | ygot is a competing implementation, not a utility library; its runtime types are incompatible with gotya's generated types | gotya's own generators |
| goldie or other golden-file libraries | 20-line stdlib alternative exists; PROJECT.md restricts new deps | os.ReadFile + bytes.Equal + -update flag |
| jennifer (code-gen library) | gotya already has a working text/template-based generator; jennifer would require a full generator rewrite for no correctness gain | text/template (already in use) |
| stretchr/testify v2 (go-openapi fork) | The go-openapi fork is a separate project; stretchr/testify v1.11.1 is the correct canonical version; no v2 exists at stretchr/testify | stretchr/testify v1.11.1 |

---

## Patterns the Ecosystem Uses That gotya Should Adopt

These are patterns, not dependencies. Zero new deps required.

### 1. Golden file testing for generators

**Pattern:** Store complete expected generator output in `testdata/*.golden` files. Compare full output, not substrings. Accept `-update` flag to regenerate.

**Why:** `assert.Contains` catches regressions on specific snippets but misses: extra unexpected fields, wrong struct ordering, malformed imports, missing methods. A golden file test is the full contract.

**Where to apply:** `generator/golang/generator_test.go` and `generator/protobuf/generator_test.go`.

### 2. `go/format.Source()` validation in generators

**Pattern:** After building the output buffer, pass it through `go/format.Source()`. Return an error if it fails. Write the formatted bytes, not the raw buffer.

**Why:** Catches generator bugs at the point of generation, not when the user tries to compile. Also eliminates the need to manually maintain whitespace in templates.

**Where to apply:** `generator/golang/generator.go` output path.

### 3. Corpus smoke tests against real YANG assets

**Pattern:** A single test that iterates all files in `test/assets/yangs/`, runs parse→compile→generate on each, and asserts no panic and (for Go generator) no `go/format.Source()` error.

**Why:** gotya already has 160+ real BBF/IETF/IEEE models. Running them through the pipeline is the cheapest way to find augment resolution crashes, typedef panics, and nil dereferences before users do. ygot uses the OpenConfig public model set for the same purpose.

**Where to apply:** New `test/integration_test.go` (or alongside existing `test/test.go`).

### 4. Error sentinel values over string matching in tests

**Pattern:** Define exported `var Err... = errors.New(...)` sentinels for expected compiler errors. In tests, use `errors.Is()` or `require.ErrorIs()` rather than `assert.Contains(t, err.Error(), "some string")`.

**Why:** String-matched error messages are fragile; wording changes break unrelated tests. Compiler error types also make it easier for library users to handle errors programmatically.

**Where to apply:** compiler package, wherever `errors.New()` or `fmt.Errorf()` produce errors that tests currently match by string.

---

## Stack Variants

**If adding RFC 7951 codec test coverage:**
- Use the existing `codec/rfc7951/` package directly in tests
- Test against the same YANG corpus as the generator tests
- No new dependencies

**If building generation audit tooling (for the "audit Go/Proto generators" work item):**
- Parse `test/assets/yangs/` in a standalone main program or test binary
- Use `go/format.Source()` to validate Go output syntax
- Diff actual output against expected via standard `diff` semantics (bytes.Equal)
- No new dependencies

---

## Version Compatibility

| Package | Compatible With | Notes |
|---------|-----------------|-------|
| stretchr/testify v1.11.1 | Go 1.21+ | v1.11.1 is the current latest at stretchr; not the go-openapi fork |
| golangci-lint v2 | Go 1.22+ | v2 config schema differs from v1; run `migrate` if upgrading |
| mockery v3.x | Go 1.21+ | v3 config schema differs from v2; see migration guide |
| go/format (stdlib) | all supported Go versions | Part of Go toolchain; no separate installation |

---

## Sources

- [openconfig/goyang on GitHub](https://github.com/openconfig/goyang) — v1.6.3 (July 2025), confirmed active; HIGH confidence
- [openconfig/ygot on GitHub](https://github.com/openconfig/ygot) — v0.34.0 (September 2025), golden file testing pattern; HIGH confidence
- [pkg.go.dev — go/format](https://pkg.go.dev/go/format) — Source() and Node() API; HIGH confidence (stdlib)
- [pkg.go.dev — stretchr/testify](https://pkg.go.dev/github.com/stretchr/testify) — v1.11.1 current; HIGH confidence
- [goldie v2 on pkg.go.dev](https://pkg.go.dev/github.com/sebdah/goldie/v2) — v2.8.0 (October 2025); MEDIUM confidence (WebSearch + pkg.go.dev)
- [golangci-lint v2 official docs](https://golangci-lint.run/docs/product/changelog/) — v2 config migration; HIGH confidence
- [mockery v3 announcement](https://topofmind.dev/blog/2025/04/08/announcing-mockery-v3/) — v3.6.0 current; MEDIUM confidence (WebSearch)
- [YangModels/yang on GitHub](https://github.com/YangModels/yang) — reference YANG module corpus for integration testing; HIGH confidence

---

*Stack research for: gotya — YANG parsing and Go code generation library*
*Researched: 2026-03-14*
