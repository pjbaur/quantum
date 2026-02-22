# Architecture Decision Records

This directory contains Architecture Decision Records (ADRs) for the quantum computing simulator project.

## What is an ADR?

An ADR is a document that captures an important architectural decision made along with its context and consequences. ADRs help us:

- Document the "why" behind architectural choices
- Provide context for future contributors
- Avoid re-litigating settled decisions
- Create a decision history for the project

## ADR Format

Each ADR follows this structure:

```markdown
# ADR-NNNN: Brief Title

## Status
[Proposed | Accepted | Deprecated | Superseded]

## Context
What is the issue we're addressing? What constraints exist?

## Decision
What is the change we're proposing/have made?

## Consequences
What becomes easier or harder as a result?

## Decision Log
- YYYY-MM-DD: Initial decision
- YYYY-MM-DD: Any updates or amendments
```

## ADR Index

| Number | Title | Status | Date |
|--------|-------|--------|------|
| 0001 | State-Vector-First API Direction | Accepted | 2026-02-21 |

## Process

1. **Proposing**: Create a new ADR with "Proposed" status
2. **Review**: Discuss the proposal, gather feedback
3. **Accepting**: Update status to "Accepted" once approved
4. **Implementing**: Execute the decision
5. **Amending**: Update the decision log if circumstances change
6. **Deprecating**: Mark as "Deprecated" or "Superseded" if replaced

## Naming Convention

- Files are named `NNNN-short-title.md` (e.g., `0001-state-vector-first-api.md`)
- Numbers are zero-padded to 4 digits
- Titles use kebab-case
