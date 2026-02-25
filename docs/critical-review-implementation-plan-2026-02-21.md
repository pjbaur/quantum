# Implementation Plan for `critical-review-2026-02-21.md`

This plan implements all findings and recommendations in `docs/critical-review-2026-02-21.md`, with explicit sequencing and parallelizable workstreams.

## Goals

- [x] Resolve all highest-priority correctness and API risks.
- [x] Resolve all medium-priority maintainability and UX gaps.
- [x] Close documented test coverage gaps.
- [x] Keep behavior changes deliberate, documented, and reviewable.

## Worktree Topology (Parallel Execution)

Use dedicated worktrees so teams can ship independent PRs in parallel with low merge contention.

- [x] `ws1-api-core`: state-vector-first API, gate interface cleanup, constructor consistency, shared validation helpers.
- [x] `ws2-concurrency`: `ExecuteAllParallel` safety guarantees, misuse detection, concurrency tests.
- [x] `ws3-sparse-backend`: sparse backend capability strategy and implementation/contract enforcement.
- [x] `ws4-algorithms`: Grover + Deutsch-Jozsa scalability refactor and negative-path test coverage.
- [x] `ws5-cli-docs-tests`: CLI/help drift fixes, dead parameter resolution, visualization helper tests, docs alignment.
- [ ] `ws6-integration`: final integration pass, conflict resolution, cross-package verification, release notes.

## Phase 0 (Do First): “What To Do Differently” Decisions and Guardrails

This phase sets project-level direction before code-heavy changes. No feature work starts before these checkboxes are done.

### 0.1 State-vector-first API direction (`ws1-api-core`) ✅ COMPLETE

- [x] Write an API decision note defining matrix+targets `ApplyGate` as the primary execution model.
  - See: `docs/adr/0001-state-vector-first-api.md`
- [x] Decide deprecation strategy for `Apply(q Qubit)` paths (soft deprecate vs remove in major version).
  - Decision: Hard removal in v2.0
  - See: `docs/deprecation-policy-v2.md`
- [x] Define compatibility policy for existing callers that still use qubit-level APIs.
  - No external consumers; hard removal acceptable
  - See: `docs/compatibility-policy.md`

### 0.2 Enforceable concurrency contract (`ws2-concurrency`) ✅ COMPLETE

- [x] Choose contract: fail fast on shared pointers, or clone internally before parallel execution.
  - Decision: Fail-fast with shared-state detection
  - See: `docs/adr/0002-concurrency-contract.md`
- [x] Define deterministic error behavior for unsafe shared-state submissions.
  - `SharedStateError` with duplicate indices
  - See: `docs/adr/0002-concurrency-contract.md`
- [x] Document performance tradeoff of the chosen contract in package docs.
  - O(n) pointer validation overhead documented in ADR

### 0.3 Explicit backend capability model (`ws3-sparse-backend`) ✅ COMPLETE

- [x] Choose strategy: full generic k-qubit sparse gate support, or explicit capability limits.
  - Decision: Explicit capability limits (1-2 qubit gates only)
  - See: `docs/adr/0003-sparse-backend-capability.md`
- [x] If keeping limits, define capability-check API so unsupported operations fail early/clearly.
  - `BackendCapabilities` interface with `SupportsGateQubits()` and `MaxGateQubits()`
  - `UnsupportedOperationError` for 3+ qubit gates
  - See: `docs/adr/0003-sparse-backend-capability.md`
- [x] Define compatibility behavior between `circuit` execution and backend capability checks.
  - Circuit compilation checks backend capabilities before execution

### 0.4 Diagnostics and constructor consistency policy (`ws1-api-core` + `ws3-sparse-backend`) ✅ COMPLETE

- [x] Define normalization error contract to report attempted (pre-rollback) sum.
  - `NormalizationError` extended with `AttemptedSum` and `CurrentSum` fields
  - See: `docs/adr/0004-diagnostics-constructor-consistency.md`
- [x] Standardize invalid qubit-count constructor behavior across `circuit`, dense `state`, and sparse `state`.
  - All constructors return error for `numQubits <= 0`
  - New `InvalidQubitCountError` type
  - See: `docs/adr/0004-diagnostics-constructor-consistency.md`
- [x] Decide whether coercion-to-1 is removed or gated for backward compatibility.
  - Decision: Removed entirely (no external consumers)
  - See: `docs/adr/0004-diagnostics-constructor-consistency.md`

### 0.5 UX drift prevention policy (`ws5-cli-docs-tests`) ✅ COMPLETE

- [x] Define CLI contract source of truth (flags, demos, optional params, examples).
  - Source of truth: `cmd/quantum/main.go` - `usage()` function and `runDemos()` switch
  - See: `docs/adr/0006-cli-contract-and-drift-prevention.md`
- [x] Decide whether the currently unused optional CLI parameter is removed or implemented.
  - Decision: Removed (`-param` flag and positional param were dead code)
  - See: `docs/adr/0006-cli-contract-and-drift-prevention.md`
- [x] Add a docs synchronization checklist to PR template or release checklist.
  - Created: `.github/PULL_REQUEST_TEMPLATE.md` with CLI/docs sync section

### 0.6 Regression prevention test strategy (`ws2/ws4/ws5`) ✅ COMPLETE

- [x] Define minimum new test matrix: negative paths, misuse paths, helper-level output tests.
  - See: `docs/adr/0005-regression-prevention-test-strategy.md`
- [x] Add race-focused test command expectations for concurrency-sensitive packages.
  - Race test patterns and CI expectations documented in ADR-0005

## Phase 1: Highest-Priority Correctness and API Safety

### 1.1 Gate API and legacy qubit path cleanup (`ws1-api-core`) ✅ COMPLETE

- [x] Refactor `quantum.Gate` usage so multi-qubit execution is centered on matrix+targets paths.
- [x] Isolate/deprecate misleading single-qubit `Apply(q Qubit)` gate methods.
- [x] Remove or mark clearly non-physical helper methods that bypass entanglement-correct simulation.
- [x] Update inline package docs to warn against legacy single-qubit simulation paths for circuit execution.
- [x] Add migration notes for downstream callers.

### 1.2 Enforced parallel execution safety (`ws2-concurrency`) ✅ COMPLETE

- [x] Update `ExecuteAllParallel` to enforce independence contract at runtime.
- [x] Implement duplicate state detection (pointer identity or equivalent robust keying).
- [x] Return explicit, actionable error messages on shared-state misuse.
- [x] Add fast-path behavior for valid independent states with minimal overhead.

### 1.3 Sparse backend compatibility guarantees (`ws3-sparse-backend`) ✅ COMPLETE

- [x] Implement chosen sparse strategy from Phase 0:
- [x] If generic support: add k-qubit gate application path and validation.
- [x] If explicit limits: add capability interface/checks and fail early during planning/execution.
- [x] Ensure circuit execution path does not silently proceed into unsupported sparse operations.
- [x] Add backend capability documentation and examples.

### 1.4 Correct normalization diagnostics and rollback reporting (`ws1-api-core` + `ws3-sparse-backend`) ✅ COMPLETE

- [x] Capture attempted normalization sum before rollback in dense `SetAmplitude`.
- [x] Capture attempted normalization sum before rollback in sparse `SetAmplitude`.
- [x] Ensure error payloads and messages are consistent across dense/sparse implementations.
- [x] Add precise tests asserting attempted-vs-post-rollback sums.

### 1.5 Constructor invalid-input consistency (`ws1-api-core` + `ws3-sparse-backend`) ✅ COMPLETE

- [x] Align `state.New`, sparse `New`, and `circuit.New` behavior for invalid qubit counts.
- [x] Add consistent error type/message semantics across constructors.
- [x] Update call sites and tests to match the standardized behavior.

## Phase 2: Medium-Priority Maintainability and UX

### 2.1 CLI help/behavior consistency (`ws5-cli-docs-tests`) ✅ COMPLETE

- [x] Add missing `visual` demo option in `-demo` help text.
- [x] Resolve optional parameter drift by either implementing behavior or removing dead parsing.
- [x] Add CLI tests or golden help-output assertions to prevent future drift.
- [x] Update user-facing docs/examples to match implemented CLI behavior.

### 2.2 Algorithm scalability refactor (`ws4-algorithms`) ✅ COMPLETE

- [x] Replace full dense matrix construction in Grover with direct state-vector transformations where possible.
- [x] Replace full dense matrix construction in Deutsch-Jozsa with scalable operator application strategy.
- [x] Preserve algorithm correctness with deterministic output/state assertions.
- [x] Add focused benchmarks to compare pre/post memory/time for representative qubit counts.

### 2.3 Shared gate-matrix validation (`ws1-api-core`) ✅ COMPLETE

- [x] Extract `gateQubitCount` into a single shared location/package.
- [x] Migrate `circuit`, dense `state`, and sparse backend to the shared helper.
- [x] Remove duplicated implementations and unify error strings.
- [x] Add table-driven validation tests that cover all current call sites.

## Phase 3: Test Coverage Expansion (Documented Gaps)

### 3.1 Algorithm negative-path tests (`ws4-algorithms`) ✅ COMPLETE

- [x] Add table-driven invalid-input tests for `DeutschJozsa` error paths.
- [x] Add table-driven invalid-input tests for `Grover` error paths.
- [x] Assert error type/message specificity, not only generic failure.

### 3.2 Parallel shared-state hazard tests (`ws2-concurrency`) ✅ COMPLETE

- [x] Add tests that intentionally pass shared state pointers to `ExecuteAllParallel`.
- [x] Assert fail-fast behavior (or clone semantics) based on Phase 0 decision.
- [x] Add race-focused tests for concurrent independent-state executions.

### 3.3 Visualization helper direct tests (`ws5-cli-docs-tests`) ✅ COMPLETE

- [x] Add direct unit tests for `FormatBlochVector`.
- [x] Add direct unit tests for `BlochCSV`.
- [x] Include formatting edge cases (rounding, sign, delimiter, invalid inputs if applicable).

## Phase 4: Integration, Hardening, and Release Readiness (`ws6-integration`)

### 4.1 Merge and conflict control

- [ ] Merge `ws1` first (API/core contracts), then `ws2` + `ws3`, then `ws4` + `ws5`.
- [ ] Resolve cross-package conflicts in `circuit`, `state`, and `quantum` interfaces.
- [ ] Re-run full test suite after each merge step.

### 4.2 Verification checklist

- [ ] `gofmt` on all touched files.
- [ ] `go test ./...`
- [ ] `go vet ./...`
- [ ] `go test -race ./...` (or targeted race suites if environment constraints persist).
- [ ] Ensure no documented high/medium findings remain unaddressed.

### 4.3 Documentation and migration output ✅ COMPLETE

- [x] Update `docs/critical-review-2026-02-21.md` with completion links/PR references.
  - Added commit hash links to all resolution entries
- [x] Add migration notes for API/deprecation/constructor behavior changes.
  - Created `docs/MIGRATION-v2.md` with comprehensive migration guide
- [x] Update CLI and backend capability docs.
  - Updated `docs/sparse-state.md` with BackendCapabilities interface documentation
- [x] Publish final “resolved findings” summary in `docs/`.
  - Created `docs/resolved-findings-summary.md` with complete findings-to-resolution mapping

## Worktree-by-Worktree Deliverables

### `ws1-api-core`

- [x] API decision note + migration strategy.
- [x] Gate API cleanup/deprecation implementation.
- [x] Constructor consistency implementation.
- [x] Shared gate validation helper.
- [x] Normalization diagnostics fix (dense side).

### `ws2-concurrency`

- [x] Enforced `ExecuteAllParallel` contract.
- [x] Shared-state misuse detection and explicit errors.
- [x] Hazard/race regression tests.
- [x] Package docs for concurrency contract.
- [x] Race-focused tests for concurrent independent-state executions (Phase 3.2).

### `ws3-sparse-backend`

- [x] Sparse capability strategy implementation (generic support or explicit contract).
- [x] Early failure hooks for unsupported gates.
- [x] Normalization diagnostics fix (sparse side).
- [x] Sparse backend docs/tests aligned with selected capability model.

### `ws4-algorithms`

- [x] Grover scalability refactor.
- [x] Deutsch-Jozsa scalability refactor.
- [x] Negative-path coverage for both algorithms.
- [x] Benchmark evidence and regression checks.

### `ws5-cli-docs-tests`

- [x] CLI help and behavior alignment.
- [x] Dead optional parameter resolution.
- [x] Direct visualization helper tests.
- [x] Documentation and examples sync pass.

### `ws6-integration`

- [ ] Merge-order execution and integration fixes.
- [ ] Full verification execution and evidence capture.
- [x] Final completion report mapping to every review finding.

## Dependency Map (Execution Order)

- [x] Phase 0 decisions complete before any breaking API/code-path changes.
- [x] `ws1-api-core` must land before `ws4-algorithms` finalization if shared validation API changes.
- [x] `ws2-concurrency` and `ws3-sparse-backend` can proceed in parallel after Phase 0 decisions.
- [x] `ws5-cli-docs-tests` can run in parallel with `ws2/ws3/ws4`.
- [x] `ws6-integration` starts only after all feature worktrees are merged or ready to merge.

## Traceability Checklist: Review Item Coverage

### Highest-Priority Findings

- [x] `quantum.Gate` misleading for multi-qubit usage.
- [x] Fragile `ExecuteAllParallel` caller contract.
- [x] Sparse backend not drop-in for general circuits.
- [x] Normalization diagnostics wrong after rollback.

### Medium-Priority Findings

- [x] CLI help/behavior inconsistency (`visual` + optional parameter).
- [x] Algorithm scalability issues due to dense matrix construction.
- [x] Duplicated gate matrix validation logic.
- [x] Constructor inconsistency on invalid qubit counts.

### Test Coverage Gaps

- [x] Missing negative-path tests for `DeutschJozsa` and `Grover`.
- [x] Missing shared-state misuse tests for parallel execution.
- [x] Missing race-focused tests for concurrent independent-state executions.
- [x] Missing direct tests for `FormatBlochVector` and `BlochCSV`.

## Suggested PR Sequence

- [x] PR1: Phase 0 decision docs + non-breaking guardrails.
- [x] PR2: `ws1-api-core` correctness/API contract updates.
- [x] PR3: `ws2-concurrency` enforcement + tests.
- [x] PR4: `ws3-sparse-backend` capability implementation + tests.
- [x] PR5: `ws4-algorithms` scalability + negative-path tests.
- [x] PR6: `ws5-cli-docs-tests` UX/docs/test fixes.
- [ ] PR7: `ws6-integration` final merge, verification, and closure report.
