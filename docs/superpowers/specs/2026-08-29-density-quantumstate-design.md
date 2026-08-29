# Density Backend as QuantumState — Design

Date: 2026-08-29
Backlog: `docs/enhancement-backlog-2026-08-27.md` item 9
Supersedes (in part): ADR-0007 (`docs/adr/0007-density-backend-scope.md`)
Status: approved

## Problem

`internal/density.Matrix` is scoped by ADR-0007 to noise-and-analysis: it
applies single-qubit gates and Kraus channels but does not implement
`quantum.QuantumState`, so a density matrix cannot participate in circuit
execution. ADR-0007's revisit trigger is a concrete consumer that needs to
run a full circuit against a density-matrix backend interchangeably with
the state-vector backends.

That consumer now exists: a noisy-Bell demonstration that builds one
`circuit.Circuit` and executes it unchanged against both a dense state
vector and a density matrix, then applies depolarizing noise to the
density matrix — showing what circuit-level density simulation adds
(purity decay, coherence loss) that no state vector can represent.

## The core tension

`QuantumState` is state-vector-shaped in two places:

- `Amplitude(basisState) complex128` — a mixed state has no amplitude
  vector, and the signature cannot return an error.
- `SetAmplitude(basisState, value) error` — writing one amplitude of ρ
  has no coherent meaning.

Chosen resolution (over an interface split, and over a dishonest
`√ρᵢᵢ` diagonal amplitude):

- **`Amplitude(i)`: pure-only reconstruction, NaN when mixed.** When
  `|Purity() − 1| ≤ quantum.NormalizationTolerance`, ρ = |ψ⟩⟨ψ| and ψ is
  recovered up to global phase: find the first k with ρₖₖ above
  tolerance, then ψᵢ = ρᵢₖ/√ρₖₖ (convention: ψₖ real positive). When ρ
  is mixed, return `cmplx.NaN()`. The existing NaN-safe
  `UnnormalizedStateError` guards in `Sample`, `Expectation`, and
  `Fidelity` then reject mixed states cleanly instead of silently
  computing nonsense.
- **`SetAmplitude` returns `UnsupportedOperationError`.** The
  capability-refusal precedent (`BulkAmplitudeSetter`, QFT) already
  establishes that a backend may decline an operation with a typed error.
- **`Probability(i)` = `real(ρᵢᵢ)`** — exact for pure and mixed states;
  0 for out-of-range indices, matching backend convention.

`Matrix` does not implement `BulkAmplitudeSetter` (so `QFT` refuses with
`UnsupportedOperationError`, correctly) and does not implement `Resetter`
(no consumer; the demo constructs fresh matrices).

## Design

### ApplyGate — generic k-qubit, no cap

`ApplyGate(gate quantum.Gate, targets ...int) error` on `*Matrix`:

- Validation mirrors the dense backend: `backendmath.ValidateTargets`,
  matrix-dimension check against `1 << len(targets)` →
  `InvalidGateApplicationError`.
- k = 1 delegates to the existing `applySingleQubitOperator`, which is
  kept because the Kraus channels use it with distinct src/dst buffers.
- k ≥ 2 computes ρ → UρU† in two passes reusing
  `backendmath.ComboMasks` and `backendmath.MixCombos`:
  1. Left pass (Uρ): each column of ρ is mixed exactly like a state
     vector — walk anchor bases with no target bit set, gather the 2ᵏ
     entries, `MixCombos` with the gate matrix, scatter.
  2. Right pass ((Uρ)U†): each row is mixed with the conjugated gate
     matrix, built once per call.
  Scratch buffers reuse the existing `ensure*` helpers.
- `BackendCapabilities`: `SupportsGateQubits(k)` true for k ≥ 1,
  `MaxGateQubits()` 0. Cost O(4ⁿ·4ᵏ) documented on the method.

### Measure — projective measurement on ρ

`Measure(qubitIndex int) (int, error)` plus
`SetRandSource(quantum.RandomSource)` (same shape as both other
backends, so `MeasureInto`/`ApplyIfSet`/`Teleport`-style forcing works
unmodified):

- p₁ = Σ ρᵢᵢ over basis states with the qubit's bit set; p₀ likewise.
- Outcome via `backendmath.PlanCollapse(qubitIndex,
  backendmath.RandFloat64(src), p0, p1)` — inherits the zero-branch and
  unnormalized-state safety already shared by dense and sparse.
- Collapse in place: ρᵢⱼ survives iff `Keeps(i) && Keeps(j)`, divided by
  the measured branch's probability; everything else zeroed. This is
  ΠρΠ/p, and leaves a pure post-measurement state pure.
- Out-of-range index → `QubitsOutOfRangeError`.

### Clone

`Clone() quantum.QuantumState` — deep copy of `data`, carries the rand
source, does not copy scratch buffers.

### Consumer: noisy-Bell demo

In `internal/examples` (wired into `RunAllNoiseDemos`):

- Build the Bell circuit (H on 0, CNOT 0→1) once with the `circuit`
  package.
- `Execute` it on `state.New(2)` and on `density.New(2)` — the same call,
  which is the interchangeability ADR-0007 documented as missing.
- Apply depolarizing noise at increasing p to both qubits of ρ; print
  purity, coherence |ρ₀₃|, and populations P(00)/P(11) against the ideal
  state-vector run.

### Cleanup

- `ApplySingleQubitGate` is removed as redundant with
  `ApplyGate(g, target)`; its two call sites in the existing noise demos
  are updated. The compatibility policy permits hard removal.
- Package doc rewritten to describe the promoted scope.
- ADR-0009 records the decision and supersedes ADR-0007's scope
  restriction (ADR-0007 status updated the way ADR-0008 amended
  ADR-0003).
- Backlog item 9 checked off with a done-note; CHANGELOG entry.

## Error handling

Existing taxonomy only: `InvalidQubitCountError`,
`QubitsOutOfRangeError`, `InvalidGateApplicationError`,
`UnsupportedOperationError`, plus `PlanCollapse`'s unnormalized-state
error. No new error types.

## Testing

TDD throughout:

- Compile-time `var _ quantum.QuantumState = (*Matrix)(nil)` (and
  `BackendCapabilities`).
- Noiseless circuit equality: the same circuit executed on dense and
  density backends agrees on `Probability` and reconstructed `Amplitude`
  — Bell (k=2) and a Toffoli circuit (k=3 generic path).
- `Amplitude`: known-value reconstruction table for pure states
  (including phase convention), NaN once a channel mixes the state;
  `Sample`/`Fidelity`/`Expectation` reject the mixed state through their
  existing guards.
- `Measure`: forced outcomes via a stub rand source; post-measurement ρ
  is the correct projection (purity 1 from a pure pre-state, populations
  correct); measuring one half of a Bell pair collapses the other;
  out-of-range and unnormalized error paths.
- `SetAmplitude` returns `UnsupportedOperationError`; QFT on a density
  state refuses.
- `Clone` independence (mutating the clone leaves the original intact).
- `circuit.Execute` end-to-end including the capability check.
- Existing noise-channel and Bloch-vector tests keep passing unchanged
  (channels and analysis methods are untouched).
