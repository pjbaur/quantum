# ADR-0009: Density Backend Implements QuantumState

## Status

Accepted (2026-08-29). Supersedes the scope restriction in
[ADR-0007](0007-density-backend-scope.md); the noise-channel and
Bloch-vector analysis API decided there is retained unchanged.

## Context

ADR-0007 scoped `internal/density` to noise-and-analysis and deferred
`quantum.QuantumState` conformance until a concrete consumer needed to
run a full circuit against a density matrix interchangeably with the
state-vector backends. That consumer now exists: the noisy-Bell
demonstration executes one `circuit.Circuit` unchanged on a dense state
vector and on a density matrix, then applies depolarizing noise that
only the density representation can carry.

The interface is state-vector-shaped in two places a mixed state cannot
honor: `Amplitude(basisState) complex128` (no error return) and
`SetAmplitude`.

## Decision

`Matrix` implements `quantum.QuantumState` and
`quantum.BackendCapabilities`:

- `ApplyGate` computes ρ → UρU† for any gate width, reusing the shared
  `backendmath` kernel (U down columns, conj(U) along rows). No width
  cap: `SupportsGateQubits` is true for k ≥ 1, `MaxGateQubits` is 0.
  Cost O(4ⁿ·4ᵏ).
- `Measure` is projective measurement on ρ via the shared
  `PlanCollapse`: outcome from the diagonal, collapse ΠρΠ/p in place,
  randomness injectable via `SetRandSource`.
- `Probability(i)` is ρᵢᵢ, exact for pure and mixed states.
- `Amplitude(i)` reconstructs the state vector when ρ is pure (global
  phase fixed by making the first nonzero-probability basis amplitude
  real positive) and returns NaN when ρ is mixed. The NaN flows into
  the NaN-safe guards of `Sample`, `Expectation`, and `Fidelity`, which
  therefore refuse mixed states with `UnnormalizedStateError`.
- `SetAmplitude` always returns `UnsupportedOperationError` — a density
  matrix has no amplitude vector to write one entry of.
- `Matrix` deliberately does not implement `BulkAmplitudeSetter` (so
  `QFT` refuses it) or `Resetter` (no consumer).

`ApplySingleQubitGate` was removed as redundant with
`ApplyGate(g, target)`.

## Consequences

### Positive

- One circuit definition runs on dense, sparse, or density backends;
  noise channels compose with circuit execution.
- Measurement semantics on ρ share the collapse-planning code (and its
  zero-branch safety) with the other backends.
- Mixed states fail loudly, not wrongly, in amplitude-based helpers.

### Negative

- `Amplitude` on a mixed state returns NaN rather than an error — the
  interface signature allows nothing better; callers that skip the
  helpers' guards can propagate NaN.
- Density execution costs O(4ⁿ) memory and O(4ⁿ·4ᵏ) per gate, far
  beyond the state-vector backends; it remains a small-register tool.

## Decision Log

- 2026-08-29: Initial decision accepted
