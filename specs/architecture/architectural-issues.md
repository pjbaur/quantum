# Architectural Issues

Tracked findings from quality/architecture assessments. See
`specs/architecture/code_quality_assessment.md` for full context.

---

### Grover silently discards SetAmplitudes errors

**Skill:** code-quality-assessment
**Category:** RECOMMENDATION (P1)
**Status:** Resolved (2026-08-24) — both helpers now return `error`, checked in the iteration loop; covered by `TestGroverHelpersReturnSetAmplitudesErrors`.

**Issue:** `algorithm/grover.go:94,125` — `applyGroverOracle` and `applyGroverDiffusion` ignore the error returned by `state.SetAmplitudes`, which rejects updates whose probability sum drifts beyond 1e-10.

**Implication:** Accumulated floating-point drift in long Grover runs turns oracle/diffusion steps into silent no-ops; the function returns a wrong final state with no error. Deutsch-Jozsa propagates the same call's error correctly, so the codebase is internally inconsistent.

**Direction:** Make both helpers return `error` and check them in the iteration loop, mirroring `applyDeutschJozsaOracle`.

---

### Sparse backend dispatches CNOT by gate name, ignoring the matrix

**Skill:** code-quality-assessment
**Category:** Subsystem concern (internal/sparsestate ★★★☆☆, correctness hazard)
**Status:** Resolved (2026-08-24) — fast path now selected by `isCanonicalCNOT` matrix comparison with fallthrough to the general 4×4 path; covered by `TestSparseGateNamedCNOTUsesItsMatrix` (dense/sparse equivalence with a CZ-matrix gate named "CNOT").

**Issue:** `internal/sparsestate/state.go:162` takes a fast path when `gate.Name() == "CNOT"` and never reads the gate's matrix.

**Implication:** Any user-defined `Gate` named "CNOT" with a different 4×4 matrix silently gets standard-CNOT semantics. Magic-literal dispatch on user-controlled identity.

**Direction:** Detect the fast path by matrix content (compare against the canonical CNOT matrix) with fallthrough to the general 4×4 path; add dense/sparse equivalence tests.

---

### qubit package has zero test coverage

**Skill:** code-quality-assessment
**Category:** Subsystem concern (qubit ★★☆☆☆)
**Status:** Resolved (2026-08-25) — `qubit/qubit_test.go` added, 100% statement coverage; README testing section corrected. `Measure` corner cases and collapse are deterministic tests, distribution is a coarse ±6σ statistical check; exact assertions blocked on the injectable-randomness issue below. The 1e-6 vs 1e-10 tolerance inconsistency is characterized by `TestIsNormalizedStricterThanSet`, not fixed.

**Issue:** `qubit/` has no test file and 0% statement coverage. README line 113 claims qubit operations are covered in `quantum/quantum_test.go`; coverage data contradicts this.

**Implication:** Core public type's `Measure` collapse semantics, `Set` tolerance boundary (1e-6, inconsistent with the backends' 1e-10), and `Clone` independence are unverified.

**Direction:** Add `qubit/qubit_test.go`; deterministic `Measure` assertions want injectable randomness first.

---

### Density backend is orphaned and violates ADR-0004

**Skill:** code-quality-assessment
**Category:** Subsystem concern (internal/density ★★☆☆☆)
**Status:** Resolved (2026-08-25) — wired in: `New` now returns `(*Matrix, error)` with `InvalidQubitCountError` per ADR-0004; added `Purity()` and `ReducedBlochVector(target)` (partial-trace Bloch export for mixed states); new `noise` CLI demo (`internal/examples/noise.go`) exercises dephasing, amplitude damping, and depolarizing channels; CLI drift tests and README updated. `quantum.QuantumState` conformance deliberately not added — the amplitude-based interface (`Amplitude`/`SetAmplitude`) has no meaning for mixed states.

**Issue:** `internal/density` (271 LOC, 79.5% coverage) has zero importers outside its own tests. `New` coerces `numQubits <= 0` to 1 (`state.go:25-27`) instead of returning `InvalidQubitCountError` as ADR-0004 requires and the other backends do. It does not implement `quantum.QuantumState`.

**Implication:** Tested capability unreachable from any public API or demo; constructor behavior contradicts the project's own documented decision.

**Direction:** Either wire it in (noise-channel demo, Bloch export for mixed states) with an ADR-0004-conformant constructor, or delete it.

---

### Dead and speculative code across gates/ and algorithm/

**Skill:** code-quality-assessment
**Category:** RECOMMENDATION (P1)
**Status:** Resolved (2026-08-25) — data-driven route per the issue's recommendation: `matrixGate` promoted to exported `gates.MatrixGate` (validated constructor, deep-copying `Matrix()`, `NumQubits()`); the eight concrete gate structs replaced by canonical matrix tables behind unchanged constructor names (golden tests lock exact values); `gates.Builtin()`/`Registry.Names()` added and consumed, with `DecomposeSwap`, by the new `gates` CLI demo; `algorithm/algorithm.go` (incl. `descendingTargets` and the v1 `Apply`) deleted; empty `measurement/` directory removed and README TODO dropped. Design: `docs/superpowers/specs/2026-08-25-data-driven-gates-design.md`.

**Issue:** staticcheck U1000 ×6 in `algorithm/algorithm.go` (`matrixGate`, `newMatrixGate`, three methods, `descendingTargets`); `gates.Registry` and `gates.DecomposeSwap` have zero consumers; empty `measurement/` directory persists against a README TODO.

**Implication:** ~400 LOC of unwired surface; reviewers and readers pay for API that does nothing. `matrixGate.Apply(Qubit)` implements a retired v1 interface.

**Direction:** Delete, or connect each to a consumer. If gates become data-driven (recommended), promote the `matrixGate` design instead of deleting it.

---

### CI gates lag the project's actual standard

**Skill:** code-quality-assessment
**Category:** Cross-cutting (Pre-Commit & CI Pipeline ★★★☆☆)
**Status:** Resolved (2026-08-25) — coverage threshold 15% → 40% (actual 44.8%); `go test -race ./...` step added (parallel executor now runs under the detector in CI); staticcheck step added and clean (the two `rand.Seed` SA1019 findings removed along with the calls — global rand self-seeds since Go 1.20); Go bumped from EOL 1.21 to `go 1.25` in go.mod with a CI matrix of 1.25.x/1.26.x; README prerequisite updated.

**Issue:** Coverage threshold 15% vs 43.9% actual; no `-race` in CI despite a parallel executor with race-focused tests; no staticcheck/golangci-lint (8 findings pass CI today); Go pinned to EOL 1.21; deprecated `rand.Seed` at `cmd/quantum/main.go:111` and `internal/examples/hadamard.go:203`.

**Implication:** CI is green while shipping dead code and deprecations; the race suite never runs under the race detector where it matters; the coverage gate protects nothing.

**Direction:** Threshold →40%, add `go test -race ./...` and staticcheck, bump Go, remove `rand.Seed` calls.

---

### No LICENSE and no version tags despite v2 docs

**Skill:** code-quality-assessment
**Category:** RECOMMENDATION (P1)
**Status:** Resolved (2026-08-25) — MIT LICENSE added (Copyright (c) 2026 Paul Baur); annotated tag `v0.2.0` created (v0.x series chosen over a literal `/v2` module path — the v2 docs describe an API generation, not a module major version; noted in MIGRATION-v2.md and a new README Versioning section). Tag is local until pushed (`git push origin main --tags`).

**Issue:** Public GitHub repo with no LICENSE file; `git tag` is empty while `docs/MIGRATION-v2.md` and `docs/deprecation-policy-v2.md` describe a versioned v2.

**Implication:** All-rights-reserved by default — the README invites cloning that the license status doesn't permit; no `go get`-able stable version; versioning docs promise what the repo doesn't do.

**Direction:** Add MIT/Apache-2.0; tag a release (v0.x, or v2.0.0 + `/v2` module path if the v2 docs are meant literally).

---

### Measurement bound to global math/rand — no determinism

**Skill:** code-quality-assessment
**Category:** Cross-cutting

**Issue:** `state.Measure`, `sparsestate.Measure`, and `qubit.Measure` call `rand.Float64()` on the global source; no seam exists to inject a seeded source.

**Implication:** Simulations are non-reproducible (table stakes for a simulator); measurement behavior can only be tested statistically.

**Direction:** Optional rand source per state/qubit (`WithRand(*rand.Rand)` or small `Source` interface), defaulting to current behavior.
