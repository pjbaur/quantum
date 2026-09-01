# Changelog

Notable changes to this project, newest first. Releases are tagged in the v0.x
series; see [Versioning](README.md#versioning) for why the "v2" in
[`docs/MIGRATION-v2.md`](docs/MIGRATION-v2.md) names the current API
generation rather than a Go module major version.

An "Unreleased" section describes what is on `main` but not yet tagged.

## Unreleased

### Added

#### Density-matrix backend implements `QuantumState`

`internal/density.Matrix` now satisfies `quantum.QuantumState` and
`quantum.BackendCapabilities` (ADR-0009): circuits execute on a density
matrix interchangeably with the state-vector backends, including
projective measurement with injectable randomness. Mixed states report
NaN amplitudes, so `Sample`/`Expectation`/`Fidelity` refuse them with
`UnnormalizedStateError`, and `SetAmplitude` returns
`UnsupportedOperationError`. New `NoisyBellDemo` in the noise demos runs
one Bell circuit on both backend families and applies depolarizing
noise.

#### CHSH, QPE, and QAOA protocol demos

`algorithm` gains three protocol modules with exact-value tests:
`algorithm/chsh.go` (rotated-basis CHSH correlations, exact and sampled S
values — the plain {Z, X} settings cannot violate the bound, the bases at
±45° are reached by Ry pre-rotation), `algorithm/qpe.go` (`EstimatePhase` /
`PhaseProbabilities` with a gate-level inverse QFT on the counting
subregister, since `quantum.InverseQFT` DFTs the whole register), and
`algorithm/qaoa.go` (`MaxCutHamiltonian`, `QAOATemplate` with per-edge
parameters so the VQE driver's one-gate-per-parameter rule holds,
`CutOfBitstring`). Demos: `quantum chsh`, `quantum qpe`, `quantum qaoa`.

### Breaking

#### `density.ApplySingleQubitGate` removed

Use `ApplyGate(gate, target)` — same behavior, interface-shaped.

## v0.3.0 — 2026-08-25

### Breaking

#### `algorithm.Grover` and `algorithm.DeutschJozsa` take the state to run on

Both algorithms used to create their own `*state.State` from a qubit count.
They now take a `quantum.QuantumState`, evolve it in place, and return the same
state for convenience. The caller picks the backend by picking the state, which
is what lets the same algorithm run on either the dense or the sparse
representation.

**Before:**
```go
s, err := algorithm.Grover(3, []int{5})
```

**After:**
```go
start, err := state.New(3)         // or sparsestate.New(3)
s, err := algorithm.Grover(start, []int{5})
```

`DeutschJozsa` no longer takes the input-qubit count either. The state's last
qubit is the ancilla, so a state of *n* qubits queries an oracle over *n-1*
input qubits: what used to be `DeutschJozsa(3, oracle)` is now a 4-qubit state
passed to `DeutschJozsa(start, oracle)`.

Two new requirements come with the signatures:

- The state must be **freshly created, in |0…0⟩**. Both algorithms build their
  superposition from that starting point and size their work by the register
  alone, so a state carrying earlier work would run to completion and return a
  wrong answer rather than an error. A state that does not start there is now
  rejected.
- The backend must implement `quantum.BulkAmplitudeSetter`. The oracles and
  Grover's diffusion step pass through an intermediate amplitude vector that no
  sequence of `SetAmplitude` writes could reach, since each of those has to
  leave the state normalized on its own. A backend without it gets an
  `UnsupportedOperationError` naming the alternative.

#### `quantum.GateQubitCount` rejects a 1x1 matrix

A 1x1 matrix passed the power-of-two test as 2^0 and yielded a qubit count of
zero, letting a scalar be "applied" to no qubits and handing callers a zero to
size their target list with. Matrices must now be at least 2x2;
`GateQubitCount` returns an `InvalidGateMatrixError` otherwise, and
`gates.NewControlled` rejects the same input for the same reason.

#### `SetAmplitude` and `SetAmplitudes` reject non-finite amplitudes

A NaN or infinite amplitude used to slip past the normalization check, because
|NaN|² is NaN and NaN compares false against any tolerance — so
`math.Abs(sum-1.0) > 1e-10` was false and the vector passed as normalized, then
poisoned every later measurement and gate application. Both backends now check
finiteness explicitly, before the normalization check, and return a
`quantum.NonFiniteAmplitudeError` naming the offending basis state. A rejected
write leaves the state untouched.

### Added

**`quantum`**

- `Probability(c complex128) float64` — |c|², computed as `re*re + im*im`
  rather than by squaring `cmplx.Abs`, whose square root the squaring only
  undoes again. Meant for normalized amplitudes; it has no overflow-safe
  scaling.
- `IsFiniteAmplitude(c complex128) bool` — the predicate the backends use to
  catch what a normalization check cannot see.
- `NonFiniteAmplitudeError` — typed error with `BasisState` and `Value`.
- `BulkAmplitudeSetter` — optional backend capability for replacing the whole
  amplitude vector in one validated call. Kept out of `QuantumState` so a
  backend that cannot offer it stays a `QuantumState`; callers type-assert and
  report an `UnsupportedOperationError` when the assertion fails.

**`gates`**

- `NewToffoli()` — the CCNOT gate (8x8), also now part of the built-in
  registry.
- `NewRx(theta)`, `NewRy(theta)`, `NewRz(theta)`, `NewPhase(phi)` —
  parameterized rotation and phase-shift gates. The angle is part of the gate's
  name (`"Rx(0.5)"`), so rotations by different angles stay distinguishable,
  including in a `Registry`, which keys on name.
- `NewControlled(gate)` — the controlled version of any gate, as the
  block-diagonal matrix diag(I, U). The control is `targets[0]`, the same
  convention CNOT already follows, so `NewControlled(NewPauliX())` reproduces
  CNOT and `NewControlled(NewCNOT())` reproduces Toffoli.

These five describe families of gates rather than single gates, so they are not
in `Builtin()`; build the member you want and register it in your own copy.

**State backends**

- The sparse backend gained `SetAmplitudes`, satisfying
  `quantum.BulkAmplitudeSetter` with the same length, finiteness, and
  normalization checks the dense backend applies, in the same order. This is
  what lets the `algorithm` package run on either backend. See
  [`docs/sparse-state.md`](docs/sparse-state.md) for the one storage difference
  (amplitudes at or below 1e-12 are pruned, not stored).

**CLI**

- `quantum gate <name>` prints one registered gate's qubit count and matrix;
  `quantum gate` with no name lists the registry.

### Changed / Internal

- The state-vector math the dense and sparse backends share — target
  validation, combo-mask construction, and measurement collapse — moved to the
  new `internal/backendmath` package, so a correction to the physics lands in
  both backends at once; the shared collapse also guards against a draw landing
  on a branch that holds no probability, which previously filled the dense state
  with NaN and emptied the sparse map.
- Go-native fuzz targets now cover the state and gate validation boundary
  (`quantum/gateutil_fuzz_test.go`, `state/state_fuzz_test.go`,
  `internal/sparsestate/state_fuzz_test.go`); the first run of them found the
  two behaviors tightened above.

## v0.2.0

The state-vector-first API. See [`docs/MIGRATION-v2.md`](docs/MIGRATION-v2.md)
for the breaking changes it introduced and the migration path for each, and
[`docs/deprecation-policy-v2.md`](docs/deprecation-policy-v2.md) for the
detailed removal list.
