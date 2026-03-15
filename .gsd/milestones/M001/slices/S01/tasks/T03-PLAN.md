# T03: 01-compiler-correctness 03

**Slice:** S01 — **Milestone:** M001

## Description

Add an in-progress guard to DirectoryLoader.Load() so circular module imports return an error instead of recursing infinitely, and implement the TestLoader_CircularImport test.

Purpose: COMP-03 — the loader's schemaCache is written after compilation completes, so a circular import causes Load("A") → Compile("A") → Load("B") → Compile("B") → Load("A") again with nothing in the cache, producing infinite recursion. The fix is a single in-progress set checked at the top of Load().
Output: loader.go with inProgress guard; loader_test.go with TestLoader_CircularImport passing GREEN.

## Must-Haves

- [ ] "Loading a module whose import graph contains a cycle returns an error, not infinite recursion"
- [ ] "TestLoader_CircularImport passes GREEN"
- [ ] "Non-circular modules still load and compile correctly after the guard is added"

## Files

- `cmd/gotya/loader.go`
- `cmd/gotya/loader_test.go`
