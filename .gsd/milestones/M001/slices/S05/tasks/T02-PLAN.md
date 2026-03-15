# T02: 05-public-api-stabilization 02

**Slice:** S05 — **Milestone:** M001

## Description

Add structured parser diagnostics and replace os.ErrInvalid with *ParseError.

Purpose: Satisfies API-01 and API-02 — callers can now type-assert parse errors to get structured file/line/column/message data, and all parse errors are propagated (not just the first).
Output: Diagnostic and ParseError types in gotya.go; Diagnostics() method added to parser.Parser; Parse() and ParseFile() rewritten to use them.

## Must-Haves

- [ ] "Parse() on malformed YANG returns *ParseError — not os.ErrInvalid"
- [ ] "errors.As(err, &parseErr) succeeds on a Parse() error, giving access to Errors []Diagnostic"
- [ ] "Parse() on multi-error YANG returns all diagnostics, not just the first"
- [ ] "ParseFile() fills Diagnostic.File from the path argument"
- [ ] "Diagnostic has File, Line, Column, Message fields with non-zero Line values"

## Files

- `parser/parser.go`
- `gotya.go`
