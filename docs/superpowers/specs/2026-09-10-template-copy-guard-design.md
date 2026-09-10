# `Template` Copy Guard — Design

Date: 2026-09-10
Status: Approved (backlog item text plus run-lead dispatch; no brainstorming gate).
Resolves: backlog item 21 (`.superpowers/backlog/enhancement-backlog-2026-08-27/item-21.md`)

## Context

`parameterized.Template` holds four fields: `numQubits int`, `steps []step`,
`paramOrder []string`, and `seen map[string]bool`. A Go value copy (`b := a`,
`a := *NewTemplate(2)`, passing or returning a `Template` by value, storing
one in a slice that later grows) copies the int, copies the two slice
headers, and copies the map reference. After at least one declaration the
copies therefore share one `seen` map and the backing arrays of `steps` and
`paramOrder`, while each owns its own slice lengths. `AddParamGate`
(`parameterized/parameterized.go:112-131`) consults `seen` to decide whether
a name is new and appends to `paramOrder` only when it is; `ParamStepCounts`
scans `steps`; `Bind` iterates `paramOrder` for completeness and reads
`seen` for unknown keys.

Amended 2026-09-10 (round 1 fix): the map described here is removed;
`AddParamGate` and `Bind` now read the name list itself. See Decision 1's
amendment and the copy-back ruling under "Rulings on edge cases". The
rest of this section describes the code as it was when the design was
written.

Amended 2026-09-10 (round 2 fix): the slices described here no longer sit
in the `Template` value either; `steps` and `paramOrder` live in a
heap-allocated `templateState` that the first accepted declaration
allocates and every later copy shares by pointer. See the round 2
amendment under "Rulings on edge cases" ("A copy assigned back over the
original, twice").

Observed today (`go test -tags redtests ./parameterized -run '^TestRedCopyAfterDeclarationKeepsNamesAndCountsConsistent' -v`),
with `a := *NewTemplate(2)`, `a.AddParamGate("theta", Ry, 0)`, `b := a`:

| Step | `seen` (shared) | `a.paramOrder` | `a.steps` | `b.paramOrder` | `b.steps` |
|---|---|---|---|---|---|
| after `b := a` | `{theta}` | `[theta]` | `[theta]` | `[theta]` | `[theta]` |
| `b.AddParamGate("phi", Rx, 1)` | `{theta, phi}` | `[theta]` | `[theta]` | `[theta, phi]` | `[theta, phi]` |
| `a.AddParamGate("phi", Rx, 1)` | `{theta, phi}` | `[theta]` (phi "already seen") | `[theta, phi]` | `[theta, phi]` | `[theta, phi]` |

`a.ParamStepCounts()` is `map[phi:1 theta:1]`, `a.ParamNames()` is
`[theta]`, and `a.Bind(Params{"theta": 0.1})` returns a circuit: the
completeness loop runs over `paramOrder` only, so `phi` is never demanded,
and the `phi` step is built from `values["phi"]`, which is 0 for a missing
key. A parameter silently bound at angle 0 is exactly the failure `Bind`'s
`MissingParameterError` exists to prevent.

The slices are hazardous on their own, independent of `seen`. Once
`append` has left spare capacity (three declarations give `len 3, cap 4`),
a copy and its original both write their next step into the same backing
slot: `b.AddGate(H, 0)` then `a.AddGate(X, 1)` leaves `b.steps[3]` holding
`a`'s `X` gate, because `a`'s append overwrote the shared slot that `b`'s
header still covers. Nothing in `b` records that the swap happened.

The item offers two contracts: document `Template` as not safe to copy by
value after any declaration, or make `AddParamGate`, `ParamNames`, and
`ParamStepCounts` independent per copy. Nothing in the module copies a
`Template` by value: `algorithm/h2.go`, `algorithm/qaoa.go`,
`algorithm/vqe.go`, `internal/examples/qaoa.go`, and every test hold a
`*parameterized.Template` or use a zero-value variable in place. The defect
is reachable only by a caller outside the module.

## Classification

**Standard.** One new unexported field, one unexported helper, a guard at
the top of three existing methods, and a one-line pin in two of them; no
signature, exported type, or cross-package change. Short design doc plus a
one-task plan. (Amended 2026-09-10, round 1 fix: also one field removed,
`seen`, and a second unexported helper, `declared`; still no signature,
exported type, or cross-package change. Amended 2026-09-10, round 2 fix:
the two slice fields move into one unexported struct, `templateState`,
held through one pointer field, `state`, with a third unexported helper,
`pin`; still no signature, exported type, or cross-package change.)

## Decision 1: a `Template` is not copyable after a declaration, and copies are refused at runtime

**Make copies independent.** Rejected because Go cannot do it. A value
copy is a shallow copy the language performs without calling any method,
so the only ways to make `b := a` independent are to have no shared
reference in the struct at all, or to detect the copy afterwards. Removing
`seen` (the map) does not remove the sharing: the slice backing arrays are
shared too, and the slot-overwrite in Context corrupts one copy's step list
through the other's `append` whether or not a map exists. Copy-on-write
(each mutator cloning `steps` and `paramOrder` before appending) would make
every declaration O(n) and still leave `seen`, which cannot be cloned
without a copy signal. Keeping `seen` as a cache validated against
`paramOrder` (store the index, trust the entry only if
`paramOrder[idx] == name`) was traced through and fails: two copies
interleaving declarations write conflicting indexes into the one map, and
the third declaration in such a sequence re-appends a name already listed.
"Independent per copy" is not achievable for a value type holding slices
or maps; this is why `sync.Mutex` documents, and `strings.Builder`
enforces, a rule against copying after first use rather than supporting
it.

Amended 2026-09-10 (round 1 fix): `seen` is now removed, though not to
make copies independent, which this paragraph still rules out: the
backing arrays stay shared and the guard stays what refuses a copy. It is
removed because it was the one field a by-value copy shared as a
reference rather than as a snapshot, and round 1's black-box red testing
found a copy the guard cannot see, a copy assigned back over the original
(ruling under "Rulings on edge cases"), on which the shared map lied about
which names the restored list holds. `AddParamGate` and `Bind` now ask
`paramOrder` through the unexported `declared`, a linear scan. Item 18's
spec rejected that scan for `Bind`'s hot path (O(declared names) per
key); `Bind` now runs it only when `values` has more keys than the
template has names, which after the completeness loop is exactly when an
undeclared key exists (every declared name is present and the names are
distinct), so the accepted path stays O(names) and the scan is paid only
on the way to `UnknownParameterError`. The validated-cache variant traced
above is moot with no map.

Amended 2026-09-10 (round 2 fix): "the backing arrays stay shared" is no
longer the layout. Round 1's re-review showed that per-copy slice headers
over shared arrays let a two-step copy-back desync the pinned original
itself (ruling under "Rulings on edge cases"). The lists now live behind
one pointer, so what a copy shares is the whole declaration state, one
generation of it, and never a header of its own. Copies are still not
independent, which this paragraph still rules out, and the guard is still
what refuses them; what changed is that a copy can no longer carry a view
of the lists that differs from the original's.

**Document only.** Rejected by the dispatch and on its merits: the
observed failure is a `Bind` that succeeds with a parameter silently at 0,
the quietest possible wrong answer, and a doc sentence leaves it exactly
as quiet.

**Static detection through `go vet`'s `copylocks` (a `noCopy` field).**
Rejected. It fires only when someone runs `vet`, which no caller outside
the module is obliged to do, so the desync at runtime is unchanged. It
would also flag the acceptance test's own `a := *NewTemplate(2)` and
`b := a` lines, which the plan runs `go vet ./...` over, so the test that
proves the defect fixed could not exist alongside it.

**Document as copy-unsafe and refuse copies at runtime, `strings.Builder`
style.** Chosen. `strings.Builder` keeps `addr *Builder`, sets it to the
receiver on first use, and on every later mutator compares it with the
receiver: a by-value copy carries the original's address and is caught
(`strings/builder.go`, `copyCheck`). A `Template` gains the same field,
`addr *Template`, and the same comparison. This is the precedent item 18's
design already cited for the zero-value contract, so the two rulings on
`Template` come from the same standard-library type: the zero value is
usable, and a used value is not copyable. The contract stated on the type:
a `Template` is used in place or through a pointer, as `NewTemplate`
returns it; a copy taken after a declaration has been accepted is refused
by `AddParamGate`, `AddGate`, and `Bind` with an error; a copy taken
before that shares nothing and is an independent template.

## Decision 2: the refusal is an error, not a panic

`strings.Builder` panics. Here the three guarded methods all already
return `error`, `AGENTS.md` says to prefer explicit errors over panics,
item 18's contract on the type says "no method panics on it", and the one
consumer in the module, `algorithm.VQE`, wraps whatever `Bind` returns
into `InvalidVQEInputError` through `wrapEvaluationError`
(`algorithm/vqe.go:308`), so an error reaches a variational caller as the
same typed failure every other bad template produces, while a panic would
end the run. Panic rejected.

The error is one package-level unexported value,
`errCopiedTemplate = errors.New("Template copied by value after a declaration; a declared Template is used in place or through a pointer, never by copy")`,
returned by all three methods. It is untyped and unexported for the
reason item 19's Decision 2 gave for the declaration-time argument checks:
the package's typed errors are `Bind`-time binding errors a variational
loop may branch on, and a misuse of the type is a caller bug of the same
family as a nil factory or an empty target list, which are `fmt.Errorf`.
It is one shared value rather than three `fmt.Errorf` calls because it
carries no per-call data. No caller in the module needs to branch on it;
tests match the phrase `copied by value` as item 19's tests match theirs.

## Decision 3: which methods check, and where the pin happens

**Checked: `AddParamGate`, `AddGate`, `Bind`.** These are the three
methods that read `seen` or append to a slice. On a copy, either write
would desync the copies (Context), and `Bind`'s unknown-key check reads
the shared `seen`, so a copy would accept a key its own `paramOrder` never
lists. (Amended 2026-09-10, round 1 fix: `seen` is gone and `Bind` reads
`paramOrder`, so a copy's `Bind` would now answer from the copy's own
lists. It stays refused: those lists share backing arrays with the
original's and, after a copy-back, may no longer hold what they held when
copied, see the copy-back ruling, and the contract as stated is that a
copy is unusable as a whole.) (Amended 2026-09-10, round 3 fix: both
reasons above are superseded; what the guard now does is stated in the
paragraph below.) Each method calls `checkNotCopied` as its first
statement, before its argument checks: a copied template is unusable
whatever the arguments, so `AddParamGate("x", nil, 1)` on a copy reports
the copy, not the factory, and `Bind` on a copy reports the copy before
any missing or unknown name.

Amended 2026-09-10 (round 3 fix, the round 3 review's I1). Under the
round 2 layout neither reason above survives. A copy holds the original's
`state` pointer and no list of its own, so deleting the guard would not
desync anything: a copy's `AddParamGate` would append to the one shared
state through the same code path the original uses, and `ParamNames`,
`ParamStepCounts`, and `Bind` on the copy and on the original would go on
agreeing. The same holds for every route past the guard the rounds have
found, the copy-back and the theoretical address coincidence included:
under the shared state a bypass yields an alias, never a desync. What
keeps the item's invariant is the `templateState`, not the guard.

The guard is kept for the contract, not for consistency. Item 21 offers
two acceptable contracts, "not safe to copy by value after any
declaration (only passed and stored by pointer)" and "independent per
copy"; two values of a value type quietly writing one declaration list is
neither. Decision 1 chose the first, so a used `Template` is refused as a
copy rather than allowed to act as a silent alias, and that is the whole
of what the guard now enforces. Removing it would be a contract change,
giving `Template` map-like alias semantics, not a consistency fix; it
would also reclaim the receiver escape (Decision 3's `noescape` note).
Neither is proposed here.

As one statement of the contract, for a reader who should not have to
reconstruct it from three amendments: the state pointer keeps
`ParamNames`, `ParamStepCounts`, and `Bind` consistent under any sequence
of by-value copies, copy-backs, and declarations; the guard refuses
copies as a matter of contract; and a copy-back restores a state the same
variable founded, which is the variable's current state unless it was
reset to an undeclared template in between (the round 3 note on the
copy-back ruling under "Rulings on edge cases").

**Not checked: `NumQubits`, `ParamNames`, `ParamStepCounts`.** They read
only `numQubits` and the elements inside the copy's own slice lengths.
After a copy the original can only append beyond those lengths (the copy
never writes, since its writers are refused), so the copy's accessors keep
describing the template as it was when copied, consistent with each
other. They return no error, so a check there could only panic, which
Decision 2 rules out, or return zero values that lie. They stay as they
are. (Amended 2026-09-10, round 1 fix: "can only append beyond those
lengths" holds unless the original is rolled back below the copy's length
by a copy-back and declares again; the copy-back ruling records that
residual. The accessors still stay as they are, for the reasons given.)
(Amended 2026-09-10, round 2 fix: the accessors now read through the
shared state pointer, so on a refused copy they report the template as it
is now, declarations the original has made since the copy included, and
`ParamNames` and `ParamStepCounts` on any value always describe the same
declarations. The round 1 residual is gone. The accessors' signatures are
untouched and they stay unguarded, for the reasons given; pinned by
`TestCopiedTemplateAccessorsReportSharedState`.) (Amended 2026-09-10,
round 3 fix: "the accessors" in the round 2 note means `ParamNames` and
`ParamStepCounts`. `NumQubits` returns `numQubits`, a plain field of the
`Template` value that is copied at copy time and never written after
construction; a copy's count agrees with the original's because it never
changes, not because it is read through the state pointer. The same
correction applies to the `Template` doc comment and the CHANGELOG
entry.)

**Pinned by the two writers, on an accepted declaration.** `t.addr = t`
runs in `AddParamGate` and `AddGate` immediately after `checkTargets`
succeeds and before the first mutation, so the pin and the first shared
state come into being in the same call. (Amended 2026-09-10, round 2 fix:
literally so. The pin is now the unexported `pin`, which sets `addr` and,
when `state` is nil, allocates the `templateState`; both writers call it
at the same point, so the two fields are always set together and a
rejected declaration touches neither.) A rejected declaration (nil
factory, no targets, out-of-range target) leaves the template untouched,
pin included, matching items 18 and 19's "nothing declared after a
rejection". `Bind` checks but never pins: pinning is a write, and `Bind`
is documented safe for concurrent calls once all `Add` calls complete;
two goroutines binding a template with no declarations would otherwise
race on `addr`. The `-race` run in Testing covers this. `NewTemplate`
does not pin either, so `*NewTemplate(n)` before any declaration is a
plain independent template, as the acceptance test's first line assumes.

**No `noescape` trick.** `strings.Builder` stores its address through
`abi.NoEscape` so that a stack-allocated `Builder` stays on the stack. A
plain `t.addr = t` makes the receiver of `AddParamGate` and `AddGate`
escape (`go build -gcflags=-m` reports `leaking param: t` for both after
the change, against `leaking param content: t` before), so a zero-value
`Template` declared as a local variable moves to the heap on its first
declaration. Every template built with `NewTemplate` is on the heap
already, templates are built once per run rather than per iteration, and
the trick needs `unsafe` and an internal package the module does not
import. Accepted as is. The escape is also what rules out address reuse:
a `Template` that has pinned itself is on the heap and kept reachable by
every copy's `addr`, so no later `Template` can come to occupy the pinned
address while a copy holding it exists, and the comparison in
`checkNotCopied` cannot pass a copy by coincidence. `strings.Builder`,
through `abi.NoEscape`, leaves that theoretical hole open; this design
does not. (Amended 2026-09-10, round 3 fix: the escape argument stands,
but what it closes is a contract hole rather than a correctness one.
Under the shared state a coincidental pass could at most let a copy
write the state it already shares, as an alias; `ParamNames`,
`ParamStepCounts`, and `Bind` would stay consistent on both values. See
Decision 3's round 3 amendment.)

## Rulings on edge cases

- **Copy of an undeclared template.** `var t Template; u := t`,
  `u := *NewTemplate(2)`, or a copy after only rejected declarations:
  `addr` is nil in both, `seen` is nil, the slices are nil; both go on as
  independent templates. Pinned by `TestCopyBeforeDeclarationIsIndependent`.
  (Amended 2026-09-10, round 3 fix: under the round 2 layout the fields
  are `addr` and `state`, both nil in both values; each founds a state of
  its own on its first accepted declaration. The ruling is unchanged.)
- **Copy after an accepted `AddGate` only.** `AddGate` pins too, since its
  `append` is enough to corrupt the other copy's step list. A copy taken
  after only fixed gates is refused like any other. (Amended 2026-09-10,
  round 3 fix: under the shared state there is no other copy's step list
  to corrupt. `AddGate` pins because a fixed gate is an accepted
  declaration and the state has to exist to hold it, and a copy taken
  after one is refused for the contract reason in Decision 3's round 3
  amendment.)
- **A copy that outlives the original.** A function that declares into a
  local `Template` and returns it by value hands back a copy whose `addr`
  points at the original, which the copy's `addr` keeps alive on the
  heap (Decision 3: the receiver escapes on its first declaration); every
  guarded method on the copy errors. This is
  the `strings.Builder` rule ("must not be copied after first use") and
  the reason the doc comment says to use a template in place or through
  a pointer. Return `*Template`, as `NewTemplate` does.
- **Refused calls change nothing.** The check runs before any write, so
  the copy's and the original's `ParamNames`, `ParamStepCounts`, and
  `steps` are as they were, and the original stays fully usable. Pinned
  by `TestCopiedTemplateIsRefusedByAddAndBind`.
- **The original is never affected by a copy.** The copy's writers are
  refused, so the shared map and arrays are written only through the
  original; the original's names and counts agree and its `Bind` demands
  every declared name. This is the acceptance test's assertion. (Amended
  2026-09-10, round 3 fix: the shared `templateState` is what is written
  only through the original; there is no map, and no arrays a copy holds
  a header over. The ruling is unchanged.)
- **A copy of a copy** carries the same foreign `addr` and is refused the
  same way.
- **A copy assigned back over the original.** Amended 2026-09-10 (round 1
  fix, `.superpowers/backlog/enhancement-backlog-2026-08-27/item-21-round-1-red.md`
  survivors 1 and 2). `b := a` after a declaration, then
  `a.AddParamGate("phi", Rx, 1)`, then `a = b` leaves `a` holding the
  header pair from before `phi` with `addr == &a`, so `checkNotCopied`
  passes; `strings.Builder`'s `copyCheck` is the same comparison and has
  the same hole. While `seen` was a shared map it still held `phi`, so
  `a.AddParamGate("phi", Rx, 1)` appended a step without listing the
  name, and `a.Bind(Params{"theta": 0.1, "phi": 0.2})` accepted `phi` as
  known while `ParamNames` omitted it: the item's desync, reached through
  the guard. The fix removes the map (Decision 1's amendment):
  `AddParamGate` and `Bind` read membership from `paramOrder`, the list
  the copy-back restored. A copy-back is therefore a snapshot restore: `a`
  holds the lists as they were when `b` was taken, every declaration made
  in between is dropped, parameter and fixed gates alike, and `a` goes on
  consistently from there. Detecting the copy-back instead was considered
  and rejected: `len(seen)` against `len(paramOrder)` would miss a
  copy-back that dropped only fixed gates or repeat declarations, a
  shared step counter compared with `len(steps)` would catch every drop,
  but survivor 2 asserts that `Bind` on the written-back `a` rejects
  `phi` with `UnknownParameterError`, the answer `a`'s own lists give,
  which any refusal contradicts; and dropping the intervening
  declarations is what assigning an older value means, not a desync, so
  it is outside the item's contract and is documented on the type rather
  than detected. Completeness of the fix: only the pinned original writes
  (every other copy is refused); it appends a step on every accepted
  declaration and a name on the first declaration of each, always at the
  index equal to its own length; so the writer's view is at every moment
  its own appends in order, every step it holds names a parameter its
  own list holds, and every listed name has a step below the length of
  any snapshot taken after that name's declaration. A copy-back restores
  both headers from one such snapshot, so the pair it restores is
  consistent, and the writer's later appends keep it so. Hence
  `ParamStepCounts` never lists a name `ParamNames` omits or vice versa,
  and `Bind` demands exactly the listed names and rejects any other, on
  the original and on any written-back value. Pinned by
  `TestCopyAssignedBackOverOriginalKeepsNamesAndCountsConsistent` and
  `TestCopyAssignedBackOverOriginalBindRejectsUnlistedName`. Residual:
  after a copy-back the writer appends into slots that a copy taken
  between the copy and the copy-back may still cover, when the slice had
  spare capacity, so that refused copy's `ParamNames` and
  `ParamStepCounts` can change under it and, if only one of the two
  slices was reallocated, disagree with each other. The copy's `Bind` is
  refused, and its accessors stay unguarded (Decision 3), so no binding
  is ever built from inconsistent lists; the `Template` doc comment
  states the residual. (Amended 2026-09-10, round 2 fix: the
  completeness argument above is wrong and the residual is superseded;
  see the next bullet. `TestCopyAssignedBackOverOriginalKeepsNamesAndCountsConsistent`
  still passes; the contract it pins, that a re-declaration after a
  copy-back is listed, counted, and demanded, holds under the new layout
  too, and its comment now says why. The other named test is replaced,
  as recorded under "Red test rulings".)
- **A copy assigned back over the original, twice.** Amended 2026-09-10
  (round 2 fix,
  `.superpowers/backlog/enhancement-backlog-2026-08-27/item-21-round-1-rereview.md`,
  finding I1). The argument above assumed that the slots below a
  snapshot's lengths still hold what they held when the snapshot was
  taken, which is true only while the writer's length never drops below
  them, and a copy-back is what drops it. Through the public API only:
  `a` declares `x`, `y`, `z` (both lists `len 3, cap 4`); `b := a`; `a`
  declares `x` again (a step only) and then `w` (name slot 3 = `w`, steps
  reallocate); `s := a` (names `len 4`, steps `len 5`); `a = b` (back to
  `len 3`); `a` declares `v`, which rewrites name slot 3 to `v` in the
  array `s` still covers; `a = s` restores `s`'s headers over the
  rewritten array. `a.ParamNames()` is `[x y z v]`,
  `a.ParamStepCounts()` is `map[w:1 x:2 y:1 z:1]`, and
  `a.Bind` with exactly the listed names builds a circuit with `w` at 0:
  item 21's own symptom on the pinned original, `addr == &a`, every guard
  passing. The same holds with a fixed `AddGate` in place of the repeat
  declaration. A `Template` whose value holds slice headers cannot keep
  the item's contract under copy-backs, because a copy-back restores
  headers and nothing can make the restored lengths agree with the
  array's contents.

  Fix: the `Template` value holds no headers. `steps` and `paramOrder`
  move into an unexported `templateState`, and `Template` holds it
  through one pointer field, `state`, allocated by `pin` on the first
  accepted declaration, in the same statement as `addr`. `NewTemplate(n)`
  is still `&Template{numQubits: n}`, so the zero value and every
  constructed template start with `state == nil`, which every reader
  treats as no declarations (`names` and `stepList` on a nil state return
  nil), and item 18's contract is unchanged: no allocation on the zero
  value until a declaration is accepted, nothing declared after a
  rejection, `NewTemplate(0)` and the zero value identical. The `addr`
  guard is exactly where it was. (Amended 2026-09-10, round 3 fix: where
  it is, which is unchanged, and not why it is there; Decision 3's round 3
  amendment states what it now enforces.)

  Completeness. Every write to `addr` or `state` is in `pin`, which runs
  only in a writer after `checkNotCopied` has passed, and a struct copy
  copies both fields together. So every `Template` value that exists
  holds either `(nil, nil)` or `(&X, S_X)`, where `S_X` is the one state
  `pin` allocated when the variable at `X` accepted its first
  declaration: if `addr` is nil then `state` is nil (they are only ever
  set together) and `pin` founds a new pair for the receiver; if `addr`
  is the receiver then `state` is already that receiver's, and `pin`
  changes nothing. A copy-back `X = v` therefore writes `(&X, S_X)` over
  `(&X, S_X)`: a no-op on both fields, whatever `v` holds and however
  many copies and copy-backs preceded it. (Amended 2026-09-10, round 3
  fix: `S_X` is not unique to the address `X`, so the no-op holds only
  while the variable at `X` has not been reset to an undeclared template
  in between; the round 3 note at the end of this bullet states the
  guarantee the code provides. The consistency conclusion in the rest of
  this paragraph is unaffected.) Only the value at address `X`
  passes the guard, so every write to `S_X` is an append by `X`'s own
  writers, through one code path that adds a step on every accepted
  declaration and a name exactly when the name is absent; `S_X`'s lists
  are at every moment the accepted declarations on `X` in order, with
  `paramOrder` the distinct parameter names among the steps in first-use
  order. Every reader, on the original or on any copy, reads `S_X`. Hence
  `ParamNames` and `ParamStepCounts` always describe the same
  declarations, `Bind` demands exactly the listed names and rejects any
  other, and no interleaving of by-value copies, copy-backs, and
  declarations drops, rewrites, or duplicates a declaration. In the
  two-step sequence above, `a`, `b`, and `s` all hold `(&a, S_a)`, both
  assignments are no-ops on state, and the five accepted declarations
  are all in `S_a`: `ParamNames` is `[x y z w v]`, `ParamStepCounts` is
  `map[v:1 w:1 x:2 y:1 z:1]`, `Bind` with those five succeeds, and `Bind`
  without any one of them is `MissingParameterError`. Pinned by
  `TestCopyAssignedBackTwiceKeepsNamesAndCountsConsistent`, both variants.

  Consequence for the round 1 copy-back ruling: a copy-back no longer
  drops the declarations made between the copy and the copy-back, because
  it restores nothing older than the original's own state. (Amended
  2026-09-10, round 3 fix: unless the original was reset to an undeclared
  template in between; see the round 3 note at the end of this bullet.
  The round 1 sequence below has no such reset, so its answers stand.) In the round 1
  sequence (`b := a` after `theta`, `a` declares `phi`, `a = b`), `a`
  still lists `phi`, `a.Bind(Params{"theta": 0.1, "phi": 0.2})` succeeds,
  and `a.Bind(Params{"theta": 0.1})` is `MissingParameterError`. Round
  1's survivor 2 asserted the opposite (`UnknownParameterError` for
  `phi`, the answer the restored headers gave); that expectation is the
  snapshot semantics this amendment replaces, so the moved test
  `TestCopyAssignedBackOverOriginalBindRejectsUnlistedName` is replaced by
  `TestCopyAssignedBackOverOriginalKeepsEveryDeclaration`, which pins the
  new answer and still checks that `Bind` rejects a name no declaration
  made. Round 1's "detecting the copy-back" discussion is moot: there is
  nothing to detect, since the copy-back changes nothing. The
  `Template` doc comment, `checkNotCopied`'s comment, and the CHANGELOG
  entry say so. Residual: none for consistency. A refused copy's
  accessors report the shared state (Decision 3's round 2 note), which is
  the template as it is now rather than as it was when copied.

  Amended 2026-09-10 (round 3 fix,
  `.superpowers/backlog/enhancement-backlog-2026-08-27/item-21-round-2-rereview.md`,
  "New Breakage in the Fix Diff", and the round 3 review's M1). The
  completeness argument above assumes that a variable founds at most one
  state, so that `S_X` names one state per address. It does not: `pin`
  allocates whenever `state` is nil, and a reset to an undeclared template
  clears both fields together, so a variable founds a new state every time
  it is reset. Through the public API only: `a := *NewTemplate(2)`;
  `a.AddParamGate("x", Ry, 0)` founds `S1`; `b := a`, which holds
  `(&a, S1)`; `a = *NewTemplate(2)`, back to `(nil, nil)`;
  `a.AddParamGate("y", Ry, 0)`, which founds `S2` at the same address;
  `a = b`, which passes the guard and installs `S1` over `S2`, dropping
  the declaration of `y`. So "a no-op on both fields, whatever `v` holds",
  "the copy-back changes nothing", and the doc comment's "drops nothing"
  are false as written.

  What the code provides, and what the `Template` doc comment, the `state`
  field's comment, `checkNotCopied`'s comment, and the CHANGELOG entry now
  state: a copy-back installs a state the same variable founded and is the
  sole writer of, so the restored value is always internally consistent
  and `ParamNames`, `ParamStepCounts`, and `Bind` always agree on it; it
  is a no-op whenever the variable has not been reset to an undeclared
  template in between, and otherwise it restores that variable's earlier
  state and drops what was declared into the later one. Two states never
  share backing arrays, since `pin` allocates `&templateState{}` with nil
  slices, so switching a variable between two of its own states presents
  one consistent list or the other, never a mixture. Item 21's contract is
  therefore untouched: a dropped declaration is what assigning an older
  value means, not a desync, the ruling the round 1 bullet above already
  made and the reason this stays documented rather than detected. The
  residual is a wording residual only: nothing about consistency changes.
- **Error precedence.** The copy error precedes the nil-factory,
  nil-gate, no-target, and range errors in the `Add` methods and the
  missing, non-finite, and unknown errors in `Bind`. Pinned for the
  nil-factory case. (Amended 2026-09-10, round 3 fix: pinned for the
  nil-factory case and, since round 1, for `Bind`'s missing and unknown
  cases too, at `parameterized/parameterized_test.go:519-523`.)
- **Concurrency.** The pin is a write inside `AddParamGate` and `AddGate`,
  which are already writes. `Bind`'s check is a read of `addr`, so "safe
  for concurrent reads after all `Add` calls complete" is unchanged;
  `Bind`'s doc comment says so. (Amended 2026-09-10, round 3 fix: the
  rule is unchanged but it now binds across copies. Before round 2 a
  refused copy's `ParamNames` read the copy's own header and elements the
  original never rewrote, since the original could only append beyond
  them; it now reads `state.paramOrder`, the header the original's
  `AddParamGate` writes, so a read on a copy concurrent with a
  declaration on the original is a data race. The documented rule already
  forbids it, so this is a scope note, not a defect; the `Template` doc
  comment says the rule covers reads on any copy.)
- **Zero value.** Item 18's contract holds: the zero value is the template
  `NewTemplate(0)` returns (`addr` is nil in both), no method panics on
  it, and `TestZeroValueTemplateIsAZeroQubitTemplate` runs every call on
  the same variable.

## Effect on callers

- `algorithm/h2.go` (`H2Ansatz`), `algorithm/qaoa.go` (`QAOATemplate`),
  `algorithm/vqe.go` (`evaluate`, `parameterShiftGradient`,
  `validateVQEStructure`, `VQE`), `internal/examples/qaoa.go`, and every
  test take or return `*parameterized.Template` or use a zero-value
  variable in place; none copies by value, so none changes behavior. Each
  `AddParamGate` and `AddGate` call now runs one nil-pointer comparison
  and one pointer store more; each `Bind` runs one comparison more.
  (Amended 2026-09-10, round 1 fix: each `AddParamGate` also scans the
  declared names instead of reading a map, and `Bind`'s accepted path
  replaces one map read per key with one length comparison. Amended
  2026-09-10, round 2 fix: each template also makes one heap allocation,
  the `templateState`, on its first accepted declaration, and every
  reader adds one pointer indirection.) The
  prototype's `go run ./cmd/quantum -demo qaoa` still prints
  `VQE from the symmetric start: energy = -1.0000, expected cut = 2.0000`
  followed by `iterations accepted: 29, energy evaluations: 378`.
- A `Template` that a future caller copies by value after a declaration
  now fails at its next `AddParamGate`, `AddGate`, or `Bind` with
  `errCopiedTemplate`; through `VQE` that surfaces as
  `InvalidVQEInputError` wrapping it.
- No Go file outside `parameterized/` changes. `CHANGELOG.md` gains an
  entry.

## Backward compatibility (ADR-style note)

Inputs that were accepted and now error: `AddParamGate`, `AddGate`, or
`Bind` called on a by-value copy of a `Template` taken after an accepted
declaration. Nothing used through a pointer or in place behaves
differently. No signature changes; the new field and helper are
unexported. Per `docs/compatibility-policy.md` this is a personal project
with no external consumers; every internal caller is listed above and
unaffected.

- **CHANGELOG:** following items 15 through 20, an Unreleased entry under
  the existing `### Fixed` heading, appended after item 20's entry
  (entries within a section are appended in landing order).
- **ADR-0010:** no amendment. Its decision (template materialization over
  symbolic gates) and consequences (validation at `Bind`; QAOA reuses the
  template) are untouched; how a template detects its own misuse is below
  the ADR's level.
- **The 2026-08-30 spec** (`docs/superpowers/specs/2026-08-30-parameter-binding-vqe-design.md`)
  lists `NewTemplate(numQubits int) *Template` and never mentions value
  copies; nothing in it becomes false. No amendment.
- **Item 18's spec:** its "Copying a `Template` by value" ruling says the
  copies share state and that it is out of scope there; this design is
  that scope. Its Decision 1 cites `strings.Builder` as the zero-value
  precedent, and this design follows the same type's copy rule. No
  amendment. (Amended 2026-09-10, round 2 fix: its zero-value contract is
  re-checked against the new layout in the "twice" ruling above, and
  `TestZeroValueTemplateAddParamGateDoesNotPanic` and
  `TestZeroValueTemplateIsAZeroQubitTemplate` pass unchanged; still no
  amendment.)
- **Item 19's plan** (`docs/superpowers/plans/2026-09-09-empty-target-list.md`,
  the task's "Produces" note): says, for item 21, that the `seen`,
  `paramOrder`, and `steps` handling is byte for byte what item 18 left
  it; item 19's spec says the same in its Decision 1 context without
  naming item 21. This design adds a pin line before that handling and a
  check before the argument checks. (Amended 2026-09-10, round 1 fix: the
  round 1 fix then replaced the `seen` handling with a scan of
  `paramOrder`; plans are historical records and are not amended, and
  item 19's spec makes no claim that becomes false.)

## Red test rulings

`TestRedCopyAfterDeclarationKeepsNamesAndCountsConsistent` moves into
`parameterized/parameterized_test.go` as
`TestCopyAfterDeclarationKeepsNamesAndCountsConsistent`, the package's
`Test<Subject><Verb>...` style. Its assertions were checked against the
contract:

- `a := *parameterized.NewTemplate(2)` then `a.AddParamGate("theta", Ry, 0)`
  must succeed: the copy precedes any declaration, so `addr` is nil and
  `a` is an independent template (Decision 3). Consistent; unchanged.
- `b := a` then `b.AddParamGate("phi", Rx, 1)` must succeed (`t.Fatal(err)`
  on error): **ruled against the contract and rewritten.** The red test
  was written to reproduce the desync, so it drives the copy and expects
  the drive to be accepted. Under Decision 1 that call is the misuse the
  guard exists to catch, and accepting it is what let the desync happen;
  the contract says it returns an error. The moved test asserts
  `err == nil` is a failure, with a message saying the copy shares the
  original's bookkeeping. This is the one expectation adjusted, and the
  dispatch authorized exactly this adjustment for a copy-guard ruling.
- `a.AddParamGate("phi", Rx, 1)` must succeed: `a` is the pinned original.
  Consistent; unchanged.
- Every name in `a.ParamStepCounts()` appears in `a.ParamNames()`: the
  copy never wrote to `seen`, so `a`'s second declaration is new to it
  and is appended to `paramOrder`. Consistent; unchanged, byte for byte.
- `a.Bind(Params{"theta": 0.1})` returns `MissingParameterError`: `phi`
  is in `a.paramOrder`. Consistent; unchanged, byte for byte.

The leading comment is rewritten to describe the contract. The moved test
says nothing about the error's text, about `AddGate` or `Bind` on a copy,
about the accessors on a copy, or about copies taken before a
declaration, so two tests pin those alongside it rather than editing the
moved test further:

- `TestCopiedTemplateIsRefusedByAddAndBind`: on `c := *orig` after one
  declaration, `AddParamGate`, `AddParamGate` with a nil factory,
  `AddGate`, and `Bind` each return an error containing `copied by value`;
  `c.NumQubits()`, `c.ParamNames()`, `c.ParamStepCounts()` are `2`,
  `[theta]`, `map[theta:1]`; `orig.ParamNames()` is still `[theta]`, and
  `orig.AddGate` and `orig.Bind` succeed.
- `TestCopyBeforeDeclarationIsIndependent`: `fresh := *NewTemplate(2)`,
  `twin := fresh`, each declares a different name and each lists only its
  own; `twin.Bind` succeeds; a copy taken after only a rejected
  declaration (nil factory) accepts a declaration. Passes before and
  after the change; it pins the half of the contract the code already
  honors so a later change to the pin site cannot move it silently.

With item 21's test gone, `parameterized/backlog_red_test.go` holds only
item 22's test, which uses `testing`, `circuit`, `gates`, `parameterized`,
and `state`; the `errors` import, used only by item 21's test, is removed
with it. Item 22's test and comment are not edited, and the file's header
comment is unchanged.

Amended 2026-09-10 (round 1 fix): round 1's black-box red testing left
two survivors in `parameterized/backlog_item21_red_test.go`, both on the
copy-back sequence (ruling above). Both are in scope and both moved into
`parameterized/parameterized_test.go` with their assertions intact,
renamed to the package's style by dropping the `Item21` prefix:
`TestCopyAssignedBackOverOriginalKeepsNamesAndCountsConsistent` (an
accepted re-declaration after the copy-back must be listed, counted, and
demanded; a refusal would also satisfy it, but under the fix the
declaration is accepted and the assertions run) and
`TestCopyAssignedBackOverOriginalBindRejectsUnlistedName` (`Bind` on the
written-back value rejects the dropped name as `UnknownParameterError`).
The survivor file is deleted.

Amended 2026-09-10 (round 2 fix): one expectation from those survivors is
changed, and it is the only test expectation this round changes.
Survivor 2 (`TestCopyAssignedBackOverOriginalBindRejectsUnlistedName`)
asserted that after `b := a`, `a` declares `phi`, `a = b`, `ParamNames`
is `[theta]` and `Bind` with `phi` is `UnknownParameterError`. Under the
shared state the copy-back drops nothing, so `ParamNames` is
`[theta phi]` and that `Bind` succeeds; the "twice" ruling above records
why (the survivor's expectation was the snapshot semantics that the
two-step hole shows cannot be kept). The test is replaced by
`TestCopyAssignedBackOverOriginalKeepsEveryDeclaration`, which keeps the
sequence and pins the new answers: `ParamNames` is `[theta phi]`,
`ParamStepCounts` is `map[phi:1 theta:1]`, `Bind` with both succeeds,
`Bind` without `phi` is `MissingParameterError` for `phi`, and `Bind`
with an undeclared name is still `UnknownParameterError`. Survivor 1
(`TestCopyAssignedBackOverOriginalKeepsNamesAndCountsConsistent`) passes
unchanged: its re-declaration of `phi` after the copy-back is accepted as
a second step of a listed name, and its assertions hold.

## Testing

All in `parameterized/parameterized_test.go` (package `parameterized_test`):

- `TestCopyAfterDeclarationKeepsNamesAndCountsConsistent`: the moved
  test. Fails today at the rewritten expectation with
  `AddParamGate on a by-value copy taken after a declaration succeeded, want an error: the copy shares the original's bookkeeping`.
- `TestCopiedTemplateIsRefusedByAddAndBind`: fails today at its first
  assertion with
  `AddParamGate on a copy: err = <nil>, want an error mentioning "copied by value"`.
- `TestCopyBeforeDeclarationIsIndependent`: passes before and after.
- Amended 2026-09-10 (round 1 fix):
  `TestCopyAssignedBackOverOriginalKeepsNamesAndCountsConsistent` and
  `TestCopyAssignedBackOverOriginalBindRejectsUnlistedName`, the moved
  survivors; both failed against the guard alone (the first at the
  names-versus-counts loop and the `Bind` assertion, the second with
  `err = <nil>`) and pass with the map removed.
  `TestCopiedTemplateIsRefusedByAddAndBind` also gained two `Bind` calls
  on the copy, one with an empty binding and one with an undeclared
  name, expecting the copy error ahead of `MissingParameterError` and
  `UnknownParameterError` (round 1 review, minor 4).
- Amended 2026-09-10 (round 2 fix):
  `TestCopyAssignedBackTwiceKeepsNamesAndCountsConsistent`, the
  re-review's two-step sequence as two subtests (`repeat declaration`
  and `fixed gate`), asserting the exact name list `[x y z w v]`, that
  names and counts list each other, the count for `x`, and `Bind`'s
  three answers (exact names succeed, each omission is
  `MissingParameterError` for that name, an extra name is
  `UnknownParameterError`); failed at c9083d9 with
  `ParamNames() after two copy-backs = ["x" "y" "z" "v"], want ["x" "y" "z" "w" "v"]`
  in both subtests. `TestCopyAssignedBackOverOriginalKeepsEveryDeclaration`
  replaces survivor 2 as recorded under "Red test rulings".
  `TestCopiedTemplateAccessorsReportSharedState` pins Decision 3's round
  2 note. `TestCopiedTemplateIsRefusedByAddAndBind`,
  `TestCopyBeforeDeclarationIsIndependent`, and both zero-value tests
  pass unchanged.
- Amended 2026-09-10 (round 3 fix, the round 3 review's M4):
  `TestParamAccessorsReturnFreshContainers` pins that the two accessors
  return containers a caller cannot use to reach the shared state. It
  overwrites the returned slice's first element and appends to it,
  rewrites and deletes entries in the returned map, does both again
  through a refused copy, then re-reads both accessors on the original
  and on the copy and re-checks `Bind`'s three answers. This mattered
  less while the lists lived in the value; now that every copy reads one
  state, an accessor handing out `state.paramOrder` would let the holder
  of a refused copy rewrite the original's names. Verified to catch that
  regression: with `ParamNames` returning `t.state.names()` directly it
  fails with
  `ParamNames()[0] after mutating a returned slice = "hacked through a copy", want "theta"`.
  The accessors are unchanged; this round adds the test only.
- `go test -race ./parameterized ./algorithm` clean: `Bind`'s check is a
  read. (Round 2: `Bind` reads the `state` pointer and then the lists,
  still without writing; the run stays clean.)
- `go test -tags redtests ./parameterized -run '^TestRed'` must list
  exactly `TestRedDeclaredTargetsAreNotAliasedToCallerSlice` (item 22) as
  failing; `algorithm` has no tagged tests left (`no tests to run`).

Prototype: every code and test change in the plan was applied to a
scratch copy of the repository at `d52e18f`; the moved test and the
refusal test failed as stated against the original methods, all
seventeen tests in the package passed after the change, and `gofmt -l`,
`go vet` (plain and `-tags redtests` on `parameterized` and `algorithm`),
`staticcheck`, `go build`, `go test ./...`,
`go test -race ./parameterized ./algorithm`, the red-tag run, and the
QAOA demo were all as stated.

## Out of scope

Item 22 (declared targets alias the caller's slice; its red test stays
tagged and still fails after this item, since the `append` line that
stores `targets` is not edited). A `Clone` method for callers who want a
second template from the first (nobody in the module does; declare
twice). The `noescape` optimization (Decision 3). Typing or exporting the
copy error. Any change to `NumQubits`, `ParamNames`, `ParamStepCounts`,
`checkTargets`, `NewTemplate`, or any exported signature. (Round 2 fix:
`ParamNames` and `ParamStepCounts` read through the state pointer; their
signatures, `NumQubits`, `checkTargets`, and `NewTemplate` are unchanged.)
