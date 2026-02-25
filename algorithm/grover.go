package algorithm

import (
	"errors"
	"fmt"
	"math"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/state"
)

// Grover executes Grover's search algorithm and returns the final state.
// The marked slice contains basis states to amplify.
func Grover(numQubits int, marked []int) (*state.State, error) {
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

	hGate := gates.NewHadamard()
	search, err := state.New(numQubits)
	if err != nil {
		return nil, err
	}
	for i := 0; i < numQubits; i++ {
		if err := search.ApplyGate(hGate, i); err != nil {
			return nil, err
		}
	}

	iterations := groverIterations(totalStates, len(markedSet))
	for i := 0; i < iterations; i++ {
		// Apply oracle: flip sign of marked states
		applyGroverOracle(search, markedSet)
		// Apply diffusion: inversion about average
		applyGroverDiffusion(search)
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
func applyGroverOracle(s *state.State, marked []int) {
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
	s.SetAmplitudes(amps)
}

// applyGroverDiffusion applies the inversion-about-average operator.
// Formula: |ψ⟩ → 2|s⟩⟨s|ψ⟩ - |ψ⟩ where |s⟩ is the uniform superposition.
// This is computed as: new_amp[i] = 2*mean - old_amp[i]
// This is O(n) where n = 2^numQubits, instead of O(n^2) for constructing
// the full diffusion matrix.
func applyGroverDiffusion(s *state.State) {
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
	s.SetAmplitudes(amps)
}
