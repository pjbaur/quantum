# Empty Parameter Name and the One-Gate-Per-Parameter Check — Design

Date: 2026-09-08
Status: Approved (backlog item text plus run-lead dispatch; no brainstorming gate).
Resolves: backlog item 15 (`.superpowers/backlog/enhancement-backlog-2026-08-27/item-15.md`)

## Context

`parameterized.Template` keeps every instruction as a `step` carrying a
name, a factory, a fixed gate, and targets. Two kinds exist: a fixed gate
(`AddGate`, factory nil) and a parameter-driven gate (`AddParamGate`,
factory non-nil, `AddParamGate` rejects a nil one). The package tells them
apart in two places, and the two places disagree:

| Where | Discriminator | What a parameter named `""` looks like |
|---|---|---|
| `Bind` (`parameterized/parameterized.go:147`) | `s.factory != nil` | a parameter step: bound from `values[""]` |
| `ParamStepCounts` (`parameterized/parameterized.go:77`) | `s.param != ""` | a fixed step: never counted |

`AddParamGate` accepts `""` like any other string, so `ParamNames` lists
it, `Bind` requires it, and `ParamStepCounts` omits it. `VQE`'s
one-gate-per-parameter check (`validateVQEStructure` in
`algorithm/vqe.go:181-187`) reads `ParamStepCounts()[name]` for each name in
`ParamNames()` and rejects a count above one, so for `""` it reads zero and
passes. Observed today
(`go test -tags redtests ./algorithm -run '^TestRedVQEEmptyName' -v`):

| Input | Today | Contract |
|---|---|---|
| `""` driving two `Ry` gates on qubit 0, H2 Hamiltonian | `ParamNames() = [""]`, `ParamStepCounts() = map[]`, `VQE` returns `Converged: true` after 1 iteration and 4 evaluations, no error | `InvalidVQEInputError`, as for any other name driving two gates |

The item's contract: the one-gate-per-parameter rule stated on `VQE` and
`parameterShiftGradient` holds for every declared name, so a `""`
parameter that drives several steps is rejected with
`InvalidVQEInputError` like any other.

## Classification

**Standard.** One-expression behavior change inside
`parameterized.Template.ParamStepCounts`, no signature or type change; the
acceptance test lives in `algorithm` and `algorithm/vqe.go` does not
change. Short design doc plus a one-task plan.

## Decision 1: the root fix is `ParamStepCounts`'s discriminator

The item's own observation names the defect: `ParamStepCounts` "treats the
empty string as its fixed-step marker". `AddParamGate` accepting `""` is
only what exposes it. Three places were weighed for the fix.

**`validateVQEStructure`** (the extension point item 14's spec named):
add a rejection of the empty name to its template loop. Rejected. It
treats the symptom in one consumer and leaves `ParamStepCounts` violating
its own doc comment ("per declared parameter name, how many template steps
consume it") for every other caller. It also over-rejects relative to the
contract: "like any other" means a `""` parameter driving exactly one gate
satisfies the rule and must stay accepted, so the check would have to be
"empty name and count 0", which encodes the sentinel bug into the consumer
instead of removing it.

**`AddParamGate`** rejecting `""` at the source with a new error type.
Rejected. The package has no rule on what a name is and no
`InvalidParameterNameError`; a name is an opaque map key, and `Bind`,
`MissingParameterError`, `UnknownParameterError`, and
`InvalidParameterValueError` all already handle `""` correctly. Forbidding
one string while accepting `" "` or `"\n"` would be an arbitrary rule
invented to protect an accessor from its own sentinel. The item's contract
presumes the template accepts the name (`VQE` is to reject it, with
`InvalidVQEInputError`), and the red test's setup fails the test if
`AddParamGate("")` errors. And it would leave the latent disagreement
between `Bind` and `ParamStepCounts` in place, guarded only by an input
rule.

**`ParamStepCounts`** discriminating on `s.factory != nil`, the test
`Bind` already applies. Chosen. This removes the disagreement rather than
fencing it off: a step is parameter-driven exactly when it carries a
factory, `AddParamGate` rejects a nil factory and `AddGate` stores none, so
the two kinds partition the steps and no name can collide with a marker.
Every consumer of `ParamStepCounts` is protected, `VQE`'s existing loop
rejects the template with its existing message, and nothing in
`algorithm/` changes. Not "both": with the counts correct, an empty-name
branch in `validateVQEStructure` is either dead code (count 0 can no longer
occur for a declared name) or an over-rejection.

The `step` type's comment, which said a fixed gate is one with
`param == ""`, is corrected to state the factory-based invariant so the
sentinel is not reintroduced.

## Decision 2: the empty string stays a legal name

`AddParamGate`'s doc comment is unchanged. The `step` and `ParamStepCounts`
comments say that `""` is a name like any other and that the factory, not
the name, distinguishes the kinds of step. No name validation is added to
`parameterized`; see Decision 1.

## Decision 3: Reason string

Unchanged from item 14's Decision 3 table:

| Condition | Reason |
|---|---|
| `""` drives several gates | `parameter "" drives 2 template steps; the parameter-shift gradient requires exactly one gate per parameter` |

`%q` renders the empty name as `""`, which reads as "the parameter whose
name is the empty string" and quotes the name the way every other Reason
does. No `algorithm/` code changes to produce it.

## Rulings on edge cases

- **`""` driving exactly one gate** is accepted and optimized; `VQE`
  treats the name as an opaque key throughout (`params`, `formatPoint`,
  the result). Pinned by `TestVQEAcceptsEmptyNameDrivingOneGate`, which
  runs the H2 ansatz with its parameter renamed to `""` and expects the
  same energy, iteration count, and final angle as with `"theta"`.
- **Fixed gates** stay absent from the counts; a template of only fixed
  gates returns an empty map. Pinned by a new `TestParamStepCounts` case.
- **A fixed gate between two `""` steps** is neither counted for `""` nor
  does it hide them. Pinned by the other new `TestParamStepCounts` case.

## Effect on callers

- `algorithm/h2.go` (`"theta"`) and `algorithm/qaoa.go`
  (`gamma_e<k>`, `beta_q<k>`) declare non-empty names; their counts are
  unchanged.
- `internal/examples/qaoa.go` calls `ParamNames` only. The prototype's
  `go run ./cmd/quantum -demo qaoa` still prints
  `VQE from the symmetric start: energy = -1.0000, expected cut = 2.0000`.
- `algorithm/vqe.go` is the only caller of `ParamStepCounts` and needs no
  change; `TestVQERejectsMultiGateParameter` keeps passing.

## Backward compatibility (ADR-style note)

Inputs that were accepted and now error: `VQE` on a template whose `""`
parameter drives several gates (previously ran and reported `Converged`
against the higher-harmonic gradient the check exists to prevent).
Outputs that change: `ParamStepCounts` includes `""` when it is declared
(previously absent). Still accepted: `AddParamGate("", ...)`. Per
`docs/compatibility-policy.md` this is a personal project with no external
consumers; every internal caller is listed above and unaffected.

- **CHANGELOG:** following item 14's precedent of an Unreleased entry for a
  changed validation contract, this fix gets an entry under Unreleased. It
  goes under a new `### Fixed` heading (the file has none yet) rather than
  `### Added`, because it corrects an accessor that contradicted its own
  doc comment and adds no contract.
- **ADR-0010:** no amendment. Its decision (template materialization over
  symbolic gates) and its consequences (parameter validation at `Bind`;
  QAOA reuses the template unchanged) are untouched; the fix aligns one
  accessor with the discriminator `Bind` already used. The ADR carries no
  Decision Log section, and `docs/adr/README.md` reserves amendments for
  changed circumstances.
- **Item 14's spec** (`docs/superpowers/specs/2026-09-08-vqe-input-validation-design.md`,
  "Helper shape and the items that follow") says `validateVQEStructure`'s
  loop is where an empty-name check belongs. One sentence is appended to
  that paragraph recording that item 15 landed in `ParamStepCounts` and the
  loop needed no extension, so a later reader does not add a redundant
  check.

## Red test rulings

`TestRedVQEEmptyNameParameterEscapesStepCountCheck` moves into
`algorithm/vqe_test.go` renamed `TestVQEEmptyNameParameterIsCheckedLikeAnyOther`,
setup and assertions unchanged. Each assertion was checked against the
contract:

- `AddParamGate("", parameterized.Ry, 0)` must succeed, twice: the
  contract presumes the template accepts the name (Decision 1, second
  option rejected). Consistent.
- `ParamNames()` must equal `[""]`: `ParamNames` is untouched and already
  returns this. Consistent.
- `VQE(h, tmpl, VQEOptions{})` must return `InvalidVQEInputError`: the
  contract's own sentence. Consistent.

No assertion was ruled against the contract. The failure message's
`tmpl.ParamStepCounts()` read now prints `map[:2]` on a regression, which
is the diagnostic the message was written to show.

## Testing

- Moved red test as above.
- `TestParamStepCounts` (`parameterized/parameterized_test.go`) gains
  `"empty name driving two gates around a fixed gate"` (`map[string]int{"": 2}`)
  and `"fixed gates only"` (empty map). The first fails today
  (`ParamStepCounts() = map[], want map[:2]`); the second passes today and
  pins that the new discriminator still skips fixed steps.
- `TestVQEAcceptsEmptyNameDrivingOneGate` (`algorithm/vqe_test.go`): passes
  before and after; pins the other half of "like any other".
- `go test -tags redtests ./algorithm -run '^TestRed'` must list exactly
  `TestRedParameterShiftMissingParamIsRejected` and
  `TestRedParameterShiftCountsEvaluationsBeforeFailure` (items 16 and 17)
  as failing.

Prototype: every code and test change in the plan was applied to a scratch
copy of the repository at `b50ac23`; the two red tests failed as stated
against the original `ParamStepCounts` and passed after the one-expression
change, with `gofmt -l`, `go vet` (plain and `-tags redtests`),
`staticcheck`, `go build`, `go test ./...`, `go test -race` on both
packages, and the QAOA demo all clean.

## Out of scope

Items 16 and 17. Any rule on what a parameter name may be. Rewording the
one-gate-per-parameter Reason.
