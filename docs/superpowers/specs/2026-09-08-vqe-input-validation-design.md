# VQE Input Validation and Error Taxonomy — Design

Date: 2026-09-08
Status: Approved (backlog item text plus run-lead dispatch; no brainstorming gate)
Resolves: backlog item 14 (`.superpowers/backlog/enhancement-backlog-2026-08-27/item-14.md`)

## Context

`VQE` in `algorithm/vqe.go` validates only nil inputs, the
one-gate-per-parameter rule, and undeclared initial parameter names.
Everything else either runs or fails with another package's error type.
Observed today (`go test -tags redtests ./algorithm -run '^TestRedVQE(Option|Structural|NonFinite)' -v`):

| Input | Today | Contract |
|---|---|---|
| `StepSize: -0.3` | climbs once, then the 1e-6 floor clamps it positive; returns unconverged after 199 iterations | `InvalidVQEInputError` |
| `StepSize: NaN` or `+Inf` | `parameterized.InvalidParameterValueError` naming `theta` from the first stepped Bind | `InvalidVQEInputError` |
| `MaxIterations: -1` | returns unconverged after zero iterations | `InvalidVQEInputError` |
| `Tolerance: -1` or `NaN` | can never converge; runs to `MaxIterations` | `InvalidVQEInputError` |
| `InitialParams{"theta": NaN}` or `-Inf` | `parameterized.InvalidParameterValueError` | `InvalidVQEInputError` |
| Hamiltonian term with 1 or 3 axes on a 2-qubit template | `quantum.IncompatibleQubitCountError` from the first evaluation | `InvalidVQEInputError` |
| factory returning a 2-qubit gate on 1 target | `quantum.InvalidGateApplicationError` from the first evaluation | `InvalidVQEInputError` |
| coefficient `NaN` or `+Inf` | NaN gradient, NaN step, then `InvalidParameterValueError` blaming `theta`, which the caller supplied finite | `InvalidVQEInputError`; no error names a finite-on-entry parameter |

The item's contract: out-of-domain options, Hamiltonian/template
structural mismatches, and non-finite Hamiltonian coefficients are
`InvalidVQEInputError`, and no error attributes the failure to a parameter
that was finite on entry.

## Classification

**Standard.** Behavior change inside `algorithm/vqe.go` within the existing
driver structure; one exported type gains a field and a method; no new
package, no cross-package change.

## Decision 1: what is checked up front, what is caught at evaluation

The dividing line is whether the driver can see the problem without
running the caller's code. Options, initial parameters, and the
Hamiltonian's terms are plain data the driver can inspect; a
`parameterized.Factory` is an opaque `func(float64) quantum.Gate`, so
what it returns is only known once it runs.

**Up front, before the first evaluation** (two unexported helpers called
from `VQE` right after the nil checks):

- `validateVQEOptions(opts VQEOptions) error`
  - `StepSize` must be finite and `>= 0`; zero keeps meaning "default 0.3".
    A negative step ascends. NaN or Inf turns the first update into NaN
    arithmetic.
  - `MaxIterations` must be `>= 0`; zero keeps meaning "default 200".
  - `Tolerance` must be finite and `>= 0`; zero keeps meaning "default
    1e-10". Negative or NaN can never be met. `+Inf` is rejected too
    (ruling below).
- `validateVQEStructure(h *Hamiltonian, t *parameterized.Template) error`
  - the existing one-gate-per-parameter check, moved here unchanged (it is
    the template-side structural precondition and belongs with the others);
  - every term's coefficient must be finite;
  - every non-identity term's Pauli string must have exactly
    `t.NumQubits()` axes. Identity terms (no axes) are exempt because
    `Energy` adds their coefficient without consulting the state.
- `InitialParams` stays inline in `VQE` (it needs the declared names): the
  existing undeclared-name check plus a new finiteness check.

**At evaluation time, wrapped:** every error returned by `evaluate` or
`parameterShiftGradient` inside `VQE` passes through
`wrapEvaluationError(where string, err error) error`, which returns
`&InvalidVQEInputError{Reason: where + " failed: " + err.Error(), Err: err}`.
Reasoning: once the up-front checks pass, the driver hands every
evaluation a complete, declared, finite parameter set, so the only
remaining failure sources are properties of the caller's template or
Hamiltonian that only running them reveals: a factory returning a gate of
the wrong width (`quantum.InvalidGateApplicationError` from
`circuit.AddGate`), a factory returning a malformed matrix
(`quantum.InvalidGateMatrixError`), a Pauli axis outside the enum
(`quantum.InvalidPauliAxisError` from `quantum.Expectation`), or a
zero-qubit template (`quantum.InvalidQubitCountError` from `circuit.New`).
All are input problems, so all are reported as the VQE type. `where` is
one of `"initial energy evaluation"`, `"gradient evaluation at iteration
N"`, `"step evaluation at iteration N"`.

**Gradient guard.** After `parameterShiftGradient` returns, `VQE` checks
each component is finite and otherwise fails with
`InvalidVQEInputError{Reason: "gradient of parameter %q is non-finite (%v) at iteration %d: the Hamiltonian's energy overflows float64"}`.
This closes the one path the up-front coefficient check leaves open:
coefficients that are each finite but whose sum overflows float64 (for
example two `math.MaxFloat64` terms). Verified on the current code: the
energy is `+Inf` at the start and at one shifted point, the gradient is
`-Inf`, the step carries `theta` to `+Inf`, and `Bind` blames `theta`.
The contract says no error may do that, so the driver stops at the
gradient. Coefficients are finite and every Pauli expectation is bounded
by 1, so a non-finite gradient can only mean overflow, which the message
says.

## Decision 2: wrapping preserves the cause

`InvalidVQEInputError` gains `Err error` and `func (e *InvalidVQEInputError) Unwrap() error`.
`Error()` is unchanged (`"invalid VQE input: " + Reason`); `Reason` already
embeds the cause's text, so nothing is printed twice. `Err` is nil for
every problem `VQE` detects itself and non-nil only for wrapped evaluation
errors. This keeps `errors.As(err, &quantum.InvalidGateApplicationError)`
working for any caller that matched the old type, and follows the
`fmt.Errorf("...: %w", err)` contract the module already gives callers in
`gates/matrix.go` and `circuit/parallel.go`. No typed error in the module
had an `Unwrap` before; this is the first, and the doc comment says why.

## Decision 3: Reason strings

| Condition | Reason |
|---|---|
| `StepSize` out of domain | `StepSize must be finite and positive (zero selects the default 0.3), got -0.3` |
| `MaxIterations` negative | `MaxIterations must not be negative (zero selects the default 200), got -1` |
| `Tolerance` out of domain | `Tolerance must be finite and positive (zero selects the default 1e-10), got NaN` |
| initial parameter non-finite | `initial parameter "theta" has non-finite value -Inf` |
| initial parameter undeclared | unchanged: `initial parameter "nope" is not declared in the template` |
| parameter drives several gates | unchanged: `parameter "theta" drives 2 template steps; the parameter-shift gradient requires exactly one gate per parameter` |
| coefficient non-finite | `Hamiltonian term 1 has non-finite coefficient NaN` |
| Pauli string length mismatch | `Hamiltonian term 0 has 1 Pauli axes but the template has 2 qubits` |
| evaluation failure | `initial energy evaluation failed: gate CNOT requires 2 qubits but got 1` |
| gradient overflow | `gradient of parameter "theta" is non-finite (-Inf) at iteration 0: the Hamiltonian's energy overflows float64` |

Every message names the field, term index, or parameter, and the offending
value, so the caller can go straight to the literal. Option messages say
that zero would have selected the default because zero is the sentinel and
"must be positive" alone would read as forbidding it.

## Rulings on edge cases

- **Axis validity is not duplicated.** A `quantum.PauliAxis` outside
  `PauliI..PauliZ` is left to `quantum.Expectation`, which owns that domain
  and reports `InvalidPauliAxisError`; `VQE` wraps it at the first
  evaluation. Copying the enum's range into `algorithm` would couple two
  packages on a detail one of them already enforces.
- **`Tolerance: +Inf` is rejected**, although today it "converges" after
  the first accepted step. It is not a usable threshold, the rule "finite
  and non-negative, zero for the default" is one sentence for both float
  fields, and no caller uses it.
- **Zero-qubit template** (`parameterized.NewTemplate(0)`) is not checked
  up front; `circuit.New` rejects it inside the first `Bind`, and the wrap
  reports it as `InvalidVQEInputError`. The Pauli-length check compares
  against `t.NumQubits()`, which is 0 there, so a Hamiltonian with only
  identity terms would pass that check and fail at the wrap. Acceptable:
  the caller still sees the VQE type.
- **Several bad initial parameters.** Which one is named depends on map
  iteration order. Pre-existing for the undeclared-name check; unchanged.

## Helper shape and the items that follow

`validateVQEStructure` is the extension point for **item 15** (a parameter
named `""` escapes `ParamStepCounts`): its template loop is where an
empty-name check belongs, and the plan does not add one. **Items 16 and
17** live in `parameterShiftGradient` (missing-parameter shift from an
implicit zero; evaluation undercount on failure) and are untouched: the
wrap in `VQE` passes the gradient's error and count straight through, so a
later fix to the count needs no change here. Their red tests stay under
the `redtests` tag; the prototype of this design leaves all three still
failing.

## Effect on callers

- `internal/examples/qaoa.go` calls `VQE` with `InitialParams` from a
  landscape scan and `MaxIterations: 100`: all finite, in domain. The
  `MaxCutHamiltonian` terms are `numQubits` axes long by construction and
  `QAOATemplate` declares each parameter once. The prototype's
  `go run ./cmd/quantum -demo qaoa` still prints the VQE result (energy
  -1.0000, cut 2.0000).
- `algorithm/h2.go` (`H2Hamiltonian`, `H2Ansatz`): every term is two axes or
  identity; `TestVQEConvergesToH2GroundState` and the rest of
  `vqe_driver_test.go` pass unchanged, including
  `TestVQERejectsMultiGateParameter`, whose message text is preserved.
- `cmd/quantum/main.go` dispatches to the demos and does not call `VQE`.
- No file outside `algorithm/` changes.

## Backward compatibility (ADR-style note)

Inputs that were accepted and now error: negative `StepSize`, negative
`MaxIterations`, negative or non-finite `Tolerance` (including `+Inf`).
Inputs that already errored but change type: non-finite `StepSize` or
`InitialParams` (was `parameterized.InvalidParameterValueError`, now
`InvalidVQEInputError` with `Err == nil`); Pauli-length mismatches (was
`quantum.IncompatibleQubitCountError` from the loop, now
`InvalidVQEInputError` with `Err == nil`); factory width errors (was
`quantum.InvalidGateApplicationError`, now `InvalidVQEInputError` wrapping
it, so `errors.As` on the old type still matches). Per
`docs/compatibility-policy.md` this is a personal project with no external
consumers and hard changes are acceptable; every internal caller is listed
above and unaffected. No new ADR: the decision is local to one driver and
recorded here.

## Red test rulings

The three red tests move into `algorithm/vqe_test.go` renamed
`TestVQEOptionValidation`, `TestVQEStructuralMismatchIsInvalidInput`, and
`TestVQENonFiniteHamiltonianIsNotBlamedOnParams`, assertions unchanged.
None is stricter than the contract. The third is weaker than the contract
(it asserts only "an error, and not a parameter blame"); a new test pins
the type and the Reason text alongside it rather than editing the moved
test.

## Testing

- Moved red tests as above.
- `TestVQEOptionErrorNamesTheField`: exact Reason per option field.
- `TestVQEStructuralErrorReasons`: exact Reason for length mismatch and NaN
  coefficient, and `errors.Unwrap` is nil for up-front detection.
- `TestVQEEvaluationErrorKeepsItsCause`: the wide-factory case is
  `InvalidVQEInputError` and `errors.As` reaches
  `quantum.InvalidGateApplicationError`; Reason carries the phase and the
  cause; a bad axis reaches `quantum.InvalidPauliAxisError`.
- `TestVQEOverflowingHamiltonianIsNotBlamedOnParams`: two `MaxFloat64`
  terms give `InvalidVQEInputError` naming the gradient, never
  `InvalidParameterValueError`.

## Out of scope

Items 15, 16, 17. Deterministic ordering of initial-parameter errors.
Validating `Hamiltonian` at `AddTerm` time (its doc says it delegates to
the caller; a one-line doc addition points that caller at `VQE`).
