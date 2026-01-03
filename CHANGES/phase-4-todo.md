# Phase 4 TODO List (Performance and Extensibility)

Organized for parallel development across profiling, performance, extensibility, and algorithms workstreams.

## Workstream 1: Profiling and allocation reduction

- [x] Establish baseline benchmarks for state operations (apply gate, measure, clone).
  Baseline (`go test ./state -bench Benchmark -benchmem -run '^$'`):
  ApplySingleQubitGate 35158 ns/op 65760 B/op 7 allocs/op; ApplyMultiQubitGate 80211 ns/op 66368 B/op 13 allocs/op; Measure 82836 ns/op 65536 B/op 1 allocs/op; Clone 12477 ns/op 65536 B/op 1 allocs/op.
- [x] Capture CPU and memory profiles for representative circuits.
  Generated `CHANGES/profiles/ws1-circuit-cpu.pprof` and `CHANGES/profiles/ws1-circuit-mem.pprof` from `go test ./circuit -bench BenchmarkCircuitExecute -benchmem -run '^$' -cpuprofile ... -memprofile ...` (BenchmarkCircuitExecute 220851 ns/op 41637 B/op 156 allocs/op).
- [x] Identify hot paths and unnecessary allocations in state evolution.
  Found per-apply allocations for amplitude buffers, multi-qubit scratch slices, and target validation maps in state operations.
- [x] Implement targeted allocation reductions without changing public APIs.
  Reused state scratch buffers for apply/measure, reused multi-qubit scratch slices, and switched target uniqueness checks to a bitmask fast-path.
- [x] Re-run benchmarks and record deltas in `CHANGES/phase-4-todo.md`.
  After changes (`go test ./state -bench Benchmark -benchmem -run '^$'`):
  ApplySingleQubitGate 24310 ns/op 225 B/op 6 allocs/op; ApplyMultiQubitGate 67674 ns/op 707 B/op 10 allocs/op; Measure 74846 ns/op 0 B/op 0 allocs/op; Clone 12431 ns/op 65536 B/op 1 allocs/op.

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

- [ ] Introduce `algorithm` package with minimal public surface.
- [ ] Implement Deutsch-Jozsa using existing gates and circuit/state APIs.
- [ ] Implement Grover's algorithm for small qubit counts.
- [ ] Add tests validating algorithm output distributions.
- [ ] Add short examples that compile and run in the current module layout.
