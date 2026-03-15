# S01: Compiler Correctness

**Goal:** Write failing test stubs for all 8 behaviors Phase 1 will implement.
**Demo:** Write failing test stubs for all 8 behaviors Phase 1 will implement.

## Must-Haves


## Tasks

- [x] **T01: 01-compiler-correctness 01** `est:1min`
  - Write failing test stubs for all 8 behaviors Phase 1 will implement.

Purpose: Nyquist compliance — every subsequent implementation task must have an automated verify command that runs against a pre-existing test. Stubs fail now (RED) and turn green after each fix is applied.
Output: Two test files containing 8 test functions that compile, run, and fail with t.Fatal("not yet implemented") or equivalent.
- [x] **T02: 01-compiler-correctness 02** `est:5min`
  - Remove the two debug printf statements from compiler.go, add a nil-return guard where findNode returns nil inside the augment-resolution loop, and fix all five ignored AddChild errors.

Purpose: COMP-06 (no stdout/stderr pollution), COMP-01 (nil-safety at findNode call site), and COMP-05 (AddChild error propagation) are all mechanical changes to co-located sites in compiler.go. Doing them together avoids touching the same file twice.
Output: compiler.go with no fmt.Printf calls, nil-safe findNode usage, and all AddChild errors surfaced.
- [x] **T03: 01-compiler-correctness 03** `est:1min`
  - Add an in-progress guard to DirectoryLoader.Load() so circular module imports return an error instead of recursing infinitely, and implement the TestLoader_CircularImport test.

Purpose: COMP-03 — the loader's schemaCache is written after compilation completes, so a circular import causes Load("A") → Compile("A") → Load("B") → Compile("B") → Load("A") again with nothing in the cache, producing infinite recursion. The fix is a single in-progress set checked at the top of Load().
Output: loader.go with inProgress guard; loader_test.go with TestLoader_CircularImport passing GREEN.
- [x] **T04: 01-compiler-correctness 04** `est:3min`
  - Implement the three remaining compiler fixes: circular typedef detection in getType(), augment loop iteration cap, and bounded error accumulation via MaxErrors. Turn the final three failing test stubs GREEN.

Purpose: COMP-02 prevents stack overflow from user-crafted circular typedefs; COMP-04 prevents the augment loop from spinning indefinitely on pathological input; COMP-07 bounds memory use and provides predictable behavior when a YANG file contains hundreds of validation errors.
Output: compiler.go with all three fixes applied; all 8 Phase 1 tests GREEN.

## Files Likely Touched

- `compiler/compiler_test.go`
- `cmd/gotya/loader_test.go`
- `compiler/compiler.go`
- `cmd/gotya/loader.go`
- `cmd/gotya/loader_test.go`
- `compiler/compiler.go`
- `compiler/compiler_test.go`
