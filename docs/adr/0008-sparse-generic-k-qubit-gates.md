# ADR-0008: Sparse Backend Generic k-Qubit Gate Support

## Status

Accepted (2026-08-27). Partially supersedes the gate-width limit in
[ADR-0003](0003-sparse-backend-capability.md); the `BackendCapabilities`
capability-check API from that ADR is retained unchanged.

## Context

ADR-0003 chose explicit capability limits for the sparse backend: only 1-
and 2-qubit gates were supported, and 3+ qubit gates were rejected with an
`UnsupportedOperationError` directing callers to the dense backend. That
kept the sparse backend from being a drop-in for general circuits — any
circuit containing a Toffoli or a wider controlled gate could not run on
it, which the critical review had flagged and the capability API made
loud but did not remove.

Meanwhile the 2-qubit path already implemented the general algorithm:
group the non-zero amplitudes into bases (registers with every target
qubit zero) and multiply each group by the gate matrix. Nothing about it
was specific to k = 2 except two hardcoded `4`s.

## Decision

The sparse backend applies gates of any width. The 2-qubit path is
generalized to `applyMultiQubitGate(gate, targets)` for k >= 2, with the
existing specializations kept: the 2x2 single-qubit path and the canonical
CNOT permutation fast path. `SupportsGateQubits` returns true for every
k >= 1 and `MaxGateQubits` returns 0 (no limit) per the interface
contract.

Cost is O(nonzero * 4^k): each occupied base pays a 2^k x 2^k
matrix-vector product. Wide gates on sparse states are therefore
supported but not necessarily efficient — the caller trades the gate
matrix's density against the state's sparsity. This parity-with-density
tradeoff is documented in `docs/sparse-state.md` and in the method
comments.

## Consequences

- Sparse is a true drop-in backend: circuits with 3+ qubit gates
  (Toffoli, Fredkin, higher controls) execute and match dense results,
  verified by dense/sparse equivalence tests at k = 3 and k = 4.
- The `UnsupportedOperationError` for gate width disappears; the
  remaining boundary the backend owns is matrix/target-count mismatch,
  which `TestSparseGateMatrixSizeMismatch` pins.
- The capability-check API stays: `BackendCapabilities` still lets other
  backends declare limits, and `circuit` still enforces both halves of
  the contract (`SupportsGateQubits` and `MaxGateQubits`) before
  execution.
- Callers who relied on the sparse backend refusing wide gates (as an
  accidental guard against accidentally expensive operations) lose that
  guard; the cost model is now the caller's judgment call.

## References

- Supersedes the gate-width limit in
  [ADR-0003](0003-sparse-backend-capability.md) (capability API retained)
- Enhancement backlog item 7 in
  `docs/enhancement-backlog-2026-08-27.md`
