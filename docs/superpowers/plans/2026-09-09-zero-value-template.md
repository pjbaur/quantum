# Zero-Value Template (Backlog Item 18) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close backlog item 18 by making a zero-value `parameterized.Template` a valid, usable template for zero qubits, so no method panics on a `Template` declared without `NewTemplate`.

**Architecture:** `AddParamGate` is the only writer of the `seen` map; it now allocates the map on the first declaration, and `NewTemplate` stops pre-allocating it, so there is one initialization path exercised by every template and the zero value is structurally the template `NewTemplate(0)` returns. Every other method already gives the zero-qubit answer on the zero value (`checkTargets` rejects every target, `Bind` reads a nil map safely and then fails in `circuit.New`); the `Template` doc comment states the contract.

**Tech Stack:** Go standard library only. No new dependencies.

**Spec:** `docs/superpowers/specs/2026-09-09-zero-value-template-design.md`

## Classification

**Standard.** One-branch behavior change inside `parameterized.Template.AddParamGate` plus a one-line simplification of `NewTemplate`; no signature, type, or cross-package change. One task, one test cycle.

## Global Constraints

- Never edit a failing test to make code pass. The moved red test is the item's acceptance test; its setup, recover guard, call, and conditional assertions are copied verbatim. If it fails after implementation, the implementation is wrong. Report it.
- Error style in `parameterized/`: typed package-local errors (`MissingParameterError`, `UnknownParameterError`, `InvalidParameterValueError`) with `Name` fields and `%q`-quoted messages; `AddParamGate` and `AddGate` return `fmt.Errorf` for a nil factory or gate and `quantum.QubitsOutOfRangeError` for targets. This plan adds no error and changes no message (spec Decision 1 rules out a "not constructed" error).
- Doc-comment voice: explain the why, cite conventions (`NewTemplate`, `circuit.New`, `Bind`), no filler. Every comment in this plan is verbatim; do not reword it.
- `checkTargets`, `AddGate`, `Bind`, `ParamNames`, `ParamStepCounts`, and `NumQubits` are not edited. The target handling in `AddParamGate` is not edited; the allocation goes after the target check (spec "Item 19 follows on locally").
- `NewTemplate` keeps its signature and does not validate `numQubits` (spec Decision 3).
- Item 19's test stays tagged and must still fail. `TestRedAddParamGateRejectsNoTargets` stays in `parameterized/backlog_red_test.go` under `//go:build redtests`, unedited; the plan removes only the item 18 test, and the import block is unchanged because item 19's test uses all three imports.
- No new dependencies. Go standard library only.
- The task ends with `gofmt -l .` printing nothing, `go vet ./...`, `staticcheck ./...`, `go build ./...`, `go test ./...` green, `go vet -tags redtests ./parameterized ./algorithm` compiling, then a commit whose message ends with a blank line and the trailers `Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>` and `Claude-Session: https://claude.ai/code/session_01XMPFmYskZD8uji6xSh5x11`.
- Do not edit `.superpowers/backlog/` or `docs/enhancement-backlog-2026-08-27.md`; the run lead ticks the item after the QA gate.

---

### Task 1: Allocate `seen` on first use so the zero-value `Template` works

**Sizing estimate:** about 900 lines of existing code to read (`parameterized/parameterized.go` 194, `parameterized/parameterized_test.go` 303, `parameterized/backlog_red_test.go` 64, `circuit/circuit.go` lines 25-55 (about 30), `quantum/errortypes.go` lines 8-15 and 65-72, the spec in full (about 210), `CHANGELOG.md` lines 1-115); 2 non-test files modified (`parameterized/parameterized.go` +15/-6, `CHANGELOG.md` +14) plus 2 test files (`parameterized/parameterized_test.go` +65, `parameterized/backlog_red_test.go` -24); net diff about +65 lines; one test cycle. Under a quarter of a context window.

**Files:**
- Modify: `parameterized/parameterized.go:46-61` (the `Template` type with its doc comment, and `NewTemplate`), `parameterized/parameterized.go:108-114` (inside `AddParamGate`, between the target check and the `seen` lookup)
- Modify: `CHANGELOG.md` (insert one entry at the end of `### Fixed` under `## Unreleased`, directly before `### Breaking` at line 114)
- Test: `parameterized/parameterized_test.go:3-14` (add the `quantum` import) and the end of the file (append two tests after `TestParamStepCounts`)
- Test: `parameterized/backlog_red_test.go:23-46` (delete the item 18 test and its comment; no import change)

**Interfaces:**
- Consumes (existing, unchanged):
  - `type Template struct { numQubits int; steps []step; paramOrder []string; seen map[string]bool }` in `parameterized/parameterized.go:48-53`
  - `func NewTemplate(numQubits int) *Template` in `parameterized/parameterized.go:56` (signature unchanged; body no longer allocates `seen`)
  - `func (t *Template) AddParamGate(name string, factory Factory, targets ...int) error` in `parameterized/parameterized.go:104` (signature unchanged; gains the allocation after `checkTargets`)
  - `func (t *Template) checkTargets(targets []int) error` in `parameterized/parameterized.go:91` (rejects any target outside `[0, numQubits-1]` as `*quantum.QubitsOutOfRangeError`; on the zero value that is every target; not modified)
  - `func (t *Template) Bind(values Params) (*circuit.Circuit, error)` in `parameterized/parameterized.go:134` (reads `t.seen[name]` for each key of `values`, then calls `circuit.New(t.numQubits)`; not modified)
  - `func New(numQubits int) (*Circuit, error)` in `circuit/circuit.go:25` (returns `*quantum.InvalidQubitCountError` for `numQubits <= 0`)
  - `type QubitsOutOfRangeError struct { Index, MaxIndex int }` and `type InvalidQubitCountError struct { Requested int; Reason string }` in `quantum/errortypes.go` (pointer receivers on `Error()`, so `errors.As` targets are `*quantum.QubitsOutOfRangeError` and `*quantum.InvalidQubitCountError`)
  - `gates.NewHadamard()` in `gates/`
- Produces: nothing new. For item 19 (a later plan): `checkTargets` and the first two statements of `AddParamGate` (the nil-factory check and the `checkTargets` call) and of `AddGate` are byte for byte what they were; item 19's empty-target guard can go at the top of either method or inside `checkTargets` without touching the allocation, which sits after the target check.

- [ ] **Step 1: Read the context**

1. `parameterized/parameterized.go` in full. Note the `Template` type at lines 46-53 and `NewTemplate` at lines 55-61 (the only allocation of `seen`), `checkTargets` at lines 91-98 (`target >= t.numQubits` is true for every target when `numQubits` is 0), `AddParamGate` at lines 104-117 (the write `t.seen[name] = true` at line 112 is the panic), `AddGate` at lines 120-129 (never touches `seen`), and `Bind` at lines 134-164 (line 145 reads `t.seen[name]`, safe on a nil map; line 150 calls `circuit.New(t.numQubits)`).
2. `circuit/circuit.go` lines 25-37: `New` returns `&quantum.InvalidQubitCountError{Requested: numQubits, Reason: "must be positive"}` for a count of 0. This is the error `Bind` returns on the zero value once its own checks pass.
3. `parameterized/backlog_red_test.go` lines 23-46: the test you are moving. Its recover guard and its conditional assertions are the acceptance criteria. Lines 47-64 are item 19, which stays.
4. `parameterized/parameterized_test.go` lines 1-14 (imports: `errors`, `gates`, `parameterized` are present; `quantum` is not) and lines 243-303 (`TestParamStepCounts`, the last function; you append after it).
5. The spec, in full: Decision 1 says why the zero value is made usable rather than `NewTemplate` made mandatory; Decision 2 says why the allocation lives in `AddParamGate` and why `NewTemplate` stops allocating.

- [ ] **Step 2: Move the red test and add the new test**

In `parameterized/backlog_red_test.go`, delete exactly this block (lines 23-46: the comment, the function, and the blank line after it), so that the `// Backlog item 19:` comment directly follows the closing parenthesis of the import block plus one blank line:

```go
// Backlog item 18: zero-value Template panics in AddParamGate.
//
// Template zero value × nil seen map in AddParamGate → panic instead of an
// error (or success).
func TestRedZeroValueTemplateAddParamGateDoesNotPanic(t *testing.T) {
	var tmpl parameterized.Template
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("AddParamGate on a zero-value Template panicked: %v; want an error or success, never a panic (NewTemplate is not documented as required)", r)
		}
	}()
	// A zero-value template has no qubits, so any target is out of range;
	// the no-target call is the one that reaches the parameter bookkeeping.
	err := tmpl.AddParamGate("", parameterized.Ry)
	if err == nil {
		if names := tmpl.ParamNames(); len(names) != 1 || names[0] != "" {
			t.Fatalf("after an accepted AddParamGate(%q): ParamNames() = %q, want [%q]", "", names, "")
		}
		if counts := tmpl.ParamStepCounts(); counts[""] != 1 {
			t.Fatalf("after an accepted AddParamGate(%q): ParamStepCounts() = %v, want map[\"\":1]", "", counts)
		}
	}
}

```

The import block is unchanged: item 19's test uses `testing`, `gates`, and `parameterized`.

In `parameterized/parameterized_test.go`, change the import block (lines 3-14) to add `quantum`, keeping the groups gofmt-sorted:

```go
import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/pjbaur/quantum/circuit"
	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/internal/sparsestate"
	"github.com/pjbaur/quantum/parameterized"
	"github.com/pjbaur/quantum/quantum"
	"github.com/pjbaur/quantum/state"
)
```

Then append to the end of the file (after the closing brace of `TestParamStepCounts`). The first function is the moved test: same setup, recover guard, call, and conditional assertions, renamed to the package's style, comment rewritten to describe the contract instead of the defect. The second is new and passes before and after; it pins what the zero value is through calls item 19 does not change.

```go

// TestZeroValueTemplateAddParamGateDoesNotPanic pins that a Template
// declared without NewTemplate is safe to call (backlog item 18). Before
// the fix AddParamGate wrote to the nil seen map unconditionally and
// panicked with "assignment to entry in nil map"; the map is now allocated
// on the first declaration, so the call errors or succeeds like any other.
func TestZeroValueTemplateAddParamGateDoesNotPanic(t *testing.T) {
	var tmpl parameterized.Template
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("AddParamGate on a zero-value Template panicked: %v; want an error or success, never a panic (NewTemplate is not documented as required)", r)
		}
	}()
	// A zero-value template has no qubits, so any target is out of range;
	// the no-target call is the one that reaches the parameter bookkeeping.
	err := tmpl.AddParamGate("", parameterized.Ry)
	if err == nil {
		if names := tmpl.ParamNames(); len(names) != 1 || names[0] != "" {
			t.Fatalf("after an accepted AddParamGate(%q): ParamNames() = %q, want [%q]", "", names, "")
		}
		if counts := tmpl.ParamStepCounts(); counts[""] != 1 {
			t.Fatalf("after an accepted AddParamGate(%q): ParamStepCounts() = %v, want map[\"\":1]", "", counts)
		}
	}
}

// TestZeroValueTemplateIsAZeroQubitTemplate pins what the zero value is:
// the template NewTemplate(0) returns. It declares nothing, rejects every
// target as out of range, leaves nothing declared after a rejection, and
// Bind fails the way circuit.New fails for a qubit count of zero, after
// its own parameter checks. Nothing here depends on AddParamGate accepting
// a call with no targets, so the test keeps its meaning once such calls
// are rejected (backlog item 19).
func TestZeroValueTemplateIsAZeroQubitTemplate(t *testing.T) {
	var tmpl parameterized.Template
	if got := tmpl.NumQubits(); got != 0 {
		t.Fatalf("NumQubits() = %d, want 0", got)
	}
	if got := tmpl.ParamNames(); len(got) != 0 {
		t.Fatalf("ParamNames() = %q, want none", got)
	}
	if got := tmpl.ParamStepCounts(); len(got) != 0 {
		t.Fatalf("ParamStepCounts() = %v, want none", got)
	}
	var rangeErr *quantum.QubitsOutOfRangeError
	if err := tmpl.AddParamGate("theta", parameterized.Ry, 0); !errors.As(err, &rangeErr) {
		t.Fatalf("AddParamGate(%q, Ry, 0) on a zero-value Template: err = %v, want QubitsOutOfRangeError", "theta", err)
	}
	if err := tmpl.AddGate(gates.NewHadamard(), 0); !errors.As(err, &rangeErr) {
		t.Fatalf("AddGate(H, 0) on a zero-value Template: err = %v, want QubitsOutOfRangeError", err)
	}
	if got := tmpl.ParamNames(); len(got) != 0 {
		t.Fatalf("ParamNames() after rejected declarations = %q, want none", got)
	}
	var unknownErr *parameterized.UnknownParameterError
	if _, err := tmpl.Bind(parameterized.Params{"theta": 0.1}); !errors.As(err, &unknownErr) {
		t.Fatalf("Bind with an undeclared name on a zero-value Template: err = %v, want UnknownParameterError", err)
	}
	var countErr *quantum.InvalidQubitCountError
	if _, err := tmpl.Bind(parameterized.Params{}); !errors.As(err, &countErr) {
		t.Fatalf("Bind on a zero-value Template: err = %v, want InvalidQubitCountError, the error circuit.New returns for zero qubits", err)
	}
}
```

- [ ] **Step 3: Run the tests to verify they fail**

Run:

```bash
go test ./parameterized -run '^TestZeroValueTemplate' -v 2>&1 | grep -E '^\s*(--- |parameterized_test)|^(ok|FAIL)'
go vet -tags redtests ./parameterized
```

Expected: `TestZeroValueTemplateAddParamGateDoesNotPanic` fails with `AddParamGate on a zero-value Template panicked: assignment to entry in nil map; want an error or success, never a panic (NewTemplate is not documented as required)`. `TestZeroValueTemplateIsAZeroQubitTemplate` passes (every call it makes is rejected before the map write, or only reads the map). The `go vet` under the `redtests` tag compiles cleanly.

- [ ] **Step 4: Implement the fix**

Two edits in `parameterized/parameterized.go`.

(a) Replace the `Template` type and `NewTemplate` (lines 46-61)

```go
// Template is a circuit recipe with named parameter holes. A Template is
// safe for concurrent reads after all Add calls complete.
type Template struct {
	numQubits  int
	steps      []step
	paramOrder []string
	seen       map[string]bool
}

// NewTemplate returns a template for circuits on numQubits qubits.
func NewTemplate(numQubits int) *Template {
	return &Template{
		numQubits: numQubits,
		seen:      make(map[string]bool),
	}
}
```

with

```go
// Template is a circuit recipe with named parameter holes. The zero value
// is ready to use and is the template NewTemplate(0) returns: it declares
// nothing, rejects every target as out of range, and Bind fails as
// circuit.New does for a qubit count of zero. No method panics on it;
// NewTemplate is how a template for a positive qubit count is made, not a
// precondition of the methods. A Template is safe for concurrent reads
// after all Add calls complete.
type Template struct {
	numQubits  int
	steps      []step
	paramOrder []string
	// seen is allocated by AddParamGate on the first declaration, the only
	// place it is written, so the zero value needs no constructor.
	seen map[string]bool
}

// NewTemplate returns a template for circuits on numQubits qubits.
func NewTemplate(numQubits int) *Template {
	return &Template{numQubits: numQubits}
}
```

(b) Inside `AddParamGate`, replace lines 108-114

```go
	if err := t.checkTargets(targets); err != nil {
		return err
	}
	if !t.seen[name] {
		t.seen[name] = true
		t.paramOrder = append(t.paramOrder, name)
	}
```

with

```go
	if err := t.checkTargets(targets); err != nil {
		return err
	}
	if t.seen == nil {
		t.seen = make(map[string]bool)
	}
	if !t.seen[name] {
		t.seen[name] = true
		t.paramOrder = append(t.paramOrder, name)
	}
```

No import changes. `AddParamGate`'s doc comment, `checkTargets`, `AddGate`, and `Bind` are not edited.

- [ ] **Step 5: Run the tests to verify they pass**

Run:

```bash
go test ./parameterized -v 2>&1 | grep -E '^(--- |ok|FAIL)'
```

Expected: every test in the package reports `--- PASS`, including `TestZeroValueTemplateAddParamGateDoesNotPanic`, `TestZeroValueTemplateIsAZeroQubitTemplate`, `TestParamStepCounts` (whose `no declared parameters` and `fixed gates only` rows now run with a nil `seen`), and `TestBindErrors` (whose `unknown key` row reads the map), then `ok`.

- [ ] **Step 6: Record the change in the CHANGELOG**

In `CHANGELOG.md`, under `## Unreleased`, `### Fixed`, insert a new entry directly before `### Breaking` (line 114). The three lines immediately above the insertion point are the end of item 17's entry:

```markdown
stays tagged in `algorithm/backlog_red_numeric_test.go`. Design:
`docs/superpowers/specs/2026-09-08-parameter-shift-evaluation-count-design.md`.

### Breaking
```

Replace that with:

```markdown
stays tagged in `algorithm/backlog_red_numeric_test.go`. Design:
`docs/superpowers/specs/2026-09-08-parameter-shift-evaluation-count-design.md`.

#### `parameterized.Template`'s zero value is usable

`AddParamGate` wrote to the template's name-tracking map unconditionally,
so a `Template` declared as a plain variable rather than through
`NewTemplate` panicked with "assignment to entry in nil map" on its first
parameter declaration, although nothing documented `NewTemplate` as
required. The map is now allocated on the first declaration, the only
place it is written, and `NewTemplate` no longer pre-allocates it, so the
zero value is exactly the template `NewTemplate(0)` returns: it declares
nothing, rejects every target as out of range, and `Bind` fails as
`circuit.New` does for a qubit count of zero. No method panics on it;
templates built with `NewTemplate` behave as before. Design:
`docs/superpowers/specs/2026-09-09-zero-value-template-design.md`.

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

Expected: `gofmt -l` prints nothing; vet (both), staticcheck, build, the full suite, and the race run are clean; the red run lists exactly two failures, `--- FAIL: TestRedAddParamGateRejectsNoTargets` (item 19, untouched by design) and `--- FAIL: TestRedParameterShiftAtHugeAngleMatchesReducedAngle` (item 20), and no `TestRedZeroValueTemplate` line; the demo prints `VQE from the symmetric start: energy = -1.0000, expected cut = 2.0000` and then `  iterations accepted: 29, energy evaluations: 378`; `git status` lists exactly the four files named under **Files** and nothing else. If item 19's test passes, a third red test fails, or the demo lines differ, stop and report: something outside this item changed.

- [ ] **Step 8: Commit**

```bash
git add parameterized/parameterized.go parameterized/parameterized_test.go parameterized/backlog_red_test.go CHANGELOG.md
git commit -m "fix(parameterized): make the zero-value Template usable

AddParamGate wrote to the seen map unconditionally, and only NewTemplate
allocated it, so a Template declared as a plain variable panicked with
\"assignment to entry in nil map\" on its first parameter declaration
although nothing documented NewTemplate as required. Every other method
already gave the zero-qubit answer on the zero value: checkTargets
rejects every target, Bind reads the nil map safely and then fails in
circuit.New. AddParamGate now allocates the map on the first declaration,
the only place it is written, and NewTemplate no longer pre-allocates it,
so there is one initialization path exercised by every template and the
zero value is exactly what NewTemplate(0) returns. The Template doc
comment states the contract. checkTargets and the target handling in both
Add methods are unchanged so item 19's empty-target guard stays local.

TestRedZeroValueTemplateAddParamGateDoesNotPanic moves into the regular
suite as TestZeroValueTemplateAddParamGateDoesNotPanic. Backlog item 18.

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01XMPFmYskZD8uji6xSh5x11"
```

If the commit fails with `.git/index.lock` present, another agent is committing; wait a few seconds and run the `git commit` again unchanged.

---

## Self-Review

- **Spec coverage.** Decision 1 (zero value usable, contract stated on `Template`, no "not constructed" error): Step 4a's doc comment and the Global Constraints line on error style; pinned by both tests in Step 2. Decision 2 (single allocation in `AddParamGate`, `NewTemplate` drops the `make`, `seen` field comment): Step 4a and 4b. Decision 3 (`NewTemplate` does not validate): Global Constraints and the unchanged signature in Step 4a. Item 19 follow-on: the Interfaces block's "Produces" note, the Global Constraints line on target handling, and Step 7's red-tag expectation. Rulings: nothing declared after a rejection, `Bind`'s check order, accessors on the zero value (all in `TestZeroValueTemplateIsAZeroQubitTemplate`, Step 2); the no-target call's success today (the moved test's `err == nil` branch). Effect on callers: Step 5 names the `TestParamStepCounts` and `TestBindErrors` rows that now run on a nil map; Step 7 runs the QAOA demo. Backward compatibility: CHANGELOG entry under `### Fixed` (Step 6); no ADR or spec amendment (none planned, by design). Red test rulings: the moved test keeps setup, guard, call, and assertions byte for byte (Step 2). Testing section: both tests appear in Step 2; the red-tag expectation is in Step 7.
- **Placeholder scan.** No TBD/TODO; every code and doc step carries its full text; every command names its expected result.
- **Type consistency.** `NewTemplate(numQubits int) *Template`, `AddParamGate(name string, factory Factory, targets ...int) error`, `AddGate(gate quantum.Gate, targets ...int) error`, `Bind(values Params) (*circuit.Circuit, error)`, and the `seen map[string]bool` field match `parameterized/parameterized.go`. `*quantum.QubitsOutOfRangeError` and `*quantum.InvalidQubitCountError` have pointer-receiver `Error()` methods in `quantum/errortypes.go`, so the `errors.As` targets in Step 2 are pointers to pointers of those types. `gates.NewHadamard()` is what item 19's tagged test already uses. The test names cited in Steps 3, 5, and 7 match the functions written in Step 2.
- **Prototype.** Every code and test block above was applied to a scratch copy of the repository at `ab6e2b5` in this exact sequence: Step 2 red (the panic message as stated in Step 3, the new test passing), Step 4 green (Step 5 as stated), Step 6 anchor matched exactly once, then `gofmt -l`, `go vet` (plain and `-tags redtests` on both packages), `staticcheck`, `go build`, `go test ./...`, `go test -race ./parameterized ./algorithm`, the red-tag run (exactly items 19 and 20 failing), and the QAOA demo lines, all as stated in Step 7.
