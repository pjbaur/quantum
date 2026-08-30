# Symbolic Parameter Binding + VQE Driver — Design

Date: 2026-08-30
Status: Approved (brainstorming session)
Resolves: deferred residual of backlog item 4 (`docs/enhancement-backlog-2026-08-27.md`)

## Context

Backlog item 4 shipped the cheap-rebuild half of parameter rebinding
(`quantum.Resetter`): variational loops can rebuild gates, `Reset`, and
re-`Execute` on one state. The deferred half — symbolic parameter binding —
was explicitly parked until a real VQE/QAOA driver existed to justify the
concept layer. This design delivers both: a template-based binding layer and
a VQE driver for the 2-qubit reduced H2 Hamiltonian, the ladder item from
`docs/example-ideas.md` stage 7.

Decisions made in brainstorming:

1. Scope: binding layer + VQE driver. QAOA follows later, reusing binding.
2. Mechanism: template + `Bind` materializing a concrete circuit.
   `quantum.Gate` stays metadata-only; no backend changes.
3. Optimizer: parameter-shift gradient with fixed-step descent.
4. Target: published 2-qubit reduced H2 coefficients (not a toy Ising chain).

## Component 1: package `parameterized` (new, top-level)

A `Template` is a circuit recipe with named parameter holes. `Bind` fills
the holes and returns an ordinary `*circuit.Circuit`.

```go
t := parameterized.NewTemplate()
t.AddParamGate("theta", parameterized.Ry, 0) // Ry(theta) on qubit 0
t.AddGate(gates.NewCNOT(), 0, 1)             // fixed gate
t.AddParamGate("phi", parameterized.Rx, 1)

c, err := t.Bind(parameterized.Params{"theta": 0.4, "phi": 1.1})
```

### API

- `Params` is `map[string]float64`.
- `NewTemplate() *Template`.
- `(*Template) AddParamGate(name string, factory Factory, targets ...int) error`
  — declares use of a parameter. `Factory` is `func(value float64) quantum.Gate`.
- `(*Template) AddGate(gate quantum.Gate, targets ...int) error` — fixed step.
- `(*Template) ParamNames() []string` — declared names, first-use order.
- `(*Template) Bind(values Params) (*circuit.Circuit, error)` — materializes.

### Factory constructors

`parameterized.Rx`, `Ry`, `Rz`, `Phase` — each `func(value float64) quantum.Gate`,
delegating to the existing `gates.NewRx` etc. No new gate types.

### Errors (typed, package-local)

- `MissingParameterError` — `Bind` called without a declared name.
- `UnknownParameterError` — `Bind` given a name the template never declared
  (typo detection).
- `InvalidParameterValueError` — non-finite value (NaN, ±Inf).
- Duplicate parameter registration on the same name is legal (a parameter may
  drive several gates); duplicate name with different factories is also legal —
  both are validated at `Bind` time by value, not at declaration.

### Non-goals

- No symbolic gate types in `gates`; no change to the `quantum.Gate`
  interface, `Registry`, or any backend.
- No expression support (`theta/2`, `theta+phi`). A parameter maps to exactly
  one gate angle. Callers needing derived angles declare extra parameters.
- No shared/multi-parameter gates.

### Import direction

`parameterized` imports `circuit`, `gates`, `quantum`. Nothing imports
`parameterized` except `algorithm` (the driver) and tests. No cycles.

## Component 2: package `algorithm` additions

### `Hamiltonian` — sum of Pauli strings with coefficients

```go
h := algorithm.NewHamiltonian().
    AddTerm(-1.05237324).                                  // identity offset
    AddTerm(0.39793742, algorithm.PauliZ, algorithm.PauliZ). // Z⊗Z
```

- `AddTerm(coefficient float64, axes ...quantum.PauliAxis) *Hamiltonian` —
  empty axes = identity term. Chainable.
- `Energy(s quantum.QuantumState) (float64, error)` — exact energy:
  sum of `coefficient * quantum.Expectation(s, axes)` per term; identity
  terms add their coefficient directly. Error taxonomy reuses
  `quantum.Expectation`'s (nil state, unnormalized, axis-length mismatch).
- Terms stored in insertion order; `Energy` iterates in that order so
  floating-point summation is deterministic.

### `H2Hamiltonian()` — published 2-qubit reduced H2

The standard tutorial coefficient set for H2 at R = 0.735 Angstrom
(O'Malley et al., PRA 93, 052337 (2016), XZX/parity-basis reduction; exact
values pinned at implementation time with the source cited in the doc
comment). Representation: `g0*I + g1*Z0 + g2*Z1 + g3*Z0Z1 + g4*Y0Y1`.

The expected ground energy is NOT a hand-copied constant: the test suite
computes it by independently diagonalizing the 4x4 matrix built from the same
Pauli sum (see Testing). This guards against coefficient-convention mistakes.

### `H2Ansatz()` — canonical template

`X` on qubit 1 (seeds the odd-parity sector carrying the ground state in
the O'Malley parity-basis Hamiltonian), then `Ry("theta")` on qubit 0,
then `CNOT(0 -> 1)`. One parameter. Amended 2026-08-30 after measurement:
the original Ry+CNOT-from-|00> form spans only the even-parity sector
(reachable minimum -1.2446 Ha) and cannot reach the -1.8573 Ha ground
state. Returns `*parameterized.Template`.

### `VQE` — the driver

```go
res, err := algorithm.VQE(h, tmpl, algorithm.VQEOptions{
    InitialParams: parameterized.Params{"theta": 0.1},
    StepSize:      0.3,
    MaxIterations: 200,
    Tolerance:     1e-10,
})
```

- `VQE(h *Hamiltonian, t *parameterized.Template, opts VQEOptions) (*VQEResult, error)`.
- `VQEResult`: `Energy float64`, `Params parameterized.Params`,
  `Iterations int`, `Evaluations int`, `Converged bool`.
- Defaults for zero-valued options: `StepSize` 0.3, `MaxIterations` 200,
  `Tolerance` 1e-10; `InitialParams` defaults to 0.0 for every declared name.
- Loop per iteration:
  1. `t.Bind(params)` — materialize circuit.
  2. `state.New(numQubits)` + `circuit.Execute` — fresh dense state.
     (Allocator churn vs `Resetter` reuse is measured in the benchmark;
     either way the iteration is O(2^n) execution-dominated.)
  3. `h.Energy(state)` — exact.
- Gradient: parameter shift per parameter,
  `dE/dtheta = (E(theta + pi/2) - E(theta - pi/2)) / 2`, two evaluations per
  parameter. Exact and deterministic because `Expectation` is exact.
- Update: `theta -= StepSize * grad` for every parameter, then re-evaluate.
  If the stepped energy is higher than the current energy, revert to the
  previous parameters and halve the step (decay carries across iterations,
  floor 1e-6); the reverted energy stands as the iteration result. Terminate
  when the absolute change in energy between consecutive iterations is below
  `Tolerance` (converged) or `MaxIterations` is reached.
- Shot-noise mode (energy via `SampleExpectation`) is deliberately excluded;
  exact mode is deterministic and testable. Revisit with a noise-aware demo.

### Errors

- `nil` Hamiltonian or template: typed error (`InvalidVQEInputError` or
  matching existing taxonomy style).
- `Bind`/`Execute`/`Energy` errors propagate unwrapped-in-meaning: the first
  failure aborts the loop and returns.

## Testing

- `parameterized`: bound circuit equals manually built circuit state-for-state
  across several param values; mixed fixed+param steps; all error paths
  (missing, unknown, non-finite); `ParamNames` order; same name driving two
  gates.
- Gradient: parameter-shift vs central finite difference on seeded random
  states and templates, agreement within 1e-6.
- VQE: converges to the Jacobi-computed ground energy of `H2Hamiltonian`
  within 1e-6 in energy; `Converged` true; iteration/evaluation counts sane
  (evaluations = iterations + 2*params*iterations accounting); non-finite and
  error-propagation paths; default options work.
- Test-only helper: 4x4 Hermitian Jacobi eigensolver (test package, not
  exported) computing the true minimum of the H2 Pauli sum independently.
- Benchmark: VQE iteration cost vs the manual rebuild loop from backlog
  item 4 (bind+execute vs new-gates+reset+execute).

## Documentation

- ADR-0010: template materialization over symbolic gates — records why
  `quantum.Gate` stays metadata-only.
- `docs/enhancement-backlog-2026-08-27.md` item 4: deferred half now done,
  pointer to this design and ADR-0010.
- `docs/example-ideas.md` stage 7: parameter-binding capability satisfied.

## Out of scope (future)

- QAOA driver (stage 8) — reuses `parameterized` unchanged.
- Shot-noise VQE mode.
- Derived-angle expressions, shared parameters across gates with scaling.
- Sparse-backend VQE (dense is the variational workhorse; sparse gains little
  at 2 qubits).
