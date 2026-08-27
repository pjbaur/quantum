package algorithm

import (
	"errors"
	"fmt"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/quantum"
)

// Teleport teleports the state of qubit 0 into qubit 2 of s, using qubits
// 0 and 1 as scratch, and returns s for convenience. The protocol:
//
//	prepare a Bell pair on qubits 1 and 2 (H on 1, CNOT 1→2)
//	entangle the input with the pair (CNOT 0→1, H on 0)
//	measure qubits 0 and 1 into classical bits m0 and m1
//	apply X to qubit 2 if m1 is set, Z to qubit 2 if m0 is set
//
// The two classical corrections are exactly the feed-forward control
// teleportation exists to demonstrate: gates conditioned on measurement
// outcomes. After the call, qubits 0 and 1 hold the measured values and
// qubit 2 holds the input state up to global phase, which carries no
// observable meaning.
//
// Measurement randomness comes from the state's own source, so
// SetRandSource makes the run reproducible; with the outcomes fixed, the
// result is exact (no sampling error). Verify with
// quantum.Expectation on qubit 2, e.g. axes I,I,X / I,I,Y / I,I,Z to
// compare against the input's Bloch vector, or quantum.Fidelity against a
// re-prepared register where the comparison fits the qubit count.
//
// The state is manipulated directly rather than through a circuit.Circuit
// because mid-circuit measurement plus classical conditioning is
// state-level control flow, which the circuit executor (a fixed gate
// sequence) does not model.
//
// Returns an error if s is nil or has fewer than 3 qubits. Gate and
// measurement errors propagate.
func Teleport(s quantum.QuantumState) (quantum.QuantumState, error) {
	if s == nil {
		return nil, errors.New("state must not be nil")
	}
	if s.NumQubits() < 3 {
		return nil, fmt.Errorf("state needs at least 3 qubits (input, scratch, target), got %d",
			s.NumQubits())
	}

	// Bell pair on qubits 1 and 2.
	if err := s.ApplyGate(gates.NewHadamard(), 1); err != nil {
		return nil, err
	}
	if err := s.ApplyGate(gates.NewCNOT(), 1, 2); err != nil {
		return nil, err
	}

	// Entangle the input qubit with the pair.
	if err := s.ApplyGate(gates.NewCNOT(), 0, 1); err != nil {
		return nil, err
	}
	if err := s.ApplyGate(gates.NewHadamard(), 0); err != nil {
		return nil, err
	}

	// Measure and correct: X on the target if m1, Z if m0.
	creg, err := quantum.NewClassicalRegister(2)
	if err != nil {
		return nil, err
	}
	if err := quantum.MeasureInto(s, 0, creg, 0); err != nil {
		return nil, err
	}
	if err := quantum.MeasureInto(s, 1, creg, 1); err != nil {
		return nil, err
	}
	if err := quantum.ApplyIfSet(creg, 1, s, gates.NewPauliX(), 2); err != nil {
		return nil, err
	}
	if err := quantum.ApplyIfSet(creg, 0, s, gates.NewPauliZ(), 2); err != nil {
		return nil, err
	}
	return s, nil
}
