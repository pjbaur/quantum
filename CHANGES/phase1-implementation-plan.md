# Phase 1 Gaps: Implementation Plan

This plan covers the "Partially completed" and "Missing" items listed in
`CHANGES/phase1-implementation.md`. It focuses on multi-qubit gate support,
documentation, and tests, with minimal scope changes.

## Goals
- [x] Enable multi-qubit gate application in `state.State.ApplyGate`.
- [x] Ensure Bell-state examples work as written.
- [x] Add required package-level documentation for `package quantum`.
- [x] Add tests for multi-qubit gate application and coverage in `QuantumState`.

## Plan

1. [x] Implement multi-qubit gate application in `state.State.ApplyGate`.
   - Add an internal path for 4x4 (2-qubit) gates (CNOT, SWAP) using the
     existing `gates` matrices.
   - Validate target indices and target count; preserve current error types
     (`QubitsOutOfRangeError`, `InvalidGateApplicationError`).
   - Confirm single-qubit code path remains unchanged for 2x2 matrices.
   - Files: `state/state.go`.

2. [x] Make Bell-state examples succeed with multi-qubit gates.
   - Re-run the example flow mentally to ensure `ApplyGate(cnot, 0, 1)` now
     executes without error.
   - No API changes expected; only behavior change in state gate application.
   - Files: `internal/examples/bell.go` (verify, no edits unless needed).

3. [x] Add package-level documentation for `package quantum`.
   - Add a package comment in either `quantum/interfaces.go` or
     `quantum/errortypes.go`.
   - Keep the comment consistent with existing README usage.
   - Files: `quantum/interfaces.go` (preferred) or `quantum/errortypes.go`.

4. [x] Add tests for multi-qubit gates and `QuantumState.ApplyGate`.
   - Table-driven tests for:
     - CNOT on |00> and |10> cases (control qubit behavior).
     - SWAP on a simple two-qubit state.
     - Error paths for incorrect target count or invalid targets.
   - Ensure tests validate normalization and expected amplitudes.
   - Files: `state/state_test.go` (new or existing test file).

## Validation
- [x] `go test ./...`
- [x] Optional: run the Bell demo in `cmd/quantum` to manually verify output.

## Assumptions / Notes
- Multi-qubit gates are limited to 2-qubit (4x4) matrices for this phase.
- No new dependencies are introduced.
