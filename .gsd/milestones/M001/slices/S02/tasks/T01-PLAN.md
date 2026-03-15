# T01: 02-testing-infrastructure 01

**Slice:** S02 — **Milestone:** M001

## Description

Introduce typed error sentinels in the compiler package and migrate all string-matching
error assertions in compiler_test.go to errors.Is / errors.As.

Purpose: Tests must not be coupled to error message wording. Refactoring a message must
not break a test. Sentinels provide stable, typed identities for error conditions.

Output:
- compiler/compiler.go — exported Err* vars, c.errors changed from []string to []error,
  addError signature changed to addError(sentinel error, msg string), Compile() returns
  errors.Join(c.errors...)
- compiler/compiler_test.go — all assert.Contains(t, err.Error(), ...) replaced with
  assert.ErrorIs(t, err, compiler.ErrXxx), table-driven tests updated to wantSentinel field

## Must-Haves

- [ ] "A test can assert on a compiler error kind using errors.Is(err, compiler.ErrCircularTypedef) without checking any string"
- [ ] "All existing compiler_test.go assertions that used assert.Contains on err.Error() now use assert.ErrorIs"
- [ ] "go test ./compiler/... passes with the race detector enabled"

## Files

- `compiler/compiler.go`
- `compiler/compiler_test.go`
