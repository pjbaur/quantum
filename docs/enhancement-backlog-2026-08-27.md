# Enhancement Backlog (2026-08-27)

Forward-looking work items, distinct from the closed critical-review and
quality-assessment queues. Sources: the capability ladder in
`docs/example-ideas.md`, residuals documented in
`specs/architecture/code_quality_assessment.md`, and code inspection.

## Capability Gaps

These block the example-ideas trio (CHSH, teleportation, VQE-toy) and the
later ladder stages (QPE, QAOA, noise-aware demos).

> **Demos done (2026-08-31)**: the trio and both later algorithm stages now
> have demos — CHSH, QPE, and QAOA in `internal/examples/{chsh,qpe,qaoa}.go`
> (`quantum chsh|qpe|qaoa`), on protocol code in
> `algorithm/{chsh,qpe,qaoa}.go`. Teleportation and VQE demos predate this
> (`internal/examples/bell.go`, `algorithm/vqe.go`). Design:
> `docs/superpowers/specs/2026-08-31-example-demos-design.md`.

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
- [x] 4. **Parameter rebinding** — `NewRx(theta)` bakes the matrix at
  construction, so variational outer loops must rebuild the whole circuit
  per iteration. Provide symbolic parameters or a cheap rebuild path.
  > **Done (2026-08-27)** — via the cheap-rebuild path, which is the half
  > the loop actually lacked. Circuits were already safe to `Execute`
  > against many states, and gate reconstruction is trivial next to the
  > O(2ⁿ) execution; what forced reallocation each iteration was the
  > state. Added optional capability `quantum.Resetter` (interface
  > following the `BulkAmplitudeSetter` precedent, so backends without it
  > still satisfy `QuantumState`): `Reset()` restores |0…0⟩ in place,
  > reusing the amplitude vector (dense) or clearing the map (sparse),
  > and preserves an injected `RandomSource`. Variational pattern:
  > rebuild gates with new angles, `Reset`, `Execute` again on the same
  > state. Symbolic parameter binding deliberately deferred — it adds a
  > concept layer no current consumer needs; revisit with a real VQE/QAOA
  > driver. Tests: reset-to-zero, rand-source survival across reset
  > (stub draws consumed in order), reset-state vs fresh-state amplitude
  > equality after dirtying with gates and a measurement, `Resetter`
  > assertions on both backends.
  >
  > **Addendum (2026-08-30)**: the deferred symbolic half is now done too —
  > `parameterized.Template` + `Bind` materializes circuits from named
  > parameter maps, with the H2 VQE driver as the consuming evidence.
  > See `docs/adr/0010-parameter-binding-template.md` and
  > `docs/superpowers/specs/2026-08-30-parameter-binding-vqe-design.md`.
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
- [x] 6. **QFT / inverse QFT** — absent. Blocks phase estimation.
  `NewControlled` already provides controlled-U powers, so QFT is the
  missing half.
  > **Done (2026-08-27)**: `quantum.QFT(s)` / `quantum.InverseQFT(s)` in
  > `quantum/qft.go` — the exact 2ⁿ-dimensional DFT unitary
  > F|x⟩ = (1/√N)Σ_y e^{2πixy/N}|y⟩, computed by an in-place radix-2 FFT
  > over the amplitude vector (O(N log N), no matrix construction, no
  > gate decomposition, so no bit-reversal caveat — the basis index maps
  > directly onto the phase register the way QPE expects) and written
  > back through one `BulkAmplitudeSetter` call. Requires that capability
  > (`UnsupportedOperationError` otherwise); normalization/non-finite
  > rejections come from `SetAmplitudes` and leave the state untouched.
  > Sparse backends work but lose sparsity by nature — the QFT of a
  > sparse vector is dense. Tests: QFT|0…0⟩ uniform, exact QFT|j⟩ values
  > for n=2, inverse-QFT of phase gradients returns |j⟩ exactly,
  > forward/inverse roundtrips on seeded random states, normalization
  > preserved, nil/typed-nil, missing-bulk-writes error, dense/sparse
  > agreement plus roundtrip, sparse post-QFT probabilities.

## Backend Work

- [x] 7. **Generic k-qubit sparse gates** — sparse backend is capped at
  2-qubit gates by explicit contract. Lifting the cap makes sparse a true
  drop-in; the `BackendCapabilities` machinery can enforce either policy.
  > **Done (2026-08-27)**: `applyTwoQubitGate` generalized to
  > `applyMultiQubitGate(gate, targets)` for any k ≥ 2 — the algorithm was
  > already generic, only two hardcoded `4`s were 2-specific. Single-qubit
  > and canonical-CNOT fast paths kept. `SupportsGateQubits` true for all
  > k ≥ 1, `MaxGateQubits` 0 (no limit). Cost O(nonzero · 4^k) documented
  > in `docs/sparse-state.md` and method comments. ADR-0008 records the
  > decision and partially supersedes ADR-0003's width limit (capability
  > API retained). Tests: Toffoli and CC-S sparse-vs-dense equality,
  > controlled-Toffoli at k=4, Toffoli truth table, any-width capability
  > assertions, matrix/target-count mismatch still refused without partial
  > application, circuit execution with 3-qubit gates now matches dense
  > (three old contract tests updated as part of this approved change:
  > sparse caps, sparse unsupported-gate → size-mismatch, circuit sparse
  > capability checks). Full suite, race, vet, gofmt clean.
- [x] 8. **Dedupe backend math residuals** — `SetAmplitude(s)` validation
  and rollback, `isNormalized`/`probabilitySum`, and the 2×2/combo mixing
  loops remain near-verbatim in dense and sparse backends (guarded by the
  dense-vs-sparse equivalence fuzz).
  > **Done (2026-08-28)**: residuals deduped into the two shared homes.
  > Amplitude-vector policy (`NormalizationTolerance`, `IsNormalizedSum`,
  > `CheckNormalization`, `ValidateAmplitudeVector`) lives in
  > `quantum/normalization.go` beside `IsFiniteAmplitude`/`Probability`
  > and replaces the six 1e-10 literals that lived in four packages —
  > sample's const (shared by fidelity and expectation), two in each
  > backend, and qubit's check. Gate-application
  > physics (`MixCombos` matmul kernel, `ValidateQubitCount`,
  > `RandFloat64`) joins `ValidateTargets`/`ComboMasks`/`PlanCollapse` in
  > `internal/backendmath`. Both backends' `New`/`SetAmplitude`/
  > `SetAmplitudes`/`Measure` now call the shared helpers;
  > `isNormalized` and `randFloat64` are gone; `probabilitySum` stays
  > per-backend (slice vs map iteration is a real difference). Dense
  > `applyMultiQubitGate` keeps its caller-shaped buffers and loop
  > structure (BCE-sensitive) and shares only the middle matmul;
  > benchmarks before/after: no regression beyond noise. Single-qubit
  > 2×2 loops stay per-backend by design. Full suite, race, vet, gofmt,
  > fuzz seeds (including `FuzzDenseSparseGateEquivalence`) clean.
- [x] 9. **Density backend as `QuantumState`** — ADR-0007 deliberately
  scopes `internal/density` to noise-and-analysis. Revisit trigger: a
  concrete consumer needing full circuits on density matrices (for example
  trajectory-style noisy execution).
  > **Done (2026-08-29)**: `Matrix` implements `quantum.QuantumState` and
  > `BackendCapabilities` (ADR-0009, superseding ADR-0007's restriction).
  > `ApplyGate` is generic-k ρ → UρU† through the shared backendmath
  > kernel; `Measure` is projective collapse via `PlanCollapse` with
  > injectable randomness; `Probability` reads the diagonal; `Amplitude`
  > reconstructs pure states and returns NaN for mixed ones, so
  > Sample/Expectation/Fidelity refuse mixed states through their
  > existing guards; `SetAmplitude` returns `UnsupportedOperationError`;
  > no `BulkAmplitudeSetter` (QFT refuses) or `Resetter`.
  > `ApplySingleQubitGate` removed. Consumer: `NoisyBellDemo` executes
  > one Bell circuit on dense and density backends via `circuit.Execute`
  > and applies depolarizing noise. Tests: dense-equality circuits
  > (k=1..3), forced measurements, entangled-partner collapse, helper
  > guards, QFT refusal, capability assertions.

## Quality / Infrastructure

- [x] 10. **Extended fuzzing** — CI runs fuzz targets over seed corpora
  only. Commit grown corpora or add a periodic long `-fuzz` job.
  > **Done (2026-08-29)**: periodic job, not committed corpora — a
  > snapshot goes stale while a scheduled run keeps exploring.
  > `.github/workflows/fuzz.yml`: weekly (Mon 04:17 UTC) plus
  > `workflow_dispatch` with configurable per-target `fuzztime`
  > (default 10m). Targets are discovered dynamically
  > (`go test -list '^Fuzz'` per package), so new fuzz tests join the
  > rotation without workflow edits; each runs in its own
  > `-fuzz '^Name$'` invocation because Go allows only one fuzz target
  > per `go test` run. All targets run even after a failure (single
  > `failed` flag), and new crashers under `testdata/fuzz/` are
  > uploaded as artifacts so they reproduce locally. Loop verified
  > locally under bash at 5s/target — all six targets discovered and
  > run; workflow actionlint-clean.
- [x] 11. **Coverage floor** — CI threshold is 40.0, actual coverage 48.0.
  Raise the threshold to ~45 to lock in gains.
  > **Done (2026-08-29)**: threshold raised to 50.0, not 45 — coverage had
  > grown to 54.4% since this item was written, and the original ~45 was
  > chosen as roughly three points under the then-actual 48.0. Applying
  > the same margin to today's number gives 50.0: gains locked in,
  > ~4 points of headroom so unrelated PRs don't flake the gate. Gate
  > logic verified locally (awk comparison passes at 54.4 vs 50.0);
  > workflow actionlint-clean.
- [x] 12. **`.gitignore` fix** — `.claude/settings.local.json` is tracked
  and not ignored; it should be gitignored and untracked.
  > **Done (2026-08-29)**: `git rm --cached` only — `.claude/` was already
  > in `.gitignore`, but a tracked file overrides the ignore rule, so no
  > `.gitignore` edit was needed. Local copy stays on disk;
  > `git check-ignore` confirms the rule now applies. Only file tracked
  > under `.claude/`, so the directory is now fully ignored.

## Suggested Order

1 and 2 and 5 first — small, additive, unlock the CHSH demo end to end.
Then 3 for teleportation. Items 4 and 6 are larger; do them when VQE or
QPE is actually wanted.

- [x] 13. **`parameterShiftGradient` precondition doc** — `vqe.go` should
  state that the bound value must enter the gate as exp(-i*theta*P/2); the
  driver cannot detect a rescaling factory. First occurrence: QAOA template.
  > **Done (2026-09-07)**: `parameterShiftGradient` and `VQE` doc comments in
  > `algorithm/vqe.go` (lines 28–37 and 118–149) state both preconditions: each
  > parameter drives exactly one gate (enforced by VQE via
  > `parameterized.Template.ParamStepCounts`), and bound value enters its gate as
  > exp(-i*theta*P/2) per `gates.NewRx/NewRy/NewRz` and inherited
  > `parameterized.Rx/Ry/Rz/Phase` convention. Second precondition uncheckable:
  > `parameterized.Factory` is opaque; rescaled factory (e.g.
  > `gates.NewRz(2*value)`) yields E periodic with period pi, so the shift
  > difference E(theta+pi/2) − E(theta−pi/2) vanishes—indistinguishable from a
  > true stationary point—and `parameterShiftGradient` returns zero while the
  > true dE/dtheta is nonzero. `VQE` doc also states what `Converged` certifies
  > and does not (|delta E| < Tolerance, not a minimum; rounding can shift
  > parameters and energy in their last bits; parameterless templates accepted),
  > and cites `QAOATemplate` (cf09732) as precedent. `TestParameterShiftIsBlindToRescaledFactory`
  > in `algorithm/vqe_test.go` pins the failure mode: zero shift gradient
  > against nonzero finite difference across four base points, and VQE Converged
  > after one-iteration no-op. Edits to `vqe.go` are comment-only. Red testing
  > during QA gate found six pre-existing driver defects, preserved as tests
  > under `redtests` build tag and queued as items 14–17 in
  > `algorithm/backlog_red_test.go`.
- [x] 14. **VQE input validation and error taxonomy** — `VQE` validates only
  nil inputs, the one-gate-per-parameter rule, and undeclared initial
  parameter names; everything else it either runs with or reports through
  another package's error type. Observed: a negative `StepSize` climbs once
  and is then clamped to +1e-6 by the floor, a negative `MaxIterations`
  returns unconverged after zero iterations, a negative or NaN `Tolerance`
  can never converge, and a NaN or Inf `StepSize` or `InitialParams` value
  surfaces as `parameterized.InvalidParameterValueError`. A Hamiltonian
  whose Pauli strings do not match the template's qubit count, or a factory
  returning a gate wider than its target list, surfaces as
  `quantum.IncompatibleQubitCountError` or
  `quantum.InvalidGateApplicationError` from inside the first evaluation.
  A NaN or Inf Hamiltonian coefficient yields a NaN gradient and a NaN
  step, and again `InvalidParameterValueError` blames a parameter the
  caller supplied finite. Expected contract: out-of-domain options,
  Hamiltonian/template structural mismatches, and non-finite Hamiltonian
  coefficients are `InvalidVQEInputError`, and no error attributes the
  failure to a parameter that was finite on entry.
  > **Red tests**: `TestRedVQEOptionValidation`,
  > `TestRedVQEStructuralMismatchIsInvalidInput`, and
  > `TestRedVQENonFiniteHamiltonianIsNotBlamedOnParams` in
  > `algorithm/backlog_red_test.go`; reproduce with
  > `go test -tags redtests ./algorithm -run '^TestRedVQE(Option|Structural|NonFinite)'`.
  > **Done (2026-09-08)**: `VQE` in `algorithm/vqe.go` validates option domains
  > (`StepSize`, `MaxIterations`, `Tolerance`), initial-parameter finiteness,
  > Hamiltonian/template qubit-count structure, and non-finite coefficients up front
  > through `validateVQEOptions` and `validateVQEStructure` (which absorbs the
  > one-gate-per-parameter check and serves as item 15's extension point);
  > `InvalidVQEInputError` gained `Err` field and `Unwrap`, so evaluation-time
  > errors (wide factory gate, bad axis) wrap through `wrapEvaluationError` and
  > stay reachable via `errors.As`. Finiteness guards on every evaluated energy,
  > gradient, and step ensure no error blames a parameter that was finite on
  > entry, with Reasons naming the Hamiltonian or `StepSize` and the evaluation
  > point instead. `Tolerance: +Inf` now rejected (previously accepted; recorded
  > in the spec's backward-compatibility note and in CHANGELOG Unreleased); the
  > 2026-08-30 parameter-binding spec carries a supersession note for "errors
  > propagate unwrapped". Tests: the three red tests moved from `redtests` file
  > plus four found by this item's own red round, all now in `algorithm/vqe_test.go`
  > as `TestVQE...`, plus Reason-literal and wrap-phase tests; `evaluate` and
  > `parameterShiftGradient` untouched. Commits: c195317, 7c589c3, cd17d0a,
  > f4cdbe6, 2691726.
- [x] 15. **Empty parameter name defeats the one-gate-per-parameter check**
  — `parameterized.Template.AddParamGate` accepts `""` as a parameter name,
  but `ParamStepCounts` treats the empty string as its fixed-step marker and
  never counts it. Observed: a `""` parameter driving two `Ry` gates is
  listed by `ParamNames`, absent from `ParamStepCounts`, and passes `VQE`'s
  precondition check, so `VQE` optimizes it against the higher-harmonic
  gradient the check exists to prevent and reports Converged. Expected
  contract: the one-gate-per-parameter rule stated on `VQE` and
  `parameterShiftGradient` holds for every declared name, so a `""`
  parameter that drives several steps is rejected with
  `InvalidVQEInputError` like any other.
  > **Red tests**: `TestRedVQEEmptyNameParameterEscapesStepCountCheck` in
  > `algorithm/backlog_red_test.go`; reproduce with
  > `go test -tags redtests ./algorithm -run '^TestRedVQEEmptyName'`.
  > **Done (2026-09-08)**: root cause was a sentinel collision — `ParamStepCounts`
  > used `s.param != ""` while `Bind` used `s.factory != nil` to discriminate
  > parameter-driven steps from fixed steps. Fix switches `ParamStepCounts` to
  > the factory test (`parameterized/parameterized.go:77`, one expression) and
  > documents the invariant on the `step` type so `validateVQEStructure`'s
  > existing loop rejects a `""` parameter driving multiple gates with the
  > existing Reason; `algorithm/vqe.go` untouched. Rejecting `""` at
  > `AddParamGate` or in `VQE` was ruled out (would invent a name rule the
  > package does not have, or be dead code). `""` remains a legal name, now
  > documented on `AddParamGate`'s godoc. Backward compatibility: `VQE` on
  > such a template now errors where it previously ran and reported Converged,
  > recorded in CHANGELOG Unreleased. Tests: moved red test
  > `TestVQEEmptyNameParameterIsCheckedLikeAnyOther`, accept-pin
  > `TestVQEAcceptsEmptyNameDrivingOneGate` (via `h2AnsatzNamed`), two
  > `TestParamStepCounts` cases, `TestBindWithEmptyNameParameter`. Red round
  > found two pre-existing `AddParamGate` gaps, preserved under
  > `parameterized/backlog_red_test.go` (`redtests` tag) and queued as items
  > 18 and 19. Commits: 5f185dc, add554f, e47e46a.
- [x] 16. **`parameterShiftGradient` shifts missing parameters from an
  implicit zero** — when `params` lacks a name that appears in `names`, the
  shifted copies read the map's zero value, so the helper evaluates the
  +/- pi/2 points as if that parameter were 0 and returns a gradient with a
  nil error. Observed: with `params = {a: 0.3}` on a template declaring `a`
  and `b`, `names = [b]` returns `{b: 0}` and no error, while any `names`
  that shifts `a` errors because `Bind` then sees `b` missing; whether the
  call fails depends on the order of `names`. `VQE` always passes complete
  params, so nothing reaches this today, but the helper's contract is
  silent. Expected contract: a name in `names` absent from `params` is an
  error regardless of order, or the helper states that callers must pass
  every declared parameter.
  > **Red tests**: `TestRedParameterShiftMissingParamIsRejected` in
  > `algorithm/backlog_red_test.go`; reproduce with
  > `go test -tags redtests ./algorithm -run '^TestRedParameterShiftMissing'`.
  > **Done (2026-09-08)**: `parameterShiftGradient` in `algorithm/vqe.go`
  > begins with a completeness check over `t.ParamNames()` returning
  > `*parameterized.MissingParameterError{Name}` with 0 evaluations, in
  > declaration order like `Bind`, so a missing declared parameter is an
  > error regardless of `names` order. The check lives in the helper because
  > the shift itself hides the gap from `Bind` (a missing name reads as 0 and
  > is written into the ±π/2 copies). `names` stays a parameter (selects
  > which parameters to differentiate). `VQE` and the loop body untouched.
  > Doc comment states the contract as completeness-only, with value
  > validity and undeclared keys left to `Bind` at the first shifted
  > evaluation that reaches it (spec Decision 5, adopted after the gate's red
  > round). `MissingParameterError`'s doc in `parameterized/parameterized.go`
  > now states the condition rather than naming `Bind`; CHANGELOG Unreleased
  > entry; amendment notes in the 2026-08-30 spec and item 14's spec. Tests:
  > moved `TestParameterShiftRejectsMissingParam`, plus
  > `TestParameterShiftMissingParamErrorMatchesBind`,
  > `TestParameterShiftUndeclaredNameIsRejectedByBind`,
  > `TestParameterShiftDifferentiatesOnlyNamedParams` in
  > `algorithm/vqe_test.go`. Red round found a pre-existing float64 limit
  > (θ ± π/2 unrepresentable at huge magnitudes gives an exactly zero
  > gradient), preserved under `algorithm/backlog_red_numeric_test.go`
  > (`redtests` tag) and queued as item 20. Commits: 21626ff, 2079f68,
  > 84da0b7, 00529b9.
- [x] 17. **`parameterShiftGradient` undercounts evaluations on failure** —
  the helper adds two to its evaluation count only after both shifted
  evaluations succeed. Observed: when the +pi/2 evaluation succeeds and the
  -pi/2 evaluation fails, it returns 0 evaluations consumed though one ran.
  `VQE` discards the count on error, so the miscount is invisible today,
  but the doc promises "the number of energy evaluations consumed".
  Expected contract: the returned count includes every evaluation that ran,
  on the error path as well.
  > **Red tests**: `TestRedParameterShiftCountsEvaluationsBeforeFailure` in
  > `algorithm/backlog_red_test.go`; reproduce with
  > `go test -tags redtests ./algorithm -run '^TestRedParameterShiftCounts'`.
  > **Done (2026-09-08)**: `parameterShiftGradient` in `algorithm/vqe.go`
  > (lines 93–102 at 21626ff) now counts each shifted evaluation the moment
  > `evaluate` returns nil (`evals++` after each nil-error check, replacing
  > the single `evals += 2`), so the count returned with an error includes
  > every evaluation that completed and excludes the failing one. The count
  > is returned exact alongside the error rather than zeroed, because the
  > doc promises it and zeroing would turn an undercount into a larger one
  > (Decision 2, io.Reader precedent). `VQE` unchanged — it discards the
  > count on error and sums it on success, where the total is unchanged;
  > `go run ./cmd/quantum -demo qaoa` still reports 378 evaluations. Item 16's
  > early return of 0 stays exact since it precedes any evaluation. Doc
  > comment states the count contract and, after red round 1, that a repeated
  > name in `names` is evaluated and counted per occurrence (Amendment to
  > Decision 1). `algorithm/backlog_red_test.go` deleted since its last test
  > moved to the regular suite; item 20's test stays tagged in
  > `algorithm/backlog_red_numeric_test.go`. CHANGELOG Unreleased entry;
  > item 14 spec amendment. Tests: moved
  > `TestParameterShiftCountsEvaluationsBeforeFailure`, plus four-row
  > `TestParameterShiftEvaluationCountOnFailure` pinning counts 0/1/2/3
  > across failure positions. Commits: 9a0c2c0, 7828e60, 49445e7.
- [ ] 18. **Zero-value `Template` panics in `AddParamGate`** — a
  `Template` declared without `NewTemplate` (so its `seen` map is nil)
  panics on the first `AddParamGate` call instead of returning an error,
  because the call writes to `seen` unconditionally. Observed:
  `var tmpl parameterized.Template` then
  `tmpl.AddParamGate("", parameterized.Ry)` panics with "assignment to
  entry in nil map". Expected contract: `Template`'s exported methods
  never panic on a value the caller constructed with the zero value;
  either `NewTemplate` becomes a documented requirement enforced by every
  method, or a zero-value `Template` is a valid, safely usable state.
  > **Red tests**: `TestRedZeroValueTemplateAddParamGateDoesNotPanic` in
  > `parameterized/backlog_red_test.go`; reproduce with
  > `go test -tags redtests ./parameterized -run '^TestRedZeroValueTemplate'`.
- [ ] 19. **`AddParamGate` and `AddGate` accept an empty target list** —
  `checkTargets` loops over the given targets, so a call with none passes
  vacuously: the step is appended and later reported by `ParamNames` and
  `ParamStepCounts`, but `Bind` fails deep inside `circuit.AddGate` with
  "at least one target is required" instead of at the call that declared
  the gate. `AddGate` has the same gap as `AddParamGate` — neither method
  checks that at least one target was given. Expected contract: a gate
  declared with zero targets is rejected at the `Add*Gate` call, with an
  error naming the gate, not deferred to `Bind`.
  > **Red tests**: `TestRedAddParamGateRejectsNoTargets` in
  > `parameterized/backlog_red_test.go`; reproduce with
  > `go test -tags redtests ./parameterized -run '^TestRedAddParamGateRejectsNoTargets'`.
- [ ] 20. **`parameterShiftGradient` returns an exactly zero gradient where
  the +/- pi/2 shift is not representable** — the helper computes
  `theta + pi/2` and `theta - pi/2` in float64. From about 2^54 the spacing
  between adjacent float64 values exceeds pi, so both shifted values round
  back to `theta`, the two evaluations bind the same circuit, and the half
  difference is exactly 0 with a nil error; at somewhat smaller magnitudes
  the shift is rounded to a multiple of the spacing, so the rule is applied
  at the wrong offset. The energy is 2*pi-periodic in every angle, so the
  slope at the same angle reduced mod 2*pi is the true one. Observed:
  `a = 2^60`, `b = 0.1` on a template declaring `a` and `b` returns
  `{a: 0}` after 2 evaluations, where the reduced angle (about 4.1219) has
  slope 0.1019. Reachable through `VQE`: a huge finite `InitialParams`
  value passes the finiteness guard, that parameter then never moves, and
  the run reports `Converged` with it frozen at its start. Expected
  contract: either an error for a parameter whose shift is not
  representable, or a documented reduction of angles into a representable
  range before shifting; which one is not prescribed here.
  > **Red tests**: `TestRedParameterShiftAtHugeAngleMatchesReducedAngle` in
  > `algorithm/backlog_red_numeric_test.go`; reproduce with
  > `go test -tags redtests ./algorithm -run '^TestRedParameterShiftAtHugeAngle'`.
