package algorithm

import (
	"fmt"
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

// VQEOptions configures the VQE loop. Zero values select defaults.
type VQEOptions struct {
	// InitialParams names starting angles. Missing declared parameters
	// default to 0. Unknown names are rejected.
	InitialParams parameterized.Params
	// StepSize is the initial gradient-descent step (default 0.3).
	StepSize float64
	// MaxIterations caps the loop (default 200).
	MaxIterations int
	// Tolerance is the convergence threshold on |delta E| between
	// consecutive iterations (default 1e-10).
	Tolerance float64
}

// VQEResult reports the outcome of a VQE run.
type VQEResult struct {
	Energy      float64
	Params      parameterized.Params
	Iterations  int
	Evaluations int
	Converged   bool
}

// InvalidVQEInputError indicates a malformed VQE invocation.
type InvalidVQEInputError struct {
	Reason string
}

func (e *InvalidVQEInputError) Error() string {
	return "invalid VQE input: " + e.Reason
}

// VQE minimizes <psi(params)|H|psi(params)> with parameter-shift gradients
// and plain descent. Each iteration binds, executes on a fresh dense state,
// evaluates exactly, shifts every parameter, and steps. A step that raises
// the energy reverts the parameters and halves the step size (floor 1e-6).
// The loop stops when |delta E| < Tolerance (Converged) or MaxIterations.
func VQE(h *Hamiltonian, t *parameterized.Template, opts VQEOptions) (*VQEResult, error) {
	if h == nil {
		return nil, &InvalidVQEInputError{Reason: "Hamiltonian must not be nil"}
	}
	if t == nil {
		return nil, &InvalidVQEInputError{Reason: "template must not be nil"}
	}

	step := opts.StepSize
	if step == 0 {
		step = 0.3
	}
	maxIter := opts.MaxIterations
	if maxIter == 0 {
		maxIter = 200
	}
	tol := opts.Tolerance
	if tol == 0 {
		tol = 1e-10
	}

	names := t.ParamNames()
	params := parameterized.Params{}
	for _, name := range names {
		params[name] = 0
	}
	for name, value := range opts.InitialParams {
		if _, ok := params[name]; !ok {
			return nil, &InvalidVQEInputError{Reason: fmt.Sprintf("initial parameter %q is not declared in the template", name)}
		}
		params[name] = value
	}

	evals := 0
	energy, err := evaluate(h, t, params)
	if err != nil {
		return nil, err
	}
	evals++

	converged := false
	accepted := 0
	for iter := 0; iter < maxIter; iter++ {
		grad, gradEvals, err := parameterShiftGradient(h, t, params, names)
		if err != nil {
			return nil, err
		}
		evals += gradEvals

		steps := parameterized.Params{}
		for _, name := range names {
			steps[name] = params[name] - step*grad[name]
		}
		newEnergy, err := evaluate(h, t, steps)
		if err != nil {
			return nil, err
		}
		evals++

		if newEnergy > energy {
			// Revert; shrink the step and retry next iteration.
			step /= 2
			if step < 1e-6 {
				step = 1e-6
			}
			continue
		}

		accepted++
		delta := math.Abs(newEnergy - energy)
		params, energy = steps, newEnergy
		if delta < tol {
			converged = true
			break
		}
	}

	return &VQEResult{
		Energy:      energy,
		Params:      params,
		Iterations:  accepted,
		Evaluations: evals,
		Converged:   converged,
	}, nil
}
