# Parameter-Shift Missing Parameter (Backlog Item 16) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close backlog item 16 by making `parameterShiftGradient` in `algorithm/vqe.go` reject a `params` binding that lacks any declared parameter, before any evaluation and whatever `names` asks to shift, with the same `parameterized.MissingParameterError` that `Bind` returns for the same gap.

**Architecture:** A completeness check over `t.ParamNames()` runs ahead of the helper's shift loop and returns `&parameterized.MissingParameterError{Name: name}` with a zero evaluation count; the loop body, the signature (`names` stays), and `VQE` are untouched. The helper's doc comment gains the precondition and why the check lives here rather than in `Bind`: the shift writes a missing name into the plus and minus copies at +/- pi/2, so `Bind` never sees it missing.

**Tech Stack:** Go standard library only. No new dependencies.

**Spec:** `docs/superpowers/specs/2026-09-08-parameter-shift-missing-param-design.md`

## Classification

**Standard.** Behavior change inside one unexported helper in `algorithm/vqe.go`; no signature, type, or cross-package change. One task, one test cycle.

## Global Constraints

- Never edit a failing test to make code pass. The moved red test is the item's acceptance test; its setup, cases, and assertion are copied verbatim. If it fails after implementation, the implementation is wrong. Report it.
- Error style in `algorithm/`: exported entry points return typed `Invalid<X>InputError { Reason string; Err error }`; unexported helpers (`evaluate`, `parameterShiftGradient`) return the detecting package's error unchanged and `VQE` wraps it through `wrapEvaluationError`. This plan adds no `algorithm` error type and produces no new `Reason`: the helper returns `*parameterized.MissingParameterError` unwrapped, the error `Bind` returns for the same gap, following the pass-through convention (spec Decision 2).
- Error style in `parameterized/`: typed package-local errors (`MissingParameterError`, `UnknownParameterError`, `InvalidParameterValueError`) with `Name` fields and `%q`-quoted messages. Not modified; the helper constructs `MissingParameterError` with its exported `Name` field, as `algorithm/backend.go` and `algorithm/qpe.go` construct `quantum` errors.
- Doc-comment voice: explain the why, cite conventions (`Bind`, `ParamNames`, `MissingParameterError`, `UnknownParameterError`), no filler. Every comment in this plan is verbatim; do not reword it.
- `parameterShiftGradient` keeps its signature `(h *Hamiltonian, t *parameterized.Template, params parameterized.Params, names []string) (map[string]float64, int, error)`. `names` stays a parameter (spec Decision 3). The loop body is not edited; item 17's fix lands there later.
- `VQE`'s body is not edited. It already binds every declared name and cannot reach the new check (spec Decision 4).
- Item 17's test stays tagged and must still fail. `TestRedParameterShiftCountsEvaluationsBeforeFailure` stays in `algorithm/backlog_red_test.go` under `//go:build redtests`, unedited; the plan removes only the item 16 test, and the import block is unchanged because item 17's test uses all four imports.
- No new dependencies. Go standard library only.
- The task ends with `gofmt -l .` printing nothing, `go vet ./...`, `staticcheck ./...`, `go build ./...`, `go test ./...` green, `go vet -tags redtests ./algorithm ./parameterized` compiling, then a commit whose message ends with a blank line and the trailers `Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>` and `Claude-Session: https://claude.ai/code/session_01EFGpaJg7SZbiGqv7CFqaVa`.
- Do not edit `.superpowers/backlog/` or `docs/enhancement-backlog-2026-08-27.md`; the run lead ticks the item after the QA gate.

---

### Task 1: Reject an incomplete binding before the shift loop

**Sizing estimate:** about 1,050 lines of existing code to read (`algorithm/vqe.go` 416, `algorithm/backlog_red_test.go` 83, `algorithm/vqe_test.go` lines 1-65 and 700-772 (about 140), `parameterized/parameterized.go` lines 121-165 (about 45), the spec in full (about 230), `CHANGELOG.md` lines 1-80, item 14's spec lines 209-224); 3 non-test files modified (`algorithm/vqe.go` +20, `CHANGELOG.md` +13, `docs/superpowers/specs/2026-09-08-vqe-input-validation-design.md` +6) plus 2 test files (`algorithm/vqe_test.go` +123, `algorithm/backlog_red_test.go` -31); net diff about +130 lines; one test cycle. Under a quarter of a context window.

**Files:**
- Modify: `algorithm/vqe.go:57-62` (the tail of `parameterShiftGradient`'s doc comment, its signature, and the first line of its body)
- Modify: `CHANGELOG.md` (insert one entry at the end of `### Fixed` under `## Unreleased`, directly before `### Breaking` at line 77)
- Modify: `docs/superpowers/specs/2026-09-08-vqe-input-validation-design.md:222` (append to the "Helper shape and the items that follow" paragraph)
- Test: `algorithm/vqe_test.go` (append four tests at the end of the file; no import change)
- Test: `algorithm/backlog_red_test.go:24-54` (delete the item 16 test and its comment; no import change)

**Interfaces:**
- Consumes (existing, unchanged):
  - `func parameterShiftGradient(h *Hamiltonian, t *parameterized.Template, params parameterized.Params, names []string) (map[string]float64, int, error)` in `algorithm/vqe.go:61` (signature unchanged; gains the check as its first statement)
  - `func evaluate(h *Hamiltonian, t *parameterized.Template, params parameterized.Params) (float64, error)` in `algorithm/vqe.go:14` (not modified)
  - `func (t *Template) ParamNames() []string` in `parameterized/parameterized.go:68` (declaration order, no duplicates)
  - `func (t *Template) Bind(values Params) (*circuit.Circuit, error)` in `parameterized/parameterized.go:134` (rejects a missing declared name first, in declaration order, as `*MissingParameterError`; then an undeclared key as `*UnknownParameterError`)
  - `type MissingParameterError struct { Name string }` and `type UnknownParameterError struct { Name string }` in `parameterized/parameterized.go` (`Error()` texts `parameter %q missing from Bind values` and `parameter %q was never declared in the template`)
  - `func gradientTargetTemplate() *parameterized.Template` in `algorithm/vqe_test.go:17` (declares `a` then `b`: `Ry(a)` q0, CNOT 0 to 1, `Rx(b)` q1)
  - `func H2Hamiltonian() *Hamiltonian` in `algorithm/h2.go`
- Produces: nothing new. For item 17 (a later plan): the check occupies the first statement of `parameterShiftGradient` and returns the literal `0`; the loop body from `grad := make(...)` on is byte for byte what it was, so item 17's count fix edits only lines inside the loop.

- [ ] **Step 1: Read the context**

1. `algorithm/vqe.go` in full. Note `parameterShiftGradient` at lines 29-85: the doc comment ends at line 60 with "Returns the gradient keyed by name plus the number of energy evaluations consumed.", the body copies `params` into `plus` and `minus` (lines 65-70) and then does `plus[name] += math.Pi / 2` (line 71): for a `name` absent from `params` that reads the map's zero value, so the copies carry the name at +/- pi/2 and `Bind` sees a complete binding. Note `VQE` at lines 322-335 (every declared name set to 0, then overwritten from `InitialParams`) and line 350 (`names` is `t.ParamNames()`): `VQE` never passes an incomplete binding.
2. `parameterized/parameterized.go` lines 131-165: `Bind` checks declared names first, in `paramOrder`, returning `&MissingParameterError{Name: name}` for the first one absent; then rejects undeclared keys as `UnknownParameterError`. The new check reproduces exactly the first of those two loops.
3. `algorithm/backlog_red_test.go` lines 24-54: the test you are moving. Its four cases and its assertion are the acceptance criteria. Lines 55-83 are item 17, which stays.
4. `algorithm/vqe_test.go` lines 1-29 (imports `errors`, `math`, `testing`, `gates`, `parameterized`, `quantum` are all present; `gradientTargetTemplate` at line 17) and lines 700-772 (the end of the file; you append after `TestVQEAcceptsEmptyNameDrivingOneGate`).
5. The spec, in full: Decision 1 says why the check is a loop over `ParamNames` ahead of the shift loop and not a per-name check inside it; Decision 2 says why the error is `parameterized.MissingParameterError` unwrapped; Decision 3 says why `names` stays.

- [ ] **Step 2: Move the red test and add the new tests**

In `algorithm/backlog_red_test.go`, delete exactly this block (lines 24-54: the comment, the function, and the blank line after it), so that the `// Backlog item 17:` comment directly follows the closing parenthesis of the import block plus one blank line:

```go
// Backlog item 16: parameterShiftGradient shifts missing parameters from an
// implicit zero.
//
// parameterShiftGradient(params lacking a declared name that appears in
// names) × implicit zero default in the shifted copies → a gradient is
// returned with nil error, evaluated at value 0, and whether the call errors
// depends on the order of names.
func TestRedParameterShiftMissingParamIsRejected(t *testing.T) {
	h := H2Hamiltonian()
	tmpl := gradientTargetTemplate() // declares a and b
	incomplete := parameterized.Params{"a": 0.3}

	cases := []struct {
		name  string
		names []string
	}{
		{"a only", []string{"a"}},
		{"b only", []string{"b"}},
		{"a then b", []string{"a", "b"}},
		{"b then a", []string{"b", "a"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			grad, evals, err := parameterShiftGradient(h, tmpl, incomplete, c.names)
			if err == nil {
				t.Fatalf("names=%v with params=%v (declared %v): grad = %v, evals = %d, err = nil; want an error for a missing declared parameter", c.names, incomplete, tmpl.ParamNames(), grad, evals)
			}
		})
	}
}

```

The import block is unchanged: item 17's test uses `testing`, `gates`, `parameterized`, and `quantum`.

In `algorithm/vqe_test.go`, append to the end of the file (after the closing brace of `TestVQEAcceptsEmptyNameDrivingOneGate`). The first function is the moved test: same setup, same four cases, same assertion, renamed to the package's `TestParameterShift...` style, comment rewritten to describe the contract instead of the defect. The other three are new; the second fails today on four of its six rows, the third and fourth pass today and pin behavior the spec relies on. No import change.

```go

// TestParameterShiftRejectsMissingParam pins that a declared parameter
// absent from params is an error whatever names asks to shift (backlog
// item 16). Before the check, the shift wrote a missing name into the
// plus and minus copies at +/- pi/2, so Bind saw a complete binding and
// the helper returned the slope at an implicit 0 with a nil error when
// that name was the only one shifted, while any names that left the gap
// unshifted failed in Bind.
func TestParameterShiftRejectsMissingParam(t *testing.T) {
	h := H2Hamiltonian()
	tmpl := gradientTargetTemplate() // declares a and b
	incomplete := parameterized.Params{"a": 0.3}

	cases := []struct {
		name  string
		names []string
	}{
		{"a only", []string{"a"}},
		{"b only", []string{"b"}},
		{"a then b", []string{"a", "b"}},
		{"b then a", []string{"b", "a"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			grad, evals, err := parameterShiftGradient(h, tmpl, incomplete, c.names)
			if err == nil {
				t.Fatalf("names=%v with params=%v (declared %v): grad = %v, evals = %d, err = nil; want an error for a missing declared parameter", c.names, incomplete, tmpl.ParamNames(), grad, evals)
			}
		})
	}
}

// TestParameterShiftMissingParamErrorMatchesBind pins what the rejection
// looks like: the error is parameterized.MissingParameterError naming the
// first missing parameter in declaration order, the same error with the
// same text that Bind returns for the unshifted params, whatever the order
// of names and even when names is empty; and nothing is evaluated first,
// so the returned gradient is nil and the count is zero.
func TestParameterShiftMissingParamErrorMatchesBind(t *testing.T) {
	h := H2Hamiltonian()
	tmpl := gradientTargetTemplate() // declares a and b

	cases := []struct {
		name   string
		params parameterized.Params
		names  []string
		want   string
	}{
		{"b missing, b shifted", parameterized.Params{"a": 0.3}, []string{"b"}, "b"},
		{"a missing, a shifted", parameterized.Params{"b": 0.1}, []string{"a"}, "a"},
		{"b missing, a shifted", parameterized.Params{"a": 0.3}, []string{"a"}, "b"},
		{"b missing, b then a", parameterized.Params{"a": 0.3}, []string{"b", "a"}, "b"},
		{"both missing, b then a", parameterized.Params{}, []string{"b", "a"}, "a"},
		{"both missing, nothing shifted", parameterized.Params{}, nil, "a"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			grad, evals, err := parameterShiftGradient(h, tmpl, c.params, c.names)
			var me *parameterized.MissingParameterError
			if !errors.As(err, &me) {
				t.Fatalf("names=%v with params=%v: grad = %v, evals = %d, err = %v; want parameterized.MissingParameterError", c.names, c.params, grad, evals, err)
			}
			if me.Name != c.want {
				t.Errorf("missing parameter named %q, want %q (first missing in declaration order, not in names order)", me.Name, c.want)
			}
			if grad != nil || evals != 0 {
				t.Errorf("grad = %v, evals = %d, want nil and 0: an incomplete binding is rejected before any evaluation", grad, evals)
			}
			_, bindErr := tmpl.Bind(c.params)
			if bindErr == nil || err.Error() != bindErr.Error() {
				t.Errorf("err = %q, want Bind's own error for the same params, %q", err, bindErr)
			}
		})
	}
}

// TestParameterShiftUndeclaredNameIsRejectedByBind pins the case the
// completeness check leaves to Bind on purpose: a name in names that the
// template never declared is written into the shifted copies, but no
// shift can hide it, so Bind rejects it as UnknownParameterError at the
// first evaluation with nothing consumed.
func TestParameterShiftUndeclaredNameIsRejectedByBind(t *testing.T) {
	h := H2Hamiltonian()
	tmpl := gradientTargetTemplate() // declares a and b
	complete := parameterized.Params{"a": 0.3, "b": 0.1}

	grad, evals, err := parameterShiftGradient(h, tmpl, complete, []string{"c"})
	var ue *parameterized.UnknownParameterError
	if !errors.As(err, &ue) {
		t.Fatalf("names=[c] on a template declaring %v: grad = %v, evals = %d, err = %v; want parameterized.UnknownParameterError", tmpl.ParamNames(), grad, evals, err)
	}
	if ue.Name != "c" {
		t.Errorf("unknown parameter named %q, want %q", ue.Name, "c")
	}
	if grad != nil || evals != 0 {
		t.Errorf("grad = %v, evals = %d, want nil and 0", grad, evals)
	}
}

// TestParameterShiftDifferentiatesOnlyNamedParams pins the other half of
// the names contract: with a complete binding, names may be any subset in
// any order, the returned map holds exactly those names, each component
// equals the same component of the full gradient, and only the named
// parameters cost evaluations.
func TestParameterShiftDifferentiatesOnlyNamedParams(t *testing.T) {
	h := H2Hamiltonian()
	tmpl := gradientTargetTemplate() // declares a and b
	params := parameterized.Params{"a": 0.45, "b": 0.325}

	full, fullEvals, err := parameterShiftGradient(h, tmpl, params, []string{"a", "b"})
	if err != nil {
		t.Fatalf("full gradient: %v", err)
	}
	if fullEvals != 4 || len(full) != 2 {
		t.Fatalf("full gradient: %d evaluations over %d components, want 4 over 2", fullEvals, len(full))
	}

	reversed, evals, err := parameterShiftGradient(h, tmpl, params, []string{"b", "a"})
	if err != nil {
		t.Fatalf("reversed names: %v", err)
	}
	if evals != 4 || reversed["a"] != full["a"] || reversed["b"] != full["b"] {
		t.Errorf("names=[b a]: grad = %v after %d evaluations, want %v after 4: order must not change the result", reversed, evals, full)
	}

	only, evals, err := parameterShiftGradient(h, tmpl, params, []string{"b"})
	if err != nil {
		t.Fatalf("names=[b]: %v", err)
	}
	if evals != 2 || len(only) != 1 || only["b"] != full["b"] {
		t.Errorf("names=[b]: grad = %v after %d evaluations, want map[b:%v] after 2: only the named parameter is shifted", only, evals, full["b"])
	}
}
```

- [ ] **Step 3: Run the tests to verify they fail**

Run:

```bash
go test ./algorithm -run '^TestParameterShift(RejectsMissingParam|MissingParamErrorMatchesBind|UndeclaredNameIsRejectedByBind|DifferentiatesOnlyNamedParams)' -v 2>&1 | grep -E '^\s*(--- |vqe_test)|^(ok|FAIL)'
go vet -tags redtests ./algorithm
```

Expected: `TestParameterShiftRejectsMissingParam` fails on `b_only` only, with `names=[b] with params=map[a:0.3] (declared [a b]): grad = map[b:0], evals = 2, err = nil; want an error for a missing declared parameter`; its `a_only`, `a_then_b`, and `b_then_a` subtests pass (Bind catches the unshifted gap). `TestParameterShiftMissingParamErrorMatchesBind` fails on four rows: `b_missing,_b_shifted` (`grad = map[b:0], evals = 2, err = <nil>`), `a_missing,_a_shifted` (`grad = map[a:-0.18002729741406104], evals = 2, err = <nil>`), `b_missing,_b_then_a` (`grad = map[], evals = 2, want nil and 0`), and `both_missing,_nothing_shifted` (`grad = map[], evals = 0, err = <nil>`); its `b_missing,_a_shifted` and `both_missing,_b_then_a` rows pass. `TestParameterShiftUndeclaredNameIsRejectedByBind` and `TestParameterShiftDifferentiatesOnlyNamedParams` pass (they pin existing behavior). The `go vet` under the `redtests` tag compiles cleanly.

- [ ] **Step 4: Implement the check**

One edit in `algorithm/vqe.go`. Replace lines 57-62, which read

```go
// TestParameterShiftIsBlindToRescaledFactory pins this failure mode.
//
// Returns the gradient keyed by name plus the number of energy evaluations
// consumed.
func parameterShiftGradient(h *Hamiltonian, t *parameterized.Template, params parameterized.Params, names []string) (map[string]float64, int, error) {
	grad := make(map[string]float64, len(names))
```

with

```go
// TestParameterShiftIsBlindToRescaledFactory pins this failure mode.
//
// params must bind every parameter the template declares, the precondition
// Bind states; names selects which of those to differentiate and may list
// any subset in any order. Completeness is checked here rather than left
// to Bind because the shift would hide the gap: a name in names that
// params lacks reads as 0 from the map, so the plus and minus copies carry
// it at +/- pi/2, Bind sees a complete binding, and the slope at an
// implicit 0 comes back with a nil error, whereas a missing name that is
// not shifted stays absent and Bind rejects it, so whether the call failed
// depended on which names were shifted. The check reports the first
// missing name in declaration order as parameterized.MissingParameterError,
// the error Bind returns for the unshifted params, and consumes no
// evaluation. A name in names that the template never declared needs no
// check here: no shift can hide it from Bind, which rejects it as
// UnknownParameterError at the first evaluation.
//
// Returns the gradient keyed by name plus the number of energy evaluations
// consumed.
func parameterShiftGradient(h *Hamiltonian, t *parameterized.Template, params parameterized.Params, names []string) (map[string]float64, int, error) {
	for _, name := range t.ParamNames() {
		if _, ok := params[name]; !ok {
			return nil, 0, &parameterized.MissingParameterError{Name: name}
		}
	}
	grad := make(map[string]float64, len(names))
```

The rest of the function (from `evals := 0` through `return grad, evals, nil`) is unchanged. No import change: `parameterized` is already imported. `VQE` is not edited.

- [ ] **Step 5: Run the tests to verify they pass**

Run:

```bash
go test ./algorithm -run '^TestParameterShift' -v 2>&1 | grep -E '^(--- |ok|FAIL)'
```

Expected, in this order: `--- PASS: TestParameterShiftMatchesFiniteDifference`, `--- PASS: TestParameterShiftIsBlindToRescaledFactory`, `--- PASS: TestParameterShiftRejectsMissingParam`, `--- PASS: TestParameterShiftMissingParamErrorMatchesBind`, `--- PASS: TestParameterShiftUndeclaredNameIsRejectedByBind`, `--- PASS: TestParameterShiftDifferentiatesOnlyNamedParams`, then `ok`.

- [ ] **Step 6: Record the change in the CHANGELOG and item 14's spec**

In `CHANGELOG.md`, under `## Unreleased`, `### Fixed`, insert a new entry directly before `### Breaking` (line 77). The three lines immediately above the insertion point are the end of item 15's entry:

```markdown
gate optimizes as before. Design:
`docs/superpowers/specs/2026-09-08-empty-parameter-name-step-count-design.md`.

### Breaking
```

Replace that with:

```markdown
gate optimizes as before. Design:
`docs/superpowers/specs/2026-09-08-empty-parameter-name-step-count-design.md`.

#### `algorithm`'s parameter-shift gradient rejects an incomplete parameter binding

The unexported `parameterShiftGradient` helper behind `VQE` shifted each
named parameter in a copy of the caller's `Params`, so a declared name the
caller had left out was written into the shifted copies at +/- pi/2 by the
shift itself: `Bind` saw a complete binding and the helper returned the
slope at an implicit 0 with a nil error, while the same gap left unshifted
failed in `Bind`, so whether the call errored depended on which names were
passed. The helper now checks that every declared parameter is bound
before any evaluation and reports the first missing one, in declaration
order, as `parameterized.MissingParameterError`, the error `Bind` returns
for the same gap. `VQE` binds every declared parameter itself and is
unaffected; no exported behavior changes. Design:
`docs/superpowers/specs/2026-09-08-parameter-shift-missing-param-design.md`.

### Breaking
```

In `docs/superpowers/specs/2026-09-08-vqe-input-validation-design.md`, in the section `## Helper shape and the items that follow`, the paragraph currently ends at line 222 with

```markdown
the loop needed no extension and must not gain an empty-name check.
```

Replace that line with:

```markdown
the loop needed no extension and must not gain an empty-name check.
Amended 2026-09-08 (item 16): the missing-parameter shift was closed
inside `parameterShiftGradient` by a completeness check ahead of its loop
(`docs/superpowers/specs/2026-09-08-parameter-shift-missing-param-design.md`);
`VQE` and the wrap are unchanged, and no Decision 3 row is added because
`VQE` binds every declared name before calling the helper and cannot
reach the check. Item 17 stays open in the helper's loop body.
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
go run ./cmd/quantum -demo qaoa | grep -A1 'VQE from'
git status --short
```

Expected: `gofmt -l` prints nothing; vet (both), staticcheck, build, the full suite, and the race run are clean; the red run lists exactly one failure, `--- FAIL: TestRedParameterShiftCountsEvaluationsBeforeFailure` (item 17, untouched by design), and no `TestRedParameterShiftMissingParam` line; the demo prints `VQE from the symmetric start: energy = -1.0000, expected cut = 2.0000` and then `  iterations accepted: 29, energy evaluations: 378`; `git status` lists exactly the five files named under **Files** and nothing else. If item 17's test passes, a second red test fails, or the demo lines differ, stop and report: something outside this item changed.

- [ ] **Step 8: Commit**

```bash
git add algorithm/vqe.go algorithm/vqe_test.go algorithm/backlog_red_test.go CHANGELOG.md docs/superpowers/specs/2026-09-08-vqe-input-validation-design.md
git commit -m "fix(algorithm): reject an incomplete binding in parameterShiftGradient

parameterShiftGradient shifted each named parameter in a copy of params,
so a declared name the caller left out read as 0 from the map and was
written into the plus and minus copies at +/- pi/2 by the shift itself:
Bind saw a complete binding and the helper returned the slope at an
implicit 0 with a nil error, while the same gap left unshifted failed in
Bind, so whether the call errored depended on which names were passed.
The helper now checks every declared name against params before its loop
and reports the first missing one, in declaration order, as
parameterized.MissingParameterError, the error Bind returns for the
unshifted params, with no evaluation consumed. names stays a parameter:
dropping it would not remove the defect and the acceptance test is about
its orderings. VQE binds every declared name and is unchanged; the loop
body is unchanged so item 17's count fix stays local to it.

TestRedParameterShiftMissingParamIsRejected moves into the regular suite
as TestParameterShiftRejectsMissingParam. Backlog item 16.

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01EFGpaJg7SZbiGqv7CFqaVa"
```

If the commit fails with `.git/index.lock` present, another agent is committing; wait a few seconds and run the `git commit` again unchanged.

---

## Self-Review

- **Spec coverage.** Decision 1 (check over `ParamNames` ahead of the loop, `0` evaluations, undeclared names left to `Bind`): Step 4 and the doc comment's last two sentences; pinned by `TestParameterShiftMissingParamErrorMatchesBind` and `TestParameterShiftUndeclaredNameIsRejectedByBind` (Step 2). Decision 2 (`*parameterized.MissingParameterError`, unwrapped, same text as `Bind`): the `errors.As` and `err.Error() != bindErr.Error()` assertions in Step 2. Decision 3 (`names` stays): Global Constraints, the unchanged signature in Step 4, `TestParameterShiftDifferentiatesOnlyNamedParams` (Step 2). Decision 4 (no Reason row; item 14 spec amendment): Step 6. Item 17 follow-on: the Interfaces block's "Produces" note and Step 7's red-tag expectation. Rulings: empty `names` with an incomplete binding (the `nothing shifted` row), several names missing reported in declaration order (both `both missing` rows), no-parameter template unchanged (Step 7's full suite runs `TestVQENoParameterTemplateReportsOverflowingHamiltonian`). Effect on callers: Step 7 runs the QAOA demo. Backward compatibility: CHANGELOG entry under `### Fixed` (Step 6); no ADR amendment (none planned, by design). Red test rulings: the moved test keeps setup, cases, and assertion byte for byte (Step 2). Testing section: every test it names appears in Step 2; the red-tag expectation is in Step 7.
- **Placeholder scan.** No TBD/TODO; every code and doc step carries its full text; every command names its expected result.
- **Type consistency.** `parameterShiftGradient(h *Hamiltonian, t *parameterized.Template, params parameterized.Params, names []string) (map[string]float64, int, error)` matches `algorithm/vqe.go:61` and every call in Step 2. `&parameterized.MissingParameterError{Name: name}` and `*parameterized.UnknownParameterError` match the struct definitions in `parameterized/parameterized.go` (`Name string`). `tmpl.Bind(c.params)` returns `(*circuit.Circuit, error)`; the test discards the circuit. The test names cited in Steps 3, 5, and 7 match the functions written in Step 2. `go vet -tags redtests ./algorithm ./parameterized` in Step 7 also covers the `parameterized` red file that items 18 and 19 added on 2026-09-08.
- **Prototype.** Every code and test block above was applied to a scratch copy of the repository at `23d60a1` in this exact sequence: Step 2 red (the failures and passes as stated in Step 3, verified against the original `parameterShiftGradient`), Step 4 green (Step 5 as stated), Step 6 anchors matched exactly once each, then `gofmt -l`, `go vet` (plain and `-tags redtests` on both packages), `staticcheck`, `go build`, `go test ./...`, `go test -race ./algorithm`, the red-tag run (exactly item 17 failing), and the QAOA demo lines, all as stated in Step 7.
