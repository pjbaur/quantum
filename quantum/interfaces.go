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

// Gate represents a quantum gate operation
type Gate interface {
	// Apply applies the gate to the given qubit
	// For multi-qubit gates, specific implementations will handle the details
	Apply(q Qubit) error

	// Name returns the name of the gate
	Name() string

	// Matrix returns the matrix representation of the gate
	Matrix() [][]complex128
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
