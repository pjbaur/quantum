package visualization

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pjbaur/quantum/quantum"
)

// StateViewOptions controls how state views are rendered.
type StateViewOptions struct {
	MinProbability float64
	Precision      int
	IncludeHeader  bool
}

// DefaultStateViewOptions returns a baseline configuration for state views.
func DefaultStateViewOptions() StateViewOptions {
	return StateViewOptions{
		MinProbability: 0,
		Precision:      4,
		IncludeHeader:  true,
	}
}

// FormatStateView renders a text table showing basis index, amplitude, and probability.
func FormatStateView(s quantum.QuantumState, opts StateViewOptions) string {
	if s == nil {
		return ""
	}
	precision := opts.Precision
	if precision <= 0 {
		precision = DefaultStateViewOptions().Precision
	}
	minProb := opts.MinProbability
	if minProb < 0 {
		minProb = 0
	}

	numQubits := s.NumQubits()
	numStates := 1 << uint(numQubits)
	indexWidth := len(strconv.Itoa(numStates - 1))
	basisWidth := numQubits + 2

	var b strings.Builder
	if opts.IncludeHeader {
		fmt.Fprintf(&b, "%*s  %-*s  %s  %s\n", indexWidth, "idx", basisWidth, "basis", "amplitude", "probability")
	}

	for i := 0; i < numStates; i++ {
		prob := s.Probability(i)
		if prob < minProb {
			continue
		}
		amplitude := s.Amplitude(i)
		basis := fmt.Sprintf("|%0*b>", numQubits, i)
		ampText := formatComplex(amplitude, precision)
		probText := fmt.Sprintf("%.*f", precision, prob)
		fmt.Fprintf(&b, "%*d  %-*s  %s  %s\n", indexWidth, i, basisWidth, basis, ampText, probText)
	}

	return strings.TrimRight(b.String(), "\n")
}

func formatComplex(value complex128, precision int) string {
	return fmt.Sprintf("%.*f%+.*fi", precision, real(value), precision, imag(value))
}
