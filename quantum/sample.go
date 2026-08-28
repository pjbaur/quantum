package quantum

import (
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
)

// Sample draws `shots` measurement outcomes from the computational-basis
// probability distribution of s and returns them as a histogram keyed by
// bitstring. Keys are the basis state's index in binary, zero-padded to the
// qubit count with the highest-index qubit as the most significant bit
// (qubit 0 is the least significant bit), matching FormatStateView's
// |%0*b> formatting.
//
// Sampling is non-destructive: it reads amplitudes through Amplitude and
// never collapses the state, so repeated calls resample the same
// distribution. This is the batch counterpart to Measure, which measures a
// single qubit and collapses.
//
// rng supplies the uniform [0, 1) randomness, so a seeded generator or stub
// makes outcomes reproducible; a nil rng falls back to the global math/rand
// source. Implementations need not be safe for concurrent use.
//
// Returns an error if s is nil or a typed nil (e.g. a nil *state.State boxed
// in the interface), if shots is not positive (InvalidShotCountError), or if
// the state's probabilities do not sum to 1 within the backends' 1e-10
// tolerance (UnnormalizedStateError; a NaN probability sum fails the same
// check).
//
// Cost is O(2^n) to walk the amplitude vector plus O(shots log 2^n) for the
// draws. Sparse backends are walked in full, which gives up sparsity for
// large registers; their nonzero-only advantage would need a backend-side
// override.
func Sample(s QuantumState, shots int, rng RandomSource) (map[string]int, error) {
	if isNilQuantumState(s) {
		return nil, errors.New("state must not be nil")
	}
	if shots <= 0 {
		return nil, &InvalidShotCountError{
			Requested: shots,
			Reason:    "must be positive",
		}
	}

	numQubits := s.NumQubits()
	size := 1 << numQubits

	// Build the cumulative probability table once, then binary-search it
	// per draw. The final total doubles as the normalization check: the
	// negated comparison rejects a NaN sum too, since NaN compares false
	// against any tolerance.
	cumulative := make([]float64, size)
	total := 0.0
	for i := 0; i < size; i++ {
		total += Probability(s.Amplitude(i))
		cumulative[i] = total
	}
	if !IsNormalizedSum(total) {
		return nil, &UnnormalizedStateError{Sum: total}
	}

	counts := make(map[string]int, shots)
	for shot := 0; shot < shots; shot++ {
		var r float64
		if rng != nil {
			r = rng.Float64()
		} else {
			r = rand.Float64()
		}

		// First cumulative entry strictly above the draw. Strictness keeps a
		// zero-probability bucket from ever being selected: a draw of 0.0
		// against cumulative[0] == 0 must skip to the first state holding
		// probability. A draw exactly on a boundary falls to the later
		// outcome; the size clamp is paranoia against total landing epsilon
		// below a final draw.
		idx := sort.Search(size, func(i int) bool { return cumulative[i] > r })
		if idx >= size {
			idx = size - 1
		}
		counts[fmt.Sprintf("%0*b", numQubits, idx)]++
	}
	return counts, nil
}

// isNilQuantumState reports whether s is nil or a typed nil — a non-nil
// interface carrying a nil pointer, whose methods would panic if called.
func isNilQuantumState(s QuantumState) bool {
	if s == nil {
		return true
	}
	v := reflect.ValueOf(s)
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func:
		return v.IsNil()
	default:
		return false
	}
}
