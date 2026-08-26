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

// NormalizationError indicates that a quantum state is not properly normalized.
// AttemptedSum is the probability sum that would have resulted from the change.
// CurrentSum is the probability sum after rollback (should be ~1.0).
type NormalizationError struct {
	AttemptedSum float64
	CurrentSum   float64
}

func (e *NormalizationError) Error() string {
	return fmt.Sprintf("normalization violation: attempted change would result in "+
		"probability sum %.6f (must be 1.0); state rolled back to valid sum %.6f",
		e.AttemptedSum, e.CurrentSum)
}

// NonFiniteAmplitudeError indicates that an amplitude is NaN or infinite.
// Such a value is rejected before the normalization check, which cannot see
// it: NaN compares false against any tolerance.
type NonFiniteAmplitudeError struct {
	BasisState int
	Value      complex128
}

func (e *NonFiniteAmplitudeError) Error() string {
	return fmt.Sprintf("amplitude for basis state %d is not finite: %v", e.BasisState, e.Value)
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

// InvalidQubitCountError indicates that an invalid number of qubits was specified.
type InvalidQubitCountError struct {
	Requested int
	Reason    string
}

func (e *InvalidQubitCountError) Error() string {
	return fmt.Sprintf("invalid qubit count %d: %s", e.Requested, e.Reason)
}

// SharedStateError indicates that the same quantum state was passed
// to multiple parallel executions, which would cause data races.
type SharedStateError struct {
	DuplicateIndices []int
}

func (e *SharedStateError) Error() string {
	return fmt.Sprintf("parallel execution requires independent states: "+
		"executions %v share the same state pointer", e.DuplicateIndices)
}

// UnsupportedOperationError indicates that an operation is valid but not
// supported by this backend. Callers should use a different backend.
type UnsupportedOperationError struct {
	Operation   string
	Backend     string
	Alternative string
}

func (e *UnsupportedOperationError) Error() string {
	return fmt.Sprintf("%s does not support %s: use %s instead",
		e.Backend, e.Operation, e.Alternative)
}

// UncomparableStateError indicates that an execution's state has a dynamic
// type that is not comparable (e.g. a struct value containing a slice, map,
// or function field), so it cannot be used as a map key when checking for
// shared state across parallel executions.
type UncomparableStateError struct {
	Index    int
	TypeName string
}

func (e *UncomparableStateError) Error() string {
	return fmt.Sprintf("execution %d: state has uncomparable type %s: "+
		"QuantumState implementations must use a comparable dynamic type "+
		"(e.g. a pointer implementation) to support parallel execution", e.Index, e.TypeName)
}

// InvalidGateMatrixError indicates that a gate's matrix representation is invalid.
type InvalidGateMatrixError struct {
	GateName string
	Reason   string
	Size     int // Matrix size, if applicable
}

func (e *InvalidGateMatrixError) Error() string {
	if e.Size > 0 {
		return fmt.Sprintf("gate %s has invalid matrix: %s (size %d)", e.GateName, e.Reason, e.Size)
	}
	return fmt.Sprintf("gate %s has invalid matrix: %s", e.GateName, e.Reason)
}
