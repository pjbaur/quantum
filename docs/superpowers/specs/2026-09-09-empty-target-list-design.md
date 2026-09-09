# Empty Target List in `AddParamGate` and `AddGate` — Design

Date: 2026-09-09
Status: Approved (backlog item text plus run-lead dispatch; no brainstorming gate).
Resolves: backlog item 19 (`.superpowers/backlog/enhancement-backlog-2026-08-27/item-19.md`)

## Context

`parameterized.Template` validates targets in one private helper,
`checkTargets`, which loops over the given targets and rejects any outside
`[0, numQubits-1]` as `*quantum.QubitsOutOfRangeError`. On an empty list
the loop body never runs, so the helper returns nil and both callers
proceed: `AddParamGate` records the parameter name and appends the step,
`AddGate` appends the step. Nothing else in the package looks at the
target count. The step surfaces only at `Bind`, where `circuit.AddGate`
rejects it with `errors.New("at least one target is required")`
(`circuit/circuit.go`, the `len(targets) == 0` check in `AddGate`), an
untyped error that names neither the gate nor the parameter. Observed
today (`go test -tags redtests ./parameterized -run '^TestRedAddParamGateRejectsNoTargets' -v`):

| Call on `NewTemplate(2)` | Today | Contract |
|---|---|---|
| `AddParamGate("", parameterized.Ry)` (no targets) | nil; `ParamNames()` is `[""]`, `ParamStepCounts()` is `map["":1]` | error naming the parameter; nothing declared |
| then `Bind(Params{"": 0.1})` | `at least one target is required` from `circuit.AddGate` | not reached |
| `AddGate(gates.NewHadamard())` (no targets) | nil; step appended | error naming the gate; nothing appended |
| then `Bind(Params{})` | `at least one target is required` from `circuit.AddGate` | succeeds (empty template) |
| `AddParamGate("theta", parameterized.Ry, 0)`, `AddGate(gates.NewCNOT(), 0, 1)` | accepted | unchanged |

The item's contract: a gate declared with zero targets is rejected at the
`Add*Gate` call, with an error naming the gate, not deferred to `Bind`.
Both methods have the gap.

Every caller in the module passes at least one target: `algorithm/h2.go`
(`H2Ansatz`), `algorithm/qaoa.go` (`QAOATemplate`), and every test. The
defect is reachable only by a caller outside the module.

## Classification

**Standard.** One new branch in each of `AddParamGate` and `AddGate`; no
signature, type, or cross-package change. Short design doc plus a
one-task plan.

## Decision 1: the guard lives at the top of each method, not in `checkTargets`

Item 18's spec left both placements open. Three were weighed.

**Inside `checkTargets`, before the loop.** One edit instead of two, and
the helper's name suggests it should own every target rule. Rejected:
the contract requires an error *naming the gate*, and `checkTargets`
knows only the target list and the qubit count. It would have to grow a
label argument (`what string`) and a formatting rule that differs per
caller (`%q` for a parameter name, `%s` for a gate name), which is more
plumbing than the two three-line guards it saves. Its one existing
error, `QubitsOutOfRangeError`, names the offending index and the range
and is shared with `circuit`; keeping the helper a pure range check keeps
that shape.

**In `AddParamGate` only.** Rejected: the item states both methods have
the gap, and the red test exercises both.

**At the top of each method, after its nil check and before
`checkTargets`.** Chosen. Each method already opens with an argument
check that names what it is checking (`parameter %q: factory must not be
nil`, `fixed gate must not be nil`); the empty-target guard is a second
argument check of the same kind, placed right after it, and each method
has the name it needs in hand: the parameter name in `AddParamGate`,
`gate.Name()` in `AddGate` (safe, since the nil check precedes it).
`checkTargets`, the `seen` allocation, and everything after the guard
are byte for byte what item 18 left them.

Argument order is check order: a nil factory is reported before missing
targets, because `factory` is the earlier argument and the existing nil
check already runs first. `AddParamGate("theta", nil)` with no targets
therefore returns the nil-factory error, as today.

## Decision 2: `fmt.Errorf`, matching the adjacent nil checks

The item requires an error naming the gate. Three shapes were weighed.

**Reuse `circuit`'s error.** `circuit.AddGate` returns
`errors.New("at least one target is required")` inline; there is no
exported sentinel to compare against or wrap, and the message names
nothing. Rejected as a type; its phrase is reused as the message's tail
(below) so the rejection reads the same at the declaration as it used to
at `Bind`.

**Reuse `quantum.InvalidGateApplicationError{Gate, RequiredLen, ActualLen}`.**
Its message is `gate %s requires %d qubits but got %d`. In `AddGate` the
required count is available through `quantum.GateQubitCount`, but in
`AddParamGate` no gate exists yet (the factory runs at `Bind`), so
`RequiredLen` would be a made-up value and `Gate` would have to hold a
parameter name. The type also states a width rule (targets must match
the gate's arity) that this check does not apply: width stays
`circuit.AddGate`'s to enforce at `Bind`, as it is for every other
mismatch (a two-qubit gate on one target is still accepted here and
rejected there). Rejected.

**A new package-local typed error** (say `NoTargetsError{Name string}`),
in the style of `MissingParameterError` and friends. Rejected. The
package's typed errors are `Bind`-time binding errors: each carries the
parameter name so a variational loop can react to a missing, unknown, or
non-finite value programmatically, and `algorithm.VQE` does. Amended
2026-09-09 (round 1 fix): more precisely, `VQE`'s `parameterShiftGradient`
constructs and returns a `MissingParameterError` itself and
`InvalidVQEInputError` unwraps so `errors.As` can reach the package's
types; no code in `algorithm` branches on one of them via `errors.As`
today. `VQE` produces and propagates these errors rather than reacting to
them; the argument that follows is unaffected. The declaration-time
argument checks are a different family: a nil factory
and a nil gate are caller bugs reported with `fmt.Errorf`, and an empty
target list is the third check of that kind. Typing one of the three
while its two siblings stay untyped would be the inconsistency; typing
all three is beyond the item. No caller in the module needs to branch on
this error.

**`fmt.Errorf`, mirroring the nil checks.** Chosen. Exact messages:

| Call | Error |
|---|---|
| `AddParamGate(name, factory)` with no targets | `parameter %q: at least one target is required` with `name` |
| `AddGate(gate)` with no targets | `fixed gate %s: at least one target is required` with `gate.Name()` |

The prefixes copy the two existing checks (`parameter %q: ...`, `fixed
gate ...`), so a parameter name is `%q`-quoted as everywhere in the
package (`""` renders as `""`) and a gate name is bare `%s` as in
`quantum.InvalidGateApplicationError`. The tail is `circuit.AddGate`'s
phrase verbatim.

## Decision 3: doc comments state the rejection

`AddParamGate`'s and `AddGate`'s doc comments each gain a sentence saying
at least one target is required and that a call with none is rejected at
the call, naming the parameter or the gate, rather than left for
`circuit.AddGate` to reject at `Bind`. The `Template` doc comment is
unchanged: "rejects every target as out of range" on the zero value stays
true, and the new rule is per method, not per template.

## Effect on item 18's tests

`TestZeroValueTemplateAddParamGateDoesNotPanic` calls
`AddParamGate("", Ry)` with no targets on a zero-value template and
asserts inside an `err == nil` branch. Item 18's spec anticipated this:
after item 19 the call errors, the branch is never entered, and the test
passes because its contract is the absence of a panic. Its setup, recover
guard, call, and assertions stay byte for byte. One in-test comment,
"the no-target call is the one that reaches the parameter bookkeeping",
becomes false (the call is now rejected before the bookkeeping) and is
rewritten to say the call is rejected too and must error rather than
panic; the comment is the only edit to that test.

`TestZeroValueTemplateIsAZeroQubitTemplate` makes no no-target call, by
design, and is unchanged.

## Rulings on edge cases

- **Nil factory and no targets.** `AddParamGate(name, nil)` returns the
  nil-factory error (Decision 1, check order). Pinned.
- **Nil gate and no targets.** `AddGate(nil)` returns `fixed gate must
  not be nil`, as today; `gate.Name()` is never reached on a nil gate.
- **Nothing declared after a rejection.** The guard precedes the `seen`
  write and the `steps` append, so `ParamNames()` and `ParamStepCounts()`
  are as before the call and `Bind` on the template still succeeds.
  Pinned.
- **Zero-value template, no targets.** Now rejected with the no-target
  error rather than accepted; with targets, still `QubitsOutOfRangeError`.
  No test asserts the old acceptance except inside a branch item 18's
  spec already ruled unreachable after this item.
- **Width mismatches** (a one-qubit gate on two targets, a two-qubit gate
  on one) are unchanged: accepted here, rejected by `circuit.AddGate` at
  `Bind`. Out of scope; the item is about the empty list only.
- **Duplicate targets** (`AddGate(cnot, 0, 0)`) are likewise unchanged
  and `Bind`'s to reject.
- **Concurrency.** The guard is a read of the argument list inside a
  method that is already a write; "safe for concurrent reads after all
  Add calls complete" is unchanged.

## Effect on callers

- `algorithm/h2.go`, `algorithm/qaoa.go`, `internal/examples/qaoa.go`,
  and every test pass at least one target to every `Add*Gate` call; none
  changes behavior. The prototype's `go run ./cmd/quantum -demo qaoa`
  still prints `VQE from the symmetric start: energy = -1.0000, expected cut = 2.0000`
  followed by `iterations accepted: 29, energy evaluations: 378`.
  Amended 2026-09-09 (round 1 fix): `internal/examples/qaoa.go` makes no
  `Add*Gate` call itself; it calls `algorithm.QAOATemplate`, which does,
  so it reaches `Template` only through that function, already listed
  here. The claim was vacuously true of the file, not false, but it does
  not belong in this bullet's list of direct callers.
- `Bind` is unchanged. A template built through the public API can no
  longer hold a step with zero targets, so the "at least one target is
  required" path in `circuit.AddGate` is unreachable from `Bind`; the
  call stays as it is, since `circuit.AddGate` still enforces width and
  uniqueness there.
- No Go file outside `parameterized/` changes. `CHANGELOG.md` gains an
  entry.

## Backward compatibility (ADR-style note)

Inputs that were accepted and now error: `AddParamGate` or `AddGate`
with no targets (previously accepted, then failed at the first `Bind`
with an untyped error naming nothing). Nothing with at least one target
behaves differently. No signature changes. Per
`docs/compatibility-policy.md` this is a personal project with no
external consumers; every internal caller is listed above and unaffected.

- **CHANGELOG:** following items 15 through 18, an Unreleased entry under
  the existing `### Fixed` heading, appended after item 18's entry
  (entries within a section are appended in landing order).
- **ADR-0010:** no amendment. Its consequence "parameter validation
  (missing/unknown/non-finite) fails at `Bind`" is about parameter
  values and stays true; target validation at declaration time is what
  the 2026-08-30 spec's API section already records for range.
- **The 2026-08-30 spec** (`docs/superpowers/specs/2026-08-30-parameter-binding-vqe-design.md`)
  says `AddParamGate`/`AddGate` "reject out-of-range targets at
  declaration time, mirroring `circuit.New`"; an empty list is now
  rejected at the same point. Nothing in it becomes false. No amendment.
- **Item 18's spec:** its "Item 19 follows on locally" section and the
  no-target ruling under "Rulings on edge cases" both say item 19 changes
  the acceptance; this design does exactly that. No amendment.

## Red test rulings

`TestRedAddParamGateRejectsNoTargets` moves into
`parameterized/parameterized_test.go` as `TestAddRejectsNoTargets`, the
package's `Test<Subject><Verb>...` style next to
`TestAddRejectsOutOfRangeTargets`. Setup, calls, and assertions are
unchanged; only the leading comment is rewritten to describe the
contract. It already exercises both methods (`AddParamGate("", Ry)` and
`AddGate(gates.NewHadamard())`), so the dispatch's "matching `AddGate`
case" is the verbatim copy's second half. Each assertion was checked
against the contract:

- `AddParamGate("", Ry)` with no targets must return an error: the
  contract's own sentence, and `""` is a legal name (item 15) so nothing
  else rejects the call. Consistent.
- `AddGate(H)` with no targets must return an error: same. Consistent.
- The failure messages print `ParamNames`, `ParamStepCounts`, and the
  `Bind` error on a regression, which is the diagnostic they were
  written to show. Consistent.

No assertion was ruled against the contract. The moved test says nothing
about the error's text or about the template afterwards, so
`TestAddNoTargetsErrorNamesTheGateAndDeclaresNothing` pins Decision 2's
naming, the nothing-declared ruling, and the nil-factory ordering
alongside it, rather than editing the moved test.

With item 19's test gone, `parameterized/backlog_red_test.go` holds only
item 21's test, which uses `errors`, `testing`, and `parameterized`; the
`gates` import is removed with the test.

## Testing

All in `parameterized/parameterized_test.go` (package `parameterized_test`):

- `TestAddRejectsNoTargets`: the moved red test. Fails today with
  `AddParamGate("", Ry) with no targets accepted: ParamNames() = [""], ParamStepCounts() = map[:1]; Bind then fails with at least one target is required; want the step rejected when added`.
- `TestAddNoTargetsErrorNamesTheGateAndDeclaresNothing`:
  `AddParamGate("theta", Ry)` errors with a message containing `"theta"`;
  `ParamNames()` and `ParamStepCounts()` are empty afterwards;
  `AddGate(gates.NewHadamard())` errors with a message containing
  `Hadamard`; `Bind(Params{})` then succeeds; `AddParamGate("theta", nil)`
  errors with `factory must not be nil`. Fails today at the first
  assertion (`err = <nil>`). Uses `strings.Contains`, as
  `TestBindErrorMessagesCarryNames` does; no new import.
- `go test -tags redtests ./parameterized -run '^TestRed'` must list
  exactly `TestRedCopyAfterDeclarationKeepsNamesAndCountsConsistent`
  (item 21) as failing; `algorithm`'s red run still lists exactly
  `TestRedParameterShiftAtHugeAngleMatchesReducedAngle` (item 20).

Prototype: every code and test change in the plan was applied to a
scratch copy of the repository at `44befc7`; both tests failed as stated
against the original methods and every test in the package passed after
the change, with `gofmt -l`, `go vet` (plain and `-tags redtests` on
`parameterized` and `algorithm`), `staticcheck`, `go build`,
`go test ./...`, `go test -race ./parameterized ./algorithm`, the two
red-tag runs, and the QAOA demo all as stated.

## Out of scope

Item 20. Item 21 (value-copy semantics of `Template`). Width or
uniqueness checks on targets at declaration time. Typing the three
declaration-time argument errors. Any change to `checkTargets`, `Bind`,
`ParamNames`, `ParamStepCounts`, or any exported signature.
