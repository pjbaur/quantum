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
//   - The matrix is at least 2x2, so the gate acts on at least one qubit
//
// Returns an InvalidGateMatrixError if validation fails.
//
// Gates implementing QubitCounter with a positive NumQubits are trusted
// and short-circuit the matrix inspection entirely, avoiding the copy
// that Matrix() implies on every gate application.
func GateQubitCount(gate Gate) (int, error) {
	if counter, ok := gate.(QubitCounter); ok {
		if n := counter.NumQubits(); n > 0 {
			return n, nil
		}
	}

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

	// 1x1 passes the power-of-two test as 2^0, but a gate acting on no
	// qubits is not a gate: callers use the count to size their target
	// list, and a zero would let a scalar be "applied" to nothing.
	if size < 2 {
		return 0, &InvalidGateMatrixError{
			GateName: gate.Name(),
			Reason:   "matrix must be at least 2x2",
			Size:     size,
		}
	}

	return bits.Len(uint(size)) - 1, nil
}
