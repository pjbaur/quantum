# Enhancement Backlog (2026-08-27)

Forward-looking work items, distinct from the closed critical-review and
quality-assessment queues. Sources: the capability ladder in
`docs/example-ideas.md`, residuals documented in
`specs/architecture/code_quality_assessment.md`, and code inspection.

## Capability Gaps

These block the example-ideas trio (CHSH, teleportation, VQE-toy) and the
later ladder stages (QPE, QAOA, noise-aware demos).

- [x] 1. **Shot sampling API** — `Measure(qubitIndex)` measures one qubit at
  a time and collapses the state. Add a non-destructive
  `Sample(state, shots, rng)` helper returning a bitstring histogram.
  Unlocks CHSH, demo output, and expectation estimation. Lowest effort,
  highest unlock.
  > **Done (2026-08-27)**: `quantum.Sample(s, shots, rng)` in
  > `quantum/sample.go` — cumulative-distribution walk over
  > `Amplitude(i)`, binary search per draw, keys formatted like
  > `FormatStateView` (`%0*b`, qubit 0 = least significant bit).
  > Non-destructive, injectable randomness (nil rng = global default),
  > typed errors `InvalidShotCountError` and `UnnormalizedStateError`
  > (NaN-safe). Tests in `quantum/sample_test.go` and
  > `quantum/sample_backends_test.go` (dense + sparse agreement,
  > non-destructiveness, deterministic stub draws, zero-probability-bucket
  > skip). Sparse backends are walked in full — O(2^n) — documented in the
  > doc comment; a nonzero-only path would need a backend-side override.
- [x] 2. **Expectation values** — helpers for ⟨ψ|P|ψ⟩ over Pauli strings and
  basis-change measurement (X/Y-basis via pre-rotation). Blocks CHSH
  correlation, VQE, QAOA.
  > **Done (2026-08-27)**: `quantum.Expectation(s, axes)` computes the exact
  > value from the amplitude vector (permutation + phase sum, O(2ⁿ), no
  > matrix construction); `quantum.SampleExpectation(s, axes, shots, rng)`
  > estimates it from shots the way a device would — rotate X qubits by H,
  > Y qubits by S† then H, sample non-destructively on a Clone, average
  > outcome parity as ±1. `PauliAxis` type (`PauliI/X/Y/Z`), new typed
  > error `InvalidPauliAxisError`; length/nil/normalization errors reuse
  > the existing taxonomy. The rotation gates live in the quantum package
  > as local matrices because `gates` imports `quantum` (no reverse
  > import possible). Tests: single-qubit eigenvalue table, Bell-state
  > correlations (⟨XX⟩=1, ⟨YY⟩=-1, ⟨ZZ⟩=1), dense/sparse agreement over
  > all 16 two-qubit strings, exact-vs-sampled agreement, non-destructive
  > behavior, error propagation. TDD caught a real sign bug: the Pauli
  > phase must be evaluated at the permutation source index, not the
  > destination — strings with an odd Y count were sign-flipped.
- [x] 3. **Classical control / conditional gates** — mid-circuit `Measure`
  collapses correctly, but there is no classical register and no
  "if bit then gate". Blocks teleportation feed-forward, error correction,
  adaptive algorithms.
  > **Done (2026-08-27)**: `quantum.ClassicalRegister` (bool bits,
  > `Set`/`Bit`/`NumBits`/`String`) with new typed errors
  > `InvalidBitCountError` and `BitOutOfRangeError`;
  > `quantum.MeasureInto(s, qubit, creg, bit)` bridges quantum to
  > classical (measures and collapses like `Measure`, stores the 0/1);
  > `quantum.ApplyIfSet(creg, bit, s, gate, targets...)` is the
  > "if bit then gate" of feed-forward control — no-op on a cleared bit,
  > `ApplyGate` errors propagate. Proving consumer:
  > `algorithm.Teleport(s)` runs the full protocol (Bell pair, entangle,
  > measure to classical bits, X/Z corrections conditioned on outcomes)
  > on any backend with ≥3 qubits; measurement randomness via the state's
  > `SetRandSource`, so outcomes are forceable and the result exact.
  > Tests: register roundtrip and error paths, conditional apply/skip
  > semantics, teleport verified by Bloch-vector equality through
  > `Expectation` (I,I,X / I,I,Y / I,I,Z) across all four correction
  > branches, 15 fixed+random input states, sparse backend, measured
  > qubits collapsed, nil/short-state errors.
- [ ] 4. **Parameter rebinding** — `NewRx(theta)` bakes the matrix at
  construction, so variational outer loops must rebuild the whole circuit
  per iteration. Provide symbolic parameters or a cheap rebuild path.
- [x] 5. **State fidelity** — an F(|ψ⟩,|φ⟩) helper. Blocks teleportation
  verification; useful as a cross-backend assertion in tests.
  > **Done (2026-08-27)**: `quantum.Fidelity(a, b)` in
  > `quantum/fidelity.go` — |⟨ψ|φ⟩|² computed directly from both
  > amplitude vectors, O(2ⁿ), non-destructive, symmetric, global-phase
  > insensitive. Errors reuse the taxonomy (nil/typed-nil with
  > first/second naming, IncompatibleQubitCountError,
  > UnnormalizedStateError per side). `UnnormalizedStateError`'s message
  > generalized from "cannot sample" to "state is not normalized" since
  > it now guards Sample, Expectation, and Fidelity. Tests: known-value
  > table (orthogonal, half-overlap, phase-insensitivity),
  > Bell-vs-product 0.25, dense/sparse Bell agreement, symmetry,
  > non-destructiveness, full error paths.
- [ ] 6. **QFT / inverse QFT** — absent. Blocks phase estimation.
  `NewControlled` already provides controlled-U powers, so QFT is the
  missing half.

## Backend Work

- [ ] 7. **Generic k-qubit sparse gates** — sparse backend is capped at
  2-qubit gates by explicit contract. Lifting the cap makes sparse a true
  drop-in; the `BackendCapabilities` machinery can enforce either policy.
- [ ] 8. **Dedupe backend math residuals** — `SetAmplitude(s)` validation
  and rollback, `isNormalized`/`probabilitySum`, and the 2×2/combo mixing
  loops remain near-verbatim in dense and sparse backends (guarded by the
  dense-vs-sparse equivalence fuzz).
- [ ] 9. **Density backend as `QuantumState`** — ADR-0007 deliberately
  scopes `internal/density` to noise-and-analysis. Revisit trigger: a
  concrete consumer needing full circuits on density matrices (for example
  trajectory-style noisy execution).

## Quality / Infrastructure

- [ ] 10. **Extended fuzzing** — CI runs fuzz targets over seed corpora
  only. Commit grown corpora or add a periodic long `-fuzz` job.
- [ ] 11. **Coverage floor** — CI threshold is 40.0, actual coverage 48.0.
  Raise the threshold to ~45 to lock in gains.
- [ ] 12. **`.gitignore` fix** — `.claude/settings.local.json` is tracked
  and not ignored; it should be gitignored and untracked.

## Suggested Order

1 and 2 and 5 first — small, additive, unlock the CHSH demo end to end.
Then 3 for teleportation. Items 4 and 6 are larger; do them when VQE or
QPE is actually wanted.
