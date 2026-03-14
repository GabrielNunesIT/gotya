# Roadmap: gotya

## Overview

gotya is a brownfield Go library with a functioning parse→compile→generate pipeline. The
path to v1 follows a strict dependency chain: first harden the compiler so it cannot panic
or loop on any input, then add testing infrastructure so every subsequent change is captured
by tests, then fill the Go generator feature gaps, then fill the Protobuf generator feature
gaps, and finally stabilize the public API before tagging v1. Each phase delivers a
coherent, independently verifiable capability that unblocks the next.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Compiler Correctness** - Eliminate all crash-class bugs so the compiler handles any malformed or pathological YANG input gracefully (completed 2026-03-14)
- [x] **Phase 2: Testing Infrastructure** - Add corpus smoke test, golden file harness, and structured error sentinels so every future change is locked in by tests (completed 2026-03-14)
- [ ] **Phase 3: Go Generator** - Audit and fill Go generator gaps so all YANG statement types produce valid, correct Go output
- [ ] **Phase 4: Protobuf Generator** - Audit and fill Protobuf generator gaps so all relevant YANG constructs produce valid Proto output
- [ ] **Phase 5: Public API Stabilization** - Harden the public API surface before v1 tag so it can be depended on without a future major version bump

## Phase Details

### Phase 1: Compiler Correctness
**Goal**: The compiler handles any YANG input — valid, malformed, or pathological — without panicking, infinite-looping, or emitting debug output
**Depends on**: Nothing (first phase)
**Requirements**: COMP-01, COMP-02, COMP-03, COMP-04, COMP-05, COMP-06, COMP-07
**Success Criteria** (what must be TRUE):
  1. Feeding any malformed YANG file (nil augment path, duplicate identifier, bad refine path) through the compiler returns an error and never panics or crashes the process
  2. A YANG module with a circular typedef (`A uses B uses A`) produces a compile error with a useful message, not a stack overflow
  3. Two mutually-importing YANG modules (`A imports B imports A`) produce a compile error, not infinite recursion
  4. A YANG file with unresolvable augments produces a compile error that lists the unresolved augment paths, not a silently incomplete schema
  5. Running the compiler against any input produces no output on stdout or stderr unless an error or warning is explicitly requested by the caller
**Plans**: 4 plans

Plans:
- [x] 01-01-PLAN.md — Write failing test stubs for all 8 Phase 1 behaviors (Wave 1) (completed 2026-03-14)
- [ ] 01-02-PLAN.md — Remove debug printfs, nil-guard findNode, fix AddChild error propagation (COMP-01, COMP-05, COMP-06)
- [ ] 01-03-PLAN.md — Add circular import guard to DirectoryLoader (COMP-03)
- [ ] 01-04-PLAN.md — Circular typedef visited set, augment loop cap, MaxErrors bound (COMP-02, COMP-04, COMP-07)

### Phase 2: Testing Infrastructure
**Goal**: Every compiler and generator behavior is verifiable by a test that does not depend on string matching or manual inspection
**Depends on**: Phase 1
**Requirements**: TEST-01, TEST-02, TEST-03
**Success Criteria** (what must be TRUE):
  1. Running `go test ./...` against the YANG corpus in `test/assets/yangs/` completes without any panic and reports no failures — the corpus is a regression gate, not just documentation
  2. Generating Go output from any valid YANG module and passing the result to `go/format.Source()` succeeds — invalid generated Go is caught at generation time and reported as a generation error
  3. A test that asserts on a compiler error can use a typed sentinel value (e.g., `errors.Is(err, compiler.ErrCircularTypedef)`) instead of checking whether an error string contains a substring
**Plans**: 3 plans

Plans:
- [ ] 02-01-PLAN.md — Error sentinels: change c.errors to []error, add Err* vars, migrate compiler_test.go to errors.Is (TEST-03)
- [ ] 02-02-PLAN.md — go/format.Source() in GenerateDevice + TestGoGenerator_FormatValidation (TEST-02)
- [ ] 02-03-PLAN.md — Corpus smoke test: test/corpus_test.go with corpusLoader and TestCorpus (TEST-01)

### Phase 3: Go Generator
**Goal**: The Go generator produces correct, complete output for all YANG statement types encountered in real-world OpenConfig and standard YANG modules
**Depends on**: Phase 2
**Requirements**: GOGEN-01, GOGEN-02, GOGEN-03, GOGEN-04
**Success Criteria** (what must be TRUE):
  1. A YANG module that defines an identity hierarchy and uses `identityref` leaf types generates a typed Go `const` block for that identity, not a `*string` field — the const values are usable in switch statements without string comparison
  2. A YANG module containing `anydata` or `anyxml` nodes generates a Go field (e.g., `json.RawMessage`) for each — no nodes are silently dropped from the generated struct
  3. A YANG module containing `rpc`, `action`, or `notification` statements generates typed Go request/response structs for each — input and output containers appear as nested struct types accessible to the caller
  4. A vendor YANG module that applies deviations generates Go output that reflects the deviated schema (removed nodes absent, replaced types updated) — the generated code matches what the device actually supports
**Plans**: TBD

### Phase 4: Protobuf Generator
**Goal**: The Protobuf generator produces correct, complete output for all YANG constructs relevant to a network management gRPC service, with all gaps explicitly documented
**Depends on**: Phase 3
**Requirements**: PBGEN-01, PBGEN-02, PBGEN-03, PBGEN-04
**Success Criteria** (what must be TRUE):
  1. A `docs/proto-coverage.md` document exists listing every RFC 7950 YANG statement and marking it as supported, unsupported, or out-of-scope for the Protobuf generator — a consumer knows exactly what they get before generating
  2. A YANG module containing `anydata` or `anyxml` nodes generates a valid Protobuf field (`google.protobuf.Any` or `bytes`) for each — no nodes are silently dropped from the generated `.proto` file
  3. A YANG module containing `rpc` or `action` statements generates a Protobuf `service` block with typed `rpc` methods referencing the correct request and response message types
  4. After generating a `.proto` file, all CEL annotation paths in the output are validated against the compiled schema — a path that does not correspond to a real schema node produces a generation error before the file is written
**Plans**: TBD

### Phase 5: Public API Stabilization
**Goal**: The public API in `gotya.go` is intentional, documented, and stable — callers can depend on it without anticipating a breaking change before v1 is tagged
**Depends on**: Phase 4
**Requirements**: API-01, API-02, API-03, API-04
**Success Criteria** (what must be TRUE):
  1. Calling `gotya.Parse()` on a YANG file with syntax errors returns a value that can be type-asserted to `*gotya.ParseError` (or unwrapped to one), providing file name, line number, column number, and message — not `os.ErrInvalid`
  2. Calling `gotya.Parse()` on a YANG file with multiple errors returns all parse diagnostics, not just the first — a caller can present the full error list to the user without re-parsing
  3. Every exported symbol in `gotya.go` is either a deliberate public API or unexported — no internal package types appear in any exported function signature, and the file contains no `// TODO`, `// placeholder`, or `// for demonstration` comments
**Plans**: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4 → 5

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Compiler Correctness | 4/4 | Complete    | 2026-03-14 |
| 2. Testing Infrastructure | 3/3 | Complete   | 2026-03-14 |
| 3. Go Generator | 0/TBD | Not started | - |
| 4. Protobuf Generator | 0/TBD | Not started | - |
| 5. Public API Stabilization | 0/TBD | Not started | - |
