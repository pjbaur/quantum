# Compatibility Policy

## Project Context

This quantum computing simulator is a **personal project** with **no external consumers**. The sole user and maintainer is the project owner.

## v2.0 Compatibility Stance

Given the project context:

1. **No backward compatibility required** for the legacy single-qubit API
2. **Hard removal** of deprecated methods is acceptable
3. **Migration** involves only internal code (tests, examples, algorithms)

## Why Hard Removal is Acceptable

### No External Consumers

All affected code is internal to this repository:

| Category | Files | Impact |
|----------|-------|--------|
| Tests | `quantum/quantum_test.go` | Update test code |
| Examples | `internal/examples/*` | Rewrite examples |
| Implementation | `gates/gates.go` | Remove deprecated methods |

No public API consumers exist outside this repository.

### Benefits of Clean Break

- Simpler codebase (no deprecated paths to maintain)
- Clear migration story (not "which API should I use?")
- Faster development (no compatibility shims)

## Migration Responsibilities

Since all code is internal, migration is straightforward:

1. Update interface definition
2. Remove deprecated implementations
3. Update all call sites
4. Update tests
5. Update examples

## Future Compatibility Considerations

If this project ever gains external users:

1. Announce breaking changes in advance
2. Provide a deprecation period (minimum 1 major version)
3. Document migration paths clearly
4. Consider semantic versioning more strictly

For now, these considerations do not apply.

## References

- `docs/adr/0001-state-vector-first-api.md` - API direction decision
- `docs/deprecation-policy-v2.md` - Specific method removals
- `docs/critical-review-implementation-plan-2026-02-21.md` - Overall implementation plan
