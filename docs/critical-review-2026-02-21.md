# Critical Project Review (2026-02-21)

This document captures a critical assessment of the repository with prioritized findings and concrete changes to improve correctness, maintainability, and API safety.

## Highest-Priority Findings

1. `quantum.Gate` is architecturally misleading for multi-qubit usage.
`quantum/interfaces.go:32` requires `Apply(q Qubit)`, but multi-qubit gates either always error (`gates/gates.go:217`, `gates/gates.go:277`) or provide simplified helpers that are not physically correct for entangled states (`gates/gates.go:229`).

2. Parallel circuit execution relies on a fragile caller contract.
`ExecuteAllParallel` documents that states must be independent (`circuit/parallel.go:13`) but does not enforce it, while dense state mutates internal reusable buffers (`state/state.go:18`). Reusing a state across executions can corrupt results under concurrency.

3. Sparse state is not a drop-in backend for general circuits.
`internal/sparsestate/state.go:101` and `internal/sparsestate/state.go:105` only handle 1- and 2-qubit gates. Gates above 2 qubits are rejected (`internal/sparsestate/state.go:109`), which breaks compatibility with general circuit execution.

4. Normalization diagnostics are wrong after rollback.
Both dense and sparse `SetAmplitude` restore old values before returning `NormalizationError.Sum` (`state/state.go:70`, `internal/sparsestate/state.go:60`), so the error reports a post-rollback normalized sum instead of the invalid attempted sum.

## Medium-Priority Findings

1. CLI help and behavior are inconsistent.
`visual` is implemented (`cmd/quantum/main.go:48`) but omitted from the `-demo` help string (`cmd/quantum/main.go:84`). The optional parameter is parsed but unused (`cmd/quantum/main.go:138`).

2. Algorithm implementation scales poorly.
Grover and Deutsch-Jozsa construct full dense matrices (`algorithm/grover.go:86`, `algorithm/grover.go:109`, `algorithm/deutsch_jozsa.go:68`), causing avoidable memory/time growth.

3. ~~Gate matrix validation is duplicated.~~ ✅ RESOLVED
`gateQubitCount` exists in three packages (`circuit/circuit.go:154`, `state/state.go:145`, `internal/sparsestate/state.go:116`), increasing drift risk.
> **Resolution**: Extracted to `quantum.GateQubitCount` with `InvalidGateMatrixError` type. All packages now use the shared implementation. See Phase 2.3.

4. ~~Constructor behavior is inconsistent on invalid qubit counts.~~ ✅ RESOLVED
`state.New` and sparse `New` silently coerce invalid counts to 1 (`state/state.go:26`, `internal/sparsestate/state.go:24`), while `circuit.New` returns an error (`circuit/circuit.go:25`).

## Test Coverage Gaps

1. Algorithm negative paths are largely untested.
Error cases in `DeutschJozsa` and `Grover` are not covered (`algorithm/deutsch_jozsa.go:19`, `algorithm/grover.go:15`).

2. Parallel tests do not validate unsafe shared-state usage.
Current tests verify successful independent execution (`circuit/parallel_test.go:42`) but do not test shared-state misuse.

3. Visualization formatting helpers lack direct tests.
`FormatBlochVector` and `BlochCSV` are exported (`visualization/bloch.go:33`) but not directly tested (`visualization/bloch_test.go:11`).

## What To Do Differently

1. Unify around a state-vector-first API.
Deprecate or isolate legacy single-qubit gate-application paths and make matrix+targeted `ApplyGate` the only primary execution model.

2. Make concurrency contracts enforceable.
In `ExecuteAllParallel`, fail fast on reused state pointers or clone states internally before concurrent execution.

3. Define backend capability explicitly.
Either implement generic k-qubit gate support in sparse state or expose capability limits as an explicit contract so mismatches fail early and clearly.

4. Improve diagnostics and API consistency.
Report attempted (pre-rollback) normalization sums, and standardize invalid constructor behavior across core packages.

5. Eliminate obvious UX/documentation drift.
Align CLI flags/help/examples and remove or implement currently dead parameters.

6. Expand tests for regressions and misuse.
Add negative-path algorithm tests, parallel shared-state hazard tests, and direct tests for formatting/output helpers.

## Verification Snapshot

- `go test ./...` passed during this review.
- `go vet ./...` passed during this review.
- `go test -race ./...` failed in this environment due race-runtime/package-resolution setup issues, so concurrency findings are based on static analysis and API behavior review.
