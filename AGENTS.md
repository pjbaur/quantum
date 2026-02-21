# AGENTS.md

## Role
You are an automated coding agent working in this Go repository.
Make small, correct, reviewable changes that preserve existing behavior
unless explicitly instructed otherwise.

Prefer correctness, clarity, and simplicity over cleverness.

---

## Instruction Precedence
When instructions conflict, apply this order:
1. System or platform instructions
2. Direct user request in the current task
3. This `AGENTS.md`

If requirements are unclear and could change public API or behavior,
ask for clarification before implementing.

---

## Language & Tooling
- Language: Go
- Minimum Go version: 1.21
- Dependency management: go modules
- Formatting: `gofmt` (mandatory on edited Go files)
- Linting: `go vet` (at least on affected packages)

Do not introduce new dependencies without explicit approval.
If a dependency is necessary, explain why it is needed and keep module
changes minimal.

---

## Coding Conventions
- Follow Effective Go and standard library idioms
- Prefer explicit error handling over panic
- Wrap returned errors with context when appropriate (for example,
  `fmt.Errorf("...: %w", err)`)
- Avoid global state unless already present in the codebase
- Keep functions small and single-purpose
- Favor composition over inheritance-style patterns

---

## Tests & Verification
- Run `go test` on affected packages at minimum
- Prefer `go test ./...` when the change is broad or risk is unclear
- Run `go vet` on affected packages (or `./...` when practical)
- New behavior requires tests unless explicitly told otherwise
- Prefer table-driven tests
- Do not weaken assertions to make tests pass

If tests fail:
- Fix production code by default
- Update tests only when they are incorrect, outdated, or mismatched with
  approved behavior
- If baseline failures already exist, do not hide them; report them
  separately and avoid making them worse

---

## Scope & Safety Rules
- Do not refactor unrelated code
- Do not rename exported symbols without instruction
- Do not change public APIs unless explicitly required by the task
- Avoid broad formatting or mechanical changes unless requested

Make the smallest change that solves the problem.
If behavior must change, document the reason and impact.

---

## Output Expectations
- Changes should be logically grouped
- Code should be ready for human review
- Explain *why* a change is made, not just *what* changed

In the final response, include:
- Files changed
- Behavior impact
- Verification commands run and outcomes
- Assumptions, open questions, or residual risks

If something is unclear or underspecified, state assumptions explicitly.
