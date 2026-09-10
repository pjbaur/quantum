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

#### `algorithm.VQE` validates its inputs and reports every input problem as `InvalidVQEInputError`

The VQE driver checks its inputs before the first evaluation: `StepSize`
and `Tolerance` must be finite and non-negative and `MaxIterations`
non-negative (zero still selects each default); `InitialParams` must be
declared and finite; each parameter must drive one template gate; each
Hamiltonian term must be finite and, unless it is the identity, as long as
the template's qubit count. Problems only running the template reveals (a
factory returning a gate wider than its target list or with a malformed
matrix, a Pauli axis outside the enum) are wrapped in the same type:
`InvalidVQEInputError` gains `Err` and `Unwrap`, so `errors.As` still
reaches `quantum.InvalidGateApplicationError` and the like. An energy,
gradient, or descent step that overflows float64 is reported the same way,
blaming the Hamiltonian or `StepSize`; no error blames a parameter the
caller supplied finite. Inputs that used to run silently (negative
`StepSize`, negative `MaxIterations`, negative or non-finite `Tolerance`,
an overflowing Hamiltonian) now error; inputs that used to fail as
`parameterized.InvalidParameterValueError` (non-finite `StepSize` or
`InitialParams`) or `quantum.IncompatibleQubitCountError` (Pauli-length
mismatch) are now `InvalidVQEInputError` with a nil cause. Design:
`docs/superpowers/specs/2026-09-08-vqe-input-validation-design.md`.

### Fixed

#### `parameterized.Template.ParamStepCounts` counts a parameter named `""`

`ParamStepCounts` used a non-empty name as its test for a parameter-driven
step, so a parameter declared as `""` (accepted by `AddParamGate`, listed
by `ParamNames`, bound by `Bind`) was never counted. `algorithm.VQE` reads
those counts for its one-gate-per-parameter rule, so a `""` parameter
driving several gates passed the check and was optimized against a
parameter-shift gradient that is wrong for such a template, reporting
`Converged`. Steps are now classified by whether they carry a factory, the
test `Bind` already applied, so `""` is counted like any other name and
`VQE` rejects it with `InvalidVQEInputError` when it drives more than one
gate. `AddParamGate("")` remains accepted; a `""` parameter driving one
gate optimizes as before. Design:
`docs/superpowers/specs/2026-09-08-empty-parameter-name-step-count-design.md`.

#### `algorithm`'s parameter-shift gradient rejects an incomplete parameter binding

The unexported `parameterShiftGradient` helper behind `VQE` shifted each
named parameter in a copy of the caller's `Params`, so a declared name the
caller had left out was written into the shifted copies at +/- pi/2 by the
shift itself: `Bind` saw a complete binding and the helper returned the
slope at an implicit 0 with a nil error, while the same gap left unshifted
failed in `Bind`, so whether the call errored depended on which names were
passed. The helper now checks that every declared parameter is bound
before any evaluation and reports the first missing one, in declaration
order, as `parameterized.MissingParameterError`, the error `Bind` itself
returns for an absent declared name. The check guarantees completeness
only, the one `Bind` rule a shift can hide; a non-finite value or an
undeclared key in `Params`, or an undeclared name among those to
differentiate, stays `Bind`'s to reject at the first shifted evaluation
that reaches it; with `names` empty no evaluation runs and nothing beyond
completeness is checked. `VQE` binds every declared parameter itself and
is unaffected; no exported behavior changes. Design:
`docs/superpowers/specs/2026-09-08-parameter-shift-missing-param-design.md`.

#### `algorithm`'s parameter-shift gradient counts every completed evaluation on failure

The unexported `parameterShiftGradient` helper behind `VQE` added two to
its evaluation count only after both shifted evaluations of a parameter
had succeeded, so when the +pi/2 evaluation completed and the -pi/2 one
failed, the count returned with the error omitted the evaluation that had
run, although the helper promises "the number of energy evaluations
consumed". Each evaluation is now counted as it returns an energy, the
rule `VQE` applies to its own evaluations, so the count returned with an
error covers every evaluation that completed before the failure; the
failing evaluation is not counted. `VQE` discards the count with the
error and sums it only on success, where the total is unchanged; no
exported behavior changes. With its last test moved into the regular
suite, `algorithm/backlog_red_test.go` is removed; item 20's red test
stays tagged in `algorithm/backlog_red_numeric_test.go`. Design:
`docs/superpowers/specs/2026-09-08-parameter-shift-evaluation-count-design.md`.

#### `parameterized.Template`'s zero value is usable

`AddParamGate` wrote to the template's name-tracking map unconditionally,
so a `Template` declared as a plain variable rather than through
`NewTemplate` panicked with "assignment to entry in nil map" on its first
parameter declaration, although nothing documented `NewTemplate` as
required. The fix allocated the map on the first declaration, its only
writer, and stopped `NewTemplate` pre-allocating it (the map has since
been removed altogether; see the by-value copy entry below), so the zero
value is exactly the template `NewTemplate(0)` returns: it declares
nothing, rejects every target as out of range, and, once `Bind`'s own
parameter checks pass, it fails as `circuit.New` does for a qubit count
of zero. No method panics on it; templates built with `NewTemplate`
behave as before. Design:
`docs/superpowers/specs/2026-09-09-zero-value-template-design.md`.

#### `parameterized.Template` rejects a gate declared with no targets

`AddParamGate` and `AddGate` checked each given target for range and so
accepted a call with none: the step was appended, a parameter declared
that way was listed by `ParamNames` and counted by `ParamStepCounts`, and
only `Bind` failed, with circuit's "at least one target is required" from
deep inside `circuit.AddGate` rather than at the call that declared the
gate. Both methods now reject an empty target list at the call, after
their nil check and before the range check, with an error naming the
parameter (`parameter "theta": at least one target is required`) or the
gate (`fixed gate "Hadamard": at least one target is required`); the
template is left as it was. Declarations with at least one target are
unchanged. Design:
`docs/superpowers/specs/2026-09-09-empty-target-list-design.md`.

#### `algorithm.VQE` rejects an angle too large for the parameter shift

The parameter-shift gradient behind `VQE` evaluates `theta +/- pi/2` in
float64, and above 2^54 the spacing between adjacent values exceeds pi, so
both shifted values rounded back to `theta`, the two evaluations bound the
same circuit, and the gradient came back exactly 0 with a nil error;
below that, from about 2^48, the shift was applied at a rounded offset
and the descent step was lost to the spacing. A huge finite
`InitialParams` value therefore passed the finiteness guard, never moved,
and the run reported `Converged` with it frozen at its start. `VQE` now
rejects an initial parameter whose magnitude exceeds 2^26 (67,108,864
rad, where the shift is still exact to 7.5e-9 rad) with
`InvalidVQEInputError`, and stops with the same type when a descent step
carries a parameter past that bound; the unexported helper rejects such
an angle before any evaluation for direct callers. Angles are rejected,
not reduced mod 2*pi: the energy is periodic only under the
`exp(-i*theta*P/2)` factory convention `VQE` cannot check, and the
reduction itself is not computable in float64 at those magnitudes.
Callers with a larger angle reduce it themselves. Angles within the bound,
including every value a run from an angle and step size of ordinary size
reaches, are unaffected. With its only test replaced by tests of the
chosen contract, `algorithm/backlog_red_numeric_test.go` is removed. Design:
`docs/superpowers/specs/2026-09-09-parameter-shift-angle-bound-design.md`.

#### `parameterized.Template` refuses a by-value copy taken after a declaration

A `Template` copied by value after a declaration shared the original's
name-tracking map but not its slice headers, so a name declared on the
copy was already "known" to the original: the original's next declaration
of that name was counted by `ParamStepCounts` but never listed by
`ParamNames`, and `Bind` accepted values that omitted it, binding the gate
at angle 0. Go cannot make such copies independent (the two also share the
backing arrays of their step lists, so a declaration on one can overwrite
one on the other), so a `Template` is now documented as used in place or
through a pointer, as `NewTemplate` returns it, and, like
`strings.Builder`, records the receiver of its first accepted declaration:
`AddParamGate`, `AddGate`, and `Bind` on a copy taken after that return an
error, before any other check. The step and name lists themselves have
moved behind one pointer that the first accepted declaration allocates
and every later copy shares (the name-tracking map is gone; whether a
name is declared is read from the list), so a copy assigned back over the
original, which the receiver check cannot tell from the original,
restores a state that same variable founded, and drops nothing unless the
variable was reset to an undeclared template between the copy and the
copy-back, in which case it restores the earlier of that variable's own
states: no sequence of by-value copies, copy-backs, and declarations can
leave `ParamNames` and `ParamStepCounts` disagreeing or let `Bind` accept
a binding that omits a declared name. `ParamNames` and `ParamStepCounts`
on a refused copy read that shared state, so they describe the template
as it is now, and `NumQubits` agrees with the original because a
template's qubit count is fixed at construction; a copy taken before any
declaration is accepted is an independent template; no method panics.
Nothing in the module copies a `Template` by value, so no caller changes.
Design: `docs/superpowers/specs/2026-09-10-template-copy-guard-design.md`.

#### `parameterized.Template` copies the targets a declaration is given

`AddParamGate` and `AddGate` stored the caller's variadic `targets` slice
by reference, so mutating it after a call that had returned success
changed what `Bind` later built. A caller that fills one target slice and
reuses it across declarations, the usual shape of a generated circuit, got
gates on the targets written last rather than on the ones each declaration
named, and the range check the declaration had passed no longer described
the result. Both writers now store their own copy, as `circuit.AddGate`
already does with the same argument, so an accepted declaration is fixed;
the copy is made only after the argument checks pass, so a rejected
declaration still allocates nothing. Nothing in the module mutates a
target slice after declaring with it, so no caller changes. With this the
last red test moves into the regular suite and
`parameterized/backlog_red_test.go` is removed, so no test carries the
`redtests` build tag any more. Design:
`docs/superpowers/specs/2026-09-10-declared-targets-copy-design.md`.

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
