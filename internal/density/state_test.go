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
