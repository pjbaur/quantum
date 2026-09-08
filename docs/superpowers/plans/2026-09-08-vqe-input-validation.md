# VQE Input Validation (Backlog Item 14) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close backlog item 14 by making `algorithm.VQE` reject out-of-domain options, non-finite initial parameters, Hamiltonian/template structural mismatches, and non-finite Hamiltonian coefficients with `InvalidVQEInputError` before the first evaluation, and by wrapping the errors only an evaluation can reveal in the same type without losing the underlying cause.

**Architecture:** Two unexported validators in `algorithm/vqe.go` (`validateVQEOptions`, `validateVQEStructure`) run right after the nil checks; `InvalidVQEInputError` gains an `Err` field and `Unwrap` so every error from `evaluate` or `parameterShiftGradient` inside `VQE` is wrapped through `wrapEvaluationError` and still reachable with `errors.As`; a finiteness guard on the gradient stops float64 overflow from ever blaming a parameter. `evaluate` and `parameterShiftGradient` themselves are untouched.

**Tech Stack:** Go standard library only. No new dependencies.

**Spec:** `docs/superpowers/specs/2026-09-08-vqe-input-validation-design.md`

## Classification

**Standard.** Behavior change within the existing driver structure, one package. Two tasks, each one test cycle.

## Global Constraints

- Never edit a failing test to make code pass. The moved red tests are the item's acceptance tests; their assertions are copied verbatim. If one fails after implementation, the implementation is wrong. Report it.
- Error style in `algorithm/`: typed `Invalid<Input>Error { Reason string }` following `InvalidVQEInputError`; `Reason` is the complete message; `Error()` prefixes `"invalid VQE input: "`. The only addition is the `Err error` field plus `Unwrap` specified in Task 2.
- No changes outside `algorithm/`. The design lists every caller (`internal/examples/qaoa.go`, `cmd/quantum/main.go`) and shows none needs to change.
- Doc-comment voice: explain the why, cite conventions (`parameterized.Bind`, `quantum.Expectation`, `circuit.AddGate`, `fmt.Errorf` `%w`), no filler. Every comment in this plan is verbatim; do not reword it.
- Items 15, 16, 17 are not fixed here. Their tests stay in `algorithm/backlog_red_test.go` under `//go:build redtests` and must still fail after both tasks.
- No new dependencies. Go standard library only.
- Every task ends with `gofmt -l algorithm` printing nothing, `go vet ./...`, `staticcheck ./...`, `go build ./...`, `go test ./...` green, `go vet -tags redtests ./algorithm` compiling, then a commit. Commit messages are conventional style; append the attribution trailers your session specifies.
- Do not edit `.superpowers/backlog/` or `docs/enhancement-backlog-2026-08-27.md`; the run lead ticks the item after the QA gate.

---

### Task 1: Option and initial-parameter domains

**Sizing estimate:** about 700 lines of existing code to read (`algorithm/vqe.go` 240, `algorithm/vqe_test.go` 170, `algorithm/backlog_red_test.go` 207, `parameterized/parameterized.go` lines 1-160); 1 non-test file modified (`algorithm/vqe.go`, +44/-5) plus 2 test files (`vqe_test.go` +63, `backlog_red_test.go` -32); net diff about +70 lines; one test cycle. Under a quarter of a context window.

**Files:**
- Modify: `algorithm/vqe.go:86-98` (the `VQEOptions` doc and field comments), `algorithm/vqe.go:116` (insert two helpers after `Error()`), `algorithm/vqe.go:151-156` (call the validator after the nil checks), `algorithm/vqe.go:182-187` (initial-parameter loop)
- Test: `algorithm/vqe_test.go` (import `errors`; append two tests)
- Test: `algorithm/backlog_red_test.go` (delete `TestRedVQEOptionValidation` and its comment; nothing else)

**Interfaces:**
- Consumes (existing, unchanged):
  - `func VQE(h *Hamiltonian, t *parameterized.Template, opts VQEOptions) (*VQEResult, error)` in `algorithm/vqe.go:150`
  - `type InvalidVQEInputError struct { Reason string }` in `algorithm/vqe.go:110`
  - `func H2Hamiltonian() *Hamiltonian`, `func H2Ansatz() *parameterized.Template` in `algorithm/h2.go`
  - `parameterized.Params` (`map[string]float64`)
- Produces (Task 2 relies on these exact names):
  - `func isFinite(v float64) bool`
  - `func validateVQEOptions(opts VQEOptions) error` (returns `*InvalidVQEInputError` or nil; does not look at `InitialParams`)
  - `VQE` calls `validateVQEOptions(opts)` immediately after the nil checks; Task 2 inserts its structural validator directly below that call.

- [ ] **Step 1: Read the context**

1. `algorithm/vqe.go` in full. Note the defaults block at lines 158-169 (`step == 0`, `maxIter == 0`, `tol == 0` select defaults): zero stays the sentinel; the validator rejects only non-zero out-of-domain values. Note the floor `if step < 1e-6 { step = 1e-6 }` at line 218, which is what turned a negative step positive after one climb.
2. `algorithm/backlog_red_test.go` lines 83-115: the test you are moving. Its assertions are the acceptance criteria.
3. `parameterized/parameterized.go` lines 118-131: `Bind` rejects non-finite values with `InvalidParameterValueError` naming the parameter. That is the error the item says must no longer surface for a non-finite `StepSize`.
4. `algorithm/vqe_driver_test.go`: the external-package driver tests. `TestVQERespectsMaxIterations` uses `Tolerance: 1e-15`; `TestVQEInvalidInputs` checks the undeclared-name path. Both must keep passing.

- [ ] **Step 2: Move the red test and add the Reason test**

In `algorithm/backlog_red_test.go`, delete exactly this block (lines 83-116: the comment, the function, and the blank line after it). The `"math"` import stays; the two remaining item-14 tests still use it until Task 2.

```go
// Backlog item 14: VQE input validation and error taxonomy.
//
// VQE(out-of-domain options) × missing option validation → the run
// proceeds silently (a negative StepSize ascends once, then the floor clamp
// turns it into +1e-6; a negative MaxIterations returns at once) or a
// parameterized error leaks in place of InvalidVQEInputError.
func TestRedVQEOptionValidation(t *testing.T) {
	h := H2Hamiltonian()
	tmpl := H2Ansatz()

	cases := []struct {
		name string
		opts VQEOptions
	}{
		{"negative StepSize", VQEOptions{StepSize: -0.3, InitialParams: parameterized.Params{"theta": 0.1}}},
		{"NaN StepSize", VQEOptions{StepSize: math.NaN(), InitialParams: parameterized.Params{"theta": 0.1}}},
		{"Inf StepSize", VQEOptions{StepSize: math.Inf(1), InitialParams: parameterized.Params{"theta": 0.1}}},
		{"negative MaxIterations", VQEOptions{MaxIterations: -1}},
		{"negative Tolerance", VQEOptions{Tolerance: -1, MaxIterations: 5}},
		{"NaN Tolerance", VQEOptions{Tolerance: math.NaN(), MaxIterations: 5}},
		{"NaN InitialParams", VQEOptions{InitialParams: parameterized.Params{"theta": math.NaN()}}},
		{"Inf InitialParams", VQEOptions{InitialParams: parameterized.Params{"theta": math.Inf(-1)}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var e *InvalidVQEInputError
			res, err := VQE(h, tmpl, c.opts)
			if !errors.As(err, &e) {
				t.Errorf("VQE(%+v): err = %v, result = %+v; want InvalidVQEInputError", c.opts, err, res)
			}
		})
	}
}

```

In `algorithm/vqe_test.go`, change the import block (lines 3-10) to:

```go
import (
	"errors"
	"math"
	"testing"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/parameterized"
	"github.com/pjbaur/quantum/quantum"
)
```

Then append to the end of `algorithm/vqe_test.go`. The first function is the moved test: same name minus the `Red` prefix, same cases, same assertion; only the comment is rewritten to describe the contract instead of the defect.

```go

// TestVQEOptionValidation pins the option domains (backlog item 14).
// Before validation existed, a negative StepSize ascended once and was
// then clamped to +1e-6 by the floor, a negative MaxIterations returned
// unconverged after zero iterations, a negative or NaN Tolerance ran to
// MaxIterations, and a non-finite StepSize or InitialParams value
// surfaced as parameterized.InvalidParameterValueError from the first
// Bind. All are the caller's option literal, so all are
// InvalidVQEInputError.
func TestVQEOptionValidation(t *testing.T) {
	h := H2Hamiltonian()
	tmpl := H2Ansatz()

	cases := []struct {
		name string
		opts VQEOptions
	}{
		{"negative StepSize", VQEOptions{StepSize: -0.3, InitialParams: parameterized.Params{"theta": 0.1}}},
		{"NaN StepSize", VQEOptions{StepSize: math.NaN(), InitialParams: parameterized.Params{"theta": 0.1}}},
		{"Inf StepSize", VQEOptions{StepSize: math.Inf(1), InitialParams: parameterized.Params{"theta": 0.1}}},
		{"negative MaxIterations", VQEOptions{MaxIterations: -1}},
		{"negative Tolerance", VQEOptions{Tolerance: -1, MaxIterations: 5}},
		{"NaN Tolerance", VQEOptions{Tolerance: math.NaN(), MaxIterations: 5}},
		{"NaN InitialParams", VQEOptions{InitialParams: parameterized.Params{"theta": math.NaN()}}},
		{"Inf InitialParams", VQEOptions{InitialParams: parameterized.Params{"theta": math.Inf(-1)}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var e *InvalidVQEInputError
			res, err := VQE(h, tmpl, c.opts)
			if !errors.As(err, &e) {
				t.Errorf("VQE(%+v): err = %v, result = %+v; want InvalidVQEInputError", c.opts, err, res)
			}
		})
	}
}

// TestVQEOptionErrorNamesTheField pins the Reason wording: each message
// names the offending field and its value, so a caller reading the error
// can go straight to the option literal, and says that zero would have
// selected the default.
func TestVQEOptionErrorNamesTheField(t *testing.T) {
	cases := []struct {
		name string
		opts VQEOptions
		want string
	}{
		{"StepSize", VQEOptions{StepSize: -0.3}, "StepSize must be finite and positive (zero selects the default 0.3), got -0.3"},
		{"MaxIterations", VQEOptions{MaxIterations: -1}, "MaxIterations must not be negative (zero selects the default 200), got -1"},
		{"Tolerance", VQEOptions{Tolerance: math.NaN()}, "Tolerance must be finite and positive (zero selects the default 1e-10), got NaN"},
		{"InitialParams", VQEOptions{InitialParams: parameterized.Params{"theta": math.Inf(-1)}}, `initial parameter "theta" has non-finite value -Inf`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var e *InvalidVQEInputError
			_, err := VQE(H2Hamiltonian(), H2Ansatz(), c.opts)
			if !errors.As(err, &e) {
				t.Fatalf("VQE(%+v): err = %v (%T); want InvalidVQEInputError", c.opts, err, err)
			}
			if e.Reason != c.want {
				t.Fatalf("Reason = %q, want %q", e.Reason, c.want)
			}
		})
	}
}
```

- [ ] **Step 3: Run the tests to verify they fail**

Run:

```bash
go test ./algorithm -run '^TestVQEOption' -v 2>&1 | grep -E '^\s*(--- FAIL|vqe_test)|^(ok|FAIL)'
go vet -tags redtests ./algorithm
```

Expected: all eight `TestVQEOptionValidation` subtests fail. The `negative StepSize`, `negative MaxIterations`, `negative Tolerance`, and `NaN Tolerance` cases report `err = <nil>` with a result; the four NaN/Inf cases report `err = parameter "theta" has non-finite value ...`. All four `TestVQEOptionErrorNamesTheField` subtests fail with `want InvalidVQEInputError`. The `go vet` under the `redtests` tag compiles cleanly (the tagged file still uses `math`).

- [ ] **Step 4: Implement the option validator**

Three edits in `algorithm/vqe.go`.

(a) Replace lines 86-98, the `VQEOptions` type with its doc comment, with:

```go
// VQEOptions configures the VQE loop. Zero values select defaults. Values
// outside a field's domain are rejected with InvalidVQEInputError rather
// than run with: a negative StepSize climbs, a negative or NaN Tolerance
// can never be met, and a non-finite value reaches every parameter
// through the first update.
type VQEOptions struct {
	// InitialParams names starting angles. Missing declared parameters
	// default to 0. Unknown names and non-finite values are rejected.
	InitialParams parameterized.Params
	// StepSize is the initial gradient-descent step (default 0.3). Must
	// be finite and positive; zero selects the default.
	StepSize float64
	// MaxIterations caps the loop (default 200). Must not be negative;
	// zero selects the default.
	MaxIterations int
	// Tolerance is the convergence threshold on |delta E| between
	// consecutive iterations (default 1e-10). Must be finite and
	// positive; zero selects the default.
	Tolerance float64
}
```

(b) Directly after the `Error()` method of `InvalidVQEInputError` (after the closing brace on line 116, before the `// VQE minimizes` doc comment), insert:

```go

// isFinite reports whether v is neither NaN nor infinite.
func isFinite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

// validateVQEOptions checks the scalar option domains. Zero is the
// "use the default" sentinel for every field, so the checks reject only
// values that are neither zero nor usable: a negative StepSize ascends
// (and the loop's 1e-6 floor would then silently clamp it positive), a
// negative MaxIterations is meaningless, a negative or NaN Tolerance can
// never be met, and a non-finite StepSize or Tolerance turns the first
// update or the convergence test into NaN arithmetic. InitialParams needs
// the template's declared names and is checked in VQE itself.
func validateVQEOptions(opts VQEOptions) error {
	if !isFinite(opts.StepSize) || opts.StepSize < 0 {
		return &InvalidVQEInputError{Reason: fmt.Sprintf("StepSize must be finite and positive (zero selects the default 0.3), got %v", opts.StepSize)}
	}
	if opts.MaxIterations < 0 {
		return &InvalidVQEInputError{Reason: fmt.Sprintf("MaxIterations must not be negative (zero selects the default 200), got %d", opts.MaxIterations)}
	}
	if !isFinite(opts.Tolerance) || opts.Tolerance < 0 {
		return &InvalidVQEInputError{Reason: fmt.Sprintf("Tolerance must be finite and positive (zero selects the default 1e-10), got %v", opts.Tolerance)}
	}
	return nil
}
```

(c) Inside `VQE`, two edits. First, replace

```go
	if t == nil {
		return nil, &InvalidVQEInputError{Reason: "template must not be nil"}
	}

	step := opts.StepSize
```

with

```go
	if t == nil {
		return nil, &InvalidVQEInputError{Reason: "template must not be nil"}
	}
	if err := validateVQEOptions(opts); err != nil {
		return nil, err
	}

	step := opts.StepSize
```

Second, in the `for name, value := range opts.InitialParams` loop, replace

```go
		if _, ok := params[name]; !ok {
			return nil, &InvalidVQEInputError{Reason: fmt.Sprintf("initial parameter %q is not declared in the template", name)}
		}
		params[name] = value
```

with

```go
		if _, ok := params[name]; !ok {
			return nil, &InvalidVQEInputError{Reason: fmt.Sprintf("initial parameter %q is not declared in the template", name)}
		}
		if !isFinite(value) {
			return nil, &InvalidVQEInputError{Reason: fmt.Sprintf("initial parameter %q has non-finite value %v", name, value)}
		}
		params[name] = value
```

`math` and `fmt` are already imported; no import change.

- [ ] **Step 5: Run the tests to verify they pass**

Run:

```bash
go test ./algorithm -run '^TestVQEOption' -v 2>&1 | grep -E '^(--- |ok|FAIL)'
```

Expected: `--- PASS: TestVQEOptionValidation` and `--- PASS: TestVQEOptionErrorNamesTheField`, then `ok`.

- [ ] **Step 6: Full verification**

Run:

```bash
gofmt -l algorithm
go vet ./...
go vet -tags redtests ./algorithm
staticcheck ./...
go build ./...
go test ./...
go test -tags redtests ./algorithm -run '^TestRed' 2>&1 | grep -E '^(--- FAIL|ok|FAIL)'
```

Expected: `gofmt -l` prints nothing; vet (both), staticcheck, build, and the full suite are clean; the last command lists exactly five failing red tests (`TestRedVQEEmptyNameParameterEscapesStepCountCheck`, `TestRedParameterShiftMissingParamIsRejected`, `TestRedVQEStructuralMismatchIsInvalidInput`, `TestRedVQENonFiniteHamiltonianIsNotBlamedOnParams`, `TestRedParameterShiftCountsEvaluationsBeforeFailure`), the two item-14 ones being Task 2's job.

- [ ] **Step 7: Commit**

```bash
git add algorithm/vqe.go algorithm/vqe_test.go algorithm/backlog_red_test.go
git commit -m "fix(algorithm): validate VQE option domains and initial parameter values

A negative StepSize climbed once and was then clamped positive by the
1e-6 floor, a negative MaxIterations returned unconverged after zero
iterations, a negative or NaN Tolerance could never be met, and a
non-finite StepSize or InitialParams value surfaced from the first Bind
as parameterized.InvalidParameterValueError. All are the caller's option
literal, so validateVQEOptions and the initial-parameter loop now reject
them with InvalidVQEInputError before the first evaluation. Zero keeps
selecting each default.

TestRedVQEOptionValidation moves into the regular suite as
TestVQEOptionValidation. Backlog item 14, part 1 of 2."
```

---

### Task 2: Structural and non-finite Hamiltonian input; wrapped evaluation errors

**Sizing estimate:** about 1,000 lines of existing code to read (`algorithm/vqe.go` 280 after Task 1, `algorithm/vqe_test.go` 235 after Task 1, `algorithm/backlog_red_test.go` 175 after Task 1, `algorithm/hamiltonian.go` 52, `quantum/errortypes.go` 150, `quantum/expectation.go` lines 60-70 and 182-196, `circuit/circuit.go` lines 44-75); 2 non-test files modified (`algorithm/vqe.go` +87/-10, `algorithm/hamiltonian.go` +3/-1) plus 2 test files (`vqe_test.go` +140, `backlog_red_test.go` -59); net diff about +160 lines; one test cycle. Under half a context window.

**Files:**
- Modify: `algorithm/vqe.go` (the `InvalidVQEInputError` block; insert two helpers after `validateVQEOptions`; the `VQE` doc comment; four edits inside `VQE`)
- Modify: `algorithm/hamiltonian.go:27-30` (the `AddTerm` doc comment)
- Test: `algorithm/vqe_test.go` (import `strings`; append five tests)
- Test: `algorithm/backlog_red_test.go` (delete the two remaining item-14 tests and the now-unused `"math"` import)

**Interfaces:**
- Consumes (from Task 1, exact names): `func isFinite(v float64) bool`; `func validateVQEOptions(opts VQEOptions) error`; the call `if err := validateVQEOptions(opts); err != nil { return nil, err }` in `VQE`, below which this task inserts its own call.
- Consumes (existing, unchanged): `func evaluate(h *Hamiltonian, t *parameterized.Template, params parameterized.Params) (float64, error)`; `func parameterShiftGradient(h *Hamiltonian, t *parameterized.Template, params parameterized.Params, names []string) (map[string]float64, int, error)`; `Hamiltonian.terms []HamiltonianTerm` (unexported, same package) with `Coeff float64` and `Axes []quantum.PauliAxis`; `(*parameterized.Template).ParamStepCounts() map[string]int`, `.ParamNames() []string`, `.NumQubits() int`; `quantum.InvalidGateApplicationError`, `quantum.InvalidPauliAxisError`, `quantum.PauliAxis`.
- Produces (exported surface change and the extension point for item 15):
  - `type InvalidVQEInputError struct { Reason string; Err error }` with `func (e *InvalidVQEInputError) Unwrap() error`
  - `func validateVQEStructure(h *Hamiltonian, t *parameterized.Template) error` (item 15 adds its empty-name check to this function's template loop)
  - `func wrapEvaluationError(where string, err error) error`

- [ ] **Step 1: Read the context**

1. `algorithm/vqe.go` as left by Task 1, in full. The inline one-gate-per-parameter loop (`stepCounts := t.ParamStepCounts()` ... inside `VQE`) moves into the new validator unchanged, message included: `TestVQERejectsMultiGateParameter` in `vqe_driver_test.go` checks that message names the parameter.
2. `algorithm/hamiltonian.go` in full: `terms` is unexported but visible here; `Energy` skips `Expectation` for identity terms (no axes), which is why the length check exempts them.
3. `quantum/expectation.go` lines 60-70 (the `Expectation` error list) and 182-196 (`validatePauliAxes`: length mismatch is `IncompatibleQubitCountError`, axis `> PauliZ` is `InvalidPauliAxisError`). The design leaves axis validity to this function.
4. `circuit/circuit.go` lines 44-75: `AddGate` returns `InvalidGateApplicationError` when the factory's gate width differs from the target count. This is what `Bind` returns for the wide-factory case and what the wrap must preserve.
5. `quantum/errortypes.go`: none of the module's typed errors has an `Unwrap`; `fmt.Errorf` with `%w` is used in `gates/matrix.go` and `circuit/parallel.go`. The new `Unwrap` cites that convention.
6. `algorithm/backlog_red_test.go` after Task 1: the two item-14 tests you are moving, and the items 15/16/17 tests that must remain.

- [ ] **Step 2: Move the two red tests and add the three new ones**

In `algorithm/backlog_red_test.go`, delete the block that begins with the comment `// Backlog item 14: VQE input validation and error taxonomy.` followed by `// VQE(Hamiltonian or factory structurally incompatible with the` and ends with the closing brace of `TestRedVQENonFiniteHamiltonianIsNotBlamedOnParams` and the blank line after it, so that the `// Backlog item 17:` comment directly follows the closing brace of `TestRedParameterShiftMissingParamIsRejected` plus one blank line. The deleted text is exactly:

```go
// Backlog item 14: VQE input validation and error taxonomy.
//
// VQE(Hamiltonian or factory structurally incompatible with the
// template register) × late detection → the error surfaces from
// quantum/circuit inside the loop after evaluations were spent, not as
// InvalidVQEInputError up front.
func TestRedVQEStructuralMismatchIsInvalidInput(t *testing.T) {
	twoQubitFactory := func(v float64) quantum.Gate { return gates.NewCNOT() }
	wideFactoryTemplate := parameterized.NewTemplate(2)
	if err := wideFactoryTemplate.AddParamGate("theta", twoQubitFactory, 0); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		h    *Hamiltonian
		tmpl *parameterized.Template
	}{
		{"three-axis term on two-qubit template", NewHamiltonian().AddTerm(1, quantum.PauliZ, quantum.PauliZ, quantum.PauliZ), H2Ansatz()},
		{"one-axis term on two-qubit template", NewHamiltonian().AddTerm(1, quantum.PauliZ), H2Ansatz()},
		{"factory gate wider than its target list", H2Hamiltonian(), wideFactoryTemplate},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var e *InvalidVQEInputError
			res, err := VQE(c.h, c.tmpl, VQEOptions{})
			if !errors.As(err, &e) {
				t.Errorf("VQE: err = %v (%T), result = %+v; want InvalidVQEInputError", err, err, res)
			}
		})
	}
}

// Backlog item 14: VQE input validation and error taxonomy.
//
// VQE(Hamiltonian with a non-finite coefficient) × non-finite energy
// propagation → the failure is attributed to a template parameter
// (parameterized.InvalidParameterValueError) although every parameter is
// finite; the Hamiltonian doc delegates non-finite detection to the caller.
func TestRedVQENonFiniteHamiltonianIsNotBlamedOnParams(t *testing.T) {
	cases := []struct {
		name string
		h    *Hamiltonian
	}{
		{"NaN Pauli term", NewHamiltonian().AddTerm(math.NaN(), quantum.PauliZ, quantum.PauliI)},
		{"Inf identity term", NewHamiltonian().AddTerm(math.Inf(1))},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res, err := VQE(c.h, H2Ansatz(), VQEOptions{InitialParams: parameterized.Params{"theta": 0.1}})
			if err == nil {
				t.Fatalf("VQE returned %+v with no error for a non-finite Hamiltonian", res)
			}
			var pe *parameterized.InvalidParameterValueError
			if errors.As(err, &pe) {
				t.Fatalf("VQE blamed parameter %q (value %v) for a non-finite Hamiltonian coefficient: %v", pe.Name, pe.Value, err)
			}
		})
	}
}

```

Then, in the same file, change the import block to drop `"math"` (nothing left in the file uses it; leaving it would break `go vet -tags redtests`):

```go
import (
	"errors"
	"testing"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/parameterized"
	"github.com/pjbaur/quantum/quantum"
)
```

In `algorithm/vqe_test.go`, change the import block to:

```go
import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/parameterized"
	"github.com/pjbaur/quantum/quantum"
)
```

Then append to the end of `algorithm/vqe_test.go`. The first two functions are the moved tests: same names minus the `Red` prefix, same cases, same assertions; only the comments are rewritten to describe the contract.

```go

// TestVQEStructuralMismatchIsInvalidInput pins the structural contract
// (backlog item 14): a Hamiltonian whose Pauli strings do not match the
// template's register, or a factory returning a gate wider than its
// target list, is InvalidVQEInputError. Before the check existed the
// first two surfaced as quantum.IncompatibleQubitCountError and the third
// as quantum.InvalidGateApplicationError from inside the first
// evaluation.
func TestVQEStructuralMismatchIsInvalidInput(t *testing.T) {
	twoQubitFactory := func(v float64) quantum.Gate { return gates.NewCNOT() }
	wideFactoryTemplate := parameterized.NewTemplate(2)
	if err := wideFactoryTemplate.AddParamGate("theta", twoQubitFactory, 0); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		h    *Hamiltonian
		tmpl *parameterized.Template
	}{
		{"three-axis term on two-qubit template", NewHamiltonian().AddTerm(1, quantum.PauliZ, quantum.PauliZ, quantum.PauliZ), H2Ansatz()},
		{"one-axis term on two-qubit template", NewHamiltonian().AddTerm(1, quantum.PauliZ), H2Ansatz()},
		{"factory gate wider than its target list", H2Hamiltonian(), wideFactoryTemplate},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var e *InvalidVQEInputError
			res, err := VQE(c.h, c.tmpl, VQEOptions{})
			if !errors.As(err, &e) {
				t.Errorf("VQE: err = %v (%T), result = %+v; want InvalidVQEInputError", err, err, res)
			}
		})
	}
}

// TestVQENonFiniteHamiltonianIsNotBlamedOnParams pins the attribution
// contract (backlog item 14): a NaN or Inf coefficient is the
// Hamiltonian's fault. Before the check existed the non-finite energy
// flowed into a NaN gradient and a NaN step, and parameterized.Bind
// rejected the stepped parameter by name although the caller supplied it
// finite.
func TestVQENonFiniteHamiltonianIsNotBlamedOnParams(t *testing.T) {
	cases := []struct {
		name string
		h    *Hamiltonian
	}{
		{"NaN Pauli term", NewHamiltonian().AddTerm(math.NaN(), quantum.PauliZ, quantum.PauliI)},
		{"Inf identity term", NewHamiltonian().AddTerm(math.Inf(1))},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res, err := VQE(c.h, H2Ansatz(), VQEOptions{InitialParams: parameterized.Params{"theta": 0.1}})
			if err == nil {
				t.Fatalf("VQE returned %+v with no error for a non-finite Hamiltonian", res)
			}
			var pe *parameterized.InvalidParameterValueError
			if errors.As(err, &pe) {
				t.Fatalf("VQE blamed parameter %q (value %v) for a non-finite Hamiltonian coefficient: %v", pe.Name, pe.Value, err)
			}
		})
	}
}

// TestVQEStructuralErrorReasons pins the Reason wording for problems
// found before any evaluation: the message names the term by index and
// states the mismatch or the value, and nothing is wrapped because VQE
// detected the problem itself.
func TestVQEStructuralErrorReasons(t *testing.T) {
	cases := []struct {
		name string
		h    *Hamiltonian
		want string
	}{
		{"axes length", NewHamiltonian().AddTerm(1, quantum.PauliZ), "Hamiltonian term 0 has 1 Pauli axes but the template has 2 qubits"},
		{"NaN coefficient", NewHamiltonian().AddTerm(1, quantum.PauliZ, quantum.PauliI).AddTerm(math.NaN()), "Hamiltonian term 1 has non-finite coefficient NaN"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var e *InvalidVQEInputError
			res, err := VQE(c.h, H2Ansatz(), VQEOptions{})
			if !errors.As(err, &e) {
				t.Fatalf("err = %v (%T), result = %+v; want InvalidVQEInputError", err, err, res)
			}
			if e.Reason != c.want {
				t.Fatalf("Reason = %q, want %q", e.Reason, c.want)
			}
			if cause := errors.Unwrap(err); cause != nil {
				t.Fatalf("errors.Unwrap(err) = %v, want nil for a problem detected before any evaluation", cause)
			}
		})
	}
}

// TestVQEEvaluationErrorKeepsItsCause pins the wrapping contract for
// problems only running the template can reveal: the caller sees
// InvalidVQEInputError, and errors.As still reaches the error type of the
// package that detected the problem. The wide-factory case fails inside
// parameterized.Bind (circuit.AddGate rejects a 2-qubit gate on 1 target);
// the bad-axis case fails inside quantum.Expectation, whose axis domain
// VQE deliberately does not duplicate.
func TestVQEEvaluationErrorKeepsItsCause(t *testing.T) {
	wide := parameterized.NewTemplate(2)
	if err := wide.AddParamGate("theta", func(v float64) quantum.Gate { return gates.NewCNOT() }, 0); err != nil {
		t.Fatal(err)
	}
	var ve *InvalidVQEInputError
	var ge *quantum.InvalidGateApplicationError
	_, err := VQE(H2Hamiltonian(), wide, VQEOptions{})
	if !errors.As(err, &ve) || !errors.As(err, &ge) {
		t.Fatalf("VQE(wide factory): err = %v (%T); want InvalidVQEInputError wrapping InvalidGateApplicationError", err, err)
	}
	if !strings.Contains(ve.Reason, "initial energy evaluation") || !strings.Contains(ve.Reason, ge.Error()) {
		t.Fatalf("Reason = %q, want the evaluation phase and the cause %q", ve.Reason, ge.Error())
	}

	badAxis := NewHamiltonian().AddTerm(1, quantum.PauliAxis(7), quantum.PauliI)
	var ae *quantum.InvalidPauliAxisError
	_, err = VQE(badAxis, H2Ansatz(), VQEOptions{})
	if !errors.As(err, &ve) || !errors.As(err, &ae) {
		t.Fatalf("VQE(bad axis): err = %v (%T); want InvalidVQEInputError wrapping InvalidPauliAxisError", err, err)
	}
}

// TestVQEOverflowingHamiltonianIsNotBlamedOnParams covers the gap the
// up-front coefficient check leaves: coefficients that are each finite
// but whose sum passes float64. The energy is +Inf at the start and at
// one of the two shifted points, so the parameter-shift difference is
// -Inf and the descent step would carry theta to +Inf, where
// parameterized.Bind would reject it by name. The contract is that no
// error blames a parameter that was finite on entry, so VQE must stop at
// the non-finite gradient instead.
func TestVQEOverflowingHamiltonianIsNotBlamedOnParams(t *testing.T) {
	h := NewHamiltonian().AddTerm(math.MaxFloat64).AddTerm(math.MaxFloat64, quantum.PauliZ, quantum.PauliI)
	var ve *InvalidVQEInputError
	var pe *parameterized.InvalidParameterValueError
	res, err := VQE(h, H2Ansatz(), VQEOptions{InitialParams: parameterized.Params{"theta": 0.1}})
	if errors.As(err, &pe) {
		t.Fatalf("VQE blamed parameter %q (value %v) for an overflowing Hamiltonian: %v", pe.Name, pe.Value, err)
	}
	if !errors.As(err, &ve) {
		t.Fatalf("VQE: err = %v (%T), result = %+v; want InvalidVQEInputError", err, err, res)
	}
	if !strings.Contains(ve.Reason, `"theta"`) || !strings.Contains(ve.Reason, "non-finite") {
		t.Fatalf("Reason = %q, want it to name the non-finite gradient of %q", ve.Reason, "theta")
	}
}
```

Why the overflow numbers work (verified against the current code): with `X` on qubit 1 and `Ry(theta)` on qubit 0, `<Z0> = cos(theta)`, so at `theta = 0.1` the energy is `MaxFloat64 * (1 + 0.995) = +Inf`; at `theta + pi/2` it is `MaxFloat64 * 0.900`, finite; at `theta - pi/2` it is `+Inf` again. The gradient `(finite - Inf)/2 = -Inf` is what the guard must catch.

- [ ] **Step 3: Run the tests to verify they fail**

Run:

```bash
go test ./algorithm -run '^TestVQE(Structural|NonFinite|Evaluation|Overflowing)' -v 2>&1 | grep -E '^\s*(--- FAIL|vqe_test)|^(ok|FAIL)'
go vet -tags redtests ./algorithm
```

Expected: every subtest fails at runtime (the file compiles because `errors.Unwrap` needs no method). The three `TestVQEStructuralMismatchIsInvalidInput` cases report `*quantum.IncompatibleQubitCountError` (twice) and `*quantum.InvalidGateApplicationError`; both `TestVQENonFiniteHamiltonianIsNotBlamedOnParams` cases report `VQE blamed parameter "theta" (value NaN)`; `TestVQEStructuralErrorReasons` reports the quantum and parameterized types; `TestVQEEvaluationErrorKeepsItsCause` reports `want InvalidVQEInputError wrapping InvalidGateApplicationError`; `TestVQEOverflowingHamiltonianIsNotBlamedOnParams` reports `VQE blamed parameter "theta" (value +Inf)`. The tagged file still compiles under `go vet`.

- [ ] **Step 4: Implement the structural validator, the wrap, and the guard**

Edits in `algorithm/vqe.go`, in file order.

(a) Replace the `InvalidVQEInputError` block, currently

```go
// InvalidVQEInputError indicates a malformed VQE invocation.
type InvalidVQEInputError struct {
	Reason string
}

func (e *InvalidVQEInputError) Error() string {
	return "invalid VQE input: " + e.Reason
}
```

with

```go
// InvalidVQEInputError indicates a malformed VQE invocation: an option
// outside its domain, a template or Hamiltonian VQE cannot optimize, or
// an evaluation failure that traces back to one of them. Reason is the
// complete message. Err is set only when another package detected the
// problem during an evaluation (parameterized, circuit, quantum); Unwrap
// exposes it so errors.As still reaches that package's type, the same
// contract fmt.Errorf's %w gives callers elsewhere in this module
// (gates/matrix.go, circuit/parallel.go).
type InvalidVQEInputError struct {
	Reason string
	Err    error
}

func (e *InvalidVQEInputError) Error() string {
	return "invalid VQE input: " + e.Reason
}

// Unwrap returns the evaluation error this input error carries, or nil
// for problems VQE detected itself before the first evaluation.
func (e *InvalidVQEInputError) Unwrap() error { return e.Err }
```

(b) Directly after the closing brace of `validateVQEOptions` (before the `// VQE minimizes` doc comment), insert:

```go

// validateVQEStructure checks that the template and the Hamiltonian can
// be optimized against each other before any evaluation is paid for.
// The one-gate-per-parameter rule is the template-side precondition of
// the parameter-shift gradient (see parameterShiftGradient). A term whose
// Pauli string is not the register's length would fail inside
// quantum.Expectation on the first evaluation as
// IncompatibleQubitCountError; an identity term (no axes) is exempt
// because Energy adds its coefficient without consulting the state. A
// non-finite coefficient would flow through Energy into a non-finite
// gradient and then a non-finite parameter that parameterized.Bind
// rejects by name, blaming a value the caller supplied finite. All three
// are properties of the inputs, so all are InvalidVQEInputError here.
//
// Axis validity (each axis one of PauliI..PauliZ) is deliberately left to
// quantum.Expectation, which owns that domain and reports
// InvalidPauliAxisError; VQE wraps it at the first evaluation rather than
// duplicating the enum's range.
func validateVQEStructure(h *Hamiltonian, t *parameterized.Template) error {
	stepCounts := t.ParamStepCounts()
	for _, name := range t.ParamNames() {
		if count := stepCounts[name]; count > 1 {
			return &InvalidVQEInputError{Reason: fmt.Sprintf("parameter %q drives %d template steps; the parameter-shift gradient requires exactly one gate per parameter", name, count)}
		}
	}
	for i, term := range h.terms {
		if !isFinite(term.Coeff) {
			return &InvalidVQEInputError{Reason: fmt.Sprintf("Hamiltonian term %d has non-finite coefficient %v", i, term.Coeff)}
		}
		if len(term.Axes) != 0 && len(term.Axes) != t.NumQubits() {
			return &InvalidVQEInputError{Reason: fmt.Sprintf("Hamiltonian term %d has %d Pauli axes but the template has %d qubits", i, len(term.Axes), t.NumQubits())}
		}
	}
	return nil
}

// wrapEvaluationError converts an error from evaluate or
// parameterShiftGradient into InvalidVQEInputError. By the time an
// evaluation runs, VQE has checked every option, every initial parameter,
// the template's parameter structure, and the Hamiltonian's terms, and it
// hands each evaluation a complete, declared, finite parameter set. What
// can still fail is what only running the template reveals: a factory
// returning a gate of the wrong width or with a malformed matrix, or a
// Pauli axis outside the enum. Those are input properties, so the caller
// sees the VQE type, with the detecting package's error kept in Err.
func wrapEvaluationError(where string, err error) error {
	return &InvalidVQEInputError{Reason: where + " failed: " + err.Error(), Err: err}
}
```

(c) At the end of the `VQE` doc comment, replace

```go
// stationary point. See parameterShiftGradient.
func VQE(
```

with

```go
// stationary point. See parameterShiftGradient.
//
// Inputs are validated before the first evaluation: nil arguments, option
// values outside their domains (see VQEOptions), initial parameters that
// are undeclared or non-finite, a parameter driving several gates, a
// Hamiltonian term whose Pauli string does not match the template's qubit
// count, and a non-finite Hamiltonian coefficient are all rejected with
// InvalidVQEInputError. Problems only running the template can reveal,
// such as a factory returning a gate wider than its target list, surface
// from the first evaluation and are wrapped in the same type with the
// detecting package's error reachable through errors.As. Coefficients
// that are each finite but whose sum overflows float64 give a non-finite
// gradient, also reported as InvalidVQEInputError; no error names a
// parameter that was finite on entry.
func VQE(
```

(d) Inside `VQE`, four edits. First, call the validator: replace

```go
	if err := validateVQEOptions(opts); err != nil {
		return nil, err
	}
```

with

```go
	if err := validateVQEOptions(opts); err != nil {
		return nil, err
	}
	if err := validateVQEStructure(h, t); err != nil {
		return nil, err
	}
```

Second, delete the inline loop that the validator now owns: replace

```go
	names := t.ParamNames()
	stepCounts := t.ParamStepCounts()
	for _, name := range names {
		if count := stepCounts[name]; count > 1 {
			return nil, &InvalidVQEInputError{Reason: fmt.Sprintf("parameter %q drives %d template steps; the parameter-shift gradient requires exactly one gate per parameter", name, count)}
		}
	}
	params := parameterized.Params{}
```

with

```go
	names := t.ParamNames()
	params := parameterized.Params{}
```

Third, wrap the initial evaluation: replace

```go
	energy, err := evaluate(h, t, params)
	if err != nil {
		return nil, err
	}
	evals++
```

with

```go
	energy, err := evaluate(h, t, params)
	if err != nil {
		return nil, wrapEvaluationError("initial energy evaluation", err)
	}
	evals++
```

Fourth, wrap the gradient and step evaluations and add the guard: replace

```go
		grad, gradEvals, err := parameterShiftGradient(h, t, params, names)
		if err != nil {
			return nil, err
		}
		evals += gradEvals

		steps := parameterized.Params{}
		for _, name := range names {
			steps[name] = params[name] - step*grad[name]
		}
		newEnergy, err := evaluate(h, t, steps)
		if err != nil {
			return nil, err
		}
		evals++
```

with

```go
		grad, gradEvals, err := parameterShiftGradient(h, t, params, names)
		if err != nil {
			return nil, wrapEvaluationError(fmt.Sprintf("gradient evaluation at iteration %d", iter), err)
		}
		evals += gradEvals
		// Coefficients are finite (validateVQEStructure) and every Pauli
		// expectation is bounded by 1, so a non-finite gradient means the
		// energy sum overflowed float64. Stop here: the step would carry a
		// non-finite value into a parameter, and Bind would then blame that
		// parameter although the caller supplied it finite.
		for _, name := range names {
			if !isFinite(grad[name]) {
				return nil, &InvalidVQEInputError{Reason: fmt.Sprintf("gradient of parameter %q is non-finite (%v) at iteration %d: the Hamiltonian's energy overflows float64", name, grad[name], iter)}
			}
		}

		steps := parameterized.Params{}
		for _, name := range names {
			steps[name] = params[name] - step*grad[name]
		}
		newEnergy, err := evaluate(h, t, steps)
		if err != nil {
			return nil, wrapEvaluationError(fmt.Sprintf("step evaluation at iteration %d", iter), err)
		}
		evals++
```

(e) In `algorithm/hamiltonian.go`, replace the `AddTerm` doc comment (lines 27-30)

```go
// AddTerm appends coeff * (axes as a Pauli string) and returns h for
// chaining. Empty axes denote the identity, contributing coeff directly.
// Coefficients are not validated: NaN and Inf flow into Energy results,
// where they surface as non-finite energies the caller can detect.
```

with

```go
// AddTerm appends coeff * (axes as a Pauli string) and returns h for
// chaining. Empty axes denote the identity, contributing coeff directly.
// Coefficients are not validated: NaN and Inf flow into Energy results,
// where they surface as non-finite energies the caller can detect. VQE is
// such a caller and rejects them up front with InvalidVQEInputError, along
// with a Pauli string whose length is not the template's qubit count.
```

No import changes in either file.

- [ ] **Step 5: Run the tests to verify they pass**

Run:

```bash
go test ./algorithm -run '^TestVQE' -v 2>&1 | grep -E '^(--- |ok|FAIL)'
```

Expected: thirteen `--- PASS` lines for the `TestVQE*` tests in the internal and external test packages (the seven from `vqe_test.go` added by this plan, `TestVQEOptimizesQAOATriangle`, and the five in `vqe_driver_test.go`), then `ok`. `TestVQERejectsMultiGateParameter` passing confirms the moved one-gate-per-parameter message is intact.

- [ ] **Step 6: Full verification**

Run:

```bash
gofmt -l algorithm
go vet ./...
go vet -tags redtests ./algorithm
staticcheck ./...
go build ./...
go test ./...
go test -tags redtests ./algorithm -run '^TestRed' 2>&1 | grep -E '^(--- FAIL|ok|FAIL)'
go run ./cmd/quantum -demo qaoa | grep -A1 'VQE from'
```

Expected: `gofmt -l` prints nothing; vet (both), staticcheck, build, and the full suite are clean; the red run lists exactly three failures, `TestRedVQEEmptyNameParameterEscapesStepCountCheck`, `TestRedParameterShiftMissingParamIsRejected`, and `TestRedParameterShiftCountsEvaluationsBeforeFailure` (items 15, 16, 17, untouched by design); the demo prints `VQE from the symmetric start: energy = -1.0000, expected cut = 2.0000`. If a fourth red test fails or one of the three passes, stop and report: a later item's behavior changed.

- [ ] **Step 7: Commit**

```bash
git add algorithm/vqe.go algorithm/hamiltonian.go algorithm/vqe_test.go algorithm/backlog_red_test.go
git commit -m "fix(algorithm): reject structural and non-finite Hamiltonian input in VQE; wrap evaluation errors

A Hamiltonian term whose Pauli string did not match the template's qubit
count, or a factory returning a gate wider than its target list, failed
inside the first evaluation with a quantum error type; a NaN or Inf
coefficient flowed into a NaN gradient and step and was then blamed on a
parameter the caller supplied finite. validateVQEStructure now checks
coefficients and Pauli-string lengths up front (and owns the moved
one-gate-per-parameter rule), every error from an evaluation is wrapped
as InvalidVQEInputError with the cause kept behind Unwrap, and a
non-finite gradient stops the loop before it can reach a parameter.

TestRedVQEStructuralMismatchIsInvalidInput and
TestRedVQENonFiniteHamiltonianIsNotBlamedOnParams move into the regular
suite. Backlog item 14, part 2 of 2."
```

---

## Self-Review

- **Spec coverage.** Decision 1 up-front checks: options (Task 1 Step 4b), initial parameters (Task 1 Step 4c), one-gate-per-parameter, coefficients, and Pauli length (Task 2 Step 4b). Evaluation-time wrap with the three `where` phrases (Task 2 Step 4d). Gradient guard (Task 2 Step 4d, fourth edit). Decision 2 `Err`/`Unwrap` (Task 2 Step 4a), pinned by `TestVQEEvaluationErrorKeepsItsCause` and the nil-cause check in `TestVQEStructuralErrorReasons`. Decision 3 Reason strings: every row of the spec's table appears verbatim in a `fmt.Sprintf` in Step 4 of one task and, for the option, coefficient, and length rows, in a test's `want`. Rulings: axis validity left to `Expectation` (Task 2 comment and the bad-axis half of the cause test); `Tolerance: +Inf` rejected by `isFinite`; zero-qubit template covered by the wrap (no task, by design). Helper extension point for item 15 named in the Interfaces block. Callers: Task 2 Step 6 runs the QAOA demo. `AddTerm` doc pointer (Task 2 Step 4e).
- **Red test rulings.** All three moved tests keep their cases and assertions byte for byte; only names and comments change. No assertion was ruled against the contract.
- **Placeholder scan.** No TBD/TODO; every code step carries its full text; every command names its expected result.
- **Type consistency.** `isFinite(v float64) bool` and `validateVQEOptions(opts VQEOptions) error` are defined in Task 1 Step 4b and called in Task 2 Step 4b and 4d. `validateVQEStructure(h *Hamiltonian, t *parameterized.Template) error` and `wrapEvaluationError(where string, err error) error` are defined in Task 2 Step 4b and called in 4d. `InvalidVQEInputError{Reason, Err}` matches every literal in both tasks (Task 1 literals set only `Reason`, valid before and after the field is added). `errors.Unwrap(err)` in the reasons test returns nil for `Err == nil`. Test names cited in Step 5 and Step 6 match the functions in Step 2 of each task.
- **Prototype.** Every code and test block above was applied to a scratch copy of the repository at `560012d` in this exact sequence: Task 1 red then green, Task 2 red then green, with `gofmt -l`, `go vet` (plain and `-tags redtests`), `staticcheck`, `go build`, `go test ./...`, `go test -race ./algorithm`, and the QAOA demo all as stated in the Expected lines.
