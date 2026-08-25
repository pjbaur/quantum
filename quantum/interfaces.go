// Package quantum defines core interfaces for qubits, gates, and multi-qubit
// state simulation used throughout the project.
package quantum

// Qubit represents a quantum bit with amplitude coefficients
type Qubit interface {
	// Alpha returns the amplitude of the |0⟩ basis state
	Alpha() complex128

	// Beta returns the amplitude of the |1⟩ basis state
	Beta() complex128

	// Set updates the amplitudes of the qubit
	// Returns error if the resulting state would not be normalized
	Set(alpha, beta complex128) error

	// Probability0 returns the probability of measuring |0⟩
	Probability0() float64

	// Probability1 returns the probability of measuring |1⟩
	Probability1() float64

	// Measure collapses the qubit to either |0⟩ or |1⟩ based on probabilities
	// Returns the measured value (0 or 1)
	Measure() int

	// Clone creates a copy of this qubit
	Clone() Qubit
}

// Gate represents a quantum gate operation.
// Gates provide metadata (name, matrix) but do not apply themselves directly.
// Use QuantumState.ApplyGate(gate, targets...) to apply gates to state vectors.
type Gate interface {
	// Name returns the name of the gate
	Name() string

	// Matrix returns the matrix representation of the gate
	Matrix() [][]complex128
}

// RandomSource supplies the uniform [0, 1) randomness that measurement
// consumes. *math/rand.Rand satisfies it, so a seeded generator can be
// injected for reproducible simulations. Implementations need not be
// safe for concurrent use; states measured in parallel should each get
// their own source.
type RandomSource interface {
	// Float64 returns a pseudo-random number in [0, 1).
	Float64() float64
}

// QubitCounter is implemented by gates that know how many qubits they
// operate on without exposing their matrix. GateQubitCount uses it to
// skip matrix inspection (and the copy it implies) for such gates.
type QubitCounter interface {
	// NumQubits returns the number of qubits the gate operates on.
	// A non-positive value is ignored and the matrix is inspected instead.
	NumQubits() int
}

// QuantumState represents a multi-qubit quantum state
type QuantumState interface {
	// NumQubits returns the number of qubits in the state
	NumQubits() int

	// Amplitude returns the amplitude of a specific basis state
	// The basisState parameter is the integer representation of the basis state
	Amplitude(basisState int) complex128

	// SetAmplitude sets the amplitude for a specific basis state
	// Returns error if the resulting state would not be normalized
	SetAmplitude(basisState int, value complex128) error

	// ApplyGate applies a gate to the specified qubit(s)
	// For single-qubit gates, targets contains one index
	// For multi-qubit gates, targets contains multiple indices in specific order
	ApplyGate(gate Gate, targets ...int) error

	// Measure measures the specified qubit and collapses the state
	// Returns the measured value (0 or 1)
	Measure(qubitIndex int) (int, error)

	// Probability returns the probability of measuring a specific basis state
	Probability(basisState int) float64

	// Clone creates a copy of this quantum state
	Clone() QuantumState
}

// BulkAmplitudeSetter is implemented by state backends that can replace the
// entire amplitude vector in one call, validating it once instead of once per
// amplitude. That is what lets an algorithm pass through an intermediate
// vector no sequence of QuantumState.SetAmplitude writes could reach, since
// each of those has to leave the state normalized on its own.
//
// It is an optional capability, kept out of QuantumState so a backend that
// cannot offer it (a hardware or network-backed state, say) can still be a
// QuantumState. Callers type-assert and report an UnsupportedOperationError
// when the assertion fails.
type BulkAmplitudeSetter interface {
	// SetAmplitudes replaces every amplitude at once, rejecting the write
	// unless the vector has exactly 2^NumQubits finite entries whose
	// probabilities sum to 1 within the backend's tolerance. A rejected
	// write leaves the state untouched.
	SetAmplitudes(values []complex128) error
}

// BackendCapabilities describes what operations a quantum state backend supports.
type BackendCapabilities interface {
	// SupportsGateQubits returns whether this backend can apply gates
	// operating on the specified number of qubits.
	SupportsGateQubits(qubitCount int) bool

	// MaxGateQubits returns the maximum number of qubits a gate can operate on.
	// Returns 0 if there is no limit.
	MaxGateQubits() int
}
