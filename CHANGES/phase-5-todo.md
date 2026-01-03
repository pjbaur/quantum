# Phase 5 TODO List (Visualization and Advanced Features)

Organized for parallel development across visualization, density matrices/noise, and parallel simulation.

## Workstream 1: Visualization tools

- [ ] Define text-based state views for multi-qubit states (basis index, amplitude, probability).
- [ ] Add a single-qubit Bloch sphere visualization output (ASCII or data export for plotting).
- [ ] Provide small examples/demos for visualization outputs.
- [ ] Document visualization usage in README or docs.

## Workstream 2: Density matrices and noise

- [ ] Introduce a density matrix representation behind internal types or build tags.
- [ ] Implement density matrix evolution for single-qubit gates.
- [ ] Add common noise models (e.g., depolarizing, dephasing, amplitude damping).
- [ ] Validate density matrix behavior with tests (trace, positivity, expected distributions).

## Workstream 3: Parallel simulation

- [ ] Identify independent circuit execution paths that can be parallelized.
- [ ] Add a parallel execution strategy for independent circuits.
- [ ] Benchmark parallel execution vs. serial execution on representative workloads.
- [ ] Document any concurrency limits or configuration knobs.

## Suggestions from Workstreams:

### Visualization
1. Run the new demo: go run ./cmd/quantum visual
2. If you want sample output in docs, I can add a short snippet to README.md

### Parallel batch circuit execution
Next steps if you want to validate perf:
1. go test ./circuit -bench BenchmarkCircuitExecuteBatch -benchmem -run '^$'
2. go test ./circuit -bench BenchmarkCircuitExecuteBatchParallel -benchmem -run '^$'