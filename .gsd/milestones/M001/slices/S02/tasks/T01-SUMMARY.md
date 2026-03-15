# T01: Compiler Error Sentinels

**One-liner:** Typed sentinel errors (Err* vars + errors.Join) replacing string accumulation, with all test assertions migrated to errors.Is.

## What Was Built

Introduced 12 exported sentinel error variables in `compiler/compiler.go` and migrated the entire compiler error infrastructure from `[]string` to `[]error`. The `addError` method now takes `(sentinel error, msg string)` and wraps each error as `fmt.Errorf("%w: %s", sentinel, msg)`. The final `Compile()` return uses `errors.Join(c.errors...)` instead of `strings.Join`. The `Validator` was refactored to call `c.addError` directly with per-category sentinels instead of accumulating strings.

All `assert.Contains(t, err.Error(), ...)` assertions in `compiler_test.go` were replaced with `assert.ErrorIs(t, err, compiler.ErrXxx)`. The `TestCompiler_IdentityrefValidation` table-driven test had its `errorMsg string` field replaced with `wantSentinel error`.

## Verification Results

- `go test ./compiler/... -race -count=1`: PASS
- `grep -c "assert.Contains.*err.Error" compiler/compiler_test.go`: 0
- `grep -c "ErrCircularTypedef\|ErrAugmentNotFound\|ErrDuplicateIdent" compiler/compiler.go`: 21

## Decisions

- Validator refactored to call c.addError(sentinel, msg) directly rather than accumulating []string and returning a joined error
- ErrMaxErrors check uses errors.Is(c.errors[last], ErrMaxErrors) instead of strings.HasPrefix
- validator.Validate() now always returns nil; all errors propagate via c.addError

## Files Modified

- compiler/compiler.go
- compiler/compiler_test.go
- compiler/validator.go
