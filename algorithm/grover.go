package algorithm

import (
	"errors"
	"fmt"
	"math"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/quantum"
)

// Grover executes Grover's search algorithm on s and returns it.
// The marked slice contains basis states to amplify.
//
// The caller chooses the backend by choosing s, which must be a freshly
// created state in |0…0⟩ with the register the search is over; the algorithm
// evolves it in place and returns the same state for convenience. The backend
// must implement quantum.BulkAmplitudeSetter — the oracle and the diffusion
// step rewrite the whole amplitude vector — and an UnsupportedOperationError
// says so if it does not.
func Grover(s quantum.QuantumState, marked []int) (quantum.QuantumState, error) {
	if s == nil {
		return nil, errors.New("state must not be nil")
	}

	numQubits := s.NumQubits()
	if numQubits <= 0 {
		return nil, errors.New("numQubits must be positive")
	}
	if len(marked) == 0 {
		return nil, errors.New("marked set must not be empty")
	}

	totalStates := 1 << numQubits
	markedSet, err := uniqueMarked(marked, totalStates)
	if err != nil {
		return nil, err
	}

	search, err := requireBulkState(s)
	if err != nil {
		return nil, err
	}
	if err := requireGroundState(search); err != nil {
		return nil, err
	}

	hGate := gates.NewHadamard()
	for i := 0; i < numQubits; i++ {
		if err := search.ApplyGate(hGate, i); err != nil {
			return nil, err
		}
	}

	iterations := groverIterations(totalStates, len(markedSet))
	for i := 0; i < iterations; i++ {
		// Apply oracle: flip sign of marked states
		if err := applyGroverOracle(search, markedSet); err != nil {
			return nil, err
		}
		// Apply diffusion: inversion about average
		if err := applyGroverDiffusion(search); err != nil {
			return nil, err
		}
	}

	return search, nil
}

func groverIterations(totalStates, marked int) int {
	estimate := (math.Pi / 4) * math.Sqrt(float64(totalStates)/float64(marked))
	iterations := int(math.Round(estimate))
	if iterations < 1 {
		return 1
	}
	return iterations
}

func uniqueMarked(marked []int, totalStates int) ([]int, error) {
	seen := make(map[int]struct{}, len(marked))
	unique := make([]int, 0, len(marked))
	for _, state := range marked {
		if state < 0 || state >= totalStates {
			return nil, fmt.Errorf("marked state %d out of range", state)
		}
		if _, ok := seen[state]; ok {
			continue
		}
		seen[state] = struct{}{}
		unique = append(unique, state)
	}
	return unique, nil
}

// applyGroverOracle flips the sign of amplitudes at marked basis states.
// This is O(m + n) where m is the number of marked states and n = 2^numQubits,
// instead of O(n^2) for constructing the full oracle matrix.
func applyGroverOracle(s bulkState, marked []int) error {
	n := s.NumQubits()
	size := 1 << n

	// Get all amplitudes
	amps := make([]complex128, size)
	for i := 0; i < size; i++ {
		amps[i] = s.Amplitude(i)
	}

	// Flip sign at marked indices
	for _, idx := range marked {
		amps[idx] = -amps[idx]
	}

	// Set all amplitudes at once (single normalization check)
	return s.SetAmplitudes(amps)
}

// applyGroverDiffusion applies the inversion-about-average operator.
// Formula: |ψ⟩ → 2|s⟩⟨s|ψ⟩ - |ψ⟩ where |s⟩ is the uniform superposition.
// This is computed as: new_amp[i] = 2*mean - old_amp[i]
// This is O(n) where n = 2^numQubits, instead of O(n^2) for constructing
// the full diffusion matrix.
func applyGroverDiffusion(s bulkState) error {
	n := s.NumQubits()
	size := 1 << n

	// Get all amplitudes
	amps := make([]complex128, size)
	for i := 0; i < size; i++ {
		amps[i] = s.Amplitude(i)
	}

	// Compute mean of all amplitudes
	var sum complex128
	for i := 0; i < size; i++ {
		sum += amps[i]
	}
	mean := sum / complex(float64(size), 0)

	// Apply inversion about average: new_amp = 2*mean - old_amp
	for i := 0; i < size; i++ {
		amps[i] = 2*mean - amps[i]
	}

	// Set all amplitudes at once (single normalization check)
	return s.SetAmplitudes(amps)
}
