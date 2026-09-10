# Declared Targets Copied Out of the Caller's Slice — Design

Date: 2026-09-10
Status: Approved (backlog item text plus run-lead dispatch; no brainstorming gate).
Resolves: backlog item 22 (`.superpowers/backlog/enhancement-backlog-2026-08-27/item-22.md`)

## Context

`parameterized.Template`'s two writers take their targets variadically,
`targets ...int`, and store the resulting slice directly:

```go
t.state.steps = append(t.state.steps, step{param: name, factory: factory, targets: targets})   // AddParamGate
t.state.steps = append(t.state.steps, step{gate: gate, targets: targets})                      // AddGate
```

(`parameterized/parameterized.go:253` and `:281`.) When the caller spreads
a slice of its own (`tmpl.AddParamGate("theta", Ry, targets...)`), Go
passes that slice, not a copy, so `step.targets` and the caller's variable
share one backing array. The call returns nil, `ParamNames` and
`ParamStepCounts` list the declaration, and everything reads as complete;
but the caller still owns the array, and `Bind` reads it at bind time,
however long after. Observed today
(`go test -tags redtests ./parameterized -run '^TestRedDeclaredTargetsAreNotAliasedToCallerSlice' -v`):

| Sequence on `NewTemplate(2)` | Today | Contract |
|---|---|---|
| `targets := []int{0}`; `AddParamGate("theta", Ry, targets...)` | nil | unchanged |
| then `targets[0] = 1`; `Bind(Params{"theta": 0.2})` | `Ry(0.2)` on qubit 1 | `Ry(0.2)` on qubit 0, as declared |
| `pair := []int{0, 1}`; `AddGate(CNOT, pair...)`; `pair[1] = 0` | `Bind` fails: `targets must be unique` | `CNOT` on 0, 1, as declared |
| `targets[0] = 7` after any declaration | `Bind` fails: `qubit index 7 is out of range [0,1]` | unaffected; the range was checked at declaration |

Three things go wrong, in increasing order of how quietly they go wrong.
`Bind` builds a different circuit from the declared one and says nothing,
the item's own reproduction. `Bind` fails on a template every accepted
declaration said was legal, so a declaration-time error surfaces as a
bind-time one, which is the failure mode items 18 and 19 spent their whole
scope moving in the other direction. And the range check `checkTargets`
runs (`parameterized/parameterized.go:218-225`) stops meaning anything
about what will be built: it validates an array the caller may rewrite
afterwards.

The trigger is not exotic. A generator that emits a layer of rotations
fills one `[]int` and reuses it per qubit; that is the pattern the item
names, and it is why the variadic form matters here. A caller writing
`AddParamGate("theta", Ry, 0)` never allocates a slice and can never hit
this, so the sharing is invisible at exactly the call sites that read as
the normal way to use the API, and visible only to the caller who reaches
for the spread form.

Nothing in the module is affected. `algorithm/h2.go` (`H2Ansatz`),
`algorithm/qaoa.go` (`QAOATemplate`), and every test pass literal targets
or spread a slice they never touch again. The defect is reachable only by
a caller outside the module.

Item 21, which landed immediately before this one, moved `steps` and
`paramOrder` into a heap-allocated `templateState` that every by-value
copy of a `Template` shares by pointer. That makes the sharing here
strictly wider than it was: the caller's array is now reachable from every
copy of the template, through one state that only the pinned original may
write. It does not change what the fix is, but it does settle where the
fix belongs (Decision 2).

## Classification

**Standard.** One new unexported helper and one changed expression in each
of the two writers, plus three doc comments; no signature, exported type,
or cross-package change. Short design doc plus a one-task plan.

## Decision 1: `AddParamGate` and `AddGate` copy the targets they store

The item leaves the contract open: either the writers copy, or the doc
comments state that the caller must not mutate the slice after the call.
Four options were weighed.

**Document only ("do not mutate the slice you passed").** Rejected. It is
an acceptable contract in the abstract and the item allows it, but it is
the wrong one here and the dispatch rules out any version of it that
deletes the red test rather than satisfying it. Three reasons. First, the
rule is unenforceable and unobservable: nothing errors, nothing panics,
`go vet` has no check for it, and the symptom is a circuit that is wrong
by one qubit index, which a variational loop will happily optimize. Second
it is a rule about a slice the caller may never have written: the variadic
form means `AddParamGate("theta", Ry, 0)` and
`AddParamGate("theta", Ry, targets...)` are the same call, so the
documented hazard attaches to a parameter that usually has no caller-side
identity at all. Third, it contradicts the package's own direction of
travel. Items 18, 19, and 21 each moved a caller mistake from a silent
wrong answer to a checked, reported one at the earliest point; a
documentation-only ruling here would move in the opposite direction, and
would leave `checkTargets`' declaration-time range check validating an
array that is not the one `Bind` will read.

**Copy in `Bind` instead, when the circuit is built.** Rejected, and not
as a tradeoff: it does not fix the item. The mutation happens between the
declaration and the bind, so whatever `Bind` copies is already the
rewritten list. A `Bind`-side copy would only protect the built circuit
from mutations made after `Bind` returned, and `circuit.AddGate` already
does exactly that (Decision 4). It is also the more expensive site by a
wide margin: declarations happen once per template, binds happen once per
energy evaluation, 378 of them in the QAOA demo alone.

**Copy on the way in, in both writers.** Chosen. This is what Go's own
convention says to do with a slice retained past the call that received
it: `append` and `copy` exist for it, and the standard library copies what
it stores (`bytes.Buffer.Write`, `strings.Builder.Write`,
`time.Time.AppendFormat` and friends all treat the caller's bytes as
borrowed). This repository already follows it one layer down, in the
function these very targets are forwarded to: `circuit.AddGate` stores
`Targets: append([]int(nil), targets...)` (`circuit/circuit.go:79-82`).
The template is the one place in the chain that does not, so this makes
`parameterized` consistent with `circuit` rather than introducing a new
rule. The cost is smaller than it looks, because a writer that stops
retaining its argument stops forcing it onto the heap. Measured with
`go build -gcflags=-m ./parameterized` on go1.26.3, `targets` goes from
`leaking param: targets` in both writers to `targets does not escape`,
and at the call site the variadic slice the compiler builds stops
escaping with it. What a declaration pays then depends on its call shape:

| Per accepted declaration | Before | After |
|---|---|---|
| literal targets, `AddGate(g, 0, 1)` | 4 allocs, 152 B | 4 allocs, 152 B |
| a caller-held slice spread, `AddGate(g, targets...)` | 3 allocs, 136 B | 4 allocs, 152 B |
| rejected on an out-of-range target, literal | 3 allocs, 56 B | 2 allocs, 40 B |
| `H2Ansatz()`, three declarations, all literal | 19 allocs, 1128 B | 19 allocs, 1128 B |

(One call on a fresh two-qubit template per operation, `-benchmem`, three
runs at each commit with identical counts; commands in Testing.) So the
guarantee is free for a declaration written with literal targets, which
is every declaration in this module: the copy takes over the heap
allocation the argument slice used to make, at the same size. The caller
that holds a slice and spreads it, the caller this item is about, is the
only one that pays, and it pays one allocation, the copy itself. A
rejected declaration makes one allocation fewer than before. This is
escape analysis and not a language guarantee, so it is what the current
toolchain does rather than something the design promises; the copy is a
promise either way. Against that cost, a class of defect whose symptom is
a silently different circuit. `BenchmarkVQEIteration`, which measures
bind-plus-execute-plus-energy with the template built before
`ResetTimer`, does not touch the changed code at all: two runs at 2s gave
2239 ns/op before and 2322 ns/op after, within run-to-run noise on an
unchanged path.

**Copy, and also freeze the step list against later declaration.**
Rejected as a bigger contract than the item states. Nothing here is about
declaring after a bind, and item 21 already settled who may write a
template's state.

## Decision 2: the copy is made by one unexported helper, at the append, after the checks pass

**Where.** At the `append` in each writer, after `checkNotCopied`, the
argument checks, `checkTargets`, and `pin` have all succeeded. Two
consequences are wanted here. A rejected declaration allocates nothing,
which keeps items 18, 19, and 21's shared ruling that a rejected
declaration leaves the template exactly as it was. And the copy is made at
the moment the value enters `templateState`, which after item 21 is shared
by every by-value copy of the template: putting a caller-owned array into
that state is what this item is about, so the boundary the copy guards is
precisely the state's edge. Copying at the top of each method instead
would be equivalent in behavior but would allocate on paths that reject.

**Not inside `checkTargets`.** The helper is a pure range check that
returns an error and no value; making it return a copy too would give it
two jobs and would allocate before `pin`, on a path that may still reject.
Item 19's Decision 1 deliberately kept it a pure check, and this design
does not reopen that.

**One helper, not an inline expression.** `append([]int(nil), targets...)`
written twice inside the two composite literals makes the `AddParamGate`
line 116 characters, counting its leading tab as one, against 107 with
the helper, and puts the reason for the copy nowhere. An unexported `cloneTargets(targets []int) []int` keeps both
appends readable, matches the package's habit of tiny unexported helpers
(`checkTargets`, `checkNotCopied`, `pin`, `declared`, `names`,
`stepList`), and gives the one place where the reason is written down.
Its body is `append([]int(nil), targets...)`, the exact expression
`circuit.AddGate` uses, so the two copies in the chain read the same.

**Not `slices.Clone`.** The module is on Go 1.25 and could use it, but no
file in the repository imports `slices` today and `circuit.AddGate` uses
the `append` form; matching the neighbor is worth more than saving the
literal. Behaviorally they agree here in any case, since both writers
reject an empty target list before the copy is reached.

## Decision 3: the doc comments state the contract as a guarantee

`AddParamGate` and `AddGate` each gain one sentence: the targets are
copied, so the slice a caller passed may be reused or mutated after the
call without changing what was declared. This is the promise a caller
needs in order to write the loop that reuses one slice, and it is the
sentence the alternative contract would have inverted, so it belongs in
the same place either way.

`step.targets` gains a field comment saying it is the template's own copy,
made by `cloneTargets`, and never the caller's slice, so a later editor
does not reintroduce the alias by writing the obvious `targets: targets`.

The `Template` doc comment is unchanged. It states the zero-value and
copy-by-value contracts of the type; target ownership is a per-call
property of the two writers, and both now say it. `Bind`'s doc comment is
unchanged: it builds from the declarations, which is what it already says,
and Decision 4 leaves its behavior alone.

## Decision 4: the way out needs no change, and routes no new item

`Bind` forwards a step's targets with `c.AddGate(gate, s.targets...)`
(`parameterized/parameterized.go:325`). Spreading a slice into a variadic
parameter passes the slice, so `circuit.AddGate` receives `s.targets`
itself, not a copy. It reads it for the range, width, and uniqueness
checks and then stores `Targets: append([]int(nil), targets...)`
(`circuit/circuit.go:79-82`): it copies what it retains. The built
circuit therefore never aliases the template's step targets, a later
declaration on the template cannot change a circuit already bound, and two
circuits bound from one template share nothing.

So there is no defect of this class on the way out, nothing to fix here,
and no new backlog item to route. For completeness, the target lists
inside a `Circuit` are not reachable by a caller either: `operations` is
unexported and `circuit` has no accessor that returns operations, so the
exported `Operation.Targets` field is only reachable inside the `circuit`
package. `parameterized` exposes no accessor for a step's targets at all.

## Rulings on edge cases

- **A caller that never spreads a slice.** `AddParamGate("theta", Ry, 0)`
  and `AddGate(CNOT, 0, 1)` build their variadic slice at the call site,
  and nobody else holds it. The copy is redundant for them, and on the
  measured toolchain it is also free: the writer no longer retains the
  slice, so the compiler leaves it on the stack and the copy takes over
  the heap allocation it used to make (Decision 1). They pay the same
  copy once more one layer down, in `circuit.AddGate`, as they always
  did.
- **One slice reused across declarations.** The pattern the item names.
  Each declaration keeps its own copy, so the steps hold the targets each
  was given. Pinned by `TestAddCopiesTheCallerTargetsSlice`.
- **Mutation to an illegal value after the call.** Setting a target
  out of range, or duplicating one within a step, after an accepted
  declaration no longer reaches `Bind`. The declaration-time range check
  now describes what will be built, and width and uniqueness stay
  `circuit.AddGate`'s to enforce at `Bind` for the list as declared.
  Pinned in the same test.
- **A rejected declaration.** The copy is made after every check, so a nil
  factory, a nil gate, an empty target list, or an out-of-range target
  leaves the template untouched and makes no copy. It is cheaper than
  before rather than the same: the argument slice no longer escapes, so a
  rejection written with literal targets allocates its error value and
  nothing else, one allocation fewer than it made before (Decision 1).
- **The zero value.** Unchanged. `cloneTargets` is called only on a path
  that has already pinned the template and allocated its state, and no
  method gains a panic.
- **A by-value copy of a `Template`.** Unchanged. Item 21's guard runs
  first in both writers, so a refused copy never reaches the copy.
  Because every copy shares one `templateState`, the change also means no
  copy of a template can be reached through a caller's slice.
- **Concurrency.** `cloneTargets` reads its argument and writes only a
  fresh array, inside methods that are already writes. "Safe for
  concurrent reads after all `Add` calls complete" is unchanged, and
  `Bind` is not edited. A caller mutating its slice concurrently with the
  `Add` call that receives it was a data race before and still is; the
  copy narrows the window to the call itself rather than to the
  template's lifetime.
- **Empty target list.** Unreachable: item 19's guard rejects it in both
  writers before the copy. `append([]int(nil))` would return nil in any
  case.
- **Aliasing between two steps.** Two declarations that spread the same
  caller slice now hold two independent copies, so neither can be reached
  through the other.

## Effect on callers

- `algorithm/h2.go` (`H2Ansatz`), `algorithm/qaoa.go` (`QAOATemplate`),
  `internal/examples/qaoa.go` (through `QAOATemplate`), and every test
  declare with literal targets or with a slice they do not touch again;
  none changes behavior. None allocates more either: they declare with
  literal targets, the call shape the copy is free for, and `H2Ansatz`
  builds its three-declaration template in the same 19 allocations and
  1128 bytes it did before (Decision 1). The prototype's
  `go run ./cmd/quantum -demo qaoa` still prints
  `VQE from the symmetric start: energy = -1.0000, expected cut = 2.0000`
  followed by `iterations accepted: 29, energy evaluations: 378`.
- A caller outside the module that reuses one target slice across
  declarations now gets the circuit it declared instead of one built from
  the last targets written. No caller that did not mutate its slice sees
  any difference.
- No Go file outside `parameterized/` changes. `CHANGELOG.md` gains an
  entry.

## Backward compatibility (ADR-style note)

No input that was accepted now errors, and no input that errored now
succeeds. What changes is what `Bind` builds for a caller that mutated a
target slice after declaring with it: the declared circuit rather than a
rewritten one, and success rather than a range or uniqueness failure at
`Bind`. Any caller relying on the old behavior was relying on rewriting a
completed declaration through an aliased slice. No signature changes; the
new helper is unexported. Per `docs/compatibility-policy.md` this is a
personal project with no external consumers; every internal caller is
listed above and unaffected.

- **CHANGELOG:** following items 15 through 21, an Unreleased entry under
  the existing `### Fixed` heading, appended after item 21's entry
  (entries within a section are appended in landing order).
- **ADR-0010:** no amendment. Its decision (template materialization over
  symbolic gates) and its consequences (validation at `Bind`; QAOA reuses
  the template) are untouched; whether a template stores its own target
  list is below the ADR's level, as item 21's ruling on the copy guard
  already found for the same ADR. No new ADR either: this is a defect fix
  inside one type, not a structural decision, and the repository's ADRs
  (`docs/adr/`) record cross-cutting choices such as backend contracts and
  API generations.
- **The 2026-08-30 spec** (`docs/superpowers/specs/2026-08-30-parameter-binding-vqe-design.md`)
  says `AddParamGate`/`AddGate` "reject out-of-range targets at
  declaration time, mirroring `circuit.New`". That claim becomes fully
  true only with this change, since until now the checked list could be
  rewritten afterwards. Nothing in it becomes false. No amendment.
- **Item 19's spec** (`docs/superpowers/specs/2026-09-09-empty-target-list-design.md`)
  says everything after its guard is byte for byte what item 18 left it,
  and rules width and uniqueness out of scope as `circuit.AddGate`'s at
  `Bind`; both stay true (this design edits the `append` expression, not
  the guard, and moves no check). No amendment.
- **Item 21's spec** (`docs/superpowers/specs/2026-09-10-template-copy-guard-design.md`)
  lists item 22 under "Out of scope" and says its red test still fails
  after item 21 because the `append` line that stores `targets` is not
  edited. This design edits exactly that line. No amendment: the statement
  was true when written and is a scope note, not a contract.
- **Item 21's plan** (`docs/superpowers/plans/2026-09-10-template-copy-guard.md`)
  says, in its Interfaces block, that item 22's copy "goes into those
  lines or just above them without touching the pin or the check"; it goes
  into those lines. Plans are historical records and are not amended.

## Red test rulings

`TestRedDeclaredTargetsAreNotAliasedToCallerSlice` moves into
`parameterized/parameterized_test.go` as
`TestDeclaredTargetsAreNotAliasedToCallerSlice`, the package's
`Test<Subject><Verb>...` style. Setup, calls, and assertions are copied
byte for byte; only the leading comment is rewritten to describe the
contract instead of the defect. Each assertion was checked against the
contract:

- `AddParamGate("theta", Ry, targets...)` with `targets := []int{0}` must
  succeed: one in-range target on a two-qubit template. Consistent.
- `targets[0] = 1` then `Bind(Params{"theta": 0.2})` must succeed: the
  binding names the one declared parameter with a finite value, and the
  declared target is still 0. Consistent.
- The bound circuit must match a manual `Ry(0.2)` on qubit 0, amplitude
  by amplitude with exact equality: both circuits apply the same gate
  built by the same constructor to the same index of a fresh state, so
  the float64 results are bit-identical, not merely close. Consistent.

No assertion was ruled against the contract, so none is adjusted. The
moved test exercises `AddParamGate` only, and says nothing about `AddGate`,
about reusing one slice across declarations, or about a mutation that
would make a declaration illegal at `Bind`, so one test pins those
alongside it rather than editing the moved test:

- `TestAddCopiesTheCallerTargetsSlice`: declares `theta` on 0 and `phi` on
  1 through one reused `[]int`, declares a `CNOT` through a second slice,
  then writes a duplicate target into the second and an out-of-range
  target into the first, and asserts `Bind` succeeds and matches a manual
  `Ry(0.2)` on 0, `Rx(0.3)` on 1, `CNOT` 0-1.

With item 22's test gone, `parameterized/backlog_red_test.go` holds no
tests. It is deleted, as `algorithm/backlog_red_test.go` was when item 17
emptied it: a tagged file with only a header comment documents nothing.

Effect on the `redtests` build tag, checked across the repository. No Go
file in any package will carry the tag. Nothing in the tooling invokes it:
`.github/workflows/ci.yml` runs gofmt, `go vet ./...`, staticcheck,
`go test -race ./...`, and coverage, all untagged; `.github/workflows/fuzz.yml`
runs fuzz targets untagged; `.superpowers/backlog/enhancement-backlog-2026-08-27/gate.sh`
mirrors CI and passes no tags; there is no Makefile and no script that
names the tag. So no CI step or QA gate breaks or turns into a no-op, and
none needs editing. `go vet -tags redtests ./parameterized ./algorithm`
still compiles cleanly against packages with no tagged file, and
`go test -tags redtests ./parameterized ./algorithm -run '^TestRed'`
reports `ok ... [no tests to run]` for both, the same outcome item 17's
plan recorded for `algorithm`. The convention itself (a red test lives
under the tag until its item is closed, then moves into the regular suite)
survives in the backlog document, which states it per item along with the
command to reproduce; the next item that needs a red test recreates the
file. `CLAUDE.md` does not exist in this repository and `AGENTS.md` never
mentions the tag, so neither needs a change; the plans and specs that name
it are historical records and are not amended.

## Testing

All in `parameterized/parameterized_test.go` (package `parameterized_test`);
no import changes, since `circuit`, `gates`, `state`, and `parameterized`
are all already imported there:

- `TestDeclaredTargetsAreNotAliasedToCallerSlice`: the moved red test.
  Fails today with
  `amplitude 1 after mutating the caller's slice: got (0+0i), want (0.09983341664682815+0i) (Ry on qubit 0 as declared)`.
- `TestAddCopiesTheCallerTargetsSlice`: fails today at `Bind` with
  `Bind after mutating both caller slices: qubit index 7 is out of range [0,1], want the circuit as declared`.
- `go test -race ./parameterized ./algorithm` clean: the copy is a write
  to a fresh array inside a method that already writes.
- `go test -tags redtests ./parameterized ./algorithm -run '^TestRed'`
  reports `[no tests to run]` for both packages, with no tagged file left
  anywhere in the repository.
- Decision 1's allocation figures, on go1.26.3 darwin/amd64:
  `go build -gcflags=-m ./parameterized` for the escape decisions on both
  writers, and `go test -run XXX -bench . -benchmem` over a probe package
  holding one `AddGate` per operation on a fresh two-qubit template, in
  each call shape, plus one `algorithm.H2Ansatz()` per operation. The
  probe is not committed: it is run out of tree, in `git archive` copies
  of `8ee8579` and of this change, so the two commits are measured with
  the same file.

Prototype: every code and test change in the plan was applied to a scratch
copy of the repository at `8ee8579`; both tests failed as stated against
the original writers, all twenty-four tests in the package passed after the
change, and `gofmt -l`, `go vet` (plain and `-tags redtests` on
`parameterized` and `algorithm`), `staticcheck`, `go build`,
`go test ./...`, `go test -race ./parameterized ./algorithm`, the red-tag
runs, and the QAOA demo were all as stated. The diff was +27/-8 on
`parameterized/parameterized.go`, +18 on `CHANGELOG.md`, +90 on
`parameterized/parameterized_test.go`, and the 61-line red file removed.

## Out of scope

Width and uniqueness checks on targets at declaration time (item 19 ruled
them `circuit.AddGate`'s at `Bind`, and this design does not move them).
Copying anything else a caller hands the package: the `Params` map `Bind`
takes is read during the call and never retained, and the `Factory` and
`quantum.Gate` values a declaration stores are the caller's by design,
since a gate is immutable metadata and a factory is a function. Any
`Clone` method on `Template`. Typing the declaration-time argument errors.
Any change to `checkTargets`, `Bind`, `NewTemplate`, `NumQubits`,
`ParamNames`, `ParamStepCounts`, or any exported signature.
