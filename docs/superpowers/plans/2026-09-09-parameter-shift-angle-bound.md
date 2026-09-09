# Parameter-Shift Angle Bound (Backlog Item 20) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close backlog item 20 by rejecting, with `InvalidVQEInputError`, any angle of magnitude beyond 2^26 that the parameter-shift rule in `algorithm/vqe.go` would shift, so that `parameterShiftGradient` never returns a zero or wrong-offset gradient for an angle float64 cannot shift by +/- pi/2, and `VQE` never reports `Converged` with a huge initial parameter frozen at its start.

**Architecture:** One unexported constant `maxShiftMagnitude = 1 << 26` carries the derivation in its doc comment. `parameterShiftGradient` checks each name in `names` against it ahead of its loop (finite values only, after the item 16 completeness check) and returns `InvalidVQEInputError` with 0 evaluations. `VQE` guarantees the invariant by construction so that check is unreachable from it: the inline `InitialParams` loop gains a magnitude clause beside its finiteness clause, and the step guard gains a magnitude clause beside its overflow clause. Angles are rejected, never reduced mod 2*pi (spec Decision 1). The item's tagged red test asserts the reduction and is deleted with its file (spec "Red test ruling"); tests of the chosen contract replace it.

**Tech Stack:** Go standard library only. No new dependencies.

**Spec:** `docs/superpowers/specs/2026-09-09-parameter-shift-angle-bound-design.md`

## Classification

**Standard.** Behavior change inside `algorithm/vqe.go` within the existing driver structure; no signature, type, or cross-package change. One task, one test cycle.

## Global Constraints

- Never edit a failing test to make code pass. The new tests are the item's acceptance tests; if one fails after implementation, the implementation is wrong. Report it.
- The red test `TestRedParameterShiftAtHugeAngleMatchesReducedAngle` is **deleted, not moved and not edited**: spec "Red test ruling" rules it contradicted by Decision 1 (it asserts a reduced-angle slope with a nil error where the contract is an error with 0 evaluations). Do not port its `reduceMod2Pi` helper or its `math/big` import anywhere; nothing in the regular suite reduces an angle.
- Error style in `algorithm/`: exported entry points return typed `Invalid<X>InputError { Reason string; Err error }`; `VQE` wraps helper errors through `wrapEvaluationError`. This plan adds no error type. The helper's new error is `*InvalidVQEInputError` with `Err` nil (spec Decision 3); `VQE`'s two new errors are the same type with `Err` nil, produced before any evaluation or before the step evaluation.
- Non-finite values stay `Bind`'s. The helper's check is `isFinite(v) && math.Abs(v) > maxShiftMagnitude`; do not drop the `isFinite` guard, or `+Inf` would be reported here instead of by `Bind` and item 16's Decision 5 would become false.
- The bound is `>`-strict: exactly `2^26` and `-2^26` are accepted everywhere.
- Doc-comment voice: explain the why, cite conventions (`Bind`, `VQE`, `exp(-i*theta*P/2)`, `math.Mod`), no filler. Every comment and every Reason string in this plan is verbatim; do not reword them. The Reasons print the bound as `2^26 (6.7108864e+07)` via `float64(maxShiftMagnitude)` formatted with `%v`.
- `parameterShiftGradient` keeps its signature. The item 16 completeness check (the `for _, name := range t.ParamNames()` loop returning `MissingParameterError`) is not edited; the new check goes directly after it. The loop body (item 17's `evals++` counting) is not edited.
- `evaluate`, `validateVQEOptions`, `validateVQEStructure`, `wrapEvaluationError`, `formatPoint`, `checkEnergy`, the gradient guard, and the overflow clause of the step guard are not edited.
- No new dependencies. Go standard library only.
- The task ends with `gofmt -l .` printing nothing, `go vet ./...`, `staticcheck ./...`, `go build ./...`, `go test ./...` green, `go vet -tags redtests ./parameterized ./algorithm` compiling, then a commit whose message ends with a blank line and the trailers `Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>` and `Claude-Session: https://claude.ai/code/session_01XMPFmYskZD8uji6xSh5x11`.
- Another item's implementer may be editing `parameterized/` and appending to `CHANGELOG.md` concurrently. Locate the CHANGELOG insertion point by its neighboring headings, never by line number, and stage only the files this plan names. If `git status` shows files outside this plan's list, leave them alone and do not stage them.
- Do not edit `.superpowers/backlog/` or `docs/enhancement-backlog-2026-08-27.md`; the run lead ticks the item after the QA gate.

---

### Task 1: Bound the angles the parameter shift shifts, in the helper and in `VQE`

**Sizing estimate:** about 1,300 lines of existing code to read (`algorithm/vqe.go` 459 in full; `algorithm/backlog_red_numeric_test.go` 81; `algorithm/vqe_test.go` lines 1-30, 181-245, and 389-462, about 180; `algorithm/vqe_driver_test.go` lines 13-41, about 30; the spec in full, about 330; `CHANGELOG.md` from `### Fixed` under `## Unreleased` to the first `### Breaking`, about 85; excerpts of four prior specs named in Step 1, about 140). Files modified: 6 non-test (`algorithm/vqe.go` +80, `CHANGELOG.md` +24, four spec amendments totaling +63) plus 2 test files (`algorithm/vqe_test.go` +178, `algorithm/backlog_red_numeric_test.go` -81); net diff about +240 lines; one test cycle. The non-test file count exceeds the 1-to-4 bound by the four dated amendment notes the dispatch requires; splitting them into a second task would create a task with no test cycle, which the sizing rule merges, so they stay here. Under half a context window.

**Files:**
- Modify: `algorithm/vqe.go:29-30` (insert the constant before `parameterShiftGradient`'s doc comment), `algorithm/vqe.go:57-59` (insert a paragraph into that doc comment), `algorithm/vqe.go:75-76` and `96-97` (two sentences of that doc comment), `algorithm/vqe.go:99-104` (the check, after the item 16 loop), `algorithm/vqe.go:136-137` (`InitialParams` field comment), `algorithm/vqe.go:323-324` and `336-338` (`VQE` doc comment), `algorithm/vqe.go:374-377` (the `InitialParams` loop), `algorithm/vqe.go:421-424` (the step guard)
- Modify: `CHANGELOG.md` (insert one entry at the end of `### Fixed` under `## Unreleased`, directly before the first `### Breaking` heading)
- Modify: `docs/superpowers/specs/2026-09-08-vqe-input-validation-design.md` (four amendment notes), `docs/superpowers/specs/2026-09-08-parameter-shift-missing-param-design.md` (two), `docs/superpowers/specs/2026-09-08-parameter-shift-evaluation-count-design.md` (two), `docs/superpowers/specs/2026-08-30-parameter-binding-vqe-design.md` (one)
- Test: `algorithm/vqe_test.go:196` and `:222` (two and one table rows), and the end of the file (append three tests after `TestParameterShiftEvaluationCountOnFailure`)
- Test: `algorithm/backlog_red_numeric_test.go` (delete the file)

**Interfaces:**
- Consumes (existing, unchanged):
  - `func parameterShiftGradient(h *Hamiltonian, t *parameterized.Template, params parameterized.Params, names []string) (map[string]float64, int, error)` in `algorithm/vqe.go:98` (signature unchanged; gains one pre-loop check)
  - `func VQE(h *Hamiltonian, t *parameterized.Template, opts VQEOptions) (*VQEResult, error)` in `algorithm/vqe.go:338` (signature unchanged; gains two clauses)
  - `type InvalidVQEInputError struct { Reason string; Err error }` with `Error()` and `Unwrap()` on the pointer receiver in `algorithm/vqe.go:168-179`, so `errors.As` targets are `*InvalidVQEInputError`
  - `func isFinite(v float64) bool` in `algorithm/vqe.go:182`
  - `func evaluate(h *Hamiltonian, t *parameterized.Template, params parameterized.Params) (float64, error)` in `algorithm/vqe.go:14`
  - `func wrapEvaluationError(where string, err error) error` in `algorithm/vqe.go:252` (not called on any new path; the spec's Decision 3 explains why the step guard keeps the helper's error out of it)
  - Test helpers in `algorithm/vqe_test.go`: `gradientTargetTemplate()` (line 17: `Ry(a)` on qubit 0, CNOT, `Rx(b)` on qubit 1), `zOnQubit0(coeff float64) *Hamiltonian` (line 389: `coeff * Z0`, so on `H2Ansatz` `E = coeff * cos(theta)`), and `H2Ansatz()`, `H2Hamiltonian()` from `algorithm/h2.go`
  - `type InvalidParameterValueError struct { Name string; Value float64 }` in `parameterized/parameterized.go:206` (pointer receiver on `Error()`)
- Produces: `const maxShiftMagnitude = 1 << 26` in `algorithm/vqe.go`, unexported, untyped; read by the new tests as `float64(maxShiftMagnitude)` and compared with `math.Ldexp(1, 26)`. No later task.

- [ ] **Step 1: Read the context**

1. `algorithm/vqe.go` in full. Note `parameterShiftGradient`'s doc comment at lines 29-97 (the "First" paragraph at 35-39, the "Second" at 41-57, the `params`/`names` paragraph at 59-86, the count paragraph at 88-97), its item 16 completeness check at lines 99-103, its loop at 106-126 (`plus[name] += math.Pi / 2` at 113 is the defect: at `2^60` the sum rounds back to `plus[name]`), `VQEOptions.InitialParams` at 136-138, `isFinite` at 182, `VQE`'s doc comment at 290-337 (the validation paragraph at 323-337), its `InitialParams` loop at 370-378, and its step guard at 418-424.
2. `algorithm/backlog_red_numeric_test.go` in full: the test you are deleting, and why the spec rules it contradicted (its `err != nil` fatal at line 76 and `evals != 2` at line 78 cannot hold under an error contract).
3. `algorithm/vqe_test.go` lines 1-30 (imports: `errors`, `fmt`, `math`, `strings`, `testing`, `gates`, `parameterized`, `quantum` are all present; `gradientTargetTemplate`), lines 181-245 (`TestVQEOptionValidation` and `TestVQEOptionErrorNamesTheField`, the tables you add rows to), and lines 389-462 (`zOnQubit0` and the two step-guard tests whose style the new step-bound test follows: gradient read back from the helper, exact Reason, `errors.Unwrap` nil).
4. `algorithm/vqe_driver_test.go` lines 13-41: the `1 + 3k` evaluation invariant the new `VQE` test reuses at the bound.
5. The spec, in full: Decision 1 says why an error and not a reduction; Decision 2 derives `2^26`; Decision 3 says where each check lives and why the helper's is unreachable from `VQE`; "Red test ruling" is the deletion's authority.
6. Prior spec excerpts, so the amendment notes in Step 6 land where they cite: `docs/superpowers/specs/2026-09-08-vqe-input-validation-design.md` lines 130-150 (step guard, order inside the loop), 165-182 (Decision 3 table), and 209-238 (the "Helper shape and the items that follow" paragraph); `docs/superpowers/specs/2026-09-08-parameter-shift-missing-param-design.md` lines 185-246 (Decision 5) and 348-385 (round 1 red tests); `docs/superpowers/specs/2026-09-08-parameter-shift-evaluation-count-design.md` lines 183-212 (Decision 5) and 258-268 (the round 1 note about item 20's file); `docs/superpowers/specs/2026-08-30-parameter-binding-vqe-design.md` lines 174-187 (`VQE` Errors).

- [ ] **Step 2: Delete the red test and write the new tests**

Delete `algorithm/backlog_red_numeric_test.go`:

```bash
git rm algorithm/backlog_red_numeric_test.go
```

In `algorithm/vqe_test.go`, `TestVQEOptionValidation`, replace the last row of `cases` and the closing brace

```go
		{"Inf InitialParams", VQEOptions{InitialParams: parameterized.Params{"theta": math.Inf(-1)}}},
	}
```

with

```go
		{"Inf InitialParams", VQEOptions{InitialParams: parameterized.Params{"theta": math.Inf(-1)}}},
		{"huge InitialParams", VQEOptions{InitialParams: parameterized.Params{"theta": math.Ldexp(1, 60)}}},
		{"huge negative InitialParams", VQEOptions{InitialParams: parameterized.Params{"theta": -math.Nextafter(math.Ldexp(1, 26), math.Inf(1))}}},
	}
```

In `TestVQEOptionErrorNamesTheField`, replace the last row of `cases` and the closing brace

```go
		{"InitialParams", VQEOptions{InitialParams: parameterized.Params{"theta": math.Inf(-1)}}, `initial parameter "theta" has non-finite value -Inf`},
	}
```

with

```go
		{"InitialParams", VQEOptions{InitialParams: parameterized.Params{"theta": math.Inf(-1)}}, `initial parameter "theta" has non-finite value -Inf`},
		{"InitialParams magnitude", VQEOptions{InitialParams: parameterized.Params{"theta": math.Ldexp(1, 60)}}, `initial parameter "theta" must have magnitude at most 2^26 (6.7108864e+07), beyond which float64 cannot resolve the +/- pi/2 parameter shift, got 1.152921504606847e+18`},
	}
```

Then append to the end of the file (after the closing brace of `TestParameterShiftEvaluationCountOnFailure`):

```go

// TestParameterShiftRejectsUnresolvableAngle pins the magnitude bound on
// the angles parameterShiftGradient shifts (backlog item 20). Before the
// check, a = 2^60 returned {a: 0} after two evaluations with a nil error:
// float64 spacing there is 256, so a +/- pi/2 rounded back to a and both
// evaluations bound the same circuit. Now any shifted name whose finite
// value has magnitude beyond maxShiftMagnitude (2^26) is rejected before
// any evaluation; at the bound itself the rule is still exact to the
// spacing (7.5e-9 rad), pinned against the analytic slope of cos; an
// unshifted huge value is harmless because it enters both evaluations of
// every other name identically; and a non-finite value stays Bind's.
func TestParameterShiftRejectsUnresolvableAngle(t *testing.T) {
	bound := math.Ldexp(1, 26)
	if bound != maxShiftMagnitude {
		t.Fatalf("maxShiftMagnitude = %v, want 2^26 = %v", float64(maxShiftMagnitude), bound)
	}
	h := H2Hamiltonian()
	tmpl := gradientTargetTemplate()

	rejected := []struct {
		name string
		a    float64
	}{
		{"2^60, where both shifts round back", math.Ldexp(1, 60)},
		{"-2^60", -math.Ldexp(1, 60)},
		{"2^54, where the spacing first exceeds pi", math.Ldexp(1, 54)},
		{"one ulp past the bound", math.Nextafter(bound, math.Inf(1))},
		{"one ulp past the negative bound", -math.Nextafter(bound, math.Inf(1))},
	}
	for _, c := range rejected {
		t.Run(c.name, func(t *testing.T) {
			grad, evals, err := parameterShiftGradient(h, tmpl, parameterized.Params{"a": c.a, "b": 0.1}, []string{"a"})
			var ve *InvalidVQEInputError
			if !errors.As(err, &ve) {
				t.Fatalf("a=%v: grad = %v, evals = %d, err = %v (%T); want InvalidVQEInputError", c.a, grad, evals, err, err)
			}
			if grad != nil || evals != 0 {
				t.Errorf("a=%v: grad = %v, evals = %d; want nil and 0, the check precedes the first evaluation", c.a, grad, evals)
			}
			want := fmt.Sprintf("parameter %q cannot be shifted by +/- pi/2: its magnitude must be at most 2^26 (%v), got %v", "a", bound, c.a)
			if ve.Reason != want {
				t.Errorf("Reason = %q, want %q", ve.Reason, want)
			}
			if cause := errors.Unwrap(err); cause != nil {
				t.Errorf("errors.Unwrap(err) = %v, want nil for a problem the helper detected itself", cause)
			}
		})
	}

	t.Run("first offender in names order", func(t *testing.T) {
		_, evals, err := parameterShiftGradient(h, tmpl, parameterized.Params{"a": math.Ldexp(1, 60), "b": -math.Ldexp(1, 60)}, []string{"b", "a"})
		var ve *InvalidVQEInputError
		if !errors.As(err, &ve) || evals != 0 {
			t.Fatalf("err = %v, evals = %d; want InvalidVQEInputError after 0 evaluations", err, evals)
		}
		if !strings.HasPrefix(ve.Reason, `parameter "b" `) {
			t.Fatalf("Reason = %q, want it to name %q, the first offender in names order", ve.Reason, "b")
		}
	})

	// E = cos(theta) on H2Ansatz against Z0, so dE/dtheta = -sin(theta).
	// The shift at |theta| = 2^26 lands within half the spacing (7.5e-9)
	// of +/- pi/2, and the slope of cos is at most 1, so the rule is
	// exact to 1e-8.
	for _, theta := range []float64{bound, -bound} {
		grad, evals, err := parameterShiftGradient(zOnQubit0(1), H2Ansatz(), parameterized.Params{"theta": theta}, []string{"theta"})
		if err != nil {
			t.Fatalf("theta=%v is at the bound and must be accepted: %v", theta, err)
		}
		if evals != 2 {
			t.Fatalf("theta=%v: evals = %d, want 2", theta, evals)
		}
		if want := -math.Sin(theta); math.Abs(grad["theta"]-want) > 1e-8 {
			t.Errorf("theta=%v: gradient %v, want -sin(theta) = %v within 1e-8", theta, grad["theta"], want)
		}
	}

	t.Run("unshifted huge value is harmless", func(t *testing.T) {
		params := parameterized.Params{"a": math.Ldexp(1, 60), "b": 0.1}
		grad, evals, err := parameterShiftGradient(h, tmpl, params, []string{"b"})
		if err != nil || evals != 2 {
			t.Fatalf("grad = %v, evals = %d, err = %v; want a gradient for b after 2 evaluations", grad, evals, err)
		}
		const hstep = 1e-6
		ePlus, err := evaluate(h, tmpl, parameterized.Params{"a": params["a"], "b": params["b"] + hstep})
		if err != nil {
			t.Fatalf("evaluate plus: %v", err)
		}
		eMinus, err := evaluate(h, tmpl, parameterized.Params{"a": params["a"], "b": params["b"] - hstep})
		if err != nil {
			t.Fatalf("evaluate minus: %v", err)
		}
		fd := (ePlus - eMinus) / (2 * hstep)
		if math.Abs(grad["b"]-fd) > 1e-6 {
			t.Errorf("gradient of b at a=2^60: parameter-shift %v != finite-diff %v", grad["b"], fd)
		}
	})

	t.Run("non-finite value stays Bind's", func(t *testing.T) {
		_, evals, err := parameterShiftGradient(h, tmpl, parameterized.Params{"a": math.Inf(1), "b": 0.1}, []string{"a"})
		var pe *parameterized.InvalidParameterValueError
		if !errors.As(err, &pe) || pe.Name != "a" || evals != 0 {
			t.Fatalf("err = %v (%T), evals = %d; want Bind's InvalidParameterValueError for %q after 0 evaluations", err, err, evals, "a")
		}
	})
}

// TestVQEHugeInitialParamIsRejected pins the driver side of backlog item
// 20. Before the check, a = 2^60 passed the finiteness guard, its gradient
// came back exactly 0, the descent step of 0.03 was lost to the spacing of
// 256, and VQE reported Converged after one iteration with a frozen at its
// start. Now the value is rejected up front with a Reason that names the
// bound and the value; at the bound itself the run proceeds.
func TestVQEHugeInitialParamIsRejected(t *testing.T) {
	h := H2Hamiltonian()
	tmpl := gradientTargetTemplate()
	huge := math.Ldexp(1, 60)

	res, err := VQE(h, tmpl, VQEOptions{InitialParams: parameterized.Params{"a": huge, "b": 0.1}})
	var ve *InvalidVQEInputError
	if !errors.As(err, &ve) {
		t.Fatalf("VQE(a=%v): result = %+v, err = %v (%T); want InvalidVQEInputError, not a Converged result with a frozen", huge, res, err, err)
	}
	want := fmt.Sprintf("initial parameter %q must have magnitude at most 2^26 (%v), beyond which float64 cannot resolve the +/- pi/2 parameter shift, got %v", "a", float64(maxShiftMagnitude), huge)
	if ve.Reason != want {
		t.Fatalf("Reason = %q, want %q", ve.Reason, want)
	}
	if cause := errors.Unwrap(err); cause != nil {
		t.Fatalf("errors.Unwrap(err) = %v, want nil for a problem VQE detected itself", cause)
	}

	atBound := math.Ldexp(1, 26)
	res, err = VQE(h, tmpl, VQEOptions{InitialParams: parameterized.Params{"a": atBound, "b": 0.1}, MaxIterations: 3})
	if err != nil {
		t.Fatalf("VQE(a=%v) at the bound must run: %v", atBound, err)
	}
	if res.Evaluations < 1+3*res.Iterations {
		t.Fatalf("Evaluations = %d, want >= 1+3*%d: the gradient at the bound was evaluated", res.Evaluations, res.Iterations)
	}
}

// TestVQEStepBeyondShiftBoundIsReported pins the step guard's second
// clause (backlog item 20): a descent that carries a parameter past 2^26
// stops before the next gradient would shift an angle float64 cannot
// resolve, so parameterShiftGradient's own check is unreachable from VQE
// and no error arrives wrapped. The start sits 0.125 below the bound;
// E = cos(theta) has slope -sin(theta) = -0.53 there, so the default 0.3
// step moves theta up by 0.16 and crosses on iteration 0.
func TestVQEStepBeyondShiftBoundIsReported(t *testing.T) {
	h := zOnQubit0(1)
	tmpl := H2Ansatz()
	start := math.Ldexp(1, 26) - 0.125
	params := parameterized.Params{"theta": start}
	grad, _, err := parameterShiftGradient(h, tmpl, params, tmpl.ParamNames())
	if err != nil {
		t.Fatalf("setup gradient: %v", err)
	}
	stepped := start - 0.3*grad["theta"]
	if !(stepped > maxShiftMagnitude) {
		t.Fatalf("setup: the step reaches %v, want beyond %v (gradient %v)", stepped, float64(maxShiftMagnitude), grad["theta"])
	}

	var ve *InvalidVQEInputError
	res, err := VQE(h, tmpl, VQEOptions{InitialParams: params, MaxIterations: 3})
	if !errors.As(err, &ve) {
		t.Fatalf("result = %+v, err = %v (%T); want InvalidVQEInputError", res, err, err)
	}
	want := fmt.Sprintf("the step of parameter %q reaches %v at iteration 0, beyond the 2^26 (%v) within which float64 resolves the +/- pi/2 parameter shift (step size %v times gradient %v)", "theta", stepped, float64(maxShiftMagnitude), 0.3, grad["theta"])
	if ve.Reason != want {
		t.Fatalf("Reason = %q, want %q", ve.Reason, want)
	}
	if cause := errors.Unwrap(err); cause != nil {
		t.Fatalf("errors.Unwrap(err) = %v, want nil for a problem VQE detected itself", cause)
	}
}
```

- [ ] **Step 3: Run the tests to verify they fail**

Run:

```bash
go test ./algorithm -run 'TestParameterShiftRejectsUnresolvableAngle|TestVQEHugeInitialParamIsRejected|TestVQEStepBeyondShiftBoundIsReported|TestVQEOption' 2>&1 | head -8
go vet -tags redtests ./algorithm
```

Expected: the package fails to build with `undefined: maxShiftMagnitude` on six lines of `algorithm/vqe_test.go` (the constant does not exist yet). The `go vet` under the `redtests` tag compiles cleanly with the tagged file gone.

To see the behavioral failure the tests pin (optional, not required to proceed): temporarily append `const maxShiftMagnitude = 1 << 26` to the end of `algorithm/vqe.go`, rerun the first command with `-v`, and expect `TestParameterShiftRejectsUnresolvableAngle` to fail on its five rejected rows and its ordering row (`grad = map[a:0], evals = 2, err = <nil>` at `2^60`; `grad = map[a:-0.07969369734038356]` at `2^54`; a nonzero slope at one ulp past the bound), `TestVQEOptionValidation` on its two new rows and `TestVQEOptionErrorNamesTheField` on its new row (`err = <nil>` with a `Converged` result), `TestVQEHugeInitialParamIsRejected` with `Converged:true` after 123 iterations and `a` still `1.152921504606847e+18`, and `TestVQEStepBeyondShiftBoundIsReported` with `Converged:false` after 3 iterations and a nil error. Remove the temporary constant before Step 4.

- [ ] **Step 4: Implement the bound**

Nine edits in `algorithm/vqe.go`, in file order. Each "replace" block matches exactly once at `0712b96`.

(a) Insert the constant directly before `parameterShiftGradient`'s doc comment. Replace

```go
// parameterShiftGradient computes dE/dtheta per parameter via the exact
// parameter-shift rule: dE/dtheta = (E(theta+pi/2) - E(theta-pi/2)) / 2.
```

with

```go
// maxShiftMagnitude is the largest |theta| the parameter-shift rule
// shifts: 2^26. float64 rounds theta +/- pi/2 to a multiple of theta's
// spacing, so the shift lands within half that spacing of its true
// offset. Below 2^26 the spacing is at most 2^-27 (1.5e-8), the offset is
// exact to 7.5e-9 rad, and a gradient component is off by at most that
// times the largest slope, which the Hamiltonian's coefficient sum
// bounds. Beyond it the rule degrades until it fails: from 2^48 the
// default 0.3 step times a gradient of 0.1 rounds to no change, so VQE
// would freeze the parameter and report Converged; from 2^51 the shift is
// applied at the wrong offset (a multiple of 0.5 or coarser); from 2^54
// the spacing exceeds pi, theta +/- pi/2 rounds back to theta, both
// evaluations bind the same circuit, and the difference is exactly 0. The
// bound is the even split of the 53-bit significand, 26 bits before the
// binary point and 27 after: the parameter keeps a resolution finer than
// 1e-8 rad, and 2^26 rad is more than 10^7 turns, far beyond any angle a
// descent reaches from an angle a caller chose (200 iterations of a 0.3
// step times a unit gradient walk 60 rad). Larger angles are rejected,
// not reduced mod 2*pi: see parameterShiftGradient.
const maxShiftMagnitude = 1 << 26

// parameterShiftGradient computes dE/dtheta per parameter via the exact
// parameter-shift rule: dE/dtheta = (E(theta+pi/2) - E(theta-pi/2)) / 2.
```

(b) Insert the "Third" paragraph into the doc comment. Replace

```go
// TestParameterShiftIsBlindToRescaledFactory pins this failure mode.
//
// params must bind every parameter the template declares, the precondition
```

with

```go
// TestParameterShiftIsBlindToRescaledFactory pins this failure mode.
//
// Third, float64 must resolve the shift. theta +/- pi/2 is rounded to a
// multiple of theta's spacing, and from 2^54 that spacing exceeds pi, so
// both shifted values round back to theta, the two evaluations bind the
// same circuit, and the half difference is exactly 0 with a nil error,
// again indistinguishable from a stationary point. This one is checked:
// each name in names must be bound to a finite value of magnitude at most
// maxShiftMagnitude (2^26, where the shift lands within 7.5e-9 rad of its
// offset; the constant's comment derives the bound), or the call returns
// InvalidVQEInputError naming the first such name in names order, before
// any evaluation and with the count 0. Only shifted names are checked,
// because an unshifted value enters both evaluations identically, and
// only finite values, because a non-finite one is Bind's to reject as
// before. A larger angle is rejected rather than reduced mod 2*pi. The
// energy is 2*pi-periodic only under the previous paragraph's
// precondition, which this function cannot check, so a reduction would
// silently move the point evaluated for a factory of another period; and
// the reduction itself exceeds float64 (math.Mod(2^60, 2*pi) with the
// float64 constant is off by about 80 rad). A caller who knows the
// factory's period reduces before calling, and VQE rejects such an
// initial parameter up front so this check is unreachable from it.
//
// params must bind every parameter the template declares, the precondition
```

(c) Narrow the "completeness only" sentence to `Bind`'s rules. Replace

```go
// evaluation. It guarantees completeness only: that is the one rule of
// Bind's a shift can hide, since the copies keep every key of params, a
```

with

```go
// evaluation. Of Bind's rules it guarantees completeness only: that is
// the one a shift can hide, since the copies keep every key of params, a
```

(d) The doc comment's last sentence. Replace

```go
// error, so only a direct caller sees it. The completeness check above
// returns 0 because it precedes the first evaluation.
func parameterShiftGradient(
```

with

```go
// error, so only a direct caller sees it. The completeness and magnitude
// checks above return 0 because they precede the first evaluation.
func parameterShiftGradient(
```

(e) The check, directly after the item 16 loop. Replace

```go
			return nil, 0, &parameterized.MissingParameterError{Name: name}
		}
	}
	grad := make(map[string]float64, len(names))
```

with

```go
			return nil, 0, &parameterized.MissingParameterError{Name: name}
		}
	}
	for _, name := range names {
		if v := params[name]; isFinite(v) && math.Abs(v) > maxShiftMagnitude {
			return nil, 0, &InvalidVQEInputError{Reason: fmt.Sprintf("parameter %q cannot be shifted by +/- pi/2: its magnitude must be at most 2^26 (%v), got %v", name, float64(maxShiftMagnitude), v)}
		}
	}
	grad := make(map[string]float64, len(names))
```

(f) The `InitialParams` field comment. Replace

```go
	// InitialParams names starting angles. Missing declared parameters
	// default to 0. Unknown names and non-finite values are rejected.
```

with

```go
	// InitialParams names starting angles. Missing declared parameters
	// default to 0. Unknown names, non-finite values, and magnitudes
	// beyond 2^26 (see maxShiftMagnitude) are rejected.
```

(g) `VQE`'s doc comment, the validation paragraph. Replace

```go
// values outside their domains (see VQEOptions), initial parameters that
// are undeclared or non-finite, a parameter driving several gates, a
```

with

```go
// values outside their domains (see VQEOptions), initial parameters that
// are undeclared, non-finite, or beyond 2^26 in magnitude (where float64
// no longer resolves the +/- pi/2 shift; see maxShiftMagnitude, which
// says why such an angle is rejected rather than reduced mod 2*pi), a
// parameter driving several gates, a
```

and replace

```go
// InvalidVQEInputError blaming the Hamiltonian or StepSize. No error
// blames a parameter that was finite on entry.
func VQE(
```

with

```go
// InvalidVQEInputError blaming the Hamiltonian or StepSize. A stepped
// value that leaves the 2^26 bound is rejected the same way, naming the
// step that crossed it, so the gradient never shifts an angle it cannot
// resolve. No error blames a parameter that was finite on entry.
func VQE(
```

(h) The `InitialParams` loop. Replace

```go
			return nil, &InvalidVQEInputError{Reason: fmt.Sprintf("initial parameter %q has non-finite value %v", name, value)}
		}
		params[name] = value
```

with

```go
			return nil, &InvalidVQEInputError{Reason: fmt.Sprintf("initial parameter %q has non-finite value %v", name, value)}
		}
		if math.Abs(value) > maxShiftMagnitude {
			return nil, &InvalidVQEInputError{Reason: fmt.Sprintf("initial parameter %q must have magnitude at most 2^26 (%v), beyond which float64 cannot resolve the +/- pi/2 parameter shift, got %v", name, float64(maxShiftMagnitude), value)}
		}
		params[name] = value
```

(i) The step guard. Replace

```go
			if !isFinite(steps[name]) {
				return nil, &InvalidVQEInputError{Reason: fmt.Sprintf("StepSize is too large: the step of parameter %q overflows float64 at iteration %d (step size %v times gradient %v)", name, iter, step, grad[name])}
			}
		}
```

with

```go
			if !isFinite(steps[name]) {
				return nil, &InvalidVQEInputError{Reason: fmt.Sprintf("StepSize is too large: the step of parameter %q overflows float64 at iteration %d (step size %v times gradient %v)", name, iter, step, grad[name])}
			}
			// The same bound the initial parameters met: past it the next
			// gradient would shift an angle float64 cannot resolve, and
			// parameterShiftGradient would reject it inside the wrap.
			// Stopping here keeps that check unreachable from VQE and
			// names the step that crossed, since the caller chose neither
			// the point nor the iteration.
			if math.Abs(steps[name]) > maxShiftMagnitude {
				return nil, &InvalidVQEInputError{Reason: fmt.Sprintf("the step of parameter %q reaches %v at iteration %d, beyond the 2^26 (%v) within which float64 resolves the +/- pi/2 parameter shift (step size %v times gradient %v)", name, steps[name], iter, float64(maxShiftMagnitude), step, grad[name])}
			}
		}
```

No import changes: `fmt` and `math` are already imported.

- [ ] **Step 5: Run the tests to verify they pass**

Run:

```bash
go test ./algorithm -v 2>&1 | grep -E '^(--- |ok|FAIL)'
```

Expected: every test in the package reports `--- PASS`, including `TestParameterShiftRejectsUnresolvableAngle` (eight subtests plus the two at-the-bound checks), `TestVQEOptionValidation` (ten subtests), `TestVQEOptionErrorNamesTheField` (five), `TestVQEHugeInitialParamIsRejected`, `TestVQEStepBeyondShiftBoundIsReported`, and the unchanged `TestParameterShiftMatchesFiniteDifference`, `TestParameterShiftMissingParamErrorMatchesBind` (the helper's new check runs after the completeness check and sees complete bindings of order 1), and `TestVQEStepOverflowReasonBlamesStepSize` (the overflow clause still fires first for a non-finite step), then `ok`.

- [ ] **Step 6: Record the change in the CHANGELOG and amend the prior specs**

In `CHANGELOG.md`, under `## Unreleased`, `### Fixed`, insert a new entry at the end of the section, directly before the first `### Breaking` heading. Find the insertion point by the headings: the entry goes after whatever `####` entry is last under `### Fixed` (at `0712b96` that is `#### \`parameterized.Template\` rejects a gate declared with no targets`, whose last two lines are `unchanged. Design:` and `` `docs/superpowers/specs/2026-09-09-empty-target-list-design.md`. ``; if another item has appended an entry since, go after that one instead) and before `### Breaking`. Insert, with one blank line on each side:

```markdown
#### `algorithm.VQE` rejects an angle too large for the parameter shift

The parameter-shift gradient behind `VQE` evaluates `theta +/- pi/2` in
float64, and from 2^54 the spacing between adjacent values exceeds pi, so
both shifted values rounded back to `theta`, the two evaluations bound the
same circuit, and the gradient came back exactly 0 with a nil error;
below that, from about 2^48, the shift was applied at a rounded offset
and the descent step was lost to the spacing. A huge finite
`InitialParams` value therefore passed the finiteness guard, never moved,
and the run reported `Converged` with it frozen at its start. `VQE` now
rejects an initial parameter whose magnitude exceeds 2^26 (67,108,864
rad, where the shift is still exact to 7.5e-9 rad) with
`InvalidVQEInputError`, and stops with the same type when a descent step
carries a parameter past that bound; the unexported helper rejects such
an angle before any evaluation for direct callers. Angles are rejected,
not reduced mod 2*pi: the energy is periodic only under the
`exp(-i*theta*P/2)` factory convention `VQE` cannot check, and the
reduction itself is not computable in float64 at those magnitudes.
Callers with a larger angle reduce it themselves. Angles within the bound,
including every value a run from an angle of ordinary size reaches, are
unaffected. With its only test replaced by tests of the chosen contract,
`algorithm/backlog_red_numeric_test.go` is removed. Design:
`docs/superpowers/specs/2026-09-09-parameter-shift-angle-bound-design.md`.
```

In `docs/superpowers/specs/2026-09-08-vqe-input-validation-design.md`, four amendments. Each anchor matches exactly once.

(1) At the end of the "Step guard" paragraph, replace

```markdown
at any iteration. `Err` is nil.
```

with

```markdown
at any iteration. `Err` is nil. Amended 2026-09-09 (item 20): the guard
also rejects a stepped value whose magnitude exceeds 2^26, the bound
`parameterShiftGradient` enforces on the angles it shifts, so the
helper's own check stays unreachable from `VQE`
(`docs/superpowers/specs/2026-09-09-parameter-shift-angle-bound-design.md`,
Decision 3); that Reason names the step and the iteration rather than
`StepSize`, since a start near the bound crosses it at any step size.
```

(2) At the end of the "Order inside the loop" paragraph, replace

```markdown
four guards `VQE` never hands `Bind` a non-finite value, never compares a
non-finite energy, and never returns one.
```

with

```markdown
four guards `VQE` never hands `Bind` a non-finite value, never compares a
non-finite energy, and never returns one. Amended 2026-09-09 (item 20):
with the magnitude check on `InitialParams` and the step guard's second
clause, `VQE` also never hands `parameterShiftGradient` an angle beyond
2^26.
```

(3) In the Decision 3 table, after the `step overflow (added 2026-09-08)` row, add two rows:

```markdown
| initial parameter beyond the shift bound (added 2026-09-09, item 20) | `initial parameter "theta" must have magnitude at most 2^26 (6.7108864e+07), beyond which float64 cannot resolve the +/- pi/2 parameter shift, got 1.152921504606847e+18` |
| step beyond the shift bound (added 2026-09-09, item 20) | `the step of parameter "theta" reaches 6.710886403417353e+07 at iteration 0, beyond the 2^26 (6.7108864e+07) within which float64 resolves the +/- pi/2 parameter shift (step size 0.3 times gradient -0.5305784211229922)` |
```

(4) At the end of the "Helper shape and the items that follow" paragraph, replace

```markdown
tagged in `algorithm/backlog_red_numeric_test.go`.
```

with

```markdown
tagged in `algorithm/backlog_red_numeric_test.go`.
Amended 2026-09-09 (item 20): the huge-angle defect was closed by a
magnitude bound of 2^26 on every angle the helper shifts
(`docs/superpowers/specs/2026-09-09-parameter-shift-angle-bound-design.md`);
this time `VQE` does change, in two places this design's shape
anticipated: the inline `InitialParams` check gains a magnitude clause
beside its finiteness clause, and the step guard gains a second clause,
so the helper's check is unreachable from `VQE` as items 16 and 17's
were. Two Decision 3 rows record the Reasons. Item 20's red test was
ruled contradicted by the chosen contract and deleted with its file.
```

In `docs/superpowers/specs/2026-09-08-parameter-shift-missing-param-design.md`, two amendments.

(1) In "Round 1 red tests", the bullet for `TestRedParameterShiftAtHugeAngleMatchesReducedAngle` ends with `to parameter completeness.`; replace that line

```markdown
  to parameter completeness.
```

with

```markdown
  to parameter completeness. Amended 2026-09-09 (item 20): deleted, not
  moved. The test asserted a gradient at `a = 2^60` equal to the slope at
  the angle reduced mod 2*pi; item 20 chose an error for any shifted angle
  beyond 2^26 rather than a reduction
  (`docs/superpowers/specs/2026-09-09-parameter-shift-angle-bound-design.md`,
  "Red test ruling"), and its file, empty after the deletion, went with
  it.
```

(2) At the end of Decision 5 (its last paragraph ends `match; no assertion changed.`, directly before `## Item 17 follows on locally`), replace

```markdown
match; no assertion changed.
```

with

```markdown
match; no assertion changed.

Amended 2026-09-09 (item 20): the helper now runs a second check of its
own ahead of the loop, a magnitude bound of 2^26 on the finite value of
each name it shifts
(`docs/superpowers/specs/2026-09-09-parameter-shift-angle-bound-design.md`,
Decision 2). That bound is the helper's precondition, not one of `Bind`'s,
so "completeness only" stands as a statement about `Bind`'s rules and the
doc comment now says "Of Bind's rules it guarantees completeness only".
Non-finite values are still left to `Bind`: the magnitude check skips
them, so every sentence above about where a non-finite value is reported
holds as written.
```

In `docs/superpowers/specs/2026-09-08-parameter-shift-evaluation-count-design.md`, two amendments.

(1) In "Effect on the red-test file", the round 1 note ends `deleted \`backlog_red_test.go\`, that carries the tag.`; replace

```markdown
deleted `backlog_red_test.go`, that carries the tag.
```

with

```markdown
deleted `backlog_red_test.go`, that carries the tag. Amended 2026-09-09
(item 20): that file is now deleted too, its test ruled contradicted by
the contract item 20 chose
(`docs/superpowers/specs/2026-09-09-parameter-shift-angle-bound-design.md`,
"Red test ruling"), so the prediction above holds again: the red run of
`algorithm` reports `[no tests to run]`.
```

(2) At the end of Decision 5, replace

```markdown
to an error, who can observe it, and how the item 16 return fits. Nothing
else in the comment changes.
```

with

```markdown
to an error, who can observe it, and how the item 16 return fits. Nothing
else in the comment changes. Amended 2026-09-09 (item 20): the last
sentence now reads "The completeness and magnitude checks above return 0
because they precede the first evaluation", since the helper gained a
second pre-loop check (a magnitude bound on shifted angles) that returns
the literal `0` for the same reason.
```

In `docs/superpowers/specs/2026-08-30-parameter-binding-vqe-design.md`, one amendment: in the `VQE` "Errors" list, after the bullet that ends `Direct callers of \`Bind\`, \`Execute\`, and \`Energy\` are\n  unaffected.`, add a bullet. Replace

```markdown
  aborts the loop. Direct callers of `Bind`, `Execute`, and `Energy` are
  unaffected.
```

with

```markdown
  aborts the loop. Direct callers of `Bind`, `Execute`, and `Energy` are
  unaffected.
- Amended 2026-09-09 (backlog item 20): the parameter shift above is
  computed in float64, and from 2^54 `theta +/- pi/2` rounds back to
  `theta`, so `VQE` rejects an initial parameter of magnitude beyond 2^26
  with `InvalidVQEInputError` and stops the same way when a step carries a
  parameter past that bound; angles are not reduced mod 2*pi because the
  energy is periodic only under the factory convention `VQE` cannot check
  (`docs/superpowers/specs/2026-09-09-parameter-shift-angle-bound-design.md`).
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
go test -race ./algorithm
go test -tags redtests ./algorithm -run '^TestRed' 2>&1 | grep -E '^(--- FAIL|ok|FAIL)'
go test -tags redtests ./parameterized -run '^TestRed' 2>&1 | grep -E '^(--- FAIL|ok|FAIL)'
go run ./cmd/quantum -demo qaoa | grep -A1 'VQE from'
git status --short
```

Expected: `gofmt -l` prints nothing; vet (both), staticcheck, build, the full suite, and the race run are clean; the `algorithm` red run prints `ok  	github.com/pjbaur/quantum/algorithm ... [no tests to run]` and no `--- FAIL` line; the `parameterized` red run's outcome depends on which other items have landed (at `0712b96` it lists `--- FAIL: TestRedCopyAfterDeclarationKeepsNamesAndCountsConsistent`, item 21): record what it prints, do not act on it; the demo prints `VQE from the symmetric start: energy = -1.0000, expected cut = 2.0000` and then `  iterations accepted: 29, energy evaluations: 378`; `git status` lists `D  algorithm/backlog_red_numeric_test.go` (staged by `git rm`), ` M algorithm/vqe.go`, ` M algorithm/vqe_test.go`, ` M CHANGELOG.md`, and ` M` for the four spec files named under **Files**. If it also lists files outside that set, another item's implementer is working; leave them unstaged. If the `algorithm` red run lists any test or the demo lines differ, stop and report: something outside this item changed.

- [ ] **Step 8: Commit**

```bash
git add algorithm/vqe.go algorithm/vqe_test.go algorithm/backlog_red_numeric_test.go CHANGELOG.md docs/superpowers/specs/2026-09-08-vqe-input-validation-design.md docs/superpowers/specs/2026-09-08-parameter-shift-missing-param-design.md docs/superpowers/specs/2026-09-08-parameter-shift-evaluation-count-design.md docs/superpowers/specs/2026-08-30-parameter-binding-vqe-design.md
git commit -m "fix(algorithm): reject angles the parameter shift cannot resolve

parameterShiftGradient formed theta +/- pi/2 in float64, and from 2^54
the spacing between adjacent values exceeds pi, so both shifted values
rounded back to theta, the two evaluations bound the same circuit, and
the gradient came back exactly 0 with a nil error; from about 2^48 the
shift was applied at a rounded offset and the descent step was lost to
the spacing. A huge finite InitialParams value passed VQE's finiteness
guard, never moved, and the run reported Converged with it frozen.

A new unexported constant maxShiftMagnitude = 2^26 bounds every angle the
rule shifts; its comment derives the bound (the shift is exact to 7.5e-9
rad below it; the even split of the 53-bit significand; 10^7 turns of
headroom). The helper rejects a shifted name bound beyond it with
InvalidVQEInputError before any evaluation, finite values only so that
non-finite ones stay Bind's. VQE rejects such an initial parameter up
front and stops when a step carries a parameter past the bound, so the
helper's check is unreachable from VQE. Angles are rejected, not reduced
mod 2*pi: the energy is periodic only under the exp(-i*theta*P/2)
convention VQE cannot check, a reduction inside the helper would leave
VQE frozen anyway, and math.Mod(2^60, 2*pi) is off by about 80 rad.

TestRedParameterShiftAtHugeAngleMatchesReducedAngle asserted the reduced
slope and is deleted as contradicted by the chosen contract, and
algorithm/backlog_red_numeric_test.go, now empty, with it; tests of the
bound replace it. Backlog item 20.

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01XMPFmYskZD8uji6xSh5x11"
```

`git add` on the deleted path is a no-op after `git rm` and stages the removal if `rm` was used instead; either way the deletion is in the commit. If the commit fails with `.git/index.lock` present, another agent is committing; wait a few seconds and run the `git commit` again unchanged.

---

## Self-Review

- **Spec coverage.** Decision 1 (error, not reduction; no reduction anywhere): Global Constraints' deletion rule and the "Third" paragraph in Step 4b; no test or code reduces an angle. Decision 2 (`2^26`, derivation in the constant's comment, printed as `2^26 (6.7108864e+07)`): Step 4a and every Reason in 4e, 4h, 4i; `TestParameterShiftRejectsUnresolvableAngle` pins the constant's value and the at-the-bound accuracy. Decision 3 (helper pre-loop check on `names`, finite only, `InvalidVQEInputError`, 0 evaluations; `VQE` up front; `VQE` step guard; helper unreachable from `VQE`): Steps 4e, 4h, 4i and their tests in Step 2 (rejected rows, ordering row, non-finite row; `TestVQEHugeInitialParamIsRejected`; `TestVQEStepBeyondShiftBoundIsReported` with `errors.Unwrap` nil proving nothing arrived wrapped). Decision 4 (Reason strings): every Reason in Step 4 appears verbatim in a Step 2 assertion, and the two `VQE` Reasons in the Decision 3 rows of Step 6. Rulings: at the bound accepted (both signs, both call paths); one ulp past rejected (both signs); unshifted huge value harmless; non-finite stays `Bind`'s; `names` order; crossing step rejected before evaluation (the test's `MaxIterations: 3` with iteration 0 in the Reason). Red test ruling (deleted, file deleted, `[no tests to run]`, `go vet -tags redtests` clean): Global Constraints, Step 2's `git rm`, Step 3's vet, Step 7's red run. Effect on callers: Step 5 names the unchanged tests that exercise the new code on ordinary angles; Step 7 runs the driver tests through `go test ./...` and the QAOA demo. Backward compatibility: CHANGELOG entry under `### Fixed` (Step 6); no ADR amendment (none planned, by design); the four spec amendments, each in Step 6 with its anchor. Testing section: all five test changes appear in Step 2; the red-tag expectations are in Step 7.
- **Placeholder scan.** No TBD/TODO; every code and doc step carries its full text; every command names its expected result.
- **Type consistency.** `parameterShiftGradient(h *Hamiltonian, t *parameterized.Template, params parameterized.Params, names []string) (map[string]float64, int, error)` matches `algorithm/vqe.go:98` and every call in Step 2. `VQE(h *Hamiltonian, t *parameterized.Template, opts VQEOptions) (*VQEResult, error)` and `VQEResult.Evaluations`/`Iterations` match `algorithm/vqe.go`. `*InvalidVQEInputError` and `*parameterized.InvalidParameterValueError` have pointer-receiver `Error()` methods, so the `errors.As` targets in Step 2 are pointers to those pointer types, as `TestVQEStepOverflowReasonBlamesStepSize` and `TestVQENonFiniteHamiltonianIsNotBlamedOnParams` already do. `maxShiftMagnitude` is an untyped integer constant, so `math.Abs(v) > maxShiftMagnitude` and `bound != maxShiftMagnitude` compare as float64 and `float64(maxShiftMagnitude)` formats as `6.7108864e+07`. `zOnQubit0`, `gradientTargetTemplate`, `H2Ansatz`, `H2Hamiltonian`, `evaluate`, and `isFinite` are the existing helpers named in **Interfaces**. `errors`, `fmt`, `math`, `strings`, and `parameterized` are already imported by `algorithm/vqe_test.go`. The test names cited in Steps 3, 5, and 7 match the functions written in Step 2.
- **Prototype.** Every code, test, and documentation block above was applied to a scratch copy of the repository at `0712b96` in this exact sequence: Step 2 (the build failure of Step 3 as stated; with the bare constant appended, the behavioral failures as stated), Step 4 green (Step 5 as stated), Step 6 anchors matched exactly once each, then `gofmt -l`, `go vet` (plain and `-tags redtests` on both packages), `staticcheck`, `go build`, `go test ./...`, `go test -race ./algorithm`, both red-tag runs (`algorithm` `[no tests to run]`; `parameterized` failing on item 21's test), and the QAOA demo lines, all as stated in Step 7.
