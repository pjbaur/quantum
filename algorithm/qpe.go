package algorithm

import (
	"fmt"
	"math"

	"github.com/pjbaur/quantum/circuit"
	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/quantum"
	"github.com/pjbaur/quantum/state"
)

// qftOp is one gate of the subregister QFT decomposition: a Hadamard, a
// controlled-phase, or a swap.
type qftOp struct {
	kind   int // 0 Hadamard, 1 controlled-phase, 2 swap
	ctrl   int
	target int
	angle  float64 // controlled-phase only
}

// subregisterQFTOps returns the gate sequence implementing the QFT on
// qubits 0..n-1 (qubit 0 = LSB) with quantum.QFT's index convention
// F|x> = (1/sqrt(N)) sum_y e^{2*pi*i*x*y/N} |y>: the Nielsen & Chuang
// circuit (Fig. 5.1, H then controlled-R_k chain per qubit, output-order
// swaps) translated from their 1-based MSB-first numbering, where their
// qubit k is our qubit n-k. The swaps at the end are what make the basis
// index map directly onto the register.
func subregisterQFTOps(n int) []qftOp {
	var ops []qftOp
	for q := n - 1; q >= 0; q-- {
		ops = append(ops, qftOp{kind: 0, target: q})
		for ctrl := q - 1; ctrl >= 0; ctrl-- {
			ops = append(ops, qftOp{
				kind:   1,
				ctrl:   ctrl,
				target: q,
				angle:  math.Pi / math.Pow(2, float64(q-ctrl)),
			})
		}
	}
	for q := 0; q < n/2; q++ {
		ops = append(ops, qftOp{kind: 2, ctrl: q, target: n - 1 - q})
	}
	return ops
}

// appendInverseQFTSub appends the inverse QFT on qubits 0..n-1 of c: the
// forward decomposition run backwards with conjugated (negated) phase
// angles. quantum.InverseQFT cannot be used inside phase estimation — it
// DFTs the entire statevector, which would mix the eigenstate qubit into
// the counting register — so the counting subregister gets the gate-level
// form instead.
func appendInverseQFTSub(c *circuit.Circuit, n int) error {
	ops := subregisterQFTOps(n)
	for i := len(ops) - 1; i >= 0; i-- {
		op := ops[i]
		switch op.kind {
		case 0:
			if err := c.AddGate(gates.NewHadamard(), op.target); err != nil {
				return err
			}
		case 1:
			cp, err := gates.NewControlled(gates.NewPhase(-op.angle))
			if err != nil {
				return err
			}
			if err := c.AddGate(cp, op.ctrl, op.target); err != nil {
				return err
			}
		case 2:
			if err := c.AddGate(gates.NewSwap(), op.ctrl, op.target); err != nil {
				return err
			}
		}
	}
	return nil
}

// InvalidQPEInputError indicates a malformed phase-estimation invocation.
type InvalidQPEInputError struct {
	Reason string
}

func (e *InvalidQPEInputError) Error() string {
	return "invalid phase estimation input: " + e.Reason
}

// PhaseProbabilities runs the phase-estimation circuit for the unitary u on
// its 1-qubit eigenstate and returns P(counting register = m) for
// m = 0..2^numCounting-1. Register layout: counting qubits 0..numCounting-1
// (qubit 0 = LSB), eigenstate as qubit numCounting. The circuit is the
// textbook one: Hadamards on the counting register, controlled-u applied
// 2^j times with counting qubit j as control (repetition rather than gate
// powers — no matrix-power machinery, 2^numCounting-1 applications, trivial
// for numCounting <= 4, and it works for any unitary), then the gate-level
// inverse QFT on the counting subregister.
//
// For an exact eigenvector the eigenstate qubit never entangles with the
// counting register, so the probabilities come straight off the counting
// basis states: P(m) = Probability(m) + Probability(m | eigenstate bit).
func PhaseProbabilities(u quantum.Gate, eigenstate quantum.QuantumState, numCounting int) ([]float64, error) {
	if u == nil {
		return nil, &InvalidQPEInputError{Reason: "gate must not be nil"}
	}
	if eigenstate == nil {
		return nil, &InvalidQPEInputError{Reason: "eigenstate must not be nil"}
	}
	if eigenstate.NumQubits() != 1 {
		return nil, &InvalidQPEInputError{Reason: fmt.Sprintf("eigenstate must have 1 qubit, got %d", eigenstate.NumQubits())}
	}
	if numCounting < 1 {
		return nil, &InvalidQPEInputError{Reason: fmt.Sprintf("numCounting must be at least 1, got %d", numCounting)}
	}

	total := numCounting + 1
	sRaw, err := state.New(total)
	if err != nil {
		return nil, err
	}
	var s quantum.QuantumState = sRaw
	// Embed the eigenstate as the top qubit in one bulk write: SetAmplitude
	// would reject each intermediate vector as unnormalized.
	setter, ok := s.(quantum.BulkAmplitudeSetter)
	if !ok {
		return nil, &quantum.UnsupportedOperationError{
			Operation:   "phase estimation",
			Backend:     fmt.Sprintf("%T", s),
			Alternative: "a backend implementing BulkAmplitudeSetter (dense or sparse state)",
		}
	}
	amplitudes := make([]complex128, 1<<total)
	amplitudes[0] = eigenstate.Amplitude(0)
	amplitudes[1<<numCounting] = eigenstate.Amplitude(1)
	if err := setter.SetAmplitudes(amplitudes); err != nil {
		return nil, err
	}

	c, err := circuit.New(total)
	if err != nil {
		return nil, err
	}
	for j := 0; j < numCounting; j++ {
		if err := c.AddGate(gates.NewHadamard(), j); err != nil {
			return nil, err
		}
	}
	cu, err := gates.NewControlled(u)
	if err != nil {
		return nil, err
	}
	for j := 0; j < numCounting; j++ {
		for k := 0; k < 1<<j; k++ {
			// Control first, then target: NewControlled documents control as
			// targets[0], matching CNOT's convention.
			if err := c.AddGate(cu, j, total-1); err != nil {
				return nil, err
			}
		}
	}
	if err := appendInverseQFTSub(c, numCounting); err != nil {
		return nil, err
	}
	if err := c.Execute(s); err != nil {
		return nil, err
	}

	probs := make([]float64, 1<<numCounting)
	eigenBit := 1 << numCounting
	for m := range probs {
		probs[m] = s.Probability(m) + s.Probability(m|eigenBit)
	}
	return probs, nil
}

// EstimatePhase returns the argmax counting outcome and its interpretation
// phaseTurns = bestCount / 2^numCounting of the distribution
// PhaseProbabilities produces. When the eigenphase is exactly representable
// in numCounting bits the peak has probability 1 and the readout is exact.
func EstimatePhase(u quantum.Gate, eigenstate quantum.QuantumState, numCounting int) (int, float64, error) {
	probs, err := PhaseProbabilities(u, eigenstate, numCounting)
	if err != nil {
		return 0, 0, err
	}
	best := 0
	for m, p := range probs {
		if p > probs[best] {
			best = m
		}
	}
	return best, float64(best) / float64(uint(1)<<uint(numCounting)), nil
}
