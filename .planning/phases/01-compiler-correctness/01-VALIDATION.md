---
phase: 1
slug: compiler-correctness
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-14
---

# Phase 1 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing stdlib + stretchr/testify v1.11.1 |
| **Config file** | none (standard `go test`) |
| **Quick run command** | `go test ./compiler/ ./cmd/gotya/ -count=1` |
| **Full suite command** | `go test ./... -count=1` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./compiler/ ./cmd/gotya/ -count=1`
- **After every plan wave:** Run `go test ./... -count=1`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 1-W0-01 | 01 | 0 | COMP-01 | unit stub | `go test ./compiler/ -run TestCompiler_MalformedAugmentPath -count=1` | ❌ Wave 0 | ⬜ pending |
| 1-W0-02 | 01 | 0 | COMP-01 | unit stub | `go test ./compiler/ -run TestCompiler_MalformedRefinePath -count=1` | ❌ Wave 0 | ⬜ pending |
| 1-W0-03 | 01 | 0 | COMP-02 | unit stub | `go test ./compiler/ -run TestCompiler_CircularTypedef -count=1` | ❌ Wave 0 | ⬜ pending |
| 1-W0-04 | 01 | 0 | COMP-03 | unit stub | `go test ./cmd/gotya/ -run TestLoader_CircularImport -count=1` | ❌ Wave 0 | ⬜ pending |
| 1-W0-05 | 01 | 0 | COMP-04 | unit stub | `go test ./compiler/ -run TestCompiler_UnresolvableAugment -count=1` | ❌ Wave 0 | ⬜ pending |
| 1-W0-06 | 01 | 0 | COMP-05 | unit stub | `go test ./compiler/ -run TestCompiler_DuplicateRPCInput -count=1` | ❌ Wave 0 | ⬜ pending |
| 1-W0-07 | 01 | 0 | COMP-06 | unit stub | `go test ./compiler/ -run TestCompiler_NoDebugOutput -count=1` | ❌ Wave 0 | ⬜ pending |
| 1-W0-08 | 01 | 0 | COMP-07 | unit stub | `go test ./compiler/ -run TestCompiler_MaxErrors -count=1` | ❌ Wave 0 | ⬜ pending |
| 1-01-01 | 01 | 1 | COMP-06 | unit | `go test ./compiler/ -run TestCompiler_NoDebugOutput -count=1` | ❌ W0 | ⬜ pending |
| 1-01-02 | 01 | 1 | COMP-07 | unit | `go test ./compiler/ -run TestCompiler_MaxErrors -count=1` | ❌ W0 | ⬜ pending |
| 1-02-01 | 02 | 2 | COMP-01 | unit | `go test ./compiler/ -run TestCompiler_MalformedAugmentPath -count=1` | ❌ W0 | ⬜ pending |
| 1-02-02 | 02 | 2 | COMP-01 | unit | `go test ./compiler/ -run TestCompiler_MalformedRefinePath -count=1` | ❌ W0 | ⬜ pending |
| 1-02-03 | 02 | 2 | COMP-02 | unit | `go test ./compiler/ -run TestCompiler_CircularTypedef -count=1` | ❌ W0 | ⬜ pending |
| 1-02-04 | 02 | 2 | COMP-04 | unit | `go test ./compiler/ -run TestCompiler_UnresolvableAugment -count=1` | ❌ W0 | ⬜ pending |
| 1-03-01 | 03 | 2 | COMP-05 | unit | `go test ./compiler/ -run TestCompiler_DuplicateRPCInput -count=1` | ❌ W0 | ⬜ pending |
| 1-04-01 | 04 | 3 | COMP-03 | unit | `go test ./cmd/gotya/ -run TestLoader_CircularImport -count=1` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `compiler/compiler_test.go` — add failing stubs: `TestCompiler_MalformedAugmentPath`, `TestCompiler_MalformedRefinePath`, `TestCompiler_CircularTypedef`, `TestCompiler_UnresolvableAugment`, `TestCompiler_DuplicateRPCInput`, `TestCompiler_NoDebugOutput`, `TestCompiler_MaxErrors`
- [ ] `cmd/gotya/loader_test.go` — create new file; add `TestLoader_CircularImport` stub (requires pair of temp YANG files that mutually import each other)

---

## Manual-Only Verifications

All phase behaviors have automated verification.

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
