# Code Quality Assessment Report

Date: 2026-08-24 (commit 4fa3a81)
Repository: github.com/pjbaur/quantum ("Schrödinger's Gopher")
Mode: Full Assessment

## Repository Metrics Dashboard

- **Production Code**: 3,396 lines of Go across 25 files
- **Test Code**: 3,961 lines across 20 test files (1.17:1 test-to-production ratio)
- **Test Functions**: 85 tests, 17 benchmarks, 5 examples
- **Documentation**: ~2,900 lines across ~20 files (README, 11 docs, 7 ADRs, AGENTS.md)
- **Dependencies**: 0 direct (stdlib-only — exemplary for Go)
- **CI/CD**: GitHub Actions — gofmt, go vet, go test, coverage report with 15% threshold gate. No lint (golangci-lint/staticcheck), no `-race`, no build verification of `cmd/`
- **Pre-commit**: none configured
- **Hygiene**: 0 TODO/FIXME, 0 panics, 0 nolint suppressions, 3 `interface{}`/`any` uses; gofmt and go vet clean

## Executive Summary

An educational quantum-computing simulator in pure Go: dense and sparse state-vector backends, a density-matrix backend, a gate/circuit abstraction, parallel batch execution, and Grover/Deutsch-Jozsa implementations. Engineering discipline is well above hobby-project norm — typed error taxonomy, ADRs, a completed critical-review remediation cycle (docs/critical-review-2026-02-21.md → resolved-findings-summary.md), race-focused parallel tests, and even an AST-based guard test against debug prints. The main weaknesses are unfinished integration (three components — density backend, gate registry, SWAP decomposition — are built and tested but wired to nothing), silent error swallowing in Grover, a stringly-typed dispatch hazard in the sparse backend, an untested core `qubit` package, and release hygiene gaps (no LICENSE, no version tags despite v2 migration/deprecation docs).

**Key Strengths:**
- Zero-dependency stdlib-only module with a clean typed error taxonomy (`quantum/errortypes.go`: 8 structured error types, all carrying diagnostic fields, with rollback semantics documented on `NormalizationError`)
- Demonstrated quality feedback loop: a prior critical review was tracked to full resolution across 5 workstreams, with ADRs (7) capturing the decisions — rare at this scale
- Parallel execution is defensively engineered: `circuit/parallel.go` validates state-pointer independence before spawning workers (`SharedStateError`), uses `sync.Once` + `atomic.Bool` for first-error capture, and is backed by 390 lines of race-focused tests run under `-race`
- Algorithms avoid O(4ⁿ) matrix construction: Grover oracle/diffusion and the DJ oracle operate directly on amplitudes in O(2ⁿ), with the complexity trade-off documented in comments
- Healthy 1.17:1 test-to-production ratio with benchmarks and committed profiles for performance work

**Areas for Improvement:**
- `algorithm/grover.go:94,125` discards `SetAmplitudes` errors — after enough iterations for float drift to exceed the 1e-10 normalization tolerance, oracle/diffusion steps silently become no-ops (Deutsch-Jozsa propagates the same error correctly)
- `internal/sparsestate/state.go:162` dispatches on `gate.Name() == "CNOT"` and ignores the gate's actual matrix — any user-defined gate named "CNOT" gets standard-CNOT semantics regardless of its matrix
- `qubit/` has no test file and 0% coverage — a core public package (README even claims its tests live in `quantum_test.go`)
- Speculative/orphaned code: `internal/density` (271 LOC, 14 functions) has zero importers; `gates.Registry` and `gates.DecomposeSwap` have zero consumers; `algorithm/algorithm.go` carries 6 staticcheck-confirmed unused declarations
- Measurement uses the global `math/rand` with no injectable source — simulations are non-reproducible and measurement statistics are hard to test deterministically; `rand.Seed` (deprecated since Go 1.20) still used in `cmd/quantum/main.go:111` and `internal/examples/hadamard.go:203`
- Release hygiene: no LICENSE file (public GitHub repo = all rights reserved), no git tags despite MIGRATION-v2/deprecation-policy-v2 docs implying a versioned v2, coverage gate at 15% vs 43.9% actual (enforcement theater), Go pinned to EOL 1.21

**Overall Rating: B+ (Good — solid engineering with a working quality loop; the deduction from A- is for the swallowed Grover errors, the name-based CNOT dispatch, the untested qubit package, and the accumulation of built-but-unwired components)**

---

## Detailed Subsystem Analysis

### Core Interfaces (`quantum/`) ★★★★☆

**Strengths:**
- Small, focused interface layer: `Qubit`, `Gate`, `QuantumState`, `BackendCapabilities` (80 LOC) with every method documented; `Gate` correctly reduced to metadata-only after the v1 `Apply(Qubit)` design was retired (ADR-backed)
- `GateQubitCount` centralizes matrix validation (square, power-of-two) — extracted from three duplicated copies during the prior review cycle
- 66.7% coverage plus a novel AST-walking guard test (`no_debug_prints_test.go`) that fails the build if `fmt.Print*` appears in test files

**Concerns:**
- `GateQubitCount` accepts a 1×1 matrix and returns 0 qubits without error (`gateutil.go:35`: `size&(size-1)` passes for size=1) — a degenerate gate flows through to a confusing `InvalidGateApplicationError` downstream instead of `InvalidGateMatrixError` at the source
- No RNG abstraction in the interface layer — `Measure` implementations bind directly to the global `math/rand`, so determinism cannot be injected (see cross-cutting)
- `GateQubitCount` re-validates the full matrix on every `ApplyGate` call — validation cost is per-operation, not per-gate (minor at current scale)

### Dense State Backend (`state/`) ★★★★☆

**Strengths:**
- Correct and careful bit-twiddling for single- and multi-qubit gate application with scratch-buffer reuse (`scratch`, `comboMasks`, `inputs`, `outputs`) to avoid per-operation allocation — informed by committed pprof profiles
- Target validation adapts data structure to scale (`state.go:132-161`: uint64 bitmask under 64 qubits, map above)
- `SetAmplitude` rolls back on normalization failure and reports both attempted and restored sums; bulk `SetAmplitudes` amortizes the normalization check

**Concerns:**
- `math.Pow(cmplx.Abs(x), 2)` in O(2ⁿ) loops (`probabilitySum`, `Measure`, `Probability`) — a `Pow` call per amplitude where `re*re+im*im` (or `abs*abs`) suffices; 14 occurrences across the module
- `Measure` (`state.go:288-332`) can divide by zero: if float rounding leaves `prob0` marginally below 1.0 while all set-bit amplitudes are exactly zero, `rand.Float64() >= prob0` can select result=1, `normalizationFactor` is 0, and the state fills with NaN — no guard, no error path
- 55.5% statement coverage is the lowest of the three backends despite being the primary one
- Normalization tolerance is 1e-10 here but 1e-6 in `qubit.Set` (`qubit.go:51`) — same physical invariant, two thresholds, no documented rationale

### Sparse State Backend (`internal/sparsestate/`) ★★★☆☆

**Strengths:**
- Genuine sparse representation (map of non-zero amplitudes) with epsilon pruning (`pruneEpsilon = 1e-12`) and an O(non-zero) CNOT fast path; 80.9% coverage, best in module, plus comparison benchmarks against dense
- Correctly declares its limits via `BackendCapabilities` (1–2 qubit gates), letting `circuit.Execute` fail fast with `UnsupportedOperationError`

**Concerns:**
- `applyTwoQubitGate` (`state.go:162`) special-cases on the string `gate.Name() == "CNOT"` and never reads the matrix — a custom `Gate` named "CNOT" with any other 4×4 matrix silently gets standard-CNOT semantics. Magic-literal dispatch on user-supplied identity; correct fix is matrix-shape detection (permutation-matrix check) or comparing against the known CNOT matrix, not a name
- Duplicates dense-backend logic wholesale: target validation, `buildComboMasks` vs `state.go`'s inline combo-mask loop, measurement collapse, normalization — two implementations of the same math that can drift (the prior review fixed exactly this class of drift for `GateQubitCount`)
- No bulk `SetAmplitudes` — interface asymmetry with the dense backend means `algorithm/` cannot run on sparse states at all (Grover/DJ take `*state.State` concretely)

### Density Matrix Backend (`internal/density/`) ★★☆☆☆

**Strengths:**
- Only backend supporting noise channels (depolarizing, dephasing, amplitude damping) via Kraus operators, with scratch-buffer discipline matching the dense backend; 79.5% coverage

**Concerns:**
- **Orphaned**: zero importers outside its own tests — not reachable from any public API, example, or demo. 271 LOC of tested capability wired to nothing
- `New` silently coerces `numQubits <= 0` to 1 (`state.go:25-27`) — directly violates ADR-0004 (diagnostics-constructor-consistency), which the other two backends follow by returning `InvalidQubitCountError`
- Does not implement `quantum.QuantumState` (no `ApplyGate(gate, targets...)`, `Measure`, `Clone`), so it cannot participate in the circuit abstraction even if exposed

### Circuit (`circuit/`) ★★★★☆

**Strengths:**
- Validation at both add-time (`AddGate`: gate arity, target range, uniqueness) and execute-time (qubit-count match, backend capability pre-check) — errors surface before any state mutation
- `ExecuteAllParallel` enforces state independence up front and degrades to serial for `workers == 1`; worker pool with first-error semantics is idiomatic and race-tested (77.2% coverage, 390-line parallel test suite)

**Concerns:**
- `checkCapabilities` only consults `SupportsGateQubits`; `MaxGateQubits` exists on the interface but nothing calls it — half the `BackendCapabilities` contract is dead on arrival
- `validateIndependentStates` uses `QuantumState` interface values as map keys — panics at runtime if a caller implements the interface with an uncomparable type (unlikely, undocumented)

### Gates (`gates/`) ★★★☆☆

**Strengths:**
- Correct matrices for H, X, Y, Z, S, T, CNOT, SWAP; `ComposeMatrices`/`TensorProduct` utilities validate squareness and dimension compatibility

**Concerns:**
- Imperative ceremony: 8 near-identical struct+constructor+Name+Matrix quadruplets (180 LOC that could be ~40) — this is data expressed as code. Ironically, the right design already exists in the codebase as `algorithm.matrixGate` (name + matrix + qubit count), but it is unexported and unused
- `Matrix()` allocates a fresh `[][]complex128` on every call — every `ApplyGate` triggers 1–2 allocations (once in `GateQubitCount`, once in the apply path); Hadamard additionally recomputes `math.Sqrt(2)` four times per call
- Gate set is thin for the algorithm ambitions: no rotation gates (Rx/Ry/Rz), no phase gate with parameter, no Toffoli — one reason Grover's oracle/diffusion bypass the gate model entirely
- `Registry` (registry.go) and `DecomposeSwap` (decompose.go) have zero consumers outside their own tests — speculative API surface
- 57.8% coverage

### Algorithms (`algorithm/`) ★★★☆☆

**Strengths:**
- O(2ⁿ) oracle/diffusion implementations with the asymptotic trade-off documented; correct iteration count ⌊π/4·√(N/M)⌉ with multi-marked-state support and deduplication; negative-path tests added during the prior review cycle

**Concerns:**
- `applyGroverOracle` and `applyGroverDiffusion` discard the `SetAmplitudes` error (`grover.go:94,125`). `SetAmplitudes` rejects updates whose probability sum drifts beyond 1e-10 — so accumulated float error in a long Grover run turns iterations into silent no-ops and returns a wrong state with no error. `applyDeutschJozsaOracle` propagates the same call's error correctly (`deutsch_jozsa.go:103`); the inconsistency is one `if err :=` away
- Coupled to the dense backend concretely: `Grover`/`DeutschJozsa` accept and return `*state.State`, not `quantum.QuantumState` — the backend abstraction stops at the algorithm layer
- Dead code: `matrixGate`, `newMatrixGate`, its three methods, and `descendingTargets` are all unused (staticcheck U1000 ×6) — remnants of the pre-refactor matrix-construction approach, including an `Apply(Qubit)` method implementing an interface that no longer exists
- Grover's oracle and diffusion mutate amplitudes directly rather than applying gates — pedagogically significant for a teaching simulator (the "circuit" never sees the algorithm), and unmentioned in docs
- 53.1% coverage

### Qubit (`qubit/`) ★★☆☆☆

**Strengths:**
- Minimal, correct single-qubit implementation with normalization enforcement on `Set` and rollback-free constructor validation

**Concerns:**
- **Zero test coverage, no test file** — `Measure`'s collapse semantics, `Set`'s tolerance boundary, and `Clone` independence are all unverified. README (line 113) claims qubit operations are covered in `quantum/quantum_test.go`; coverage data says otherwise
- Normalization tolerance 1e-6 vs the state backends' 1e-10 (see dense-state concern)
- `Measure` uses global `math/rand` — same determinism gap as the backends

### CLI & Examples (`cmd/quantum/`, `internal/examples/`) ★★★☆☆

**Strengths:**
- Clean flag/positional handling with conflict detection and correct exit codes; CLI contract covered by `main_test.go` (164 LOC) per ADR-0006; examples are `internal/`, correctly excluded from the public API

**Concerns:**
- `runDemos("all")` blocks on `fmt.Scanln()` between sections (`main.go:56-69`) — `quantum all` hangs forever when piped or run in CI; no flag to disable interaction, and `Scanln` errors are discarded
- Deprecated `rand.Seed` at `main.go:111` and `hadamard.go:203` (staticcheck SA1019); seeding is redundant since Go 1.20
- 1,082 LOC of examples (0% coverage, acceptable for demos) with heavy copy-paste between `bell.go`, `hadamard.go`, `tgate.go` banner/formatting code — `utils.go` exists but captures little of it

### Visualization (`visualization/`) ★★★★☆

**Strengths:**
- Clean options-struct API (`StateViewOptions` with defaults constructor), nil-safe formatting, CSV export for external plotting; 78.5% coverage with direct formatting tests

**Concerns:**
- `DefaultStateViewOptions` itself is 0% covered (trivial, but it is the documented entry point)
- Bloch-vector export exists only for single `Qubit`s — no reduced-density-matrix path from a multi-qubit state, which the orphaned `density` package could provide

---

## Testing & Quality Infrastructure ★★★★☆

**Strengths:**
- 85 tests / 17 benchmarks / 5 Example functions at a 1.17:1 test:prod ratio; race-focused parallel suite; comparison benchmarks between dense and sparse backends; AST-based debug-print guard; CLI contract tests backed by ADR-0006
- Prior-review remediation added negative-path tests systematically (constructor errors, nil oracles, out-of-range marked states)

**Concerns:**
- Coverage is bimodal: sparse 80.9%, density 79.5%, visualization 78.5%, circuit 77.2% — versus qubit 0%, algorithm 53.1%, state 55.5%. The best-covered backend (density) is unreachable; the primary backend (state) is the worst-covered
- Measurement is statistically tested at best — with no injectable RNG, collapse behavior can't be asserted deterministically (e.g. seeded sequences pinning exact outcomes)
- No fuzz tests despite ideal targets (`GateQubitCount`, `SetAmplitudes`, gate application against random unitaries)

## Pre-Commit & CI Pipeline ★★★☆☆

**Strengths:**
- CI runs on every push/PR: gofmt (blocking), go vet, full test suite, coverage computation with a blocking threshold

**Concerns:**
- Coverage threshold is 15% against an actual 43.9% — the gate would only trip after catastrophic test deletion; it enforces nothing about the current standard
- No `-race` in CI despite the project shipping a parallel executor whose safety was a prior top-priority finding — the race tests exist but CI never runs them under the race detector
- No staticcheck/golangci-lint — CI is green while staticcheck reports 8 issues (6 dead-code, 2 deprecations) locally
- Go pinned to 1.21.x (EOL since Go 1.23's release); `go.mod` says `go 1.21`
- Tests run twice (once plain, once for coverage) — minor CI waste; no pre-commit hooks configured

## Documentation & Specifications ★★★★☆

**Strengths:**
- ~2,900 lines: README with runnable commands, 7 ADRs with a README index, migration guide, deprecation and compatibility policies, sparse-state design doc, and a fully closed review→resolution→summary loop — exceptional documentation culture for a solo educational project
- AGENTS.md gives coding agents precise, sensible working rules (smallest change, no API renames, table-driven tests)

**Concerns:**
- Versioning docs promise what the repo doesn't do: MIGRATION-v2.md and deprecation-policy-v2.md imply a released, tagged v2 — `git tag` is empty and the module has never been versioned. A consumer cannot `go get` any stable version
- **No LICENSE file** — public GitHub repository defaults to all-rights-reserved; nobody can legally use, copy, or learn-by-forking the code the README invites them to clone
- README drift: claims qubit tests live in `quantum_test.go` (they don't); "a dedicated `measurement` package is TODO" — the empty `measurement/` directory has sat untracked since Apr 2025
- Committed pprof binaries (`CHANGES/profiles/*.pprof`) with a README workflow that regenerates them into the repo — binary churn in version control; profiles belong in a gitignored path or CI artifacts

---

## Refactoring Recommendations by Priority

### Priority 1: High Impact / Low Risk

> **Status (2026-08-25)**: All five items completed and verified on `main` as of v0.3.0 (846268f).

#### 1.1 Propagate `SetAmplitudes` errors in Grover
- **What**: `algorithm/grover.go` — make `applyGroverOracle`/`applyGroverDiffusion` return `error` (mirroring `applyDeutschJozsaOracle`) and check them in the iteration loop.
- **Risk**: Low — signature change on two unexported functions.
- **Impact**: Eliminates the only silent-wrong-answer path found in the module; restores consistency with DJ.
- **Result (2026-08-25, verified at v0.3.0 / 846268f)**: ✅ Done. Both helpers return `error` and the iteration loop checks each (`algorithm/grover.go:56-66,98,122`); no discarded `SetAmplitudes` calls remain in `algorithm/`. Regression test `TestGroverHelpersReturnSetAmplitudesErrors` asserts the run stops at the first refusal. Commits ae4decb, later reworked by 9e8654f.

#### 1.2 Add a LICENSE and tag a release
- **What**: Choose a license (MIT/Apache-2.0 typical for educational Go), commit LICENSE, tag `v0.x.y` (or `v2.0.0` + `/v2` module path if the v2 docs are meant literally).
- **Risk**: Low — no code change.
- **Impact**: Makes the project legally usable and `go get`-able; reconciles the v2 migration/deprecation docs with reality.
- **Result (2026-08-25, verified at v0.3.0 / 846268f)**: ✅ Done. MIT `LICENSE` committed at repo root (commit fe4cf9c, with versioning policy); releases tagged `v0.2.0` and `v0.3.0` — HEAD is the v0.3.0 release commit.

#### 1.3 Delete dead and orphaned code (or wire it in)
- **What**: Remove `algorithm/algorithm.go`'s `matrixGate`/`descendingTargets` (staticcheck-confirmed unused). Decide the fate of `internal/density`, `gates.Registry`, `gates.DecomposeSwap`: delete, or connect (density → a noise demo; DecomposeSwap → a decomposition example). Delete the empty `measurement/` dir or create the package README promises.
- **Risk**: Low — nothing references any of it.
- **Impact**: ~400 LOC of unwired surface stops taxing readers and reviews; "built means reachable" becomes true again.
- **Result (2026-08-25, verified at v0.3.0 / 846268f)**: ✅ Done. `algorithm/algorithm.go` deleted (its `matrixGate` design was promoted to `gates/matrixgate.go`, now widely consumed). Everything else was wired in rather than deleted: `internal/density` → noise demo (`internal/examples/noise.go:24`), `gates.Registry` → `quantum gate <name>` CLI lookup (`cmd/quantum/main.go:47`, commit 9b627da), `DecomposeSwap` → SWAP demo (`internal/examples/gates.go:99`). No `measurement/` directory remains; `staticcheck ./...` and `go vet ./...` clean.

#### 1.4 Raise CI to the standard the project already meets
- **What**: `.github/workflows/ci.yml` — coverage threshold 15→40; add `go test -race ./...`; add staticcheck (it's already clean except the 8 real findings); bump Go to a supported version (1.23+/stable matrix); replace `rand.Seed` calls (SA1019).
- **Risk**: Low — all gates pass today after 1.3 and the two-line Seed fixes.
- **Impact**: CI stops being green-while-stale; the race suite actually runs under the race detector; the coverage floor protects the actual standard.
- **Result (2026-08-25, verified at v0.3.0 / 846268f)**: ✅ Done. `.github/workflows/ci.yml` now: coverage threshold 40.0 (actual 48.0%), dedicated `go test -race ./...` step, staticcheck step (on newest matrix Go), Go matrix `1.25.x`/`1.26.x` (`go.mod` says `go 1.25`). Zero `rand.Seed` occurrences left in code — measurement randomness became injectable (P2 item 2.3). Commits f10d4f2, a03cad2, a2055ba.

#### 1.5 Replace `math.Pow(cmplx.Abs(x), 2)` with direct computation
- **What**: 14 sites across `state/`, `sparsestate/`, `qubit/` — introduce one shared `probability(c complex128) float64 { re,im := real(c), imag(c); return re*re + im*im }` (natural home: `quantum` package).
- **Risk**: Low — mechanical, benchmark-verifiable.
- **Impact**: Removes a transcendental-function call per amplitude from every O(2ⁿ) measurement/normalization loop; also unifies the operation the two backends currently duplicate.
- **Result (2026-08-25, verified at v0.3.0 / 846268f)**: ✅ Done. Shared helper `quantum.Probability` (`quantum/probability.go:10`, commit 9104be0) replaced all 14 sites across `state/`, `internal/sparsestate/`, and `qubit/`. Zero `math.Pow(cmplx.Abs` left in production code — the only remaining uses are in `quantum/probability_test.go`, deliberately asserting the helper matches the old formula.

### Priority 2: Medium Impact / Medium Risk

> **Status (2026-08-25)**: All five items completed and verified on `main` as of v0.3.0 (846268f); one documented deviation on 2.4 (defensive-copy `Matrix()` contract kept in place of matrix caching). Two residual cleanups applied in this pass: `invSqrt2` hoisted in `gates/gates.go`, stale RNG comment fixed in `qubit/qubit_test.go`.

#### 2.1 Test the `qubit` package
- **What**: Add `qubit/qubit_test.go`: `Set` tolerance boundaries (including the 1e-6 vs 1e-10 question), `NewWithValues` error paths, `Clone` independence, `Measure` collapse post-conditions; fix or remove the README claim.
- **Risk**: Low-Medium — tests only, but `Measure` assertions want 2.3 first.
- **Impact**: Closes the only zero-coverage public package; likely forces the tolerance-inconsistency decision.
- **Result (2026-08-25, verified at v0.3.0 / 846268f)**: ✅ Done. `qubit/qubit_test.go` added (commit 4d55bcb, extended by 3e97b15) — package coverage now **100.0%**. Covers `Set` tolerance boundaries (9e-7 accepted / 2e-6 rejected, rejected `Set` leaves state unmutated), `NewWithValues` error paths, `Clone` independence + rand-source inheritance, and deterministic `Measure` collapse via injected `RandomSource`. The 1e-6/1e-10 tolerance question was decided by characterization, not unification: `TestIsNormalizedStricterThanSet` pins both thresholds, rationale tracked in `specs/architecture/architectural-issues.md`. README claim fixed — now links `qubit/qubit_test.go`.

#### 2.2 Fix sparse CNOT name-dispatch
- **What**: `internal/sparsestate/state.go:162` — replace `gate.Name() == "CNOT"` with a matrix check (compare against the canonical CNOT matrix, or detect permutation structure) before taking the fast path; fall through to the general 4×4 path otherwise.
- **Risk**: Medium — hot path; needs equivalence tests between fast and general paths.
- **Impact**: Removes the one correctness trap for user-defined gates; eliminates magic-literal dispatch on user-controlled identity.
- **Result (2026-08-25, verified at v0.3.0 / 846268f)**: ✅ Done. Fast path now selected by matrix content: `isCanonicalCNOT` compares element-wise against a package-level canonical table (`internal/sparsestate/state.go:237,295-318`); `gate.Name()` survives only in error messages. Regression test `TestSparseGateNamedCNOTUsesItsMatrix` (impostor gate named "CNOT" carrying a CZ matrix must match dense), plus dense/sparse equivalence tests and `FuzzDenseSparseGateEquivalence` whose seeds include the canonical-CNOT fast-path case. Commit 4ec82ca.

#### 2.3 Injectable randomness for measurement
- **What**: Give `state.State`, `sparsestate.State`, and `qubit.Qubit` an optional rand source (e.g. `WithRand(*rand.Rand)` option or a package-level `Source` interface defaulting to global rand). Remove the deprecated `rand.Seed` calls.
- **Risk**: Medium — touches all Measure paths; additive API.
- **Impact**: Reproducible simulations (table stakes for a simulator) and deterministic measurement tests; unblocks 2.1's strongest assertions.
- **Result (2026-08-25, verified at v0.3.0 / 846268f)**: ✅ Done. `quantum.RandomSource` interface (`quantum/interfaces.go:47`, satisfied by `*math/rand.Rand`) with `SetRandSource` on `state.State`, `sparsestate.State`, and `qubit.Qubit`; `Clone` propagates the source in all three; nil restores the global default. Deterministic tests exist in all three packages (stub-source exact outcomes, seeded dense/sparse equivalence via `TestSeededMeasurementDenseSparseEquivalence`, fuzz harnesses inject stubs). Zero `rand.Seed` occurrences remain. Commit 3e97b15.

#### 2.4 Make gates data, not ceremony
- **What**: Replace the 8 boilerplate types in `gates/gates.go` with one matrix-backed gate type (promote the design of the dead `algorithm.matrixGate`) plus package-level constructors; cache matrices as package vars (return defensive copies or document immutability) so `Matrix()` stops allocating per call.
- **Risk**: Medium — public type names change (`*HadamardGate` → shared type); constructors can be kept source-compatible.
- **Impact**: ~140 LOC removed; adding Rx/Ry/Rz/Toffoli becomes a one-liner each; kills the per-`ApplyGate` allocation pair.
- **Result (2026-08-25, verified at v0.3.0 / 846268f)**: ✅ Done, with one documented deviation. The 8 struct+constructor+Name+Matrix quadruplets are gone — one exported `MatrixGate` type (`gates/matrixgate.go`, commits 952ef01 + 990c8e4), every built-in a one-line `mustGate` call; the one-liner claim was proven when Rx/Ry/Rz/Phase/Toffoli landed as single constructors (cb06d2c). Package coverage 92.1%. **Deviation**: matrices are not cached as package vars — `Matrix()` deliberately returns a defensive deep copy per call, a safety-over-allocation contract chosen in the design spec (`docs/superpowers/specs/2026-08-25-data-driven-gates-design.md`) and pinned by `TestMatrixGateMatrixIsDeepCopy`; the per-call allocation therefore remains by design. `math.Sqrt(2)` recomputation moved from per-`Matrix()`-call to construction-time in the refactor; the residual 4× recompute inside `NewHadamard` was hoisted to a package-level `invSqrt2` var in this pass (bit-identical value preserved).

#### 2.5 Guard the zero-probability measurement branch
- **What**: In both backends' `Measure`, if the selected branch's `normalizationFactor` is 0 (or below epsilon), select the other outcome (or return an internal-consistency error) instead of dividing.
- **Risk**: Low-Medium — edge-case semantics need a comment and a test with adversarial amplitudes.
- **Impact**: Removes the NaN-state failure mode; documents the invariant instead of assuming it.
- **Result (2026-08-25, verified at v0.3.0 / 846268f)**: ✅ Done. Guard lives in shared `internal/backendmath.PlanCollapse` with `minBranchProbability = 1e-24` (= sparse `pruneEpsilon`²): a drawn branch below the floor flips to the other outcome; both below the floor returns an internal-consistency error naming the qubit. Edge case documented at three sites (backendmath rationale + per-backend pointers); adversarial tests in both backends (`TestMeasureZeroProbabilityBranch` with amplitudes `{1-1e-16, 1e-200}` and a `Nextafter(1,0)` draw, sparse variant, empty-state error case) plus `TestPlanCollapseFloorIsPruneEpsilonSquared`. Commit 33376b8, later centralized into backendmath by e484434.

### Priority 3: Strategic / Long-term

> **Status (2026-08-25)**: All four items completed and verified on `main` as of v0.3.0 (846268f). Known residuals, recorded per item: some backend math beyond 3.2's stated scope remains duplicated, and 3.4's fuzz targets run in CI over seed corpora only (no extended `-fuzz` step).

#### 3.1 Decouple algorithms from the dense backend
- **What**: Have `Grover`/`DeutschJozsa` accept a `quantum.QuantumState` (or a state factory), moving bulk amplitude access behind an optional `AmplitudeBatcher` interface that `state.State` already satisfies via `SetAmplitudes`.
- **Risk**: Medium-High — public signatures change; sparse backend needs bulk ops to participate.
- **Impact**: The backend abstraction reaches the layer users actually call; enables sparse-backend Grover for few-marked-state instances (its best case).
- **Result (2026-08-25, verified at v0.3.0 / 846268f)**: ✅ Done. `Grover` and `DeutschJozsa` now take and return `quantum.QuantumState` (`algorithm/grover.go:21`, `algorithm/deutsch_jozsa.go:25`); bulk access lives behind `quantum.BulkAmplitudeSetter` (`quantum/interfaces.go:90` — the interface shipped under this name, not the proposed "AmplitudeBatcher"), kept out of `QuantumState` deliberately. Sparse backend gained `SetAmplitudes` and runs both algorithms — backend-table tests plus cross-backend equivalence (`TestGroverBackendsAgree`, `TestDeutschJozsaBackendsAgree`). A backend without bulk ops gets a typed `*quantum.UnsupportedOperationError` (tested). Shipped as a documented breaking change in v0.3.0 (`BREAKING CHANGE:` trailer, before/after in CHANGELOG). Package coverage 53.1% → **92.2%**. Commit 9e8654f.

#### 3.2 Extract shared backend math
- **What**: Factor duplicated logic between `state/` and `internal/sparsestate/` (target validation, combo-mask construction, collapse-and-renormalize) into an internal package.
- **Risk**: Medium — refactor across both hot paths, benchmark before/after.
- **Impact**: One implementation of the physics; drift class the prior review fixed once cannot recur.
- **Result (2026-08-25, verified at v0.3.0 / 846268f)**: ✅ Done for the three named pieces. `internal/backendmath` (97.7% coverage) holds `ValidateTargets`, `ComboMasks`, and `PlanCollapse`/`Renormalize` (with the 1e-24 zero-probability floor from 2.5); both backends call all three symmetrically (`state/state.go:158,235,300` ↔ `internal/sparsestate/state.go:171,242,345`). Benchmarked before/after per the commit bodies — dense measurement ~7% faster, sparse gate paths +7–14%, dense gate application unchanged (interleaved against a prior-commit worktree). **Residual**: physics beyond the item's list is still duplicated — `SetAmplitude(s)` validation/rollback, `isNormalized`/`probabilitySum`, and the 2×2/combo mixing loops remain near-verbatim in both backends, guarded by the dense-vs-sparse equivalence fuzz (3.4). Commits e484434, 80a1a10.

#### 3.3 Grow the gate set toward the algorithm ambitions
- **What**: After 2.4, add parameterized rotations (Rx/Ry/Rz/Phase), Toffoli, and controlled-U construction (via existing `TensorProduct`/`ComposeMatrices`); then express Grover's diffusion as gates in an example, keeping the O(2ⁿ) path as the fast implementation.
- **Risk**: Medium — new API surface, needs matrix-identity tests.
- **Impact**: Closes the pedagogical gap where the flagship algorithms bypass the gate model the project exists to teach; `DecomposeSwap` and `Registry` gain reasons to exist.
- **Result (2026-08-25, verified at v0.3.0 / 846268f)**: ✅ Done, one documented deviation. `NewRx/NewRy/NewRz/NewPhase/NewToffoli` in `gates/gates.go` plus `NewControlled` (`gates/controlled.go:23`). **Deviation**: controlled-U builds `diag(I, U)` by direct block copy rather than via `TensorProduct`/`ComposeMatrices` — the doc comment explains why (no matrix-addition helper exists; composing would add unreachable error paths). Matrix-identity tests are thorough: Toffoli truth table, Rx/Ry/Rz(π) = Pauli up to asserted global phase, Phase(π/2ᵏ) = Z/S/T, unitarity sweep, Controlled(X)=CNOT, Controlled(CNOT)=Toffoli. Grover's diffusion expressed as gates in `internal/examples/algorithm.go` (H⊗n·X⊗n·MCZ·X⊗n·H⊗n via stacked `NewControlled`, plus `Rz(2π)` global-phase restore), with the O(2ⁿ) closed form explicitly kept as the fast path and the demo diffing both amplitude sets; wired into the CLI and covered by `TestCLIAlgorithmDemoRuns`. `Registry` gained Toffoli (parameterized gates deliberately excluded, rationale at `gates/registry.go:70-74`). Commit cb06d2c.

#### 3.4 Fuzz the validation boundary
- **What**: Go-native fuzz tests for `GateQubitCount` (fix the 1×1-matrix acceptance while at it), `SetAmplitude(s)` normalization boundaries, and dense-vs-sparse gate-application equivalence with random 1–2 qubit unitaries.
- **Risk**: Low — additive.
- **Impact**: The dense/sparse equivalence fuzz would have caught the CNOT-name-dispatch hazard mechanically; guards 3.2's refactor.
- **Result (2026-08-25, verified at v0.3.0 / 846268f)**: ✅ Done. Six fuzz targets: `FuzzGateQubitCount` (1×1 case seeded), `FuzzSetAmplitude(s)` in both backends with seeds straddling the 1e-10 tolerance edge, and `FuzzDenseSparseGateEquivalence` over random 1–2 qubit unitaries (angle-built, unitarity-guarded, both target orders, canonical-CNOT fast-path seed). The 1×1 acceptance was fixed at the source: `GateQubitCount` now returns `InvalidGateMatrixError` ("matrix must be at least 2x2", `quantum/gateutil.go:57-63`) with unit + fuzz coverage. Bonus landed in the same commit: non-finite amplitude validation (`quantum.IsFiniteAmplitude`, `NonFiniteAmplitudeError`) enforced in both backends' setters. **Residual**: CI runs the fuzz targets over their seed corpora only (via `go test ./...`) — no extended `-fuzz` step or committed corpora; long fuzzing stays manual. Commit 93fbfcc.

### Priority 4: Remaining Subsystem Concerns (queued 2026-08-25)

> Sourced from a post-P3 sweep of the Detailed Subsystem Analysis: concerns raised above that no P1–P3 recommendation covered. Verified still present on `main` at 15b6c9d. Not queued: `GateQubitCount` per-call validation — the `QubitCounter` fast path (`quantum/gateutil.go:21-26`, implemented by every built-in gate) short-circuits before the matrix walk, resolving the cost concern in practice.

#### 4.1 Make `quantum all` safe when piped
- **What**: `cmd/quantum/main.go:112-132` — six unguarded `fmt.Scanln()` calls with discarded errors hang the `all` demo under pipes/CI. Add a non-interactive path (flag such as `-no-pause`, or skip pauses when stdin is not a TTY) and stop discarding the `Scanln` error.
- **Risk**: Low — CLI-only; extend the ADR-0006 contract tests.
- **Impact**: Removes the most user-visible defect left in the module; `quantum all` becomes scriptable.
- **Result (2026-08-26)**: ✅ Done. Both escape hatches landed: a `-no-pause` flag and TTY auto-detect (`os.Stdin.Stat()` + `os.ModeCharDevice`, stdlib only) — piped/CI `quantum all` now runs straight through and exits 0. The six inlined pauses collapsed into a `pausePrompter` helper sharing one `bufio.Reader`; read errors are no longer discarded (a genuine mid-run stdin failure disables later pauses instead of spamming prompts, typed content is discarded harmlessly — the review caught that zero-arg `Scanln` would have turned a stray "y"+Enter into silent pause suppression, fixed via `ReadString('\n')`). ADR-0006 contract tests extended: piped no-hang, `-no-pause` prompt suppression, and direct prompter unit tests; interactive TTY output pty-verified byte-identical. Commits ebfc2fb, 610f392.

#### 4.2 Untrack the committed pprof binaries
- **What**: `CHANGES/profiles/*.pprof` are tracked, and README:132-136 documents a workflow that regenerates them into the tracked path. `git rm --cached`, gitignore the pattern, point the README workflow at an ignored path.
- **Risk**: Low — no code change.
- **Impact**: Ends binary churn in version control; profiling docs stop dirtying the tree.

#### 4.3 Decide the fate of `MaxGateQubits`
- **What**: Half the `BackendCapabilities` contract is still dead — zero non-test callers (`quantum/interfaces.go:114`). Either consult it in `circuit.checkCapabilities` alongside `SupportsGateQubits`, or deprecate it per the deprecation policy.
- **Risk**: Low to wire in; removing is a breaking interface change.
- **Impact**: "Built means reachable" for the capabilities contract; same class of debt P1 item 1.3 cleared elsewhere.

#### 4.4 Document the comparability assumption in `validateIndependentStates`
- **What**: `circuit/parallel.go:16` keys a map by `quantum.QuantumState` interface values — an uncomparable implementation panics at runtime, and the doc comment is silent about it. Minimum: state the requirement in the doc comment; optional: guard with `reflect.TypeOf(s).Comparable()` and return a typed error.
- **Risk**: Low — doc-only, or a cold-path check.
- **Impact**: Turns an undocumented panic into a stated contract.

#### 4.5 Bloch vectors for multi-qubit states
- **What**: `visualization.BlochVectorFromQubit` remains single-qubit; `density.ReducedBlochVector` exists but is `internal/` with no bridge from a state vector. Add a state-vector → reduced-density path (e.g. `density.FromState` + a visualization entry point taking `quantum.QuantumState` and a target index).
- **Risk**: Medium — new public API; needs entangled-state tests (reduced vector of a Bell pair is the zero vector).
- **Impact**: Bloch export works for the states users actually build; gives the density package its second consumer.

#### 4.6 Decide density's relationship to `QuantumState`
- **What**: `internal/density` still implements a fixed-arity `ApplySingleQubitGate` rather than `quantum.QuantumState` (`ApplyGate(gate, targets...)`, `Measure`, `Clone`, ...). Either implement the interface so it can join the circuit abstraction, or record the deliberate scope (noise-demo backend only) in an ADR/doc comment.
- **Risk**: Medium-High to implement (measurement on density matrices is real design work); Low to document.
- **Impact**: Closes the last "built but not integrated" question; 4.5 gets easier if implemented.

#### 4.7 Extract the shared example banner code
- **What**: `internal/examples/utils.go` holds one function while the `=====` banner/section blocks are hand-inlined in `bell.go:257-265`, `hadamard.go:201-219`, `tgate.go:234-243`, and `cmd/quantum/main.go:105-108,136-138`. Extract a shared banner/section helper.
- **Risk**: Low — demo output only; CLI contract tests pin the strings.
- **Impact**: Cosmetic debt cleared; new demos stop copy-pasting.

#### 4.8 Small-sweep leftovers
- **What**: (a) test `visualization.DefaultStateViewOptions` (the package's only 0% function); (b) add the versioning note to `docs/deprecation-policy-v2.md` that `MIGRATION-v2.md:5-10` already carries; (c) trim CI's triple suite execution (plain + race + coverage per matrix leg — race and coverage suffice) and consider pre-commit hooks.
- **Risk**: Low — each independent and mechanical.
- **Impact**: Coverage floor honesty, doc consistency, ~⅓ less CI compute.

---

## Summary

For a solo educational project, this codebase is unusually disciplined: zero dependencies, a typed error taxonomy, ADR-documented decisions, race-aware concurrency tests, and a genuinely closed review-remediate-document loop. Its problems are the problems of momentum, not carelessness — components built and tested but never wired in (density backend, registry, decompositions), an error return dropped in Grover's hot loop, one stringly-typed dispatch shortcut in the sparse backend, and release hygiene (LICENSE, tags, CI teeth) lagging the code's actual maturity. The P1 list is a weekend of low-risk work that removes the only silent-wrong-answer path and makes the project legally and practically consumable; P2 closes the determinism and coverage gaps; P3 finishes the backend abstraction the interfaces already promise.

**Overall Rating: B+ (Good — solid engineering with a working quality loop; the deduction from A- is for the swallowed Grover errors, the name-based CNOT dispatch, the untested qubit package, and the accumulation of built-but-unwired components)**
