# Parameter-Shift Angle Bound — Design

Date: 2026-09-09
Status: Approved (backlog item text plus run-lead dispatch; no brainstorming gate).
Resolves: backlog item 20 (`.superpowers/backlog/enhancement-backlog-2026-08-27/item-20.md`)

## Context

`parameterShiftGradient(h, t, params, names)` in `algorithm/vqe.go`
evaluates, for each name in `names`, `E(theta + pi/2)` and
`E(theta - pi/2)` and stores the half difference. Both shifted angles are
formed in float64: `plus[name] += math.Pi / 2`, `minus[name] -= math.Pi / 2`.
float64 rounds each sum to a multiple of `theta`'s spacing (one ulp,
`2^(e-52)` for `|theta|` in `[2^e, 2^(e+1))`), so the shift lands within
half a spacing of its true offset. The spacing grows with the angle:

| `theta` | spacing | `(theta + pi/2) - theta` | `(theta - pi/2) - theta` |
|---|---|---|---|
| `2^26` | `1.5e-8` | `1.5707963258` | `-1.5707963258` |
| `2^40` | `2.4e-4` | `1.57080078125` | `-1.57080078125` |
| `2^48` | `0.0625` | `1.5625` | `-1.5625` |
| `2^51` | `0.5` | `1.5` | `-1.5` |
| `2^52` | `1` | `2` | `-1.5` |
| `2^54` | `4` | `0` | `-2` |
| `2^60` | `256` | `0` | `0` |

From `2^54` the spacing exceeds pi, both shifted values round back to
`theta`, the two evaluations bind the same circuit, and the helper returns
exactly 0 with a nil error. (Amended 2026-09-10, round 1 fix: strictly,
above `2^54`; at exactly `2^54` the minus shift still lands in
`[2^53, 2^54)`, where the spacing is `2`, and moves by `-2`, as the table
row above and its `-0.0797` observation below already show. Both round
back only from `theta` in `(2^54, 2^55)` on.) Observed at `0712b96` on
`gradientTargetTemplate` (declares `a` then `b`, H2 Hamiltonian) with the
prototype of this design's tests run against the unchanged helper:

| `params` | `names` | Today | Contract |
|---|---|---|---|
| `{a: 2^60, b: 0.1}` | `[a]` | `{a: 0}`, 2 evaluations, nil error | `InvalidVQEInputError`, 0 evaluations |
| `{a: 2^54, b: 0.1}` | `[a]` | `{a: -0.0797}`, 2 evaluations, nil error: the rule applied at offsets `+0` and `-2` | `InvalidVQEInputError`, 0 evaluations |
| `{a: 2^26 + 1 ulp, b: 0.1}` | `[a]` | `{a: 0.1625}`, a usable slope | `InvalidVQEInputError`, 0 evaluations (the bound is conservative; Decision 2) |
| `{a: 2^26, b: 0.1}` | `[a]` | a slope exact to `1e-8` | unchanged |
| `{a: 2^60, b: 0.1}` | `[b]` | the correct slope of `b` at that `a` | unchanged |

Reachable through `VQE`: `InitialParams{"a": 2^60, "b": 0.1}` passes the
finiteness guard, the gradient of `a` is 0 at every iteration, and the
step `a - 0.3 * 0` leaves it at `2^60`; the run optimizes `b` alone and
reports `Converged` after 123 accepted iterations with `a` frozen at its
start. On `H2Ansatz` with `theta = 2^60` the run is one no-op iteration,
`Converged`, 4 evaluations. Below `2^54` the descent step is lost before
the shift is: at `2^48` the spacing is `0.0625`, so the default `0.3`
step times a gradient of `0.1` rounds to no change, and `VQE` reports
`Converged` at the start with a non-zero gradient it could not apply.

The item's contract is open: "either an error for a parameter whose shift
is not representable, or a documented reduction of angles into a
representable range before shifting". This design chooses the error.

## Classification

**Standard.** Behavior change inside `algorithm/vqe.go` within the existing
driver structure: one unexported constant, one pre-loop check in the
helper, one clause in `VQE`'s `InitialParams` check, one clause in its
step guard. No signature, type, or cross-package change. Short design doc
plus a one-task plan.

## Decision 1: an error, not a reduction mod 2*pi

Four reasons, any one sufficient.

**Periodicity is a precondition the helper cannot check.** The energy is
`2*pi`-periodic in an angle only when the angle enters its gate as
`exp(-i*theta*P/2)`, the convention item 13 documented on
`parameterShiftGradient` (second paragraph of its doc comment) and that
`VQE` cannot verify because a `parameterized.Factory` is opaque. A
factory of another period (`gates.NewRz(value/2)`, period `4*pi`; a
closure over `value*value`, no period) would have its point silently
moved by a reduction: `E(theta mod 2*pi)` is not `E(theta)` for it. That
is the same class of silent failure item 13 documented for a rescaling
factory, introduced this time by the helper itself. An error cannot
produce a wrong number.

Amended 2026-09-10 (round 1 fix): this reason is defense in depth rather
than the load-bearing one. The parameter-shift rule itself is exact only
under the same `exp(-i*theta*P/2)` precondition (`E = a + b*cos(theta) +
c*sin(theta)`, which is `2*pi`-periodic); a factory of another period
already yields a wrong gradient from this helper before any reduction is
considered, so periodicity failure coincides with the rule's own
precondition failing rather than being introduced fresh by a reduction.
The reason still stands: an error cannot compound an already-wrong number
into a differently-wrong one, and the decision does not rest on it alone
(reasons 2 and 4 are each independently sufficient).

**A reduction inside the helper does not fix `VQE`.** With the gradient
correct at `a = 2^60`, the descent computes `a - step * grad` and float64
loses a step of `0.03` to a spacing of `256`: `a` stays frozen and the
run still reports `Converged` with a non-zero gradient. Closing the
item's `VQE` symptom by reduction would need `VQE` to rewrite the
parameter and report `4.1219` in `VQEResult.Params` where the caller
supplied `2^60`. No field of `VQEResult` is currently anything but what
the run computed from what the caller gave, and the package has no
precedent for normalizing an input.

**The reduction is not computable in float64 at those magnitudes.**
`math.Mod(2^60, 2*math.Pi)` returns `5.0824`; the true reduction is
`4.1219`, the value the red test computes with a 256-bit `big.Float`. The
float64 constant `2*pi` carries an error of about `4e-16`, multiplied by a
quotient of `1.8e17`: the result is off by about 80 radians. A correct
reduction needs `math/big` or a Payne-Hanek table in the driver, for an
input no caller has.

Amended 2026-09-10 (round 1 fix): the error figures in the previous
paragraph were wrong. Measured against a 512-bit `big.Float` value of
`2*pi`: the float64 value of `2*math.Pi` is short by
`2.4492935982947064e-16` (twice `math.Pi`'s own error of
`1.2246467991473532e-16`), the quotient `2^60 / (2*math.Pi)` is
`1.834931564551251e17`, and the accumulated error, quotient times
per-term error, is `44.94` rad, not `80`. `5.0824 - 4.1219 = 0.9605` rad,
which is `44.94 mod 2*pi`, so the two reduced values already quoted are
correct; only the `4e-16` and `80` figures were not. The conclusion is
unaffected: `45` rad of accumulated error is just as disqualifying as
`80` for computing the reduction in float64.

**An error is what the package does with an input outside its domain.**
Item 14 (`docs/superpowers/specs/2026-09-08-vqe-input-validation-design.md`)
made every option and every initial parameter subject to a domain check
before the first evaluation, reported as `InvalidVQEInputError` with a
Reason naming the field and the value. A magnitude bound is one more such
check beside the finiteness clause, checked in the same place, with a
Reason that says what the bound is and why. A caller with a legitimately
huge angle reduces it at the precision they can vouch for, knowing their
factory's period.

Reduction was also weighed in a milder form, reducing only inside the
helper for the shifted copies and leaving `params` untouched. Rejected
for the first three reasons above (the second applies unchanged: `VQE`
would still freeze).

## Decision 2: the bound is 2^26, applied to |theta|

The rule's shifted angle is exact to `ulp(theta)/2`. A gradient
component is off by at most that times the largest slope of `E`, which
the Hamiltonian's coefficient sum bounds (every Pauli expectation is
bounded by 1). Any bound is therefore a choice of how much shift error to
tolerate against how much angle to allow; the two extremes are far apart
(the rule fails outright only from `2^54`; angles a caller means are of
order `2*pi`), so the choice is stated with its derivation rather than
presented as forced.

**Chosen: `|theta| <= 2^26 = 67,108,864`.** Below it the spacing is at
most `2^-27 = 1.49e-8`, the shifted angle is within `7.45e-9` rad of
`theta +/- pi/2`, and a gradient component is exact to `7.45e-9` per unit
of coefficient sum. `2^26` is the even split of the 53-bit significand:
26 bits before the binary point and 27 after, the square root of `2^53`
rounded down to a power of two. That is the largest power of two at
which the parameter keeps a resolution finer than `1e-8` rad, and it
allows angles of more than `10^7` turns, far beyond any angle a descent
reaches from an angle a caller chose: 200 iterations of a `0.3` step
times a unit gradient walk 60 rad.

Amended 2026-09-10 (round 1 fix): `2^-27 = 7.45e-9`, not `1.49e-8`; the
two figures in the previous paragraph's first sentence contradicted each
other. The correct chain: for `|theta| <= 2^26`, `theta +/- pi/2` crosses
into `[2^26, 2^27)` for any `theta` within `pi/2` of the bound, so it is
the shifted value's spacing that bounds the offset error, and that
spacing is at most `2^-26 = 1.49e-8`. The offset error, half that
spacing, is `7.45e-9`, as stated; only the intermediate `2^-27` label was
wrong, not the `7.45e-9` conclusion or anything downstream of it
(`algorithm/vqe.go`'s `maxShiftMagnitude` comment carries the same fix).

**Why not the exact-defect threshold** (`theta + pi/2 == theta`, from
`2^54`). It leaves the wrong-offset regime (`2^51` to `2^54`, shifts of
`1.5` and `2`) and the frozen-step regime (from `2^48` with default
options) unaddressed, so `VQE` would still report `Converged` with a
frozen parameter for `2^50`, against the dispatch's requirement. (Amended
2026-09-10, round 2 fix: strictly, above `2^48`; at exactly `2^48` the
default step of `0.3` times a gradient of `0.1` still moves the parameter
by one ulp in the descent direction (`2^48 - 0.03` rounds to
`2^48 - 0.03125`, verified in a scratch program), because that value and
`2^48` are both representable in the finer, `[2^47, 2^48)` spacing of
`0.03125`. Freezing in both directions holds only from the next
representable float above `2^48` on; the smallest power of two at which
it holds is `2^49`. `algorithm/vqe.go`'s `maxShiftMagnitude` comment
carries the same fix.)

**Why not a tighter bound** (`2^10`, where the shift error is comparable
to the energy's own rounding). A run from an ordinary start with a large
`StepSize` and many iterations can walk past `1024` rad legitimately, and
the step guard (Decision 3) would then stop it mid-run.

**Why not a bound derived from the options** (the smallest step that
still moves the parameter, `StepSize`-dependent). The helper has no
options; a bound the helper and `VQE` share must be a constant.

The constant is unexported (`maxShiftMagnitude = 1 << 26` in
`algorithm/vqe.go`) with a doc comment that carries this derivation in
short: the spacing argument, the three failure magnitudes (`2^48`,
`2^51`, `2^54`), the even-split rationale, the headroom. The Reasons print
it as `2^26 (6.7108864e+07)` so a caller sees both the power and the
value they can compare against.

## Decision 3: where the bound is checked, and by what error

Three places, mirroring items 14 and 16: the helper checks for its own
callers, `VQE` guarantees the invariant by construction so the helper's
check is unreachable from it.

**`parameterShiftGradient`, ahead of its loop, after the item 16
completeness check.** For each name in `names`, in the order given: if
the bound value is finite and its magnitude exceeds `maxShiftMagnitude`,
return `nil, 0, &InvalidVQEInputError{Reason: ...}`. Before any
evaluation, so the count is the literal `0` (item 17's rule: nothing
completed). Only names that are shifted are checked: an unshifted value
enters both evaluations of every other name identically, so its
magnitude cannot affect the difference (the Context table's last row
pins this; Go's `math.Sin`/`math.Cos` reduce large arguments exactly, so
the evaluation itself is deterministic). Only finite values are checked:
a non-finite value is left to `Bind`, exactly as item 16's Decision 5
states, so every sentence of that decision about where a non-finite
value is reported holds as written; the check is
`isFinite(v) && math.Abs(v) > maxShiftMagnitude`. `names` order rather
than declaration order because this is the helper's own precondition
with no `Bind` parity to keep, and `names` order is the order the loop
would have hit the values.

The error is `*InvalidVQEInputError` with `Err` nil. The helper's
convention (item 16, Decision 2) is to return the detecting package's
error unchanged; here the detecting package is `algorithm` and the
condition is a VQE-driver precondition on an input value, which is what
that type is for. `parameterized.InvalidParameterValueError` was
considered and rejected: its message says "non-finite value", which would
be false. A new error type was rejected as in item 16: no convention for
helper-only errors, and the condition is the same one `VQE` reports up
front.

**`VQE`, inline in the `InitialParams` loop, after the finiteness
clause.** `math.Abs(value) > maxShiftMagnitude` returns
`InvalidVQEInputError` before the first evaluation, in the same loop and
style as the undeclared-name and non-finite checks. This is what closes
the item's `VQE` symptom: a huge finite initial parameter is rejected,
never frozen.

**`VQE`, in the step guard, after the overflow clause.** A descent that
carries a parameter past the bound would otherwise hand the helper an
angle it rejects, and the wrap would report
`gradient evaluation at iteration N failed: invalid VQE input: parameter "a" cannot be shifted ...`,
an `InvalidVQEInputError` wrapping another. `VQE` instead checks each
stepped value against the bound where it already checks it for overflow
and stops with its own Reason. The Reason names the step, the iteration,
the bound, and the step arithmetic, not `StepSize`: a start `0.125` below
the bound crosses it at the default step size, so the field is not at
fault the way it is for overflow. With this clause `VQE` never hands the
helper an angle beyond the bound, as it never hands `Bind` a non-finite
value; the helper's check is unreachable from `VQE`, as items 16 and 17's
were. The check fires before the step is evaluated, so a step that would
have been reverted for raising the energy is still rejected, the same
order the overflow clause already has.

## Decision 4: Reason strings

| Condition | Where | Reason |
|---|---|---|
| shifted angle beyond the bound (helper) | `parameterShiftGradient` | `parameter "a" cannot be shifted by +/- pi/2: its magnitude must be at most 2^26 (6.7108864e+07), got 1.152921504606847e+18` |
| initial parameter beyond the bound | `VQE`, up front | `initial parameter "theta" must have magnitude at most 2^26 (6.7108864e+07), beyond which float64 cannot resolve the +/- pi/2 parameter shift, got 1.152921504606847e+18` |
| step beyond the bound | `VQE`, step guard | `the step of parameter "theta" reaches 6.710886403417353e+07 at iteration 0, beyond the 2^26 (6.7108864e+07) within which float64 resolves the +/- pi/2 parameter shift (step size 0.3 times gradient -0.5305784211229922)` |

Every message names the parameter, the bound in both forms, and the
value; the two `VQE` messages say what the bound protects so a caller
reading only the error knows to reduce the angle rather than look for a
bug in their template. `Err` is nil in all three: each is detected by the
package itself. The two `VQE` rows are added to item 14's Decision 3
table by amendment.

## Rulings on edge cases

- **Exactly `2^26` and `-2^26` are accepted** (`>` not `>=`), in the
  helper and in `VQE`. Pinned against the analytic slope: on `H2Ansatz`
  against `Z0`, `E = cos(theta)`, and the shift at `|theta| = 2^26`
  reproduces `-sin(theta)` to `1e-8`.
- **One ulp past the bound is rejected**, in both signs, although the
  slope there is still usable (Context table). The bound is a stated
  constant, not a per-value test of the shift's accuracy.
- **A huge unshifted value is not an error** for the helper (`names`
  omits it). `VQE` shifts every declared name, so it never has one.
- **Non-finite values are `Bind`'s**, as before (Decision 3). `+Inf`
  shifted returns `parameterized.InvalidParameterValueError` from `Bind`
  with 0 evaluations; `NaN` likewise (`math.Abs(NaN) > bound` is false,
  and `isFinite` excludes it anyway).
- **Several offenders**: the helper reports the first in `names` order;
  `VQE` reports whichever `InitialParams` map iteration reaches first,
  the pre-existing behavior of that loop (item 14's ruling).
- **Undeclared name in `names`, absent from `params`**: reads as 0,
  passes the bound, and is rejected by `Bind` when its own shift is
  evaluated, as item 16's Decision 5 states. Unchanged.
- **Undeclared name in `names`, present in `params` with a huge value**
  (round 1 fix, 2026-09-10): Decision 3's magnitude check runs over every
  name in `names`, in order, ahead of either loop; nothing in its text
  restricts it to names the template declares, and it cannot: it runs
  before `Bind` is ever called, so it has no way to know which names are
  declared. So this case is caught there, at 0 evaluations, not by
  `Bind`'s `UnknownParameterError` after the preceding names' evaluations
  that item 16's Decision 5 describes for the general undeclared-name
  case; that description now holds only when the undeclared name's bound
  value (if any) is within `maxShiftMagnitude`. Round 1's black-box red
  testing (`.superpowers/backlog/enhancement-backlog-2026-08-27/item-20-round-1-red.md`,
  survivor 1) found the doc comment and this ruling silent on the
  interaction; `algorithm/vqe.go`'s `parameterShiftGradient` doc comment
  now states the ordering explicitly, and
  `docs/superpowers/specs/2026-09-08-parameter-shift-missing-param-design.md`
  carries the same amendment. (Amended 2026-09-10, round 2 fix: precisely,
  finite and within `maxShiftMagnitude`. A bound value of `+Inf` or
  `-Inf` is not finite, so the magnitude check's `isFinite` guard skips it
  regardless of its magnitude, and `Bind`'s `UnknownParameterError` still
  applies after the preceding names' evaluations, exactly as the general
  undeclared-name case states. The missing-param spec's parallel note and
  `algorithm/vqe.go`'s doc comment carry the same fix.)
- **Empty `names`**: no name is checked, no evaluation runs. Unchanged.
- **A step that crosses the bound and would have been reverted** is
  rejected, not reverted: the guard precedes the step evaluation, as the
  overflow clause does.
- **A run that starts within the bound and converges within it** is
  unaffected, including the QAOA demo and every existing test: the walk
  of any such run is bounded by `MaxIterations * StepSize * |grad|`.

## Red test ruling

`TestRedParameterShiftAtHugeAngleMatchesReducedAngle` in
`algorithm/backlog_red_numeric_test.go` asserts that
`parameterShiftGradient` at `a = 2^60` returns, after exactly 2
evaluations and with a nil error, the slope at `a` reduced mod `2*pi`
(computed at 256-bit precision). **Ruled contradicted by this design,
Decision 1**: the contract for a shifted angle beyond `2^26` is
`InvalidVQEInputError` with 0 evaluations and a nil gradient, so the
test's `err != nil` is fatal under the contract and its `evals != 2`
assertion can never hold. The test is deleted, not moved and not edited
(editing a red test's assertions to the new contract would leave a test
named for the reduction it no longer asserts). Its `reduceMod2Pi` helper
and its `math/big` import go with it; nothing in the regular suite
reduces an angle, by Decision 1. The file then holds no test and is
deleted, as `algorithm/backlog_red_test.go` was under item 17
(`docs/superpowers/specs/2026-09-08-parameter-shift-evaluation-count-design.md`,
"Effect on the red-test file"). `go vet -tags redtests ./algorithm`
still passes with no file in the package carrying the tag, and
`go test -tags redtests ./algorithm -run '^TestRed'` reports
`[no tests to run]`.

What the red test was right about is kept: the helper must not return
`{a: 0}` with a nil error at `2^60`, and `VQE` must not report
`Converged` with `a` frozen there. Both are pinned by the replacement
tests in Testing, which assert the error instead of the reduced slope.

## Effect on callers

- `algorithm/vqe.go` `VQE`: two new clauses, both on inputs no existing
  caller supplies. `algorithm/vqe_driver_test.go`,
  `algorithm/vqe_bench_test.go`, and every test in `algorithm/vqe_test.go`
  pass unchanged. The prototype's `go run ./cmd/quantum -demo qaoa` still
  prints `VQE from the symmetric start: energy = -1.0000, expected cut = 2.0000`
  followed by `  iterations accepted: 29, energy evaluations: 378`.
- Tests calling the helper with angles of order 1: one extra pass of
  `len(names)` map lookups per call; unaffected.
- `internal/examples/qaoa.go` and `cmd/quantum/main.go` call `VQE` only,
  with angles from a landscape scan over `[0, pi]`.
- No Go file outside `algorithm/` changes. `CHANGELOG.md` and four prior
  specs gain text (below).

## Backward compatibility (ADR-style note)

Inputs that were accepted and now error: an `InitialParams` value of
magnitude beyond `2^26`, and a run whose descent carries a parameter past
that bound. Internally, `parameterShiftGradient` called with a shifted
name bound beyond `2^26` now errors before any evaluation. Per
`docs/compatibility-policy.md` this is a personal project with no
external consumers; every internal caller is listed above and unaffected.

- **CHANGELOG:** following items 18 and 19, an Unreleased entry under the
  existing `### Fixed` heading, at the end of that section (entries are
  appended in landing order), saying what was wrong, what the bound is,
  why an error and not a reduction, and that the `algorithm` red-test
  file is removed.
- **ADR-0010:** no amendment. Parameter validation at the driver is a
  consequence it already records; the bound is one more domain check and
  changes no decision.
- **Prior specs, dated amendment notes** in the style of the existing
  notes in `docs/superpowers/specs/2026-08-30-parameter-binding-vqe-design.md`:
  - `2026-09-08-vqe-input-validation-design.md`: the step guard paragraph
    (second clause), the "Order inside the loop" paragraph (the helper
    never sees an angle beyond the bound), two Decision 3 rows, and the
    "Helper shape and the items that follow" paragraph (item 20 landed;
    this time `VQE` changes, in the two places that paragraph's shape
    anticipated; the red test was deleted).
  - `2026-09-08-parameter-shift-missing-param-design.md`: the round 1
    bullet that routed the huge-angle test to item 20 "kept under the
    redtests tag" (now deleted, with the citation above), and Decision 5
    ("completeness only" is a statement about `Bind`'s rules; the helper
    now has a precondition of its own; non-finite values still `Bind`'s).
  - `2026-09-08-parameter-shift-evaluation-count-design.md`: the round 1
    note that item 20's tagged file makes the red run list one test (the
    file is gone; `[no tests to run]` holds again), and Decision 5's
    quoted last sentence ("The completeness and magnitude checks above
    return 0 because they precede the first evaluation").
  - `2026-08-30-parameter-binding-vqe-design.md`: one bullet in the
    `VQE` Errors list recording the bound and why angles are not reduced.

## Testing

All in `algorithm/vqe_test.go` (package `algorithm`, which can call the
helper and read the constant):

- `TestParameterShiftRejectsUnresolvableAngle`: pins that the constant is
  `2^26`; five rejected magnitudes (`2^60`, `-2^60`, `2^54`, one ulp past
  the bound in each sign) each give `InvalidVQEInputError` with the exact
  Reason, nil gradient, 0 evaluations, nil `Unwrap`; with both names
  beyond the bound and `names = [b, a]` the Reason names `b`; at exactly
  `+/- 2^26` on `zOnQubit0(1)` and `H2Ansatz` the gradient equals
  `-sin(theta)` to `1e-8` after 2 evaluations; with `a = 2^60` unshifted
  the gradient of `b` matches a finite difference at that `a`; with
  `a = +Inf` shifted the error is `Bind`'s `InvalidParameterValueError`
  for `a` after 0 evaluations. Fails today on the five rejected rows and
  the ordering row (a slope with a nil error each time).
- `TestVQEOptionValidation`: two rows added (`2^60`, and one ulp past
  `-2^26`), asserting `InvalidVQEInputError`. Fail today with a
  `Converged` result.
- `TestVQEOptionErrorNamesTheField`: one row added with the exact
  `InitialParams` Reason. Fails today with a nil error.
- `TestVQEHugeInitialParamIsRejected`: the item's own scenario
  (`gradientTargetTemplate`, `a = 2^60`, `b = 0.1`) is
  `InvalidVQEInputError` with the exact Reason and nil `Unwrap`, not a
  `Converged` result; `a = 2^26` with `MaxIterations: 3` runs and
  evaluates its gradient. Fails today with `Converged` after 123
  iterations.
- `TestVQEStepBeyondShiftBoundIsReported`: `zOnQubit0(1)`, `H2Ansatz`,
  `theta = 2^26 - 0.125` (slope `-0.53`, so the default step moves
  `theta` up by `0.16` and crosses on iteration 0): the exact step-guard
  Reason with the stepped value and gradient read back, nil `Unwrap`.
  Fails today with an unconverged result after 3 iterations.
- `go test -tags redtests ./algorithm -run '^TestRed'` reports
  `[no tests to run]`. The `parameterized` red run's outcome depends on
  what other items have landed and is reported, not asserted.

Prototype: every code, test, and documentation change in the plan was
applied to a scratch copy of the repository at `0712b96` in the plan's
order; the red results above were observed with the tests compiled
against the original helper and driver plus the bare constant, every test
passed after the four edits, and `gofmt -l`, `go vet` (plain and
`-tags redtests` on `algorithm` and `parameterized`), `staticcheck`,
`go build`, `go test ./...`, `go test -race ./algorithm`, the red-tag
run, and the QAOA demo were all as stated.

## Out of scope

Reducing angles mod `2*pi` anywhere. Making the bound configurable or
deriving it from the options. Checking unshifted values in the helper.
Changing `VQEResult` or any exported signature. Item 21 (copy aliasing in
`parameterized`). Any change to `evaluate`, `checkEnergy`, or the
gradient guard.
