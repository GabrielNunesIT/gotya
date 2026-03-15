---
phase: 05-public-api-stabilization
plan: 03
subsystem: api
tags: [go, public-api, opaque-types, godoc, yang]

# Dependency graph
requires:
  - phase: 05-02
    provides: ParseError/Diagnostic types and structured parser diagnostics
provides:
  - Opaque ASTModule struct with Name()/Keyword()/Argument() methods
  - Opaque Module struct with Schema() *schema.Module accessor
  - CompileOptions struct eliminating compiler package import from callers
  - Clean godoc on all exported symbols in gotya.go
  - Stable public API ready for v1 tagging
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Opaque wrapper struct over internal ast.Module hides all BaseNode/SubStatements methods from public API"
    - "Schema accessor pattern: Module.Schema() returns *schema.Module for use with generator packages"
    - "CompileOptions mirrors internal compiler.Options fields at public boundary — callers import zero internal packages"

key-files:
  created: []
  modified:
    - gotya.go
    - cmd/gotya/main.go

key-decisions:
  - "ASTModule changed from type alias (= ast.Module) to defined struct with unexported *ast.Module field — blocks access to BaseNode/SubStatements from external packages"
  - "Keyword() and Argument() exposed on *ASTModule — callers need to distinguish module from submodule (Keyword) and cache by name (Argument); both are legitimate public API"
  - "CompileOptions has no Loader field — loader is an internal CLI concern not exposed at v1 public API level; cmd/gotya uses loader.Load() directly bypassing gotya.Compile"
  - "Removed loader.astCache manual population in main.go — loader.Load->LoadAST already caches AST from disk; manual population was defensive and incompatible with opaque ASTModule type"
  - "cmd/gotya retains direct schema package import — cmd is an internal CLI binary not a public API consumer; schema.Module usage in generateGo/generateProtobuf is legitimate"

patterns-established:
  - "Opaque struct pattern: wrap internal type in exported struct with explicit public methods, zero field access"

requirements-completed: [API-03, API-04]

# Metrics
duration: 2min
completed: 2026-03-15
---

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
