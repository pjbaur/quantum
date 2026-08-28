# Backend Math Residual Dedupe Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deduplicate the math residuals shared by the dense (`state`) and sparse (`internal/sparsestate`) backends into `quantum` (amplitude-vector policy) and `internal/backendmath` (gate-application physics), per `docs/superpowers/specs/2026-08-28-backend-math-dedupe-design.md`.

**Architecture:** Behavior-preserving refactor. Policy helpers (tolerance const, normalization checks, vector validation) join `IsFiniteAmplitude`/`Probability` in the `quantum` package, which every package can import. The matmul kernel and two cold-path helpers join the existing physics helpers in `internal/backendmath`. Both backends call the shared helpers; container-specific loops, writes, and rollbacks stay per-backend.

**Tech Stack:** Go standard library only. No new dependencies.

## Global Constraints

- **No behavior change.** Every error type, field, and message byte-identical. Existing tests must pass unmodified — **never edit an existing test to make it pass**; if one fails, the refactor changed behavior: stop and fix the refactor.
- Length-mismatch message must remain exactly: `values slice length %d does not match state size %d`
- `algorithm.groundStateTolerance` and `qubit.go`'s 1e-6 check are untouched (different semantics, out of scope).
- Single-qubit gate loops in both backends are untouched (hottest path, deliberately not shared).
- Dense `applyMultiQubitGate` keeps caller-shaped buffers and loop structure (bounds-check-elimination sensitive, in-code "Re-benchmark before changing this" note). Benchmark gate: no dense multi-qubit regression beyond run-to-run noise; escape hatch in Task 5 if it regresses.
- Verification commands: `go build ./...`, `go test ./...`, `go test -race ./...`, `go vet ./...`, `gofmt -l .` (empty output = clean).
- Fuzz seed corpora: `go test ./state/ ./internal/sparsestate/ -run 'Fuzz' -v` (seeds run as tests; includes `FuzzDenseSparseGateEquivalence`).
- Remove imports that become unused (`fmt`, `math`, `math/rand` in the backend files); the compiler flags them.

---

### Task 1: Normalization policy helpers in `quantum`

**Files:**
- Create: `quantum/normalization.go`
- Test: `quantum/normalization_test.go`

**Interfaces:**
- Consumes: `quantum.IsFiniteAmplitude`, `quantum.Probability`, types `NormalizationError`, `NonFiniteAmplitudeError` (all exist).
- Produces (Task 2, 5, 6, 7 consume):
  - `const NormalizationTolerance = 1e-10`
  - `func IsNormalizedSum(sum float64) bool`
  - `func CheckNormalization(attemptedSum, currentSum float64) error`
  - `func ValidateAmplitudeVector(values []complex128, size int) (float64, error)`

- [ ] **Step 1: Write the failing tests**

Create `quantum/normalization_test.go`:

```go
package quantum

import (
	"math"
	"testing"
)

func TestNormalizationToleranceValue(t *testing.T) {
	if NormalizationTolerance != 1e-10 {
		t.Fatalf("NormalizationTolerance = %g, want 1e-10", NormalizationTolerance)
	}
}

func TestIsNormalizedSum(t *testing.T) {
	// Boundary cases avoid exactly-representable edges on purpose: 1±tolerance
	// does not round-trip through float64 addition, so the tests pin the
	// contract (a window around 1, NaN outside it) rather than one bit pattern.
	tests := []struct {
		name string
		sum  float64
		want bool
	}{
		{name: "exactly one", sum: 1.0, want: true},
		{name: "comfortably above", sum: 1 + 1e-11, want: true},
		{name: "comfortably below", sum: 1 - 1e-11, want: true},
		{name: "too high", sum: 1 + 1e-9, want: false},
		{name: "too low", sum: 1 - 1e-9, want: false},
		{name: "zero", sum: 0, want: false},
		{name: "NaN fails every comparison", sum: math.NaN(), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNormalizedSum(tt.sum); got != tt.want {
				t.Errorf("IsNormalizedSum(%g) = %v, want %v", tt.sum, got, tt.want)
			}
		})
	}
}

func TestCheckNormalization(t *testing.T) {
	if err := CheckNormalization(1.0, 1.0); err != nil {
		t.Fatalf("normalized sum returned error: %v", err)
	}

	err := CheckNormalization(1.5, 1.0)
	nerr, ok := err.(*NormalizationError)
	if !ok {
		t.Fatalf("error type %T, want *NormalizationError", err)
	}
	if nerr.AttemptedSum != 1.5 || nerr.CurrentSum != 1.0 {
		t.Errorf("fields = (%g, %g), want (1.5, 1.0)", nerr.AttemptedSum, nerr.CurrentSum)
	}
}

func TestValidateAmplitudeVector(t *testing.T) {
	t.Run("length mismatch message is byte-identical to the backends", func(t *testing.T) {
		_, err := ValidateAmplitudeVector([]complex128{1}, 4)
		want := "values slice length 1 does not match state size 4"
		if err == nil || err.Error() != want {
			t.Fatalf("error = %v, want %q", err, want)
		}
	})

	t.Run("first non-finite amplitude wins", func(t *testing.T) {
		values := []complex128{0, 1, complex(0, math.NaN()), complex(math.Inf(1), 0)}
		_, err := ValidateAmplitudeVector(values, 4)
		nf, ok := err.(*NonFiniteAmplitudeError)
		if !ok {
			t.Fatalf("error type %T, want *NonFiniteAmplitudeError", err)
		}
		if nf.BasisState != 2 {
			t.Errorf("BasisState = %d, want 2 (first offender)", nf.BasisState)
		}
	})

	t.Run("length check beats finite check", func(t *testing.T) {
		_, err := ValidateAmplitudeVector([]complex128{complex(math.NaN(), 0)}, 4)
		if err == nil || err.Error() != "values slice length 1 does not match state size 4" {
			t.Fatalf("error = %v, want the length message", err)
		}
	})

	t.Run("returns the probability sum", func(t *testing.T) {
		// 0.5² is exact in binary, so the sum is exactly 1.
		values := []complex128{0.5, 0.5, 0.5, 0.5}
		sum, err := ValidateAmplitudeVector(values, 4)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sum != 1.0 {
			t.Errorf("sum = %g, want 1", sum)
		}
	})
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./quantum/ -run 'TestNormalizationToleranceValue|TestIsNormalizedSum|TestCheckNormalization|TestValidateAmplitudeVector'`
Expected: compile failure — `NormalizationTolerance`, `IsNormalizedSum`, `CheckNormalization`, `ValidateAmplitudeVector` undefined.

- [ ] **Step 3: Write the implementation**

Create `quantum/normalization.go`:

```go
package quantum

import (
	"fmt"
	"math"
)

// NormalizationTolerance is the window around 1 within which a state's
// probability sum counts as normalized. Both state backends enforce it on
// every amplitude write, and Sample, Expectation, and Fidelity require it
// of the states they are given. One declaration replaces the five that
// used to cross-reference each other by comment.
const NormalizationTolerance = 1e-10

// IsNormalizedSum reports whether sum is the probability sum of a
// normalized state. A NaN sum is not: it compares false against the
// tolerance, which is what makes NaN-amplitude states fail the checks
// that matter.
func IsNormalizedSum(sum float64) bool {
	return math.Abs(sum-1.0) <= NormalizationTolerance
}

// CheckNormalization returns nil when attemptedSum is a normalized
// probability sum, or the NormalizationError a rejected amplitude write
// reports: the sum the write would have produced alongside the sum the
// state was rolled back to.
func CheckNormalization(attemptedSum, currentSum float64) error {
	if IsNormalizedSum(attemptedSum) {
		return nil
	}
	return &NormalizationError{
		AttemptedSum: attemptedSum,
		CurrentSum:   currentSum,
	}
}

// ValidateAmplitudeVector checks a candidate amplitude vector for a
// size-wide register and returns its probability sum. The length check
// runs first; the amplitudes are then scanned in order, and a non-finite
// amplitude is caught explicitly rather than by the sum — a NaN amplitude
// makes the sum NaN, and NaN fails every comparison. A vector that is
// wrong in more than one way therefore reports the same failure either
// backend reports today.
func ValidateAmplitudeVector(values []complex128, size int) (float64, error) {
	if len(values) != size {
		return 0, fmt.Errorf("values slice length %d does not match state size %d", len(values), size)
	}

	sum := 0.0
	for i, v := range values {
		if !IsFiniteAmplitude(v) {
			return 0, &NonFiniteAmplitudeError{BasisState: i, Value: v}
		}
		sum += Probability(v)
	}
	return sum, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./quantum/ -run 'TestNormalizationToleranceValue|TestIsNormalizedSum|TestCheckNormalization|TestValidateAmplitudeVector' -v`
Expected: PASS, all subtests.

- [ ] **Step 5: Run the package suite**

Run: `go test ./quantum/`
Expected: PASS (additive change, nothing else touched).

- [ ] **Step 6: Commit**

```bash
git add quantum/normalization.go quantum/normalization_test.go
git commit -m "refactor(quantum): add shared normalization policy helpers"
```

---

### Task 2: Adopt the shared tolerance inside `quantum`

**Files:**
- Modify: `quantum/sample.go:12-14` (delete const), `quantum/sample.go:66`
- Modify: `quantum/fidelity.go:51,54`
- Modify: `quantum/expectation.go:113`

**Interfaces:**
- Consumes: `NormalizationTolerance`, `IsNormalizedSum` from Task 1.
- Produces: nothing downstream (internal cleanup).

- [ ] **Step 1: Replace the four check sites and delete the private const**

In `quantum/sample.go`, delete lines 12–14 (the `sampleNormalizationTolerance` const and its comment) and change line 66:

```go
	if !IsNormalizedSum(total) {
```

In `quantum/fidelity.go`, change lines 51 and 54:

```go
	if !IsNormalizedSum(sumA) {
```

```go
	if !IsNormalizedSum(sumB) {
```

In `quantum/expectation.go`, change line 113:

```go
	if !IsNormalizedSum(sum) {
```

Do not touch the doc comments' prose ("the backends' 1e-10 tolerance") — still accurate. Check whether `math` remains used in each file (expectation keeps `math` for a second use; sample and fidelity may drop it — verify with `go build ./quantum/` and drop the import only where the compiler says it is unused).

- [ ] **Step 2: Verify**

Run: `go build ./quantum/ && go test ./quantum/`
Expected: build clean, PASS. (`!IsNormalizedSum(x)` is the same NaN-safe negation as the old `!(math.Abs(x-1.0) <= tol)` — identical semantics, existing tests confirm.)

- [ ] **Step 3: Commit**

```bash
git add quantum/sample.go quantum/fidelity.go quantum/expectation.go
git commit -m "refactor(quantum): adopt shared normalization tolerance"
```

---

### Task 3: Cold-path helpers in `backendmath`

**Files:**
- Modify: `internal/backendmath/backendmath.go` (append helpers)
- Test: `internal/backendmath/backendmath_test.go` (append tests)

**Interfaces:**
- Consumes: `quantum.RandomSource` (exists: interface with `Float64() float64`), `quantum.InvalidQubitCountError` (exists).
- Produces (Tasks 5, 6 consume):
  - `func ValidateQubitCount(numQubits int) error`
  - `func RandFloat64(src quantum.RandomSource) float64`

- [ ] **Step 1: Write the failing tests**

Append to `internal/backendmath/backendmath_test.go`:

```go
func TestValidateQubitCount(t *testing.T) {
	for _, n := range []int{1, 2, 64, 100} {
		if err := ValidateQubitCount(n); err != nil {
			t.Errorf("ValidateQubitCount(%d) = %v, want nil", n, err)
		}
	}
	for _, n := range []int{0, -1} {
		err := ValidateQubitCount(n)
		qe, ok := err.(*quantum.InvalidQubitCountError)
		if !ok {
			t.Fatalf("ValidateQubitCount(%d) error type %T, want *quantum.InvalidQubitCountError", n, err)
		}
		if qe.Requested != n || qe.Reason != "must be positive" {
			t.Errorf("fields = (%d, %q), want (%d, \"must be positive\")", qe.Requested, qe.Reason, n)
		}
	}
}

// seqSource yields its values in order, forever, so draws are assertable.
type seqSource struct {
	values []float64
	next   int
}

func (s *seqSource) Float64() float64 {
	v := s.values[s.next%len(s.values)]
	s.next++
	return v
}

func TestRandFloat64(t *testing.T) {
	src := &seqSource{values: []float64{0.25, 0.75}}
	if got := RandFloat64(src); got != 0.25 {
		t.Errorf("first draw = %g, want 0.25", got)
	}
	if got := RandFloat64(src); got != 0.75 {
		t.Errorf("second draw = %g, want 0.75", got)
	}

	// nil falls back to the global source; only assert the draw is a
	// uniform [0, 1) value, not which one.
	if v := RandFloat64(nil); v < 0 || v >= 1 {
		t.Errorf("nil-source draw %g outside [0,1)", v)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/backendmath/ -run 'TestValidateQubitCount|TestRandFloat64'`
Expected: compile failure — both functions undefined.

- [ ] **Step 3: Write the implementation**

Append to `internal/backendmath/backendmath.go` (the file's existing imports `math/rand` and `quantum` must be added to the import block — `errors`, `fmt`, `math` are already there):

```go
// ValidateQubitCount rejects a non-positive register width with the
// InvalidQubitCountError both backends' constructors return.
func ValidateQubitCount(numQubits int) error {
	if numQubits <= 0 {
		return &quantum.InvalidQubitCountError{
			Requested: numQubits,
			Reason:    "must be positive",
		}
	}
	return nil
}

// RandFloat64 draws from src, or from the global math/rand source when
// src is nil — the fallback both backends' Measure paths share.
func RandFloat64(src quantum.RandomSource) float64 {
	if src != nil {
		return src.Float64()
	}
	return rand.Float64()
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/backendmath/ -v`
Expected: PASS, new and existing tests.

- [ ] **Step 5: Commit**

```bash
git add internal/backendmath/backendmath.go internal/backendmath/backendmath_test.go
git commit -m "refactor(backendmath): add qubit-count and rand helpers"
```

---

### Task 4: The `MixCombos` matmul kernel

**Files:**
- Modify: `internal/backendmath/backendmath.go` (append kernel; amend package doc)
- Test: `internal/backendmath/backendmath_test.go` (append tests)

**Interfaces:**
- Consumes: nothing new.
- Produces (Tasks 5, 6 consume):
  - `func MixCombos(matrix [][]complex128, inputs, outputs []complex128)`
  - Contract: `matrix` is `len(outputs)` rows × `len(inputs)` columns; caller owns and shapes all buffers.

- [ ] **Step 1: Write the failing tests**

Append to `internal/backendmath/backendmath_test.go` (its existing `"math"` import covers the needs below):

```go
func TestMixCombos(t *testing.T) {
	invSqrt2 := 1 / math.Sqrt2

	t.Run("identity copies inputs to outputs", func(t *testing.T) {
		identity := [][]complex128{
			{1, 0, 0, 0},
			{0, 1, 0, 0},
			{0, 0, 1, 0},
			{0, 0, 0, 1},
		}
		inputs := []complex128{0.1, 0.2i + 0.3, 0.4, 0.5i}
		outputs := make([]complex128, 4)
		MixCombos(identity, inputs, outputs)
		for i := range inputs {
			if outputs[i] != inputs[i] {
				t.Fatalf("outputs[%d] = %v, want %v", i, outputs[i], inputs[i])
			}
		}
	})

	t.Run("CNOT permutes", func(t *testing.T) {
		cnot := [][]complex128{
			{1, 0, 0, 0},
			{0, 1, 0, 0},
			{0, 0, 0, 1},
			{0, 0, 1, 0},
		}
		inputs := []complex128{0, 0, 1, 0} // |10⟩
		outputs := make([]complex128, 4)
		MixCombos(cnot, inputs, outputs)
		want := []complex128{0, 0, 0, 1} // |11⟩
		for i := range want {
			if outputs[i] != want[i] {
				t.Fatalf("outputs[%d] = %v, want %v", i, outputs[i], want[i])
			}
		}
	})

	t.Run("hadamard on one pair", func(t *testing.T) {
		h := [][]complex128{
			{invSqrt2, invSqrt2},
			{invSqrt2, -invSqrt2},
		}
		inputs := []complex128{1, 0}
		outputs := make([]complex128, 2)
		MixCombos(h, inputs, outputs)
		want := []complex128{complex(invSqrt2, 0), complex(invSqrt2, 0)}
		for i := range want {
			if outputs[i] != want[i] {
				t.Fatalf("outputs[%d] = %v, want %v", i, outputs[i], want[i])
			}
		}
	})

	t.Run("complex 2x2 rotates phases", func(t *testing.T) {
		m := [][]complex128{
			{0, 1i},
			{1i, 0},
		}
		inputs := []complex128{1, 0}
		outputs := make([]complex128, 2)
		MixCombos(m, inputs, outputs)
		if outputs[0] != 0 || outputs[1] != 1i {
			t.Fatalf("outputs = %v, want [0 +1i]", outputs)
		}
	})

	t.Run("toffoli flips only the last combo", func(t *testing.T) {
		// CCX as the 8×8 permutation: |110⟩ → |111⟩.
		toffoli := make([][]complex128, 8)
		for i := range toffoli {
			toffoli[i] = make([]complex128, 8)
			toffoli[i][i] = 1
		}
		toffoli[6][6], toffoli[7][7] = 0, 0
		toffoli[6][7], toffoli[7][6] = 1, 1

		inputs := make([]complex128, 8)
		inputs[6] = 1
		outputs := make([]complex128, 8)
		MixCombos(toffoli, inputs, outputs)
		if outputs[7] != 1 {
			t.Fatalf("outputs[7] = %v, want 1", outputs[7])
		}
		for i := 0; i < 7; i++ {
			if outputs[i] != 0 {
				t.Fatalf("outputs[%d] = %v, want 0", i, outputs[i])
			}
		}
	})
}
```

Note: `0.2i + 0.3` is deliberately `complex(0.3, 0.2)`; adjust if you prefer the explicit form — the value is what matters. `math` and `math/cmplx` must be imported by the test file (`math` already is).

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/backendmath/ -run TestMixCombos`
Expected: compile failure — `MixCombos` undefined.

- [ ] **Step 3: Write the implementation and amend the package doc**

Append to `internal/backendmath/backendmath.go`:

```go
// MixCombos computes one base's worth of gate output, outputs = matrix ·
// inputs, where inputs and outputs hold the 2^k amplitudes a gate mixes
// for one anchor basis state and combo doubles as the matrix row/column
// index (see ComboMasks for the convention).
//
// The caller owns and shapes the buffers — matrix must be len(outputs)
// rows by len(inputs) columns — because the dense backend's inner loops
// keep their bounds checks eliminated only while the slices it iterates
// are shaped in the caller; see state.applyMultiQubitGate.
func MixCombos(matrix [][]complex128, inputs, outputs []complex128) {
	for row := 0; row < len(outputs); row++ {
		sum := complex(0, 0)
		for col := 0; col < len(inputs); col++ {
			sum += matrix[row][col] * inputs[col]
		}
		outputs[row] = sum
	}
}
```

Amend the package doc: replace the sentence

```
// basis state versus a map that prunes near-zero entries — and their loops
// over those containers are theirs alone. What sits between the loops is
// identical physics: which target qubits a gate may address, which basis
// states a gate mixes, and where a measurement leaves the state. Keeping one
```

with

```
// basis state versus a map that prunes near-zero entries — so the loops
// that walk those containers, and the buffers those loops are shaped
// around, are theirs alone. The matrix multiply at the middle of every
// mixing loop is shared here as MixCombos, and so is the rest of the
// physics between the loops: which target qubits a gate may address,
// which basis states a gate mixes, where a measurement leaves the state,
// and the register-width and randomness checks the loops' callers share.
// Keeping one
```

(Read the doc first; splice so the surrounding sentences stay grammatical.)

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/backendmath/ -v`
Expected: PASS, all tests.

- [ ] **Step 5: Commit**

```bash
git add internal/backendmath/backendmath.go internal/backendmath/backendmath_test.go
git commit -m "refactor(backendmath): add shared gate-mixing kernel"
```

---

### Task 5: Rewire the dense backend

**Files:**
- Modify: `state/state.go`

**Interfaces:**
- Consumes: `backendmath.ValidateQubitCount`, `backendmath.RandFloat64`, `backendmath.MixCombos` (Tasks 3–4); `quantum.ValidateAmplitudeVector`, `quantum.CheckNormalization`, `quantum.IsNormalizedSum` (Task 1).
- Produces: no signature changes — `State`'s public surface identical.

- [ ] **Step 1: Capture baseline benchmarks (before any edit)**

```bash
go test ./state/ -run '^$' -bench 'BenchmarkApplySingleQubitGate|BenchmarkApplyMultiQubitGate|BenchmarkApplyGenericTwoQubitGate|BenchmarkApplyThreeQubitGate' -benchtime 1s -count 6 | tee /tmp/dense-bench-before.txt
```

Expected: four benchmarks × 6 runs. Keep the file.

- [ ] **Step 2: Rewire the methods**

Edits to `state/state.go`:

1. `New` — replace the inline guard:

```go
	if err := backendmath.ValidateQubitCount(numQubits); err != nil {
		return nil, err
	}
```

2. Delete the `randFloat64` method (lines 30–35). Its two call sites become `backendmath.RandFloat64(s.randSource)`:
   - `Measure` (line 310): `collapse, err := backendmath.PlanCollapse(qubitIndex, backendmath.RandFloat64(s.randSource), prob0, prob1)`

3. `SetAmplitude` (lines 87–115) becomes:

```go
func (s *State) SetAmplitude(basisState int, value complex128) error {
	if basisState < 0 || basisState >= len(s.amplitudes) {
		return &quantum.QubitsOutOfRangeError{
			Index:    basisState,
			MaxIndex: len(s.amplitudes) - 1,
		}
	}

	if !quantum.IsFiniteAmplitude(value) {
		return &quantum.NonFiniteAmplitudeError{BasisState: basisState, Value: value}
	}

	// Make the change and check normalization
	oldValue := s.amplitudes[basisState]
	s.amplitudes[basisState] = value

	attemptedSum := s.probabilitySum()
	if quantum.IsNormalizedSum(attemptedSum) {
		return nil
	}

	// Restore the previous value
	s.amplitudes[basisState] = oldValue
	return quantum.CheckNormalization(attemptedSum, s.probabilitySum())
}
```

(The old code walked the amplitudes twice — once inside `isNormalized`, once for `attemptedSum`; nothing mutates between, so one walk is provably identical.)

4. `SetAmplitudes` (lines 122–148) becomes:

```go
func (s *State) SetAmplitudes(values []complex128) error {
	sum, err := quantum.ValidateAmplitudeVector(values, len(s.amplitudes))
	if err != nil {
		return err
	}
	if err := quantum.CheckNormalization(sum, s.probabilitySum()); err != nil {
		return err
	}

	// Copy values
	copy(s.amplitudes, values)
	return nil
}
```

Keep the existing doc comments on both methods.

5. Delete `isNormalized` (lines 352–356). Keep `probabilitySum` with its comment.

6. `applyMultiQubitGate` — replace the middle loop only (lines 272–278):

```go
		backendmath.MixCombos(matrix, inputs, outputs)
```

The gather loop (268–270), scatter loop (280–282), buffer shaping, and the BCE comment block (234–244) stay exactly as they are.

7. Imports: `fmt`, `math`, `math/rand` become unused — remove them. Remaining imports: `internal/backendmath`, `quantum`.

- [ ] **Step 3: Verify the package**

```bash
go build ./state/ && go test ./state/ -v 2>&1 | tail -5
```

Expected: build clean, PASS. If any existing test fails, the refactor changed behavior — fix the refactor, not the test.

- [ ] **Step 4: Run the fuzz seed corpora (the dense-vs-sparse guard)**

```bash
go test ./state/ -run 'Fuzz' -v 2>&1 | tail -5
```

Expected: PASS.

- [ ] **Step 5: After-benchmarks and the regression gate**

```bash
go test ./state/ -run '^$' -bench 'BenchmarkApplySingleQubitGate|BenchmarkApplyMultiQubitGate|BenchmarkApplyGenericTwoQubitGate|BenchmarkApplyThreeQubitGate' -benchtime 1s -count 6 | tee /tmp/dense-bench-after.txt
```

Compare medians per benchmark (use `benchstat /tmp/dense-bench-before.txt /tmp/dense-bench-after.txt` if installed, else eyeball the medians). Single-qubit (control) should be flat. Gate: multi-qubit medians within run-to-run noise (~±5%).

**Escape hatch** (only if the gate fails): revert change 6 — restore the inline middle loop in `applyMultiQubitGate` — re-run benchmarks to confirm recovery, and record the numbers in the `backendmath` package doc note on `MixCombos` (dense keeps its inline loop for BCE; sparse uses the kernel). The task still commits; the divergence is documented, per spec.

- [ ] **Step 6: Commit**

```bash
git add state/state.go
git commit -m "refactor(state): use shared backend math"
```

---

### Task 6: Rewire the sparse backend

**Files:**
- Modify: `internal/sparsestate/state.go`

**Interfaces:**
- Consumes: same helpers as Task 5.
- Produces: no signature changes.

- [ ] **Step 1: Rewire the methods**

Edits to `internal/sparsestate/state.go`:

1. `New` — replace the inline guard with:

```go
	if err := backendmath.ValidateQubitCount(numQubits); err != nil {
		return nil, err
	}
```

2. Delete the `randFloat64` method (lines 33–38). Its call site in `Measure` (line 353) becomes `backendmath.RandFloat64(s.randSource)`:

```go
	collapse, err := backendmath.PlanCollapse(qubitIndex, backendmath.RandFloat64(s.randSource), prob0, prob1)
```

3. `SetAmplitude` (lines 81–114) becomes:

```go
func (s *State) SetAmplitude(basisState int, value complex128) error {
	if basisState < 0 || basisState >= (1<<s.numQubits) {
		return &quantum.QubitsOutOfRangeError{
			Index:    basisState,
			MaxIndex: (1 << s.numQubits) - 1,
		}
	}

	// A non-finite amplitude has to be caught before the normalization
	// check, which cannot see it: NaN makes the probability sum NaN, and
	// NaN fails every comparison against the tolerance.
	if !quantum.IsFiniteAmplitude(value) {
		return &quantum.NonFiniteAmplitudeError{BasisState: basisState, Value: value}
	}

	oldValue, had := s.amplitudes[basisState]
	s.setAmplitudeUnsafe(basisState, value)

	attemptedSum := s.probabilitySum()
	if quantum.IsNormalizedSum(attemptedSum) {
		return nil
	}

	if had {
		s.setAmplitudeUnsafe(basisState, oldValue)
	} else {
		delete(s.amplitudes, basisState)
	}
	return quantum.CheckNormalization(attemptedSum, s.probabilitySum())
}
```

4. `SetAmplitudes` (lines 128–162) becomes (keep the doc comment, updating the phrase "the same 1e-10 tolerance the dense backend applies" to "the shared `quantum.NormalizationTolerance`"):

```go
func (s *State) SetAmplitudes(values []complex128) error {
	size := 1 << s.numQubits
	sum, err := quantum.ValidateAmplitudeVector(values, size)
	if err != nil {
		return err
	}
	if err := quantum.CheckNormalization(sum, s.probabilitySum()); err != nil {
		return err
	}

	// Rebuild rather than overwrite: entries the old vector held and the new
	// one leaves at zero must not survive as stale map keys.
	amplitudes := make(map[int]complex128, len(s.amplitudes))
	for i, v := range values {
		if isNearZero(v) {
			continue
		}
		amplitudes[i] = v
	}
	s.amplitudes = amplitudes
	return nil
}
```

5. Delete `isNormalized` (lines 391–394). Keep `probabilitySum`.

6. `applyMultiQubitGate` — replace the middle loop only (lines 265–271):

```go
		backendmath.MixCombos(matrix, inputs, outputs)
```

Gather (261–263) and the pruning scatter (273–279) stay exactly as they are.

7. Imports: `fmt`, `math`, `math/rand` become unused — remove. Keep `math/cmplx` (`isNearZero`), `internal/backendmath`, `quantum`.

- [ ] **Step 2: Verify the package and fuzz seeds**

```bash
go build ./internal/sparsestate/ && go test ./internal/sparsestate/ 2>&1 | tail -3 && go test ./internal/sparsestate/ -run 'Fuzz' 2>&1 | tail -3
```

Expected: build clean, PASS twice (unit + fuzz seeds, including the equivalence fuzz that pits sparse against dense).

- [ ] **Step 3: Sparse benchmarks sanity check**

```bash
go test ./internal/sparsestate/ -run '^$' -bench . -benchtime 1s -count 3
```

Expected: no consistent regression vs. pre-change feel; sparse is not BCE-sensitive, this is a smoke check only.

- [ ] **Step 4: Commit**

```bash
git add internal/sparsestate/state.go
git commit -m "refactor(sparsestate): use shared backend math"
```

---

### Task 7: `qubit` adoption, full verification, backlog doc

**Files:**
- Modify: `qubit/qubit.go:116-118`
- Modify: `docs/enhancement-backlog-2026-08-27.md` (item 8 blockquote)

**Interfaces:**
- Consumes: `quantum.IsNormalizedSum` (Task 1).
- Produces: nothing.

- [ ] **Step 1: Rewire `qubit.IsNormalized`**

```go
// IsNormalized checks if the qubit is properly normalized
func (q *Qubit) IsNormalized() bool {
	sum := quantum.Probability(q.alpha) + quantum.Probability(q.beta)
	return quantum.IsNormalizedSum(sum)
}
```

`math` stays imported (line 65's separate 1e-6 check uses it). Do not touch line 65 — different, deliberately looser check.

- [ ] **Step 2: Verify the package**

```bash
go test ./qubit/
```

Expected: PASS.

- [ ] **Step 3: Full verification sweep**

```bash
go build ./... && go test ./... && go test -race ./... && go vet ./... && gofmt -l .
```

Expected: all PASS, `gofmt -l .` prints nothing. If `gofmt` lists a file, run `gofmt -w <file>` and re-run.

- [ ] **Step 4: Mark backlog item 8 done**

In `docs/enhancement-backlog-2026-08-27.md`, change `- [ ] 8.` to `- [x] 8.` and append below the item (matching the convention of items 1–7):

```markdown
  > **Done (2026-08-28)**: residuals deduped into the two shared homes.
  > Amplitude-vector policy (`NormalizationTolerance`, `IsNormalizedSum`,
  > `CheckNormalization`, `ValidateAmplitudeVector`) lives in
  > `quantum/normalization.go` beside `IsFiniteAmplitude`/`Probability`
  > and replaces the five independent 1e-10 declarations (sample,
  > fidelity, expectation, both backends, qubit). Gate-application
  > physics (`MixCombos` matmul kernel, `ValidateQubitCount`,
  > `RandFloat64`) joins `ValidateTargets`/`ComboMasks`/`PlanCollapse` in
  > `internal/backendmath`. Both backends' `New`/`SetAmplitude`/
  > `SetAmplitudes`/`Measure` now call the shared helpers;
  > `isNormalized` and `randFloat64` are gone; `probabilitySum` stays
  > per-backend (slice vs map iteration is a real difference). Dense
  > `applyMultiQubitGate` keeps its caller-shaped buffers and loop
  > structure (BCE-sensitive) and shares only the middle matmul;
  > benchmarks before/after: no regression beyond noise [or, if the
  > escape hatch fired: dense keeps its inline loop, numbers in the
  > backendmath doc]. Single-qubit 2×2 loops stay per-backend by design.
  > Full suite, race, vet, gofmt, fuzz seeds (including
  > `FuzzDenseSparseGateEquivalence`) clean.
```

Adjust the bracketed benchmark sentence to whichever outcome Task 5 produced — one of the two, not both.

- [ ] **Step 5: Commit**

```bash
git add qubit/qubit.go docs/enhancement-backlog-2026-08-27.md
git commit -m "refactor(qubit): use shared normalization check; mark backlog item 8 done"
```

---

## Self-Review Notes

- Spec coverage: §2 quantum helpers = Tasks 1–2; §3 backendmath helpers = Tasks 3–4; §4 backend rewiring = Tasks 5–6; consumers beyond backends (sample/qubit) = Tasks 2, 7; benchmark gate + escape hatch = Task 5; docs = Tasks 4 (package doc) and 7 (backlog). `groundStateTolerance` untouched per spec — no task touches `algorithm/`.
- Type consistency: `MixCombos(matrix [][]complex128, inputs, outputs []complex128)` used identically in Tasks 4, 5, 6; `CheckNormalization(attemptedSum, currentSum)` argument order identical everywhere; `ValidateAmplitudeVector(values []complex128, size int) (float64, error)` identical.
- No placeholders: every step carries its exact code or command.
