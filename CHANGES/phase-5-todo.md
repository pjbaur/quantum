# Phase 5 TODO List (Visualization and Advanced Features)

Organized for parallel development across visualization, density matrices/noise, and parallel simulation.

## Workstream 1: Visualization tools

- [x] Define text-based state views for multi-qubit states (basis index, amplitude, probability).
- [x] Add a single-qubit Bloch sphere visualization output (ASCII or data export for plotting).
- [x] Provide small examples/demos for visualization outputs.
- [x] Document visualization usage in README or docs.

## Workstream 2: Density matrices and noise

- [x] Introduce a density matrix representation behind internal types or build tags.
- [x] Implement density matrix evolution for single-qubit gates.
- [x] Add common noise models (e.g., depolarizing, dephasing, amplitude damping).
- [x] Validate density matrix behavior with tests (trace, positivity, expected distributions).

## Workstream 3: Parallel simulation

- [x] Identify independent circuit execution paths that can be parallelized.
- [x] Add a parallel execution strategy for independent circuits.
- [x] Benchmark parallel execution vs. serial execution on representative workloads.
- [x] Document any concurrency limits or configuration knobs.

## Suggestions from Workstreams:

### Visualization
1. Run the new demo: go run ./cmd/quantum visual
2. If you want sample output in docs, I can add a short snippet to README.md

### Parallel batch circuit execution
Next steps if you want to validate perf:
1. go test ./circuit -bench BenchmarkCircuitExecuteBatch -benchmem -run '^$'
2. go test ./circuit -bench BenchmarkCircuitExecuteBatchParallel -benchmem -run '^$'
