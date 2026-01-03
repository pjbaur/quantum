# Phase 4 TODO List (Performance and Extensibility)

Organized for parallel development across profiling, performance, extensibility, and algorithms workstreams.

## Workstream 1: Profiling and allocation reduction

- [ ] Establish baseline benchmarks for state operations (apply gate, measure, clone).
- [ ] Capture CPU and memory profiles for representative circuits.
- [ ] Identify hot paths and unnecessary allocations in state evolution.
- [ ] Implement targeted allocation reductions without changing public APIs.
- [ ] Re-run benchmarks and record deltas in `CHANGES/phase-4-todo.md`.

## Workstream 2: Sparse state representation exploration

- [ ] Define criteria for when sparse state is beneficial (qubit count, sparsity threshold).
- [ ] Prototype a sparse state representation behind internal types or build tags.
- [ ] Implement minimal gate application for sparse states (single-qubit + CNOT).
- [ ] Compare accuracy and performance against dense state benchmarks.
- [ ] Decide on adoption path without changing public interfaces.

## Workstream 3: Gate registry and composition utilities

- [ ] Define a minimal gate registry API (name -> gate) without new dependencies.
- [ ] Add helper utilities for composing gates into larger matrices.
- [ ] Add helper utilities for decomposing common multi-qubit gates into primitives.
- [ ] Add tests for registry lookup and composition/decomposition correctness.
- [ ] Keep existing gate types and exported symbols stable.

## Workstream 4: Algorithm package (Grover, Deutsch-Jozsa)

- [x] Introduce `algorithm` package with minimal public surface.
- [x] Implement Deutsch-Jozsa using existing gates and circuit/state APIs.
- [x] Implement Grover's algorithm for small qubit counts.
- [x] Add tests validating algorithm output distributions.
- [x] Add short examples that compile and run in the current module layout.
