package algorithm

import (
	"fmt"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/parameterized"
	"github.com/pjbaur/quantum/quantum"
)

// InvalidQAOAInputError indicates a malformed QAOA graph or template.
type InvalidQAOAInputError struct {
	Reason string
}

func (e *InvalidQAOAInputError) Error() string {
	return "invalid QAOA input: " + e.Reason
}

// normalizeEdges validates the edge list and returns every edge ordered as
// (min, max), rejecting self-loops, out-of-range vertices, and duplicates
// (duplicates would double-count one interaction in both H and the ansatz).
func normalizeEdges(numQubits int, edges [][2]int) ([][2]int, error) {
	if numQubits < 2 {
		return nil, &InvalidQAOAInputError{Reason: fmt.Sprintf("numQubits must be at least 2, got %d", numQubits)}
	}
	if len(edges) == 0 {
		return nil, &InvalidQAOAInputError{Reason: "edge list must not be empty"}
	}
	seen := make(map[[2]int]bool, len(edges))
	normalized := make([][2]int, 0, len(edges))
	for _, e := range edges {
		if e[0] == e[1] {
			return nil, &InvalidQAOAInputError{Reason: fmt.Sprintf("self-loop on qubit %d", e[0])}
		}
		if e[0] < 0 || e[0] >= numQubits || e[1] < 0 || e[1] >= numQubits {
			return nil, &InvalidQAOAInputError{Reason: fmt.Sprintf("edge (%d, %d) out of range for %d qubits", e[0], e[1], numQubits)}
		}
		if e[0] > e[1] {
			e = [2]int{e[1], e[0]}
		}
		if seen[e] {
			return nil, &InvalidQAOAInputError{Reason: fmt.Sprintf("duplicate edge (%d, %d)", e[0], e[1])}
		}
		seen[e] = true
		normalized = append(normalized, e)
	}
	return normalized, nil
}

// MaxCutHamiltonian returns the MaxCut cost operator
//
//	H = sum_e w_e * Z_i Z_j
//
// as a Pauli-string Hamiltonian. Because <Z_i Z_j> = +1 when the endpoints
// agree and -1 when they differ, minimizing <H> maximizes the cut:
// expected cut = (W - <H>) / 2 with W = sum of weights. A nil weights slice
// means unit weights.
func MaxCutHamiltonian(numQubits int, edges [][2]int, weights []float64) (*Hamiltonian, error) {
	if weights != nil && len(weights) != len(edges) {
		return nil, &InvalidQAOAInputError{Reason: fmt.Sprintf("got %d weights for %d edges", len(weights), len(edges))}
	}
	normalized, err := normalizeEdges(numQubits, edges)
	if err != nil {
		return nil, err
	}
	h := NewHamiltonian()
	for i, e := range normalized {
		w := 1.0
		if weights != nil {
			w = weights[i]
		}
		// quantum.Expectation applies axes[k] to qubit k, so the Pauli
		// string is built with PauliZ at each endpoint and I elsewhere.
		axes := make([]quantum.PauliAxis, numQubits)
		for a := range axes {
			axes[a] = quantum.PauliI
		}
		axes[e[0]] = quantum.PauliZ
		axes[e[1]] = quantum.PauliZ
		h.AddTerm(w, axes...)
	}
	return h, nil
}

// QAOATemplate returns the QAOA ansatz for the graph on numQubits qubits
// with the given number of layers, starting from |+...+> (Hadamards are the
// template's fixed gates). Each layer applies, per edge (i, j), the cost
// term e^(-i*gamma*w*Z_i Z_j) as CNOT(i,j) Rz(2*gamma) CNOT(i,j) — the
// standard identity, with Rz(theta) = e^(-i*theta*Z/2) — followed by the
// mixer e^(-i*beta*X_q) = Rx(2*beta) per qubit. Weights are unit; weighted
// cost Hamiltonians can drive VQE but the template builder is unweighted.
//
// Parameters are per edge and per qubit (gamma_e<k>, beta_q<k>, suffixed
// _l<layer> when layers > 1) because the VQE driver requires exactly one
// gate per parameter: a shared gamma across edges would move several gates
// at once and break the parameter-shift gradient.
func QAOATemplate(numQubits int, edges [][2]int, layers int) (*parameterized.Template, error) {
	if layers < 1 {
		return nil, &InvalidQAOAInputError{Reason: fmt.Sprintf("layers must be at least 1, got %d", layers)}
	}
	normalized, err := normalizeEdges(numQubits, edges)
	if err != nil {
		return nil, err
	}
	t := parameterized.NewTemplate(numQubits)
	for q := 0; q < numQubits; q++ {
		if err := t.AddGate(gates.NewHadamard(), q); err != nil {
			return nil, err
		}
	}
	for l := 0; l < layers; l++ {
		suffix := ""
		if layers > 1 {
			suffix = fmt.Sprintf("_l%d", l)
		}
		for k, e := range normalized {
			gammaName := fmt.Sprintf("gamma_e%d%s", k, suffix)
			// The bound value is the rotation angle: e^(-i*gamma*Z_i Z_j)
			// needs Rz(2*gamma) between the CNOTs.
			costFactory := func(value float64) quantum.Gate { return gates.NewRz(2 * value) }
			if err := t.AddGate(gates.NewCNOT(), e[0], e[1]); err != nil {
				return nil, err
			}
			if err := t.AddParamGate(gammaName, costFactory, e[1]); err != nil {
				return nil, err
			}
			if err := t.AddGate(gates.NewCNOT(), e[0], e[1]); err != nil {
				return nil, err
			}
		}
		for q := 0; q < numQubits; q++ {
			betaName := fmt.Sprintf("beta_q%d%s", q, suffix)
			mixFactory := func(value float64) quantum.Gate { return gates.NewRx(2 * value) }
			if err := t.AddParamGate(betaName, mixFactory, q); err != nil {
				return nil, err
			}
		}
	}
	return t, nil
}

// CutOfBitstring evaluates the classical cut value of one outcome:
// the total weight of the edges whose endpoints differ. bits[k] is the
// measured value of qubit k.
func CutOfBitstring(edges [][2]int, weights []float64, bits []int) (float64, error) {
	if weights != nil && len(weights) != len(edges) {
		return 0, &InvalidQAOAInputError{Reason: fmt.Sprintf("got %d weights for %d edges", len(weights), len(edges))}
	}
	cut := 0.0
	for i, e := range edges {
		if e[0] >= len(bits) || e[1] >= len(bits) {
			return 0, &InvalidQAOAInputError{Reason: fmt.Sprintf("edge (%d, %d) needs more than %d bits", e[0], e[1], len(bits))}
		}
		if bits[e[0]] != 0 && bits[e[0]] != 1 || bits[e[1]] != 0 && bits[e[1]] != 1 {
			return 0, &InvalidQAOAInputError{Reason: fmt.Sprintf("bits must be 0 or 1, edge (%d, %d)", e[0], e[1])}
		}
		if bits[e[0]] != bits[e[1]] {
			w := 1.0
			if weights != nil {
				w = weights[i]
			}
			cut += w
		}
	}
	return cut, nil
}

// ExpectedCut converts a cost-Hamiltonian energy into the expected MaxCut
// value: (totalWeight - energy) / 2.
func ExpectedCut(totalWeight, energy float64) float64 {
	return (totalWeight - energy) / 2
}
