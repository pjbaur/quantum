# Phase 3 TODO List (Tooling and DX)

Organized for parallel development across CLI, CI, and documentation workstreams.

## Workstream 1: CLI flags and help

- [x] Replace custom CLI parsing with the standard `flag` package.
- [x] Define clear `--help` output with usage examples and option descriptions.
- [x] Verify CLI flags cover current usage paths and exit codes are consistent.

## Workstream 2: CI pipeline

- [x] Add a CI workflow that runs `go test ./...`.
- [x] Add a CI step for `go vet ./...`.
- [x] Add linting in CI (choose the existing or CI-enforced linter, if any).
- [x] Add coverage reporting in CI and set a baseline threshold.

## Workstream 3: README alignment

- [x] Update CLI usage examples to match the current command path.
- [x] Update example locations to match repository structure.
- [x] Document the current state of the measurement package (empty or TODO).
