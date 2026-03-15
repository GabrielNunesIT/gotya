# T05: 04-protobuf-generator 05

**Slice:** S04 — **Milestone:** M001

## Description

Create the RFC 7950 coverage document (PBGEN-01) and generate golden proto files from the corpus to lock in TestGoldenProto GREEN (PBGEN-02, PBGEN-03, PBGEN-04 regression).

Purpose: Delivers the consumer-facing coverage document that documents exactly what the proto generator supports, and establishes the golden file regression baseline so future changes are caught immediately.
Output: docs/proto-coverage.md with all 25 RFC 7950 statements; testdata/proto/ populated; TestGoldenProto GREEN.

## Must-Haves

- [ ] "docs/proto-coverage.md exists and contains a table row for every RFC 7950 §7 statement with columns: YANG statement, RFC section, Proto mapping, status (supported/unsupported/out-of-scope), notes"
- [ ] "All 25 RFC 7950 §7 statements are represented — no statement is omitted or left blank"
- [ ] "testdata/proto/ directory exists with one .proto golden file per loadable corpus YANG module"
- [ ] "go test ./test/... -run TestGoldenProto -count=1 passes GREEN (golden files match generated output)"

## Files

- `docs/proto-coverage.md`
- `testdata/proto/`
