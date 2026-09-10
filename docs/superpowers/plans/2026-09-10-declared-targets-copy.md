# Declared Targets Copied Out of the Caller's Slice (Backlog Item 22) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close backlog item 22 by making `parameterized.Template`'s two writers store their own copy of the target list, so mutating the slice a caller spread into `AddParamGate` or `AddGate` after a successful declaration can no longer change what `Bind` builds.

**Architecture:** One unexported helper, `cloneTargets(targets []int) []int`, returning `append([]int(nil), targets...)`, the same expression `circuit.AddGate` uses on the same argument one layer down. Each writer calls it in its `step` literal, after `checkNotCopied`, its argument checks, `checkTargets`, and `pin` have all succeeded, so a rejected declaration still allocates nothing and the copy is made exactly where the value enters the `templateState` that item 21 gave every copy of a template to share. `Bind`, `checkTargets`, the accessors, and every exported signature are untouched.

**Tech Stack:** Go standard library only. No new dependencies.

**Spec:** `docs/superpowers/specs/2026-09-10-declared-targets-copy-design.md`

## Classification

**Standard.** One new unexported helper and one changed expression in each of the two writers, plus three doc comments; no signature, exported type, or cross-package change. One task, one test cycle.

## Global Constraints

- Never edit a failing test to make code pass. The moved red test is the item's acceptance test; its setup, its calls, and all of its assertions are copied byte for byte, and only its leading comment and its name change (spec, "Red test rulings"). No assertion was ruled against the contract, so none is adjusted. If it fails after implementation, the implementation is wrong. Report it.
- The copy is made on the way in, in both writers, never in `Bind` (spec Decision 1). A `Bind`-side copy copies the already-mutated slice and fixes nothing.
- Copy placement: inside the `step` literal of the `append` in `AddParamGate` and `AddGate`, after `checkNotCopied`, the argument checks, `checkTargets`, and `pin` (spec Decision 2). A rejected declaration must still allocate nothing.
- The copy expression is `append([]int(nil), targets...)`, inside one unexported helper `cloneTargets`, matching `circuit.AddGate` (`circuit/circuit.go:79-82`). Do not use `slices.Clone`; no file in this repository imports `slices` (spec Decision 2).
- `Bind`, `checkTargets`, `checkNotCopied`, `pin`, `declared`, `names`, `stepList`, `NewTemplate`, `NumQubits`, `ParamNames`, `ParamStepCounts`, the `Template` doc comment, and every exported signature are not edited. Item 21's guard and pin lines in both writers are not moved or reworded.
- Doc-comment voice: explain the why, cite conventions (`circuit.AddGate`, the caller's slice), no filler. Every comment in this plan is verbatim; do not reword it.
- `parameterized/backlog_red_test.go` is deleted in this task: after the move it holds no tests, and a tagged file with only a header comment documents nothing, the same ruling item 17's plan made for `algorithm/backlog_red_test.go`. No Go file in the repository then carries the `redtests` tag. Nothing in the tooling invokes the tag (`.github/workflows/ci.yml`, `.github/workflows/fuzz.yml`, and `.superpowers/backlog/enhancement-backlog-2026-08-27/gate.sh` all run untagged; there is no Makefile), so no CI step or QA gate changes. `go vet -tags redtests ./parameterized ./algorithm` must still pass on packages with no tagged file, and `go test -tags redtests ./parameterized ./algorithm -run '^TestRed'` then reports `[no tests to run]` for both; that is the expected outcome, not a failure.
- No new dependencies. Go standard library only.
- The task ends with `gofmt -l .` printing nothing, `go vet ./...`, `staticcheck ./...`, `go build ./...`, `go test ./...` green, `go test -race ./parameterized ./algorithm` green, `go vet -tags redtests ./parameterized ./algorithm` compiling, then a commit whose message ends with a blank line and the trailers `Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>` and `Claude-Session: https://claude.ai/code/session_01XMPFmYskZD8uji6xSh5x11`.
- Do not edit `.superpowers/backlog/` or `docs/enhancement-backlog-2026-08-27.md`; the run lead ticks the item after the QA gate.

---

### Task 1: Store a copy of the declared target list in `AddParamGate` and `AddGate`

**Sizing estimate:** about 900 lines of existing code to read (`parameterized/parameterized.go` 361, `parameterized/backlog_red_test.go` 61, `parameterized/parameterized_test.go` lines 1-16 and 826-887 (about 80), `circuit/circuit.go` lines 44-84 (about 40), the spec in full (about 300), `CHANGELOG.md` lines 145-200 (about 56)); 2 non-test files modified (`parameterized/parameterized.go` +27/-8, `CHANGELOG.md` +18) plus 2 test files (`parameterized/parameterized_test.go` +90, `parameterized/backlog_red_test.go` deleted, 61 lines); net diff about +66 lines; one test cycle. Under a quarter of a context window.

**Files:**
- Modify: `parameterized/parameterized.go:40-45` (the `step` struct: a field comment on `targets`), `parameterized/parameterized.go:218-225` (after `checkTargets`: the new `cloneTargets` helper), `parameterized/parameterized.go:227-235` and `:253` (`AddParamGate`: doc comment and the `append` expression), `parameterized/parameterized.go:257-266` and `:281` (`AddGate`: doc comment and the `append` expression). Line numbers are at `8ee8579`, before any edit; later anchors shift as earlier edits land, so match on text.
- Modify: `CHANGELOG.md` (insert one entry at the end of `### Fixed` under `## Unreleased`, directly before the first `### Breaking`, line 200 at `8ee8579`)
- Test: `parameterized/parameterized_test.go` (append two tests at the end of the file after `TestParamAccessorsReturnFreshContainers`, which ends at line 887; no import change: `circuit`, `gates`, `state`, and `parameterized` are already imported)
- Delete: `parameterized/backlog_red_test.go` (61 lines; the last `redtests` file in the repository)

**Interfaces:**
- Consumes (existing, unchanged):
  - `type step struct { param string; factory Factory; gate quantum.Gate; targets []int }` in `parameterized/parameterized.go:40-45` (gains a field comment only; no field is added, removed, or renamed)
  - `func (t *Template) AddParamGate(name string, factory Factory, targets ...int) error` in `parameterized/parameterized.go:236` (signature unchanged; its `append` at line 253 gains the copy)
  - `func (t *Template) AddGate(gate quantum.Gate, targets ...int) error` in `parameterized/parameterized.go:267` (signature unchanged; its `append` at line 281 gains the copy)
  - `func (t *Template) checkTargets(targets []int) error` in `parameterized/parameterized.go:218` (not modified; the helper is defined after it)
  - `func (t *Template) checkNotCopied() error` and `func (t *Template) pin()` in `parameterized/parameterized.go:152` and `:164` (item 21's guard and pin; not modified, not moved)
  - `func (t *Template) Bind(values Params) (*circuit.Circuit, error)` in `parameterized/parameterized.go:291` (not modified; it forwards `s.targets...` to `circuit.AddGate` at line 325)
  - `func (c *Circuit) AddGate(gate quantum.Gate, targets ...int) error` in `circuit/circuit.go:44` (not modified; its `Operation` literal at lines 79-82 already stores `append([]int(nil), targets...)`, so the way out is already safe, spec Decision 4)
  - `circuit.New`, `gates.NewRy`, `gates.NewRx`, `gates.NewCNOT`, `state.New`, `parameterized.Ry`, `parameterized.Rx`, `parameterized.Params` (used by the tests)
- Produces (new, unexported):
  - `func cloneTargets(targets []int) []int` in `parameterized/parameterized.go`, returning `append([]int(nil), targets...)`
  - No later task.

- [ ] **Step 1: Read the context**

1. `parameterized/parameterized.go` in full. Note the `step` type at lines 35-45 (four fields; `targets []int` is the one that aliases), `checkTargets` at 218-225 (a pure range check returning an error and no value, which is why the copy does not go there), `AddParamGate` at 227-255 (doc comment, `checkNotCopied`, nil-factory check, no-target check, `checkTargets`, `pin`, the `paramOrder` append, then the `steps` append at line 253 that stores `targets`), `AddGate` at 257-283 (the same shape, its `steps` append at line 281), and `Bind` at 291-330 (its `c.AddGate(gate, s.targets...)` at line 325). The two `append` lines are the whole defect.
2. `circuit/circuit.go` lines 44-84: `AddGate`'s checks and its `Operation` literal, `Targets: append([]int(nil), targets...)`. That is the expression you copy and the precedent the spec cites; it is also why nothing needs to change on the way out.
3. `parameterized/backlog_red_test.go` in full (61 lines): the header comment, the import block, and the one test you are moving. After the move the file holds no tests and is deleted.
4. `parameterized/parameterized_test.go` lines 1-16 (imports: `errors`, `math`, `strings`, `testing`, `circuit`, `gates`, `internal/sparsestate`, `parameterized`, `quantum`, `state`, all present, so no import change) and lines 826-887 (`TestParamAccessorsReturnFreshContainers`, the last test; the file ends there and you append after it).
5. The spec, in full: Decision 1 says why the writers copy rather than the doc comments warning, and why a `Bind`-side copy fixes nothing; Decision 2 says where the copy goes, why it is a helper, and why not `slices.Clone`; Decision 3 gives the doc-comment sentences; Decision 4 rules the way out already safe; "Red test rulings" says the moved test is copied byte for byte and what happens to the `redtests` tag.

- [ ] **Step 2: Move the red test, delete the red file, and add the second test**

Delete the whole file `parameterized/backlog_red_test.go`:

```bash
git rm parameterized/backlog_red_test.go
```

In `parameterized/parameterized_test.go`, append to the end of the file (after the closing brace of `TestParamAccessorsReturnFreshContainers`). The first function is the moved red test: same template, same declarations, same mutation, same `Bind`, same manual circuit, same amplitude loop, renamed to the package's style, with only its leading comment rewritten. The second pins the contract on `AddGate` and on the reused-slice pattern the item names.

```go

// TestDeclaredTargetsAreNotAliasedToCallerSlice pins that a declaration
// keeps its own copy of the target list (backlog item 22). Before the fix
// both writers stored the caller's variadic slice by reference, so
// mutating it after a call that had returned success changed what Bind
// later built, although the declaration looked complete.
func TestDeclaredTargetsAreNotAliasedToCallerSlice(t *testing.T) {
	tmpl := parameterized.NewTemplate(2)
	targets := []int{0}
	if err := tmpl.AddParamGate("theta", parameterized.Ry, targets...); err != nil {
		t.Fatal(err)
	}
	targets[0] = 1
	bound, err := tmpl.Bind(parameterized.Params{"theta": 0.2})
	if err != nil {
		t.Fatal(err)
	}
	// As declared: Ry(0.2) on qubit 0, nothing on qubit 1.
	manual, _ := circuit.New(2)
	_ = manual.AddGate(gates.NewRy(0.2), 0)
	sa, _ := state.New(2)
	sb, _ := state.New(2)
	if err := bound.Execute(sa); err != nil {
		t.Fatal(err)
	}
	if err := manual.Execute(sb); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		if sa.Amplitude(i) != sb.Amplitude(i) {
			t.Fatalf("amplitude %d after mutating the caller's slice: got %v, want %v (Ry on qubit 0 as declared)", i, sa.Amplitude(i), sb.Amplitude(i))
		}
	}
}

// TestAddCopiesTheCallerTargetsSlice pins the contract on both writers and
// on the pattern the item names: one slice built once and reused across
// declarations. A mutation after an accepted declaration cannot rewrite,
// invalidate, or duplicate the targets that declaration recorded, so the
// range check the writers run at declaration time still describes what
// Bind builds.
func TestAddCopiesTheCallerTargetsSlice(t *testing.T) {
	tmpl := parameterized.NewTemplate(2)
	targets := []int{0}
	if err := tmpl.AddParamGate("theta", parameterized.Ry, targets...); err != nil {
		t.Fatal(err)
	}
	// The caller reuses one slice for the next declaration, as a generator
	// emitting a layer of rotations does.
	targets[0] = 1
	if err := tmpl.AddParamGate("phi", parameterized.Rx, targets...); err != nil {
		t.Fatal(err)
	}
	pair := []int{0, 1}
	if err := tmpl.AddGate(gates.NewCNOT(), pair...); err != nil {
		t.Fatal(err)
	}
	// Mutations that would make a declaration illegal if Bind saw them: a
	// duplicate target for the CNOT, an out-of-range target for the Rx.
	pair[1] = 0
	targets[0] = 7

	bound, err := tmpl.Bind(parameterized.Params{"theta": 0.2, "phi": 0.3})
	if err != nil {
		t.Fatalf("Bind after mutating both caller slices: %v, want the circuit as declared", err)
	}
	manual, err := circuit.New(2)
	if err != nil {
		t.Fatal(err)
	}
	if err := manual.AddGate(gates.NewRy(0.2), 0); err != nil {
		t.Fatal(err)
	}
	if err := manual.AddGate(gates.NewRx(0.3), 1); err != nil {
		t.Fatal(err)
	}
	if err := manual.AddGate(gates.NewCNOT(), 0, 1); err != nil {
		t.Fatal(err)
	}
	sa, _ := state.New(2)
	sb, _ := state.New(2)
	if err := bound.Execute(sa); err != nil {
		t.Fatalf("executing the bound circuit: %v", err)
	}
	if err := manual.Execute(sb); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		if sa.Amplitude(i) != sb.Amplitude(i) {
			t.Fatalf("amplitude %d after mutating both caller slices: got %v, want %v (Ry on 0, Rx on 1, CNOT 0->1 as declared)", i, sa.Amplitude(i), sb.Amplitude(i))
		}
	}
}
```

- [ ] **Step 3: Run the tests to verify they fail**

Run:

```bash
go test ./parameterized -run '^Test(DeclaredTargets|AddCopies)' -v 2>&1 | grep -E '^(=== RUN|--- |ok|FAIL)|parameterized_test'
go vet -tags redtests ./parameterized ./algorithm
```

Expected: `TestDeclaredTargetsAreNotAliasedToCallerSlice` fails with `amplitude 1 after mutating the caller's slice: got (0+0i), want (0.09983341664682815+0i) (Ry on qubit 0 as declared)`. `TestAddCopiesTheCallerTargetsSlice` fails at `Bind` with `Bind after mutating both caller slices: qubit index 7 is out of range [0,1], want the circuit as declared`. The `go vet` under the `redtests` tag compiles cleanly on two packages with no tagged file left.

- [ ] **Step 4: Implement the copy**

Six edits in `parameterized/parameterized.go`, in file order.

(a) Replace the tail of the `step` struct (lines 43-45), that is, exactly

```go
	gate    quantum.Gate
	targets []int
}
```

with

```go
	gate    quantum.Gate
	// targets is the template's own copy of the declared target list, made
	// by cloneTargets; it is never the caller's variadic slice.
	targets []int
}
```

This block occurs once in the file.

(b) Add the helper after `checkTargets`. Replace, exactly

```go
func (t *Template) checkTargets(targets []int) error {
	for _, target := range targets {
		if target < 0 || target >= t.numQubits {
			return &quantum.QubitsOutOfRangeError{Index: target, MaxIndex: t.numQubits - 1}
		}
	}
	return nil
}
```

with

```go
func (t *Template) checkTargets(targets []int) error {
	for _, target := range targets {
		if target < 0 || target >= t.numQubits {
			return &quantum.QubitsOutOfRangeError{Index: target, MaxIndex: t.numQubits - 1}
		}
	}
	return nil
}

// cloneTargets returns the template's own copy of a declared target list.
// The variadic targets a writer receives are the caller's slice, and a
// caller that builds one slice and reuses it across declarations, as a
// generated circuit does, would otherwise rewrite steps already declared:
// the call returned success and the declaration looked complete, yet Bind
// would build something else, and the range check the writer just ran
// would no longer describe it. circuit.AddGate copies the same argument
// for the same reason (the Operation literal in circuit/circuit.go), so a
// target list is copied once on the way in here and once on the way out
// there. The writers call this only after their checks pass, so a rejected
// declaration allocates nothing.
func cloneTargets(targets []int) []int {
	return append([]int(nil), targets...)
}
```

(c) Replace the tail of `AddParamGate`'s doc comment, that is, exactly

```go
// circuit.AddGate to reject at Bind. On a Template copied by value after
// an accepted declaration the call is refused before any of these checks
// (see Template).
func (t *Template) AddParamGate(name string, factory Factory, targets ...int) error {
```

with

```go
// circuit.AddGate to reject at Bind. The targets are copied, so the slice
// a caller passed may be reused or mutated after the call without changing
// what was declared. On a Template copied by value after an accepted
// declaration the call is refused before any of these checks (see
// Template).
func (t *Template) AddParamGate(name string, factory Factory, targets ...int) error {
```

(d) Replace `AddParamGate`'s `append`, that is, exactly

```go
	t.state.steps = append(t.state.steps, step{param: name, factory: factory, targets: targets})
```

with

```go
	t.state.steps = append(t.state.steps, step{param: name, factory: factory, targets: cloneTargets(targets)})
```

This line occurs once in the file. The `t.pin()`, `t.declared(name)`, and `paramOrder` lines above it are not edited.

(e) Replace the tail of `AddGate`'s doc comment, that is, exactly

```go
// panics when this method calls gate.Name() to name the gate in the
// no-target error. On a Template copied by value after an accepted
// declaration the call is refused before any of these checks (see
// Template).
```

with

```go
// panics when this method calls gate.Name() to name the gate in the
// no-target error. The targets are copied, so the slice a caller passed
// may be reused or mutated after the call without changing what was
// declared. On a Template copied by value after an accepted declaration
// the call is refused before any of these checks (see Template).
```

(f) Replace `AddGate`'s `append`, that is, exactly

```go
	t.state.steps = append(t.state.steps, step{gate: gate, targets: targets})
```

with

```go
	t.state.steps = append(t.state.steps, step{gate: gate, targets: cloneTargets(targets)})
```

This line occurs once in the file. `Bind`, `checkTargets`, `checkNotCopied`, `pin`, `declared`, `names`, `stepList`, `NewTemplate`, `NumQubits`, `ParamNames`, `ParamStepCounts`, and the `Template` doc comment are not edited.

- [ ] **Step 5: Run the tests to verify they pass**

Run:

```bash
go test ./parameterized -v 2>&1 | grep -E '^(--- |ok|FAIL)'
```

Expected: every test in the package reports `--- PASS`, twenty-four in all, including `TestDeclaredTargetsAreNotAliasedToCallerSlice`, `TestAddCopiesTheCallerTargetsSlice`, `TestAddRejectsNoTargets` and `TestAddNoTargetsErrorNamesTheGateAndDeclaresNothing` (item 19: the copy is after those guards, so nothing is declared or allocated on a rejection), and the eight copy-guard tests from item 21, then `ok`.

- [ ] **Step 6: Record the change in the CHANGELOG**

In `CHANGELOG.md`, under `## Unreleased`, `### Fixed`, insert a new entry directly before the first `### Breaking` heading (line 200 at `8ee8579`). The three lines immediately above the insertion point are the end of item 21's entry:

```markdown
Nothing in the module copies a `Template` by value, so no caller changes.
Design: `docs/superpowers/specs/2026-09-10-template-copy-guard-design.md`.

### Breaking
```

Replace that with:

```markdown
Nothing in the module copies a `Template` by value, so no caller changes.
Design: `docs/superpowers/specs/2026-09-10-template-copy-guard-design.md`.

#### `parameterized.Template` copies the targets a declaration is given

`AddParamGate` and `AddGate` stored the caller's variadic `targets` slice
by reference, so mutating it after a call that had returned success
changed what `Bind` later built. A caller that fills one target slice and
reuses it across declarations, the usual shape of a generated circuit, got
gates on the targets written last rather than on the ones each declaration
named, and the range check the declaration had passed no longer described
the result. Both writers now store their own copy, as `circuit.AddGate`
already does with the same argument, so an accepted declaration is fixed;
the copy is made only after the argument checks pass, so a rejected
declaration still allocates nothing. Nothing in the module mutates a
target slice after declaring with it, so no caller changes. With this the
last red test moves into the regular suite and
`parameterized/backlog_red_test.go` is removed, so no test carries the
`redtests` build tag any more. Design:
`docs/superpowers/specs/2026-09-10-declared-targets-copy-design.md`.

### Breaking
```

- [ ] **Step 7: Full verification**

Run:

```bash
gofmt -l .
go vet ./...
go vet -tags redtests ./parameterized ./algorithm
staticcheck ./...
go build ./...
go test ./...
go test -race ./parameterized ./algorithm
go test -tags redtests ./parameterized ./algorithm -run '^TestRed' 2>&1 | grep -E '^(--- FAIL|ok|FAIL)'
go run ./cmd/quantum -demo qaoa | grep -A1 'VQE from'
git status --short
```

Expected: `gofmt -l` prints nothing; vet (both), staticcheck, build, the full suite, and the race run are clean; the red run prints one `ok ... [no tests to run]` line per package and no `--- FAIL` line, since no file carries the tag any more; the demo prints `VQE from the symmetric start: energy = -1.0000, expected cut = 2.0000` and then `  iterations accepted: 29, energy evaluations: 378`; `git status` lists exactly the four files named under **Files** (`parameterized/parameterized.go`, `CHANGELOG.md`, `parameterized/parameterized_test.go` modified, `parameterized/backlog_red_test.go` deleted) and nothing else. If a red test still runs, if the demo lines differ, or if any other file appears, stop and report: something outside this item changed.

- [ ] **Step 8: Commit**

```bash
git add parameterized/parameterized.go parameterized/parameterized_test.go parameterized/backlog_red_test.go CHANGELOG.md
git commit -m "fix(parameterized): copy the targets a declaration is given

AddParamGate and AddGate stored the caller's variadic targets slice by
reference, so a caller that filled one slice and reused it across
declarations, the usual shape of a generated circuit, rewrote steps that
had already been accepted: Bind built gates on the targets written last,
and the range check each declaration had passed no longer described the
result. Both writers now store cloneTargets(targets), the same
append([]int(nil), targets...) that circuit.AddGate already applies to
the same argument one layer down, so a target list is copied once on the
way in and once on the way out. The copy sits after the guard, the
argument checks, checkTargets, and the pin, so a rejected declaration
still allocates nothing. Bind, checkTargets, the accessors, and every
exported signature are unchanged.

TestRedDeclaredTargetsAreNotAliasedToCallerSlice moves into the regular
suite as TestDeclaredTargetsAreNotAliasedToCallerSlice with its
assertions intact, joined by TestAddCopiesTheCallerTargetsSlice for
AddGate and the reused-slice pattern. That empties
parameterized/backlog_red_test.go, which is removed; no file in the
repository carries the redtests tag now, and no CI step or QA gate used
it. Backlog item 22.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01XMPFmYskZD8uji6xSh5x11"
```

If the commit fails with `.git/index.lock` present, another agent is committing; wait a few seconds and run the `git commit` again unchanged.

---

## Self-Review

- **Spec coverage.** Decision 1 (the writers copy; document-only, a `Bind`-side copy, and freezing the step list rejected): Step 4b, 4d, and 4f make the copy in the two writers, and the Global Constraints forbid the `Bind` site; pinned by both tests in Step 2. Decision 2 (at the `append`, after every check; one helper; `append([]int(nil), ...)`, not `slices.Clone`; not inside `checkTargets`): Step 4b's helper and its placement after `checkTargets`, Steps 4d and 4f's call sites below the pin, and two Global Constraints; the rejected-declaration half is covered by the item 19 tests Step 5 names. Decision 3 (doc comments as a guarantee; the `step` field comment; `Template` and `Bind` untouched): Steps 4a, 4c, and 4e, verbatim, and the Global Constraint listing what is not edited. Decision 4 (the way out is already safe; no new item): Step 1 point 2 reads `circuit.AddGate`'s `Operation` literal, and nothing in the plan edits `Bind` or `circuit`. Rulings: reused slice and illegal-value mutations (`TestAddCopiesTheCallerTargetsSlice`), rejected declaration (Step 5's item 19 tests), by-value copy unaffected (Step 5's item 21 tests), concurrency (Step 7's race run), empty target list (unreachable, item 19's guard, not edited). Effect on callers: Step 7 runs the full suite, the race run, and the QAOA demo. Backward compatibility: CHANGELOG entry under `### Fixed` (Step 6); no ADR and no spec amendment (none planned, by design). Red test rulings: the moved test is byte for byte in Step 2 with only its comment and name changed; the red file is deleted in Step 2 and the tag's fate is in the Global Constraints and Step 7.
- **Placeholder scan.** No TBD/TODO; every code and doc step carries its full text; every command names its expected result.
- **Type consistency.** `cloneTargets(targets []int) []int` is named identically in the Architecture, the Global Constraints, the Interfaces block, Steps 4a, 4b, 4d, 4f, and the commit message. `AddParamGate(name string, factory Factory, targets ...int) error`, `AddGate(gate quantum.Gate, targets ...int) error`, and `Bind(values Params) (*circuit.Circuit, error)` match `parameterized/parameterized.go`. `circuit.New`, `gates.NewRy`, `gates.NewRx`, `gates.NewCNOT`, `state.New`, `parameterized.Ry`, `parameterized.Rx`, and `parameterized.Params` are all reachable through the test file's existing imports. `state.State.Amplitude(int) complex128` is compared with `!=`, exact equality, which holds because both circuits apply gates from the same constructors in the same order. The test names cited in Steps 3, 5, and 7 match the functions written in Step 2.
- **Prototype.** Every code, test, and documentation block above was applied to a scratch copy of the repository at `8ee8579` in this exact sequence: Step 2 red (both failure messages as stated in Step 3, `go vet -tags redtests` clean with no tagged file left), Step 4 green (twenty-four `--- PASS` lines as stated in Step 5), Step 6 anchor matched exactly once, then `gofmt -l`, `go vet` (plain and `-tags redtests` on both packages), `staticcheck`, `go build`, `go test ./...`, `go test -race ./parameterized ./algorithm`, the red-tag run (`[no tests to run]` for both packages), and the QAOA demo lines, all as stated in Step 7. The diff was +27/-8 on `parameterized/parameterized.go`, +18 on `CHANGELOG.md`, +90 on `parameterized/parameterized_test.go`, and the 61-line `parameterized/backlog_red_test.go` removed.
