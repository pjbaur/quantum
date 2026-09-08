# Parameter-Shift Evaluation Count (Backlog Item 17) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close backlog item 17 by making the evaluation count `parameterShiftGradient` in `algorithm/vqe.go` returns with an error include every shifted evaluation that completed before the failure, so a +pi/2 evaluation that returned an energy is counted even when its -pi/2 partner fails.

**Architecture:** Inside the helper's shift loop, the single `evals += 2` that ran after both shifted evaluations of a name is replaced by an `evals++` directly after each evaluation's nil-error check, the same shape `VQE` uses for its own evaluations; the two error returns keep returning `evals`, which is now exact at every statement. The doc comment states the contract: two per name on success, the completed-before-failure count alongside an error (as `io.Reader` returns `n`), the failing evaluation excluded, the item 16 check's `0` exact because it precedes the first evaluation. `VQE`, `evaluate`, the signature, and the item 16 check are untouched. With the last tagged test moved into the regular suite, `algorithm/backlog_red_test.go` is deleted.

**Tech Stack:** Go standard library only. No new dependencies.

**Spec:** `docs/superpowers/specs/2026-09-08-parameter-shift-evaluation-count-design.md`

## Classification

**Standard.** Behavior change on the error path of one unexported helper in `algorithm/vqe.go`; no signature, type, or cross-package change. One task, one test cycle.

## Global Constraints

- Never edit a failing test to make code pass. The moved red test is the item's acceptance test; its setup, its call, and both assertions are copied verbatim. If it fails after implementation, the implementation is wrong. Report it.
- Error style in `algorithm/`: exported entry points return typed `Invalid<X>InputError { Reason string; Err error }`; unexported helpers (`evaluate`, `parameterShiftGradient`) return the detecting package's error unchanged and `VQE` wraps it through `wrapEvaluationError`. This plan adds no error type, produces no new `Reason`, and changes no error value: the helper keeps returning the error `evaluate` returned, unwrapped, with the count beside it (spec Decision 2).
- Error style in `parameterized/` and `quantum/`: typed package-local errors (`parameterized.MissingParameterError`, `quantum.InvalidGateApplicationError`) with `%q`-quoted messages. Not modified; the new test matches `quantum.InvalidGateApplicationError` with `errors.As`, as `TestVQEEvaluationErrorKeepsItsCause` does.
- Doc-comment voice: explain the why, cite conventions (`evaluate`, `VQE`, `io.Reader`, `Bind`), no filler. Every comment in this plan is verbatim; do not reword it.
- `parameterShiftGradient` keeps its signature `(h *Hamiltonian, t *parameterized.Template, params parameterized.Params, names []string) (map[string]float64, int, error)`. Only lines inside its loop body and the last paragraph of its doc comment change. The item 16 completeness check ahead of the loop (`for _, name := range t.ParamNames() { ... return nil, 0, ... }`) is not edited: its `0` is exact because it precedes the first evaluation (spec Decision 4).
- `VQE`'s body and `evaluate` are not edited. `VQE` discards the helper's count with the error and sums it only on success, where the total is unchanged (spec Decision 2).
- `algorithm/backlog_red_test.go` is deleted in this task: after the move it holds no tests, and a tagged file with only a header comment documents nothing. `parameterized/backlog_red_test.go` (items 18 and 19) is not touched. `go vet -tags redtests ./algorithm` must still pass on a package with no file carrying the tag, and `go test -tags redtests ./algorithm -run '^TestRed'` then reports `[no tests to run]`; that is the expected outcome, not a failure.
- No new dependencies. Go standard library only.
- The task ends with `gofmt -l .` printing nothing, `go vet ./...`, `staticcheck ./...`, `go build ./...`, `go test ./...` green, `go vet -tags redtests ./algorithm ./parameterized` compiling, then a commit whose message ends with a blank line and the trailers `Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>` and `Claude-Session: https://claude.ai/code/session_01EFGpaJg7SZbiGqv7CFqaVa`.
- Do not edit `.superpowers/backlog/` or `docs/enhancement-backlog-2026-08-27.md`; the run lead ticks the item after the QA gate.

---

### Task 1: Count each shifted evaluation as it completes

**Sizing estimate:** about 860 lines of existing code to read (`algorithm/vqe.go` 436, `algorithm/backlog_red_test.go` 52, `algorithm/vqe_test.go` lines 1-30 and 868-905 (about 70), `algorithm/vqe_driver_test.go` lines 25-40 (about 15), the spec in full (about 230), `CHANGELOG.md` lines 60-93 (about 35), item 14's spec lines 209-231 (about 25)); 3 non-test files modified (`algorithm/vqe.go` +11 -2, `CHANGELOG.md` +16, `docs/superpowers/specs/2026-09-08-vqe-input-validation-design.md` +8) plus 2 test files (`algorithm/vqe_test.go` +84, `algorithm/backlog_red_test.go` deleted, -52); net diff about +65 lines; one test cycle. Under a quarter of a context window.

**Files:**
- Modify: `algorithm/vqe.go:74-75` (the last paragraph of `parameterShiftGradient`'s doc comment) and `algorithm/vqe.go:93-102` (the two evaluations inside its loop)
- Modify: `CHANGELOG.md` (insert one entry at the end of `### Fixed` under `## Unreleased`, directly before `### Breaking` at line 92)
- Modify: `docs/superpowers/specs/2026-09-08-vqe-input-validation-design.md:229` (append to the "Helper shape and the items that follow" paragraph)
- Test: `algorithm/vqe_test.go` (append two tests at the end of the file; no import change)
- Test: `algorithm/backlog_red_test.go` (delete the file)

**Interfaces:**
- Consumes (existing, unchanged):
  - `func parameterShiftGradient(h *Hamiltonian, t *parameterized.Template, params parameterized.Params, names []string) (map[string]float64, int, error)` in `algorithm/vqe.go:76` (signature unchanged; the count it returns with an error changes)
  - `func evaluate(h *Hamiltonian, t *parameterized.Template, params parameterized.Params) (float64, error)` in `algorithm/vqe.go:14` (not modified; returns a nil error exactly when it returns an energy)
  - `func VQE(h *Hamiltonian, t *parameterized.Template, opts VQEOptions) (*VQEResult, error)` in `algorithm/vqe.go:315` (not modified; lines 370-374 discard the helper's count on error and add it to `evals` on success)
  - `type InvalidGateApplicationError` in `quantum` (what `parameterized.Template.Bind` returns, via `circuit.AddGate`, for a two-qubit gate on one target; message `gate CNOT requires 2 qubits but got 1`)
  - `func H2Hamiltonian() *Hamiltonian` in `algorithm/h2.go`
  - `gates.NewRy(theta float64)` and `gates.NewCNOT()` in `gates/gates.go`; `parameterized.NewTemplate(n int)` and `(*Template).AddParamGate(name string, factory Factory, targets ...int) error` in `parameterized/parameterized.go`
- Produces: nothing new. No later plan depends on this task.

- [ ] **Step 1: Read the context**

1. `algorithm/vqe.go` in full. Note `parameterShiftGradient` at lines 29-105: the doc comment's last paragraph at lines 74-75 ("Returns the gradient keyed by name plus the number of energy evaluations consumed."), the item 16 completeness check at lines 77-81 (returns the literal `0` before any evaluation), and the loop body at lines 84-103, where the two `evaluate` calls at lines 93 and 97 each return `nil, evals, err` on failure and `evals += 2` at line 101 runs only after both succeeded. Note how `VQE` counts its own evaluations: `evals++` at lines 362 and 406, each directly after the nil-error check of an `evaluate` call; and lines 370-374, where `VQE` returns `nil` with the wrapped error and adds `gradEvals` only on success.
2. `algorithm/backlog_red_test.go` in full: the test you are moving (lines 24-52). Its factory, its call, and its two assertions are the acceptance criteria. It is the only test left in the file.
3. `algorithm/vqe_test.go` lines 1-30 (imports `errors`, `fmt`, `math`, `strings`, `testing`, `gates`, `parameterized`, `quantum` are all present) and lines 868-905 (the end of the file; you append after `TestParameterShiftDifferentiatesOnlyNamedParams`).
4. `algorithm/vqe_driver_test.go` lines 25-40: `VQEResult.Evaluations` must stay `1 + 3k`; the success-path total is unchanged by this task, so it does.
5. The spec, in full: Decision 1 says why "consumed" means "returned an energy"; Decision 2 why the count is returned beside the error and why `VQE` is not edited; Decision 3 why the fix is two `evals++` and not an arithmetic correction at one exit; Decision 5 the doc comment's exact text.

- [ ] **Step 2: Move the red test and add the new test**

Delete the tagged file; it holds only the test you are moving and a header describing tests it no longer contains:

```bash
git rm algorithm/backlog_red_test.go
```

In `algorithm/vqe_test.go`, append to the end of the file (after the closing brace of `TestParameterShiftDifferentiatesOnlyNamedParams`). The first function is the moved test: same factory, same template, same call, same two assertions, renamed to the package's `TestParameterShift...` style, leading comment rewritten to describe the contract instead of the defect. The second is new; it fails today on two of its four rows. No import change.

```go

// TestParameterShiftCountsEvaluationsBeforeFailure pins that the count
// returned with an error includes every evaluation that completed before
// the failing one (backlog item 17). The loop used to add two only after
// both shifted evaluations of a name had succeeded, so a failure on the
// -pi/2 evaluation lost the +pi/2 evaluation that had already run.
func TestParameterShiftCountsEvaluationsBeforeFailure(t *testing.T) {
	h := H2Hamiltonian()
	// Valid single-qubit gate for non-negative values, a two-qubit gate (a
	// dimension mismatch Bind rejects) for negative ones: the +pi/2 shift
	// evaluates, the -pi/2 shift fails.
	factory := func(v float64) quantum.Gate {
		if v < 0 {
			return gates.NewCNOT()
		}
		return gates.NewRy(v)
	}
	tmpl := parameterized.NewTemplate(2)
	if err := tmpl.AddParamGate("a", factory, 0); err != nil {
		t.Fatal(err)
	}

	grad, evals, err := parameterShiftGradient(h, tmpl, parameterized.Params{"a": 0.3}, []string{"a"})
	if err == nil {
		t.Fatalf("grad = %v, evals = %d, err = nil; want the -pi/2 evaluation to fail", grad, evals)
	}
	if evals != 1 {
		t.Fatalf("evals = %d after one successful and one failed evaluation, want 1 (evaluations consumed); err = %v", evals, err)
	}
}

// TestParameterShiftEvaluationCountOnFailure pins the count at each point
// a shifted evaluation can fail: it is the number of evaluations that
// completed, in the order the loop runs them (+pi/2 then -pi/2 for each
// name in names), and the failing evaluation itself is not counted, the
// rule VQE applies to its own evaluations. The gradient is nil on every
// error, and the error is the one Bind returned, unwrapped.
func TestParameterShiftEvaluationCountOnFailure(t *testing.T) {
	h := H2Hamiltonian()
	// Valid single-qubit gate within [-10, 10], a two-qubit gate (a
	// dimension mismatch Bind rejects) beyond it: a parameter at 9 fails
	// on its +pi/2 shift, one at -9 on its -pi/2 shift, one at 0 on
	// neither.
	failsBeyondTen := func(v float64) quantum.Gate {
		if math.Abs(v) > 10 {
			return gates.NewCNOT()
		}
		return gates.NewRy(v)
	}
	tmpl := parameterized.NewTemplate(2)
	if err := tmpl.AddParamGate("a", failsBeyondTen, 0); err != nil {
		t.Fatal(err)
	}
	if err := tmpl.AddParamGate("b", failsBeyondTen, 1); err != nil {
		t.Fatal(err)
	}
	names := []string{"a", "b"}

	cases := []struct {
		name   string
		params parameterized.Params
		want   int
	}{
		{"first name, plus shift", parameterized.Params{"a": 9, "b": 0}, 0},
		{"first name, minus shift", parameterized.Params{"a": -9, "b": 0}, 1},
		{"second name, plus shift", parameterized.Params{"a": 0, "b": 9}, 2},
		{"second name, minus shift", parameterized.Params{"a": 0, "b": -9}, 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			grad, evals, err := parameterShiftGradient(h, tmpl, c.params, names)
			var ge *quantum.InvalidGateApplicationError
			if !errors.As(err, &ge) {
				t.Fatalf("params=%v: grad = %v, evals = %d, err = %v; want quantum.InvalidGateApplicationError from Bind", c.params, grad, evals, err)
			}
			if grad != nil {
				t.Errorf("grad = %v, want nil on error", grad)
			}
			if evals != c.want {
				t.Errorf("evals = %d, want %d: every evaluation that completed before the failure, and not the failure itself", evals, c.want)
			}
		})
	}
}
```

- [ ] **Step 3: Run the tests to verify they fail**

Run:

```bash
go test ./algorithm -run '^TestParameterShift(CountsEvaluationsBeforeFailure|EvaluationCountOnFailure)' -v 2>&1 | grep -E '^\s*(--- |vqe_test)|^(ok|FAIL)'
go vet -tags redtests ./algorithm
```

Expected: `TestParameterShiftCountsEvaluationsBeforeFailure` fails with `evals = 0 after one successful and one failed evaluation, want 1 (evaluations consumed); err = gate CNOT requires 2 qubits but got 1`. `TestParameterShiftEvaluationCountOnFailure` fails on two rows, `first_name,_minus_shift` (`evals = 0, want 1: every evaluation that completed before the failure, and not the failure itself`) and `second_name,_minus_shift` (`evals = 2, want 3: ...`); its `first_name,_plus_shift` and `second_name,_plus_shift` rows pass (nothing had completed since the last `evals += 2`). Both `errors.As` and nil-gradient checks pass on every row today. The `go vet` under the `redtests` tag compiles cleanly with no tagged file left in the package.

- [ ] **Step 4: Implement the count**

Two edits in `algorithm/vqe.go`, both inside `parameterShiftGradient`. First, replace lines 74-76, which read

```go
// Returns the gradient keyed by name plus the number of energy evaluations
// consumed.
func parameterShiftGradient(h *Hamiltonian, t *parameterized.Template, params parameterized.Params, names []string) (map[string]float64, int, error) {
```

with

```go
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
func parameterShiftGradient(h *Hamiltonian, t *parameterized.Template, params parameterized.Params, names []string) (map[string]float64, int, error) {
```

Second, inside the loop, replace the block that reads

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

with

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

Everything else in the function (the completeness check, `grad := make(...)`, `evals := 0`, the copy of `params` into `plus` and `minus`, the shifts, and `return grad, evals, nil`) is unchanged. No import change. `VQE` and `evaluate` are not edited.

- [ ] **Step 5: Run the tests to verify they pass**

Run:

```bash
go test ./algorithm -run '^TestParameterShift' -v 2>&1 | grep -E '^(--- |ok|FAIL)'
```

Expected, in this order: `--- PASS: TestParameterShiftMatchesFiniteDifference`, `--- PASS: TestParameterShiftIsBlindToRescaledFactory`, `--- PASS: TestParameterShiftRejectsMissingParam`, `--- PASS: TestParameterShiftMissingParamErrorMatchesBind`, `--- PASS: TestParameterShiftUndeclaredNameIsRejectedByBind`, `--- PASS: TestParameterShiftDifferentiatesOnlyNamedParams`, `--- PASS: TestParameterShiftCountsEvaluationsBeforeFailure`, `--- PASS: TestParameterShiftEvaluationCountOnFailure`, then `ok`. The first six pin the success path (4 and 2 evaluations) and the item 16 path (0), which this task must not change.

- [ ] **Step 6: Record the change in the CHANGELOG and item 14's spec**

In `CHANGELOG.md`, under `## Unreleased`, `### Fixed`, insert a new entry directly before `### Breaking` (line 92). The three lines immediately above the insertion point are the end of item 16's entry:

```markdown
unaffected; no exported behavior changes. Design:
`docs/superpowers/specs/2026-09-08-parameter-shift-missing-param-design.md`.

### Breaking
```

Replace that with:

```markdown
unaffected; no exported behavior changes. Design:
`docs/superpowers/specs/2026-09-08-parameter-shift-missing-param-design.md`.

#### `algorithm`'s parameter-shift gradient counts every completed evaluation on failure

The unexported `parameterShiftGradient` helper behind `VQE` added two to
its evaluation count only after both shifted evaluations of a parameter
had succeeded, so when the +pi/2 evaluation completed and the -pi/2 one
failed, the count returned with the error omitted the evaluation that had
run, although the helper promises "the number of energy evaluations
consumed". Each evaluation is now counted as it returns an energy, the
rule `VQE` applies to its own evaluations, so the count returned with an
error covers every evaluation that completed before the failure; the
failing evaluation is not counted. `VQE` discards the count with the
error and sums it only on success, where the total is unchanged; no
exported behavior changes. With both `algorithm` red tests now in the
regular suite, `algorithm/backlog_red_test.go` is removed. Design:
`docs/superpowers/specs/2026-09-08-parameter-shift-evaluation-count-design.md`.

### Breaking
```

In `docs/superpowers/specs/2026-09-08-vqe-input-validation-design.md`, in the section `## Helper shape and the items that follow`, the paragraph currently ends at line 229 with

```markdown
reach the check. Item 17 stays open in the helper's loop body.
```

Replace that line with:

```markdown
reach the check. Item 17 stays open in the helper's loop body.
Amended 2026-09-08 (item 17): the evaluation undercount was closed inside
the helper's loop body by counting each shifted evaluation as it returns
an energy
(`docs/superpowers/specs/2026-09-08-parameter-shift-evaluation-count-design.md`);
`VQE` and the wrap are unchanged, as this paragraph anticipated, because
`VQE` discards the count with the error and sums it only on success, where
the total is the same, and with both `algorithm` red tests in the regular
suite `algorithm/backlog_red_test.go` is deleted.
```

- [ ] **Step 7: Full verification**

Run:

```bash
gofmt -l .
go vet ./...
go vet -tags redtests ./algorithm ./parameterized
staticcheck ./...
go build ./...
go test ./...
go test -race ./algorithm
go test -tags redtests ./algorithm -run '^TestRed' 2>&1 | grep -E '^(--- FAIL|ok|FAIL)'
go test -tags redtests ./parameterized -run '^TestRed' 2>&1 | grep -E '^(--- FAIL|ok|FAIL)'
go run ./cmd/quantum -demo qaoa | grep -A1 'VQE from'
git status --short
```

Expected: `gofmt -l` prints nothing; vet (both), staticcheck, build, the full suite, and the race run are clean; the `algorithm` red run prints `ok  	github.com/pjbaur/quantum/algorithm ... [no tests to run]` (no `--- FAIL` line, no `FAIL` line); the `parameterized` red run lists `--- FAIL: TestRedZeroValueTemplateAddParamGateDoesNotPanic` and `--- FAIL: TestRedAddParamGateRejectsNoTargets` (items 18 and 19, untouched by design) and ends with `FAIL`; the demo prints `VQE from the symmetric start: energy = -1.0000, expected cut = 2.0000` and then `  iterations accepted: 29, energy evaluations: 378`; `git status --short` lists exactly `D  algorithm/backlog_red_test.go` (staged by `git rm`), ` M algorithm/vqe.go`, ` M algorithm/vqe_test.go`, ` M CHANGELOG.md`, ` M docs/superpowers/specs/2026-09-08-vqe-input-validation-design.md`, and nothing else. If the `algorithm` red run lists any test, a `parameterized` red test passes, or the demo lines differ, stop and report: something outside this item changed.

- [ ] **Step 8: Commit**

```bash
git add algorithm/vqe.go algorithm/vqe_test.go algorithm/backlog_red_test.go CHANGELOG.md docs/superpowers/specs/2026-09-08-vqe-input-validation-design.md
git commit -m "fix(algorithm): count every completed evaluation in parameterShiftGradient

parameterShiftGradient added two to its evaluation count only after both
shifted evaluations of a name had succeeded, so when the +pi/2 evaluation
returned an energy and the -pi/2 evaluation failed, the count returned
with the error omitted the evaluation that had run, against the doc
comment's promise of \"the number of energy evaluations consumed\". Each
evaluation is now counted as evaluate returns its energy, the rule VQE
applies to its own evaluations, so the count with an error is exact for
the evaluations that completed and excludes the failing one, which
returned no energy. The success-path total is unchanged; VQE discards the
count with the error and is not edited.

TestRedParameterShiftCountsEvaluationsBeforeFailure moves into the regular
suite as TestParameterShiftCountsEvaluationsBeforeFailure, and
algorithm/backlog_red_test.go, now empty of tests, is deleted;
parameterized/backlog_red_test.go (items 18 and 19) stays. Backlog item 17.

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01EFGpaJg7SZbiGqv7CFqaVa"
```

`git add` on the deleted path is a no-op after `git rm` and stages the removal if `rm` was used instead; either way the deletion is in the commit. If the commit fails with `.git/index.lock` present, another agent is committing; wait a few seconds and run the `git commit` again unchanged.

---

## Self-Review

- **Spec coverage.** Decision 1 (count what returned an energy; the failing evaluation excluded): Step 4's loop edit and doc comment; pinned by both tests in Step 2 (1 for the red scenario; 0, 1, 2, 3 across the four positions). Decision 2 (count beside the error, exact; `VQE` not edited): the doc comment's `io.Reader` sentence, Global Constraints, Step 1's note on `VQE` lines 370-374, the nil-gradient assertions. Decision 3 (two `evals++`, not an arithmetic correction): Step 4's exact block. Decision 4 (item 16's `0` stays): Global Constraints and the doc comment's last sentence; Step 5 runs the item 16 tests that assert 0. Decision 5 (doc comment text): Step 4, verbatim. Decision 6 (no Reason row; item 14 spec amendment; item 16 spec left as written): Step 6. Red test rulings: the moved test keeps factory, call, and both assertions byte for byte (Step 2). Effect on the red-test file (delete; `go vet -tags redtests` still passes; `[no tests to run]`): Global Constraints, Step 2's `git rm`, Step 3's vet, Step 7's red runs. Effect on callers: Step 5 runs the success-path tests, Step 7 runs the driver test through `go test ./...` and the QAOA demo. Backward compatibility: CHANGELOG entry under `### Fixed` (Step 6); no ADR amendment (none planned, by design). Testing section: both tests appear in Step 2; the red-tag expectations are in Step 7.
- **Placeholder scan.** No TBD/TODO; every code and doc step carries its full text; every command names its expected result.
- **Type consistency.** `parameterShiftGradient(h *Hamiltonian, t *parameterized.Template, params parameterized.Params, names []string) (map[string]float64, int, error)` matches `algorithm/vqe.go:76` and every call in Step 2. `*quantum.InvalidGateApplicationError` is the type `TestVQEEvaluationErrorKeepsItsCause` already matches for the same two-qubit-gate-on-one-target failure. `tmpl.AddParamGate("a", failsBeyondTen, 0)` matches `(*Template).AddParamGate(name string, factory Factory, targets ...int) error` with `Factory` being `func(float64) quantum.Gate`. `math`, `errors`, `gates`, `parameterized`, and `quantum` are all already imported by `algorithm/vqe_test.go`. The test names cited in Steps 3, 5, and 7 match the functions written in Step 2.
- **Prototype.** Every code, test, and documentation block above was applied to a scratch copy of the repository at `21626ff` in this exact sequence: Step 2 red (the two failures and the passing rows as stated in Step 3, verified against the original loop; `go vet -tags redtests ./algorithm` clean with the file gone), Step 4 green (Step 5 as stated), Step 6 anchors matched exactly once each, then `gofmt -l`, `go vet` (plain and `-tags redtests` on both packages), `staticcheck`, `go build`, `go test ./...`, `go test -race ./algorithm`, both red-tag runs, and the QAOA demo lines, all as stated in Step 7.
