---
status: complete
phase: 05-public-api-stabilization
source: [05-01-SUMMARY.md, 05-02-SUMMARY.md, 05-03-SUMMARY.md]
started: 2026-03-15T16:30:00Z
updated: 2026-03-15T16:45:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Parse returns structured error, not os.ErrInvalid
expected: When you call gotya.Parse() with invalid YANG, the returned error can be used with errors.As to get a *gotya.ParseError value. The ParseError has an Errors []Diagnostic field. Each Diagnostic has File, Line, Column, and Message fields populated. The error is NOT os.ErrInvalid.
result: pass

### 2. ParseFile fills Diagnostic.File from path
expected: When you call gotya.ParseFile("/some/path/foo.yang") on a file with YANG errors, the returned *ParseError has Diagnostics where the File field is set to the path you passed in (e.g. "foo.yang" or the cleaned path). Calling gotya.Parse() directly leaves File as empty string.
result: pass

### 3. Multiple parse errors all returned
expected: When you call gotya.Parse() on YANG with more than one syntax error, the returned *ParseError contains ALL the errors — not just the first one. Iterating parseErr.Errors gives you each diagnostic individually.
result: pass

### 4. Compile returns opaque []*gotya.Module
expected: After a successful Parse() + Compile() call, the returned slice is []*gotya.Module — you do NOT need to import github.com/gotya/gotya/schema to work with the result. You can call .Schema() on each element to get the underlying schema if you need it for generators.
result: pass

### 5. ASTModule is opaque with limited method set
expected: The value returned by gotya.Parse() is a *gotya.ASTModule. Calling .Name() on it returns the module name. Calling .Keyword() returns "module" or "submodule". You cannot call internal ast.Module methods like SubStatements() or TokenLiteral() — they are not exposed.
result: pass

### 6. gotya.go has clean godoc, no placeholders
expected: Opening gotya.go shows standard godoc comments on all exported symbols (each comment starts with the symbol name, e.g. "// Parse parses a YANG module..."). There are no lines containing TODO, NOTE:, Why:, "demonstration", "facade", "simplistic", or "placeholder". The file looks like a finished v1 API, not a prototype.
result: pass

## Summary

total: 6
passed: 6
issues: 0
pending: 0
skipped: 0

## Gaps

[none yet]
