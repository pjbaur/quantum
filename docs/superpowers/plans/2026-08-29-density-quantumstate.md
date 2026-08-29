# Density Backend as QuantumState Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Promote `internal/density.Matrix` to a full `quantum.QuantumState` backend so one `circuit.Circuit` executes unchanged against state-vector and density-matrix backends, proven by a noisy-Bell demo.

**Architecture:** `Matrix` gains the six missing interface methods in place. `ApplyGate` computes ρ → UρU† for any gate width by reusing the shared `internal/backendmath` kernel (`ComboMasks`/`MixCombos`) — U applied down columns, conj(U) along rows. `Measure` is projective measurement on ρ via the shared `PlanCollapse`. `Amplitude` reconstructs the state vector when ρ is pure and returns NaN when mixed, so the existing NaN-safe guards in `Sample`/`Expectation`/`Fidelity` reject mixed states cleanly. `SetAmplitude` refuses with `UnsupportedOperationError`.

**Tech Stack:** Go 1.25, stdlib only. Packages touched: `internal/density`, `internal/examples`, `docs`.

**Spec:** `docs/superpowers/specs/2026-08-29-density-quantumstate-design.md`

## Global Constraints

- No new dependencies; no new error types — reuse the taxonomy in `quantum/errortypes.go`.
- Tolerance is always `quantum.NormalizationTolerance` (1e-10); never a new literal.
- Qubit indices are little-endian (qubit 0 = least-significant bit); gate matrix ordering follows the targets slice with targets[0] most significant — same as the state-vector backends.
- Every task ends with `go test ./... && go vet ./...` clean and `gofmt -l .` empty before its commit.
- Kraus channels, Bloch-vector analysis, `FromState`, `Element`, `Trace`, `Purity` stay untouched.

---

### Task 1: Probability, SetAmplitude refusal, Clone, capabilities

**Files:**
- Modify: `internal/density/state.go`
- Test: `internal/density/state_test.go`

**Interfaces:**
- Consumes: existing `Matrix` fields (`numQubits`, `dim`, `data`), `density.FromState`, `quantum.UnsupportedOperationError`, `quantum.RandomSource`.
- Produces: `(*Matrix).Probability(basisState int) float64`, `(*Matrix).SetAmplitude(int, complex128) error`, `(*Matrix).Clone() quantum.QuantumState`, `(*Matrix).SetRandSource(quantum.RandomSource)`, `(*Matrix).SupportsGateQubits(int) bool`, `(*Matrix).MaxGateQubits() int`, and a new `randSource quantum.RandomSource` struct field. Task 4 relies on `randSource` and `SetRandSource`; Task 5 and 6 rely on `Probability` and `Clone`.

- [ ] **Step 1: Write the failing tests**

Append to `internal/density/state_test.go`:

```go
func TestProbabilityIsDiagonal(t *testing.T) {
	invRoot2 := complex(1/math.Sqrt2, 0)
	m, err := FromState(&stubState{numQubits: 1, amplitudes: []complex128{invRoot2, invRoot2}})
	if err != nil {
		t.Fatalf("FromState: %v", err)
	}

	for basis, want := range []float64{0.5, 0.5} {
		if got := m.Probability(basis); math.Abs(got-want) > 1e-12 {
			t.Errorf("Probability(%d) = %g, want %g", basis, got, want)
		}
	}
	if got := m.Probability(-1); got != 0 {
		t.Errorf("Probability(-1) = %g, want 0", got)
	}
	if got := m.Probability(2); got != 0 {
		t.Errorf("Probability(2) = %g, want 0", got)
	}
}

func TestSetAmplitudeUnsupported(t *testing.T) {
	m, err := New(1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	err = m.SetAmplitude(0, 1)
	var unsupported *quantum.UnsupportedOperationError
	if !errors.As(err, &unsupported) {
		t.Fatalf("SetAmplitude error = %v, want *quantum.UnsupportedOperationError", err)
	}
	if m.Element(0, 0) != 1 {
		t.Errorf("refused SetAmplitude mutated the matrix: ρ(0,0) = %v", m.Element(0, 0))
	}
}

func TestCloneIsIndependent(t *testing.T) {
	m, err := New(1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	cloned := m.Clone()
	c, ok := cloned.(*Matrix)
	if !ok {
		t.Fatalf("Clone returned %T, want *Matrix", cloned)
	}

	// Mutate the original through a noise channel; the clone must not move.
	if err := m.ApplyDepolarizing(0, 0.5); err != nil {
		t.Fatalf("ApplyDepolarizing: %v", err)
	}
	if got := c.Element(0, 0); got != 1 {
		t.Errorf("clone ρ(0,0) = %v after mutating original, want 1", got)
	}
	if got := c.Purity(); math.Abs(got-1) > 1e-12 {
		t.Errorf("clone purity = %g after mutating original, want 1", got)
	}
}

func TestBackendCapabilitiesUnlimited(t *testing.T) {
	m, err := New(2)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	for _, k := range []int{1, 2, 3, 5} {
		if !m.SupportsGateQubits(k) {
			t.Errorf("SupportsGateQubits(%d) = false, want true", k)
		}
	}
	if m.SupportsGateQubits(0) {
		t.Error("SupportsGateQubits(0) = true, want false")
	}
	if got := m.MaxGateQubits(); got != 0 {
		t.Errorf("MaxGateQubits() = %d, want 0 (no limit)", got)
	}
}
```

The file already imports `errors`, `math`, and `quantum`; add any of those that `goimports` reports missing.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/density/ -run 'TestProbabilityIsDiagonal|TestSetAmplitudeUnsupported|TestCloneIsIndependent|TestBackendCapabilitiesUnlimited' -v`
Expected: FAIL to compile — `m.Probability undefined`, `m.SetAmplitude undefined`, `m.Clone undefined`, `m.SupportsGateQubits undefined`.

- [ ] **Step 3: Implement the methods**

In `internal/density/state.go`, add a `randSource` field to the struct:

```go
type Matrix struct {
	numQubits  int
	dim        int
	data       []complex128
	temp       []complex128
	scratch    []complex128
	work       []complex128
	randSource quantum.RandomSource
}
```

Add the methods (after `Purity`, before `ReducedBlochVector`):

```go
// SetRandSource sets the randomness source used by Measure, enabling
// reproducible measurements from a seeded generator. A nil source
// restores the default (the global math/rand source).
func (m *Matrix) SetRandSource(src quantum.RandomSource) {
	m.randSource = src
}

// Probability returns the probability of measuring a specific basis
// state: the diagonal element ρᵢᵢ, exact for pure and mixed states alike.
func (m *Matrix) Probability(basisState int) float64 {
	if basisState < 0 || basisState >= m.dim {
		return 0
	}
	return real(m.data[basisState*m.dim+basisState])
}

// SetAmplitude is not supported: a density matrix has no amplitude
// vector to write one entry of. It always returns
// UnsupportedOperationError, the capability-refusal convention
// BulkAmplitudeSetter established. Prepare states with gates, FromState,
// or a state-vector backend instead.
func (m *Matrix) SetAmplitude(basisState int, value complex128) error {
	return &quantum.UnsupportedOperationError{
		Operation:   "SetAmplitude",
		Backend:     "density matrix backend",
		Alternative: "gates, FromState, or a state-vector backend",
	}
}

// Clone creates an independent copy of the density matrix. The clone
// shares the randomness source (if any), so seeded pipelines stay
// deterministic across clones; scratch buffers are not copied.
func (m *Matrix) Clone() quantum.QuantumState {
	data := make([]complex128, len(m.data))
	copy(data, m.data)
	return &Matrix{
		numQubits:  m.numQubits,
		dim:        m.dim,
		data:       data,
		randSource: m.randSource,
	}
}

// SupportsGateQubits returns whether this backend can apply gates
// operating on the specified number of qubits. The density backend
// supports all gate sizes (memory permitting).
func (m *Matrix) SupportsGateQubits(qubitCount int) bool {
	return qubitCount >= 1
}

// MaxGateQubits returns the maximum number of qubits a gate can operate
// on. Returns 0 to indicate no limit.
func (m *Matrix) MaxGateQubits() int {
	return 0
}
```

Do NOT add the `var _ quantum.QuantumState` assertion yet — `ApplyGate`, `Amplitude`, and `Measure` do not exist until Tasks 2–4.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/density/ -v`
Expected: all PASS (new tests and the existing suite).

- [ ] **Step 5: Vet, format, commit**

```bash
gofmt -l . && go vet ./... && go test ./... > /dev/null && echo OK
git add internal/density/state.go internal/density/state_test.go
git commit -m "feat(density): add Probability, Clone, capability methods; refuse SetAmplitude"
```

---

### Task 2: Amplitude — pure-state reconstruction, NaN when mixed

**Files:**
- Modify: `internal/density/state.go`
- Test: `internal/density/state_test.go`

**Interfaces:**
- Consumes: `quantum.NormalizationTolerance`, `(*Matrix).Purity()`.
- Produces: `(*Matrix).Amplitude(basisState int) complex128` — pure ρ: amplitude of the underlying state with the first nonzero-probability basis state's amplitude made real positive (global phase fixed); mixed ρ: `cmplx.NaN()`; out of range: 0. Tasks 5 and 6 rely on this contract.

- [ ] **Step 1: Write the failing tests**

Append to `internal/density/state_test.go`:

```go
func TestAmplitudePureReconstruction(t *testing.T) {
	invRoot2 := complex(1/math.Sqrt2, 0)
	cases := []struct {
		name       string
		amplitudes []complex128
	}{
		{"plus", []complex128{invRoot2, invRoot2}},
		{"relative phase", []complex128{invRoot2, complex(0, 1/math.Sqrt2)}},
		{"global phase", []complex128{complex(0, 1/math.Sqrt2), complex(-1/math.Sqrt2, 0)}},
		{"zero leading amplitude", []complex128{0, 1}},
		{"bell", []complex128{invRoot2, 0, 0, invRoot2}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			n := 1
			if len(tc.amplitudes) == 4 {
				n = 2
			}
			m, err := FromState(&stubState{numQubits: n, amplitudes: tc.amplitudes})
			if err != nil {
				t.Fatalf("FromState: %v", err)
			}

			// The reconstruction is defined up to global phase, so verify it
			// by rebuilding ρ from the reconstructed vector: ψ'ψ'† must equal
			// the matrix it was read from, whatever phase convention holds.
			dim := len(tc.amplitudes)
			recon := make([]complex128, dim)
			for i := range recon {
				recon[i] = m.Amplitude(i)
			}
			for i := 0; i < dim; i++ {
				for j := 0; j < dim; j++ {
					want := m.Element(i, j)
					got := recon[i] * cmplx.Conj(recon[j])
					if cmplx.Abs(got-want) > 1e-12 {
						t.Fatalf("outer product (%d,%d) = %v, want %v", i, j, got, want)
					}
				}
			}

			// Phase convention: the anchor amplitude is real and positive.
			for _, a := range recon {
				if cmplx.Abs(a) > 1e-12 {
					if math.Abs(imag(a)) > 1e-12 || real(a) <= 0 {
						t.Fatalf("first nonzero reconstructed amplitude %v is not real positive", a)
					}
					break
				}
			}
		})
	}
}

func TestAmplitudeNaNWhenMixed(t *testing.T) {
	m, err := New(1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := m.ApplyDepolarizing(0, 0.5); err != nil {
		t.Fatalf("ApplyDepolarizing: %v", err)
	}

	for basis := 0; basis < 2; basis++ {
		if got := m.Amplitude(basis); !cmplx.IsNaN(got) {
			t.Errorf("Amplitude(%d) on mixed state = %v, want NaN", basis, got)
		}
	}
}

func TestAmplitudeOutOfRangeIsZero(t *testing.T) {
	m, err := New(1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got := m.Amplitude(-1); got != 0 {
		t.Errorf("Amplitude(-1) = %v, want 0", got)
	}
	if got := m.Amplitude(2); got != 0 {
		t.Errorf("Amplitude(2) = %v, want 0", got)
	}
}
```

These tests use `math/cmplx`; add it to the test file's imports.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/density/ -run 'TestAmplitude' -v`
Expected: FAIL to compile — `m.Amplitude undefined`.

- [ ] **Step 3: Implement Amplitude**

Add to `internal/density/state.go` (after `Probability`):

```go
// Amplitude returns the amplitude of a specific basis state when ρ is
// pure, and NaN when it is mixed — a mixed state has no amplitude
// vector, and this signature has no error to return. The NaN flows into
// the NaN-safe UnnormalizedStateError guards of Sample, Expectation, and
// Fidelity, which is how those helpers refuse mixed states.
//
// For pure ρ = |ψ⟩⟨ψ|, the vector is recovered from the column of the
// first basis state k with nonzero probability: ψᵢ = ρᵢₖ/√ρₖₖ, which
// fixes the global phase by making ψₖ real positive. Each call scans the
// matrix (the purity check is O(4ⁿ)); callers reading the whole vector
// on the small states this backend serves are still cheap.
func (m *Matrix) Amplitude(basisState int) complex128 {
	if basisState < 0 || basisState >= m.dim {
		return 0
	}

	if math.Abs(m.Purity()-1) > quantum.NormalizationTolerance {
		return cmplx.NaN()
	}

	for k := 0; k < m.dim; k++ {
		diag := real(m.data[k*m.dim+k])
		if diag > quantum.NormalizationTolerance {
			return m.data[basisState*m.dim+k] / complex(math.Sqrt(diag), 0)
		}
	}
	return cmplx.NaN()
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/density/ -v`
Expected: all PASS.

- [ ] **Step 5: Vet, format, commit**

```bash
gofmt -l . && go vet ./... && go test ./... > /dev/null && echo OK
git add internal/density/state.go internal/density/state_test.go
git commit -m "feat(density): Amplitude reconstructs pure states, NaN for mixed"
```

---

### Task 3: ApplyGate — generic k-qubit ρ → UρU†

**Files:**
- Modify: `internal/density/state.go`
- Test: `internal/density/state_test.go`

**Interfaces:**
- Consumes: `quantum.GateQubitCount(gate) (int, error)`, `backendmath.ValidateTargets(targets, numQubits) error`, `backendmath.ComboMasks(masks, targets) int`, `backendmath.MixCombos(matrix, inputs, outputs)`, existing `applySingleQubitOperator(op, target, src, dst)`, `(*Matrix).ensureTemp()`.
- Produces: `(*Matrix).ApplyGate(gate quantum.Gate, targets ...int) error` with the same validation order and error types as `(*state.State).ApplyGate`. Tasks 4–6 rely on it.

- [ ] **Step 1: Write the failing tests**

Append to `internal/density/state_test.go`. The dense-backend comparison imports `github.com/pjbaur/quantum/state` and `github.com/pjbaur/quantum/gates` (both import only `quantum`/`internal/backendmath` — no cycle; this is a test file):

```go
func TestApplyGateMatchesDenseBackend(t *testing.T) {
	type op struct {
		gate    quantum.Gate
		targets []int
	}
	cases := []struct {
		name      string
		numQubits int
		ops       []op
	}{
		{"hadamard k=1", 1, []op{
			{gates.NewHadamard(), []int{0}},
		}},
		{"bell k=2", 2, []op{
			{gates.NewHadamard(), []int{0}},
			{gates.NewCNOT(), []int{0, 1}},
		}},
		{"toffoli k=3", 3, []op{
			{gates.NewPauliX(), []int{0}},
			{gates.NewHadamard(), []int{1}},
			{gates.NewToffoli(), []int{0, 1, 2}},
			{gates.NewCNOT(), []int{2, 0}},
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dense, err := state.New(tc.numQubits)
			if err != nil {
				t.Fatalf("state.New: %v", err)
			}
			m, err := New(tc.numQubits)
			if err != nil {
				t.Fatalf("New: %v", err)
			}

			for _, o := range tc.ops {
				if err := dense.ApplyGate(o.gate, o.targets...); err != nil {
					t.Fatalf("dense ApplyGate(%s): %v", o.gate.Name(), err)
				}
				if err := m.ApplyGate(o.gate, o.targets...); err != nil {
					t.Fatalf("density ApplyGate(%s): %v", o.gate.Name(), err)
				}
			}

			want, err := FromState(dense)
			if err != nil {
				t.Fatalf("FromState: %v", err)
			}
			dim := 1 << tc.numQubits
			for i := 0; i < dim; i++ {
				for j := 0; j < dim; j++ {
					if diff := cmplx.Abs(m.Element(i, j) - want.Element(i, j)); diff > 1e-12 {
						t.Fatalf("ρ(%d,%d) = %v, want %v (diff %g)",
							i, j, m.Element(i, j), want.Element(i, j), diff)
					}
				}
			}
			if got := m.Trace(); math.Abs(got-1) > 1e-12 {
				t.Errorf("trace after circuit = %g, want 1", got)
			}
		})
	}
}

func TestApplyGateValidation(t *testing.T) {
	m, err := New(2)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	var gateErr *quantum.InvalidGateApplicationError
	if err := m.ApplyGate(gates.NewCNOT(), 0); !errors.As(err, &gateErr) {
		t.Errorf("CNOT with one target: err = %v, want *InvalidGateApplicationError", err)
	}

	var rangeErr *quantum.QubitsOutOfRangeError
	if err := m.ApplyGate(gates.NewHadamard(), 2); !errors.As(err, &rangeErr) {
		t.Errorf("target out of range: err = %v, want *QubitsOutOfRangeError", err)
	}

	if err := m.ApplyGate(gates.NewCNOT(), 1, 1); !errors.Is(err, backendmath.ErrDuplicateTargets) {
		t.Errorf("duplicate targets: err = %v, want ErrDuplicateTargets", err)
	}

	// A rejected application leaves ρ untouched.
	if got := m.Element(0, 0); got != 1 {
		t.Errorf("ρ(0,0) = %v after rejected applications, want 1", got)
	}
}
```

Add imports `github.com/pjbaur/quantum/gates`, `github.com/pjbaur/quantum/state`, `github.com/pjbaur/quantum/internal/backendmath` to the test file.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/density/ -run 'TestApplyGate(Matches|Validation)' -v`
Expected: FAIL to compile — `m.ApplyGate undefined`.

- [ ] **Step 3: Implement ApplyGate**

Add to `internal/density/state.go` (import `github.com/pjbaur/quantum/internal/backendmath`), placed after `ApplySingleQubitGate`:

```go
// ApplyGate applies a k-qubit unitary as ρ → U ρ U†, satisfying
// quantum.QuantumState. Qubit indices are little-endian and the gate
// matrix ordering follows the targets slice, with targets[0] as the most
// significant bit — the same convention as the state-vector backends.
// Cost is O(4ⁿ·4ᵏ) for an n-qubit register.
func (m *Matrix) ApplyGate(gate quantum.Gate, targets ...int) error {
	requiredQubits, err := quantum.GateQubitCount(gate)
	if err != nil {
		return err
	}

	if len(targets) != requiredQubits {
		return &quantum.InvalidGateApplicationError{
			Gate:        gate.Name(),
			RequiredLen: requiredQubits,
			ActualLen:   len(targets),
		}
	}

	if err := backendmath.ValidateTargets(targets, m.numQubits); err != nil {
		return err
	}

	if requiredQubits == 1 {
		matrix := gate.Matrix()
		if len(matrix) != 2 || len(matrix[0]) != 2 {
			return &quantum.InvalidGateApplicationError{
				Gate:        gate.Name(),
				RequiredLen: 2,
				ActualLen:   len(matrix),
			}
		}
		m.applySingleQubitOperator(matrix, targets[0], m.data, m.data)
		return nil
	}

	m.applyMultiQubitGate(gate.Matrix(), targets)
	return nil
}

// applyMultiQubitGate computes ρ → U ρ U† for a k-qubit gate, k ≥ 2.
//
// Left pass (Uρ): every column of ρ transforms exactly like a state
// vector, so the shared ComboMasks/MixCombos kernel applies per column.
// Right pass ((Uρ)U†): ((Uρ)U†)ᵢⱼ = Σₖ (Uρ)ᵢₖ·conj(Uⱼₖ), which is the
// same mixing along each row with the conjugated matrix.
//
// Buffers are allocated per call: unlike the dense backend's
// bounds-check-sensitive loops, the two O(4ⁿ) passes dominate the cost
// here, so caller-shaped buffer plumbing buys nothing.
func (m *Matrix) applyMultiQubitGate(matrix [][]complex128, targets []int) {
	comboCount := 1 << len(targets)

	comboMasks := make([]int, comboCount)
	targetMask := backendmath.ComboMasks(comboMasks, targets)

	conj := make([][]complex128, comboCount)
	for r := range conj {
		conj[r] = make([]complex128, comboCount)
		for c := range conj[r] {
			conj[r][c] = cmplx.Conj(matrix[r][c])
		}
	}

	inputs := make([]complex128, comboCount)
	outputs := make([]complex128, comboCount)
	temp := m.ensureTemp()

	// Left pass: temp = U·ρ, mixing down each column j.
	for base := 0; base < m.dim; base++ {
		if base&targetMask != 0 {
			continue
		}
		for j := 0; j < m.dim; j++ {
			for combo := 0; combo < comboCount; combo++ {
				inputs[combo] = m.data[(base|comboMasks[combo])*m.dim+j]
			}
			backendmath.MixCombos(matrix, inputs, outputs)
			for combo := 0; combo < comboCount; combo++ {
				temp[(base|comboMasks[combo])*m.dim+j] = outputs[combo]
			}
		}
	}

	// Right pass: data = temp·U†, mixing along each row i with conj(U).
	for i := 0; i < m.dim; i++ {
		row := i * m.dim
		for base := 0; base < m.dim; base++ {
			if base&targetMask != 0 {
				continue
			}
			for combo := 0; combo < comboCount; combo++ {
				inputs[combo] = temp[row+(base|comboMasks[combo])]
			}
			backendmath.MixCombos(conj, inputs, outputs)
			for combo := 0; combo < comboCount; combo++ {
				m.data[row+(base|comboMasks[combo])] = outputs[combo]
			}
		}
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/density/ -v`
Expected: all PASS.

- [ ] **Step 5: Vet, format, commit**

```bash
gofmt -l . && go vet ./... && go test ./... > /dev/null && echo OK
git add internal/density/state.go internal/density/state_test.go
git commit -m "feat(density): generic k-qubit ApplyGate via shared backendmath kernel"
```

---

### Task 4: Measure — projective measurement on ρ; interface assertions

**Files:**
- Modify: `internal/density/state.go`
- Test: `internal/density/state_test.go`

**Interfaces:**
- Consumes: `backendmath.PlanCollapse(qubitIndex, draw, prob0, prob1) (Collapse, error)`, `backendmath.RandFloat64(src)`, `Collapse.Keeps(basisState) bool`, `Collapse.Renormalize(amplitude) complex128`, `randSource` field from Task 1.
- Produces: `(*Matrix).Measure(qubitIndex int) (int, error)`; compile-time assertions `var _ quantum.QuantumState = (*Matrix)(nil)` and `var _ quantum.BackendCapabilities = (*Matrix)(nil)`. Tasks 5–6 rely on full conformance.

- [ ] **Step 1: Write the failing tests**

Append to `internal/density/state_test.go`:

```go
// stubRandSource returns queued draws in order, so measurement outcomes
// are forced deterministically.
type stubRandSource struct {
	draws []float64
	next  int
}

func (s *stubRandSource) Float64() float64 {
	v := s.draws[s.next]
	s.next++
	return v
}

func TestMeasureForcedOutcomes(t *testing.T) {
	for _, tc := range []struct {
		draw    float64
		outcome int
		surviving int
	}{
		{0.1, 0, 0}, // draw < prob0 → outcome 0, state |0⟩⟨0|
		{0.9, 1, 1}, // draw ≥ prob0 → outcome 1, state |1⟩⟨1|
	} {
		m, err := New(1)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		if err := m.ApplyGate(gates.NewHadamard(), 0); err != nil {
			t.Fatalf("ApplyGate: %v", err)
		}
		m.SetRandSource(&stubRandSource{draws: []float64{tc.draw}})

		outcome, err := m.Measure(0)
		if err != nil {
			t.Fatalf("Measure: %v", err)
		}
		if outcome != tc.outcome {
			t.Errorf("draw %g: outcome = %d, want %d", tc.draw, outcome, tc.outcome)
		}
		if got := m.Probability(tc.surviving); math.Abs(got-1) > 1e-12 {
			t.Errorf("draw %g: P(%d) = %g after collapse, want 1", tc.draw, tc.surviving, got)
		}
		if got := m.Purity(); math.Abs(got-1) > 1e-12 {
			t.Errorf("draw %g: purity = %g after collapse, want 1", tc.draw, got)
		}
		if got := m.Trace(); math.Abs(got-1) > 1e-12 {
			t.Errorf("draw %g: trace = %g after collapse, want 1", tc.draw, got)
		}
	}
}

func TestMeasureCollapsesEntangledPartner(t *testing.T) {
	m, err := New(2)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := m.ApplyGate(gates.NewHadamard(), 0); err != nil {
		t.Fatalf("H: %v", err)
	}
	if err := m.ApplyGate(gates.NewCNOT(), 0, 1); err != nil {
		t.Fatalf("CNOT: %v", err)
	}
	m.SetRandSource(&stubRandSource{draws: []float64{0.9}})

	outcome, err := m.Measure(0)
	if err != nil {
		t.Fatalf("Measure: %v", err)
	}
	if outcome != 1 {
		t.Fatalf("outcome = %d, want 1", outcome)
	}
	// Bell pair: measuring qubit 0 as 1 leaves |11⟩ with certainty.
	if got := m.Probability(3); math.Abs(got-1) > 1e-12 {
		t.Errorf("P(11) = %g after measuring qubit 0, want 1", got)
	}
}

func TestMeasureRejectsOutOfRangeQubit(t *testing.T) {
	m, err := New(1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	var rangeErr *quantum.QubitsOutOfRangeError
	if _, err := m.Measure(1); !errors.As(err, &rangeErr) {
		t.Errorf("Measure(1) err = %v, want *QubitsOutOfRangeError", err)
	}
	if _, err := m.Measure(-1); !errors.As(err, &rangeErr) {
		t.Errorf("Measure(-1) err = %v, want *QubitsOutOfRangeError", err)
	}
}

func TestMeasureRejectsUnnormalizedMatrix(t *testing.T) {
	m, err := New(1)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	m.data[0] = 0 // zero matrix: no branch holds probability

	if _, err := m.Measure(0); err == nil {
		t.Error("Measure on zero matrix succeeded, want unnormalized-state error")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/density/ -run 'TestMeasure' -v`
Expected: FAIL to compile — `m.Measure undefined`.

- [ ] **Step 3: Implement Measure and add conformance assertions**

Add to `internal/density/state.go` (after `ApplyGate`):

```go
// Measure performs a projective measurement of one qubit and collapses
// ρ in place: outcome b with probability Σ ρᵢᵢ over basis states whose
// qubit bit is b, then ρ → ΠρΠ / p. Randomness comes from the source
// set via SetRandSource (global math/rand by default), so outcomes are
// forceable in tests. The shared PlanCollapse supplies the outcome
// choice and its zero-branch and unnormalized-state safety.
func (m *Matrix) Measure(qubitIndex int) (int, error) {
	if qubitIndex < 0 || qubitIndex >= m.numQubits {
		return 0, &quantum.QubitsOutOfRangeError{
			Index:    qubitIndex,
			MaxIndex: m.numQubits - 1,
		}
	}

	prob0, prob1 := 0.0, 0.0
	for i := 0; i < m.dim; i++ {
		p := real(m.data[i*m.dim+i])
		if (i>>qubitIndex)&1 == 0 {
			prob0 += p
		} else {
			prob1 += p
		}
	}

	collapse, err := backendmath.PlanCollapse(qubitIndex, backendmath.RandFloat64(m.randSource), prob0, prob1)
	if err != nil {
		return 0, err
	}

	// ρᵢⱼ survives only when both indices agree with the outcome, scaled
	// by 1/p. Renormalize divides by √p, so applying it to both the row
	// and column factor of each element divides by p, exactly ΠρΠ/p.
	for i := 0; i < m.dim; i++ {
		row := i * m.dim
		for j := 0; j < m.dim; j++ {
			if collapse.Keeps(i) && collapse.Keeps(j) {
				m.data[row+j] = collapse.Renormalize(collapse.Renormalize(m.data[row+j]))
			} else {
				m.data[row+j] = 0
			}
		}
	}

	return collapse.Outcome, nil
}
```

Add the conformance assertions after the `Matrix` struct declaration in `internal/density/state.go`:

```go
// Matrix implements quantum.QuantumState (see ADR-0009) and
// quantum.BackendCapabilities. It deliberately does not implement
// quantum.BulkAmplitudeSetter (a mixed state has no amplitude vector to
// replace, so QFT refuses it) or quantum.Resetter (no consumer).
var (
	_ quantum.QuantumState        = (*Matrix)(nil)
	_ quantum.BackendCapabilities = (*Matrix)(nil)
)
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/density/ -v`
Expected: all PASS.

- [ ] **Step 5: Vet, format, commit**

```bash
gofmt -l . && go vet ./... && go test ./... > /dev/null && echo OK
git add internal/density/state.go internal/density/state_test.go
git commit -m "feat(density): projective Measure; Matrix now implements QuantumState"
```

---

### Task 5: Cross-package integration — circuit execution and helper guards

**Files:**
- Create: `internal/density/integration_test.go` (package `density_test`)

**Interfaces:**
- Consumes: full `Matrix` conformance from Tasks 1–4; `circuit.New/AddGate/Execute`; `quantum.Sample`, `quantum.Fidelity`, `quantum.Expectation`, `quantum.QFT`; `quantum.PauliZ`/`PauliX` axes.
- Produces: nothing new — locks the promoted contract end to end.

- [ ] **Step 1: Write the tests**

Create `internal/density/integration_test.go`:

```go
package density_test

import (
	"errors"
	"math"
	"testing"

	"github.com/pjbaur/quantum/circuit"
	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/internal/density"
	"github.com/pjbaur/quantum/quantum"
	"github.com/pjbaur/quantum/state"
)

// bellCircuit builds the two-qubit Bell circuit used throughout: H on
// qubit 0, then CNOT with qubit 0 controlling qubit 1.
func bellCircuit(t *testing.T) *circuit.Circuit {
	t.Helper()
	c, err := circuit.New(2)
	if err != nil {
		t.Fatalf("circuit.New: %v", err)
	}
	if err := c.AddGate(gates.NewHadamard(), 0); err != nil {
		t.Fatalf("AddGate(H): %v", err)
	}
	if err := c.AddGate(gates.NewCNOT(), 0, 1); err != nil {
		t.Fatalf("AddGate(CNOT): %v", err)
	}
	return c
}

// bellPair returns a density matrix holding a Bell pair, built by
// executing the circuit — the interchangeability being tested.
func bellPair(t *testing.T) *density.Matrix {
	t.Helper()
	m, err := density.New(2)
	if err != nil {
		t.Fatalf("density.New: %v", err)
	}
	if err := bellCircuit(t).Execute(m); err != nil {
		t.Fatalf("Execute on density: %v", err)
	}
	return m
}

func TestCircuitExecutesInterchangeably(t *testing.T) {
	c := bellCircuit(t)

	dense, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	if err := c.Execute(dense); err != nil {
		t.Fatalf("Execute on dense: %v", err)
	}
	m := bellPair(t)

	for basis := 0; basis < 4; basis++ {
		if diff := math.Abs(m.Probability(basis) - dense.Probability(basis)); diff > 1e-12 {
			t.Errorf("P(%d): density %g vs dense %g", basis,
				m.Probability(basis), dense.Probability(basis))
		}
	}
}

func TestExpectationOnPureDensityState(t *testing.T) {
	m := bellPair(t)
	for _, tc := range []struct {
		name string
		axes []quantum.PauliAxis
		want float64
	}{
		{"ZZ", []quantum.PauliAxis{quantum.PauliZ, quantum.PauliZ}, 1},
		{"XX", []quantum.PauliAxis{quantum.PauliX, quantum.PauliX}, 1},
	} {
		got, err := quantum.Expectation(m, tc.axes)
		if err != nil {
			t.Fatalf("Expectation(%s): %v", tc.name, err)
		}
		if math.Abs(got-tc.want) > 1e-10 {
			t.Errorf("⟨%s⟩ = %g, want %g", tc.name, got, tc.want)
		}
	}
}

func TestHelpersRejectMixedDensityState(t *testing.T) {
	m := bellPair(t)
	if err := m.ApplyDepolarizing(0, 0.5); err != nil {
		t.Fatalf("ApplyDepolarizing: %v", err)
	}

	var unnormalized *quantum.UnnormalizedStateError

	if _, err := quantum.Sample(m, 10, nil); !errors.As(err, &unnormalized) {
		t.Errorf("Sample on mixed state: err = %v, want *UnnormalizedStateError", err)
	}
	if _, err := quantum.Expectation(m, []quantum.PauliAxis{quantum.PauliZ, quantum.PauliZ}); !errors.As(err, &unnormalized) {
		t.Errorf("Expectation on mixed state: err = %v, want *UnnormalizedStateError", err)
	}
	dense, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	if _, err := quantum.Fidelity(dense, m); !errors.As(err, &unnormalized) {
		t.Errorf("Fidelity with mixed state: err = %v, want *UnnormalizedStateError", err)
	}
}

func TestFidelityDenseVsPureDensityIsOne(t *testing.T) {
	c := bellCircuit(t)
	dense, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	if err := c.Execute(dense); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	m := bellPair(t)

	got, err := quantum.Fidelity(dense, m)
	if err != nil {
		t.Fatalf("Fidelity: %v", err)
	}
	if math.Abs(got-1) > 1e-10 {
		t.Errorf("Fidelity(dense Bell, density Bell) = %g, want 1", got)
	}
}

func TestQFTRefusesDensityBackend(t *testing.T) {
	m, err := density.New(2)
	if err != nil {
		t.Fatalf("density.New: %v", err)
	}
	var unsupported *quantum.UnsupportedOperationError
	if err := quantum.QFT(m); !errors.As(err, &unsupported) {
		t.Errorf("QFT on density backend: err = %v, want *UnsupportedOperationError", err)
	}
}

func TestSampleAgreesWithDenseOnPureState(t *testing.T) {
	c := bellCircuit(t)
	dense, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	if err := c.Execute(dense); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	m := bellPair(t)

	// Identical fixed draws must produce identical histograms, because a
	// pure density state reconstructs the same probabilities.
	draws := []float64{0.1, 0.6, 0.4, 0.9, 0.2, 0.7}
	denseHist, err := quantum.Sample(dense, len(draws), &fixedSource{draws: draws})
	if err != nil {
		t.Fatalf("Sample(dense): %v", err)
	}
	densityHist, err := quantum.Sample(m, len(draws), &fixedSource{draws: draws})
	if err != nil {
		t.Fatalf("Sample(density): %v", err)
	}
	if len(denseHist) != len(densityHist) {
		t.Fatalf("histograms differ: dense %v, density %v", denseHist, densityHist)
	}
	for k, v := range denseHist {
		if densityHist[k] != v {
			t.Errorf("histogram[%q]: dense %d, density %d", k, v, densityHist[k])
		}
	}
}

// fixedSource replays a fixed sequence of draws.
type fixedSource struct {
	draws []float64
	next  int
}

func (s *fixedSource) Float64() float64 {
	v := s.draws[s.next]
	s.next++
	return v
}
```

- [ ] **Step 2: Run the tests**

Run: `go test ./internal/density/ -v`
Expected: all PASS. These tests assert behavior already built in Tasks 1–4; a failure here is a real defect in those tasks — fix the implementation, not the test.

- [ ] **Step 3: Vet, format, commit**

```bash
gofmt -l . && go vet ./... && go test ./... > /dev/null && echo OK
git add internal/density/integration_test.go
git commit -m "test(density): lock circuit interchangeability and helper guards"
```

---

### Task 6: Noisy-Bell demo; remove ApplySingleQubitGate

**Files:**
- Modify: `internal/examples/noise.go` (new demo, updated call sites, wiring)
- Modify: `internal/density/state.go` (delete `ApplySingleQubitGate`)
- Modify: `internal/density/state_test.go` (migrate its callers to `ApplyGate`)

**Interfaces:**
- Consumes: `circuit.New/AddGate/Execute`, `density.New`, `(*Matrix).ApplyGate/Probability/Element/Purity/ApplyDepolarizing`, `state.New`, `gates.NewHadamard/NewCNOT`.
- Produces: `examples.NoisyBellDemo()`, wired into `RunAllNoiseDemos`. `ApplySingleQubitGate` no longer exists; every caller uses `ApplyGate(g, target)`.

- [ ] **Step 1: Migrate existing callers off ApplySingleQubitGate**

In `internal/examples/noise.go`, replace both call sites (currently in `DephasingDemo` and `AmplitudeDampingDemo`):

```go
		if err := m.ApplyGate(gates.NewHadamard(), 0); err != nil {
			fmt.Printf("Error applying Hadamard gate: %v\n", err)
			return
		}
```

```go
		if err := m.ApplyGate(gates.NewPauliX(), 0); err != nil {
			fmt.Printf("Error applying Pauli-X gate: %v\n", err)
			return
		}
```

In `internal/density/state_test.go`, update every `ApplySingleQubitGate(g, t)` call to `ApplyGate(g, t)` — find them with:

Run: `grep -n "ApplySingleQubitGate" internal/`  (recursive: `grep -rn "ApplySingleQubitGate" internal/`)

- [ ] **Step 2: Delete the method**

Remove the `ApplySingleQubitGate` method from `internal/density/state.go` (the block from its doc comment through its closing brace). `applySingleQubitOperator` stays — the Kraus channels and `ApplyGate`'s k=1 path use it.

- [ ] **Step 3: Verify nothing references it and tests pass**

Run: `grep -rn "ApplySingleQubitGate" . --include='*.go'`
Expected: no output.

Run: `go test ./... > /dev/null && echo OK`
Expected: OK.

- [ ] **Step 4: Write the demo**

Append to `internal/examples/noise.go` (add imports `math/cmplx`, `github.com/pjbaur/quantum/circuit`, `github.com/pjbaur/quantum/internal/density`, `github.com/pjbaur/quantum/state`):

```go
// NoisyBellDemo executes one Bell circuit on both a state-vector and a
// density-matrix backend — the same circuit.Execute call — then applies
// depolarizing noise to the density matrix. Populations stay correlated
// while off-diagonal coherence and purity decay: the mixed-state
// signature no state vector can represent.
func NoisyBellDemo() {
	fmt.Println("\n=== Noisy Bell Pair Demonstration ===")
	fmt.Println("One circuit (H, CNOT) executed unchanged on a state vector and a")
	fmt.Println("density matrix, then depolarizing noise on both qubits of the latter.")

	bell, err := circuit.New(2)
	if err != nil {
		fmt.Printf("Error creating circuit: %v\n", err)
		return
	}
	if err := bell.AddGate(gates.NewHadamard(), 0); err != nil {
		fmt.Printf("Error adding Hadamard: %v\n", err)
		return
	}
	if err := bell.AddGate(gates.NewCNOT(), 0, 1); err != nil {
		fmt.Printf("Error adding CNOT: %v\n", err)
		return
	}

	ideal, err := state.New(2)
	if err != nil {
		fmt.Printf("Error creating state: %v\n", err)
		return
	}
	if err := bell.Execute(ideal); err != nil {
		fmt.Printf("Error executing circuit on state vector: %v\n", err)
		return
	}
	fmt.Printf("\nIdeal state vector: P(00) = %.4f, P(11) = %.4f\n",
		ideal.Probability(0), ideal.Probability(3))

	for _, p := range []float64{0, 0.05, 0.2} {
		m, err := density.New(2)
		if err != nil {
			fmt.Printf("Error creating density matrix: %v\n", err)
			return
		}
		if err := bell.Execute(m); err != nil {
			fmt.Printf("Error executing circuit on density matrix: %v\n", err)
			return
		}
		for q := 0; q < 2; q++ {
			if err := m.ApplyDepolarizing(q, p); err != nil {
				fmt.Printf("Error applying depolarizing channel: %v\n", err)
				return
			}
		}
		fmt.Printf("\np = %.2f:\n", p)
		fmt.Printf("  P(00) = %.4f, P(11) = %.4f\n", m.Probability(0), m.Probability(3))
		fmt.Printf("  coherence |ρ(00,11)| = %.4f\n", cmplx.Abs(m.Element(0, 3)))
		fmt.Printf("  purity Tr(ρ²) = %.4f\n", m.Purity())
	}
	fmt.Println("\nNoise leaves the populations correlated while coherence and purity")
	fmt.Println("decay — only a density matrix can carry that mixed state.")
}
```

Wire it into `RunAllNoiseDemos` by adding one call after `DepolarizingDemo()`:

```go
	DephasingDemo()
	AmplitudeDampingDemo()
	DepolarizingDemo()
	NoisyBellDemo()
```

- [ ] **Step 5: Run the demo and full suite**

Run: `go run ./cmd/quantum noise` — or whichever subcommand `cmd/quantum/main.go:146` guards; check with `sed -n '140,150p' cmd/quantum/main.go` and use that invocation.
Expected: demo prints; p = 0 line shows purity 1.0000 and coherence 0.5000; p = 0.2 shows purity < 1 and coherence < 0.5; P(00) ≈ P(11) throughout.

Run: `go test ./... > /dev/null && echo OK`
Expected: OK.

- [ ] **Step 6: Vet, format, commit**

```bash
gofmt -l . && go vet ./... && echo OK
git add internal/examples/noise.go internal/density/state.go internal/density/state_test.go
git commit -m "feat(examples): noisy-Bell demo on density backend; drop ApplySingleQubitGate"
```

---

### Task 7: Documentation — ADR-0009, package doc, backlog, CHANGELOG

**Files:**
- Create: `docs/adr/0009-density-backend-quantumstate.md`
- Modify: `docs/adr/0007-density-backend-scope.md` (status line)
- Modify: `docs/adr/README.md` (index entry, if the file lists ADRs — check)
- Modify: `internal/density/state.go` (package doc comment)
- Modify: `docs/enhancement-backlog-2026-08-27.md` (item 9)
- Modify: `CHANGELOG.md` (Unreleased section)

**Interfaces:**
- Consumes: the shipped behavior of Tasks 1–6.
- Produces: documentation only.

- [ ] **Step 1: Write ADR-0009**

Create `docs/adr/0009-density-backend-quantumstate.md`:

```markdown
# ADR-0009: Density Backend Implements QuantumState

## Status

Accepted (2026-08-29). Supersedes the scope restriction in
[ADR-0007](0007-density-backend-scope.md); the noise-channel and
Bloch-vector analysis API decided there is retained unchanged.

## Context

ADR-0007 scoped `internal/density` to noise-and-analysis and deferred
`quantum.QuantumState` conformance until a concrete consumer needed to
run a full circuit against a density matrix interchangeably with the
state-vector backends. That consumer now exists: the noisy-Bell
demonstration executes one `circuit.Circuit` unchanged on a dense state
vector and on a density matrix, then applies depolarizing noise that
only the density representation can carry.

The interface is state-vector-shaped in two places a mixed state cannot
honor: `Amplitude(basisState) complex128` (no error return) and
`SetAmplitude`.

## Decision

`Matrix` implements `quantum.QuantumState` and
`quantum.BackendCapabilities`:

- `ApplyGate` computes ρ → UρU† for any gate width, reusing the shared
  `backendmath` kernel (U down columns, conj(U) along rows). No width
  cap: `SupportsGateQubits` is true for k ≥ 1, `MaxGateQubits` is 0.
  Cost O(4ⁿ·4ᵏ).
- `Measure` is projective measurement on ρ via the shared
  `PlanCollapse`: outcome from the diagonal, collapse ΠρΠ/p in place,
  randomness injectable via `SetRandSource`.
- `Probability(i)` is ρᵢᵢ, exact for pure and mixed states.
- `Amplitude(i)` reconstructs the state vector when ρ is pure (global
  phase fixed by making the first nonzero-probability basis amplitude
  real positive) and returns NaN when ρ is mixed. The NaN flows into
  the NaN-safe guards of `Sample`, `Expectation`, and `Fidelity`, which
  therefore refuse mixed states with `UnnormalizedStateError`.
- `SetAmplitude` always returns `UnsupportedOperationError` — a density
  matrix has no amplitude vector to write one entry of.
- `Matrix` deliberately does not implement `BulkAmplitudeSetter` (so
  `QFT` refuses it) or `Resetter` (no consumer).

`ApplySingleQubitGate` was removed as redundant with
`ApplyGate(g, target)`.

## Consequences

### Positive

- One circuit definition runs on dense, sparse, or density backends;
  noise channels compose with circuit execution.
- Measurement semantics on ρ share the collapse-planning code (and its
  zero-branch safety) with the other backends.
- Mixed states fail loudly, not wrongly, in amplitude-based helpers.

### Negative

- `Amplitude` on a mixed state returns NaN rather than an error — the
  interface signature allows nothing better; callers that skip the
  helpers' guards can propagate NaN.
- Density execution costs O(4ⁿ) memory and O(4ⁿ·4ᵏ) per gate, far
  beyond the state-vector backends; it remains a small-register tool.

## Decision Log

- 2026-08-29: Initial decision accepted
```

- [ ] **Step 2: Update ADR-0007 status**

In `docs/adr/0007-density-backend-scope.md`, replace the Status section body (`Accepted`) with:

```markdown
Accepted. The QuantumState-conformance restriction was later lifted by
[ADR-0009](0009-density-backend-quantumstate.md) when the noisy-Bell
demo became the concrete consumer this ADR's revisit trigger named; the
noise-channel and analysis API decided here is retained unchanged.
```

Check `docs/adr/README.md` for an index list; if it enumerates ADRs, add a line for 0009 matching the existing format.

- [ ] **Step 3: Rewrite the density package doc**

Replace the package comment in `internal/density/state.go`:

```go
// Package density implements a density-matrix backend. Matrix satisfies
// quantum.QuantumState (see docs/adr/0009-density-backend-quantumstate.md),
// so circuits execute on it interchangeably with the state-vector
// backends, and adds what only a density matrix can represent: Kraus
// noise channels (depolarizing, dephasing, amplitude damping),
// trace/purity, reduced single-qubit Bloch vectors, and FromState for
// bridging in a pure state. Mixed states have no amplitude vector, so
// Amplitude returns NaN once noise mixes the state and SetAmplitude is
// refused; cost is O(4ⁿ) in memory and O(4ⁿ·4ᵏ) per k-qubit gate.
package density
```

- [ ] **Step 4: Check off backlog item 9**

In `docs/enhancement-backlog-2026-08-27.md`, change item 9's `- [ ]` to `- [x]` and append a done-note quote following the pattern of items 1–8:

```markdown
  > **Done (2026-08-29)**: `Matrix` implements `quantum.QuantumState` and
  > `BackendCapabilities` (ADR-0009, superseding ADR-0007's restriction).
  > `ApplyGate` is generic-k ρ → UρU† through the shared backendmath
  > kernel; `Measure` is projective collapse via `PlanCollapse` with
  > injectable randomness; `Probability` reads the diagonal; `Amplitude`
  > reconstructs pure states and returns NaN for mixed ones, so
  > Sample/Expectation/Fidelity refuse mixed states through their
  > existing guards; `SetAmplitude` returns `UnsupportedOperationError`;
  > no `BulkAmplitudeSetter` (QFT refuses) or `Resetter`.
  > `ApplySingleQubitGate` removed. Consumer: `NoisyBellDemo` executes
  > one Bell circuit on dense and density backends via `circuit.Execute`
  > and applies depolarizing noise. Tests: dense-equality circuits
  > (k=1..3), forced measurements, entangled-partner collapse, helper
  > guards, QFT refusal, capability assertions.
```

- [ ] **Step 5: CHANGELOG entry**

In `CHANGELOG.md`, add under (or create) the `## Unreleased` section, above `## v0.3.0`:

```markdown
## Unreleased

### Added

#### Density-matrix backend implements `QuantumState`

`internal/density.Matrix` now satisfies `quantum.QuantumState` and
`quantum.BackendCapabilities` (ADR-0009): circuits execute on a density
matrix interchangeably with the state-vector backends, including
projective measurement with injectable randomness. Mixed states report
NaN amplitudes, so `Sample`/`Expectation`/`Fidelity` refuse them with
`UnnormalizedStateError`, and `SetAmplitude` returns
`UnsupportedOperationError`. New `NoisyBellDemo` in the noise demos runs
one Bell circuit on both backend families and applies depolarizing
noise.

### Breaking

#### `density.ApplySingleQubitGate` removed

Use `ApplyGate(gate, target)` — same behavior, interface-shaped.
```

If an `## Unreleased` section already exists, merge these subsections into it instead of creating a second one.

- [ ] **Step 6: Verify and commit**

```bash
go test ./... > /dev/null && go vet ./... && gofmt -l . && echo OK
git add docs/adr/0009-density-backend-quantumstate.md docs/adr/0007-density-backend-scope.md docs/adr/README.md internal/density/state.go docs/enhancement-backlog-2026-08-27.md CHANGELOG.md
git commit -m "docs: ADR-0009 density QuantumState conformance; close backlog item 9"
```

---

## Final Verification

- [ ] Full suite with race detector: `go test -race ./...` — PASS
- [ ] Fuzz seeds still pass: `go test ./... -run 'Fuzz' -v` (seed corpora run as tests) — PASS
- [ ] `go vet ./...` clean, `gofmt -l .` empty
- [ ] Demo output sane: run the noise demos via `cmd/quantum` and eyeball the noisy-Bell section
