# T02: 02-testing-infrastructure 02

**Slice:** S02 — **Milestone:** M001

## Description

Integrate go/format.Source() into the Go generator's GenerateDevice output path so that
syntactically invalid generated Go is caught at generation time and surfaces as an error
— never reaching the caller's io.Writer.

Purpose: Invalid generated Go must not silently reach the user's filesystem. Catching it
at the generation boundary gives a precise error with line/column info from the formatter.

Output:
- generator/golang/generator.go — GenerateDevice calls format.Source(buf.Bytes()) before
  w.Write; returns error if format fails
- generator/golang/generator_test.go — TestGoGenerator_FormatValidation test that asserts
  an error is returned (not written) when generation would produce bad syntax

## Must-Haves

- [ ] "Calling GenerateDevice() with a module that would produce syntactically invalid Go returns an error — never writes malformed output to the caller's Writer"
- [ ] "go test ./generator/golang/... passes with the race detector enabled"

## Files

- `generator/golang/generator.go`
- `generator/golang/generator_test.go`
