package gates

import (
	"fmt"
	"math"
	"math/cmplx"
	"testing"
)

// identityTolerance is the slack allowed when a matrix identity is exact in
// exact arithmetic and only misses by the rounding in math.Sincos. Every
// identity asserted below is of that kind, so the bound stays near the double
// epsilon rather than being widened to whatever the test happens to produce.
const identityTolerance = 1e-15

// composedTolerance covers the same identities after a matrix product, where
// each entry is a sum of rounded products. It is a handful of double epsilons
// (2.2e-16) of headroom for that sum, not a bound tuned to what this platform
// happens to produce: the assertions below pass at 1e-15 here.
const composedTolerance = 1e-14

// TestBuiltinGateGoldenValues locks the name and exact matrix of every
// built-in gate to the values the concrete types produced before the
// data-driven rewrite.
func TestBuiltinGateGoldenValues(t *testing.T) {
	cases := []struct {
		gate   *MatrixGate
		name   string
		matrix [][]complex128
	}{
		{NewHadamard(), "Hadamard", [][]complex128{
			{1.0 / complex(math.Sqrt(2), 0), 1.0 / complex(math.Sqrt(2), 0)},
			{1.0 / complex(math.Sqrt(2), 0), -1.0 / complex(math.Sqrt(2), 0)},
		}},
		{NewPauliX(), "PauliX", [][]complex128{
			{0, 1},
			{1, 0},
		}},
		{NewPauliY(), "PauliY", [][]complex128{
			{0, complex(0, -1)},
			{complex(0, 1), 0},
		}},
		{NewPauliZ(), "PauliZ", [][]complex128{
			{1, 0},
			{0, -1},
		}},
		{NewS(), "S", [][]complex128{
			{1, 0},
			{0, complex(0, 1)},
		}},
		{NewT(), "T", [][]complex128{
			{1, 0},
			{0, complex(math.Cos(math.Pi/4), math.Sin(math.Pi/4))},
		}},
		{NewCNOT(), "CNOT", [][]complex128{
			{1, 0, 0, 0},
			{0, 1, 0, 0},
			{0, 0, 0, 1},
			{0, 0, 1, 0},
		}},
		{NewSwap(), "SWAP", [][]complex128{
			{1, 0, 0, 0},
			{0, 0, 1, 0},
			{0, 1, 0, 0},
			{0, 0, 0, 1},
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.gate.Name() != tc.name {
				t.Errorf("Name() = %q, want %q", tc.gate.Name(), tc.name)
			}
			got := tc.gate.Matrix()
			if len(got) != len(tc.matrix) {
				t.Fatalf("matrix size %d, want %d", len(got), len(tc.matrix))
			}
			for i := range tc.matrix {
				for j := range tc.matrix[i] {
					// Exact comparison on purpose: values must stay
					// bit-identical to the pre-rewrite matrices.
					if got[i][j] != tc.matrix[i][j] {
						t.Errorf("matrix[%d][%d] = %v, want %v", i, j, got[i][j], tc.matrix[i][j])
					}
				}
			}
		})
	}
}

// TestMustGatePanicsOnInvalidTable documents the built-in-table safety net:
// an invalid matrix in a built-in definition is a programmer error and must
// panic at construction rather than produce a half-valid gate.
func TestMustGatePanicsOnInvalidTable(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("mustGate with a 1x1 matrix did not panic")
		}
	}()
	mustGate("Broken", [][]complex128{{1}})
}

// TestToffoliPermutesBasisStates pins the whole 8x8 table at once: each input
// basis state maps to exactly one output basis state (a permutation matrix,
// entries 0 and 1 only), and the state that moves is the target flip of the
// input, taken only when both control bits are set.
func TestToffoliPermutesBasisStates(t *testing.T) {
	const (
		controls = 0b110 // the two most significant basis bits
		target   = 0b001
	)

	matrix := NewToffoli().Matrix()
	if len(matrix) != 8 {
		t.Fatalf("Toffoli matrix is %dx%d, want 8x8", len(matrix), len(matrix))
	}

	for col := 0; col < 8; col++ {
		image := col
		if col&controls == controls {
			image ^= target
		}
		for row := 0; row < 8; row++ {
			want := complex128(0)
			if row == image {
				want = 1
			}
			// Exact comparison: a permutation matrix holds no rounded values.
			if matrix[row][col] != want {
				t.Errorf("matrix[%d][%d] = %v, want %v (|%03b⟩ maps to |%03b⟩)",
					row, col, matrix[row][col], want, col, image)
			}
		}
	}
}

// TestRotationGatesMatchPauliUpToGlobalPhase checks the defining half-angle
// property of the rotation gates: a π rotation is the corresponding Pauli
// matrix times e^{-iπ/2} = -i. The phase itself is asserted, not tolerated,
// so a rotation built with the wrong sign or a full-angle exponent fails.
func TestRotationGatesMatchPauliUpToGlobalPhase(t *testing.T) {
	wantPhase := complex(0, -1)

	cases := []struct {
		name     string
		rotation *MatrixGate
		pauli    *MatrixGate
	}{
		{"Rx(pi) is PauliX", NewRx(math.Pi), NewPauliX()},
		{"Ry(pi) is PauliY", NewRy(math.Pi), NewPauliY()},
		{"Rz(pi) is PauliZ", NewRz(math.Pi), NewPauliZ()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			phase, ok := globalPhase(tc.rotation.Matrix(), tc.pauli.Matrix(), identityTolerance)
			if !ok {
				t.Fatalf("%s is not %s times any global phase (best phase %v):\n%v",
					tc.rotation.Name(), tc.pauli.Name(), phase, tc.rotation.Matrix())
			}
			if cmplx.Abs(phase-wantPhase) > identityTolerance {
				t.Errorf("global phase = %v, want %v", phase, wantPhase)
			}
		})
	}
}

// TestRzMatchesPhaseUpToGlobalPhase checks the other relationship the two
// diagonal families have: Rz splits the phase evenly between the |0⟩ and |1⟩
// branches while Phase puts all of it on |1⟩, so Rz(θ) = e^{-iθ/2}·Phase(θ).
func TestRzMatchesPhaseUpToGlobalPhase(t *testing.T) {
	for _, theta := range []float64{0, math.Pi / 4, math.Pi / 2, math.Pi, 2 * math.Pi, -math.Pi / 3, 1.234} {
		t.Run(fmt.Sprintf("theta=%g", theta), func(t *testing.T) {
			phase, ok := globalPhase(NewRz(theta).Matrix(), NewPhase(theta).Matrix(), identityTolerance)
			if !ok {
				t.Fatalf("Rz(%g) is not Phase(%g) times any global phase (best phase %v)", theta, theta, phase)
			}
			sin, cos := math.Sincos(theta / 2)
			want := complex(cos, -sin)
			if cmplx.Abs(phase-want) > identityTolerance {
				t.Errorf("global phase = %v, want e^{-i·%g/2} = %v", phase, theta, want)
			}
		})
	}
}

// TestPhaseGateMatchesFixedPhaseGates checks that the parameterized phase gate
// reproduces the three fixed ones exactly, up to the rounding in Sincos:
// these are equalities, not equalities up to a global phase.
func TestPhaseGateMatchesFixedPhaseGates(t *testing.T) {
	cases := []struct {
		name  string
		phase *MatrixGate
		fixed *MatrixGate
	}{
		{"Phase(pi) is PauliZ", NewPhase(math.Pi), NewPauliZ()},
		{"Phase(pi/2) is S", NewPhase(math.Pi / 2), NewS()},
		{"Phase(pi/4) is T", NewPhase(math.Pi / 4), NewT()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !matrixClose(tc.phase.Matrix(), tc.fixed.Matrix(), identityTolerance) {
				t.Errorf("%s = %v, want %v", tc.phase.Name(), tc.phase.Matrix(), tc.fixed.Matrix())
			}
		})
	}
}

// TestRotationInverseAndComposition checks the group law every one-parameter
// family here obeys: the zero angle is the identity, opposite angles cancel,
// and rotations about the same axis add their angles.
func TestRotationInverseAndComposition(t *testing.T) {
	families := []struct {
		name  string
		build func(float64) *MatrixGate
	}{
		{"Rx", NewRx},
		{"Ry", NewRy},
		{"Rz", NewRz},
		{"Phase", NewPhase},
	}
	angles := []float64{0, 0.3, math.Pi / 4, math.Pi / 2, math.Pi, 2 * math.Pi, -1.7}
	pairs := [][2]float64{{0.3, 0.4}, {math.Pi / 4, math.Pi / 4}, {math.Pi, -math.Pi / 2}, {-1.7, 2.9}}

	identity := identityMatrix(2)

	for _, family := range families {
		t.Run(family.name, func(t *testing.T) {
			if !matrixClose(family.build(0).Matrix(), identity, identityTolerance) {
				t.Errorf("%s(0) = %v, want the identity", family.name, family.build(0).Matrix())
			}

			for _, angle := range angles {
				forward := family.build(angle).Matrix()
				backward := family.build(-angle).Matrix()
				product, err := ComposeMatrices(forward, backward)
				if err != nil {
					t.Fatalf("compose %s(%g) with its inverse: %v", family.name, angle, err)
				}
				if !matrixClose(product, identity, composedTolerance) {
					t.Errorf("%s(%g)·%s(%g) = %v, want the identity",
						family.name, angle, family.name, -angle, product)
				}
			}

			for _, pair := range pairs {
				product, err := ComposeMatrices(family.build(pair[0]).Matrix(), family.build(pair[1]).Matrix())
				if err != nil {
					t.Fatalf("compose %s(%g) with %s(%g): %v", family.name, pair[0], family.name, pair[1], err)
				}
				want := family.build(pair[0] + pair[1]).Matrix()
				if !matrixClose(product, want, composedTolerance) {
					t.Errorf("%s(%g)·%s(%g) = %v, want %s(%g) = %v",
						family.name, pair[0], family.name, pair[1], product,
						family.name, pair[0]+pair[1], want)
				}
			}
		})
	}
}

// TestNewGatesAreUnitary checks U·U† = I for every gate this package gained,
// including the controlled forms, which is the property that makes them
// legal quantum operations at all.
func TestNewGatesAreUnitary(t *testing.T) {
	controlledRotation, err := NewControlled(NewRx(0.7))
	if err != nil {
		t.Fatalf("NewControlled(Rx(0.7)) failed: %v", err)
	}
	controlledCNOT, err := NewControlled(NewCNOT())
	if err != nil {
		t.Fatalf("NewControlled(CNOT) failed: %v", err)
	}

	cases := []*MatrixGate{
		NewToffoli(),
		NewRx(0.3), NewRx(math.Pi),
		NewRy(-1.1), NewRy(math.Pi),
		NewRz(2.4), NewRz(math.Pi),
		NewPhase(0.9), NewPhase(math.Pi / 4),
		controlledRotation, controlledCNOT,
	}

	for _, gate := range cases {
		t.Run(gate.Name(), func(t *testing.T) {
			assertUnitary(t, gate.Name(), gate.Matrix())
		})
	}
}

// assertUnitary fails the test unless matrix·matrix† is the identity.
func assertUnitary(t *testing.T, name string, matrix [][]complex128) {
	t.Helper()
	product, err := ComposeMatrices(matrix, dagger(matrix))
	if err != nil {
		t.Fatalf("%s: compose with adjoint: %v", name, err)
	}
	if !matrixClose(product, identityMatrix(len(matrix)), composedTolerance) {
		t.Errorf("%s: U·U† = %v, want the identity", name, product)
	}
}

// globalPhase reports whether left is right scaled by a single unit-modulus
// scalar, and returns that scalar. Two matrices related this way describe the
// same physical operation, since a global phase is unobservable. The returned
// phase is meaningful even when the report is false, so a failing test can
// print the scale factor that came closest.
func globalPhase(left, right [][]complex128, tol float64) (complex128, bool) {
	if len(left) != len(right) {
		return 0, false
	}
	for i := range left {
		if len(left[i]) != len(right[i]) {
			return 0, false
		}
	}

	// Any entry of right that is not (near) zero fixes the candidate scalar;
	// zero entries divide badly and constrain nothing.
	var phase complex128
	found := false
search:
	for i := range right {
		for j := range right[i] {
			if cmplx.Abs(right[i][j]) > tol {
				phase = left[i][j] / right[i][j]
				found = true
				break search
			}
		}
	}
	if !found {
		return 0, false
	}
	if math.Abs(cmplx.Abs(phase)-1) > tol {
		return phase, false
	}

	return phase, matrixClose(left, scaleMatrix(right, phase), tol)
}

// scaleMatrix returns matrix with every entry multiplied by factor.
func scaleMatrix(matrix [][]complex128, factor complex128) [][]complex128 {
	out := make([][]complex128, len(matrix))
	for i, row := range matrix {
		out[i] = make([]complex128, len(row))
		for j, v := range row {
			out[i][j] = factor * v
		}
	}
	return out
}

// dagger returns the conjugate transpose of a square matrix.
func dagger(matrix [][]complex128) [][]complex128 {
	out := make([][]complex128, len(matrix))
	for i := range out {
		out[i] = make([]complex128, len(matrix))
		for j := range out[i] {
			out[i][j] = cmplx.Conj(matrix[j][i])
		}
	}
	return out
}

// identityMatrix returns the size x size identity.
func identityMatrix(size int) [][]complex128 {
	out := make([][]complex128, size)
	for i := range out {
		out[i] = make([]complex128, size)
		out[i][i] = 1
	}
	return out
}
