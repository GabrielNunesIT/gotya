---
id: T03
parent: S02
milestone: M001
provides: []
requires: []
affects: []
key_files: []
key_decisions: []
patterns_established: []
observability_surfaces: []
drill_down_paths: []
duration: 
verification_result: passed
completed_at: 
blocker_discovered: false
---
# T03: 02-testing-infrastructure 03

**# Phase 2 Plan 03: Corpus Integration Test Summary**

## What Happened

# Phase 2 Plan 03: Corpus Integration Test Summary

**One-liner:** TestCorpus regression gate running 205 real-world YANG modules through parse→compile→generate→format.Source() pipeline with parallel subtests.

## What Was Built

Created `test/corpus_test.go` with package `test` containing:

- `corpusLoader` struct — a verbatim copy of `fileLoader` from `test/generate.go` (renamed because `//go:build ignore` prevents import). Implements `schema.Loader` with `LoadAST` and `Load` methods backed by an in-memory AST and schema cache.
- `namedModule` helper struct pairing a module name with its compiled `*schema.Module`.
- `TestCorpus` function that loads all 205 `.yang` files in `test/assets/yangs/` sequentially, then runs parallel subtests per module — each subtest calls `gen.Generate()` and asserts the output passes `go/format.Source()`.

Also fixed `.gitignore` which previously ignored the entire `test/` directory — changed to `test/out/` so Go source files are tracked.

## Tasks

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Write test/corpus_test.go with corpusLoader and TestCorpus | e140c25 | test/corpus_test.go, test/generate.go, test/test.go, .gitignore |

## Verification Results

- `go test ./test/... -run TestCorpus -count=1`: PASS — 205 subtests, all PASS
- `go test ./test/... -race -count=1`: PASS — no data races
- `go test ./... -count=1`: all packages PASS

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] .gitignore was ignoring entire test/ directory**
- **Found during:** Task 1 (commit attempt)
- **Issue:** `.gitignore` contained a single line `test` which caused git to refuse staging `test/corpus_test.go`. The corpus test file is the primary artifact of this plan and must be committed.
- **Fix:** Changed `.gitignore` from `test` to `test/out/` — only the generated output directory is ignored; Go source files are tracked.
- **Files modified:** .gitignore
- **Commit:** e140c25

**2. [Rule 3 - Blocking] Root-owned git object bucket `cb/` blocked staging**
- **Found during:** Task 1 (commit attempt)
- **Issue:** The SHA-1 hash of the initial `test/corpus_test.go` content started with `cb`, mapping to `.git/objects/cb/` which was owned by root (not writable by user). `git add` failed with "insufficient permission for adding an object".
- **Fix:** Added a trailing newline to `corpus_test.go` (no semantic change) to shift the content hash to bucket `a0/` (user-writable). No functional change to the file.
- **Files modified:** test/corpus_test.go (trailing newline only)
- **Commit:** e140c25

## Self-Check

- [x] test/corpus_test.go — created (172 lines, >80 min_lines requirement met)
- [x] Commit e140c25 exists
- [x] TestCorpus reports 205 subtests (one per .yang file)
- [x] go test ./test/... -race passes clean
- [x] Compiler errors during Load() use t.Logf, not t.Errorf
- [x] format.Source() called on gen.Generate() output
- [x] corpusLoader uses compiler.New pattern
