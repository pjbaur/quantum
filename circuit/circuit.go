// Package circuit provides a simple circuit abstraction for sequencing gates.
package circuit

import (
	"errors"
	"fmt"
	"math/bits"

	"github.com/pjbaur/quantum/quantum"
)

// Operation represents a gate application on specific target qubits.
type Operation struct {
	Gate    quantum.Gate
	Targets []int
}

// Circuit defines a sequence of gate operations for a fixed number of qubits.
type Circuit struct {
	numQubits  int
	operations []Operation
}

// New creates a circuit for the specified number of qubits.
// Returns InvalidQubitCountError if numQubits <= 0.
func New(numQubits int) (*Circuit, error) {
	if numQubits <= 0 {
		return nil, &quantum.InvalidQubitCountError{
			Requested: numQubits,
			Reason:    "must be positive",
		}
	}

	return &Circuit{
		numQubits: numQubits,
	}, nil
}

// NumQubits returns the number of qubits the circuit operates on.
func (c *Circuit) NumQubits() int {
	return c.numQubits
}

// AddGate appends a gate operation to the circuit.
func (c *Circuit) AddGate(gate quantum.Gate, targets ...int) error {
	if c == nil {
		return errors.New("circuit is nil")
	}
	if gate == nil {
		return errors.New("gate must not be nil")
	}
	if len(targets) == 0 {
		return errors.New("at least one target is required")
	}

	required, err := gateQubitCount(gate)
	if err != nil {
		return err
	}
	if len(targets) != required {
		return &quantum.InvalidGateApplicationError{
			Gate:        gate.Name(),
			RequiredLen: required,
			ActualLen:   len(targets),
		}
	}

	seen := make(map[int]struct{}, len(targets))
	for _, target := range targets {
		if target < 0 || target >= c.numQubits {
			return &quantum.QubitsOutOfRangeError{
				Index:    target,
				MaxIndex: c.numQubits - 1,
			}
		}
		if _, exists := seen[target]; exists {
			return errors.New("targets must be unique")
		}
		seen[target] = struct{}{}
	}

	operation := Operation{
		Gate:    gate,
		Targets: append([]int(nil), targets...),
	}
	c.operations = append(c.operations, operation)
	return nil
}

// Append appends another circuit's operations to this circuit.
func (c *Circuit) Append(other *Circuit) error {
	if c == nil {
		return errors.New("circuit is nil")
	}
	if other == nil {
		return errors.New("other circuit is nil")
	}
	if c.numQubits != other.numQubits {
		return &quantum.IncompatibleQubitCountError{
			Expected: c.numQubits,
			Actual:   other.numQubits,
		}
	}

	c.operations = append(c.operations, other.operations...)
	return nil
}

// Compose returns a new circuit with operations from this circuit followed by another.
func (c *Circuit) Compose(other *Circuit) (*Circuit, error) {
	if c == nil {
		return nil, errors.New("circuit is nil")
	}
	if other == nil {
		return nil, errors.New("other circuit is nil")
	}
	if c.numQubits != other.numQubits {
		return nil, &quantum.IncompatibleQubitCountError{
			Expected: c.numQubits,
			Actual:   other.numQubits,
		}
	}

	operations := make([]Operation, 0, len(c.operations)+len(other.operations))
	operations = append(operations, c.operations...)
	operations = append(operations, other.operations...)

	return &Circuit{
		numQubits:  c.numQubits,
		operations: operations,
	}, nil
}

// Execute applies the circuit's operations to the provided quantum state.
// If the state implements BackendCapabilities, it checks compatibility first.
func (c *Circuit) Execute(state quantum.QuantumState) error {
	if c == nil {
		return errors.New("circuit is nil")
	}
	if state == nil {
		return errors.New("state is nil")
	}
	if state.NumQubits() != c.numQubits {
		return &quantum.IncompatibleQubitCountError{
			Expected: c.numQubits,
			Actual:   state.NumQubits(),
		}
	}

	// Check backend capabilities before execution if supported
	if caps, ok := state.(quantum.BackendCapabilities); ok {
		if err := c.checkCapabilities(caps); err != nil {
			return err
		}
	}

	for _, operation := range c.operations {
		if err := state.ApplyGate(operation.Gate, operation.Targets...); err != nil {
			return err
		}
	}

	return nil
}

// checkCapabilities verifies that all operations in the circuit can be
// executed by a backend with the given capabilities.
func (c *Circuit) checkCapabilities(caps quantum.BackendCapabilities) error {
	for _, operation := range c.operations {
		required, err := gateQubitCount(operation.Gate)
		if err != nil {
			return err
		}
		if !caps.SupportsGateQubits(required) {
			return &quantum.UnsupportedOperationError{
				Operation:   fmt.Sprintf("%d-qubit gate (%s)", required, operation.Gate.Name()),
				Backend:     "current backend",
				Alternative: "dense state backend (state.State)",
			}
		}
	}
	return nil
}

// CheckBackendCapabilities checks whether a backend with the given capabilities
// can execute this circuit. This allows early detection of compatibility issues
// before execution begins.
func (c *Circuit) CheckBackendCapabilities(caps quantum.BackendCapabilities) error {
	if c == nil {
		return errors.New("circuit is nil")
	}
	if caps == nil {
		return errors.New("capabilities is nil")
	}
	return c.checkCapabilities(caps)
}

func gateQubitCount(gate quantum.Gate) (int, error) {
	matrix := gate.Matrix()
	if len(matrix) == 0 {
		return 0, fmt.Errorf("gate %s has empty matrix", gate.Name())
	}

	size := len(matrix)
	for _, row := range matrix {
		if len(row) != size {
			return 0, fmt.Errorf("gate %s matrix must be square", gate.Name())
		}
	}

	if size&(size-1) != 0 {
		return 0, fmt.Errorf("gate %s matrix size %d is not a power of two", gate.Name(), size)
	}

	return bits.Len(uint(size)) - 1, nil
}
