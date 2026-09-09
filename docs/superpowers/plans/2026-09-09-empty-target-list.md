# Empty Target List (Backlog Item 19) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close backlog item 19 by making `parameterized.Template.AddParamGate` and `AddGate` reject a call with no targets at the call, with an error naming the parameter or the gate, instead of appending the step and leaving `Bind` to fail inside `circuit.AddGate`.

**Architecture:** `checkTargets` is a pure range check and passes vacuously on an empty list; it stays that way. Each of the two `Add` methods gains a three-line `len(targets) == 0` guard placed after its existing nil check and before `checkTargets`, returning a `fmt.Errorf` in the style of that nil check (`parameter %q: ...` or `fixed gate %s: ...`) whose tail is `circuit.AddGate`'s own phrase "at least one target is required". Nothing after the guard changes, so `ParamNames`, `ParamStepCounts`, the `seen` allocation, and `Bind` are untouched.

**Tech Stack:** Go standard library only. No new dependencies.

**Spec:** `docs/superpowers/specs/2026-09-09-empty-target-list-design.md`

## Classification

**Standard.** One new branch in each of `AddParamGate` and `AddGate`; no signature, type, or cross-package change. One task, one test cycle.

## Global Constraints

- Never edit a failing test to make code pass. The moved red test is the item's acceptance test; its setup, calls, and assertions are copied verbatim. If it fails after implementation, the implementation is wrong. Report it.
- Error style in `parameterized/`: typed package-local errors (`MissingParameterError`, `UnknownParameterError`, `InvalidParameterValueError`) for `Bind`-time binding problems; `fmt.Errorf` for the declaration-time argument checks in `AddParamGate` and `AddGate` (nil factory, nil gate); `quantum.QubitsOutOfRangeError` for out-of-range targets. This plan adds two `fmt.Errorf` messages and no type (spec Decision 2). The exact messages are `parameter %q: at least one target is required` (with the parameter name) and `fixed gate %s: at least one target is required` (with `gate.Name()`).
- Guard placement: after the nil check, before `checkTargets`, in each method (spec Decision 1). `checkTargets` is not edited.
- Doc-comment voice: explain the why, cite conventions (`Bind`, `circuit.AddGate`), no filler. Every comment in this plan is verbatim; do not reword it.
- The `Template` doc comment, `NewTemplate`, `checkTargets`, `Bind`, `ParamNames`, `ParamStepCounts`, and `NumQubits` are not edited.
- Item 21's test stays tagged and must still fail. `TestRedCopyAfterDeclarationKeepsNamesAndCountsConsistent` stays in `parameterized/backlog_red_test.go` under `//go:build redtests`, unedited; the plan removes only the item 19 test and the `gates` import that only it used.
- No new dependencies. Go standard library only.
- The task ends with `gofmt -l .` printing nothing, `go vet ./...`, `staticcheck ./...`, `go build ./...`, `go test ./...` green, `go vet -tags redtests ./parameterized ./algorithm` compiling, then a commit whose message ends with a blank line and the trailers `Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>` and `Claude-Session: https://claude.ai/code/session_01XMPFmYskZD8uji6xSh5x11`.
- Do not edit `.superpowers/backlog/` or `docs/enhancement-backlog-2026-08-27.md`; the run lead ticks the item after the QA gate.

---

### Task 1: Reject an empty target list in `AddParamGate` and `AddGate`

**Sizing estimate:** about 1,100 lines of existing code to read (`parameterized/parameterized.go` 201, `parameterized/parameterized_test.go` 367, `parameterized/backlog_red_test.go` 80, `circuit/circuit.go` the `AddGate` method (about 45), `quantum/errortypes.go` the `QubitsOutOfRangeError` and `InvalidGateApplicationError` types (about 20), the spec in full (about 230), `CHANGELOG.md` lines 1-137); 2 non-test files modified (`parameterized/parameterized.go` +14/-2, `CHANGELOG.md` +15) plus 2 test files (`parameterized/parameterized_test.go` +49/-2, `parameterized/backlog_red_test.go` -20); net diff about +55 lines; one test cycle. Under a quarter of a context window.

**Files:**
- Modify: `parameterized/parameterized.go` at `AddParamGate` (its doc comment and the statements between the nil-factory check and the `checkTargets` call; lines 104-114 at `44befc7`) and at `AddGate` (its doc comment and the statements between the nil-gate check and the `checkTargets` call; lines 126-133 at `44befc7`)
- Modify: `CHANGELOG.md` (insert one entry at the end of `### Fixed` under `## Unreleased`, directly before the first `### Breaking`, line 129 at `44befc7`)
- Test: `parameterized/parameterized_test.go` (one comment inside `TestZeroValueTemplateAddParamGateDoesNotPanic`, and two tests appended at the end of the file after `TestZeroValueTemplateIsAZeroQubitTemplate`; no import change)
- Test: `parameterized/backlog_red_test.go` (delete the item 19 test and its comment, and the `gates` import; lines 16-42 at `44befc7`)

**Interfaces:**
- Consumes (existing, unchanged):
  - `func (t *Template) AddParamGate(name string, factory Factory, targets ...int) error` in `parameterized/parameterized.go` (signature unchanged; gains the guard after the nil-factory check)
  - `func (t *Template) AddGate(gate quantum.Gate, targets ...int) error` in `parameterized/parameterized.go` (signature unchanged; gains the guard after the nil-gate check)
  - `func (t *Template) checkTargets(targets []int) error` in `parameterized/parameterized.go` (rejects any target outside `[0, numQubits-1]` as `*quantum.QubitsOutOfRangeError`; returns nil on an empty list; not modified)
  - `func (t *Template) Bind(values Params) (*circuit.Circuit, error)` in `parameterized/parameterized.go` (calls `circuit.AddGate` per step; not modified)
  - `func (c *Circuit) AddGate(gate quantum.Gate, targets ...int) error` in `circuit/circuit.go` (returns `errors.New("at least one target is required")` on an empty list; the phrase this plan reuses; not modified)
  - `quantum.Gate` (`Name() string` is what `AddGate` puts in its message), `gates.NewHadamard()` (whose `Name()` is `Hadamard`), `parameterized.Ry`
- Produces: nothing new. No later task. For item 21 (a later plan): the `seen` map, `paramOrder`, and `steps` handling are byte for byte what item 18 left them; only two early returns were added ahead of them.

- [ ] **Step 1: Read the context**

1. `parameterized/parameterized.go` in full. Note `checkTargets` (a `for` over `targets`; on an empty list the body never runs and it returns nil), `AddParamGate` (nil-factory check, `checkTargets`, the `seen` allocation and name bookkeeping, the `steps` append), `AddGate` (nil-gate check, `checkTargets`, the `steps` append), and `Bind` (per step, `c.AddGate(gate, s.targets...)`, where an empty list fails today).
2. `circuit/circuit.go`, the `AddGate` method: after its nil checks, `if len(targets) == 0 { return errors.New("at least one target is required") }`. This is the error `Bind` returns today for a step declared with no targets, and the phrase the new messages end with.
3. `quantum/errortypes.go`: `QubitsOutOfRangeError` (what `checkTargets` returns) and `InvalidGateApplicationError` (what `circuit.AddGate` returns for a width mismatch; the spec's Decision 2 says why it is not reused here).
4. `parameterized/backlog_red_test.go`: the item 19 test you are moving (the `// Backlog item 19:` comment through the closing brace of `TestRedAddParamGateRejectsNoTargets`). Its assertions are the acceptance criteria. The `// Backlog item 21:` comment and `TestRedCopyAfterDeclarationKeepsNamesAndCountsConsistent` stay.
5. `parameterized/parameterized_test.go`: the import block (`strings` and `gates` are already imported; nothing to add), `TestAddRejectsOutOfRangeTargets` (the naming style you follow), `TestBindErrorMessagesCarryNames` (the `strings.Contains` message check you follow), and `TestZeroValueTemplateAddParamGateDoesNotPanic` (the in-test comment you rewrite; nothing else in it changes). `TestZeroValueTemplateIsAZeroQubitTemplate` is the last function; you append after it.
6. The spec, in full: Decision 1 says why the guard sits in each method rather than in `checkTargets` and why the nil check runs first; Decision 2 says why the error is `fmt.Errorf` and gives the exact messages.

- [ ] **Step 2: Move the red test, fix the red file's imports, and add the new test**

In `parameterized/backlog_red_test.go`, replace the import block and the item 19 test, that is, exactly this text (from `import (` through the blank line after the item 19 test's closing brace):

```go
import (
	"errors"
	"testing"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/parameterized"
)

// Backlog item 19: AddParamGate and AddGate accept an empty target list.
//
// AddParamGate("") with no targets × missing guard → accepted at build time
// although circuit.AddGate rejects a gate with no targets, so the step is
// declared and counted and only Bind fails.
func TestRedAddParamGateRejectsNoTargets(t *testing.T) {
	tmpl := parameterized.NewTemplate(2)
	err := tmpl.AddParamGate("", parameterized.Ry)
	if err == nil {
		_, bindErr := tmpl.Bind(parameterized.Params{"": 0.1})
		t.Fatalf("AddParamGate(%q, Ry) with no targets accepted: ParamNames() = %q, ParamStepCounts() = %v; Bind then fails with %v; want the step rejected when added", "", tmpl.ParamNames(), tmpl.ParamStepCounts(), bindErr)
	}
	fixed := parameterized.NewTemplate(2)
	if err := fixed.AddGate(gates.NewHadamard()); err == nil {
		_, bindErr := fixed.Bind(parameterized.Params{})
		t.Fatalf("AddGate(H) with no targets accepted; Bind then fails with %v; want the step rejected when added", bindErr)
	}
}

```

with

```go
import (
	"errors"
	"testing"

	"github.com/pjbaur/quantum/parameterized"
)

```

so that the `// Backlog item 21:` comment directly follows the closing parenthesis of the import block plus one blank line. The `gates` import goes because only the item 19 test used it; item 21's test uses `errors`, `testing`, and `parameterized`.

In `parameterized/parameterized_test.go`, inside `TestZeroValueTemplateAddParamGateDoesNotPanic`, replace the two-line comment above the `AddParamGate` call:

```go
	// A zero-value template has no qubits, so any target is out of range;
	// the no-target call is the one that reaches the parameter bookkeeping.
	err := tmpl.AddParamGate("", parameterized.Ry)
```

with

```go
	// A zero-value template has no qubits, so any target is out of range,
	// and a call with no targets is rejected too (backlog item 19); either
	// way the call must return an error, never panic.
	err := tmpl.AddParamGate("", parameterized.Ry)
```

Nothing else in that test changes: its recover guard, its call, and its `err == nil` branch stay byte for byte.

Then append to the end of `parameterized/parameterized_test.go` (after the closing brace of `TestZeroValueTemplateIsAZeroQubitTemplate`). The first function is the moved test: same setup, calls, and assertions, renamed to the package's style, comment rewritten to describe the contract instead of the defect. The second is new; it pins the error's naming, that a rejected call leaves the template as it was, and that a nil factory is still reported first.

```go

// TestAddRejectsNoTargets pins that a gate declared with zero targets is
// rejected at the Add*Gate call (backlog item 19). Before the fix
// checkTargets passed vacuously on an empty list, so the step was declared,
// counted by ParamNames and ParamStepCounts, and only Bind failed, deep
// inside circuit.AddGate.
func TestAddRejectsNoTargets(t *testing.T) {
	tmpl := parameterized.NewTemplate(2)
	err := tmpl.AddParamGate("", parameterized.Ry)
	if err == nil {
		_, bindErr := tmpl.Bind(parameterized.Params{"": 0.1})
		t.Fatalf("AddParamGate(%q, Ry) with no targets accepted: ParamNames() = %q, ParamStepCounts() = %v; Bind then fails with %v; want the step rejected when added", "", tmpl.ParamNames(), tmpl.ParamStepCounts(), bindErr)
	}
	fixed := parameterized.NewTemplate(2)
	if err := fixed.AddGate(gates.NewHadamard()); err == nil {
		_, bindErr := fixed.Bind(parameterized.Params{})
		t.Fatalf("AddGate(H) with no targets accepted; Bind then fails with %v; want the step rejected when added", bindErr)
	}
}

// TestAddNoTargetsErrorNamesTheGateAndDeclaresNothing pins the shape of
// the rejection: the error names the parameter or the gate, the template
// is left as it was, and a nil factory is still reported before the
// missing targets, since it is the earlier argument.
func TestAddNoTargetsErrorNamesTheGateAndDeclaresNothing(t *testing.T) {
	tmpl := parameterized.NewTemplate(2)
	if err := tmpl.AddParamGate("theta", parameterized.Ry); err == nil || !strings.Contains(err.Error(), `"theta"`) {
		t.Fatalf("AddParamGate(%q, Ry) with no targets: err = %v, want an error naming the parameter", "theta", err)
	}
	if names := tmpl.ParamNames(); len(names) != 0 {
		t.Fatalf("ParamNames() after a rejected declaration = %q, want none", names)
	}
	if counts := tmpl.ParamStepCounts(); len(counts) != 0 {
		t.Fatalf("ParamStepCounts() after a rejected declaration = %v, want none", counts)
	}
	if err := tmpl.AddGate(gates.NewHadamard()); err == nil || !strings.Contains(err.Error(), "Hadamard") {
		t.Fatalf("AddGate(H) with no targets: err = %v, want an error naming the gate", err)
	}
	// Neither rejected step was appended, so the template still binds.
	if _, err := tmpl.Bind(parameterized.Params{}); err != nil {
		t.Fatalf("Bind after rejected declarations: %v, want success on an empty template", err)
	}
	if err := tmpl.AddParamGate("theta", nil); err == nil || !strings.Contains(err.Error(), "factory must not be nil") {
		t.Fatalf("AddParamGate(%q, nil) with no targets: err = %v, want the nil-factory error first", "theta", err)
	}
}
```

- [ ] **Step 3: Run the tests to verify they fail**

Run:

```bash
go test ./parameterized -run '^(TestAddRejectsNoTargets|TestAddNoTargetsErrorNamesTheGateAndDeclaresNothing|TestZeroValueTemplate)' -v 2>&1 | grep -E '^\s*(--- |parameterized_test)|^(ok|FAIL)'
go vet -tags redtests ./parameterized
```

Expected: `TestAddRejectsNoTargets` fails with `AddParamGate("", Ry) with no targets accepted: ParamNames() = [""], ParamStepCounts() = map[:1]; Bind then fails with at least one target is required; want the step rejected when added`. `TestAddNoTargetsErrorNamesTheGateAndDeclaresNothing` fails with `AddParamGate("theta", Ry) with no targets: err = <nil>, want an error naming the parameter`. Both `TestZeroValueTemplate` tests pass. The `go vet` under the `redtests` tag compiles cleanly with the trimmed import block.

- [ ] **Step 4: Implement the fix**

Two edits in `parameterized/parameterized.go`.

(a) Replace `AddParamGate`'s doc comment and its opening statements, that is, exactly

```go
// AddParamGate adds a gate built by factory from the named parameter's
// value at Bind time. The same name may drive several gates. The name is
// an opaque key: any string is accepted, the empty string included, and
// it is what Bind and ParamStepCounts key on.
func (t *Template) AddParamGate(name string, factory Factory, targets ...int) error {
	if factory == nil {
		return fmt.Errorf("parameter %q: factory must not be nil", name)
	}
	if err := t.checkTargets(targets); err != nil {
```

with

```go
// AddParamGate adds a gate built by factory from the named parameter's
// value at Bind time. The same name may drive several gates. The name is
// an opaque key: any string is accepted, the empty string included, and
// it is what Bind and ParamStepCounts key on. At least one target is
// required: a declaration with none is rejected here, with an error naming
// the parameter, rather than declared, counted, and left for
// circuit.AddGate to reject at Bind.
func (t *Template) AddParamGate(name string, factory Factory, targets ...int) error {
	if factory == nil {
		return fmt.Errorf("parameter %q: factory must not be nil", name)
	}
	if len(targets) == 0 {
		return fmt.Errorf("parameter %q: at least one target is required", name)
	}
	if err := t.checkTargets(targets); err != nil {
```

(b) Replace `AddGate`'s doc comment and its opening statements, that is, exactly

```go
// AddGate adds a fixed gate needing no parameter.
func (t *Template) AddGate(gate quantum.Gate, targets ...int) error {
	if gate == nil {
		return fmt.Errorf("fixed gate must not be nil")
	}
	if err := t.checkTargets(targets); err != nil {
```

with

```go
// AddGate adds a fixed gate needing no parameter. At least one target is
// required: a call with none is rejected here, with an error naming the
// gate, rather than appended and left for circuit.AddGate to reject at
// Bind.
func (t *Template) AddGate(gate quantum.Gate, targets ...int) error {
	if gate == nil {
		return fmt.Errorf("fixed gate must not be nil")
	}
	if len(targets) == 0 {
		return fmt.Errorf("fixed gate %s: at least one target is required", gate.Name())
	}
	if err := t.checkTargets(targets); err != nil {
```

No import changes (`fmt` is already imported). Everything from each `checkTargets` call onward, and `checkTargets` itself, is not edited.

- [ ] **Step 5: Run the tests to verify they pass**

Run:

```bash
go test ./parameterized -v 2>&1 | grep -E '^(--- |ok|FAIL)'
```

Expected: every test in the package reports `--- PASS`, thirteen in all, including `TestAddRejectsNoTargets`, `TestAddNoTargetsErrorNamesTheGateAndDeclaresNothing`, `TestZeroValueTemplateAddParamGateDoesNotPanic` (whose `err == nil` branch is now never entered), `TestZeroValueTemplateIsAZeroQubitTemplate`, and `TestAddRejectsOutOfRangeTargets` (whose targeted calls still reach the range check), then `ok`.

- [ ] **Step 6: Record the change in the CHANGELOG**

In `CHANGELOG.md`, under `## Unreleased`, `### Fixed`, insert a new entry directly before the first `### Breaking` heading (line 129 at `44befc7`). The three lines immediately above the insertion point are the end of item 18's entry:

```markdown
behave as before. Design:
`docs/superpowers/specs/2026-09-09-zero-value-template-design.md`.

### Breaking
```

Replace that with:

```markdown
behave as before. Design:
`docs/superpowers/specs/2026-09-09-zero-value-template-design.md`.

#### `parameterized.Template` rejects a gate declared with no targets

`AddParamGate` and `AddGate` checked each given target for range and so
accepted a call with none: the step was appended, a parameter declared
that way was listed by `ParamNames` and counted by `ParamStepCounts`, and
only `Bind` failed, with circuit's "at least one target is required" from
deep inside `circuit.AddGate` rather than at the call that declared the
gate. Both methods now reject an empty target list at the call, after
their nil check and before the range check, with an error naming the
parameter (`parameter "theta": at least one target is required`) or the
gate (`fixed gate Hadamard: at least one target is required`); the
template is left as it was. Declarations with at least one target are
unchanged. Design:
`docs/superpowers/specs/2026-09-09-empty-target-list-design.md`.

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

Expected: `gofmt -l` prints nothing; vet (both), staticcheck, build, the full suite, and the race run are clean; the red run lists exactly two failures, `--- FAIL: TestRedCopyAfterDeclarationKeepsNamesAndCountsConsistent` (item 21, untouched by design) and `--- FAIL: TestRedParameterShiftAtHugeAngleMatchesReducedAngle` (item 20), and no `TestRedAddParamGate` line; the demo prints `VQE from the symmetric start: energy = -1.0000, expected cut = 2.0000` and then `  iterations accepted: 29, energy evaluations: 378`; `git status` lists exactly the four files named under **Files** and nothing else. If item 21's test passes, a third red test fails, or the demo lines differ, stop and report: something outside this item changed.

- [ ] **Step 8: Commit**

```bash
git add parameterized/parameterized.go parameterized/parameterized_test.go parameterized/backlog_red_test.go CHANGELOG.md
git commit -m "fix(parameterized): reject a gate declared with no targets

checkTargets loops over the given targets, so AddParamGate and AddGate
accepted a call with none: the step was appended, a parameter declared
that way was listed by ParamNames and counted by ParamStepCounts, and
only Bind failed, with circuit.AddGate's untyped \"at least one target is
required\" that names neither the gate nor the parameter. Each method now
rejects an empty target list right after its nil check, before the range
check, with a fmt.Errorf in the style of that nil check naming the
parameter or the gate; checkTargets stays a pure range check and nothing
after the guard changes, so the template is left as it was on rejection.

TestRedAddParamGateRejectsNoTargets moves into the regular suite as
TestAddRejectsNoTargets; the red file keeps item 21's test and drops the
gates import only it used. Backlog item 19.

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01XMPFmYskZD8uji6xSh5x11"
```

If the commit fails with `.git/index.lock` present, another agent is committing; wait a few seconds and run the `git commit` again unchanged.

---

## Self-Review

- **Spec coverage.** Decision 1 (guard at the top of each method, after the nil check, before `checkTargets`; `checkTargets` unedited; nil factory reported first): Step 4a and 4b place the guards exactly there; the Global Constraints line on placement; the nil-factory ordering is pinned by the last assertion in `TestAddNoTargetsErrorNamesTheGateAndDeclaresNothing` (Step 2). Decision 2 (`fmt.Errorf`, exact messages): the two `fmt.Errorf` lines in Step 4 match the spec's table verbatim; the naming is pinned by the `strings.Contains` checks in Step 2. Decision 3 (doc comments): Step 4a and 4b. Effect on item 18's tests: the comment rewrite in Step 2 and the "never entered" note in Step 5. Rulings: nothing declared after a rejection and `Bind` still succeeding (the middle of the new test, Step 2); zero-value no-target call now errors (Step 3 expects both `TestZeroValueTemplate` tests to pass before and Step 5 after). Effect on callers: Step 7 runs the full suite and the QAOA demo. Backward compatibility: CHANGELOG entry under `### Fixed` (Step 6); no ADR or spec amendment (none planned, by design). Red test rulings: the moved test keeps setup, calls, and assertions byte for byte, and the `gates` import is dropped from the red file (Step 2). Testing section: both tests appear in Step 2 with the failure text in Step 3; the red-tag expectation is in Step 7.
- **Placeholder scan.** No TBD/TODO; every code and doc step carries its full text; every command names its expected result.
- **Type consistency.** `AddParamGate(name string, factory Factory, targets ...int) error`, `AddGate(gate quantum.Gate, targets ...int) error`, and `checkTargets(targets []int) error` match `parameterized/parameterized.go`. `gate.Name()` is a `quantum.Gate` method and `gates.NewHadamard().Name()` returns `Hadamard` (`gates/gates.go`, `mustGate("Hadamard", ...)`), so the `strings.Contains(err.Error(), "Hadamard")` check in Step 2 matches the `%s` in Step 4b. `%q` on `theta` renders the name with its quotes, so the new test's `strings.Contains` check for the quoted name (written as a raw-string literal in Step 2) matches Step 4a. `strings` and `gates` are already imported by `parameterized/parameterized_test.go`. The test names cited in Steps 3, 5, and 7 match the functions written in Step 2.
- **Prototype.** Every code and test block above was applied to a scratch copy of the repository at `44befc7` in this exact sequence: Step 2 red (both failure messages as stated in Step 3, both `TestZeroValueTemplate` tests passing, `go vet -tags redtests` clean), Step 4 green (thirteen `--- PASS` lines as stated in Step 5), Step 6 anchor matched exactly once, then `gofmt -l`, `go vet` (plain and `-tags redtests` on both packages), `staticcheck`, `go build`, `go test ./...`, `go test -race ./parameterized ./algorithm`, the red-tag run (exactly items 20 and 21 failing), and the QAOA demo lines, all as stated in Step 7.
