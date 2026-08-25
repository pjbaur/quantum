package gates

import (
	"testing"

	"github.com/pjbaur/quantum/quantum"
)

// stubGate is a Gate whose matrix the test supplies verbatim, including
// matrices no valid gate would have.
type stubGate struct {
	name   string
	matrix [][]complex128
}

func (g stubGate) Name() string           { return g.name }
func (g stubGate) Matrix() [][]complex128 { return g.matrix }

// TestControlledXIsCNOT is the anchor for the construction: controlling the
// X gate has to reproduce the CNOT table this package already ships, entry
// for entry, or every controlled gate built from it is in a different basis
// ordering than the rest of the library.
func TestControlledXIsCNOT(t *testing.T) {
	controlled, err := NewControlled(NewPauliX())
	if err != nil {
		t.Fatalf("NewControlled(PauliX) failed: %v", err)
	}

	if got, want := controlled.Name(), "C-PauliX"; got != want {
		t.Errorf("Name() = %q, want %q", got, want)
	}
	if got := controlled.NumQubits(); got != 2 {
		t.Errorf("NumQubits() = %d, want 2", got)
	}
	assertMatrixEqual(t, controlled.Matrix(), NewCNOT().Matrix())
}

// TestControlledCNOTIsToffoli exercises the construction over a target gate
// wider than one qubit, and cross-checks the built-in Toffoli table against
// an independent derivation of it.
func TestControlledCNOTIsToffoli(t *testing.T) {
	controlled, err := NewControlled(NewCNOT())
	if err != nil {
		t.Fatalf("NewControlled(CNOT) failed: %v", err)
	}

	if got, want := controlled.Name(), "C-CNOT"; got != want {
		t.Errorf("Name() = %q, want %q", got, want)
	}
	if got := controlled.NumQubits(); got != 3 {
		t.Errorf("NumQubits() = %d, want 3", got)
	}
	assertMatrixEqual(t, controlled.Matrix(), NewToffoli().Matrix())
}

// TestControlledKeepsTargetBlockUntouchedWhenControlClear checks the block
// structure directly with a target gate whose entries are not 0 and 1, which
// the permutation-gate identities above cannot distinguish: the upper half of
// the matrix must be the identity and the lower half must be the target gate.
func TestControlledKeepsTargetBlockUntouchedWhenControlClear(t *testing.T) {
	target := NewRx(0.7)
	controlled, err := NewControlled(target)
	if err != nil {
		t.Fatalf("NewControlled(%s) failed: %v", target.Name(), err)
	}

	matrix := controlled.Matrix()
	if len(matrix) != 4 {
		t.Fatalf("controlled matrix is %dx%d, want 4x4", len(matrix), len(matrix))
	}

	targetMatrix := target.Matrix()
	for row := 0; row < 2; row++ {
		for col := 0; col < 2; col++ {
			// Control clear: the identity block.
			want := complex128(0)
			if row == col {
				want = 1
			}
			if matrix[row][col] != want {
				t.Errorf("control-clear block [%d][%d] = %v, want %v", row, col, matrix[row][col], want)
			}
			// Control clear crossed with control set: no amplitude moves
			// between the two halves.
			if matrix[row][2+col] != 0 || matrix[2+row][col] != 0 {
				t.Errorf("off-diagonal block leaked at row %d col %d: %v, %v",
					row, col, matrix[row][2+col], matrix[2+row][col])
			}
			// Control set: the target gate, unmodified.
			if matrix[2+row][2+col] != targetMatrix[row][col] {
				t.Errorf("control-set block [%d][%d] = %v, want %v",
					row, col, matrix[2+row][2+col], targetMatrix[row][col])
			}
		}
	}
}

func TestControlledRejectsInvalidGate(t *testing.T) {
	cases := []struct {
		name string
		gate quantum.Gate
	}{
		{"nil gate", nil},
		{"nil matrix", stubGate{name: "Empty"}},
		{"ragged matrix", stubGate{name: "Ragged", matrix: [][]complex128{{1, 0}, {0}}}},
		{"scalar matrix", stubGate{name: "Scalar", matrix: [][]complex128{{1}}}},
		{"non power of two", stubGate{name: "Three", matrix: [][]complex128{
			{1, 0, 0}, {0, 1, 0}, {0, 0, 1},
		}}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gate, err := NewControlled(tc.gate)
			if err == nil {
				t.Fatalf("NewControlled succeeded, want error (got gate %q)", gate.Name())
			}
			if gate != nil {
				t.Errorf("NewControlled returned a non-nil gate alongside error %v", err)
			}
		})
	}
}

// TestControlledCopiesTargetMatrix guards the same aliasing contract
// MatrixGate makes: a caller mutating the gate it passed in must not reach
// back into the controlled gate built from it.
func TestControlledCopiesTargetMatrix(t *testing.T) {
	source := [][]complex128{{0, 1}, {1, 0}}
	controlled, err := NewControlled(stubGate{name: "X", matrix: source})
	if err != nil {
		t.Fatalf("NewControlled failed: %v", err)
	}

	source[0][1] = 99
	if got := controlled.Matrix()[2][3]; got != 1 {
		t.Errorf("controlled matrix affected by input mutation: got %v, want 1", got)
	}
}

// assertMatrixEqual compares two matrices entry for entry with no tolerance,
// which is what the callers above want: both sides are tables of 0 and 1.
func assertMatrixEqual(t *testing.T, got, want [][]complex128) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("matrix is %dx%d, want %dx%d", len(got), len(got), len(want), len(want))
	}
	for i := range want {
		for j := range want[i] {
			if got[i][j] != want[i][j] {
				t.Errorf("matrix[%d][%d] = %v, want %v", i, j, got[i][j], want[i][j])
			}
		}
	}
}
