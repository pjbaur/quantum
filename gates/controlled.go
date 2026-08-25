package gates

import (
	"errors"
	"fmt"

	"github.com/pjbaur/quantum/quantum"
)

// NewControlled returns the controlled version of gate: a gate on one more
// qubit that leaves the target qubits alone while the control is |0⟩ and
// applies gate to them while it is |1⟩. The control is targets[0] when the
// result is applied, the same most-significant-bit-first convention CNOT
// already follows, so NewControlled(NewPauliX()) reproduces CNOT and
// NewControlled(NewCNOT()) reproduces Toffoli.
//
// The result is the block-diagonal matrix diag(I, U), which is the direct sum
// |0⟩⟨0| ⊗ I + |1⟩⟨1| ⊗ U. It is written below as a block copy rather than as
// two TensorProduct calls because the package has no matrix addition: going
// through the helper would mean adding both an identity builder and an
// addMatrices helper, and their error returns could never fire once the input
// matrix is known square, all to replace two loops.
func NewControlled(gate quantum.Gate) (*MatrixGate, error) {
	if gate == nil {
		return nil, errors.New("gate must not be nil")
	}

	matrix := gate.Matrix()
	size, err := squareSize(matrix)
	if err != nil {
		return nil, fmt.Errorf("controlled %q: %w", gate.Name(), err)
	}
	// A 1x1 matrix is a scalar, not a gate; NewMatrixGate would accept the
	// 2x2 result, so the input has to be rejected here.
	if size < 2 {
		return nil, fmt.Errorf("controlled %q: matrix must be at least 2x2", gate.Name())
	}

	controlled := make([][]complex128, 2*size)
	for row := range controlled {
		controlled[row] = make([]complex128, 2*size)
	}
	for i := 0; i < size; i++ {
		// Control clear: the target block passes through unchanged.
		controlled[i][i] = 1
		// Control set: the target block is the original gate.
		copy(controlled[size+i][size:], matrix[i])
	}

	return NewMatrixGate("C-"+gate.Name(), controlled)
}
