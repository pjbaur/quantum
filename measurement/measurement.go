package measurement

import (
	"fmt"
	"math/cmplx"
	"math/rand"

	"github.com/pjbaur/quantum/qubit"
	"github.com/pjbaur/quantum/state"
)

// MeasureQubit measures a single qubit and collapses its state
// Returns the result (0 or 1) based on probability distribution
func MeasureQubit(q *qubit.Qubit) int {
	// Calculate probability of measuring |0⟩
	prob0 := cmplx.Abs(q.Alpha) * cmplx.Abs(q.Alpha)

	// Generate random number to determine outcome
	if rand.Float64() < prob0 {
		// Collapse to |0⟩
		q.Alpha = 1.0 + 0i
		q.Beta = 0.0 + 0i
		return 0
	}

	// Collapse to |1⟩
	q.Alpha = 0.0 + 0i
	q.Beta = 1.0 + 0i
	return 1
}

// MeasureState measures all qubits in a quantum state and collapses the state
// Returns the measured integer value representing the state
func MeasureState(qs *state.QuantumState) int {
	// Calculate probabilities for each basis state
	var probs []float64
	var sum float64

	for _, amp := range qs.Amplitudes {
		prob := cmplx.Abs(amp) * cmplx.Abs(amp)
		probs = append(probs, prob)
		sum += prob
	}

	// Normalize probabilities (in case of floating point errors)
	for i := range probs {
		probs[i] /= sum
	}

	// Choose outcome based on probabilities
	r := rand.Float64()
	var cumulative float64

	for i := 0; i < len(probs); i++ {
		cumulative += probs[i]
		if r < cumulative {
			// Collapse to this state
			newAmplitudes := make([]complex128, len(qs.Amplitudes))
			newAmplitudes[i] = 1.0 + 0i
			qs.Amplitudes = newAmplitudes
			return i
		}
	}

	// Fallback to the last state (should rarely happen due to floating point issues)
	return len(probs) - 1
}

// MeasureSingleQubit measures a specific qubit in a multi-qubit system
// This is a more complex operation as it doesn't fully collapse the state vector
// Returns the measured result (0 or 1) for the specific qubit
func MeasureSingleQubit(qs *state.QuantumState, qubitIndex int) (int, error) {
	// Input validation
	if qubitIndex < 0 || qubitIndex >= qs.NQubits {
		return 0, fmt.Errorf("qubit index %d out of range [0,%d)", qubitIndex, qs.NQubits)
	}

	// Calculate probability of the qubit being in state |0⟩
	var prob0 float64
	for i, amp := range qs.Amplitudes {
		// Check if the bit at qubitIndex is 0
		if (i>>qubitIndex)&1 == 0 {
			prob0 += cmplx.Abs(amp) * cmplx.Abs(amp)
		}
	}

	// Determine the measurement outcome
	result := 0
	if rand.Float64() > prob0 {
		result = 1
	}

	// Collapse the state vector according to the measurement
	newAmplitudes := make([]complex128, len(qs.Amplitudes))
	var normalizationFactor float64

	for i, amp := range qs.Amplitudes {
		// Check if this basis state has the measured value at qubitIndex
		if ((i >> qubitIndex) & 1) == result {
			newAmplitudes[i] = amp
			normalizationFactor += cmplx.Abs(amp) * cmplx.Abs(amp)
		}
	}

	// Normalize the new state vector
	normalizationFactor = 1.0 / cmplx.Abs(complex(normalizationFactor, 0))
	for i := range newAmplitudes {
		newAmplitudes[i] *= complex(normalizationFactor, 0)
	}

	// Update the quantum state
	qs.Amplitudes = newAmplitudes
	return result, nil
}

// GetProbabilities returns the probability distribution of all possible states
// without collapsing the state
func GetProbabilities(qs *state.QuantumState) map[int]float64 {
	probs := make(map[int]float64)

	for i, amp := range qs.Amplitudes {
		prob := cmplx.Abs(amp) * cmplx.Abs(amp)
		if prob > 0.0001 { // Only include non-negligible probabilities
			probs[i] = prob
		}
	}

	return probs
}

// SimulateMeasurements runs multiple measurements on the same quantum state
// and returns frequency statistics without modifying the original state
func SimulateMeasurements(qs *state.QuantumState, trials int) map[int]int {
	results := make(map[int]int)
	probabilities := make([]float64, len(qs.Amplitudes))

	// Calculate probabilities
	for i, amp := range qs.Amplitudes {
		probabilities[i] = cmplx.Abs(amp) * cmplx.Abs(amp)
	}

	// Run trials
	for i := 0; i < trials; i++ {
		r := rand.Float64()
		var cumulative float64

		for j := 0; j < len(probabilities); j++ {
			cumulative += probabilities[j]
			if r < cumulative {
				results[j]++
				break
			}
		}
	}

	return results
}

// FormatMeasurementResults formats measurement results as a string
// showing binary representations of states and their frequencies
func FormatMeasurementResults(results map[int]int, nQubits int, trials int) string {
	var output string

	output += fmt.Sprintf("Measurement results (%d trials):\n", trials)
	for state, count := range results {
		percentage := float64(count) / float64(trials) * 100
		output += fmt.Sprintf("|%0*b⟩: %d times (%.2f%%)\n",
			nQubits, state, count, percentage)
	}

	return output
}
