# Zero-Value `Template` — Design

Date: 2026-09-09
Status: Approved (backlog item text plus run-lead dispatch; no brainstorming gate).
Resolves: backlog item 18 (`.superpowers/backlog/enhancement-backlog-2026-08-27/item-18.md`)

## Context

`parameterized.Template` tracks declared parameter names twice: `paramOrder`
(a slice, first-use order) and `seen` (a `map[string]bool` for the O(1)
membership test that `AddParamGate` and `Bind` both use). `NewTemplate`
allocates `seen`; nothing else does. `AddParamGate` writes
`t.seen[name] = true` unconditionally (`parameterized/parameterized.go:111-114`),
so on a `Template` declared as a plain variable the first declaration
panics with `assignment to entry in nil map`. Every other method is safe
on the zero value already: `NumQubits`, `ParamNames`, and
`ParamStepCounts` only read fields whose zero values are meaningful,
`AddGate` never touches `seen`, `checkTargets` rejects every target as out
of range because `numQubits` is 0, and `Bind` only *reads* `seen` (a nil
map read yields `false`) before `circuit.New(0)` rejects the qubit count.
Observed today (`go test -tags redtests ./parameterized -run '^TestRedZeroValueTemplate' -v`):

| Call on `var tmpl parameterized.Template` | Today | Contract |
|---|---|---|
| `tmpl.AddParamGate("", parameterized.Ry)` (no targets) | panic: `assignment to entry in nil map` | an error or success, never a panic |
| `tmpl.AddParamGate("theta", parameterized.Ry, 0)` | `QubitsOutOfRangeError{Index: 0, MaxIndex: -1}` | unchanged |
| `tmpl.AddGate(gates.NewHadamard(), 0)` | `QubitsOutOfRangeError{Index: 0, MaxIndex: -1}` | unchanged |
| `tmpl.NumQubits()`, `tmpl.ParamNames()`, `tmpl.ParamStepCounts()` | `0`, `[]`, `map[]` | unchanged |
| `tmpl.Bind(Params{})` | `InvalidQubitCountError{Requested: 0}` from `circuit.New` | unchanged |
| `tmpl.Bind(Params{"theta": 0.1})` | `UnknownParameterError{Name: "theta"}` | unchanged |

The item offers two closures: "either `NewTemplate` becomes a documented
requirement enforced by every method, or a zero-value `Template` is a
valid, safely usable state." Nothing in the module declares a `Template`
without `NewTemplate` (`algorithm/h2.go`, `algorithm/qaoa.go`, every test,
and `internal/examples/qaoa.go` all go through the constructor), so the
defect is reachable only by a caller outside the module.

## Classification

**Standard.** One-branch behavior change inside `parameterized.Template.AddParamGate`
plus a one-line simplification of `NewTemplate`; no signature, type, or
cross-package change. Short design doc plus a one-task plan.

## Decision 1: the zero value is a valid, usable `Template`

**`NewTemplate` as a documented, enforced requirement.** Rejected on
three counts. First, "enforced by every method" cannot be honored:
`NumQubits`, `ParamNames`, and `ParamStepCounts` return no error, so on
an unconstructed value they could only return zero values, which are
indistinguishable from the answers a legitimately empty template gives.
Adding error returns to them is a breaking signature change for a defect
no caller in the module can hit. Second, the methods that do return
errors already reject every meaningful operation on a zero-qubit
template through rules that exist for other reasons: `checkTargets`
refuses every target, `circuit.New` refuses the qubit count. A
"not constructed" sentinel error would duplicate those rejections with
a new error type that `NewTemplate` callers can never see. Third, the
package has no such precedent; the module's one statement on the
subject, `VQEOptions` in `algorithm/vqe.go:130` ("Zero values select
defaults"), goes the other way.

**The zero value is usable.** Chosen. This is the Go idiom (Effective
Go: "make the zero value useful"; `sync.Mutex`, `bytes.Buffer`,
`strings.Builder` all work unconstructed), and it is where the type
already is except for one map write: the table in Context shows every
other method giving the answer a template for zero qubits should give.
The contract, stated on the `Template` type: the zero value is ready to
use and is the template `NewTemplate(0)` returns; it declares nothing,
rejects every target as out of range, and `Bind` fails as `circuit.New`
does for a qubit count of zero; no method panics on it; `NewTemplate` is
how a template for a positive qubit count is made, not a precondition of
the methods.

Amended 2026-09-09 after round-1 review: that sentence undersold the
ordering "Rulings on edge cases" already states below — `Bind` runs its
own parameter checks first, so a call with an undeclared or non-finite
key fails with `UnknownParameterError` or `InvalidParameterValueError`
before the qubit-count check in `circuit.New` is ever reached, whatever
the qubit count. The `Template` doc comment and the CHANGELOG entry now
say "once `Bind`'s own parameter checks pass" rather than stating the
`circuit.New` failure unconditionally.

## Decision 2: one allocation site, in `AddParamGate`

Three shapes for the fix were weighed.

**Allocate lazily in `AddParamGate` and keep `NewTemplate`'s `make`.**
Two initialization paths for one map. Worse, after item 19 lands the lazy
branch becomes unreachable from a zero value: every target is out of
range on zero qubits and a no-target call will be rejected before the
bookkeeping, so the branch would be dead code whose only witness is the
moved red test, which item 19 hollows out (see "Item 19 follows on
locally"). Rejected.

**Allocate lazily in `AddParamGate` and drop the `make` from
`NewTemplate`.** Chosen. `AddParamGate` is the only writer of `seen`, so
a nil check there is the single place the map can come into being, and
every `NewTemplate`-built template exercises it on its first declaration:
the branch is live in every existing test, not only in the zero-value
ones. `NewTemplate(n)` becomes `&Template{numQubits: n}`, which makes
Decision 1's "the zero value is the template `NewTemplate(0)` returns"
literally true rather than behaviorally true. `Bind`'s read of a nil
`seen` on a template with no declarations returns `false` for every key,
which is the correct answer (every key is undeclared), so `Bind` needs no
change; a comment on the `seen` field states the invariant.

**Replace `seen` with a linear scan of `paramOrder`.** Removes the map,
and with it the nil-map hazard, entirely. Rejected: `Bind` checks every
key of `values` against `seen`, so the scan turns a per-key O(1) lookup
into O(declared names) on the hot path of every variational iteration,
and it rewrites two call sites to close a one-branch defect.

## Decision 3: no validation of `numQubits` in `NewTemplate`

`NewTemplate(0)` and `NewTemplate(-1)` are accepted today and fail at the
first `Bind` through `circuit.New`; item 14's spec
(`docs/superpowers/specs/2026-09-08-vqe-input-validation-design.md`,
"Zero-qubit template" ruling) records that `VQE` relies on exactly that
and wraps the result as `InvalidVQEInputError`. Making the zero value
usable means making a zero-qubit template usable, which is the same
state; adding a constructor error now would contradict Decision 1 (the
zero value cannot be refused) and change `VQE`'s documented error path.
Unchanged and out of scope.

## Item 19 follows on locally

Item 19 (no-target `AddParamGate` and `AddGate` accepted) adds a guard
that rejects an empty target list at the `Add*Gate` call. This design
leaves `checkTargets` and both methods' target handling byte for byte as
they are, and adds the `seen` allocation after the target check inside
`AddParamGate`; item 19's guard can go at the top of either method or
inside `checkTargets` without touching the allocation. Its tagged test
stays in `parameterized/backlog_red_test.go` and still fails after this
item; the prototype's red-tag run of `parameterized` lists exactly that
one failure.

One consequence for item 19's planner: the moved red test calls
`AddParamGate("", Ry)` with no targets on a zero-value template and
checks `ParamNames` and `ParamStepCounts` only when that call succeeds.
Once item 19 rejects the call, the success branch is never entered and
the test still passes, because its contract is the absence of a panic.
`TestZeroValueTemplateIsAZeroQubitTemplate` (Testing, below) pins the
zero value's observable behavior through calls that item 19 does not
change, so the pin survives.

## Rulings on edge cases

- **A no-target `AddParamGate` on the zero value** is accepted today, as
  it is on any template (item 19's defect), so `ParamNames()` is `[""]`
  and `ParamStepCounts()` is `map["":1]` afterwards. The moved red test
  asserts exactly that inside its `err == nil` branch. Not a ruling of
  this item; item 19 changes it.
- **Declarations that are rejected leave nothing declared.** On the zero
  value every targeted declaration fails in `checkTargets` before the
  bookkeeping, so `ParamNames()` stays empty afterwards. Pinned by
  `TestZeroValueTemplateIsAZeroQubitTemplate`.
- **`Bind` on the zero value** runs its own parameter checks first (an
  undeclared key is `UnknownParameterError`, whatever the qubit count),
  then fails in `circuit.New` with `InvalidQubitCountError`. Both are
  existing behavior; pinned by the same test so the order is recorded.
- **Copying a `Template` by value** after declarations shares `steps`,
  `paramOrder`, and `seen` between the copies. Unchanged and out of
  scope; the zero value being usable says nothing about copies.
- **Concurrency.** The allocation is a write inside `AddParamGate`, which
  is already a write; "safe for concurrent reads after all Add calls
  complete" is unchanged.
- **`ParamNames` on the zero value** returns a non-nil empty slice, as it
  does for `NewTemplate(n)` before any declaration. Unchanged.

## Effect on callers

- `algorithm/h2.go` (`H2Ansatz`), `algorithm/qaoa.go` (`QAOATemplate`),
  and every test build templates with `NewTemplate` and then declare at
  least one parameter, so each now allocates `seen` on its first
  `AddParamGate` instead of in the constructor. Same map, same contents,
  one call later. The prototype's `go run ./cmd/quantum -demo qaoa`
  still prints `VQE from the symmetric start: energy = -1.0000, expected cut = 2.0000`
  followed by `iterations accepted: 29, energy evaluations: 378`.
- A template built with `NewTemplate` and never given a parameter
  (`fixedOnly` in `TestParamStepCounts`, `NewTemplate(2)` in its table)
  now carries a nil `seen` into `Bind`, where every read yields `false`.
  `Bind` with an empty `Params` makes no read; `Bind` with any key
  returns `UnknownParameterError`, as before.
- `internal/examples/qaoa.go` and `cmd/quantum/main.go` call `VQE` and
  `Bind` only.
- No Go file outside `parameterized/` changes. `CHANGELOG.md` gains an
  entry.

## Backward compatibility (ADR-style note)

Inputs that used to panic and now succeed or error: any method call on a
`Template` not built by `NewTemplate`. Nothing built by `NewTemplate`
behaves differently. No signature changes. Per
`docs/compatibility-policy.md` this is a personal project with no
external consumers; every internal caller is listed above and unaffected.

- **CHANGELOG:** following items 15 through 17, an Unreleased entry
  under the existing `### Fixed` heading, appended after item 17's entry
  (entries within a section are appended in landing order).
- **ADR-0010:** no amendment. Its decision (template materialization over
  symbolic gates) and consequences (validation at `Bind`; QAOA reuses the
  template) are untouched; where a private map is allocated is below the
  ADR's level.
- **The 2026-08-30 spec** (`docs/superpowers/specs/2026-08-30-parameter-binding-vqe-design.md`)
  lists `NewTemplate(numQubits int) *Template` and says nothing about the
  zero value or about the map; nothing in it becomes false. No amendment.
- **Item 14's spec:** its "Zero-qubit template" ruling stays exact
  (Decision 3). No amendment.

## Red test rulings

`TestRedZeroValueTemplateAddParamGateDoesNotPanic` moves into
`parameterized/parameterized_test.go` as
`TestZeroValueTemplateAddParamGateDoesNotPanic`, the package's
`Test<Subject><Verb>...` style (`TestAddRejectsOutOfRangeTargets`,
`TestSameParamDrivesTwoGates`). Setup, the recover guard, the call, and
the conditional assertions are unchanged; only the leading comment is
rewritten to describe the contract. Each assertion was checked against
the contract:

- The call must not panic: Decision 1's own sentence. Consistent.
- If the call succeeds, `ParamNames()` is `[""]` and
  `ParamStepCounts()[""]` is 1: the bookkeeping that runs after the
  allocation is unchanged, and item 15 already pins that `""` is a name
  like any other. Consistent, and today the call does succeed (item 19).
- The failure message says `NewTemplate is not documented as required`:
  after this item the documentation says the opposite outright, so the
  message stays true on a regression.

No assertion was ruled against the contract. The moved test says nothing
about what the zero value *is*, so `TestZeroValueTemplateIsAZeroQubitTemplate`
pins the accessors, both `Add` rejections, and `Bind`'s two failure modes
alongside it, rather than editing the moved test.

## Testing

All in `parameterized/parameterized_test.go` (package `parameterized_test`):

- `TestZeroValueTemplateAddParamGateDoesNotPanic`: the moved red test.
  Fails today with the panic message.
- `TestZeroValueTemplateIsAZeroQubitTemplate`: `NumQubits() == 0`, empty
  `ParamNames()` and `ParamStepCounts()`, `AddParamGate` and `AddGate`
  with target 0 each `QubitsOutOfRangeError` via `errors.As`, nothing
  declared afterwards, `Bind` with an undeclared key
  `UnknownParameterError`, `Bind` with empty `Params`
  `InvalidQubitCountError`. Passes before and after; pins Decision 1's
  contract through calls item 19 does not change. Needs the `quantum`
  import, which the file does not have yet.
- `go test -tags redtests ./parameterized -run '^TestRed'` must list
  exactly `TestRedAddParamGateRejectsNoTargets` (item 19) as failing;
  `algorithm`'s red run still lists exactly
  `TestRedParameterShiftAtHugeAngleMatchesReducedAngle` (item 20).

Prototype: every code and test change in the plan was applied to a
scratch copy of the repository at `ab6e2b5`; the moved test failed with
the panic message against the original `AddParamGate` and both tests
passed after the change, with `gofmt -l`, `go vet` (plain and
`-tags redtests` on `parameterized` and `algorithm`), `staticcheck`,
`go build`, `go test ./...`, `go test -race ./parameterized ./algorithm`,
the two red-tag runs, and the QAOA demo all as stated.

## Out of scope

Item 19 (empty target lists). Item 20. Validation of `numQubits` in
`NewTemplate` (Decision 3). Value-copy semantics of `Template`. Any
change to `Bind`, `ParamNames`, `ParamStepCounts`, or any exported
signature.
