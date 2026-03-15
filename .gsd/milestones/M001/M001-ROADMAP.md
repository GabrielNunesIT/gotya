# M001: Migration

**Vision:** gotya is a Go library and CLI tool for working with YANG data models (RFC 7950).

## Success Criteria


## Slices

- [x] **S01: Compiler Correctness** `risk:medium` `depends:[]`
  > After this: Write failing test stubs for all 8 behaviors Phase 1 will implement.
- [x] **S02: Testing Infrastructure** `risk:medium` `depends:[S01]`
  > After this: Introduce typed error sentinels in the compiler package and migrate all string-matching
error assertions in compiler_test.
- [x] **S03: Go Generator** `risk:medium` `depends:[S02]`
  > After this: Write failing test stubs (RED phase) for all four Go generator gaps.
- [x] **S04: Protobuf Generator** `risk:medium` `depends:[S03]`
  > After this: Write failing test stubs (TDD RED) for all three PBGEN implementation gaps — anydata/anyxml, rpc/action/notification service blocks, and CEL path validation — plus the golden file test harness.
- [x] **S05: Public Api Stabilization** `risk:medium` `depends:[S04]`
  > After this: Write failing test stubs for all Phase 5 public API behaviors.
- [x] **S06: V1 Gap Closure** `risk:medium` `depends:[S05]`
  > After this: Refactor compiler/validator.
