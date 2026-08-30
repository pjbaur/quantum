package algorithm

import (
	"math"

	"github.com/pjbaur/quantum/parameterized"
	"github.com/pjbaur/quantum/state"
)

// evaluate binds params, executes on a fresh dense state, and returns the
// exact energy. One variational cost unit.
func evaluate(h *Hamiltonian, t *parameterized.Template, params parameterized.Params) (float64, error) {
	c, err := t.Bind(params)
	if err != nil {
		return 0, err
	}
	s, err := state.New(t.NumQubits())
	if err != nil {
		return 0, err
	}
	if err := c.Execute(s); err != nil {
		return 0, err
	}
	return h.Energy(s)
}

// parameterShiftGradient computes dE/dtheta per parameter via the exact
// parameter-shift rule: dE/dtheta = (E(theta+pi/2) - E(theta-pi/2)) / 2.
// Valid because every parameterized factory is a generators-of-Pauli
// rotation (Rx/Ry/Rz/Phase in parameterized all are). Returns the gradient
// keyed by name plus the number of energy evaluations consumed.
func parameterShiftGradient(h *Hamiltonian, t *parameterized.Template, params parameterized.Params, names []string) (map[string]float64, int, error) {
	grad := make(map[string]float64, len(names))
	evals := 0
	for _, name := range names {
		plus := parameterized.Params{}
		minus := parameterized.Params{}
		for k, v := range params {
			plus[k] = v
			minus[k] = v
		}
		plus[name] += math.Pi / 2
		minus[name] -= math.Pi / 2
		ePlus, err := evaluate(h, t, plus)
		if err != nil {
			return nil, evals, err
		}
		eMinus, err := evaluate(h, t, minus)
		if err != nil {
			return nil, evals, err
		}
		evals += 2
		grad[name] = (ePlus - eMinus) / 2
	}
	return grad, evals, nil
}
