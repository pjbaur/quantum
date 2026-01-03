# Phase 2 TODO List

This document expands the Phase 2 items in `CHANGES/grand-assessment.md` into a
thorough, actionable checklist. The focus is on usability and testing without
changing public APIs or introducing new dependencies.

## 1) Multi-qubit gate application in `state.ApplyGate`

- [x] Identify the current behavior and limitations in `state.ApplyGate`.
- [x] Define the correct indexing and bit-order conventions used by the project.
- [x] Implement multi-qubit gate application for arbitrary target qubit indices.
- [x] Ensure correct handling of non-adjacent qubits and multi-target gates.
- [x] Verify unitarity preservation and amplitude conservation in the update logic.
- [x] Add clear error handling for invalid indices or mismatched gate sizes.
- [x] Keep single-qubit behavior unchanged to avoid regressions.
- [x] Add or update documentation comments if behavior is subtle.

## 2) Bell state validation tests

- [x] Add a table-driven test for Bell state preparation.
- [x] Cover at least the standard sequence: H on qubit 0, then CNOT(0 -> 1).
- [x] Validate amplitudes (including phase) for the four Bell states.
- [x] Verify measurement probabilities match expected distributions.
- [x] Ensure tests are deterministic and avoid debug prints.
- [x] Place tests alongside existing state or gate tests, following repo conventions.

## 3) Circuit abstraction completion

- [x] Audit the current circuit API to identify missing features or partial methods.
- [x] Implement any incomplete circuit wiring or gate scheduling behavior.
- [x] Ensure circuit execution uses `state.ApplyGate` for consistency.
- [x] Validate circuit execution with simple multi-qubit examples.
- [x] Keep the API minimal and idiomatic; do not add new exported symbols unless needed.

## 4) Examples and usability improvements

- [ ] Add or update examples that demonstrate multi-qubit circuits.
- [ ] Include at least one example that produces a Bell state.
- [ ] Ensure examples compile and run with the current module layout.
- [ ] Keep examples small and focused; avoid unnecessary complexity.

## 5) Remove debug prints from tests

- [ ] Locate any debug prints in tests (stdout/stderr noise).
- [ ] Remove or replace with proper assertions.
- [ ] Re-run relevant tests to confirm clean output and stable results.
