# T03: 05-public-api-stabilization 03

**Slice:** S05 — **Milestone:** M001

## Description

Convert ASTModule from alias to opaque struct, wrap schema.Module in gotya.Module, fix cmd/gotya call sites, and clean all godoc comments.

Purpose: Satisfies API-03 (no internal type leakage through exported signatures) and API-04 (no placeholder comments). After this plan, gotya.go presents a stable, well-documented public API — ready to tag v1.

Output: Opaque ASTModule and Module types in gotya.go; updated cmd/gotya/main.go; clean godoc throughout gotya.go.

## Must-Haves

- [ ] "gotya.ASTModule is an opaque struct — callers cannot call ast.Module methods directly"
- [ ] "gotya.Module is an opaque struct with Schema() *schema.Module accessor"
- [ ] "Compile() returns ([]*gotya.Module, error) — no schema package import needed by callers"
- [ ] "gotya.go contains no TODO, NOTE:, Why:, or 'demonstration' strings"
- [ ] "All exported symbols have godoc comments beginning with the symbol name"
- [ ] "go build ./... succeeds — cmd/gotya/main.go updated to use .Schema() and .Keyword()"

## Files

- gotya.go
- cmd/gotya/main.go
