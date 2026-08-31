# Example Demos: CHSH, QPE, QAOA — Design

Date: 2026-08-31
Status: approved in session (approach A, print + verified style, exact + sampled
output, VQE-driven QAOA with landscape)

## Goal

Close the deferred backlog item: CHSH, QPE, and QAOA exist as capabilities but
have no demos. Add three protocol implementations to `algorithm/` with real
unit tests, and thin print demos in `internal/examples/` wired into
`cmd/quantum`.

Approach A: protocol logic lives in `algorithm/` (precedent: `teleport.go`,
`h2.go`), presentation lives in `internal/examples/`. Demos print like the
existing ones, but the underlying protocol functions are exported and tested
with exact-value assertions.

## CHSH

### `algorithm/chsh.go`

`quantum.Expectation` accepts only Pauli I/X/Y/Z, and {Z,X} settings on the
Bell state give exactly S = 2 — no violation. The violation needs 45°-rotated
bases, so correlation measurement rotates each half into the Z basis before
the Pauli-string expectation:

- `ChshCorrelation(s quantum.QuantumState, thetaA, thetaB float64) (float64, error)`
  — clones `s`, applies `gates.NewRy(-thetaA)` to qubit 0 and
  `gates.NewRy(-thetaB)` to qubit 1, returns
  `quantum.Expectation(clone, [PauliZ, PauliZ])`. On the Bell state the result
  is cos(thetaA - thetaB). The original state is never modified.
- `ChshSampledCorrelation(s quantum.QuantumState, thetaA, thetaB float64, shots int, rng quantum.RandomSource) (float64, error)`
  — same rotations on a clone, then
  `quantum.SampleExpectation(clone, [PauliZ, PauliZ], shots, rng)`.
- `ChshSExact(s quantum.QuantumState) (float64, error)` and
  `ChshSSampled(s quantum.QuantumState, shots int, rng quantum.RandomSource) (float64, error)`
  — canonical settings a0=0, a1=pi/2, b0=pi/4, b1=-pi/4,
  S = E00 + E01 + E10 - E11. On the Bell state: 2*sqrt(2).

Bell-pair preparation (H + CNOT) is demo-side; not library code.

### Demo `internal/examples/chsh.go`

`ChshDemo()` / `RunAllChshDemos()`: prepare the Bell state, print a table of
the four settings with exact E and seeded sampled E (~400 shots each), print
S exact vs sampled, the classical bound 2, the Tsirelson bound 2*sqrt(2), and
a one-line verdict that the classical bound is violated.

### Tests `algorithm/chsh_test.go`

- Exact S = 2*sqrt(2) within 1e-9 on the Bell state.
- Each E(thetaA, thetaB) matches cos(thetaA - thetaB) within 1e-9.
- Product state |00>: S within 1e-9 of sqrt(2) <= 2 — no violation without
  entanglement.
- Sampled S with a fixed seed lands in [2, 2*sqrt(2) + statistical slack].
- Rotation runs on a clone: the original state's amplitudes are untouched.

## QPE

### `algorithm/qpe.go`

- `EstimatePhase(u quantum.Gate, eigenstate quantum.QuantumState, numCounting int) (bestCount int, phaseTurns float64, err error)`
  - Register layout: counting qubits 0..t-1 (qubit 0 = LSB), eigenstate
    embedded as qubit t by copying its two amplitudes.
  - Circuit: H on the counting qubits; controlled-u applied 2^j times with
    counting qubit j as control (repetition rather than gate-power — no
    matrix-power machinery needed, 2^t - 1 applications, trivial for t <= 4;
    works for any unitary); then a gate-level inverse QFT on the counting
    subregister only.
  - Readout: argmax probability over the counting outcomes. For an exact
    eigenvector the eigenstate qubit never entangles with the counting
    register, so the register is pure and the readout is deterministic when
    the phase is exactly t-bit representable.
- Subregister inverse QFT: decomposed into H + controlled-phase gates
  (`gates.NewControlled(gates.NewPhase(theta))`) + swap reversal, appended to
  the circuit. `quantum.InverseQFT` cannot be used here: it DFTs the entire
  statevector, which would mix the eigenstate qubit into the counting
  register.

### Demo `internal/examples/qpe.go`

`QpeDemo()` / `RunAllQpeDemos()`:

1. u = `gates.NewPhase(2*pi*3/8)` on eigenstate |1>, t = 3. Print the circuit
   steps and a note on why the subregister IQFT is built from gates.
2. Exact probability table over the 8 counting outcomes — P(011b = 3) = 1.0,
   exact readout of phi = 3/8.
3. Seeded shot-sample table of the same distribution.
4. Second run: phi = 1/3 (not 3-bit representable) — peaked but spread
   distribution, best estimate 3/8 against true 0.333, brief note on
   resolution vs counting-qubit count.

### Tests `algorithm/qpe_test.go`

- Subregister IQFT equivalence: the gate-decomposed circuit reproduces
  `quantum.InverseQFT` amplitudes on a t-qubit state within 1e-9 for
  t = 1..4. This test pins the bit-order and sign conventions; it is written
  first (TDD) because the decomposition is the one error-prone construction.
- Every exactly representable phase k/8 (k = 0..7): bestCount = k,
  phaseTurns = k/8, peak probability within 1e-9 of 1.
- phi = 1/3: bestCount = 3 (0.375 is the closest eighth).
- Eigenstate |0>: phase 0, bestCount 0.
- Error paths: nil gate, nil eigenstate, numCounting < 1, eigenstate with
  more than one qubit.

## QAOA

### Constraint discovered in `algorithm/vqe.go`

VQE rejects any parameter driving more than one template step ("parameter %q
drives %d template steps"). A QAOA template with one shared gamma across all
edges violates this. Adaptation: per-edge and per-qubit parameters.

### `algorithm/qaoa.go`

- `MaxCutHamiltonian(numQubits int, edges [][2]int, weights []float64) (*Hamiltonian, error)`
  — H = sum of w * Z_i Z_j. Expected cut = (W - <H>) / 2 where W is the total
  weight. Validation: numQubits >= 2, non-empty edges, no self-loops, all
  vertices in range, edges normalized to i < j, duplicate edges rejected,
  weights length matches edges (nil = unit weights).
- `QAOATemplate(numQubits int, edges [][2]int, layers int) (*parameterized.Template, error)`
  — parameters named `gamma_e<k>` (per edge) and `beta_q<k>` (per qubit),
  suffixed per layer when layers > 1, each driving exactly one gate so the
  existing VQE driver accepts the template. Cost layer per edge (i,j):
  CNOT(i,j), Rz(2*gamma) on j, CNOT(i,j) — implements e^(-i*gamma*w*Z_i Z_j)
  exactly. Mixer: Rx(2*beta) per qubit. Demo uses layers = 1.
- `CutOfBitstring(edges [][2]int, weights []float64, bits []int) (float64, error)`
  and `ExpectedCut(totalWeight, energy float64) float64` — small helpers
  shared by the demo and tests.

The p=1 triangle optimum is not hand-derived in this spec; the tests
establish it numerically first (dense brute-force scan), then pin VQE's
result against it.

### Demo `internal/examples/qaoa.go`

`QaoaDemo()` / `RunAllQaoaDemos()`:

1. Print the triangle graph, Hamiltonian terms, and template parameter names.
2. Landscape table: the symmetric-angle slice (all gamma_e = gamma, all
   beta_q = beta), ~9x9 grid, expected cut per cell, best cell marked —
   labeled honestly as the symmetric subspace of the per-edge parameter
   space.
3. VQE run with InitialParams seeded from the landscape best (expanded to
   per-edge names). Print iterations, evaluations, final energy, expected
   cut.
4. Classical brute force over the 8 bitstrings: optimal cut 2, approximation
   ratio printed.
5. Seeded shot sample of the bound circuit: bitstrings, counts, cut values.

### Tests `algorithm/qaoa_test.go`

- Basis-state energies match hand-computed cuts: |001> has E = -1, cut 2
  (qubit 0 is the LSB; z0 = -1, z1 = z2 = +1).
- Zero angles: Rz(0) and Rx(0) are identity and the per-edge CNOTs cancel, so
  the energy equals <+...+|H|+...+> = 0 for the triangle.
- Every ParamStepCounts entry is 1 (VQE precondition).
- VQE accepts the template, converges to energy <= 0, and within tolerance of
  the numerically-established p=1 optimum.
- Error paths: self-loop, out-of-range vertex, duplicate edge, weight-length
  mismatch, layers < 1, numQubits < 2.

## Wiring

- `internal/examples/{chsh,qpe,qaoa}.go` each export their demo functions and
  a `RunAll<Demos>()` wrapper, matching the existing pattern.
- `cmd/quantum/main.go`: usage lines, dispatch cases, and the `all` sequence
  gain `chsh`, `qpe`, `qaoa` (with pause prompts between groups, as the
  existing groups have).
- Docs: the backlog trio line in `docs/enhancement-backlog-2026-08-27.md` is
  marked done with a dated note; CHANGELOG entry added.

## Verification split

Exact-value verification lives in `algorithm/` tests. `internal/examples`
functions stay thin printers (the existing split: algorithm is the protocol
library, internal/examples is presentation), so no new demo-level tests are
added there beyond what `utils_test.go` already covers.

## Out of scope

- Shared-angle QAOA templates and any VQE driver change to support them.
- Multi-layer QAOA demos (the template supports layers, the demo uses 1).
- Noise-aware variants of any demo.
- Mid-circuit-measurement (semiclassical/Kitaev) QPE.
