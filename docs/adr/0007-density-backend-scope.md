# ADR-0007: Density Backend Scope

## Status

Accepted

## Context

`internal/density` implements a density-matrix representation (`Matrix`)
with Kraus noise channels (`ApplyDepolarizing`, `ApplyDephasing`,
`ApplyAmplitudeDamping`), reduced single-qubit Bloch vectors
(`ReducedBlochVector`), and a bridge from a pure state vector
(`FromState(s quantum.QuantumState) (*Matrix, error)`, which builds
ρ = |ψ⟩⟨ψ|).

Unlike `internal/state` and `internal/sparsestate`, `Matrix` does not
implement `quantum.QuantumState`. Its gate application method,
`ApplySingleQubitGate(gate quantum.Gate, target int) error`, has a fixed
one-qubit arity rather than `QuantumState.ApplyGate(gate Gate, targets
...int) error`, and `Matrix` has no `Measure`, `SetAmplitude`, or `Clone`
matching that interface's signatures.

The package currently has two consumers: the noise demo
(`internal/examples/noise.go`), which builds a `Matrix` directly via
`density.New` and applies noise channels to it, and the Bloch-vector
bridge (`visualization.BlochVectorFromState`), which uses `FromState` to
convert a `quantum.QuantumState` into a `Matrix` purely to read off a
reduced Bloch vector. Neither consumer needs `Matrix` to participate in
circuit execution as a `QuantumState`.

Making `Matrix` implement `QuantumState` would require defining
measurement and collapse semantics on a density matrix (projective
measurement followed by renormalization, expressed on ρ rather than on
amplitudes), which is real design work with no consumer currently asking
for it.

## Decision

`internal/density` is a noise-demonstration and analysis backend: it
supports constructing density matrices, applying Kraus noise channels,
computing trace/purity, and reducing to single-qubit Bloch vectors, plus
building a density matrix from an existing pure state via `FromState` for
that analysis. It deliberately does **not** implement
`quantum.QuantumState`, and adding that conformance is out of scope until
a concrete consumer needs it.

## Consequences

### Positive

- The package stays small and honest about what it does: noise-channel
  application and Bloch-vector analysis, not general circuit execution.
- No speculative measurement/collapse semantics are designed or
  maintained without a real use case to validate them against.

### Negative

- `Matrix` cannot be dropped into circuit-execution code that is written
  against `quantum.QuantumState` (e.g. a `Circuit.Run` that takes a
  `QuantumState`); density-matrix simulation of a full circuit is not
  available today.
- Callers who want to run gates on a density matrix through the same
  abstraction as state-vector backends have no path to do so.

### Revisit Trigger

Revisit this decision if a concrete consumer needs to run a full circuit
(not just individual noise channels or gate applications) against a
density-matrix backend interchangeably with the state-vector backends —
at that point, define measurement/collapse semantics on `Matrix` and
implement `quantum.QuantumState`.

## Decision Log

- 2026-08-26: Initial decision accepted
