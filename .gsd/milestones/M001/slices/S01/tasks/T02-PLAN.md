# T02: 01-compiler-correctness 02

**Slice:** S01 — **Milestone:** M001

## Description

Remove the two debug printf statements from compiler.go, add a nil-return guard where findNode returns nil inside the augment-resolution loop, and fix all five ignored AddChild errors.

Purpose: COMP-06 (no stdout/stderr pollution), COMP-01 (nil-safety at findNode call site), and COMP-05 (AddChild error propagation) are all mechanical changes to co-located sites in compiler.go. Doing them together avoids touching the same file twice.
Output: compiler.go with no fmt.Printf calls, nil-safe findNode usage, and all AddChild errors surfaced.

## Must-Haves

- [ ] "Compiler emits nothing to stdout or stderr on any input (valid or malformed)"
- [ ] "A nil node returned by findNode does not cause a subsequent nil-dereference panic"
- [ ] "All five _ = AddChild(...) sites propagate errors into c.errors"

## Files

- `compiler/compiler.go`
