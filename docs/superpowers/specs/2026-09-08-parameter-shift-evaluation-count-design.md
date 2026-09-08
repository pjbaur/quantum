# Parameter-Shift Gradient Evaluation Count on Failure — Design

Date: 2026-09-08
Status: Approved (backlog item text plus run-lead dispatch; no brainstorming gate).
Resolves: backlog item 17 (`.superpowers/backlog/enhancement-backlog-2026-08-27/item-17.md`)

## Context

`parameterShiftGradient(h, t, params, names)` in `algorithm/vqe.go`
evaluates, for each name in `names`, the energy at that parameter shifted
by +pi/2 and then by -pi/2, and stores the half difference. Its doc
comment promises to return "the number of energy evaluations consumed".
The loop body keeps that count in `evals` and does `evals += 2` only after
both shifted evaluations of a name have succeeded:

```go
ePlus, err := evaluate(h, t, plus)
if err != nil {
	return nil, evals, err
}
eMinus, err := evaluate(h, t, minus)
if err != nil {
	return nil, evals, err
}
evals += 2
grad[name] = (ePlus - eMinus) / 2
```

So when the +pi/2 evaluation completes and the -pi/2 evaluation fails,
the count returned with the error omits the evaluation that ran. Observed
at `21626ff` with
`go test -tags redtests ./algorithm -run '^TestRedParameterShiftCounts' -v`
(a factory that returns a valid `Ry` for non-negative values and a
two-qubit gate, which `Bind` rejects on one target, for negative ones;
`params = {a: 0.3}`, `names = [a]`):

```
evals = 0 after one successful and one failed evaluation, want 1 (evaluations consumed); err = gate CNOT requires 2 qubits but got 1
```

The same undercount occurs after any number of completed names: with
`names = [a, b]` and `b`'s -pi/2 evaluation failing, the helper returns 2
where 3 evaluations completed. A failure on a +pi/2 evaluation is counted
correctly today (0 for the first name, 2 for the second), because nothing
completed since the last `evals += 2`.

The count is invisible from `VQE`: `algorithm/vqe.go:370-374` reads
`grad, gradEvals, err := parameterShiftGradient(...)`, returns
`nil, wrapEvaluationError(...)` when `err != nil`, and adds `gradEvals`
to `VQEResult.Evaluations` only on the success path. A failed gradient
therefore never contributes to any reported total; the miscount is
observable only to a direct caller of the helper (today, the tests in
`algorithm/vqe_test.go`). The item's contract is nonetheless explicit:
"the returned count includes every evaluation that ran, on the error path
as well."

Item 16 (`21626ff`) added a completeness check ahead of the loop that
returns `nil, 0, &parameterized.MissingParameterError{...}` before any
evaluation, and left the loop body byte for byte unchanged so that this
item's fix edits only lines inside the loop.

## Classification

**Standard.** Behavior change on the error path of one unexported helper
in `algorithm/vqe.go`; no signature, type, or cross-package change. Short
design doc plus a one-task plan.

## Decision 1: "evaluations consumed" means evaluations that returned an energy

Three readings of "consumed" were weighed for the failure path.

**Calls to `evaluate` attempted.** Under this reading the red test's
scenario (one completed, one failed) would report 2. Rejected: the item's
own wording is "every evaluation that ran", and the failed evaluation did
not run in any sense a caller can charge for. `evaluate` fails at one of
four points: `Bind` (before any circuit exists), `state.New`, `Execute`
(part way through the circuit), or `Energy` (after the circuit ran). The
failure a factory can cause, a gate `Bind` rejects, pays nothing at all,
and a count of energies cannot express "partly executed". Counting the
attempt would also contradict the acceptance test, which asserts 1.

**Calls that completed successfully.** Chosen. An evaluation is counted
the moment `evaluate` returns a nil error, which is exactly how `VQE`
counts its own evaluations (`algorithm/vqe.go:358-362` and `402-406`:
`evals++` after the nil-error check, never before). The helper and the
driver then use one rule, and the number the helper returns is the number
`VQE` would have added to `VQEResult.Evaluations` had the gradient
succeeded. It matches the red test (1) and the item text (the +pi/2
evaluation "ran", the -pi/2 one did not).

**"Completed" as distinct from "successful".** `evaluate` returns either
an energy with a nil error or a non-nil error with a zero energy; there is
no completed-but-failed outcome to distinguish, so the two readings
coincide and the doc comment uses "returns its energy" to name the event.

Amended 2026-09-08 (round 1 fix): a black-box red test
(`TestRedParameterShiftDuplicateNameCostsOnePair`) called the helper with
`names=[a,a]` and asserted `evals == 2` for the one gradient component
returned, reading the `names` parameter's doc comment ("may list any
subset ... in any order") as promising deduplication. It does not: the
loop shifts and counts each element of names as it is reached, not each
unique name, so a repeated name runs its pair of evaluations again and
`evals == 4` is the count this decision requires (two evaluations
completed, once per occurrence). The doc comment's `names` sentence in
`algorithm/vqe.go` is amended to say so: "names selects which of those to
differentiate and may list any subset in any order, and repeats: the loop
shifts and counts each element of names as it is reached, not each unique
name, so a name listed twice runs its pair of evaluations twice while
grad still ends up with one entry for it, the last occurrence's slope.
VQE never repeats a name; it calls with t.ParamNames(), which returns
declared names without duplicates." Test deleted; ruling recorded in
`.superpowers/backlog/enhancement-backlog-2026-08-27/item-17-round-1-fix.md`.

## Decision 2: the count is returned alongside the error, and is exact

The helper already returns `evals` with every error out of the loop; the
current contract is not "0 on error" but "the count of names finished so
far, times two", which is what undercounts. The alternatives were:

**Return 0 on every error** and document the count as meaningful only on
success. This is the common Go shape when other results are unspecified on
error. Rejected: the doc comment promises the count without qualification,
the item's contract asks for the partial count on the error path, the red
test asserts it, and the information is real (a caller keeping an
evaluation budget has paid for those evaluations whether or not the
gradient came back). Zeroing it would turn an undercount into a larger
one.

**Return the partial count, made exact.** Chosen. `io.Reader.Read` is the
standard-library precedent for a count that is meaningful alongside a
non-nil error: `n` bytes were read even though `err != nil`. The doc
comment cites it so a reader does not assume the usual "ignore other
results on error" rule.

**Whether `VQE` should sum the count before returning the wrapped
error.** No. `VQE` returns a nil `*VQEResult` with the error, so the sum
would be computed and dropped; adding it would be dead code. Item 14's
spec anticipated this: "the wrap in `VQE` passes the gradient's error and
count straight through, so a later fix to the count needs no change here."
`VQE`'s body is not edited.

## Decision 3: count each evaluation as it completes

Two edits to the loop body would produce the exact count:

**Keep `evals += 2` after the pair and return `evals + 1` from the -pi/2
failure branch.** Rejected: it encodes the fix as an arithmetic
correction at one exit and leaves the counter wrong between the two
evaluations, so a later edit that adds a third exit (or moves the pair)
would have to remember the correction.

**Replace `evals += 2` with an `evals++` directly after each nil-error
check.** Chosen:

```go
ePlus, err := evaluate(h, t, plus)
if err != nil {
	return nil, evals, err
}
evals++
eMinus, err := evaluate(h, t, minus)
if err != nil {
	return nil, evals, err
}
evals++
grad[name] = (ePlus - eMinus) / 2
```

`evals` is now correct at every statement of the loop, the two return
statements need no arithmetic, and the shape is the one `VQE` uses after
its own `evaluate` calls. On the success path the total is unchanged: two
per name. The edit touches lines 93-102 of `algorithm/vqe.go` at
`21626ff` only; the item 16 check at lines 77-81 and everything after the
loop are untouched.

## Decision 4: the item 16 early return stays `0`

The completeness check returns the literal `0` before the loop starts.
Under Decision 1 that is exact: no evaluation was attempted, let alone
completed. Nothing changes there, and the doc comment says so in one
sentence so a reader does not look for a counter it might have missed.

## Decision 5: the doc comment's contract sentence

The last paragraph of `parameterShiftGradient`'s doc comment, currently

```
// Returns the gradient keyed by name plus the number of energy evaluations
// consumed.
```

becomes

```
// Returns the gradient keyed by name plus the number of energy evaluations
// consumed: two per name in names. On error the gradient is nil and the
// count is still exact: each evaluation is counted as evaluate returns its
// energy, the rule VQE applies to its own evaluations, so the count covers
// every evaluation that completed before the failure and not the failing
// one, which returned no energy (how far into evaluate it got is not
// something a count of energies can express). Like io.Reader's n, the
// count is meaningful alongside a non-nil error; VQE discards it with the
// error, so only a direct caller sees it. The completeness check above
// returns 0 because it precedes the first evaluation.
```

It states the success-path cost, the error-path meaning and its rule, why
the failing evaluation is excluded, the precedent for reading a count next
to an error, who can observe it, and how the item 16 return fits. Nothing
else in the comment changes.

## Decision 6: no `InvalidVQEInputError` Reason, no Decision 3 row

`VQE` produces no new text: it discards the count with the error and
wraps the error exactly as before. Following items 15 and 16, one sentence
is appended to the "Helper shape and the items that follow" paragraph of
`docs/superpowers/specs/2026-09-08-vqe-input-validation-design.md`
recording where item 17 landed, that `VQE` and the wrap are unchanged as
that paragraph anticipated, and that `algorithm/backlog_red_test.go` is
deleted. Item 16's spec has a section "Item 17 follows on locally" that
describes this fix as "count each evaluation as it succeeds"; it is left
as written, since that is what lands.

## Red test rulings

`TestRedParameterShiftCountsEvaluationsBeforeFailure` moves into
`algorithm/vqe_test.go` as `TestParameterShiftCountsEvaluationsBeforeFailure`,
the package's `TestParameterShift<Verb>...` style
(`TestParameterShiftMatchesFiniteDifference`,
`TestParameterShiftRejectsMissingParam`). Setup, the call, and both
assertions are unchanged; only the leading comment is rewritten to
describe the contract. Each assertion was checked against the contract:

- `err == nil` is fatal. Under the contract the -pi/2 copy binds `a` at
  `0.3 - pi/2 < 0`, the factory returns a two-qubit gate, and `Bind`
  rejects it on one target as `quantum.InvalidGateApplicationError`, which
  the helper returns unwrapped. Consistent.
- `evals != 1` is fatal. Under Decision 1 the +pi/2 evaluation returned an
  energy (1) and the -pi/2 evaluation did not (not counted). Consistent.

No assertion was ruled against the contract. The moved test covers one
failure position, so `TestParameterShiftEvaluationCountOnFailure` pins
all four (plus and minus shift of the first and of the second of two
names, expecting 0, 1, 2, 3) together with the nil gradient and the
unwrapped `Bind` error, rather than editing the moved test.

## Effect on the red-test file

After the move `algorithm/backlog_red_test.go` holds no tests. The file
is deleted rather than kept as a tagged file with only a header comment:
an empty tagged file would compile but document nothing, and its header
tells readers to run tests it no longer contains.
`parameterized/backlog_red_test.go` (items 18 and 19) is untouched.
`go vet -tags redtests ./algorithm` must still pass with no file in the
package carrying the tag; it does, since a build tag that matches no file
leaves the package as it is for the default build.
`go test -tags redtests ./algorithm -run '^TestRed'` then reports
`ok ... [no tests to run]`, which is the expected outcome, not a failure.

Amended 2026-09-08 (round 1 fix): item 20's tagged file
(`algorithm/backlog_red_numeric_test.go`) was added after this design was
written, so the prediction above no longer holds:
`go test -tags redtests ./algorithm -run '^TestRed'` now lists and fails
exactly that one test, not `[no tests to run]`. `go vet -tags redtests
./algorithm` still passes, because it is item 20's file, not this design's
deleted `backlog_red_test.go`, that carries the tag.

## Effect on callers

- `algorithm/vqe.go` `VQE`: no code change. On success the helper's count
  is the same total as before, so `VQEResult.Evaluations` is unchanged;
  `algorithm/vqe_driver_test.go`'s `1 + 3k` invariant holds. On failure
  `VQE` still returns `nil` and the wrapped error. The prototype's
  `go run ./cmd/quantum -demo qaoa` still prints
  `VQE from the symmetric start: energy = -1.0000, expected cut = 2.0000`
  followed by `  iterations accepted: 29, energy evaluations: 378`.
- Tests calling the helper on the success path
  (`TestParameterShiftMatchesFiniteDifference`,
  `TestParameterShiftDifferentiatesOnlyNamedParams`) assert 4 and 2 and
  keep passing. Tests on the item 16 path assert 0 and keep passing
  (Decision 4).
- `internal/examples/qaoa.go` and `cmd/quantum/main.go` call `VQE` only.
- No Go file outside `algorithm/` changes. `CHANGELOG.md` and item 14's
  spec gain text.

## Backward compatibility (ADR-style note)

Nothing exported changes. Internally, the count `parameterShiftGradient`
returns with an error now includes a completed +pi/2 evaluation whose
-pi/2 partner failed; previously it did not. Per
`docs/compatibility-policy.md` this is a personal project with no external
consumers; every internal caller is listed above and unaffected.

- **CHANGELOG:** following items 14 to 16, an Unreleased entry under the
  existing `### Fixed` heading, after item 16's entry (entries within a
  section are appended in landing order), saying plainly that the helper
  is unexported, that `VQE`'s totals are unchanged, and that the
  `algorithm` red-test file is removed.
- **ADR-0010:** no amendment; no decision changes.
- **Item 14's spec:** the one-sentence amendment described in Decision 6.

## Testing

All in `algorithm/vqe_test.go` (package `algorithm`, which can call the
helper):

- `TestParameterShiftCountsEvaluationsBeforeFailure`: the moved red test.
  Fails today with `evals = 0 ... want 1`.
- `TestParameterShiftEvaluationCountOnFailure`: a two-parameter template
  whose factory returns a two-qubit gate when `|v| > 10`, with
  `names = [a, b]`; four rows put the failure on the plus or minus shift
  of the first or second name (`a = 9`, `a = -9`, `b = 9`, `b = -9`, the
  other parameter at 0) and expect 0, 1, 2, 3. Every row asserts
  `errors.As` on `*quantum.InvalidGateApplicationError`, a nil gradient,
  and the count. Fails today on the two minus-shift rows (`evals = 0,
  want 1` and `evals = 2, want 3`); the two plus-shift rows pass today and
  pin that correct counts stay correct.
- `go test -tags redtests ./algorithm -run '^TestRed'` reports
  `[no tests to run]`. The `parameterized` red tests (items 18 and 19) are
  untouched and still fail under the tag.

Prototype: every code, test, and documentation change in the plan was
applied to a scratch copy of the repository at `21626ff`; the red results
above were observed against the original loop, every test passed after
the two `evals++` edits, and `gofmt -l`, `go vet` (plain and
`-tags redtests` on `algorithm` and `parameterized`), `staticcheck`,
`go build`, `go test ./...`, `go test -race ./algorithm`, the red-tag run,
and the QAOA demo were all as stated.

## Out of scope

Items 18 and 19 (`AddParamGate` gaps in `parameterized`). Any change to
`VQE`'s body, to `evaluate`, or to any exported API. Distinguishing where
inside `evaluate` a failure happened. Reporting a partial gradient on
error.
