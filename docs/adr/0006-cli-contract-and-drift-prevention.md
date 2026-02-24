# ADR-0006: CLI Contract and Drift Prevention

## Status

Accepted

## Context

The CLI (`cmd/quantum/main.go`) had several UX and documentation drift issues identified in the critical review:

### Problems Found

1. **Missing `visual` in help text**: The `visual` demo was implemented (`main.go:48`) but the `-demo` flag description (`main.go:84`) omitted it, causing confusion for users.

2. **Dead parameter parsing**: An optional `-param` flag and positional parameter were parsed (`main.go:85`, `main.go:108-121`) but never used (`main.go:138-139`: `_ = param`). This created false expectations that parameters did something.

3. **No docs synchronization process**: Without a checklist or PR template, CLI help and user-facing documentation could drift apart from actual implementation.

### CLI Contract Source of Truth

The CLI contract is defined in `cmd/quantum/main.go`:

| Element | Source of Truth | Description |
|---------|-----------------|-------------|
| Flags | `flag.*String()` calls | Command-line flags with defaults and descriptions |
| Demos | `runDemos()` switch cases | Available demo types and their implementations |
| Usage | `usage()` function | Help text printed on `-h` or error |
| Examples | `usage()` function | Example commands in help text |

**Current Contract**:

```
Usage: quantum [options] <demo>

Options:
  -demo string
        Demo to run (hadamard, tgate, bell, algorithm, visual, all)

Demos:
  - hadamard  - Hadamard gate demonstrations
  - tgate     - T-gate demonstrations
  - bell      - Bell state demonstrations
  - algorithm - Algorithm demonstrations (Deutsch-Jozsa, Grover)
  - visual    - Visualization demonstrations
  - all       - Run all demonstrations
```

## Decision

### 1. Fix Help Text Inconsistency

The `-demo` flag description now includes all implemented demos: `hadamard, tgate, bell, algorithm, visual, all`.

### 2. Remove Dead Parameter

The unused `-param` flag and positional parameter handling are removed. Rationale:

- No demo currently uses the parameter
- Parsing unused parameters creates user confusion
- The parameter can be re-added when a concrete use case emerges

### 3. Establish Docs Synchronization Checklist

A PR template is created at `.github/PULL_REQUEST_TEMPLATE.md` with a CLI/docs synchronization section to prevent future drift.

### 4. Single Source of Truth

The `usage()` function in `cmd/quantum/main.go` is the authoritative source for:
- Available demo types
- Flag descriptions
- Usage examples

Any changes to CLI behavior must update `usage()` first, then propagate to documentation.

## Consequences

### Positive

1. **Consistency**: Help text matches actual behavior
2. **Simplicity**: No confusing dead parameters
3. **Prevention**: PR checklist catches future drift
4. **Clarity**: Single source of truth for CLI contract

### Neutral

1. Future parameters can be added when needed
2. The `usage()` function must be kept in sync with implementation

## Decision Log

- 2026-02-21: Initial decision accepted
- 2026-02-21: Fixed help text to include `visual`
- 2026-02-21: Removed unused `-param` flag
- 2026-02-21: Created PR template with docs sync checklist
