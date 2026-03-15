# T03: Public API Opaque Types

**ASTModule and Module converted to opaque structs with explicit method sets, eliminating internal type leakage through gotya.go public signatures**

## What Was Built

- Replaced `type ASTModule = ast.Module` alias with defined struct wrapping `*ast.Module` — callers can only call Name()/Keyword()/Argument()
- Added `Module` opaque struct with `Schema() *schema.Module` accessor — Compile() no longer returns schema package type at boundary
- Added `CompileOptions` struct — callers configure compilation without importing `compiler` package
- Fixed `cmd/gotya/main.go`: removed manual `loader.astCache` population that became type-incompatible
- Verified all existing tests pass with zero changes to test files

## Decisions

- ASTModule changed from type alias (= ast.Module) to defined struct with unexported *ast.Module field
- Keyword() and Argument() exposed on *ASTModule — callers need to distinguish module from submodule
- CompileOptions has no Loader field — loader is an internal CLI concern not exposed at v1 public API level
- Removed loader.astCache manual population in main.go — loader.Load→LoadAST already caches AST from disk
- cmd/gotya retains direct schema package import — cmd is an internal CLI binary not a public API consumer

## Files Modified

- gotya.go
- cmd/gotya/main.go

## Verification

- `go test ./... -count=1` exits 0 across all packages
- `go build ./...` exits 0 including cmd/gotya
- Zero TODO/NOTE:/Why:/demonstration strings in gotya.go
