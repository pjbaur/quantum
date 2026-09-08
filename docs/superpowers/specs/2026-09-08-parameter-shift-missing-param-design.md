# Parameter-Shift Gradient and Incomplete Parameter Bindings — Design

Date: 2026-09-08
Status: Approved (backlog item text plus run-lead dispatch; no brainstorming gate).
Resolves: backlog item 16 (`.superpowers/backlog/enhancement-backlog-2026-08-27/item-16.md`)

## Context

`parameterShiftGradient(h, t, params, names)` in `algorithm/vqe.go` builds,
for each name in `names`, two copies of `params` with that name moved by
+/- pi/2, evaluates both, and stores the half difference. The copies are
plain map copies followed by `plus[name] += math.Pi / 2`: when `params`
lacks `name`, the map's zero value stands in, so the copies carry the
missing name at +/- pi/2 and `parameterized.Template.Bind` sees a
complete binding. The helper then returns the slope at an implicit 0 with
a nil error. A missing name that is *not* shifted stays absent from every
copy, so `Bind` rejects it with `MissingParameterError` on the first
evaluation. Whether the call fails therefore depends on which names the
caller shifts. Observed today, on the `gradientTargetTemplate` of
`algorithm/vqe_test.go` (declares `a` then `b`, H2 Hamiltonian;
`go test -tags redtests ./algorithm -run '^TestRedParameterShiftMissingParam' -v`
plus the prototype's red run):

| `params` | `names` | Today | Contract |
|---|---|---|---|
| `{a: 0.3}` | `[b]` | `{b: 0}`, 2 evaluations, nil error | `MissingParameterError{Name: "b"}`, 0 evaluations |
| `{b: 0.1}` | `[a]` | `{a: -0.18002729741406104}`, 2 evaluations, nil error: a real, wrong slope at an implicit `a = 0` | `MissingParameterError{Name: "a"}`, 0 evaluations |
| `{a: 0.3}` | `[a]` or `[a, b]` | `MissingParameterError{Name: "b"}` from `Bind`, 0 evaluations | unchanged |
| `{a: 0.3}` | `[b, a]` | `b` shifted twice at an implicit 0 (2 evaluations counted), then `Bind` rejects `a`'s shift: `MissingParameterError{Name: "b"}` with 2 evaluations reported | `MissingParameterError{Name: "b"}`, 0 evaluations |
| `{}` | `[]` | `{}`, 0 evaluations, nil error | `MissingParameterError{Name: "a"}`, 0 evaluations |

The item's contract offers two closures: "a name in `names` absent from
`params` is an error regardless of order, or the helper states that
callers must pass every declared parameter." The red test chooses the
first (it asserts an error in all four orderings). This design does both:
the helper checks, and its doc comment states the precondition.

`VQE` never reaches the defect: it builds `params` with every declared
name set to 0, overwrites from `InitialParams` (rejecting undeclared
names), and passes `names = t.ParamNames()`.

## Classification

**Standard.** Behavior change inside one unexported helper in
`algorithm/vqe.go`; no signature, type, or cross-package change. Short
design doc plus a one-task plan.

## Decision 1: the check lives in `parameterShiftGradient`, ahead of its loop

Three placements were weighed.

**`VQE` guaranteeing the invariant.** It already does (see Context), which
is why the item says nothing reaches the defect today. A change there
changes nothing observable, leaves the helper's contract silent (the
defect the item names), and cannot satisfy the acceptance test, which
calls the helper directly. Not a fix. `VQE`'s construction of `params` is
kept as is; it is what makes the helper's check unreachable from the
driver (Decision 4).

**`parameterShiftGradient` checking each name in `names` just before its
shift.** This closes the masking exactly where it happens, and the
outcome for every row above would be the same error. Rejected on two
counts: the reported name would follow `names` order, while `Bind`
reports the first missing name in declaration order, so the same gap
would be named differently depending on who caught it; and an incomplete
binding asked for no names at all (last row) would pass, making the
precondition depend on what was requested rather than on `params`.

**`parameterShiftGradient` checking every declared name against `params`
before the loop.** Chosen:

```go
for _, name := range t.ParamNames() {
	if _, ok := params[name]; !ok {
		return nil, 0, &parameterized.MissingParameterError{Name: name}
	}
}
```

The precondition becomes `Bind`'s own, stated in one sentence ("params
must bind every parameter the template declares"); the reported name is
the one `Bind` would report; no evaluation is consumed; and the check
does not depend on `names`. Cost is one map lookup per declared name per
gradient, which `VQE` computes once per iteration alongside `2n` energy
evaluations.

**"Both", as defense in depth.** That is what results: `VQE` guarantees
the invariant by construction and the helper checks it for every other
caller (tests today, a coordinate-descent or frozen-parameter caller
later). No second check is added inside `VQE`; it would be dead code.

**What is deliberately not checked.** A name in `names` that the template
never declared is left to `Bind`: the shift writes it into the copies,
but no shift can hide it, and `Bind` rejects it as
`UnknownParameterError` at the first evaluation with nothing consumed.
Likewise an undeclared key in `params`: the copies keep every key, so
`Bind` sees it. Duplicating either check would restate `Bind`'s rule with
no change in outcome. Pinned by
`TestParameterShiftUndeclaredNameIsRejectedByBind`.

## Decision 2: the error is `*parameterized.MissingParameterError`, not wrapped

The package's convention has two layers. Exported entry points return a
typed `Invalid<X>InputError{Reason}` (`InvalidVQEInputError`,
`InvalidQAOAInputError`, `InvalidQPEInputError`, `InvalidChshInputError`).
The unexported helpers `evaluate` and `parameterShiftGradient` return the
detecting package's error unchanged, and `VQE` wraps whatever they return
at its boundary through `wrapEvaluationError`, keeping the cause behind
`Unwrap`. The helper here stands in for `Bind`, catching a gap the shift
would otherwise hide from it, so it returns what `Bind` would have
returned: `&parameterized.MissingParameterError{Name: name}`.

Constructing another package's exported error type has precedent in the
module: `algorithm/backend.go:32` and `algorithm/qpe.go:126` build
`quantum.UnsupportedOperationError`, and `parameterized/parameterized.go:94`
builds `quantum.QubitsOutOfRangeError`. The type is exported with a
public `Name` field for exactly this use, and `errors.As` on it matches
whether the helper or `Bind` caught the gap.

Not wrapped: the helper is the detector and has nothing to add. Wrapping
with `fmt.Errorf("...: %w")` would change the text, and the test that
pins "the same error `Bind` returns" compares the text. The message
`parameter "b" missing from Bind values` stays accurate: `params` are the
values the helper hands to `Bind`.

Rejected alternatives: a new `algorithm` error type (the package has no
convention for helper errors, and it would split one condition across two
types depending on which function noticed it); `fmt.Errorf` or
`errors.New` (untyped, loses `errors.As`, different text from `Bind`).

If `VQE` could reach the check, `wrapEvaluationError` would report
`gradient evaluation at iteration 0 failed: parameter "b" missing from Bind values`
with the cause reachable through `errors.As`. It cannot (Decision 4).

## Decision 3: `names` stays a parameter

Every caller was assessed. `VQE` (`algorithm/vqe.go:350`) passes
`t.ParamNames()`. Five call sites in `algorithm/vqe_test.go` (lines 37,
120, 438, 501, 590) pass `tmpl.ParamNames()`. The two red tests pass
explicit slices: item 16's four orderings, item 17's `[]string{"a"}`.
Nothing else calls the helper.

Dropping `names` and deriving the list from the template was weighed and
rejected:

1. It does not remove the defect on its own. `params = {}` on a template
   declaring only `a` would still shift `a` from an implicit 0 with
   derived names, so the completeness check is needed either way.
2. The acceptance test's substance is the four orderings of `names`; with
   the parameter gone, the four cases collapse into one and the test no
   longer says what the item says.
3. Item 17's tagged test would need a signature edit to keep compiling
   under `go vet -tags redtests`, against the rule that it stays as is.
4. The order dependence was a symptom of the masking, not a defect of
   ordering. With a complete binding, order never changed the result and
   never will: each name's two evaluations are independent. Pinned by
   `TestParameterShiftDifferentiatesOnlyNamedParams`, which also pins
   that a subset costs only its own evaluations and returns only its own
   components.

The parameter is kept and its contract is now stated on the helper: any
subset of the declared names, in any order, each costing two evaluations;
the result map holds exactly those names.

## Decision 4: no `InvalidVQEInputError` Reason, no Decision 3 row

`VQE` cannot reach the check (Context), so no new `Reason` text is ever
produced by the driver. A row in item 14's Decision 3 table for an
unreachable message would misdescribe the taxonomy. Instead, following
item 15's precedent, one sentence is appended to the "Helper shape and
the items that follow" paragraph of
`docs/superpowers/specs/2026-09-08-vqe-input-validation-design.md`
recording where item 16 landed, that `VQE` and the wrap are unchanged,
and that item 17 stays open in the loop body.

## Item 17 follows on locally

Item 17 (evaluation undercount on failure): the loop body does
`evals += 2` only after both shifted evaluations succeed, so a failure on
the minus evaluation loses the plus one. Its fix is confined to the loop
body: count each evaluation as it succeeds. The item 16 check precedes
the loop, returns a literal `0` before any evaluation, and never touches
`evals`; the two changes share no lines. Item 17's test stays under the
`redtests` tag and still fails after this item; the prototype's red-tag
run of `algorithm` lists exactly that one failure.

## Rulings on edge cases

- **Empty `names` with an incomplete binding** is an error. The
  precondition is on `params`, not on what was requested. Pinned by the
  "both missing, nothing shifted" row of
  `TestParameterShiftMissingParamErrorMatchesBind`.
- **Empty `names` with a complete binding**, including a template with no
  parameters (`ParamNames()` empty, `params` empty), returns an empty map,
  0 evaluations, nil error: unchanged. `VQE`'s no-parameter path
  (`TestVQENoParameterTemplateReportsOverflowingHamiltonian`,
  `TestVQEEnergyGuardReasons`) relies on it and keeps passing.
- **Several names missing**: the first in declaration order is reported,
  as `Bind` does. Pinned by both "both missing" rows.
- **Non-finite values in `params`** are not checked here; `Bind` rejects
  them at the first evaluation as `InvalidParameterValueError`, and `VQE`
  guards them earlier. Unchanged.
- **Duplicates in `names`** each cost two evaluations and write the same
  component. No caller does it; unchanged and out of scope.

## Effect on callers

- `algorithm/vqe.go` `VQE`: no code change. One extra pass of
  `len(names)` map lookups per iteration. `algorithm/vqe_driver_test.go`
  and `algorithm/vqe_bench_test.go` are unchanged and pass. The
  prototype's `go run ./cmd/quantum -demo qaoa` still prints
  `VQE from the symmetric start: energy = -1.0000, expected cut = 2.0000`
  followed by `iterations accepted: 29, energy evaluations: 378`.
- Tests calling the helper with `tmpl.ParamNames()`: complete bindings,
  unaffected.
- `internal/examples/qaoa.go` and `cmd/quantum/main.go` call `VQE` only.
- No Go file outside `algorithm/` changes. `CHANGELOG.md` and item 14's
  spec gain text.

## Backward compatibility (ADR-style note)

Nothing exported changes. Internally, calling `parameterShiftGradient`
with an incomplete binding now always errors before any evaluation and
reports 0 evaluations; previously it returned a gradient at an implicit 0
when only the missing names were shifted, and otherwise errored after
whatever evaluations preceded the first unmasked gap. Per
`docs/compatibility-policy.md` this is a personal project with no
external consumers; every internal caller is listed above and unaffected.

- **CHANGELOG:** following items 14 and 15, an Unreleased entry. It goes
  under the existing `### Fixed` heading, after item 15's entry (entries
  within a section are appended in landing order), and says plainly that
  the helper is unexported and no exported behavior changes.
- **ADR-0010:** no amendment. Parameter validation at `Bind` is one of
  its consequences; this change applies `Bind`'s rule one step earlier in
  a helper the shift would otherwise blind, and changes no decision.
- **Item 14's spec:** the one-sentence amendment described in Decision 4.

## Red test rulings

`TestRedParameterShiftMissingParamIsRejected` moves into
`algorithm/vqe_test.go` as `TestParameterShiftRejectsMissingParam`, the
package's `TestParameterShift<Verb>...` style
(`TestParameterShiftMatchesFiniteDifference`,
`TestParameterShiftIsBlindToRescaledFactory`). Setup, the four cases, and
the assertion are unchanged; only the comment is rewritten to describe the
contract. Each assertion was checked against the contract:

- `gradientTargetTemplate()` declares `a` and `b`; `incomplete` binds only
  `a`. Consistent with the Context table.
- All four orderings of `names` must return a non-nil error. Consistent
  with Decision 1: an incomplete binding errors before any evaluation
  whatever `names` holds.

No assertion was ruled against the contract. The moved test is weaker
than the contract (it asserts only `err != nil`), so
`TestParameterShiftMissingParamErrorMatchesBind` pins the type, the
reported name, the zero evaluation count, the nil gradient, and equality
with `Bind`'s own error text alongside it, rather than editing the moved
test.

## Testing

All in `algorithm/vqe_test.go` (package `algorithm`, which can call the
helper):

- `TestParameterShiftRejectsMissingParam`: the moved red test. Fails
  today on its `b only` case only.
- `TestParameterShiftMissingParamErrorMatchesBind`: six rows covering a
  masked gap for each name, an unmasked gap, the wasted-evaluations
  ordering, both names missing, and empty `names`. Every row asserts
  `errors.As` on `*parameterized.MissingParameterError`, the expected
  `Name`, nil gradient, 0 evaluations, and `err.Error()` equal to
  `tmpl.Bind(params)`'s error text. Fails today on four rows (the two
  masked gaps, the wasted-evaluations ordering with `evals = 2`, and empty
  `names`); the two `Bind`-caught rows pass today and pin that the
  reported name and text do not change.
- `TestParameterShiftUndeclaredNameIsRejectedByBind`: `names = [c]` on a
  complete binding is `UnknownParameterError{Name: "c"}` with nil
  gradient and 0 evaluations. Passes before and after; pins Decision 1's
  "deliberately not checked".
- `TestParameterShiftDifferentiatesOnlyNamedParams`: with a complete
  binding, `[b, a]` equals `[a, b]` component for component after 4
  evaluations, and `[b]` returns exactly `{b}` after 2. Passes before and
  after; pins Decision 3.
- `go test -tags redtests ./algorithm -run '^TestRed'` must list exactly
  `TestRedParameterShiftCountsEvaluationsBeforeFailure` (item 17) as
  failing. The `parameterized` red tests (items 18 and 19) are untouched.

Prototype: every code and test change in the plan was applied to a
scratch copy of the repository at `23d60a1`; the red results above were
observed against the original helper and every test passed after the
check was added, with `gofmt -l`, `go vet` (plain and `-tags redtests` on
`algorithm` and `parameterized`), `staticcheck`, `go build`,
`go test ./...`, `go test -race ./algorithm`, the red-tag run, and the
QAOA demo all as stated.

## Out of scope

Item 17 (evaluation count on failure). Items 18 and 19 (`AddParamGate`
gaps in `parameterized`). Any change to `VQE`'s body or to any exported
API. Duplicate names in `names`. Rewording `MissingParameterError`'s
message.
