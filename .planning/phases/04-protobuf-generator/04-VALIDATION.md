---
phase: 4
slug: protobuf-generator
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-15
---

# Phase 4 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — stdlib go test |
| **Quick run command** | `go test ./generator/protobuf/... -count=1` |
| **Full suite command** | `go test ./... -count=1` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./generator/protobuf/... -count=1`
- **After every plan wave:** Run `go test ./... -count=1`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 04-01-01 | 01 | 0 | PBGEN-02 | unit stub | `go test ./generator/protobuf/... -run TestAnyDataAnyXML` | ❌ W0 | ⬜ pending |
| 04-01-02 | 01 | 0 | PBGEN-03 | unit stub | `go test ./generator/protobuf/... -run TestRPCService\|TestActionService` | ❌ W0 | ⬜ pending |
| 04-01-03 | 01 | 0 | PBGEN-04 | unit stub | `go test ./generator/protobuf/... -run TestCELPathValidation` | ❌ W0 | ⬜ pending |
| 04-02-01 | 02 | 1 | PBGEN-02 | unit GREEN | `go test ./generator/protobuf/... -run TestAnyDataAnyXML` | ✅ | ⬜ pending |
| 04-03-01 | 03 | 1 | PBGEN-03 | unit GREEN | `go test ./generator/protobuf/... -run TestRPCService\|TestActionService` | ✅ | ⬜ pending |
| 04-04-01 | 04 | 1 | PBGEN-04 | unit GREEN | `go test ./generator/protobuf/... -run TestCELPathValidation` | ✅ | ⬜ pending |
| 04-05-01 | 05 | 2 | PBGEN-01 | manual | N/A — document creation | N/A | ⬜ pending |
| 04-06-01 | 06 | 2 | PBGEN-02,03,04 | golden | `go test ./generator/protobuf/... -run TestGoldenProto` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `generator/protobuf/generator_anydata_test.go` — RED stubs for PBGEN-02 (anydata/anyxml → google.protobuf.Any)
- [ ] `generator/protobuf/generator_rpc_test.go` — RED stubs for PBGEN-03 (rpc/action → service block)
- [ ] `generator/protobuf/generator_cel_validation_test.go` — RED stubs for PBGEN-04 (CEL path validation error)
- [ ] `generator/protobuf/generator_golden_test.go` — golden file test harness with -update flag (initially empty, golden files generated after implementation)

*All proto generator test infrastructure is new — no existing test files cover the PBGEN requirements.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| docs/proto-coverage.md exists and is complete | PBGEN-01 | Document creation — no automated assertion needed | Verify file exists at `docs/proto-coverage.md`; check all RFC 7950 §7 statements are listed with support status |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
