# Example Demos: CHSH, QPE, QAOA — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the deferred backlog item — CHSH, QPE, and QAOA demos, backed by tested protocol code in `algorithm/` and thin printers in `internal/examples/`, wired into `cmd/quantum`.

**Architecture:** Approach A from `docs/superpowers/specs/2026-08-31-example-demos-design.md`: protocol logic lives in `algorithm/` (precedent: `teleport.go`, `h2.go`), presentation in `internal/examples/`. Existing capabilities carry the load: `quantum.Expectation`/`SampleExpectation`, `gates.NewControlled`, `parameterized.Template`, `algorithm.VQE`. The two non-trivial constructions are CHSH's rotated-basis correlation (plain {Z,X} settings cannot violate the bound) and QPE's gate-level subregister inverse QFT (`quantum.InverseQFT` DFTs the whole register, which would mix the eigenstate qubit).

**Tech Stack:** Go standard library only. No new dependencies.

## Global Constraints

- Spec: `docs/superpowers/specs/2026-08-31-example-demos-design.md` (approved).
- No changes to `quantum/`, `gates/`, `parameterized/`, `circuit/`, or `algorithm/vqe.go` — demos are additive only.
- QAOA template parameters each drive exactly one gate (VQE rejects multi-gate parameters).
- Error style in `algorithm/`: typed `Invalid<Input>Error { Reason string }` following `InvalidVQEInputError` in `algorithm/vqe.go:87`.
- Every task: `go build ./...` clean, new tests green, then commit. Never edit a failing test to make code pass.
- Doc comments match the codebase voice: explain the why, cite conventions (qubit 0 = LSB), no filler.

---

### Task 1: CHSH correlation and S value

**Files:**
- Create: `algorithm/chsh.go`
- Test: `algorithm/chsh_test.go`

**Interfaces:**
- Consumes: `quantum.Expectation(s, axes)`, `quantum.SampleExpectation(s, axes, shots, rng)`, `gates.NewRy(theta)`, `state.New`, `gates.NewHadamard/NewCNOT`.
- Produces (exact signatures later tasks and demos rely on):
  - `ChshCorrelation(s quantum.QuantumState, thetaA, thetaB float64) (float64, error)`
  - `ChshSampledCorrelation(s quantum.QuantumState, thetaA, thetaB float64, shots int, rng quantum.RandomSource) (float64, error)`
  - `ChshSExact(s quantum.QuantumState) (float64, error)`
  - `ChshSSampled(s quantum.QuantumState, shots int, rng quantum.RandomSource) (float64, error)`

- [ ] **Step 1: Write the failing tests**

Create `algorithm/chsh_test.go`:

```go
package algorithm

import (
	"math"
	"math/rand"
	"testing"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/quantum"
	"github.com/pjbaur/quantum/state"
)

// bellState returns |Phi+> = (|00> + |11>)/sqrt(2) on qubits 0 and 1.
func bellState(t *testing.T) quantum.QuantumState {
	t.Helper()
	s, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New(2): %v", err)
	}
	if err := s.ApplyGate(gates.NewHadamard(), 0); err != nil {
		t.Fatalf("Hadamard: %v", err)
	}
	if err := s.ApplyGate(gates.NewCNOT(), 0, 1); err != nil {
		t.Fatalf("CNOT: %v", err)
	}
	return s
}

func TestChshCorrelationMatchesCosineOnBellState(t *testing.T) {
	s := bellState(t)
	cases := []struct{ thetaA, thetaB float64 }{
		{0, 0},
		{0, math.Pi / 4},
		{math.Pi / 2, math.Pi / 4},
		{math.Pi / 2, -math.Pi / 4},
		{math.Pi / 3, math.Pi / 6},
	}
	for _, c := range cases {
		got, err := ChshCorrelation(s, c.thetaA, c.thetaB)
		if err != nil {
			t.Fatalf("ChshCorrelation(%g, %g): %v", c.thetaA, c.thetaB, err)
		}
		want := math.Cos(c.thetaA - c.thetaB)
		if math.Abs(got-want) > 1e-9 {
			t.Errorf("ChshCorrelation(%g, %g) = %g, want %g", c.thetaA, c.thetaB, got, want)
		}
	}
}

func TestChshSExactViolatesBoundOnBellState(t *testing.T) {
	s := bellState(t)
	got, err := ChshSExact(s)
	if err != nil {
		t.Fatalf("ChshSExact: %v", err)
	}
	if math.Abs(got-2*math.Sqrt2) > 1e-9 {
		t.Errorf("ChshSExact on Bell state = %g, want 2*sqrt(2) = %g", got, 2*math.Sqrt2)
	}
}

func TestChshSExactStaysUnderBoundOnProductState(t *testing.T) {
	// |00> is a product state: E(thetaA, thetaB) = cos(thetaA)cos(thetaB),
	// so S = sqrt(2) < 2 — no violation without entanglement.
	s, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New(2): %v", err)
	}
	got, err := ChshSExact(s)
	if err != nil {
		t.Fatalf("ChshSExact: %v", err)
	}
	if math.Abs(got-math.Sqrt2) > 1e-9 {
		t.Errorf("ChshSExact on |00> = %g, want sqrt(2)", got)
	}
	if got >= 2 {
		t.Errorf("ChshSExact on |00> = %g, must stay under the classical bound 2", got)
	}
}

func TestChshSSampledSeededLandsNearTsirelson(t *testing.T) {
	s := bellState(t)
	rng := rand.New(rand.NewSource(42))
	got, err := ChshSSampled(s, 400, rng)
	if err != nil {
		t.Fatalf("ChshSSampled: %v", err)
	}
	if got < 2 || got > 2*math.Sqrt2+0.5 {
		t.Errorf("ChshSSampled(400 shots, seed 42) = %g, want within [2, 2*sqrt(2)+0.5]", got)
	}
}

func TestChshCorrelationLeavesOriginalStateUntouched(t *testing.T) {
	s := bellState(t)
	before := []complex128{s.Amplitude(0), s.Amplitude(1), s.Amplitude(2), s.Amplitude(3)}
	if _, err := ChshCorrelation(s, math.Pi/2, math.Pi/4); err != nil {
		t.Fatalf("ChshCorrelation: %v", err)
	}
	for i, a := range before {
		if s.Amplitude(i) != a {
			t.Fatalf("amplitude %d changed: got %v, want %v", i, s.Amplitude(i), a)
		}
	}
}

func TestChshCorrelationRejectsBadStates(t *testing.T) {
	if _, err := ChshCorrelation(nil, 0, 0); err == nil {
		t.Error("ChshCorrelation(nil) must error")
	}
	oneQubit, err := state.New(1)
	if err != nil {
		t.Fatalf("state.New(1): %v", err)
	}
	if _, err := ChshCorrelation(oneQubit, 0, 0); err == nil {
		t.Error("ChshCorrelation on 1-qubit state must error")
	}
}
```

Note: `TestChshSExactStaysUnderBoundOnProductState` first assertion is written out fully as
`math.Abs(got - math.Sqrt2) > 1e-9` — i.e. check `got` against `math.Sqrt2` directly. Use that form.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./algorithm/ -run TestChsh -v`
Expected: FAIL — `undefined: ChshCorrelation` (compile error).

- [ ] **Step 3: Write `algorithm/chsh.go`**

```go
package algorithm

import (
	"math"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/quantum"
)

// chshZZ is the Pauli string both correlation helpers measure after rotating
// each half into the measurement basis.
var chshZZ = []quantum.PauliAxis{quantum.PauliZ, quantum.PauliZ}

// chshRotated returns a clone of s with each half rotated so that measuring
// Z on the clone equals measuring cos(theta)*Z + sin(theta)*X on s. The
// plain Pauli-string API only offers Z/X/Y settings, and those give exactly
// S = 2 on a Bell state — no violation — so the CHSH bases at +-45 degrees
// have to be reached by pre-rotation. The original state is never modified.
func chshRotated(s quantum.QuantumState, thetaA, thetaB float64) (quantum.QuantumState, error) {
	if s == nil {
		return nil, errors.New("state must not be nil")
	}
	rotated := s.Clone()
	if err := rotated.ApplyGate(gates.NewRy(-thetaA), 0); err != nil {
		return nil, err
	}
	if err := rotated.ApplyGate(gates.NewRy(-thetaB), 1); err != nil {
		return nil, err
	}
	return rotated, nil
}

// ChshCorrelation returns the exact correlation E(thetaA, thetaB) =
// <A(thetaA) B(thetaB)> where A/B are Z rotated by thetaA/thetaB in the
// X-Z plane. On the Bell state (|00>+|11>)/sqrt(2) this is
// cos(thetaA - thetaB). thetaA acts on qubit 0, thetaB on qubit 1; s must
// have exactly two qubits.
func ChshCorrelation(s quantum.QuantumState, thetaA, thetaB float64) (float64, error) {
	rotated, err := chshRotated(s, thetaA, thetaB)
	if err != nil {
		return 0, err
	}
	return quantum.Expectation(rotated, chshZZ)
}

// ChshSampledCorrelation estimates the same correlation from shots the way a
// device would, via quantum.SampleExpectation on the rotated clone. rng may
// be nil for the global math/rand source; a seeded source makes the estimate
// reproducible.
func ChshSampledCorrelation(s quantum.QuantumState, thetaA, thetaB float64, shots int, rng quantum.RandomSource) (float64, error) {
	rotated, err := chshRotated(s, thetaA, thetaB)
	if err != nil {
		return 0, err
	}
	return quantum.SampleExpectation(rotated, chshZZ, shots, rng)
}

// chshSettings are the canonical CHSH settings: a0 = 0, a1 = pi/2 for one
// half, b0 = +pi/4, b1 = -pi/4 for the other. On the Bell state they make
// every E = +-sqrt(2)/2 and S = 2*sqrt(2).
var chshSettings = [4]struct{ thetaA, thetaB float64 }{
	{0, math.Pi / 4},
	{0, -math.Pi / 4},
	{math.Pi / 2, math.Pi / 4},
	{math.Pi / 2, -math.Pi / 4},
}

// chshS computes S = E00 + E01 + E10 - E11 from a per-setting correlation
// function, so the exact and sampled variants share one definition.
func chshS(s quantum.QuantumState, correlate func(quantum.QuantumState, float64, float64) (float64, error)) (float64, error) {
	sum := 0.0
	for i, setting := range chshSettings {
		e, err := correlate(s, setting.thetaA, setting.thetaB)
		if err != nil {
			return 0, err
		}
		if i == 3 {
			e = -e
		}
		sum += e
	}
	return sum, nil
}

// ChshSExact returns the exact CHSH S value with the canonical settings.
// 2 <= S <= 2*sqrt(2) on entangled states violates the classical bound 2;
// the Bell state reaches the Tsirelson bound 2*sqrt(2).
func ChshSExact(s quantum.QuantumState) (float64, error) {
	return chshS(s, ChshCorrelation)
}

// ChshSSampled returns the shot-estimated S value with the same settings.
func ChshSSampled(s quantum.QuantumState, shots int, rng quantum.RandomSource) (float64, error) {
	return chshS(s, func(st quantum.QuantumState, a, b float64) (float64, error) {
		return ChshSampledCorrelation(st, a, b, shots, rng)
	})
}
```

Add `"errors"` to the imports.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./algorithm/ -run TestChsh -v`
Expected: PASS, all seven tests.

If `TestChshCorrelationMatchesCosineOnBellState` fails with values matching `cos(thetaA + thetaB)`, the rotation sign is inverted — use `gates.NewRy(+theta)` in `chshRotated` only if the cos(a-b) table says so; do not touch the test.

- [ ] **Step 5: Commit**

```bash
git add algorithm/chsh.go algorithm/chsh_test.go
git commit -m "feat(algorithm): CHSH correlation and S value with rotated bases"
```

---

### Task 2: Subregister inverse QFT decomposition

**Files:**
- Create: `algorithm/qpe.go`
- Test: `algorithm/qpe_test.go`

**Interfaces:**
- Consumes: `circuit.New/AddGate/Execute`, `gates.NewHadamard/NewPhase/NewSwap/NewControlled`, `quantum.InverseQFT`, `state.New`.
- Produces (used by Task 3 and the QPE demo):
  - `appendInverseQFTSub(c *circuit.Circuit, n int) error` (unexported)
  - `subregisterQFTOps(n int) []qftOp` (unexported helper)

- [ ] **Step 1: Write the failing test**

Create `algorithm/qpe_test.go`:

```go
package algorithm

import (
	"math"
	"testing"

	"github.com/pjbaur/quantum/circuit"
	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/quantum"
	"github.com/pjbaur/quantum/state"
)

// applyInverseQFTSub runs the gate decomposition on a fresh n-qubit state.
func applyInverseQFTSub(t *testing.T, n int, prepared func(*state.State)) *state.State {
	t.Helper()
	s, err := state.New(n)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	prepared(s)
	c, err := circuit.New(n)
	if err != nil {
		t.Fatalf("circuit.New: %v", err)
	}
	if err := appendInverseQFTSub(c, n); err != nil {
		t.Fatalf("appendInverseQFTSub: %v", err)
	}
	if err := c.Execute(s); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	return s
}

// TestSubregisterIQFTMatchesQuantumInverseQFT is the convention-pinning
// test: the gate decomposition on qubits 0..n-1 must reproduce
// quantum.InverseQFT amplitudes exactly, for several widths and inputs.
func TestSubregisterIQFTMatchesQuantumInverseQFT(t *testing.T) {
	for n := 1; n <= 4; n++ {
		inputs := []func(*state.State){
			// |0...0>
			func(s *state.State) {},
			// uniform |+...+>
			func(s *state.State) {
				for q := 0; q < n; q++ {
					if err := s.ApplyGate(gates.NewHadamard(), q); err != nil {
						t.Fatalf("Hadamard: %v", err)
					}
				}
			},
		}
		// one non-trivial basis state |x> per width
		x := (1 << n) - 3
		if x < 0 {
			x = 1
		}
		inputs = append(inputs, func(s *state.State) {
			for q := 0; q < n; q++ {
				if x&(1<<q) != 0 {
					if err := s.ApplyGate(gates.NewPauliX(), q); err != nil {
						t.Fatalf("PauliX: %v", err)
					}
				}
			}
		})
		for _, prepare := range inputs {
			got := applyInverseQFTSub(t, n, prepare)

			want, err := state.New(n)
			if err != nil {
				t.Fatalf("state.New: %v", err)
			}
			prepare(want)
			if err := quantum.InverseQFT(want); err != nil {
				t.Fatalf("InverseQFT: %v", err)
			}

			for i := 0; i < (1 << n); i++ {
				g := got.Amplitude(i)
				w := want.Amplitude(i)
				if math.Abs(real(g)-real(w)) > 1e-9 || math.Abs(imag(g)-imag(w)) > 1e-9 {
					t.Fatalf("n=%d input %d: amplitude %d = %v, want %v", n, x, i, g, w)
				}
			}
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./algorithm/ -run TestSubregisterIQFT -v`
Expected: FAIL — `undefined: appendInverseQFTSub` (compile error).

- [ ] **Step 3: Write the decomposition in `algorithm/qpe.go`**

```go
package algorithm

import (
	"math"

	"github.com/pjbaur/quantum/circuit"
	"github.com/pjbaur/quantum/gates"
)

// qftOp is one gate of the subregister QFT decomposition: a Hadamard, a
// controlled-phase, or a swap.
type qftOp struct {
	kind   int // 0 Hadamard, 1 controlled-phase, 2 swap
	ctrl   int
	target int
	angle  float64 // controlled-phase only
}

// subregisterQFTOps returns the gate sequence implementing the QFT on
// qubits 0..n-1 (qubit 0 = LSB) with quantum.QFT's index convention
// F|x> = (1/sqrt(N)) sum_y e^{2*pi*i*x*y/N} |y>: the Nielsen & Chuang
// circuit (Fig. 5.1, H then controlled-R_k chain per qubit, output-order
// swaps) translated from their 1-based MSB-first numbering, where their
// qubit k is our qubit n-k. The swaps at the end are what make the basis
// index map directly onto the register.
func subregisterQFTOps(n int) []qftOp {
	var ops []qftOp
	for q := n - 1; q >= 0; q-- {
		ops = append(ops, qftOp{kind: 0, target: q})
		for ctrl := q - 1; ctrl >= 0; ctrl-- {
			ops = append(ops, qftOp{
				kind:   1,
				ctrl:   ctrl,
				target: q,
				angle:  2 * math.Pi / math.Pow(2, float64(q-ctrl)),
			})
		}
	}
	for q := 0; q < n/2; q++ {
		ops = append(ops, qftOp{kind: 2, ctrl: q, target: n - 1 - q})
	}
	return ops
}

// appendInverseQFTSub appends the inverse QFT on qubits 0..n-1 of c: the
// forward decomposition run backwards with conjugated (negated) phase
// angles. quantum.InverseQFT cannot be used inside phase estimation — it
// DFTs the entire statevector, which would mix the eigenstate qubit into
// the counting register — so the counting subregister gets the gate-level
// form instead.
func appendInverseQFTSub(c *circuit.Circuit, n int) error {
	ops := subregisterQFTOps(n)
	for i := len(ops) - 1; i >= 0; i-- {
		op := ops[i]
		switch op.kind {
		case 0:
			if err := c.AddGate(gates.NewHadamard(), op.target); err != nil {
				return err
			}
		case 1:
			cp, err := gates.NewControlled(gates.NewPhase(-op.angle))
			if err != nil {
				return err
			}
			if err := c.AddGate(cp, op.ctrl, op.target); err != nil {
				return err
			}
		case 2:
			if err := c.AddGate(gates.NewSwap(), op.ctrl, op.target); err != nil {
				return err
			}
		}
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./algorithm/ -run TestSubregisterIQFT -v`
Expected: PASS.

This is the convention-pinning test. If it fails: amplitudes permuted means a swap error in `subregisterQFTOps`; phases conjugated means the angle sign in `appendInverseQFTSub`. Fix the implementation, never the test.

- [ ] **Step 5: Commit**

```bash
git add algorithm/qpe.go algorithm/qpe_test.go
git commit -m "feat(algorithm): gate-level inverse QFT on a counting subregister"
```

---

### Task 3: Phase estimation

**Files:**
- Modify: `algorithm/qpe.go`
- Test: `algorithm/qpe_test.go`

**Interfaces:**
- Consumes: `appendInverseQFTSub` (Task 2), `gates.NewControlled`, `quantum.BulkAmplitudeSetter`, `quantum.UnsupportedOperationError`.
- Produces (used by the QPE demo):
  - `PhaseProbabilities(u quantum.Gate, eigenstate quantum.QuantumState, numCounting int) ([]float64, error)`
  - `EstimatePhase(u quantum.Gate, eigenstate quantum.QuantumState, numCounting int) (bestCount int, phaseTurns float64, err error)`

- [ ] **Step 1: Write the failing tests**

Append to `algorithm/qpe_test.go`:

```go
// oneQubitEigenstate returns a 1-qubit state with bit flipped to |1> if set.
func oneQubitEigenstate(t *testing.T, flip bool) quantum.QuantumState {
	t.Helper()
	s, err := state.New(1)
	if err != nil {
		t.Fatalf("state.New(1): %v", err)
	}
	if flip {
		if err := s.ApplyGate(gates.NewPauliX(), 0); err != nil {
			t.Fatalf("PauliX: %v", err)
		}
	}
	return s
}

func TestEstimatePhaseExactEighths(t *testing.T) {
	for k := 0; k < 8; k++ {
		u := gates.NewPhase(2 * math.Pi * float64(k) / 8)
		best, phase, err := EstimatePhase(u, oneQubitEigenstate(t, true), 3)
		if err != nil {
			t.Fatalf("k=%d: EstimatePhase: %v", k, err)
		}
		if best != k {
			t.Errorf("k=%d: bestCount = %d", k, best)
		}
		if math.Abs(phase-float64(k)/8) > 1e-12 {
			t.Errorf("k=%d: phaseTurns = %g", k, phase)
		}
		probs, err := PhaseProbabilities(u, oneQubitEigenstate(t, true), 3)
		if err != nil {
			t.Fatalf("k=%d: PhaseProbabilities: %v", k, err)
		}
		if probs[k] < 1-1e-9 {
			t.Errorf("k=%d: peak probability = %g, want within 1e-9 of 1", k, probs[k])
		}
	}
}

func TestEstimatePhaseNonRepresentableThird(t *testing.T) {
	u := gates.NewPhase(2 * math.Pi / 3)
	best, _, err := EstimatePhase(u, oneQubitEigenstate(t, true), 3)
	if err != nil {
		t.Fatalf("EstimatePhase: %v", err)
	}
	// 3/8 = 0.375 is the closest eighth to 1/3 = 0.333...
	if best != 3 {
		t.Errorf("bestCount = %d, want 3 (closest eighth to 1/3)", best)
	}
}

func TestEstimatePhaseZeroPhaseEigenstate(t *testing.T) {
	u := gates.NewPhase(2 * math.Pi * 3 / 8)
	best, phase, err := EstimatePhase(u, oneQubitEigenstate(t, false), 3)
	if err != nil {
		t.Fatalf("EstimatePhase: %v", err)
	}
	if best != 0 || phase != 0 {
		t.Errorf("best=%d phase=%g, want 0/0 (|0> is a +1 eigenstate of any Phase gate)", best, phase)
	}
}

func TestEstimatePhaseRejectsBadInputs(t *testing.T) {
	u := gates.NewPhase(math.Pi / 4)
	eig := oneQubitEigenstate(t, true)
	if _, _, err := EstimatePhase(nil, eig, 3); err == nil {
		t.Error("nil gate must error")
	}
	if _, _, err := EstimatePhase(u, nil, 3); err == nil {
		t.Error("nil eigenstate must error")
	}
	if _, _, err := EstimatePhase(u, eig, 0); err == nil {
		t.Error("numCounting < 1 must error")
	}
	twoQubit, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New(2): %v", err)
	}
	if _, _, err := EstimatePhase(u, twoQubit, 3); err == nil {
		t.Error("2-qubit eigenstate must error")
	}
}
```

Note: `gates.NewPhase` takes radians; the eighth-turn phase is `2*math.Pi*float64(k)/8`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./algorithm/ -run TestEstimatePhase -v`
Expected: FAIL — `undefined: EstimatePhase` (compile error).

- [ ] **Step 3: Add `PhaseProbabilities` and `EstimatePhase` to `algorithm/qpe.go`**

Add to imports: `"errors"`, `"fmt"`, `"github.com/pjbaur/quantum/quantum"`, `"github.com/pjbaur/quantum/state"`.

```go
// InvalidQPEInputError indicates a malformed phase-estimation invocation.
type InvalidQPEInputError struct {
	Reason string
}

func (e *InvalidQPEInputError) Error() string {
	return "invalid phase estimation input: " + e.Reason
}

// PhaseProbabilities runs the phase-estimation circuit for the unitary u on
// its 1-qubit eigenstate and returns P(counting register = m) for
// m = 0..2^numCounting-1. Register layout: counting qubits 0..numCounting-1
// (qubit 0 = LSB), eigenstate as qubit numCounting. The circuit is the
// textbook one: Hadamards on the counting register, controlled-u applied
// 2^j times with counting qubit j as control (repetition rather than gate
// powers — no matrix-power machinery, 2^numCounting-1 applications, trivial
// for numCounting <= 4, and it works for any unitary), then the gate-level
// inverse QFT on the counting subregister.
//
// For an exact eigenvector the eigenstate qubit never entangles with the
// counting register, so the probabilities come straight off the counting
// basis states: P(m) = Probability(m) + Probability(m | eigenstate bit).
func PhaseProbabilities(u quantum.Gate, eigenstate quantum.QuantumState, numCounting int) ([]float64, error) {
	if u == nil {
		return nil, &InvalidQPEInputError{Reason: "gate must not be nil"}
	}
	if eigenstate == nil {
		return nil, &InvalidQPEInputError{Reason: "eigenstate must not be nil"}
	}
	if eigenstate.NumQubits() != 1 {
		return nil, &InvalidQPEInputError{Reason: fmt.Sprintf("eigenstate must have 1 qubit, got %d", eigenstate.NumQubits())}
	}
	if numCounting < 1 {
		return nil, &InvalidQPEInputError{Reason: fmt.Sprintf("numCounting must be at least 1, got %d", numCounting)}
	}

	total := numCounting + 1
	s, err := state.New(total)
	if err != nil {
		return nil, err
	}
	// Embed the eigenstate as the top qubit in one bulk write: SetAmplitude
	// would reject each intermediate vector as unnormalized.
	setter, ok := s.(quantum.BulkAmplitudeSetter)
	if !ok {
		return nil, &quantum.UnsupportedOperationError{
			Operation:   "phase estimation",
			Backend:     fmt.Sprintf("%T", s),
			Alternative: "a backend implementing BulkAmplitudeSetter (dense or sparse state)",
		}
	}
	amplitudes := make([]complex128, 1<<total)
	amplitudes[0] = eigenstate.Amplitude(0)
	amplitudes[1<<numCounting] = eigenstate.Amplitude(1)
	if err := setter.SetAmplitudes(amplitudes); err != nil {
		return nil, err
	}

	c, err := circuit.New(total)
	if err != nil {
		return nil, err
	}
	for j := 0; j < numCounting; j++ {
		if err := c.AddGate(gates.NewHadamard(), j); err != nil {
			return nil, err
		}
	}
	cu, err := gates.NewControlled(u)
	if err != nil {
		return nil, err
	}
	for j := 0; j < numCounting; j++ {
		for k := 0; k < 1<<j; k++ {
			// Control first, then target: NewControlled documents control as
			// targets[0], matching CNOT's convention.
			if err := c.AddGate(cu, j, total-1); err != nil {
				return nil, err
			}
		}
	}
	if err := appendInverseQFTSub(c, numCounting); err != nil {
		return nil, err
	}
	if err := c.Execute(s); err != nil {
		return nil, err
	}

	probs := make([]float64, 1<<numCounting)
	eigenBit := 1 << numCounting
	for m := range probs {
		probs[m] = s.Probability(m) + s.Probability(m|eigenBit)
	}
	return probs, nil
}

// EstimatePhase returns the argmax counting outcome and its interpretation
// phaseTurns = bestCount / 2^numCounting of the distribution
// PhaseProbabilities produces. When the eigenphase is exactly representable
// in numCounting bits the peak has probability 1 and the readout is exact.
func EstimatePhase(u quantum.Gate, eigenstate quantum.QuantumState, numCounting int) (int, float64, error) {
	probs, err := PhaseProbabilities(u, eigenstate, numCounting)
	if err != nil {
		return 0, 0, err
	}
	best := 0
	for m, p := range probs {
		if p > probs[best] {
			best = m
		}
	}
	return best, float64(best) / float64(1<<numCounting), nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./algorithm/ -run 'TestEstimatePhase|TestSubregisterIQFT' -v`
Expected: PASS, all tests.

- [ ] **Step 5: Commit**

```bash
git add algorithm/qpe.go algorithm/qpe_test.go
git commit -m "feat(algorithm): phase estimation with subregister IQFT readout"
```

---

### Task 4: QAOA builders — MaxCut Hamiltonian and template

**Files:**
- Create: `algorithm/qaoa.go`
- Test: `algorithm/qaoa_test.go`

**Interfaces:**
- Consumes: `NewHamiltonian/AddTerm/Energy`, `parameterized.NewTemplate/AddParamGate/AddGate/ParamNames/ParamStepCounts/Bind`, `gates.NewCNOT/NewRz/NewRx/NewHadamard`, `parameterized.Rz/Rx`.
- Produces (used by Tasks 5 and 6):
  - `MaxCutHamiltonian(numQubits int, edges [][2]int, weights []float64) (*Hamiltonian, error)`
  - `QAOATemplate(numQubits int, edges [][2]int, layers int) (*parameterized.Template, error)`
  - `CutOfBitstring(edges [][2]int, weights []float64, bits []int) (float64, error)`
  - `ExpectedCut(totalWeight, energy float64) float64`

- [ ] **Step 1: Write the failing tests**

Create `algorithm/qaoa_test.go`:

```go
package algorithm

import (
	"math"
	"testing"

	"github.com/pjbaur/quantum/parameterized"
	"github.com/pjbaur/quantum/quantum"
	"github.com/pjbaur/quantum/state"
)

var triangleEdges = [][2]int{{0, 1}, {0, 2}, {1, 2}}

// triangleBasisState prepares |bits> with bits[k] on qubit k (qubit 0 = LSB).
func triangleBasisState(t *testing.T, bits ...int) quantum.QuantumState {
	t.Helper()
	s, err := state.New(3)
	if err != nil {
		t.Fatalf("state.New(3): %v", err)
	}
	for q, b := range bits {
		if b == 1 {
			if err := s.ApplyGate(gates.NewPauliX(), q); err != nil {
				t.Fatalf("PauliX(%d): %v", q, err)
			}
		}
	}
	return s
}

func TestMaxCutHamiltonianBasisEnergies(t *testing.T) {
	h, err := MaxCutHamiltonian(3, triangleEdges, nil)
	if err != nil {
		t.Fatalf("MaxCutHamiltonian: %v", err)
	}
	cases := []struct {
		bits []int
		want float64
	}{
		{[]int{0, 0, 0}, 3},  // no edges cut: sum of +1 z_i z_j
		{[]int{1, 0, 0}, -1}, // two edges cut
		{[]int{0, 1, 0}, -1},
		{[]int{1, 1, 1}, 3},  // same partition as 000
	}
	for _, c := range cases {
		got, err := h.Energy(triangleBasisState(t, c.bits...))
		if err != nil {
			t.Fatalf("Energy(%v): %v", c.bits, err)
		}
		if math.Abs(got-c.want) > 1e-9 {
			t.Errorf("Energy(%v) = %g, want %g", c.bits, got, c.want)
		}
	}
}

func TestMaxCutHamiltonianWeights(t *testing.T) {
	h, err := MaxCutHamiltonian(3, triangleEdges, []float64{2, 1, 1})
	if err != nil {
		t.Fatalf("MaxCutHamiltonian: %v", err)
	}
	// |100>: edge (0,1) cut with weight 2, edge (0,2) cut with weight 1,
	// edge (1,2) uncut: E = -2 - 1 + 1 = -2.
	got, err := h.Energy(triangleBasisState(t, 1, 0, 0))
	if err != nil {
		t.Fatalf("Energy: %v", err)
	}
	if math.Abs(got-(-2)) > 1e-9 {
		t.Errorf("weighted Energy = %g, want -2", got)
	}
}

func TestQAOATemplateZeroAnglesStartsAtPlusState(t *testing.T) {
	tmpl, err := QAOATemplate(3, triangleEdges, 1)
	if err != nil {
		t.Fatalf("QAOATemplate: %v", err)
	}
	h, err := MaxCutHamiltonian(3, triangleEdges, nil)
	if err != nil {
		t.Fatalf("MaxCutHamiltonian: %v", err)
	}
	params := parameterized.Params{}
	for _, name := range tmpl.ParamNames() {
		params[name] = 0
	}
	c, err := tmpl.Bind(params)
	if err != nil {
		t.Fatalf("Bind: %v", err)
	}
	s, err := state.New(3)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	if err := c.Execute(s); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	// Rz(0) = Rx(0) = I and each edge's CNOT pair cancels, leaving |+++>:
	// <+|Z_i Z_j|+> = 0 per term.
	got, err := h.Energy(s)
	if err != nil {
		t.Fatalf("Energy: %v", err)
	}
	if math.Abs(got) > 1e-9 {
		t.Errorf("zero-angle energy = %g, want 0", got)
	}
}

func TestQAOATemplateParamsDriveSingleGates(t *testing.T) {
	tmpl, err := QAOATemplate(3, triangleEdges, 1)
	if err != nil {
		t.Fatalf("QAOATemplate: %v", err)
	}
	names := tmpl.ParamNames()
	if len(names) != 6 { // 3 gamma_e + 3 beta_q
		t.Fatalf("ParamNames = %v, want 6 names", names)
	}
	for name, count := range tmpl.ParamStepCounts() {
		if count != 1 {
			t.Errorf("parameter %q drives %d steps, VQE requires exactly 1", name, count)
		}
	}
}

func TestQAOATemplateMultiLayerNames(t *testing.T) {
	tmpl, err := QAOATemplate(3, triangleEdges, 2)
	if err != nil {
		t.Fatalf("QAOATemplate: %v", err)
	}
	want := map[string]bool{"gamma_e0_l0": false, "gamma_e0_l1": false, "beta_q2_l1": false}
	for _, name := range tmpl.ParamNames() {
		if _, ok := want[name]; ok {
			want[name] = true
		}
	}
	for name, seen := range want {
		if !seen {
			t.Errorf("ParamNames missing %q for layers=2", name)
		}
	}
}

func TestCutOfBitstringAndExpectedCut(t *testing.T) {
	got, err := CutOfBitstring(triangleEdges, nil, []int{1, 0, 0})
	if err != nil {
		t.Fatalf("CutOfBitstring: %v", err)
	}
	if got != 2 {
		t.Errorf("cut(|100>) = %g, want 2", got)
	}
	weighted, err := CutOfBitstring(triangleEdges, []float64{2, 1, 1}, []int{1, 0, 0})
	if err != nil {
		t.Fatalf("CutOfBitstring: %v", err)
	}
	if weighted != 3 {
		t.Errorf("weighted cut(|100>) = %g, want 3", weighted)
	}
	if c := ExpectedCut(3, -1); c != 2 {
		t.Errorf("ExpectedCut(3, -1) = %g, want 2", c)
	}
}

func TestQAOAInputValidation(t *testing.T) {
	cases := []struct {
		name    string
		num     int
		edges   [][2]int
		weights []float64
	}{
		{"numQubits too small", 1, [][2]int{{0, 1}}, nil},
		{"no edges", 3, nil, nil},
		{"self-loop", 3, [][2]int{{1, 1}}, nil},
		{"vertex out of range", 3, [][2]int{{0, 3}}, nil},
		{"duplicate edge", 3, [][2]int{{0, 1}, {1, 0}}, nil},
		{"weight length mismatch", 3, triangleEdges, []float64{1}},
	}
	for _, c := range cases {
		if _, err := MaxCutHamiltonian(c.num, c.edges, c.weights); err == nil {
			t.Errorf("%s: MaxCutHamiltonian must error", c.name)
		}
		if _, err := QAOATemplate(c.num, c.edges, 1); err == nil {
			t.Errorf("%s: QAOATemplate must error", c.name)
		}
	}
	if _, err := QAOATemplate(3, triangleEdges, 0); err == nil {
		t.Error("layers < 1 must error")
	}
}
```

Add `"github.com/pjbaur/quantum/gates"` to this file's imports.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./algorithm/ -run 'TestMaxCut|TestQAOA|TestCutOf' -v`
Expected: FAIL — `undefined: MaxCutHamiltonian` (compile error).

- [ ] **Step 3: Write `algorithm/qaoa.go`**

```go
package algorithm

import (
	"fmt"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/parameterized"
	"github.com/pjbaur/quantum/quantum"
)

// InvalidQAOAInputError indicates a malformed QAOA graph or template.
type InvalidQAOAInputError struct {
	Reason string
}

func (e *InvalidQAOAInputError) Error() string {
	return "invalid QAOA input: " + e.Reason
}

// normalizeEdges validates the edge list and returns every edge ordered as
// (min, max), rejecting self-loops, out-of-range vertices, and duplicates
// (duplicates would double-count one interaction in both H and the ansatz).
func normalizeEdges(numQubits int, edges [][2]int) ([][2]int, error) {
	if numQubits < 2 {
		return nil, &InvalidQAOAInputError{Reason: fmt.Sprintf("numQubits must be at least 2, got %d", numQubits)}
	}
	if len(edges) == 0 {
		return nil, &InvalidQAOAInputError{Reason: "edge list must not be empty"}
	}
	seen := make(map[[2]int]bool, len(edges))
	normalized := make([][2]int, 0, len(edges))
	for _, e := range edges {
		if e[0] == e[1] {
			return nil, &InvalidQAOAInputError{Reason: fmt.Sprintf("self-loop on qubit %d", e[0])}
		}
		if e[0] < 0 || e[0] >= numQubits || e[1] < 0 || e[1] >= numQubits {
			return nil, &InvalidQAOAInputError{Reason: fmt.Sprintf("edge (%d, %d) out of range for %d qubits", e[0], e[1], numQubits)}
		}
		if e[0] > e[1] {
			e = [2]int{e[1], e[0]}
		}
		if seen[e] {
			return nil, &InvalidQAOAInputError{Reason: fmt.Sprintf("duplicate edge (%d, %d)", e[0], e[1])}
		}
		seen[e] = true
		normalized = append(normalized, e)
	}
	return normalized, nil
}

// MaxCutHamiltonian returns the MaxCut cost operator
//
//	H = sum_e w_e * Z_i Z_j
//
// as a Pauli-string Hamiltonian. Because <Z_i Z_j> = +1 when the endpoints
// agree and -1 when they differ, minimizing <H> maximizes the cut:
// expected cut = (W - <H>) / 2 with W = sum of weights. A nil weights slice
// means unit weights.
func MaxCutHamiltonian(numQubits int, edges [][2]int, weights []float64) (*Hamiltonian, error) {
	if weights != nil && len(weights) != len(edges) {
		return nil, &InvalidQAOAInputError{Reason: fmt.Sprintf("got %d weights for %d edges", len(weights), len(edges))}
	}
	normalized, err := normalizeEdges(numQubits, edges)
	if err != nil {
		return nil, err
	}
	h := NewHamiltonian()
	for i, e := range normalized {
		w := 1.0
		if weights != nil {
			w = weights[i]
		}
		// quantum.Expectation applies axes[k] to qubit k, so the Pauli
		// string is built with PauliZ at each endpoint and I elsewhere.
		axes := make([]quantum.PauliAxis, numQubits)
		for a := range axes {
			axes[a] = quantum.PauliI
		}
		axes[e[0]] = quantum.PauliZ
		axes[e[1]] = quantum.PauliZ
		h.AddTerm(w, axes...)
	}
	return h, nil
}

// QAOATemplate returns the QAOA ansatz for the graph on numQubits qubits
// with the given number of layers, starting from |+...+> (Hadamards are the
// template's fixed gates). Each layer applies, per edge (i, j), the cost
// term e^(-i*gamma*w*Z_i Z_j) as CNOT(i,j) Rz(2*gamma) CNOT(i,j) — the
// standard identity, with Rz(theta) = e^(-i*theta*Z/2) — followed by the
// mixer e^(-i*beta*X_q) = Rx(2*beta) per qubit. Weights are unit; weighted
// cost Hamiltonians can drive VQE but the template builder is unweighted.
//
// Parameters are per edge and per qubit (gamma_e<k>, beta_q<k>, suffixed
// _l<layer> when layers > 1) because the VQE driver requires exactly one
// gate per parameter: a shared gamma across edges would move several gates
// at once and break the parameter-shift gradient.
func QAOATemplate(numQubits int, edges [][2]int, layers int) (*parameterized.Template, error) {
	if layers < 1 {
		return nil, &InvalidQAOAInputError{Reason: fmt.Sprintf("layers must be at least 1, got %d", layers)}
	}
	normalized, err := normalizeEdges(numQubits, edges)
	if err != nil {
		return nil, err
	}
	t := parameterized.NewTemplate(numQubits)
	for q := 0; q < numQubits; q++ {
		if err := t.AddGate(gates.NewHadamard(), q); err != nil {
			return nil, err
		}
	}
	for l := 0; l < layers; l++ {
		suffix := ""
		if layers > 1 {
			suffix = fmt.Sprintf("_l%d", l)
		}
		for k, e := range normalized {
			gammaName := fmt.Sprintf("gamma_e%d%s", k, suffix)
			// The bound value is the rotation angle: e^(-i*gamma*Z_i Z_j)
			// needs Rz(2*gamma) between the CNOTs.
			costFactory := func(value float64) quantum.Gate { return gates.NewRz(2 * value) }
			if err := t.AddGate(gates.NewCNOT(), e[0], e[1]); err != nil {
				return nil, err
			}
			if err := t.AddParamGate(gammaName, costFactory, e[1]); err != nil {
				return nil, err
			}
			if err := t.AddGate(gates.NewCNOT(), e[0], e[1]); err != nil {
				return nil, err
			}
		}
		for q := 0; q < numQubits; q++ {
			betaName := fmt.Sprintf("beta_q%d%s", q, suffix)
			mixFactory := func(value float64) quantum.Gate { return gates.NewRx(2 * value) }
			if err := t.AddParamGate(betaName, mixFactory, q); err != nil {
				return nil, err
			}
		}
	}
	return t, nil
}

// CutOfBitstring evaluates the classical cut value of one outcome:
// the total weight of the edges whose endpoints differ. bits[k] is the
// measured value of qubit k.
func CutOfBitstring(edges [][2]int, weights []float64, bits []int) (float64, error) {
	if weights != nil && len(weights) != len(edges) {
		return 0, &InvalidQAOAInputError{Reason: fmt.Sprintf("got %d weights for %d edges", len(weights), len(edges))}
	}
	cut := 0.0
	for i, e := range edges {
		if e[0] >= len(bits) || e[1] >= len(bits) {
			return 0, &InvalidQAOAInputError{Reason: fmt.Sprintf("edge (%d, %d) needs more than %d bits", e[0], e[1], len(bits))}
		}
		if bits[e[0]] != 0 && bits[e[0]] != 1 || bits[e[1]] != 0 && bits[e[1]] != 1 {
			return 0, &InvalidQAOAInputError{Reason: fmt.Sprintf("bits must be 0 or 1, edge (%d, %d)", e[0], e[1])}
		}
		if bits[e[0]] != bits[e[1]] {
			w := 1.0
			if weights != nil {
				w = weights[i]
			}
			cut += w
		}
	}
	return cut, nil
}

// ExpectedCut converts a cost-Hamiltonian energy into the expected MaxCut
// value: (totalWeight - energy) / 2.
func ExpectedCut(totalWeight, energy float64) float64 {
	return (totalWeight - energy) / 2
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./algorithm/ -run 'TestMaxCut|TestQAOA|TestCutOf' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add algorithm/qaoa.go algorithm/qaoa_test.go
git commit -m "feat(algorithm): MaxCut Hamiltonian and QAOA template builders"
```

---

### Task 5: VQE drives the QAOA ansatz

**Files:**
- Test: `algorithm/qaoa_test.go` (append)

**Interfaces:**
- Consumes: `VQE(h, t, opts)` and unexported `evaluate(h, t, params)` from `algorithm/vqe.go` (same package), plus Task 4's builders.
- Produces: proof the pieces compose — the QAOA demo (Task 6) uses this exact call shape.

- [ ] **Step 1: Write the failing test**

Append to `algorithm/qaoa_test.go`:

```go
// qaoaEnergyAt binds params, executes, and returns the energy — the same
// evaluation path VQE uses internally, exposed for setting the demo's
// starting point from the landscape scan.
func qaoaEnergyAt(t *testing.T, h *Hamiltonian, tmpl *parameterized.Template, params parameterized.Params) float64 {
	t.Helper()
	energy, err := evaluate(h, tmpl, params)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	return energy
}

func TestVQEOptimizesQAOATriangle(t *testing.T) {
	h, err := MaxCutHamiltonian(3, triangleEdges, nil)
	if err != nil {
		t.Fatalf("MaxCutHamiltonian: %v", err)
	}
	tmpl, err := QAOATemplate(3, triangleEdges, 1)
	if err != nil {
		t.Fatalf("QAOATemplate: %v", err)
	}

	// Coarse scan over the symmetric slice (all gamma_e = gamma, all
	// beta_q = beta) to find the demo's starting point. VQE then refines
	// per-edge angles from there.
	bestEnergy := math.Inf(1)
	bestParams := parameterized.Params{}
	for i := 0; i <= 12; i++ {
		gamma := math.Pi * float64(i) / 12
		for j := 0; j <= 6; j++ {
			beta := math.Pi * float64(j) / 12
			params := parameterized.Params{}
			for _, name := range tmpl.ParamNames() {
				if len(name) >= 7 && name[:7] == "gamma_e" {
					params[name] = gamma
				} else {
					params[name] = beta
				}
			}
			if e := qaoaEnergyAt(t, h, tmpl, params); e < bestEnergy {
				bestEnergy = e
				bestParams = params
			}
		}
	}

	result, err := VQE(h, tmpl, VQEOptions{InitialParams: bestParams, MaxIterations: 100})
	if err != nil {
		t.Fatalf("VQE on QAOA template: %v", err)
	}
	if result.Iterations < 1 {
		t.Fatalf("VQE performed no accepted iterations")
	}
	// VQE only accepts non-increasing energies, so it can never end worse
	// than its symmetric-slice starting point.
	if result.Energy > bestEnergy+1e-12 {
		t.Errorf("VQE energy %g worse than initial %g", result.Energy, bestEnergy)
	}
	if result.Energy > 0 {
		t.Errorf("VQE energy %g above the |+>^3 baseline of 0", result.Energy)
	}
	if cut := ExpectedCut(3, result.Energy); cut < 1.5 {
		t.Errorf("expected cut %g below the random-cut baseline 1.5", cut)
	}
}
```

- [ ] **Step 2: Run test to verify it fails (or exposes real behavior)**

Run: `go test ./algorithm/ -run TestVQEOptimizesQAOATriangle -v`
Expected: this test composes existing, individually tested pieces — it may pass on the first run. That is acceptable here: it is a composition guarantee, not new code. If it FAILS because VQE rejects the template (`parameter %q drives %d template steps`), Task 4's `TestQAOATemplateParamsDriveSingleGates` should have caught it first — do not weaken either test; re-check the template construction.

- [ ] **Step 3: Commit**

```bash
git add algorithm/qaoa_test.go
git commit -m "test(algorithm): VQE composes with the QAOA triangle template"
```

---

### Task 6: Demo files

**Files:**
- Create: `internal/examples/chsh.go`
- Create: `internal/examples/qpe.go`
- Create: `internal/examples/qaoa.go`

**Interfaces:**
- Consumes: Task 1 `ChshSExact/ChshSSampled/ChshCorrelation/ChshSampledCorrelation`; Task 3 `PhaseProbabilities/EstimatePhase`; Tasks 4–5 `MaxCutHamiltonian/QAOATemplate/CutOfBitstring/ExpectedCut/VQE/VQEOptions`; existing helpers `PrintBanner`, `PrintRule`, `printState`; `quantum.Sample`.
- Produces: `ChshDemo`, `QpeDemo`, `QaoaDemo`, `RunAllChshDemos`, `RunAllQpeDemos`, `RunAllQaoaDemos` (Task 7 wires these into `cmd/quantum`).

No new unit tests: verification lives in `algorithm/` (spec's verification split). Each demo is verified by running it (Step 4).

- [ ] **Step 1: Write `internal/examples/chsh.go`**

```go
/*
This `internal/examples/chsh.go` file demonstrates the CHSH Bell
inequality:

1. `ChshDemo()` - Prepares a Bell pair, measures the four CHSH correlations
   both exactly and from shots, and assembles the S value that violates the
   classical bound of 2 while staying under the Tsirelson bound 2*sqrt(2).
2. `RunAllChshDemos()` - Runs the demonstration with its banner.

The protocol code lives in `algorithm/chsh.go`; this file is presentation.
*/

package examples

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/pjbaur/quantum/algorithm"
	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/state"
)

// ChshDemo demonstrates CHSH inequality violation on a Bell pair.
func ChshDemo() {
	fmt.Println("\n=== CHSH Inequality Demonstration ===")

	s, err := state.New(2)
	if err != nil {
		fmt.Printf("Error creating state: %v\n", err)
		return
	}
	if err := s.ApplyGate(gates.NewHadamard(), 0); err != nil {
		fmt.Printf("Error applying Hadamard: %v\n", err)
		return
	}
	if err := s.ApplyGate(gates.NewCNOT(), 0, 1); err != nil {
		fmt.Printf("Error applying CNOT: %v\n", err)
		return
	}
	fmt.Println("Bell pair |Phi+> = (|00> + |11>)/sqrt(2):")
	printState(s)

	// The four CHSH settings: Alice measures at a0 = 0, a1 = 90 degrees,
	// Bob at b0 = +45, b1 = -45. The 45-degree bases are why the plain
	// {Z, X} settings cannot show a violation: E is cos(thetaA - thetaB),
	// and only rotated bases push S past 2.
	settings := []struct{ name            string
		thetaA, thetaB float64
	}{
		{"E(a0, b0)", 0, math.Pi / 4},
		{"E(a0, b1)", 0, -math.Pi / 4},
		{"E(a1, b0)", math.Pi / 2, math.Pi / 4},
		{"E(a1, b1)", math.Pi / 2, -math.Pi / 4},
	}
	rng := rand.New(rand.NewSource(7))
	fmt.Println("\nCorrelations (exact vs 400 shots):")
	fmt.Printf("%-12s %-10s %-10s\n", "setting", "exact", "sampled")
	for _, setting := range settings {
		exact, err := algorithm.ChshCorrelation(s, setting.thetaA, setting.thetaB)
		if err != nil {
			fmt.Printf("Error computing %s: %v\n", setting.name, err)
			return
		}
		sampled, err := algorithm.ChshSampledCorrelation(s, setting.thetaA, setting.thetaB, 400, rng)
		if err != nil {
			fmt.Printf("Error sampling %s: %v\n", setting.name, err)
			return
		}
		fmt.Printf("%-12s %-10.4f %-10.4f\n", setting.name, exact, sampled)
	}

	sExact, err := algorithm.ChshSExact(s)
	if err != nil {
		fmt.Printf("Error computing S: %v\n", err)
		return
	}
	sSampled, err := algorithm.ChshSSampled(s, 400, rng)
	if err != nil {
		fmt.Printf("Error sampling S: %v\n", err)
		return
	}
	tsirelson := 2 * math.Sqrt2
	fmt.Printf("\nS = E00 + E01 + E10 - E11\n")
	fmt.Printf("S (exact)   = %.4f\n", sExact)
	fmt.Printf("S (sampled) = %.4f\n", sSampled)
	fmt.Printf("Classical bound: 2     Tsirelson bound: %.4f\n", tsirelson)
	if sSampled > 2 {
		fmt.Println("Verdict: classical bound violated - no local hidden variable model explains this.")
	} else {
		fmt.Println("Verdict: shot noise put S under 2 this run; the exact value carries the violation.")
	}
}

// RunAllChshDemos executes the CHSH demonstration.
func RunAllChshDemos() {
	PrintBanner(45, "    CHSH INEQUALITY DEMONSTRATION")
	ChshDemo()
	PrintRule(45)
}
```

- [ ] **Step 2: Write `internal/examples/qpe.go`**

```go
/*
This `internal/examples/qpe.go` file demonstrates quantum phase estimation:

1. `QpeDemo()` - Estimates the eigenphase of a Phase gate twice: once with
   a 3-bit-exact phase (deterministic readout) and once with 1/3 (a peaked
   distribution showing the finite-register resolution).
2. `RunAllQpeDemos()` - Runs the demonstration with its banner.

The circuit builder lives in `algorithm/qpe.go`; this file is presentation.
*/

package examples

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/pjbaur/quantum/algorithm"
	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/state"
)

// qpeShotTable prints a shot histogram drawn from the exact counting
// distribution. For an exact eigenstate the counting register holds that
// distribution, so drawing from it reproduces device sampling of the
// register.
func qpeShotTable(probs []float64, shots int, rng *rand.Rand) {
	counts := make([]int, len(probs))
	for i := 0; i < shots; i++ {
		r, threshold := rng.Float64(), 0.0
		for m, p := range probs {
			threshold += p
			if r < threshold {
				counts[m]++
				break
			}
		}
	}
	t := 0
	for len(probs) > 1<<t {
		t++
	}
	fmt.Printf("\n%d shots:\n", shots)
	for m, count := range counts {
		if count > 0 {
			fmt.Printf("  |%0*b> = %3d/%d\n", t, m, count, shots)
		}
	}
}

// qpeRun estimates one phase and prints the exact distribution plus the
// shot table.
func qpeRun(label string, phaseTurns float64) {
	t := 3
	u := gates.NewPhase(2 * math.Pi * phaseTurns)
	eigenstate, err := state.New(1)
	if err != nil {
		fmt.Printf("Error creating eigenstate: %v\n", err)
		return
	}
	if err := eigenstate.ApplyGate(gates.NewPauliX(), 0); err != nil {
		fmt.Printf("Error preparing |1>: %v\n", err)
		return
	}
	probs, err := algorithm.PhaseProbabilities(u, eigenstate, t)
	if err != nil {
		fmt.Printf("Error running phase estimation: %v\n", err)
		return
	}
	best, estimate, err := algorithm.EstimatePhase(u, eigenstate, t)
	if err != nil {
		fmt.Printf("Error estimating phase: %v\n", err)
		return
	}

	fmt.Printf("\n%s: U = Phase(2*pi*%.4f), eigenstate |1>, %d counting qubits\n",
		label, phaseTurns, t)
	fmt.Println("Circuit: H on counting register, controlled-U^(2^j) per qubit j,")
	fmt.Println("         inverse QFT on the counting subregister (gate decomposition,")
	fmt.Println("         since quantum.InverseQFT would mix the eigenstate qubit).")
	fmt.Println("\nExact distribution:")
	for m, p := range probs {
		if p > 1e-9 {
			fmt.Printf("  |%0*b>: %.4f\n", t, m, p)
		}
	}
	qpeShotTable(probs, 200, rand.New(rand.NewSource(11)))
	fmt.Printf("Readout: %d/%d = %.4f (true phase %.4f)\n",
		best, 1<<t, estimate, phaseTurns)
}

// QpeDemo demonstrates quantum phase estimation on a Phase gate.
func QpeDemo() {
	fmt.Println("\n=== Quantum Phase Estimation Demonstration ===")

	qpeRun("Exact 3-bit phase", 3.0/8.0)
	fmt.Println()
	qpeRun("Non-representable phase", 1.0/3.0)
	fmt.Println("\nNote: 1/3 is not representable in 3 bits, so the probability")
	fmt.Println("spreads over neighbors of 0.375; more counting qubits narrow it.")
}

// RunAllQpeDemos executes the phase estimation demonstration.
func RunAllQpeDemos() {
	PrintBanner(45, "    QUANTUM PHASE ESTIMATION DEMONSTRATION")
	QpeDemo()
	PrintRule(45)
}
```

- [ ] **Step 3: Write `internal/examples/qaoa.go`**

```go
/*
This `internal/examples/qaoa.go` file demonstrates QAOA for MaxCut on a
triangle:

1. `QaoaDemo()` - Builds the MaxCut Hamiltonian and the per-edge-parameter
   QAOA template, scans the symmetric (gamma, beta) slice of the
   landscape, hands the best symmetric point to the VQE driver, compares
   against the brute-force optimum, and samples the final circuit.
2. `RunAllQaoaDemos()` - Runs the demonstration with its banner.

The builders live in `algorithm/qaoa.go`; this file is presentation.
*/

package examples

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/pjbaur/quantum/algorithm"
	"github.com/pjbaur/quantum/parameterized"
	"github.com/pjbaur/quantum/quantum"
	"github.com/pjbaur/quantum/state"
)

var qaoaTriangleEdges = [][2]int{{0, 1}, {0, 2}, {1, 2}}

// qaoaEnergyAt evaluates the cost at one parameter binding: bind, execute
// on a fresh state, measure the energy.
func qaoaEnergyAt(h *algorithm.Hamiltonian, tmpl *parameterized.Template, params parameterized.Params) (float64, error) {
	c, err := tmpl.Bind(params)
	if err != nil {
		return 0, err
	}
	s, err := state.New(3)
	if err != nil {
		return 0, err
	}
	if err := c.Execute(s); err != nil {
		return 0, err
	}
	return h.Energy(s)
}

// qaoaSymmetricParams expands one (gamma, beta) pair to per-parameter
// values: every gamma_e gets gamma, every beta_q gets beta.
func qaoaSymmetricParams(tmpl *parameterized.Template, gamma, beta float64) parameterized.Params {
	params := parameterized.Params{}
	for _, name := range tmpl.ParamNames() {
		if len(name) >= 7 && name[:7] == "gamma_e" {
			params[name] = gamma
		} else {
			params[name] = beta
		}
	}
	return params
}

// QaoaDemo demonstrates QAOA for MaxCut on the triangle graph.
func QaoaDemo() {
	fmt.Println("\n=== QAOA MaxCut Demonstration ===")

	numQubits := 3
	fmt.Println("Graph: triangle on qubits 0, 1, 2 (all edges weight 1)")
	h, err := algorithm.MaxCutHamiltonian(numQubits, qaoaTriangleEdges, nil)
	if err != nil {
		fmt.Printf("Error building Hamiltonian: %v\n", err)
		return
	}
	tmpl, err := algorithm.QAOATemplate(numQubits, qaoaTriangleEdges, 1)
	if err != nil {
		fmt.Printf("Error building template: %v\n", err)
		return
	}
	fmt.Println("Cost: H = Z0Z1 + Z0Z2 + Z1Z2, expected cut = (3 - E)/2")
	fmt.Printf("Template parameters (one gate each, the VQE driver's precondition):\n  %v\n",
		tmpl.ParamNames())

	// Landscape over the symmetric slice of the 6-parameter space.
	fmt.Println("\nLandscape: expected cut on the symmetric slice (gamma, beta),")
	fmt.Println("           all gamma_e = gamma, all beta_q = beta:")
	bestGamma, bestBeta, bestEnergy := 0.0, 0.0, math.Inf(1)
	fmt.Printf("%-8s", "g\\b")
	for j := 0; j <= 8; j++ {
		fmt.Printf(" %6.2f", math.Pi*float64(j)/32)
	}
	fmt.Println()
	for i := 0; i <= 8; i++ {
		gamma := math.Pi * float64(i) / 16
		fmt.Printf("%-8.4f", gamma)
		for j := 0; j <= 8; j++ {
			beta := math.Pi * float64(j) / 32
			energy, err := qaoaEnergyAt(h, tmpl, qaoaSymmetricParams(tmpl, gamma, beta))
			if err != nil {
				fmt.Printf("Error at (gamma=%g, beta=%g): %v\n", gamma, beta, err)
				return
			}
			if energy < bestEnergy {
				bestGamma, bestBeta, bestEnergy = gamma, beta, energy
			}
			fmt.Printf(" %6.2f", algorithm.ExpectedCut(3, energy))
		}
		fmt.Println()
	}
	fmt.Printf("Best symmetric point: gamma = %.4f, beta = %.4f, cut = %.4f\n",
		bestGamma, bestBeta, algorithm.ExpectedCut(3, bestEnergy))

	result, err := algorithm.VQE(h, tmpl, algorithm.VQEOptions{
		InitialParams: qaoaSymmetricParams(tmpl, bestGamma, bestBeta),
		MaxIterations: 100,
	})
	if err != nil {
		fmt.Printf("Error running VQE: %v\n", err)
		return
	}
	qaoaCut := algorithm.ExpectedCut(3, result.Energy)
	fmt.Printf("\nVQE from the symmetric start: energy = %.4f, expected cut = %.4f\n",
		result.Energy, qaoaCut)
	fmt.Printf("  iterations accepted: %d, energy evaluations: %d\n",
		result.Iterations, result.Evaluations)

	// Classical brute force.
	best := 0.0
	fmt.Println("\nBrute force over all 8 bitstrings (qubit 0 = LSB):")
	for x := 0; x < 8; x++ {
		bits := []int{x & 1, (x >> 1) & 1, (x >> 2) & 1}
		cut, err := algorithm.CutOfBitstring(qaoaTriangleEdges, nil, bits)
		if err != nil {
			fmt.Printf("Error evaluating cut: %v\n", err)
			return
		}
		if cut > best {
			best = cut
		}
		fmt.Printf("  |%03b>: cut %g\n", x, cut)
	}
	fmt.Printf("Classical optimum: %g, QAOA approximation ratio: %.3f\n",
		best, qaoaCut/best)

	// Sample the optimized circuit.
	c, err := tmpl.Bind(result.Params)
	if err != nil {
		fmt.Printf("Error binding optimal params: %v\n", err)
		return
	}
	s, err := state.New(numQubits)
	if err != nil {
		fmt.Printf("Error creating state: %v\n", err)
		return
	}
	if err := c.Execute(s); err != nil {
		fmt.Printf("Error executing circuit: %v\n", err)
		return
	}
	counts, err := quantum.Sample(s, 200, rand.New(rand.NewSource(5)))
	if err != nil {
		fmt.Printf("Error sampling: %v\n", err)
		return
	}
	fmt.Println("\n200 shots of the optimized circuit (key char 0 = qubit 2):")
	for key, count := range counts {
		b0 := int(key[2] - '0')
		b1 := int(key[1] - '0')
		b2 := int(key[0] - '0')
		cut, err := algorithm.CutOfBitstring(qaoaTriangleEdges, nil, []int{b0, b1, b2})
		if err != nil {
			fmt.Printf("Error evaluating cut: %v\n", err)
			return
		}
		fmt.Printf("  %s: %3d  cut %g\n", key, count, cut)
	}
}

// RunAllQaoaDemos executes the QAOA demonstration.
func RunAllQaoaDemos() {
	PrintBanner(45, "    QAOA MAXCUT DEMONSTRATION")
	QaoaDemo()
	PrintRule(45)
}
```

- [ ] **Step 4: Build and verify the package compiles clean**

The CLI words do not exist until Task 7, so this task's verification is compilation plus vet; the actual demo runs happen in Task 7 Step 4.

Run: `go build ./... && go vet ./internal/examples/`
Expected: clean build, no unused-import or unused-function errors (exported `RunAll*` funcs are unused for now by design).

- [ ] **Step 5: Commit**

```bash
git add internal/examples/chsh.go internal/examples/qpe.go internal/examples/qaoa.go
git commit -m "feat(examples): CHSH, QPE, and QAOA demonstrations"
```

---

### Task 7: CLI wiring, docs, full verification

**Files:**
- Modify: `cmd/quantum/main.go` (usage, dispatch, `all` sequence, flag help)
- Modify: `docs/enhancement-backlog-2026-08-27.md`
- Modify: `CHANGELOG.md`

**Interfaces:**
- Consumes: `RunAllChshDemos`, `RunAllQpeDemos`, `RunAllQaoaDemos` (Task 6).
- Produces: `quantum chsh|qpe|qaoa` CLI words; updated docs.

- [ ] **Step 1: Wire `cmd/quantum/main.go`**

Four edits:

1. In `usage()`, after the `gates` line (`fmt.Fprintln(out, "  gates      - Gate catalog and decomposition demonstrations")`), add:

```go
	fmt.Fprintln(out, "  chsh      - CHSH Bell inequality demonstration")
	fmt.Fprintln(out, "  qpe       - Quantum phase estimation demonstration")
	fmt.Fprintln(out, "  qaoa      - QAOA MaxCut demonstration")
```

(Align the dashes with the surrounding lines' column — match the existing padding style exactly.)

2. In `runDemos`, after the `case "gates":` block, add:

```go
	case "chsh":
		examples.RunAllChshDemos()
	case "qpe":
		examples.RunAllQpeDemos()
	case "qaoa":
		examples.RunAllQaoaDemos()
```

3. In the `case "all":` sequence, after `examples.RunAllAlgorithmDemos()` and its existing `p.pause("visualization demonstrations")`, insert before `RunAllVisualizationDemos()`:

```go
		p.pause("CHSH demonstrations")
		examples.RunAllChshDemos()
		p.pause("phase estimation demonstrations")
		examples.RunAllQpeDemos()
		p.pause("QAOA demonstrations")
		examples.RunAllQaoaDemos()
```

4. Update the `-demo` flag description string to include the new words: `"Demo to run (hadamard, tgate, bell, algorithm, chsh, qpe, qaoa, visual, noise, gates, gate, all)"`.

- [ ] **Step 2: Update the backlog**

In `docs/enhancement-backlog-2026-08-27.md`, after the sentence `These block the example-ideas trio (CHSH, teleportation, VQE-toy) and the later ladder stages (QPE, QAOA, noise-aware demos).` (the paragraph under `## Capability Gaps`), insert:

```markdown

> **Demos done (2026-08-31)**: the trio and both later algorithm stages now
> have demos — CHSH, QPE, and QAOA in `internal/examples/{chsh,qpe,qaoa}.go`
> (`quantum chsh|qpe|qaoa`), on protocol code in
> `algorithm/{chsh,qpe,qaoa}.go`. Teleportation and VQE demos predate this
> (`internal/examples/bell.go`, `algorithm/vqe.go`). Design:
> `docs/superpowers/specs/2026-08-31-example-demos-design.md`.
```

- [ ] **Step 3: Update `CHANGELOG.md`**

Under `## Unreleased` / `### Added`, add a new `####` subsection following the existing entries' format (place it after the last existing `####` entry in that section):

```markdown
#### CHSH, QPE, and QAOA protocol demos

`algorithm` gains three protocol modules with exact-value tests:
`algorithm/chsh.go` (rotated-basis CHSH correlations, exact and sampled S
values — the plain {Z, X} settings cannot violate the bound, the bases at
±45° are reached by Ry pre-rotation), `algorithm/qpe.go` (`EstimatePhase` /
`PhaseProbabilities` with a gate-level inverse QFT on the counting
subregister, since `quantum.InverseQFT` DFTs the whole register), and
`algorithm/qaoa.go` (`MaxCutHamiltonian`, `QAOATemplate` with per-edge
parameters so the VQE driver's one-gate-per-parameter rule holds,
`CutOfBitstring`). Demos: `quantum chsh`, `quantum qpe`, `quantum qaoa`.
```

- [ ] **Step 4: Full verification**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: all green.

Run: `go run ./cmd/quantum chsh`
Expected: S exact 2.8284, sampled S > 2, violation verdict.

Run: `go run ./cmd/quantum qpe`
Expected: first run peak probability 1.0 at |011>, readout 3/8 = 0.375; second run peaked at |011> near 0.69, readout 0.375 vs true 0.333.

Run: `go run ./cmd/quantum qaoa`
Expected: landscape table with a visible best cell, VQE cut between 1.5 and 2.0, brute-force optimum 2, sampled bitstrings mostly cut-2 outcomes.

- [ ] **Step 5: Commit**

```bash
git add cmd/quantum/main.go docs/enhancement-backlog-2026-08-27.md CHANGELOG.md
git commit -m "feat(cli): chsh, qpe, qaoa demo words; docs for the demo trio"
```

---

## Self-Review Notes

- Spec coverage: CHSH (Task 1, 6), QPE incl. subregister IQFT rationale (Tasks 2, 3, 6), QAOA incl. VQE + landscape + brute force + sampling (Tasks 4, 5, 6), wiring and docs (Task 7), verification split (algorithm tests only — Task 6 step 4 note).
- One deliberate deviation from the spec text, called out here: `PhaseProbabilities` is exported (the spec listed only `EstimatePhase`) because the QPE demo's exact-distribution table needs it; `EstimatePhase` is a thin argmax over it, so the spec's signature is unchanged.
- Type check: `ChshSExact`/`ChshSSampled` share `chshS`; demo files reference only exported Task 1/3/4 symbols plus `algorithm.VQE`/`VQEOptions`; `qaoaEnergyAt` exists twice with different signatures — once in `algorithm/qaoa_test.go` (takes `*testing.T`, wraps unexported `evaluate`) and once in `internal/examples/qaoa.go` (returns error, uses only exported API). Different packages, no collision.
