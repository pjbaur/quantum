# Critical Gaps TODO List

## 1. Multi-qubit gate application (state.ApplyGate)
- [x] Audit current `state.State.ApplyGate` behavior and confirm missing multi-qubit handling paths.
- [x] Implement 4x4 (2-qubit) gate application for CNOT and SWAP using `gates` matrices.
- [x] Validate target indices and target count with existing error types (`QubitsOutOfRangeError`, `InvalidGateApplicationError`).
- [x] Add/extend table-driven tests for CNOT and SWAP outcomes, including control/target ordering.
- [x] Add Bell-state creation test to ensure entanglement behavior is correct.
- [x] Verify state normalization is preserved after multi-qubit gate application.

## 2. Documentation drift vs repository structure
- [x] Inventory docs that reference outdated paths (CLI entry point, examples location, measurement package).
- [x] Update CLI usage instructions to reference `cmd/quantum` and current demo flow.
- [x] Update example references to `internal/examples` and ensure each example file name matches docs.
- [x] Document the current status of `measurement` package and remove or flag empty/placeholder references.
- [x] Cross-check README and guide docs against actual package layout and exported APIs.

## 3. Build friction (Go version and noisy tests)
- [x] Review `go.mod` Go version against supported toolchains and decide target version policy.
- [x] If adjustment is needed, update `go.mod` and any CI/tooling expectations accordingly.
- [x] Locate and remove debug prints from tests to reduce noisy output.
- [x] Re-run `go test ./...` to confirm clean output after changes.
