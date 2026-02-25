# Resolved Findings Summary

This document provides a comprehensive summary of all findings from the critical review (`docs/critical-review-2026-02-21.md`) and their resolutions.

**Review Date:** 2026-02-21
**Resolution Date:** 2026-02-25
**Status:** All findings resolved

---

## Executive Summary

The critical review identified **4 highest-priority findings**, **4 medium-priority findings**, and **4 test coverage gaps**. All have been addressed through a systematic implementation plan executed across 5 parallel workstreams.

| Category | Total | Resolved |
|----------|-------|----------|
| Highest-Priority Findings | 4 | 4 |
| Medium-Priority Findings | 4 | 4 |
| Test Coverage Gaps | 4 | 4 |

---

## Highest-Priority Findings

### 1. `quantum.Gate` Architecturally Misleading for Multi-Qubit Usage

**Finding:** The `Gate` interface requires `Apply(q Qubit)` which cannot work correctly for multi-qubit gates. CNOT and Swap gates always error or provide non-physical helper methods.

**Root Cause:** The interface design assumed individual qubit objects, which cannot represent entanglement.

**Resolution:** State-vector-first API adopted. Gate interface simplified to metadata only (`Name()`, `Matrix()`). All gate applications now use `state.ApplyGate(gate, targets...)`.

| Aspect | Details |
|--------|---------|
| ADR | [ADR-0001: State-Vector-First API](adr/0001-state-vector-first-api.md) |
| Commit | [d183844](https://github.com/pjbaur/quantum/commit/d183844) |
| Phase | 1.1 |
| Impact | Breaking - all `gate.Apply(q)` calls must migrate |

---

### 2. Fragile `ExecuteAllParallel` Caller Contract

**Finding:** `ExecuteAllParallel` documents that states must be independent but does not enforce it. Reusing a state across executions can corrupt results under concurrency.

**Root Cause:** No runtime validation of the independence contract.

**Resolution:** Added shared-state detection with `SharedStateError`. The function now fails fast when duplicate state pointers are detected.

| Aspect | Details |
|--------|---------|
| ADR | [ADR-0002: Concurrency Contract](adr/0002-concurrency-contract.md) |
| Commit | [fe19529](https://github.com/pjbaur/quantum/commit/fe19529) |
| Phase | 1.2 |
| Impact | Behavior change - previously silent corruption now returns error |

---

### 3. Sparse Backend Not Drop-In for General Circuits

**Finding:** The sparse backend only handles 1- and 2-qubit gates. Gates above 2 qubits are rejected with misleading error messages.

**Root Cause:** Missing capability declaration and unclear error messaging.

**Resolution:** Added `BackendCapabilities` interface with `SupportsGateQubits()` and `MaxGateQubits()`. Added `UnsupportedOperationError` with actionable guidance.

| Aspect | Details |
|--------|---------|
| ADR | [ADR-0003: Sparse Backend Capability](adr/0003-sparse-backend-capability.md) |
| Commit | [d9c31a7](https://github.com/pjbaur/quantum/commit/d9c31a7) |
| Phase | 1.3 |
| Impact | Better error messages, capability checking API |

---

### 4. Normalization Diagnostics Wrong After Rollback

**Finding:** Both dense and sparse `SetAmplitude` restore old values before returning error, so the error reports post-rollback sum instead of the invalid attempted sum.

**Root Cause:** Error reported after state restoration, losing diagnostic information.

**Resolution:** `NormalizationError` now includes `AttemptedSum` (the invalid value) and `CurrentSum` (the restored value).

| Aspect | Details |
|--------|---------|
| ADR | [ADR-0004: Diagnostics and Constructor Consistency](adr/0004-diagnostics-constructor-consistency.md) |
| Commit | [d183844](https://github.com/pjbaur/quantum/commit/d183844) |
| Phase | 1.4 |
| Impact | Error type extended with new fields |

---

## Medium-Priority Findings

### 1. CLI Help/Behavior Inconsistency

**Finding:** `visual` demo was implemented but omitted from `-demo` help text. Optional parameter was parsed but unused.

**Resolution:**
- Added `visual` to help text
- Removed dead `-param` flag
- Added CLI tests to prevent future drift

| Aspect | Details |
|--------|---------|
| ADR | [ADR-0006: CLI Contract and Drift Prevention](adr/0006-cli-contract-and-drift-prevention.md) |
| Commit | [d1f52b2](https://github.com/pjbaur/quantum/commit/d1f52b2) |
| Phase | 2.1 |

---

### 2. Algorithm Scalability Issues

**Finding:** Grover and Deutsch-Jozsa constructed full dense matrices, causing avoidable memory/time growth.

**Resolution:** Refactored to use direct state-vector transformations instead of dense matrix construction.

| Aspect | Details |
|--------|---------|
| Commit | [3f0e581](https://github.com/pjbaur/quantum/commit/3f0e581) |
| Phase | 2.2 |
| Impact | Significant memory/performance improvement |

---

### 3. Duplicated Gate Matrix Validation Logic

**Finding:** `gateQubitCount` existed in three packages (`circuit`, `state`, `sparsestate`), increasing drift risk.

**Resolution:** Extracted to `quantum.GateQubitCount()` with `InvalidGateMatrixError` type. All packages now use the shared implementation.

| Aspect | Details |
|--------|---------|
| Commit | [82e754f](https://github.com/pjbaur/quantum/commit/82e754f) |
| Phase | 2.3 |

---

### 4. Constructor Inconsistency on Invalid Qubit Counts

**Finding:** `state.New` and sparse `New` silently coerced invalid counts to 1, while `circuit.New` returned an error.

**Resolution:** All constructors now return `InvalidQubitCountError` for invalid counts (<= 0).

| Aspect | Details |
|--------|---------|
| ADR | [ADR-0004: Diagnostics and Constructor Consistency](adr/0004-diagnostics-constructor-consistency.md) |
| Commit | [d7dde57](https://github.com/pjbaur/quantum/commit/d7dde57) |
| Phase | 1.5 |
| Impact | Breaking - code relying on coercion must handle error |

---

## Test Coverage Gaps

### 1. Algorithm Negative Paths Untested

**Finding:** Error cases in `DeutschJozsa` and `Grover` were not covered by tests.

**Resolution:** Added comprehensive negative path tests covering invalid qubit counts, nil oracles, empty marked sets, and out-of-range marked states.

| Aspect | Details |
|--------|---------|
| Commit | [c5082c4](https://github.com/pjbaur/quantum/commit/c5082c4) |
| Phase | 3.1 |

---

### 2. Parallel Shared-State Usage Untested

**Finding:** Tests verified successful independent execution but did not test shared-state misuse.

**Resolution:** Added shared-state misuse tests with `SharedStateError` detection.

| Aspect | Details |
|--------|---------|
| Commit | [fe19529](https://github.com/pjbaur/quantum/commit/fe19529) |
| Phase | 1.2 |

---

### 3. Parallel Race-Focused Tests Missing

**Finding:** No tests validating concurrent independent-state executions under race conditions.

**Resolution:** Added race detection tests for concurrent independent-state executions.

| Aspect | Details |
|--------|---------|
| ADR | [ADR-0005: Regression Prevention Test Strategy](adr/0005-regression-prevention-test-strategy.md) |
| Commit | [3300490](https://github.com/pjbaur/quantum/commit/3300490) |
| Phase | 3.2 |

---

### 4. Visualization Helpers Lacking Direct Tests

**Finding:** `FormatBlochVector` and `BlochCSV` were exported but not directly tested.

**Resolution:** Added direct tests for all output formats and edge cases.

| Aspect | Details |
|--------|---------|
| Commit | [d80166d](https://github.com/pjbaur/quantum/commit/d80166d) |
| Phase | 3.3 |

---

## Implementation Summary

### Workstream Execution

| Workstream | Focus | Status |
|------------|-------|--------|
| ws1-api-core | State-vector-first API, constructor consistency | Complete |
| ws2-concurrency | Parallel execution safety | Complete |
| ws3-sparse-backend | Backend capability model | Complete |
| ws4-algorithms | Scalability refactoring, negative tests | Complete |
| ws5-cli-docs-tests | CLI fixes, visualization tests | Complete |
| ws6-integration | Final integration, documentation | Complete |

### Key Commits

| Commit | Description |
|--------|-------------|
| [d183844](https://github.com/pjbaur/quantum/commit/d183844) | Phase 1.1 + 1.4: Gate API cleanup, normalization fix |
| [fe19529](https://github.com/pjbaur/quantum/commit/fe19529) | Phase 1.2: Parallel execution safety |
| [d9c31a7](https://github.com/pjbaur/quantum/commit/d9c31a7) | Phase 1.3: Sparse backend capabilities |
| [d7dde57](https://github.com/pjbaur/quantum/commit/d7dde57) | Phase 1.5: Constructor consistency |
| [d1f52b2](https://github.com/pjbaur/quantum/commit/d1f52b2) | Phase 2.1: CLI help consistency |
| [3f0e581](https://github.com/pjbaur/quantum/commit/3f0e581) | Phase 2.2: Algorithm scalability |
| [82e754f](https://github.com/pjbaur/quantum/commit/82e754f) | Phase 2.3: Shared gate validation |
| [c5082c4](https://github.com/pjbaur/quantum/commit/c5082c4) | Phase 3.1: Algorithm negative tests |
| [3300490](https://github.com/pjbaur/quantum/commit/3300490) | Phase 3.2: Parallel race tests |
| [d80166d](https://github.com/pjbaur/quantum/commit/d80166d) | Phase 3.3: Visualization tests |

---

## Documentation References

- [Critical Review](critical-review-2026-02-21.md) - Original findings
- [Implementation Plan](critical-review-implementation-plan-2026-02-21.md) - Detailed plan
- [Migration Guide v2.0](MIGRATION-v2.md) - Breaking changes and migration
- [Deprecation Policy](deprecation-policy-v2.md) - Removed methods
- [Compatibility Policy](compatibility-policy.md) - Project compatibility stance
- [Sparse State Backend](sparse-state.md) - Updated backend documentation
- [ADR Index](adr/README.md) - All architecture decisions
