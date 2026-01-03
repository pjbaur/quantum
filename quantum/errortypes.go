package quantum

import (
	"fmt"
)

// QubitsOutOfRangeError indicates that a qubit index is out of the valid range
type QubitsOutOfRangeError struct {
	Index    int
	MaxIndex int
}

func (e *QubitsOutOfRangeError) Error() string {
	return fmt.Sprintf("qubit index %d is out of range [0,%d]", e.Index, e.MaxIndex)
}

// NormalizationError indicates that a quantum state is not properly normalized
type NormalizationError struct {
	Sum float64
}

func (e *NormalizationError) Error() string {
	return fmt.Sprintf("quantum state is not normalized (sum of probabilities = %f, should be 1.0)", e.Sum)
}

// InvalidGateApplicationError indicates that a gate cannot be applied in the requested manner
type InvalidGateApplicationError struct {
	Gate        string
	RequiredLen int
	ActualLen   int
}

func (e *InvalidGateApplicationError) Error() string {
	return fmt.Sprintf("gate %s requires %d qubits but got %d", e.Gate, e.RequiredLen, e.ActualLen)
}

// IncompatibleQubitCountError indicates mismatched qubit counts for an operation.
type IncompatibleQubitCountError struct {
	Expected int
	Actual   int
}

func (e *IncompatibleQubitCountError) Error() string {
	return fmt.Sprintf("expected %d qubits but got %d", e.Expected, e.Actual)
}
