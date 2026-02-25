package quantum

import (
	"math/bits"
)

// GateQubitCount returns the number of qubits a gate operates on,
// computed from the gate matrix dimensions.
//
// The function validates that:
//   - The matrix is not empty
//   - The matrix is square (same number of rows and columns)
//   - The matrix size is a power of 2 (e.g., 2x2 for 1-qubit, 4x4 for 2-qubit)
//
// Returns an InvalidGateMatrixError if validation fails.
func GateQubitCount(gate Gate) (int, error) {
	matrix := gate.Matrix()
	if len(matrix) == 0 {
		return 0, &InvalidGateMatrixError{
			GateName: gate.Name(),
			Reason:   "empty matrix",
		}
	}

	size := len(matrix)
	for _, row := range matrix {
		if len(row) != size {
			return 0, &InvalidGateMatrixError{
				GateName: gate.Name(),
				Reason:   "matrix must be square",
			}
		}
	}

	if size&(size-1) != 0 {
		return 0, &InvalidGateMatrixError{
			GateName: gate.Name(),
			Reason:   "matrix size is not a power of two",
			Size:     size,
		}
	}

	return bits.Len(uint(size)) - 1, nil
}
