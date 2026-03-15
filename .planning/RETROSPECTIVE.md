# Project Retrospective

*A living document updated after each milestone. Lessons feed forward into future planning.*

## Milestone: v1.0 — MVP

**Shipped:** 2026-03-15
**Phases:** 6 | **Plans:** 23 | **Timeline:** 14 days (2026-03-01 → 2026-03-15)

### What Was Built
- Crash-safe compiler: nil panic guards, circular typedef/import detection, bounded error accumulation, augment loop cap
- Testing infrastructure: corpus smoke test, `go/format` validation, 12 typed error sentinels with `errors.Is` assertions throughout
- Go generator completeness: identityref typed const blocks, anydata/anyxml fields, RPC/action/notification structs, deviation pre-pass
- Protobuf generator completeness: anydata → `google.protobuf.Any`, RPC/action service blocks, CEL path validation, RFC 7950 coverage matrix
- Stable public API: `ParseError`/`Diagnostic` types, opaque `ASTModule`/`Module` wrappers, multi-error diagnostics, clean godoc

### What Worked
- **TDD throughout**: Writing failing tests first (Wave 1 stubs) in every phase prevented scope drift and gave clear done-criteria. No phase completed without tests going GREEN.
- **Milestone audit before tagging**: The post-Phase-5 audit caught two real gaps (TEST-03 assertion migration wasn't actually done; Compile() dropped multi-module errors). Phase 6 closed both cleanly.
- **Phase dependency chain**: Strict ordering (correctness → tests → generators → API) meant each phase built on a solid foundation. No backtracking to fix earlier phases.
- **Typed sentinels before test migration**: Phase 6-01 refactoring Validator to emit per-category sentinels via `addError()` before Phase 6-03 migrated tests was the right order — tests could only be migrated after the sentinel contracts were solid.

### What Was Inefficient
- **Phase 2 SUMMARY overclaimed**: The Phase 2 summary reported that compiler_test.go had 0 `assert.Contains` calls (incorrect). The error wasn't caught until the milestone audit. More rigorous verification at summary time would have avoided Phase 6 entirely.
- **Accomplishments not extracted by CLI**: The gsd-tools `milestone complete` couldn't extract one-liners from SUMMARY.md frontmatter (empty result). Manually written — should have `one_liner` field in SUMMARY frontmatter.

### Patterns Established
- **Wave 1 = RED stubs**: Every phase started with a Wave 1 plan that only wrote failing tests. Execution waves then made them GREEN. This pattern worked consistently across all 6 phases.
- **Audit before tagging**: Run `/gsd:audit-milestone` before declaring done. Auditor caught what verifiers missed.
- **Decimal phase for post-audit gaps**: Phase 6 as gap closure (not renumbering 5 phases) kept history clean while addressing findings.
- **Sentinel-first, migration-second**: When migrating test assertions to structured errors, establish the sentinel infrastructure in a prior plan before migrating all call sites.

### Key Lessons
1. SUMMARY.md claims must be verified against actual file state — not just taken at face value. The Phase 2 overclaim cost one extra phase.
2. The milestone audit is not optional — it found a real bug (Compile() multi-module error loss) that verifiers missed because they only tested single-module flows.
3. Typing error sentinels early (Phase 2) paid dividends throughout — Phase 6 migration was mechanical once the infrastructure was in place.
4. `go/format.Source()` as a generation gate (Phase 2) is a strong quality lever — any generated code bug becomes immediately visible rather than a confusing downstream compile error.

### Cost Observations
- Model mix: ~100% Sonnet 4.6 (balanced profile throughout)
- Sessions: ~15 estimated (based on commit clusters)
- Notable: Parallel agent execution in each phase (wave-based) was efficient — phases with 5 plans completed in 2-3 sessions rather than 5

---

## Cross-Milestone Trends

### Process Evolution

| Milestone | Phases | Plans | Key Change |
|-----------|--------|-------|------------|
| v1.0 | 6 | 23 | First milestone — TDD + audit pattern established |

### Cumulative Quality

| Milestone | Typed Sentinels | Corpus Tests | Generated Code Validated |
|-----------|----------------|--------------|--------------------------|
| v1.0 | 12 | ✓ | ✓ (go/format) |

### Top Lessons (Verified Across Milestones)

1. SUMMARY.md verification must check actual file state, not just accept claimed grep results
2. Milestone audit before tagging catches integration gaps that phase-level verifiers miss
