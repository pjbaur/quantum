package gates

import "github.com/pjbaur/quantum/quantum"

// DecomposedGate represents a gate and its target qubits.
type DecomposedGate struct {
	Gate    quantum.Gate
	Targets []int
}

// DecomposeSwap returns a CNOT-based decomposition of a SWAP gate.
func DecomposeSwap(first, second int) []DecomposedGate {
	return []DecomposedGate{
		{Gate: NewCNOT(), Targets: []int{first, second}},
		{Gate: NewCNOT(), Targets: []int{second, first}},
		{Gate: NewCNOT(), Targets: []int{first, second}},
	}
}
