---
phase: 02-testing-infrastructure
plan: 02
subsystem: generator/golang
tags: [go/format, syntax-validation, generator, tdd]
dependency_graph:
  requires: [02-01]
  provides: [generator.GenerateDevice format-validated output]
  affects: [generator/golang/generator.go, generator/golang/generator_test.go]
tech_stack:
  added: [go/format]
  patterns: [buffer-then-format, format.Source pre-write validation]
key_files:
  created: []
  modified:
    - generator/golang/generator.go
    - generator/golang/generator_test.go
decisions:
  - "GenerateDevice buffers all output to bytes.Buffer internally, applies format.Source on the complete buffer, then writes formatted bytes to w — no partial writes on error"
  - "Only GenerateDevice gets format.Source treatment; Generate (single-module) does not — per pre-existing research decision"
metrics:
  duration: 3min
  completed: 2026-03-14
  tasks_completed: 2
  files_modified: 2
---

# Phase 2 Plan 02: Go Format Validation Summary

**One-liner:** go/format.Source() integrated into GenerateDevice output path — syntactically invalid generated Go surfaces as a generation error, never reaches the caller's Writer.

## What Was Built

`GenerateDevice` now buffers all generated output into an internal `bytes.Buffer` instead of writing directly to the caller's `io.Writer`. After generation completes, `format.Source(buf.Bytes())` is called on the complete buffer. If the formatter reports a syntax error, `GenerateDevice` returns `fmt.Errorf("generated Go has invalid syntax: %w", err)` and nothing is written to `w`. On success, the gofmt-formatted bytes are written to `w`.

`TestGoGenerator_FormatValidation` was added to `generator_test.go`. It calls `GenerateDevice` with a minimal module, asserts no error, then independently passes the output through `format.Source` to confirm the output is canonical Go.

## Tasks

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Write failing test for format validation (RED) | 5148790 | generator/golang/generator_test.go |
| 2 | Add go/format.Source() to GenerateDevice (GREEN) | 383adab | generator/golang/generator.go |

## Verification Results

- `go test ./generator/golang/... -race -count=1`: PASS
- `grep "format.Source" generator/golang/generator.go`: matched (2 lines)
- `go vet ./generator/golang/...`: PASS
- No regressions in existing `device_test.go` or `generator_test.go` tests

## Deviations from Plan

None - plan executed exactly as written.

## Self-Check

- [x] generator/golang/generator.go — modified (buf + format.Source + go/format import)
- [x] generator/golang/generator_test.go — TestGoGenerator_FormatValidation added
- [x] Commit 5148790 exists
- [x] Commit 383adab exists
- [x] go test ./generator/golang/... -race passes
- [x] format.Source present in generator.go
