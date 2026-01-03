# AGENTS.md

## Role of the Agent
You are an automated coding agent working in this Go repository.
Your job is to make small, correct, reviewable changes that preserve
existing behavior unless explicitly instructed otherwise.

Prefer correctness, clarity, and simplicity over cleverness.

---

## Language & Tooling
- Language: Go
- Minimum Go version: 1.21
- Dependency management: go modules
- Formatting: gofmt (mandatory)
- Linting: go vet (and CI-enforced linters, if present)

Do not introduce new dependencies without explicit instruction.

---

## Coding Conventions
- Follow Effective Go and standard library idioms
- Prefer explicit error handling over panic
- Avoid global state unless already present in the codebase
- Keep functions small and single-purpose
- Favor composition over inheritance-style patterns

---

## Tests & Verification
- Existing tests must pass
- New behavior requires tests unless explicitly told otherwise
- Prefer table-driven tests
- Do not weaken assertions to make tests pass

If tests fail, fix the code—not the tests—unless instructed.

---

## Scope & Safety Rules
- Do not refactor unrelated code
- Do not rename exported symbols without instruction
- Do not change public APIs unless required
- Avoid broad formatting or mechanical changes unless requested

Make the smallest change that solves the problem.

---

## Commit & Output Expectations
- Changes should be logically grouped
- Code should be ready for human review
- Explain *why* a change is made, not just *what* changed

If something is unclear or underspecified, state assumptions explicitly.
