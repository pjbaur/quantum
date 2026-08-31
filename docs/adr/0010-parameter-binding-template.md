# ADR-0010: Parameter Binding via Template Materialization

## Status

Accepted (2026-08-30).

## Context

Backlog item 4 deferred symbolic parameter binding until a real
variational driver existed. The VQE driver (H2, stage 7 of the
example-ideas ladder) is that driver. Two mechanisms were considered:

1. Symbolic gate types: gates carrying unevaluated angles, resolved at
   execution time.
2. Template materialization: a recipe object that produces concrete
   circuits from parameter values.

## Decision

Template materialization (`parameterized.Template` + `Bind`).

Symbolic gates would force every backend and every execution path to
resolve parameters before `Matrix()` can return numbers — touching the
metadata-only `quantum.Gate` contract that ADR-0001 just cleaned up, and
every backend for one consumer. Templates live above the core: `Bind`
emits ordinary circuits, backends never know parameters existed.

Tradeoff: each `Bind` re-allocates the gate list and re-bakes rotation
matrices. Benchmark `BenchmarkVQEIteration` vs `BenchmarkManualRebuild`
(VQEIteration 2407 ns/op vs ManualRebuild 2425 ns/op, 2 qubits,
2s runs) shows this is negligible next to the O(2^n) state execution
that dominates every variational iteration — the bound path is
statistically indistinguishable from rebuilding the gates by hand.

## Consequences

- `quantum.Gate`, `Registry`, and all backends unchanged.
- Parameter validation (missing/unknown/non-finite) fails at `Bind`,
  before any state is touched.
- Derived angles (`theta/2`) are out of scope: declare another parameter.
- QAOA (stage 8) can reuse the template unchanged.
