---
id: S05
parent: M001
milestone: M001
provides:
  - "RED test baseline in gotya_test.go: 6 failing test stubs covering all Phase 5 API contracts"
  - "Black-box external test package (package gotya_test) that imports only the gotya package"
  - "Diagnostic struct (File, Line, Column, Message) in gotya.go"
  - "ParseError struct with Errors []Diagnostic and Error() string in gotya.go"
  - "Parse() returns *ParseError (not os.ErrInvalid) with all diagnostics"
  - "ParseFile() fills Diagnostic.File from the cleaned path argument"
  - "ParserDiagnostic struct and Diagnostics() method in parser/parser.go"
  - "RFC 7950 §6.2 identifier validation in parseStatement() — invalid keywords produce errors"
  - "Name() method on ast.Module returning module argument"
  - "Schema() method on schema.Module returning itself"
  - Opaque ASTModule struct with Name()/Keyword()/Argument() methods
  - Opaque Module struct with Schema() *schema.Module accessor
  - CompileOptions struct eliminating compiler package import from callers
  - Clean godoc on all exported symbols in gotya.go
  - Stable public API ready for v1 tagging
requires: []
affects: []
key_files: []
key_decisions:
  - "gotya_test.go uses package gotya_test (external) — callers cannot import ast/schema/compiler directly"
  - "t.TempDir() + os.WriteFile used for ParseFile temp file fixture — no helper packages needed"
  - "Schema() accessor pattern chosen for TestCompile_OpaqueReturn — method call verifiable at compile time without schema import"
  - "Blank ParseError.File case (Parse with no path) not tested here — ParseFile fills it, Parse does not"
  - "ParserDiagnostic exported (capital P) from parser package — external packages can access Line/Column/Message fields; unexported struct fields not accessible across packages"
  - "parseContent() internal helper shared by Parse() and ParseFile() — both call same implementation, filename differs"
  - "ParseError.Error() returns '1 parse error: {msg}' or '{N} parse errors: {msg}' — human-readable, consistent format"
  - "isValidYANGIdentifier() checks first byte only — RFC 7950 §6.2 start rule; minimal validation to reject @@ prefix without breaking valid YANG keywords"
  - "Name() on ast.Module returns Arg field — module name is the argument to the module statement"
  - "Schema() on schema.Module returns self — minimal accessor for gotya public API; plan 03 wraps in opaque gotya.Module"
  - "ASTModule changed from type alias (= ast.Module) to defined struct with unexported *ast.Module field — blocks access to BaseNode/SubStatements from external packages"
  - "Keyword() and Argument() exposed on *ASTModule — callers need to distinguish module from submodule (Keyword) and cache by name (Argument); both are legitimate public API"
  - "CompileOptions has no Loader field — loader is an internal CLI concern not exposed at v1 public API level; cmd/gotya uses loader.Load() directly bypassing gotya.Compile"
  - "Removed loader.astCache manual population in main.go — loader.Load->LoadAST already caches AST from disk; manual population was defensive and incompatible with opaque ASTModule type"
  - "cmd/gotya retains direct schema package import — cmd is an internal CLI binary not a public API consumer; schema.Module usage in generateGo/generateProtobuf is legitimate"
patterns_established:
  - "External test package (package foo_test) verifies API encapsulation — internal types cannot leak"
  - "errors.As(err, &parseErr) pattern as primary diagnostic access mechanism"
  - "Structured diagnostic collection: parser accumulates ParserDiagnostic alongside string errors; gotya wraps into Diagnostic"
  - "addError() dual-write pattern: both p.errors ([]string for backward compat) and p.diagnostics ([]ParserDiagnostic for new API)"
  - "Opaque struct pattern: wrap internal type in exported struct with explicit public methods, zero field access"
observability_surfaces: []
drill_down_paths: []
duration: 2min
verification_result: passed
completed_at: 2026-03-15
blocker_discovered: false
---
# S05: Public Api Stabilization

**# Phase 5 Plan 01: Public API Test Stubs Summary**

## What Happened

# Phase 5 Plan 01: Public API Test Stubs Summary

**6 failing RED test stubs in gotya_test.go covering ParseError type, multiple diagnostics, opaque ASTModule/Module return types, and ParseFile file-filled diagnostics — zero internal package imports**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-15T16:13:28Z
- **Completed:** 2026-03-15T16:16:00Z
- **Tasks:** 1
- **Files modified:** 1

## Accomplishments
- Created `gotya_test.go` as `package gotya_test` with 6 test functions, all producing compile-time RED failures
- Tests confirm the three API contracts: `ParseError` type (API-01/02), opaque return types (API-03), and diagnostic File field (API-02)
- File imports only `github.com/gotya/gotya`, `errors`, `os`, `testing`, and `testify/require` — no `ast`, `schema`, or `compiler` packages
- Compilation fails with `undefined: gotya.ParseError` and related errors — establishing the RED baseline for plans 02 and 03

## Task Commits

1. **Task 1: RED test stubs** - `98a2237` (test)

**Plan metadata:** committed with docs commit below

## Files Created/Modified
- `/home/user/repos/gotya/gotya_test.go` - 6 black-box test stubs for all Phase 5 API behaviors

## Decisions Made
- `Schema()` method accessor chosen for `TestCompile_OpaqueReturn` — callers call `compiled[0].Schema()` without importing the schema package; this is the minimal verifiable interface at compile time
- `t.TempDir()` used for temp file in `TestParseFile_FilledDiagnostic` — idiomatic Go test cleanup, no external helpers needed
- Two-case `TestParseFile_FilledDiagnostic`: nonexistent file (I/O error, NOT ParseError) + invalid YANG file (ParseError WITH File field) — covers both sides of the API contract

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None — compilation failed precisely as designed. All 6 undefined symbols confirmed in error output.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- RED baseline established: `go test .` fails with `undefined: gotya.ParseError` and related errors
- Plan 05-02 turns GREEN: implement `ParseError`, `Diagnostic`, opaque `ASTModule`, opaque `Module` in `gotya.go`
- Plan 05-03 turns remaining: godoc cleanup (API-04)

---
*Phase: 05-public-api-stabilization*
*Completed: 2026-03-15*

# Phase 5 Plan 02: Structured Parser Diagnostics Summary

**ParseError/Diagnostic types in gotya.go with RFC 7950 identifier validation in parser — Parse() and ParseFile() now return *ParseError with full position data instead of os.ErrInvalid**

## Performance

- **Duration:** 10 min
- **Started:** 2026-03-15T16:09:00Z
- **Completed:** 2026-03-15T16:19:30Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Added `ParserDiagnostic` struct and `Diagnostics()` method to `parser/parser.go` — captures Line/Column from curToken.Pos on every `addError()` call
- Defined `Diagnostic` and `ParseError` public types in `gotya.go` — `ParseError.Error()` returns human-readable summary; `errors.As()` pattern gives callers full `[]Diagnostic` slice
- Rewrote `Parse()` and `ParseFile()` to use shared `parseContent()` helper — returns `*ParseError` with all diagnostics; `ParseFile` fills `Diagnostic.File` from cleaned path
- All 4 ParseError API tests pass GREEN: `TestParse_Error`, `TestParse_ErrorsAs`, `TestParse_MultipleErrors`, `TestParseFile_FilledDiagnostic`
- Full test suite: `go test ./... -count=1` exits 0 — all 9 test packages pass

## Task Commits

Each task was committed atomically:

1. **Task 1: Add ParserDiagnostic struct and Diagnostics() method to parser** - `a89ff9c` (feat)
2. **Task 2: Define Diagnostic/ParseError types and rewrite Parse()/ParseFile()** - `f5b4fb1` (feat)

**Plan metadata:** committed with docs commit below

## Files Created/Modified
- `/home/user/repos/gotya/parser/parser.go` - Added ParserDiagnostic struct, diagnostics field, updated addError(), added Diagnostics() method, added isValidYANGIdentifier() RFC 7950 §6.2 check
- `/home/user/repos/gotya/gotya.go` - Added Diagnostic and ParseError types, parseContent() helper, rewrote Parse() and ParseFile()
- `/home/user/repos/gotya/ast/ast.go` - Added Name() method to ast.Module
- `/home/user/repos/gotya/schema/schema.go` - Added Schema() method to schema.Module

## Decisions Made
- `ParserDiagnostic` is exported (capital P) — the gotya package needs to access its fields; unexported struct fields are inaccessible across packages even when returned by exported methods
- `parseContent(content, filename string)` shared helper — avoids duplicating lexer/parser construction; filename is `""` for `Parse()` and `cleanPath` for `ParseFile()`
- `isValidYANGIdentifier` checks only the first byte against RFC 7950 §6.2 start rule (`[A-Za-z_]`) — minimal fix, doesn't re-validate entire token stream

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] YANG identifier validation missing from parseStatement()**
- **Found during:** Task 2 (TestParse_MultipleErrors)
- **Issue:** Lexer's `readUnquotedString()` accepts any non-whitespace/non-structural character as an identifier, causing `@@bad-statement-one` to lex as a valid IDENTIFIER token. Parser accepted it without error, resulting in zero diagnostics for the multi-error test input.
- **Fix:** Added `isValidYANGIdentifier()` check at the top of `parseStatement()` — if the keyword's first character is not `[A-Za-z_]`, `addError()` is called and nil is returned, triggering error recovery
- **Files modified:** `parser/parser.go`
- **Verification:** `TestParse_MultipleErrors` passes; all existing parser tests still pass
- **Committed in:** `f5b4fb1` (Task 2 commit)

**2. [Rule 3 - Blocking] ast.Module.Name() and schema.Module.Schema() missing**
- **Found during:** Task 2 (test file compilation)
- **Issue:** `gotya_test.go` references `mod.Name()` (TestASTModule_Name) and `compiled[0].Schema()` (TestCompile_OpaqueReturn) which didn't exist; compilation failed for entire package, blocking verification of the 4 targeted ParseError tests
- **Fix:** Added `Name() string` to `ast.Module` returning `m.Arg`; added `Schema() *Module` to `schema.Module` returning itself
- **Files modified:** `ast/ast.go`, `schema/schema.go`
- **Verification:** Test file compiles; all 6 tests in `gotya_test.go` run; 4 ParseError tests pass, 2 remaining tests (`TestCompile_OpaqueReturn`, `TestASTModule_Name`) also now pass
- **Committed in:** `f5b4fb1` (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (1 bug, 1 blocking)
**Impact on plan:** Both fixes necessary for correctness and test compilation. The identifier validation fix is consistent with RFC 7950 §6.2 and does not affect valid YANG input.

## Issues Encountered
- `TestCompile_OpaqueReturn` and `TestASTModule_Name` test stubs from plan 01 required `Schema()` and `Name()` methods that plan 02 didn't originally cover — added minimal implementations unblocking compilation and making those tests pass ahead of plan 03

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All 4 ParseError API tests pass GREEN (API-01, API-02 satisfied)
- `TestCompile_OpaqueReturn` and `TestASTModule_Name` also pass now — plan 03 will convert `ASTModule` from alias to opaque struct and wrap `schema.Module` in `gotya.Module`, which may require updating these methods
- Plan 03 can proceed: convert `ASTModule = ast.Module` alias to opaque struct, wrap `schema.Module` in `gotya.Module` with `Schema()` accessor, clean godoc, update `cmd/gotya/main.go`

---
*Phase: 05-public-api-stabilization*
*Completed: 2026-03-15*

# Phase 5 Plan 03: Public API Opaque Types Summary

**ASTModule and Module converted to opaque structs with explicit method sets, eliminating internal type leakage through gotya.go public signatures**

## Performance

- **Duration:** ~2 min
- **Started:** 2026-03-15T16:21:45Z
- **Completed:** 2026-03-15T16:23:12Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Replaced `type ASTModule = ast.Module` alias with defined struct wrapping `*ast.Module` — callers can only call Name()/Keyword()/Argument()
- Added `Module` opaque struct with `Schema() *schema.Module` accessor — Compile() no longer returns schema package type at boundary
- Added `CompileOptions` struct — callers configure compilation without importing `compiler` package
- Fixed `cmd/gotya/main.go`: removed manual `loader.astCache` population that became type-incompatible
- Verified all existing tests pass with zero changes to test files

## Task Commits

Each task was committed atomically:

1. **Task 1: Replace ASTModule alias with opaque struct and add Module wrapper** - `72e51cb` (feat)
2. **Task 2: Update cmd/gotya/main.go call sites and clean godoc in gotya.go** - `4ce5824` (feat)

## Files Created/Modified
- `/home/user/repos/gotya/gotya.go` - ASTModule as opaque struct, Module wrapper, CompileOptions, clean godoc
- `/home/user/repos/gotya/cmd/gotya/main.go` - Removed type-incompatible astCache manual population

## Decisions Made
- `Keyword()` and `Argument()` added to `*ASTModule` — minimum was `Name()` but both are needed by `cmd/gotya` and represent legitimate public API for distinguishing module vs submodule
- `CompileOptions.Loader` intentionally absent — the Loader interface is a CLI-internal concept; public callers don't manage file loading
- Removed manual `loader.astCache[astMod.Argument()] = astMod` from main.go rather than adding a backdoor to access the unexported `.mod` field; loader already re-caches from disk correctly

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None. The `TestCompile_OpaqueReturn` and `TestASTModule_Name` tests were already written (from Plan 02 context), and incidentally passed with the old alias design because `schema.Module` has a `Schema()` method returning itself. After the opaque type change, they continue to pass with the correct semantics (calling `gotya.Module.Schema()` not `schema.Module.Schema()`).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- All v1 public API requirements satisfied (API-03: no internal type leakage, API-04: no placeholder comments)
- gotya.go is clean, well-documented, and ready for v1 tagging
- Full test suite passes: `go test ./... -count=1` exits 0 across all packages
- `go build ./...` exits 0 including cmd/gotya

---
*Phase: 05-public-api-stabilization*
*Completed: 2026-03-15*

