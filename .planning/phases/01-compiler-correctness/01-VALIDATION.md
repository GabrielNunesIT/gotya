---
phase: 1
slug: compiler-correctness
status: draft
nyquist_compliant: true
wave_0_complete: true
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
| 1-01-T1 | 01-01 | 1 | COMP-01..07 | unit stub | `go test ./compiler/ -run "TestCompiler_MalformedAugmentPath\|TestCompiler_MalformedRefinePath\|TestCompiler_CircularTypedef\|TestCompiler_UnresolvableAugment\|TestCompiler_DuplicateRPCInput\|TestCompiler_NoDebugOutput\|TestCompiler_MaxErrors" -count=1 2>&1 \| grep -E "FAIL\|not yet implemented"` | ❌ Wave 0 | ⬜ pending |
| 1-01-T2 | 01-01 | 1 | COMP-03 | unit stub | `go test ./cmd/gotya/ -run TestLoader_CircularImport -count=1 2>&1 \| grep -E "FAIL\|not yet implemented"` | ❌ Wave 0 | ⬜ pending |
| 1-02-T1 | 01-02 | 2 | COMP-01, COMP-05, COMP-06 | unit | `go build ./compiler/ && grep -c "fmt\.Printf" compiler/compiler.go \| grep "^0$" && grep -c "_ = .*AddChild" compiler/compiler.go \| grep "^0$"` | ❌ W0 | ⬜ pending |
| 1-02-T2 | 01-02 | 2 | COMP-01, COMP-05, COMP-06 | unit | `go test ./compiler/ -run "TestCompiler_NoDebugOutput\|TestCompiler_MalformedAugmentPath\|TestCompiler_MalformedRefinePath\|TestCompiler_DuplicateRPCInput" -count=1 -v 2>&1 \| grep -E "PASS\|FAIL"` | ❌ W0 | ⬜ pending |
| 1-03-T1 | 01-03 | 2 | COMP-03 | unit | `go build ./cmd/gotya/` | ❌ W0 | ⬜ pending |
| 1-03-T2 | 01-03 | 2 | COMP-03 | unit | `go test ./cmd/gotya/ -run TestLoader_CircularImport -count=1 -v 2>&1 \| grep -E "PASS\|FAIL"` | ❌ W0 | ⬜ pending |
| 1-04-T1 | 01-04 | 3 | COMP-02, COMP-04, COMP-07 | unit | `go build ./compiler/ && grep -c "c\.errors = append" compiler/compiler.go \| grep "^0$"` | ❌ W0 | ⬜ pending |
| 1-04-T2 | 01-04 | 3 | COMP-02, COMP-04, COMP-07 | unit | `go test ./compiler/ -run "TestCompiler_CircularTypedef\|TestCompiler_UnresolvableAugment\|TestCompiler_MaxErrors" -count=1 -v 2>&1 \| grep -E "PASS\|FAIL"` | ❌ W0 | ⬜ pending |

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

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 10s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved
