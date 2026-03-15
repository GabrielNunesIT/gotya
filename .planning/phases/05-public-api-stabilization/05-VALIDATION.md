---
phase: 5
slug: public-api-stabilization
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-15
---

# Phase 5 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | testify v1.11.1 (assert + require packages) |
| **Config file** | none — standard `go test ./...` |
| **Quick run command** | `go test . -count=1` |
| **Full suite command** | `go test ./... -count=1` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test . -count=1`
- **After every plan wave:** Run `go test ./... -count=1`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 5-01-01 | 01 | 1 | API-01 | unit | `go test . -run TestParse_Error -count=1` | ❌ Wave 0 | ⬜ pending |
| 5-01-02 | 01 | 1 | API-01 | unit | `go test . -run TestParse_ErrorsAs -count=1` | ❌ Wave 0 | ⬜ pending |
| 5-01-03 | 01 | 1 | API-02 | unit | `go test . -run TestParse_MultipleErrors -count=1` | ❌ Wave 0 | ⬜ pending |
| 5-02-01 | 02 | 1 | API-03 | compile-time | `go build ./...` | ❌ Wave 0 | ⬜ pending |
| 5-02-02 | 02 | 1 | API-03 | unit | `go test . -run TestCompile_OpaqueReturn -count=1` | ❌ Wave 0 | ⬜ pending |
| 5-02-03 | 02 | 1 | API-03 | unit | `go test . -run TestASTModule_Name -count=1` | ❌ Wave 0 | ⬜ pending |
| 5-03-01 | 03 | 2 | API-04 | static | `grep -n 'TODO\|NOTE:\|Why:\|demonstration' gotya.go` returns empty | ❌ Wave 0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `gotya_test.go` — black-box test stubs covering API-01, API-02, API-03 (package `gotya_test`)
- [ ] No framework install needed — testify already in go.mod

*Existing test infrastructure covers the framework; only the test file needs to be created.*

---

## Manual-Only Verifications

*All phase behaviors have automated verification.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
