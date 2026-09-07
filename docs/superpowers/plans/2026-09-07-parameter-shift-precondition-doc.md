# Parameter-Shift Precondition Doc (Backlog Item 13) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close backlog item 13 by documenting, on `parameterShiftGradient` in `algorithm/vqe.go`, the precondition that a bound value must enter its gate as exp(-i*theta*P/2), why the driver cannot detect a factory that rescales, and that `QAOATemplate` is the precedent; pin the documented failure mode with one characterization test.

**Architecture:** Documentation-only change to `algorithm/vqe.go` (the doc comment on `parameterShiftGradient`, lines 28-37, is replaced; no code changes). One new test in `algorithm/vqe_test.go` builds a two-parameter template whose factories bind `2*value` and asserts the parameter-shift gradient is identically zero while the finite difference is not, and that VQE reports `Converged` after a one-iteration no-op. Backlog item 13 in `docs/enhancement-backlog-2026-08-27.md` is ticked with a Done note in the file's existing style.

**Tech Stack:** Go standard library only. No new dependencies.

**Spec:** none (trivial item; requirement is the backlog item text)

The requirement, verbatim from `.superpowers/backlog/enhancement-backlog-2026-08-27/item-13.md`:

> 13. **`parameterShiftGradient` precondition doc** — `vqe.go` should
> state that the bound value must enter the gate as exp(-i*theta*P/2); the
> driver cannot detect a rescaling factory. First occurrence: QAOA template.

Background the implementer needs (from commit `cf09732`, "bind QAOA angles directly so parameter-shift gradients are live"): the QAOA template factories originally bound `2*value` into `gates.NewRz`/`gates.NewRx`. That gave the bound value period pi, so VQE's +/- pi/2 shift saw E(theta+pi/2) == E(theta-pi/2) for every parameter, an identically zero gradient. VQE took a zero step, measured |delta E| = 0 below Tolerance, and reported Converged without moving. The fix bound the value unscaled and documented the contract on `QAOATemplate`; item 13 asks for the same contract to be stated where the rule actually lives.

## Classification

**Trivial.** Documentation only plus one test; no behavior change in any non-test file. No design doc. One task.

## Global Constraints

- Doc comments match the codebase voice: explain the why, cite conventions (exp(-i*theta*P/2), `gates.NewRx/NewRy/NewRz`, `parameterized.Factory`), no filler.
- Never edit a failing test to make code pass. The new test is a characterization test of behavior that already holds; if it fails, the doc comment is wrong about the code, not the other way round. Report the failure.
- `algorithm/vqe.go` changes are comment-only. The diff of that file must contain no non-comment lines (verified in Step 6).
- No changes to `parameterized/`, `gates/`, `algorithm/qaoa.go`, or any file outside the three listed in Task 1.
- No new dependencies. Go standard library only.
- Every task: `gofmt -l` clean on edited files, `go vet ./algorithm`, `go build ./...`, `go test ./...` green, then commit.
- Do not edit `.superpowers/backlog/` (the run's ledger belongs to the run lead); tick the item in `docs/enhancement-backlog-2026-08-27.md`, which is the repository's record.

---

### Task 1: Document the exp(-i*theta*P/2) precondition on `parameterShiftGradient` and pin its failure mode

**Sizing estimate:** about 550 lines of existing code to read (`algorithm/vqe.go` 198, `algorithm/vqe_test.go` 91, `algorithm/qaoa.go` lines 85-150, `parameterized/parameterized.go` lines 1-110, `gates/gates.go` lines 110-160, `docs/enhancement-backlog-2026-08-27.md` lines 194-235); 2 non-test files modified (`algorithm/vqe.go`, `docs/enhancement-backlog-2026-08-27.md`) plus 1 test file; net diff about 135 lines (+25 doc comment, +15 backlog note, +95 test); one test cycle. Well under a quarter of a context window.

**Files:**
- Modify: `algorithm/vqe.go:28-37` (the doc comment on `parameterShiftGradient`; nothing else)
- Modify: `docs/enhancement-backlog-2026-08-27.md:232-234` (item 13 checkbox and Done note)
- Test: `algorithm/vqe_test.go` (append one helper and one test; add one import)

**Interfaces:**
- Consumes (all existing; do not change any signature):
  - `func parameterShiftGradient(h *Hamiltonian, t *parameterized.Template, params parameterized.Params, names []string) (map[string]float64, int, error)` in `algorithm/vqe.go:38`
  - `func evaluate(h *Hamiltonian, t *parameterized.Template, params parameterized.Params) (float64, error)` in `algorithm/vqe.go:13`
  - `func VQE(h *Hamiltonian, t *parameterized.Template, opts VQEOptions) (*VQEResult, error)` in `algorithm/vqe.go:107`; `VQEOptions{InitialParams, MaxIterations}`; `VQEResult{Energy, Params, Iterations, Converged}`
  - `func H2Hamiltonian() *Hamiltonian` (defined in `algorithm/h2.go`, already used by `algorithm/vqe_test.go`)
  - `type Factory func(value float64) quantum.Gate` in `parameterized/parameterized.go:20`; `(*Template).AddParamGate(name string, factory Factory, targets ...int) error`; `(*Template).AddGate(gate quantum.Gate, targets ...int) error`; `(*Template).ParamNames() []string`
  - `gates.NewRy(theta float64) *MatrixGate`, `gates.NewRx(theta float64) *MatrixGate`, `gates.NewCNOT()`
- Produces: no new exported API. Two new unexported test identifiers whose names the doc comment cites, so they must match exactly:
  - `func rescaledGradientTemplate() *parameterized.Template` (test helper)
  - `func TestParameterShiftIsBlindToRescaledFactory(t *testing.T)`

- [ ] **Step 1: Read the context the doc comment cites**

Read these, in this order, so the comment you write is grounded in the code's own conventions:

1. `algorithm/vqe.go` in full. Note the current doc comment (lines 28-37) claims "every parameterized factory is a generators-of-Pauli rotation". That is only true of the stock `parameterized.Rx/Ry/Rz/Phase` factories; a `parameterized.Factory` is any `func(value float64) quantum.Gate`, so the claim is what the new comment corrects.
2. `parameterized/parameterized.go` lines 1-110: `Factory` (line 20), the stock factories (lines 22-32, which delegate unscaled to `gates.NewRx` and friends), `ParamStepCounts` (line 71), `AddParamGate` (line 93).
3. `gates/gates.go` lines 110-160: `NewRx`/`NewRy`/`NewRz` are `cos(θ/2)·I − i·sin(θ/2)·P`, i.e. exp(-i*theta*P/2); `NewPhase(φ)` is `Rz(φ)` up to a global phase (stated on `NewRz`).
4. `algorithm/qaoa.go` lines 85-150: the `QAOATemplate` doc comment states the contract from the template's side; the two factory closures bind `value` unscaled with comments saying why.
5. `algorithm/vqe_test.go` in full: `gradientTargetTemplate` and `TestParameterShiftMatchesFiniteDifference` are the positive test (convention-following factories agree with finite difference). The new test is its negative twin.
6. `docs/enhancement-backlog-2026-08-27.md` lines 194-235: the `> **Done (date)**:` blockquote style every closed item uses.

- [ ] **Step 2: Write the characterization test**

In `algorithm/vqe_test.go`, add `"github.com/pjbaur/quantum/quantum"` to the import block (it is needed for `quantum.Gate` in the factory closures), so the block reads:

```go
import (
	"math"
	"testing"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/parameterized"
	"github.com/pjbaur/quantum/quantum"
)
```

Then append the following to the end of the file:

```go
// rescaledGradientTemplate is gradientTargetTemplate with every factory
// binding twice its parameter (Ry(2a), Rx(2b)), the rescaling that
// parameterShiftGradient's doc comment forbids. The stock parameterized
// factories bind unscaled by construction, so the rescale is written as a
// closure over the gates constructors, exactly how a caller would
// introduce the bug.
func rescaledGradientTemplate() *parameterized.Template {
	tmpl := parameterized.NewTemplate(2)
	if err := tmpl.AddParamGate("a", func(v float64) quantum.Gate { return gates.NewRy(2 * v) }, 0); err != nil {
		panic(err)
	}
	if err := tmpl.AddGate(gates.NewCNOT(), 0, 1); err != nil {
		panic(err)
	}
	if err := tmpl.AddParamGate("b", func(v float64) quantum.Gate { return gates.NewRx(2 * v) }, 1); err != nil {
		panic(err)
	}
	return tmpl
}

// TestParameterShiftIsBlindToRescaledFactory pins the failure mode behind
// parameterShiftGradient's exp(-i*theta*P/2) precondition. A factory that
// binds 2*theta makes E periodic in theta with period pi, so the +/- pi/2
// shift difference vanishes: the returned gradient is identically zero
// even where the finite-difference slope clearly is not. Nothing in the
// driver can tell that from a stationary point, so VQE accepts a zero
// step, sees |delta E| = 0 < Tolerance, and reports Converged at its
// starting parameters. If this test ever fails, the doc comment on
// parameterShiftGradient describes behavior that no longer holds.
func TestParameterShiftIsBlindToRescaledFactory(t *testing.T) {
	h := H2Hamiltonian()
	tmpl := rescaledGradientTemplate()

	for _, base := range []float64{-1.2, 0.0, 0.45, 1.7} {
		params := parameterized.Params{"a": base, "b": base/2 + 0.1}
		grad, _, err := parameterShiftGradient(h, tmpl, params, tmpl.ParamNames())
		if err != nil {
			t.Fatalf("parameterShiftGradient(base=%v): %v", base, err)
		}
		for _, name := range []string{"a", "b"} {
			const hstep = 1e-6
			plus := parameterized.Params{"a": params["a"], "b": params["b"]}
			minus := parameterized.Params{"a": params["a"], "b": params["b"]}
			plus[name] += hstep
			minus[name] -= hstep
			ePlus, err := evaluate(h, tmpl, plus)
			if err != nil {
				t.Fatalf("evaluate plus: %v", err)
			}
			eMinus, err := evaluate(h, tmpl, minus)
			if err != nil {
				t.Fatalf("evaluate minus: %v", err)
			}
			fd := (ePlus - eMinus) / (2 * hstep)
			// Every base point above sits on a real slope (|dE/dtheta| is
			// at least 0.15 for both parameters); the threshold keeps the
			// test from silently drifting onto a genuine stationary point,
			// where a zero gradient would prove nothing.
			if math.Abs(fd) < 0.1 {
				t.Fatalf("base=%v param=%s: finite difference %v too small to distinguish from a stationary point", base, name, fd)
			}
			if math.Abs(grad[name]) > 1e-9 {
				t.Errorf("base=%v param=%s: parameter-shift gradient %v, want 0 (a period-pi parameter is invisible to the +/- pi/2 shift)", base, name, grad[name])
			}
		}

		start, err := evaluate(h, tmpl, params)
		if err != nil {
			t.Fatalf("evaluate start: %v", err)
		}
		result, err := VQE(h, tmpl, VQEOptions{InitialParams: params, MaxIterations: 50})
		if err != nil {
			t.Fatalf("VQE(base=%v): %v", base, err)
		}
		if !result.Converged || result.Iterations != 1 {
			t.Errorf("base=%v: VQE Converged=%v Iterations=%d, want a one-iteration no-op reported as Converged", base, result.Converged, result.Iterations)
		}
		if math.Abs(result.Energy-start) > 1e-12 {
			t.Errorf("base=%v: VQE energy %v moved from start %v despite a zero gradient", base, result.Energy, start)
		}
		for name, v := range params {
			if math.Abs(result.Params[name]-v) > 1e-12 {
				t.Errorf("base=%v: VQE moved %s from %v to %v despite a zero gradient", base, name, v, result.Params[name])
			}
		}
	}
}
```

Why the tolerances: the shift gradient comes out at roughly 1e-16 (floating-point cancellation, not a real slope), so VQE's step moves a parameter by about 3e-17 and the energy by less; `1e-12` absorbs that without hiding a genuine move. The finite-difference floor of `0.1` is safe because the true |dE/dtheta| at the four base points ranges from 0.156 to 0.794 (verified when this plan was written); the point `-0.3` used by `TestParameterShiftMatchesFiniteDifference` is deliberately omitted because its `b` slope is only 0.047.

- [ ] **Step 3: Run the test and confirm it passes on the first run**

Run:

```bash
go test ./algorithm -run 'TestParameterShiftIsBlindToRescaledFactory|TestParameterShiftMatchesFiniteDifference' -v
```

Expected: both PASS.

There is no red phase for this task, and that is intended: no behavior changes, so there is nothing for a test to fail against first. The test is a characterization test whose job is to make the doc comment's three claims executable (zero shift gradient, non-zero true slope, VQE Converged after one no-op iteration) so a future change to the shift rule or the driver flags the comment for update. If it fails, do not adjust thresholds or assertions; stop and report which assertion failed, because it means the documented claim is not what the code does.

- [ ] **Step 4: Replace the doc comment on `parameterShiftGradient`**

In `algorithm/vqe.go`, replace exactly lines 28-37 (the comment block that currently begins `// parameterShiftGradient computes dE/dtheta per parameter via the exact` and ends `// number of energy evaluations consumed.`) with the following, verbatim. The `func parameterShiftGradient(...)` line on line 38 and everything below it stay untouched.

```go
// parameterShiftGradient computes dE/dtheta per parameter via the exact
// parameter-shift rule: dE/dtheta = (E(theta+pi/2) - E(theta-pi/2)) / 2.
// The rule is exact only when, with every other parameter held fixed,
// E(theta) = a + b*cos(theta) + c*sin(theta). That shape needs two things
// from the template, and this function checks neither.
//
// First, each parameter must drive exactly one template step. Shifting a
// name that feeds several gates moves all of them at once, adding higher
// harmonics (cos(k*theta)) to which the +/- pi/2 difference is blind. VQE
// enforces this up front through Template.ParamStepCounts, so such a
// template never reaches here.
//
// Second, the bound value must enter its gate as exp(-i*theta*P/2) with P a
// Pauli (eigenvalues +/-1): the convention of gates.NewRx/NewRy/NewRz and
// hence of the parameterized.Rx/Ry/Rz/Phase factories (Phase is Rz up to a
// global phase, which expectation values ignore). A parameterized.Factory
// is an arbitrary func(value) Gate, though, so a caller can break this by
// rescaling inside the factory. A factory returning NewRz(2*value) makes E
// periodic in the bound value with period pi, so E(theta+pi/2) equals
// E(theta-pi/2) at every theta and this function returns an identically
// zero gradient while the true dE/dtheta is not zero. Neither this
// function nor VQE can detect that: the template exposes only the opaque
// factory, and a vanishing shift difference is indistinguishable from a
// genuine stationary point, so VQE takes a zero step and reports Converged
// at its starting parameters. Factories must bind the value unscaled and
// leave any textbook rescaling to how callers read the parameter, as
// QAOATemplate does (its bound value is the Rz/Rx angle; textbook
// gamma/beta is half of it). TestParameterShiftIsBlindToRescaledFactory
// pins this failure mode.
//
// Returns the gradient keyed by name plus the number of energy evaluations
// consumed.
```

This text is already gofmt-clean (prose paragraphs only, no doc-comment lists, so gofmt has nothing to reflow). Do not reword it: the phrases "exp(-i*theta*P/2)", "period pi", "identically zero", and the `QAOATemplate` cross-reference are the requirement.

- [ ] **Step 5: Tick backlog item 13**

In `docs/enhancement-backlog-2026-08-27.md`, replace the three lines of item 13 (currently lines 232-234, beginning `- [ ] 13.`) with:

```markdown
- [x] 13. **`parameterShiftGradient` precondition doc** — `vqe.go` should
  state that the bound value must enter the gate as exp(-i*theta*P/2); the
  driver cannot detect a rescaling factory. First occurrence: QAOA template.
  > **Done (2026-09-07)**: the doc comment on `parameterShiftGradient` now
  > states both preconditions in the codebase's terms — one gate per
  > parameter (enforced by VQE via `ParamStepCounts`) and the bound value
  > entering its gate as exp(-i*theta*P/2), the `gates.NewRx/NewRy/NewRz`
  > convention the `parameterized` factories inherit — and explains why
  > the second cannot be checked: a `parameterized.Factory` is an opaque
  > `func(value) Gate`, and a rescaled factory's zero shift difference
  > looks exactly like a stationary point. `QAOATemplate` is cited as the
  > precedent (fixed in cf09732). `TestParameterShiftIsBlindToRescaledFactory`
  > pins the failure mode: zero parameter-shift gradient against a
  > non-zero finite difference, and VQE reporting Converged after a
  > one-iteration no-op.
```

The em dashes and blockquote form match every other closed item in that file; keep them.

- [ ] **Step 6: Verify formatting, the comment-only diff, and the full suite**

Run:

```bash
gofmt -l algorithm/vqe.go algorithm/vqe_test.go
go vet ./algorithm
go build ./...
go test ./...
git diff algorithm/vqe.go | grep '^[+-]' | grep -v '^+++' | grep -v '^---' | grep -v '^[+-]//'
```

Expected: `gofmt -l` prints nothing; vet, build, and the full test run are clean; the last command prints nothing, proving the `vqe.go` diff is comment lines only. If it prints anything, a code line was touched by mistake; restore it.

- [ ] **Step 7: Commit**

```bash
git add algorithm/vqe.go algorithm/vqe_test.go docs/enhancement-backlog-2026-08-27.md
git commit -m "docs(algorithm): state the exp(-i*theta*P/2) precondition on parameterShiftGradient

The parameter-shift rule needs the bound value to enter its gate as
exp(-i*theta*P/2); a factory that rescales (NewRz(2*value)) gives the
parameter period pi, so the +/- pi/2 shift difference vanishes and VQE
no-ops while reporting Converged. The driver cannot detect this because
a parameterized.Factory is opaque, so the contract is documented where
the rule lives, with QAOATemplate (cf09732) as the precedent.

TestParameterShiftIsBlindToRescaledFactory pins the failure mode. Closes
backlog item 13."
```

---

## Self-Review

- **Requirement coverage.** "`vqe.go` should state that the bound value must enter the gate as exp(-i*theta*P/2)": Step 4, second paragraph of the comment. "The driver cannot detect a rescaling factory": Step 4, the sentence beginning "Neither this function nor VQE can detect that", with the reason (opaque `Factory`, zero difference looks like a stationary point). "First occurrence: QAOA template": Step 4 cites `QAOATemplate` and its bound-value convention; the backlog note cites cf09732.
- **Test decision.** A test is included because the doc comment makes three concrete, checkable claims and no existing test states them: `TestParameterShiftMatchesFiniteDifference` covers only convention-following factories, and `TestVQEOptimizesQAOATriangle` catches a rescale in `QAOATemplate` end to end but says nothing about the general failure mode. The test passes on first run by design (Step 3 explains why there is no red phase).
- **Placeholder scan.** No TBD/TODO; every code step carries its full text; every command names its expected result.
- **Type consistency.** `parameterShiftGradient(h, tmpl, params, tmpl.ParamNames())` returns `(map[string]float64, int, error)`, matching `algorithm/vqe.go:38`. `VQEOptions{InitialParams, MaxIterations}` and `VQEResult{Energy, Params, Iterations, Converged}` match lines 65-85. The helper and test names in Step 2 match the names cited in Step 4 and Step 5.
