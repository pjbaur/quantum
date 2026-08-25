package state_test

import (
	"errors"
	"math"
	"math/cmplx"
	"testing"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/quantum"
	"github.com/pjbaur/quantum/state"
)

const tolerance = 1e-10

// The algorithm package selects a backend by this capability, so losing it
// here would silently take the dense backend out of that package's reach.
var _ quantum.BulkAmplitudeSetter = (*state.State)(nil)

func TestApplyGateCNOTControlBehavior(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(quantum.QuantumState) error
		targets   []int
		wantIndex int
	}{
		{
			name: "control |0> leaves |00> unchanged",
			setup: func(quantum.QuantumState) error {
				return nil
			},
			targets:   []int{1, 0},
			wantIndex: 0,
		},
		{
			name: "control |1> flips target",
			setup: func(qs quantum.QuantumState) error {
				return qs.ApplyGate(gates.NewPauliX(), 1)
			},
			targets:   []int{1, 0},
			wantIndex: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qs, err := state.New(2)
			if err != nil {
				t.Fatalf("state.New failed: %v", err)
			}
			if err := tt.setup(qs); err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			if err := qs.ApplyGate(gates.NewCNOT(), tt.targets...); err != nil {
				t.Fatalf("ApplyGate returned error: %v", err)
			}

			assertBasisState(t, qs, tt.wantIndex)
			assertNormalized(t, qs)
		})
	}
}

func TestApplyGateSwap(t *testing.T) {
	qs, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New failed: %v", err)
	}
	if err := qs.ApplyGate(gates.NewPauliX(), 0); err != nil {
		t.Fatalf("ApplyGate(PauliX) returned error: %v", err)
	}

	if err := qs.ApplyGate(gates.NewSwap(), 0, 1); err != nil {
		t.Fatalf("ApplyGate(SWAP) returned error: %v", err)
	}

	assertBasisState(t, qs, 2)
	assertNormalized(t, qs)
}

// TestApplyGateThreeQubitTargetOrdering exercises the gate width only this
// backend supports, with a gate whose result depends on how targets map onto
// the gate's basis ordering. The rest of the suite reaches three qubits only
// through an identity matrix, which any consistent (even wrong) mapping
// satisfies; a Toffoli distinguishes them, because moving the flipped pair to
// different qubits moves which basis state ends up occupied.
func TestApplyGateThreeQubitTargetOrdering(t *testing.T) {
	// Toffoli: the two most significant basis bits control a flip of the
	// least significant one, so it swaps basis 6 and 7 and fixes the rest.
	toffoli := make([][]complex128, 8)
	for row := range toffoli {
		toffoli[row] = make([]complex128, 8)
	}
	for row := 0; row < 6; row++ {
		toffoli[row][row] = 1
	}
	toffoli[6][7], toffoli[7][6] = 1, 1

	gate, err := gates.NewMatrixGate("Toffoli", toffoli)
	if err != nil {
		t.Fatalf("NewMatrixGate failed: %v", err)
	}

	tests := []struct {
		name      string
		targets   []int
		fromIndex int
		wantIndex int
	}{
		// targets[0] is the gate's most significant basis bit, so this
		// ordering lines the gate's basis up with the register's own:
		// qubits 2 and 1 control a flip of qubit 0, and |110⟩ becomes |111⟩.
		{name: "targets high to low", targets: []int{2, 1, 0}, fromIndex: 6, wantIndex: 7},
		// Reversed, qubits 0 and 1 control a flip of qubit 2 instead, so the
		// state that moves is the one with those two set: |011⟩ becomes |111⟩.
		{name: "targets low to high", targets: []int{0, 1, 2}, fromIndex: 3, wantIndex: 7},
		// A basis state missing a control is left alone either way.
		{name: "control not set", targets: []int{2, 1, 0}, fromIndex: 4, wantIndex: 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qs, err := state.New(3)
			if err != nil {
				t.Fatalf("state.New failed: %v", err)
			}
			amplitudes := make([]complex128, 8)
			amplitudes[tt.fromIndex] = 1
			if err := qs.SetAmplitudes(amplitudes); err != nil {
				t.Fatalf("SetAmplitudes failed: %v", err)
			}

			if err := qs.ApplyGate(gate, tt.targets...); err != nil {
				t.Fatalf("ApplyGate returned error: %v", err)
			}

			assertBasisState(t, qs, tt.wantIndex)
			assertNormalized(t, qs)
		})
	}
}

func TestApplyGateBellStates(t *testing.T) {
	invSqrt2 := 1 / math.Sqrt(2)
	tests := []struct {
		name      string
		postGates []struct {
			gate   quantum.Gate
			target int
		}
		want [4]complex128
	}{
		{
			name: "Phi+",
			want: [4]complex128{
				complex(invSqrt2, 0),
				0,
				0,
				complex(invSqrt2, 0),
			},
		},
		{
			name: "Phi-",
			postGates: []struct {
				gate   quantum.Gate
				target int
			}{
				{gate: gates.NewPauliZ(), target: 1},
			},
			want: [4]complex128{
				complex(invSqrt2, 0),
				0,
				0,
				complex(-invSqrt2, 0),
			},
		},
		{
			name: "Psi+",
			postGates: []struct {
				gate   quantum.Gate
				target int
			}{
				{gate: gates.NewPauliX(), target: 1},
			},
			want: [4]complex128{
				0,
				complex(invSqrt2, 0),
				complex(invSqrt2, 0),
				0,
			},
		},
		{
			name: "Psi-",
			postGates: []struct {
				gate   quantum.Gate
				target int
			}{
				{gate: gates.NewPauliX(), target: 1},
				{gate: gates.NewPauliZ(), target: 1},
			},
			want: [4]complex128{
				0,
				complex(invSqrt2, 0),
				complex(-invSqrt2, 0),
				0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qs, err := state.New(2)
			if err != nil {
				t.Fatalf("state.New failed: %v", err)
			}
			if err := qs.ApplyGate(gates.NewHadamard(), 0); err != nil {
				t.Fatalf("ApplyGate(Hadamard) returned error: %v", err)
			}
			if err := qs.ApplyGate(gates.NewCNOT(), 0, 1); err != nil {
				t.Fatalf("ApplyGate(CNOT) returned error: %v", err)
			}

			for _, step := range tt.postGates {
				if err := qs.ApplyGate(step.gate, step.target); err != nil {
					t.Fatalf("ApplyGate(%s) returned error: %v", step.gate.Name(), err)
				}
			}

			for i, want := range tt.want {
				assertAmplitude(t, qs, i, want)
			}

			for i := 0; i < len(tt.want); i++ {
				wantProb := cmplx.Abs(tt.want[i])
				wantProb *= wantProb
				assertProbability(t, qs, i, wantProb)
			}
			assertNormalized(t, qs)
		})
	}
}

func TestApplyGateErrors(t *testing.T) {
	tests := []struct {
		name     string
		gate     quantum.Gate
		targets  []int
		errCheck func(error) bool
	}{
		{
			name:    "target out of range",
			gate:    gates.NewHadamard(),
			targets: []int{2},
			errCheck: func(err error) bool {
				var targetErr *quantum.QubitsOutOfRangeError
				return errors.As(err, &targetErr)
			},
		},
		{
			name:    "invalid target count for CNOT",
			gate:    gates.NewCNOT(),
			targets: []int{0},
			errCheck: func(err error) bool {
				var targetErr *quantum.InvalidGateApplicationError
				if !errors.As(err, &targetErr) {
					return false
				}
				return targetErr.RequiredLen == 2 && targetErr.ActualLen == 1
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qs, err := state.New(2)
			if err != nil {
				t.Fatalf("state.New failed: %v", err)
			}
			err = qs.ApplyGate(tt.gate, tt.targets...)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.errCheck(err) {
				t.Fatalf("unexpected error type: %T", err)
			}
		})
	}
}

func assertBasisState(t *testing.T, qs quantum.QuantumState, wantIndex int) {
	t.Helper()

	totalStates := 1 << qs.NumQubits()
	for i := 0; i < totalStates; i++ {
		amp := qs.Amplitude(i)
		if i == wantIndex {
			if cmplx.Abs(amp-1) > tolerance {
				t.Fatalf("expected basis state %d amplitude 1, got %v", wantIndex, amp)
			}
			continue
		}
		if cmplx.Abs(amp) > tolerance {
			t.Fatalf("expected basis state %d amplitude 0, got %v", i, amp)
		}
	}
}

func assertAmplitude(t *testing.T, qs quantum.QuantumState, index int, want complex128) {
	t.Helper()

	amp := qs.Amplitude(index)
	if cmplx.Abs(amp-want) > tolerance {
		t.Fatalf("expected amplitude %v at index %d, got %v", want, index, amp)
	}
}

func assertProbability(t *testing.T, qs quantum.QuantumState, index int, want float64) {
	t.Helper()

	prob := qs.Probability(index)
	if math.Abs(prob-want) > tolerance {
		t.Fatalf("expected probability %v at index %d, got %v", want, index, prob)
	}
}

func assertNormalized(t *testing.T, qs quantum.QuantumState) {
	t.Helper()

	sum := 0.0
	totalStates := 1 << qs.NumQubits()
	for i := 0; i < totalStates; i++ {
		sum += qs.Probability(i)
	}
	if math.Abs(sum-1.0) > tolerance {
		t.Fatalf("expected normalized state, probability sum = %v", sum)
	}
}

// nonFiniteAmplitudeCases are amplitudes no state may hold. The
// normalization check cannot catch them on its own: |NaN|² is NaN, and NaN
// fails every comparison against the tolerance, so a NaN amplitude passes
// as normalized and poisons everything computed from the state afterwards.
var nonFiniteAmplitudeCases = []struct {
	name  string
	value complex128
}{
	{"NaN real part", complex(math.NaN(), 0)},
	{"NaN imaginary part", complex(0, math.NaN())},
	{"positive infinity", complex(math.Inf(1), 0)},
	{"negative infinity", complex(0, math.Inf(-1))},
	{"infinity and NaN together", complex(math.Inf(1), math.NaN())},
}

func TestSetAmplitudeRejectsNonFinite(t *testing.T) {
	for _, tt := range nonFiniteAmplitudeCases {
		t.Run(tt.name, func(t *testing.T) {
			s, err := state.New(1)
			if err != nil {
				t.Fatalf("state.New(1) failed: %v", err)
			}

			err = s.SetAmplitude(0, tt.value)

			var nonFinite *quantum.NonFiniteAmplitudeError
			if !errors.As(err, &nonFinite) {
				t.Fatalf("SetAmplitude(0, %v) = %v (%T), want *NonFiniteAmplitudeError", tt.value, err, err)
			}
			if nonFinite.BasisState != 0 {
				t.Errorf("reported basis state %d, want 0", nonFinite.BasisState)
			}
			assertBasisState(t, s, 0)
		})
	}
}

func TestSetAmplitudesRejectsNonFinite(t *testing.T) {
	for _, tt := range nonFiniteAmplitudeCases {
		t.Run(tt.name, func(t *testing.T) {
			s, err := state.New(1)
			if err != nil {
				t.Fatalf("state.New(1) failed: %v", err)
			}

			// The finite half alone carries the whole probability, so only
			// the poisoned amplitude can be what makes this invalid.
			err = s.SetAmplitudes([]complex128{1, tt.value})

			var nonFinite *quantum.NonFiniteAmplitudeError
			if !errors.As(err, &nonFinite) {
				t.Fatalf("SetAmplitudes with %v = %v (%T), want *NonFiniteAmplitudeError", tt.value, err, err)
			}
			if nonFinite.BasisState != 1 {
				t.Errorf("reported basis state %d, want 1", nonFinite.BasisState)
			}
			assertBasisState(t, s, 0)
		})
	}
}

func TestDenseBackendCapabilities(t *testing.T) {
	s, err := state.New(3)
	if err != nil {
		t.Fatalf("state.New failed: %v", err)
	}

	// Test SupportsGateQubits - dense backend supports all gate sizes
	if !s.SupportsGateQubits(1) {
		t.Error("expected dense backend to support 1-qubit gates")
	}
	if !s.SupportsGateQubits(2) {
		t.Error("expected dense backend to support 2-qubit gates")
	}
	if !s.SupportsGateQubits(3) {
		t.Error("expected dense backend to support 3-qubit gates")
	}
	if !s.SupportsGateQubits(4) {
		t.Error("expected dense backend to support 4-qubit gates")
	}
	if s.SupportsGateQubits(0) {
		t.Error("expected dense backend to NOT support 0-qubit gates")
	}
	if s.SupportsGateQubits(-1) {
		t.Error("expected dense backend to NOT support negative qubit counts")
	}

	// Test MaxGateQubits - 0 means no limit
	if max := s.MaxGateQubits(); max != 0 {
		t.Errorf("expected MaxGateQubits=0 (no limit), got %d", max)
	}
}

func TestDenseGateInterfaceAssertion(t *testing.T) {
	// Verify State implements both QuantumState and BackendCapabilities
	s1, _ := state.New(1)
	var _ quantum.QuantumState = s1
	var _ quantum.BackendCapabilities = s1
}

func TestNewInvalidQubitCount(t *testing.T) {
	tests := []struct {
		name       string
		numQubits  int
		errCheck   func(error) bool
		errMessage string
	}{
		{
			name:      "zero qubits",
			numQubits: 0,
			errCheck: func(err error) bool {
				var targetErr *quantum.InvalidQubitCountError
				return errors.As(err, &targetErr) &&
					targetErr.Requested == 0 &&
					targetErr.Reason == "must be positive"
			},
			errMessage: "should return InvalidQubitCountError for 0 qubits",
		},
		{
			name:      "negative qubits",
			numQubits: -1,
			errCheck: func(err error) bool {
				var targetErr *quantum.InvalidQubitCountError
				return errors.As(err, &targetErr) &&
					targetErr.Requested == -1 &&
					targetErr.Reason == "must be positive"
			},
			errMessage: "should return InvalidQubitCountError for -1 qubits",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := state.New(tt.numQubits)
			if err == nil {
				t.Fatalf("expected error for %d qubits, got nil", tt.numQubits)
			}
			if !tt.errCheck(err) {
				t.Fatalf("%s, got: %T: %v", tt.errMessage, err, err)
			}
		})
	}
}

func TestNewValidQubitCount(t *testing.T) {
	s, err := state.New(3)
	if err != nil {
		t.Fatalf("expected no error for 3 qubits, got: %v", err)
	}
	if s.NumQubits() != 3 {
		t.Fatalf("expected 3 qubits, got %d", s.NumQubits())
	}
}

// stubRandSource returns a fixed sequence of values, cycling.
type stubRandSource struct {
	values []float64
	index  int
}

func (s *stubRandSource) Float64() float64 {
	v := s.values[s.index%len(s.values)]
	s.index++
	return v
}

func plusState(t *testing.T) *state.State {
	t.Helper()
	s, err := state.New(1)
	if err != nil {
		t.Fatalf("state.New(1) failed: %v", err)
	}
	amp := complex(1/math.Sqrt2, 0)
	if err := s.SetAmplitudes([]complex128{amp, amp}); err != nil {
		t.Fatalf("SetAmplitudes failed: %v", err)
	}
	return s
}

func TestMeasureUsesInjectedRandSource(t *testing.T) {
	// Measure returns 1 when the draw is >= prob0 (0.5 for |+⟩).
	// prob0 for |+⟩ is 0.5 up to floating point, so draws well away
	// from the boundary give deterministic outcomes.
	cases := []struct {
		draw float64
		want int
	}{
		{0.7, 1},
		{0.3, 0},
	}
	for _, tc := range cases {
		s := plusState(t)
		s.SetRandSource(&stubRandSource{values: []float64{tc.draw}})
		got, err := s.Measure(0)
		if err != nil {
			t.Fatalf("Measure failed: %v", err)
		}
		if got != tc.want {
			t.Errorf("draw %.1f: Measure = %d, want %d", tc.draw, got, tc.want)
		}
	}
}

func TestSetRandSourceNilRestoresDefault(t *testing.T) {
	s := plusState(t)
	s.SetRandSource(&stubRandSource{values: []float64{0.7}})
	s.SetRandSource(nil)
	got, err := s.Measure(0)
	if err != nil {
		t.Fatalf("Measure failed: %v", err)
	}
	if got != 0 && got != 1 {
		t.Errorf("Measure with default source = %d, want 0 or 1", got)
	}
}

func TestCloneInheritsRandSource(t *testing.T) {
	src := &stubRandSource{values: []float64{0.7, 0.3}}
	s := plusState(t)
	s.SetRandSource(src)
	clone := s.Clone()

	got, err := s.Measure(0)
	if err != nil {
		t.Fatalf("Measure failed: %v", err)
	}
	if got != 1 {
		t.Errorf("original consumed draw 0.7: Measure = %d, want 1", got)
	}

	got, err = clone.Measure(0)
	if err != nil {
		t.Fatalf("clone Measure failed: %v", err)
	}
	if got != 0 {
		t.Errorf("clone consumed draw 0.3: Measure = %d, want 0", got)
	}
}

// TestMeasureZeroProbabilityBranch drives Measure into the outcome that
// carries no probability. The |1⟩ amplitude is small enough that squaring it
// underflows to exactly zero, so the state passes the normalization check
// while prob0 lands just short of 1 — close enough that the largest draw
// rand.Float64 can return selects outcome 1 anyway. Without the guard the
// collapse divides by that branch's zero normalization factor and leaves
// every amplitude NaN or infinite.
func TestMeasureZeroProbabilityBranch(t *testing.T) {
	s, err := state.New(1)
	if err != nil {
		t.Fatalf("state.New(1) failed: %v", err)
	}
	if err := s.SetAmplitudes([]complex128{complex(1-1e-16, 0), complex(1e-200, 0)}); err != nil {
		t.Fatalf("SetAmplitudes failed: %v", err)
	}
	s.SetRandSource(&stubRandSource{values: []float64{math.Nextafter(1, 0)}})

	got, err := s.Measure(0)
	if err != nil {
		t.Fatalf("Measure failed: %v", err)
	}
	if got != 0 {
		t.Errorf("Measure = %d, want 0 (the only outcome holding probability)", got)
	}

	for i := 0; i < 2; i++ {
		if amp := s.Amplitude(i); cmplx.IsNaN(amp) || cmplx.IsInf(amp) {
			t.Errorf("amplitude %d after collapse = %v, want a finite value", i, amp)
		}
	}
	if sum := s.Probability(0) + s.Probability(1); math.Abs(sum-1.0) > tolerance {
		t.Errorf("probability sum after collapse = %v, want 1.0", sum)
	}
}
