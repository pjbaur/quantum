# Empty Parameter Name Step Count (Backlog Item 15) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close backlog item 15 by making `parameterized.Template.ParamStepCounts` count a parameter named `""` like any other declared name, so `algorithm.VQE`'s existing one-gate-per-parameter check rejects a `""` parameter that drives several gates with `InvalidVQEInputError`.

**Architecture:** `ParamStepCounts` currently uses `s.param != ""` to tell parameter steps from fixed steps, while `Bind` uses `s.factory != nil`; the two disagree only for a parameter named `""`. The fix switches `ParamStepCounts` to the factory test `Bind` already applies and documents the invariant on the `step` type. `algorithm/vqe.go` does not change: `validateVQEStructure`'s loop already rejects any name whose count exceeds one.

**Tech Stack:** Go standard library only. No new dependencies.

**Spec:** `docs/superpowers/specs/2026-09-08-empty-parameter-name-step-count-design.md`

## Classification

**Standard.** One-expression behavior change inside `parameterized`, no signature or type change; the acceptance test lives in `algorithm`. One task, one test cycle.

## Global Constraints

- Never edit a failing test to make code pass. The moved red test is the item's acceptance test; its setup and assertions are copied verbatim. If it fails after implementation, the implementation is wrong. Report it.
- Error style in `parameterized/`: typed package-local errors (`MissingParameterError`, `UnknownParameterError`, `InvalidParameterValueError`) with `Name` fields and `%q`-quoted messages; `AddParamGate` and `AddGate` return `fmt.Errorf` for a nil factory or gate and `quantum.QubitsOutOfRangeError` for targets. This plan adds no error and changes no message.
- Error style in `algorithm/`: typed `InvalidVQEInputError { Reason string; Err error }`; `Reason` is the complete message; `Error()` prefixes `"invalid VQE input: "`. This plan changes nothing in `algorithm/` except tests; the Reason for the `""` case is the existing one-gate-per-parameter message with `%q` rendering the name as `""`.
- Doc-comment voice: explain the why, cite conventions (`Bind`, `AddParamGate`, `ParamStepCounts`), no filler. Every comment in this plan is verbatim; do not reword it.
- The empty string remains a legal parameter name. Do not add a rejection of `""` to `AddParamGate` or to `validateVQEStructure`; the spec's Decision 1 rules both out.
- Items 16 and 17 stay tagged and must still fail. Their tests stay in `algorithm/backlog_red_test.go` under `//go:build redtests`; the plan removes only the item 15 test (and the `"errors"` import that only it used).
- No new dependencies. Go standard library only.
- The task ends with `gofmt -l .` printing nothing, `go vet ./...`, `staticcheck ./...`, `go build ./...`, `go test ./...` green, `go vet -tags redtests ./algorithm` compiling, then a commit whose message ends with a blank line and the trailers `Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>` and `Claude-Session: https://claude.ai/code/session_01EFGpaJg7SZbiGqv7CFqaVa`.
- Do not edit `.superpowers/backlog/` or `docs/enhancement-backlog-2026-08-27.md`; the run lead ticks the item after the QA gate.

---

### Task 1: Count a `""` parameter by its factory, not its name

**Sizing estimate:** about 1,000 lines of existing code to read (`parameterized/parameterized.go` 184, `parameterized/parameterized_test.go` 246, `algorithm/vqe.go` lines 29-85 and 164-197 (about 90), `algorithm/backlog_red_test.go` 110, `algorithm/vqe_test.go` lines 1-30 and 668-703 (about 65), `algorithm/h2.go` lines 56-71, the spec in full (about 190), `CHANGELOG.md` lines 1-60, item 14's spec lines 209-219); 3 non-test files modified (`parameterized/parameterized.go` +11/-4, `CHANGELOG.md` +12, `docs/superpowers/specs/2026-09-08-vqe-input-validation-design.md` +5) plus 3 test files (`parameterized/parameterized_test.go` +19, `algorithm/vqe_test.go` +62, `algorithm/backlog_red_test.go` -27); net diff about +80 lines; one test cycle. Under a quarter of a context window.

**Files:**
- Modify: `parameterized/parameterized.go:34-35` (the `step` type comment), `parameterized/parameterized.go:71-82` (`ParamStepCounts` doc and body)
- Modify: `CHANGELOG.md` (insert a `### Fixed` section under `## Unreleased`, directly before `### Breaking` at line 60)
- Modify: `docs/superpowers/specs/2026-09-08-vqe-input-validation-design.md:211-219` (append one sentence to the "Helper shape and the items that follow" paragraph)
- Test: `parameterized/parameterized_test.go:212-228` (two new `TestParamStepCounts` fixtures and cases)
- Test: `algorithm/vqe_test.go` (append two tests at the end of the file; no import change)
- Test: `algorithm/backlog_red_test.go:16-50` (delete the item 15 test and the `"errors"` import)

**Interfaces:**
- Consumes (existing, unchanged):
  - `func (t *Template) ParamStepCounts() map[string]int` in `parameterized/parameterized.go:74` (signature and doc contract unchanged; only which steps count changes)
  - `func (t *Template) AddParamGate(name string, factory Factory, targets ...int) error` in `parameterized/parameterized.go:95` (rejects a nil factory, so every parameter step carries one)
  - `func (t *Template) AddGate(gate quantum.Gate, targets ...int) error` in `parameterized/parameterized.go:111` (stores no factory)
  - `type step struct { param string; factory Factory; gate quantum.Gate; targets []int }` in `parameterized/parameterized.go:36-41`
  - `func validateVQEStructure(h *Hamiltonian, t *parameterized.Template) error` in `algorithm/vqe.go:181` (reads `ParamStepCounts` and rejects counts above one; not modified)
  - `func VQE(h *Hamiltonian, t *parameterized.Template, opts VQEOptions) (*VQEResult, error)`, `type InvalidVQEInputError`, `func H2Hamiltonian() *Hamiltonian`, `func H2Ansatz() *parameterized.Template` in `algorithm/`
  - `gates.NewCNOT()`, `gates.NewPauliX()` in `gates/`
- Produces: nothing new. No later task.

- [ ] **Step 1: Read the context**

1. `parameterized/parameterized.go` in full. Note the `step` comment at lines 34-35 (`param == ""` is described as the fixed-gate marker), `ParamStepCounts` at lines 74-82 (`if s.param != ""`), `AddParamGate` at lines 95-108 (rejects a nil factory, then records the name and appends a step with the factory), `AddGate` at lines 110-120 (appends a step with a gate and no factory), and `Bind` at lines 145-150 (`if s.factory != nil { gate = s.factory(values[s.param]) }`). `Bind` and `ParamStepCounts` are the two places that classify steps; the fix makes them agree.
2. `algorithm/vqe.go` lines 29-39 (the `parameterShiftGradient` doc: "VQE enforces this up front through parameterized.Template.ParamStepCounts") and lines 164-197 (`validateVQEStructure`: the loop `for _, name := range t.ParamNames() { if count := stepCounts[name]; count > 1 { ... } }`). Nothing here changes; once `""` is counted, this loop produces the rejection.
3. `algorithm/backlog_red_test.go` lines 25-49: the test you are moving. Its assertions are the acceptance criteria. Lines 51-110 are items 16 and 17, which stay.
4. `algorithm/vqe_test.go` lines 1-13 (imports: `errors`, `gates`, `parameterized` are already there) and `algorithm/h2.go` lines 56-71 (`H2Ansatz`: X on qubit 1, `Ry("theta")` on qubit 0, CNOT 0 to 1; the accept test rebuilds it with the parameter named `""`).
5. `parameterized/parameterized_test.go` lines 204-246 (`TestParamStepCounts`, table-driven; you add fixtures and rows).
6. The spec, in full: Decision 1 explains why the fix is here and not in `AddParamGate` or `validateVQEStructure`.

- [ ] **Step 2: Move the red test and add the new tests**

In `algorithm/backlog_red_test.go`, delete exactly this block (lines 25-50: the comment, the function, and the blank line after it), so that the `// Backlog item 16:` comment directly follows the closing parenthesis of the import block plus one blank line:

```go
// Backlog item 15: empty parameter name defeats the one-gate-per-parameter
// check.
//
// VQE(template whose parameter is named "") × sentinel collision with the
// fixed-step marker in Template.ParamStepCounts → a "" parameter driving
// several gates passes the up-front check and reaches parameterShiftGradient.
func TestRedVQEEmptyNameParameterEscapesStepCountCheck(t *testing.T) {
	h := H2Hamiltonian()
	tmpl := parameterized.NewTemplate(2)
	if err := tmpl.AddParamGate("", parameterized.Ry, 0); err != nil {
		t.Fatal(err)
	}
	if err := tmpl.AddParamGate("", parameterized.Ry, 0); err != nil {
		t.Fatal(err)
	}
	if got := tmpl.ParamNames(); len(got) != 1 || got[0] != "" {
		t.Fatalf("ParamNames = %q, want the single declared name %q", got, "")
	}

	var e *InvalidVQEInputError
	res, err := VQE(h, tmpl, VQEOptions{})
	if !errors.As(err, &e) {
		t.Fatalf("VQE with parameter %q driving 2 template steps: err = %v, result = %+v; want InvalidVQEInputError (the doc says VQE enforces one gate per parameter through ParamStepCounts, whose counts are %v)", "", err, res, tmpl.ParamStepCounts())
	}
}

```

Then, in the same file, change the import block (lines 16-23) to drop `"errors"` (the items 16 and 17 tests do not use it; leaving it would break `go vet -tags redtests ./algorithm` with `"errors" imported and not used`):

```go
import (
	"testing"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/parameterized"
	"github.com/pjbaur/quantum/quantum"
)
```

In `algorithm/vqe_test.go`, append to the end of the file (after the closing brace of `TestVQEEvaluationErrorNamesThePhase`). The first function is the moved test: same setup and assertions, renamed to the package's `TestVQE...` style, comment rewritten to describe the contract instead of the defect. The second passes before and after the fix and pins that `""` driving one gate stays accepted. The imports `errors`, `gates`, and `parameterized` are already present; no import change.

```go

// TestVQEEmptyNameParameterIsCheckedLikeAnyOther pins the
// one-gate-per-parameter rule for a parameter named "" (backlog item 15).
// The empty string is a declared name like any other: AddParamGate
// accepts it and ParamNames lists it. Before the fix ParamStepCounts
// mistook it for its fixed-step marker and never counted it, so a ""
// parameter driving two gates passed validateVQEStructure and VQE
// optimized it against the higher-harmonic gradient the check exists to
// prevent, reporting Converged.
func TestVQEEmptyNameParameterIsCheckedLikeAnyOther(t *testing.T) {
	h := H2Hamiltonian()
	tmpl := parameterized.NewTemplate(2)
	if err := tmpl.AddParamGate("", parameterized.Ry, 0); err != nil {
		t.Fatal(err)
	}
	if err := tmpl.AddParamGate("", parameterized.Ry, 0); err != nil {
		t.Fatal(err)
	}
	if got := tmpl.ParamNames(); len(got) != 1 || got[0] != "" {
		t.Fatalf("ParamNames = %q, want the single declared name %q", got, "")
	}

	var e *InvalidVQEInputError
	res, err := VQE(h, tmpl, VQEOptions{})
	if !errors.As(err, &e) {
		t.Fatalf("VQE with parameter %q driving 2 template steps: err = %v, result = %+v; want InvalidVQEInputError (the doc says VQE enforces one gate per parameter through ParamStepCounts, whose counts are %v)", "", err, res, tmpl.ParamStepCounts())
	}
}

// TestVQEAcceptsEmptyNameDrivingOneGate pins the other half of "like any
// other": a parameter named "" that drives exactly one gate satisfies
// the rule, so VQE optimizes it and reaches the same energy as the same
// ansatz with the parameter called "theta". The name is an opaque map key
// to every part of the driver; this test passes before and after the
// ParamStepCounts fix and guards against rejecting "" outright.
func TestVQEAcceptsEmptyNameDrivingOneGate(t *testing.T) {
	h := H2Hamiltonian()
	tmpl := parameterized.NewTemplate(2)
	if err := tmpl.AddGate(gates.NewPauliX(), 1); err != nil {
		t.Fatal(err)
	}
	if err := tmpl.AddParamGate("", parameterized.Ry, 0); err != nil {
		t.Fatal(err)
	}
	if err := tmpl.AddGate(gates.NewCNOT(), 0, 1); err != nil {
		t.Fatal(err)
	}

	ref, err := VQE(h, H2Ansatz(), VQEOptions{InitialParams: parameterized.Params{"theta": 0.1}})
	if err != nil {
		t.Fatalf("reference VQE: %v", err)
	}
	res, err := VQE(h, tmpl, VQEOptions{InitialParams: parameterized.Params{"": 0.1}})
	if err != nil {
		t.Fatalf("VQE with parameter %q driving one gate: %v; want it accepted like any other name", "", err)
	}
	if res.Energy != ref.Energy || res.Iterations != ref.Iterations || res.Params[""] != ref.Params["theta"] {
		t.Fatalf("VQE with parameter %q: energy %v after %d iterations at %v, want the same run as with %q: energy %v after %d iterations at %v", "", res.Energy, res.Iterations, res.Params[""], "theta", ref.Energy, ref.Iterations, ref.Params["theta"])
	}
}
```

In `parameterized/parameterized_test.go`, inside `TestParamStepCounts`, replace

```go
	multi := parameterized.NewTemplate(2)
	if err := multi.AddParamGate("theta", parameterized.Ry, 0); err != nil {
		t.Fatal(err)
	}
	if err := multi.AddParamGate("theta", parameterized.Rz, 1); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		tmpl *parameterized.Template
		want map[string]int
	}{
		{"single-use names", single, map[string]int{"theta": 1, "phi": 1}},
		{"name driving two gates", multi, map[string]int{"theta": 2}},
		{"no declared parameters", parameterized.NewTemplate(2), map[string]int{}},
	}
```

with

```go
	multi := parameterized.NewTemplate(2)
	if err := multi.AddParamGate("theta", parameterized.Ry, 0); err != nil {
		t.Fatal(err)
	}
	if err := multi.AddParamGate("theta", parameterized.Rz, 1); err != nil {
		t.Fatal(err)
	}
	// The empty string is a declared name like any other; a fixed gate in
	// between must not be mistaken for a step it drives, nor it for one.
	empty := parameterized.NewTemplate(2)
	if err := empty.AddParamGate("", parameterized.Ry, 0); err != nil {
		t.Fatal(err)
	}
	if err := empty.AddGate(gates.NewCNOT(), 0, 1); err != nil {
		t.Fatal(err)
	}
	if err := empty.AddParamGate("", parameterized.Rz, 1); err != nil {
		t.Fatal(err)
	}
	fixedOnly := parameterized.NewTemplate(2)
	if err := fixedOnly.AddGate(gates.NewCNOT(), 0, 1); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		tmpl *parameterized.Template
		want map[string]int
	}{
		{"single-use names", single, map[string]int{"theta": 1, "phi": 1}},
		{"name driving two gates", multi, map[string]int{"theta": 2}},
		{"no declared parameters", parameterized.NewTemplate(2), map[string]int{}},
		{"empty name driving two gates around a fixed gate", empty, map[string]int{"": 2}},
		{"fixed gates only", fixedOnly, map[string]int{}},
	}
```

The file already imports `gates`; no import change.

- [ ] **Step 3: Run the tests to verify they fail**

Run:

```bash
go test ./parameterized -run '^TestParamStepCounts$' -v 2>&1 | grep -E '^\s*(--- |parameterized_test)|^(ok|FAIL)'
go test ./algorithm -run '^TestVQE(EmptyName|AcceptsEmptyName)' -v 2>&1 | grep -E '^\s*(--- |vqe_test)|^(ok|FAIL)'
go vet -tags redtests ./algorithm
```

Expected: `TestParamStepCounts/empty_name_driving_two_gates_around_a_fixed_gate` fails with `ParamStepCounts() = map[], want map[:2]`; its other four subtests pass. `TestVQEEmptyNameParameterIsCheckedLikeAnyOther` fails with `err = <nil>, result = &{Energy:-1.0636533500290943 Params:map[:0] Iterations:1 Evaluations:4 Converged:true}; want InvalidVQEInputError (... whose counts are map[])`. `TestVQEAcceptsEmptyNameDrivingOneGate` passes (it pins existing behavior). The `go vet` under the `redtests` tag compiles cleanly.

- [ ] **Step 4: Implement the fix**

Two edits in `parameterized/parameterized.go`.

(a) Replace the `step` type comment (lines 34-35)

```go
// step is one template instruction: either a fixed gate (param == "") or a
// parameter-driven factory.
type step struct {
```

with

```go
// step is one template instruction: either a fixed gate (factory == nil,
// param unused) or a parameter-driven factory (factory != nil; param is
// the declared name, which may be any string, "" included). The factory,
// not the name, is what tells the two kinds apart: Bind and
// ParamStepCounts both test it, so no name can collide with a sentinel.
type step struct {
```

(b) Replace `ParamStepCounts` with its doc comment (lines 71-82)

```go
// ParamStepCounts returns, per declared parameter name, how many template
// steps consume it. A name driving exactly one gate counts 1; a name driving
// several gates counts one per step. Names never declared are absent.
func (t *Template) ParamStepCounts() map[string]int {
	counts := make(map[string]int, len(t.paramOrder))
	for _, s := range t.steps {
		if s.param != "" {
			counts[s.param]++
		}
	}
	return counts
}
```

with

```go
// ParamStepCounts returns, per declared parameter name, how many template
// steps consume it. A name driving exactly one gate counts 1; a name driving
// several gates counts one per step. Names never declared are absent, and
// every declared name is present: a step is parameter-driven when it
// carries a factory (AddParamGate rejects a nil one), the same test Bind
// applies, so a parameter named "" is counted like any other rather than
// mistaken for a fixed step.
func (t *Template) ParamStepCounts() map[string]int {
	counts := make(map[string]int, len(t.paramOrder))
	for _, s := range t.steps {
		if s.factory != nil {
			counts[s.param]++
		}
	}
	return counts
}
```

No import changes. `algorithm/vqe.go` is not edited.

- [ ] **Step 5: Run the tests to verify they pass**

Run:

```bash
go test ./parameterized -run '^TestParamStepCounts$' -v 2>&1 | grep -E '^(--- |ok|FAIL)'
go test ./algorithm -run '^TestVQE(EmptyName|AcceptsEmptyName|RejectsMultiGate)' -v 2>&1 | grep -E '^(--- |ok|FAIL)'
```

Expected: `--- PASS: TestParamStepCounts` then `ok`; `--- PASS: TestVQEEmptyNameParameterIsCheckedLikeAnyOther`, `--- PASS: TestVQEAcceptsEmptyNameDrivingOneGate`, `--- PASS: TestVQERejectsMultiGateParameter` (the named-parameter path still rejects with the same message) then `ok`.

- [ ] **Step 6: Record the change in the CHANGELOG and item 14's spec**

In `CHANGELOG.md`, under `## Unreleased`, insert a `### Fixed` section directly before `### Breaking` (line 60). The three lines immediately above the insertion point are the end of item 14's entry:

```markdown
`InitialParams`) or `quantum.IncompatibleQubitCountError` (Pauli-length
mismatch) are now `InvalidVQEInputError` with a nil cause. Design:
`docs/superpowers/specs/2026-09-08-vqe-input-validation-design.md`.

### Breaking
```

Replace that with:

```markdown
`InitialParams`) or `quantum.IncompatibleQubitCountError` (Pauli-length
mismatch) are now `InvalidVQEInputError` with a nil cause. Design:
`docs/superpowers/specs/2026-09-08-vqe-input-validation-design.md`.

### Fixed

#### `parameterized.Template.ParamStepCounts` counts a parameter named `""`

`ParamStepCounts` used a non-empty name as its test for a parameter-driven
step, so a parameter declared as `""` (accepted by `AddParamGate`, listed
by `ParamNames`, bound by `Bind`) was never counted. `algorithm.VQE` reads
those counts for its one-gate-per-parameter rule, so a `""` parameter
driving several gates passed the check and was optimized against a
parameter-shift gradient that is wrong for such a template, reporting
`Converged`. Steps are now classified by whether they carry a factory, the
test `Bind` already applied, so `""` is counted like any other name and
`VQE` rejects it with `InvalidVQEInputError` when it drives more than one
gate. `AddParamGate("")` remains accepted; a `""` parameter driving one
gate optimizes as before. Design:
`docs/superpowers/specs/2026-09-08-empty-parameter-name-step-count-design.md`.

### Breaking
```

In `docs/superpowers/specs/2026-09-08-vqe-input-validation-design.md`, in the section `## Helper shape and the items that follow`, replace the paragraph's last two lines

```markdown
the `redtests` tag; the prototype of this design leaves all three still
failing.
```

with

```markdown
the `redtests` tag; the prototype of this design leaves all three still
failing. Amended 2026-09-08 (item 15): the empty-name defect was in
`ParamStepCounts`'s fixed-step sentinel, not in this loop, and was fixed
there (`docs/superpowers/specs/2026-09-08-empty-parameter-name-step-count-design.md`);
the loop needed no extension and must not gain an empty-name check.
```

- [ ] **Step 7: Full verification**

Run:

```bash
gofmt -l .
go vet ./...
go vet -tags redtests ./algorithm
staticcheck ./...
go build ./...
go test ./...
go test -tags redtests ./algorithm -run '^TestRed' 2>&1 | grep -E '^(--- FAIL|ok|FAIL)'
go run ./cmd/quantum -demo qaoa | grep -A1 'VQE from'
git status --short
```

Expected: `gofmt -l` prints nothing; vet (both), staticcheck, build, and the full suite are clean; the red run lists exactly two failures, `TestRedParameterShiftMissingParamIsRejected` and `TestRedParameterShiftCountsEvaluationsBeforeFailure` (items 16 and 17, untouched by design); the demo prints `VQE from the symmetric start: energy = -1.0000, expected cut = 2.0000`; `git status` lists exactly the six files named under **Files** and nothing else. If a third red test fails, one of the two passes, or the demo line differs, stop and report: a later item's behavior changed.

- [ ] **Step 8: Commit**

```bash
git add parameterized/parameterized.go parameterized/parameterized_test.go algorithm/vqe_test.go algorithm/backlog_red_test.go CHANGELOG.md docs/superpowers/specs/2026-09-08-vqe-input-validation-design.md
git commit -m "fix(parameterized): count a parameter named \"\" in ParamStepCounts

ParamStepCounts used a non-empty name as its test for a parameter-driven
step while Bind used a non-nil factory, so a parameter declared as \"\" was
listed by ParamNames and bound by Bind but never counted. VQE reads the
counts for its one-gate-per-parameter rule, so a \"\" parameter driving
two gates passed validateVQEStructure and was optimized against a
parameter-shift gradient that is wrong for such a template. ParamStepCounts
now classifies steps by the factory, the test Bind already applies; the
step comment states the invariant so the sentinel is not reintroduced. No
change in algorithm/: the existing loop rejects the template with its
existing message once the count is right. AddParamGate(\"\") stays legal.

TestRedVQEEmptyNameParameterEscapesStepCountCheck moves into the regular
suite as TestVQEEmptyNameParameterIsCheckedLikeAnyOther. Backlog item 15.

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01EFGpaJg7SZbiGqv7CFqaVa"
```

If the commit fails with `.git/index.lock` present, another agent is committing; wait a few seconds and run the `git commit` again unchanged.

---

## Self-Review

- **Spec coverage.** Decision 1 (fix in `ParamStepCounts`, factory discriminator, `step` comment states the invariant): Step 4a and 4b. Decision 2 (`""` stays legal, no `AddParamGate` change): Global Constraints and Step 4 (no edit to `AddParamGate`). Decision 3 (Reason unchanged, produced by the untouched `validateVQEStructure`): Step 5 runs `TestVQERejectsMultiGateParameter` alongside the moved test. Rulings: `""` driving one gate accepted (`TestVQEAcceptsEmptyNameDrivingOneGate`, Step 2); fixed gates still absent and a fixed gate between two `""` steps (the two new `TestParamStepCounts` rows, Step 2). Effect on callers: Step 7 runs the QAOA demo. Backward compatibility: CHANGELOG `### Fixed` entry and the item 14 spec amendment (Step 6); no ADR amendment (none planned, by design). Red test rulings: the moved test keeps setup and assertions byte for byte (Step 2). Testing section: every test it names appears in Step 2; the red-tag expectation is in Step 7.
- **Placeholder scan.** No TBD/TODO; every code and doc step carries its full text; every command names its expected result.
- **Type consistency.** `ParamStepCounts() map[string]int`, `AddParamGate(name string, factory Factory, targets ...int) error`, `AddGate(gate quantum.Gate, targets ...int) error`, and the `step` fields `param`, `factory`, `gate`, `targets` match `parameterized/parameterized.go`. The test names cited in Steps 3, 5, and 7 match the functions written in Step 2. `res.Params[""]` and `ref.Params["theta"]` index `parameterized.Params` (`map[string]float64`).
- **Prototype.** Every code and test block above was applied to a scratch copy of the repository at `b50ac23` in this exact sequence: Step 2 red (both failures as stated in Step 3, the accept test passing), Step 4 green, then `gofmt -l`, `go vet` (plain and `-tags redtests`), `staticcheck`, `go build`, `go test ./...`, `go test -race ./parameterized ./algorithm`, the red-tag run (exactly items 16 and 17 failing), and the QAOA demo line, all as stated in Step 7.
