package algorithm

import (
	"fmt"
	"math"

	"github.com/pjbaur/quantum/quantum"
)

// groundStateTolerance is the slack allowed when checking that a caller's
// state still holds all of its probability in |0…0⟩. It matches the
// normalization tolerance both backends apply.
const groundStateTolerance = 1e-10

// bulkState is what the algorithms in this package need of a backend: a
// quantum.QuantumState that can also take a whole amplitude vector at once.
// Their oracles and Grover's diffusion pass through vectors that are only
// normalized once every amplitude has been written, which is precisely what a
// sequence of QuantumState.SetAmplitude calls cannot express — each of those
// has to leave the state normalized on its own.
type bulkState interface {
	quantum.QuantumState
	quantum.BulkAmplitudeSetter
}

// requireBulkState adapts s to that contract, reporting a typed error rather
// than panicking when the backend cannot take bulk writes. The backend is the
// caller's choice now, so an unsupported one is a usage error to report.
func requireBulkState(s quantum.QuantumState) (bulkState, error) {
	bulk, ok := s.(bulkState)
	if !ok {
		return nil, &quantum.UnsupportedOperationError{
			Operation:   "bulk amplitude writes (quantum.BulkAmplitudeSetter)",
			Backend:     fmt.Sprintf("%T", s),
			Alternative: "a backend implementing SetAmplitudes (state.State)",
		}
	}
	return bulk, nil
}

// requireGroundState rejects a state that does not start in |0…0⟩. Both
// algorithms build their superposition from that starting point and size their
// work by the register alone, so a state carrying earlier work would run to
// completion and return a wrong answer rather than an error.
func requireGroundState(s quantum.QuantumState) error {
	if prob := s.Probability(0); math.Abs(prob-1.0) > groundStateTolerance {
		return fmt.Errorf("state must start in |0...0> but holds probability %.6f there", prob)
	}
	return nil
}
