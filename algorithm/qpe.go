package algorithm

import (
	"math"

	"github.com/pjbaur/quantum/circuit"
	"github.com/pjbaur/quantum/gates"
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
				angle:  2 * math.Pi / math.Pow(2, float64(q-ctrl)),
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
