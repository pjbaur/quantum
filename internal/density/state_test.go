package density

import (
	"math"
	"math/cmplx"
	"testing"

	"github.com/pjbaur/quantum/gates"
)

const tol = 1e-12

func TestApplySingleQubitGateHadamard(t *testing.T) {
	state := newFromAmplitudes(1, 0)
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
	state := newFromAmplitudes(1, 0)
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
	state := newFromAmplitudes(complex(amp, 0), complex(amp, 0))
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
	state := newFromAmplitudes(0, 1)
	gamma := 0.4
	if err := state.ApplyAmplitudeDamping(0, gamma); err != nil {
		t.Fatalf("ApplyAmplitudeDamping failed: %v", err)
	}

	assertCloseComplex(t, state.Element(0, 0), complex(gamma, 0))
	assertCloseComplex(t, state.Element(1, 1), complex(1-gamma, 0))
	assertCloseFloat(t, state.Trace(), 1)
	assertPositiveSemidefinite2x2(t, state)
}

func newFromAmplitudes(alpha, beta complex128) *Matrix {
	state := New(1)
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
