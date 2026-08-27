package quantum

import (
	"errors"
)

// ClassicalRegister holds the classical bits produced by mid-circuit
// measurements so classical control flow can act on them: correction
// gates in teleportation, syndrome handling in error correction, adaptive
// algorithm steps. A register is not safe for concurrent use.
type ClassicalRegister struct {
	bits []bool
}

// NewClassicalRegister creates a register of numBits cleared bits.
// Returns InvalidBitCountError if numBits <= 0.
func NewClassicalRegister(numBits int) (*ClassicalRegister, error) {
	if numBits <= 0 {
		return nil, &InvalidBitCountError{
			Requested: numBits,
			Reason:    "must be positive",
		}
	}
	return &ClassicalRegister{bits: make([]bool, numBits)}, nil
}

// NumBits returns the number of classical bits in the register.
func (c *ClassicalRegister) NumBits() int {
	return len(c.bits)
}

// Bit returns the value of classical bit index.
// Returns BitOutOfRangeError if index is outside [0, NumBits).
func (c *ClassicalRegister) Bit(index int) (bool, error) {
	if index < 0 || index >= len(c.bits) {
		return false, &BitOutOfRangeError{
			Index:    index,
			MaxIndex: len(c.bits) - 1,
		}
	}
	return c.bits[index], nil
}

// Set writes the value of classical bit index.
// Returns BitOutOfRangeError if index is outside [0, NumBits).
func (c *ClassicalRegister) Set(index int, value bool) error {
	if index < 0 || index >= len(c.bits) {
		return &BitOutOfRangeError{
			Index:    index,
			MaxIndex: len(c.bits) - 1,
		}
	}
	c.bits[index] = value
	return nil
}

// MeasureInto measures qubit qubitIndex of s (collapsing it, exactly like
// QuantumState.Measure) and stores the 0/1 outcome in classical bit
// bitIndex of creg. This is the bridge from quantum to classical data that
// mid-circuit control flow needs.
//
// Returns an error if s or creg is nil, if the qubit index is outside the
// state's range (QubitsOutOfRangeError), or if the bit index is outside
// the register's range (BitOutOfRangeError). Measurement errors propagate.
func MeasureInto(s QuantumState, qubitIndex int, creg *ClassicalRegister, bitIndex int) error {
	if isNilQuantumState(s) {
		return errors.New("state must not be nil")
	}
	if creg == nil {
		return errors.New("classical register must not be nil")
	}
	if qubitIndex < 0 || qubitIndex >= s.NumQubits() {
		return &QubitsOutOfRangeError{
			Index:    qubitIndex,
			MaxIndex: s.NumQubits() - 1,
		}
	}
	if bitIndex < 0 || bitIndex >= len(creg.bits) {
		return &BitOutOfRangeError{
			Index:    bitIndex,
			MaxIndex: len(creg.bits) - 1,
		}
	}

	outcome, err := s.Measure(qubitIndex)
	if err != nil {
		return err
	}
	creg.bits[bitIndex] = outcome == 1
	return nil
}

// ApplyIfSet applies gate to targets on s only when classical bit bitIndex
// of creg is set — the "if bit then gate" of feed-forward control. When the
// bit is clear the call is a no-op. ApplyGate errors (bad target count,
// range, backend capability) propagate unchanged.
//
// Returns an error if s or creg is nil, or if the bit index is outside the
// register's range (BitOutOfRangeError).
func ApplyIfSet(creg *ClassicalRegister, bitIndex int, s QuantumState, gate Gate, targets ...int) error {
	if isNilQuantumState(s) {
		return errors.New("state must not be nil")
	}
	if creg == nil {
		return errors.New("classical register must not be nil")
	}
	set, err := creg.Bit(bitIndex)
	if err != nil {
		return err
	}
	if !set {
		return nil
	}
	return s.ApplyGate(gate, targets...)
}

// String renders the register as its bit values, index 0 leftmost, for
// demos and test failure messages.
func (c *ClassicalRegister) String() string {
	out := make([]byte, len(c.bits))
	for i, b := range c.bits {
		if b {
			out[i] = '1'
		} else {
			out[i] = '0'
		}
	}
	return string(out)
}
