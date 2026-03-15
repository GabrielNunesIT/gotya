---
id: T02
parent: S05
milestone: M001
provides:
  - "Diagnostic struct (File, Line, Column, Message) in gotya.go"
  - "ParseError struct with Errors []Diagnostic and Error() string in gotya.go"
  - "Parse() returns *ParseError (not os.ErrInvalid) with all diagnostics"
  - "ParseFile() fills Diagnostic.File from the cleaned path argument"
  - "ParserDiagnostic struct and Diagnostics() method in parser/parser.go"
  - "RFC 7950 §6.2 identifier validation in parseStatement() — invalid keywords produce errors"
  - "Name() method on ast.Module returning module argument"
  - "Schema() method on schema.Module returning itself"
requires: []
affects: []
key_files: []
key_decisions: []
patterns_established: []
observability_surfaces: []
drill_down_paths: []
duration: 10min
verification_result: passed
completed_at: 2026-03-15
blocker_discovered: false
---
# T02: 05-public-api-stabilization 02

**# Phase 5 Plan 02: Structured Parser Diagnostics Summary**

## What Happened

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
