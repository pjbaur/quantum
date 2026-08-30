package density

import (
	"errors"
	"math"
	"math/cmplx"
	"testing"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/quantum"
)

const tol = 1e-12

func TestNewRejectsNonPositiveQubitCount(t *testing.T) {
	for _, n := range []int{0, -3} {
		state, err := New(n)
		if state != nil {
			t.Errorf("New(%d) returned non-nil matrix", n)
		}
		var invalidErr *quantum.InvalidQubitCountError
		if !errors.As(err, &invalidErr) {
			t.Fatalf("New(%d) error = %v, want *quantum.InvalidQubitCountError", n, err)
		}
		if invalidErr.Requested != n {
			t.Errorf("New(%d) error Requested = %d, want %d", n, invalidErr.Requested, n)
		}
	}
}

func TestNewValidCount(t *testing.T) {
	state, err := New(2)
	if err != nil {
		t.Fatalf("New(2) failed: %v", err)
	}
	if state.NumQubits() != 2 {
		t.Errorf("NumQubits() = %d, want 2", state.NumQubits())
	}
	assertCloseComplex(t, state.Element(0, 0), 1)
	assertCloseFloat(t, state.Trace(), 1)
}

func TestPurityPureState(t *testing.T) {
	state := newFromAmplitudes(t, 1, 0)
	assertCloseFloat(t, state.Purity(), 1)
}

func TestPurityMaximallyMixed(t *testing.T) {
	amp := complex(1/math.Sqrt2, 0)
	state := newFromAmplitudes(t, amp, amp)
	if err := state.ApplyDephasing(0, 0.5); err != nil {
		t.Fatalf("ApplyDephasing failed: %v", err)
	}
	assertCloseFloat(t, state.Purity(), 0.5)
}

func TestReducedBlochVectorPureStates(t *testing.T) {
	amp := complex(1/math.Sqrt2, 0)
	cases := []struct {
		name        string
		alpha, beta complex128
		x, y, z     float64
	}{
		{"zero", 1, 0, 0, 0, 1},
		{"one", 0, 1, 0, 0, -1},
		{"plus", amp, amp, 1, 0, 0},
		{"plus-i", amp, complex(0, 1/math.Sqrt2), 0, 1, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := newFromAmplitudes(t, tc.alpha, tc.beta)
			x, y, z, err := state.ReducedBlochVector(0)
			if err != nil {
				t.Fatalf("ReducedBlochVector failed: %v", err)
			}
			assertCloseFloat(t, x, tc.x)
			assertCloseFloat(t, y, tc.y)
			assertCloseFloat(t, z, tc.z)
		})
	}
}

func TestReducedBlochVectorShrinksUnderDephasing(t *testing.T) {
	amp := complex(1/math.Sqrt2, 0)
	p := 0.3
	state := newFromAmplitudes(t, amp, amp)
	if err := state.ApplyDephasing(0, p); err != nil {
		t.Fatalf("ApplyDephasing failed: %v", err)
	}
	x, y, z, err := state.ReducedBlochVector(0)
	if err != nil {
		t.Fatalf("ReducedBlochVector failed: %v", err)
	}
	assertCloseFloat(t, x, 1-2*p)
	assertCloseFloat(t, y, 0)
	assertCloseFloat(t, z, 0)
}

func TestReducedBlochVectorTwoQubit(t *testing.T) {
	state, err := New(2)
	if err != nil {
		t.Fatalf("New(2) failed: %v", err)
	}
	if err := state.ApplySingleQubitGate(gates.NewHadamard(), 0); err != nil {
		t.Fatalf("ApplySingleQubitGate failed: %v", err)
	}

	x, y, z, err := state.ReducedBlochVector(0)
	if err != nil {
		t.Fatalf("ReducedBlochVector(0) failed: %v", err)
	}
	assertCloseFloat(t, x, 1)
	assertCloseFloat(t, y, 0)
	assertCloseFloat(t, z, 0)

	x, y, z, err = state.ReducedBlochVector(1)
	if err != nil {
		t.Fatalf("ReducedBlochVector(1) failed: %v", err)
	}
	assertCloseFloat(t, x, 0)
	assertCloseFloat(t, y, 0)
	assertCloseFloat(t, z, 1)
}

func TestReducedBlochVectorRejectsOutOfRangeTarget(t *testing.T) {
	state := newFromAmplitudes(t, 1, 0)
	for _, target := range []int{-1, 1} {
		_, _, _, err := state.ReducedBlochVector(target)
		var rangeErr *quantum.QubitsOutOfRangeError
		if !errors.As(err, &rangeErr) {
			t.Errorf("ReducedBlochVector(%d) error = %v, want *quantum.QubitsOutOfRangeError", target, err)
		}
	}
}

func TestApplySingleQubitGateHadamard(t *testing.T) {
	state := newFromAmplitudes(t, 1, 0)
	if err := state.ApplySingleQubitGate(gates.NewHadamard(), 0); err != nil {
		t.Fatalf("ApplySingleQubitGate failed: %v", err)
	}

	assertCloseComplex(t, state.Element(0, 0), 0.5)
	assertCloseComplex(t, state.Element(0, 1), 0.5)
	assertCloseComplex(t, state.Element(1, 0), 0.5)
	assertCloseComplex(t, state.Element(1, 1), 0.5)
	assertPositiveSemidefinite2x2(t, state)
}

func TestDepolarizingChannel(t *testing.T) {
	state := newFromAmplitudes(t, 1, 0)
	p := 0.3
	if err := state.ApplyDepolarizing(0, p); err != nil {
		t.Fatalf("ApplyDepolarizing failed: %v", err)
	}

	expected00 := 1 - 2*p/3
	expected11 := 2 * p / 3

	assertCloseComplex(t, state.Element(0, 0), complex(expected00, 0))
	assertCloseComplex(t, state.Element(1, 1), complex(expected11, 0))
	assertCloseFloat(t, state.Trace(), 1)
	assertPositiveSemidefinite2x2(t, state)
}

func TestDephasingChannel(t *testing.T) {
	amp := 1 / math.Sqrt2
	state := newFromAmplitudes(t, complex(amp, 0), complex(amp, 0))
	p := 0.25
	if err := state.ApplyDephasing(0, p); err != nil {
		t.Fatalf("ApplyDephasing failed: %v", err)
	}

	expectedOffDiag := (1 - 2*p) / 2

	assertCloseComplex(t, state.Element(0, 0), 0.5)
	assertCloseComplex(t, state.Element(1, 1), 0.5)
	assertCloseComplex(t, state.Element(0, 1), complex(expectedOffDiag, 0))
	assertCloseComplex(t, state.Element(1, 0), complex(expectedOffDiag, 0))
	assertPositiveSemidefinite2x2(t, state)
}

func TestAmplitudeDampingChannel(t *testing.T) {
	state := newFromAmplitudes(t, 0, 1)
	gamma := 0.4
	if err := state.ApplyAmplitudeDamping(0, gamma); err != nil {
		t.Fatalf("ApplyAmplitudeDamping failed: %v", err)
	}

	assertCloseComplex(t, state.Element(0, 0), complex(gamma, 0))
	assertCloseComplex(t, state.Element(1, 1), complex(1-gamma, 0))
	assertCloseFloat(t, state.Trace(), 1)
	assertPositiveSemidefinite2x2(t, state)
}

func TestFromStateBuildsOuterProduct(t *testing.T) {
	invSqrt2 := 1 / math.Sqrt2

	cases := []struct {
		name       string
		numQubits  int
		amplitudes []complex128
		expected   [][]complex128
	}{
		{
			name:       "single_qubit_zero",
			numQubits:  1,
			amplitudes: []complex128{1, 0},
			expected: [][]complex128{
				{1, 0},
				{0, 0},
			},
		},
		{
			name:       "single_qubit_plus_i",
			numQubits:  1,
			amplitudes: []complex128{complex(invSqrt2, 0), complex(0, invSqrt2)},
			expected: [][]complex128{
				{0.5, complex(0, -0.5)},
				{complex(0, 0.5), 0.5},
			},
		},
		{
			name:       "bell_pair",
			numQubits:  2,
			amplitudes: []complex128{complex(invSqrt2, 0), 0, 0, complex(invSqrt2, 0)},
			expected: [][]complex128{
				{0.5, 0, 0, 0.5},
				{0, 0, 0, 0},
				{0, 0, 0, 0},
				{0.5, 0, 0, 0.5},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state, err := FromState(&stubState{numQubits: tc.numQubits, amplitudes: tc.amplitudes})
			if err != nil {
				t.Fatalf("FromState failed: %v", err)
			}
			if state.NumQubits() != tc.numQubits {
				t.Errorf("NumQubits() = %d, want %d", state.NumQubits(), tc.numQubits)
			}

			for row := range tc.expected {
				for col := range tc.expected[row] {
					if cmplx.Abs(state.Element(row, col)-tc.expected[row][col]) > tol {
						t.Errorf("Element(%d, %d) = %v, want %v",
							row, col, state.Element(row, col), tc.expected[row][col])
					}
				}
			}

			assertCloseFloat(t, state.Trace(), 1)
			assertCloseFloat(t, state.Purity(), 1)
		})
	}
}

func TestFromStateRejectsNilState(t *testing.T) {
	state, err := FromState(nil)
	if state != nil {
		t.Errorf("FromState(nil) returned non-nil matrix")
	}
	if err == nil {
		t.Fatal("FromState(nil) error = nil, want an error")
	}
}

func TestFromStateRejectsTypedNilState(t *testing.T) {
	// A nil *stubState boxed in the quantum.QuantumState interface is not
	// == nil (the interface carries a concrete type), so FromState must
	// detect it via reflection rather than a plain nil comparison. Calling
	// NumQubits() on the nil receiver would otherwise panic.
	var typedNil *stubState

	got, err := FromState(typedNil)
	if got != nil {
		t.Errorf("FromState(typed nil) returned non-nil matrix")
	}
	if err == nil {
		t.Fatal("FromState(typed nil) error = nil, want an error")
	}
}

func TestFromStateRejectsNonPositiveQubitCount(t *testing.T) {
	for _, n := range []int{0, -2} {
		state, err := FromState(&stubState{numQubits: n})
		if state != nil {
			t.Errorf("FromState(%d qubits) returned non-nil matrix", n)
		}
		var invalidErr *quantum.InvalidQubitCountError
		if !errors.As(err, &invalidErr) {
			t.Fatalf("FromState(%d qubits) error = %v, want *quantum.InvalidQubitCountError", n, err)
		}
		if invalidErr.Requested != n {
			t.Errorf("FromState(%d qubits) error Requested = %d, want %d", n, invalidErr.Requested, n)
		}
	}
}

// stubState is a minimal quantum.QuantumState that hands FromState a fixed
// amplitude vector, including combinations no real backend can produce, such
// as a non-positive qubit count.
type stubState struct {
	numQubits  int
	amplitudes []complex128
}

func (s *stubState) NumQubits() int { return s.numQubits }

func (s *stubState) Amplitude(basisState int) complex128 {
	if basisState < 0 || basisState >= len(s.amplitudes) {
		return 0
	}
	return s.amplitudes[basisState]
}

func (s *stubState) SetAmplitude(int, complex128) error { return errors.New("not supported") }

func (s *stubState) ApplyGate(quantum.Gate, ...int) error { return errors.New("not supported") }

func (s *stubState) Measure(int) (int, error) { return 0, errors.New("not supported") }

func (s *stubState) Probability(basisState int) float64 {
	return quantum.Probability(s.Amplitude(basisState))
}

func (s *stubState) Clone() quantum.QuantumState {
	clone := &stubState{numQubits: s.numQubits}
	clone.amplitudes = append(clone.amplitudes, s.amplitudes...)
	return clone
}

func newFromAmplitudes(t *testing.T, alpha, beta complex128) *Matrix {
	t.Helper()
	state, err := New(1)
	if err != nil {
		t.Fatalf("New(1) failed: %v", err)
	}
	state.data[0] = alpha * cmplx.Conj(alpha)
	state.data[1] = alpha * cmplx.Conj(beta)
	state.data[2] = beta * cmplx.Conj(alpha)
	state.data[3] = beta * cmplx.Conj(beta)
	return state
}

func assertCloseComplex(t *testing.T, actual, expected complex128) {
	t.Helper()
	if cmplx.Abs(actual-expected) > tol {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}

func assertCloseFloat(t *testing.T, actual, expected float64) {
	t.Helper()
	if math.Abs(actual-expected) > tol {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}

func assertPositiveSemidefinite2x2(t *testing.T, state *Matrix) {
	t.Helper()
	r00 := state.Element(0, 0)
	r01 := state.Element(0, 1)
	r10 := state.Element(1, 0)
	r11 := state.Element(1, 1)

	if cmplx.Abs(r01-cmplx.Conj(r10)) > 1e-10 {
		t.Fatalf("matrix is not Hermitian: %v vs %v", r01, r10)
	}

	det := real(r00*r11 - r01*r10)
	if det < -1e-10 {
		t.Fatalf("matrix is not positive semidefinite, determinant %f", det)
	}
	if real(r00) < -1e-10 || real(r11) < -1e-10 {
		t.Fatalf("matrix has negative diagonal entries: %v, %v", r00, r11)
	}
}

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

	c := m.Clone()

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
