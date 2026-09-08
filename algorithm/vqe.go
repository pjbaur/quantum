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
// The rule is exact only when, with every other parameter held fixed,
// E(theta) = a + b*cos(theta) + c*sin(theta). That shape needs two things
// from the template, and this function checks neither.
//
// First, each parameter must drive exactly one template step. Shifting a
// name that feeds several gates moves all of them at once, adding higher
// harmonics (cos(k*theta)) to which the +/- pi/2 difference is blind. VQE
// enforces this up front through parameterized.Template.ParamStepCounts,
// so such a template never reaches here.
//
// Second, the bound value must enter its gate as exp(-i*theta*P/2) with P a
// Pauli (eigenvalues +/-1): the convention of gates.NewRx/NewRy/NewRz and
// hence of the parameterized.Rx/Ry/Rz/Phase factories (Phase is Rz up to a
// global phase, which expectation values ignore). A parameterized.Factory
// is an arbitrary func(value) Gate, though, so a caller can break this by
// rescaling inside the factory. A factory returning gates.NewRz(2*value)
// makes E periodic in the bound value with period pi, so E(theta+pi/2)
// equals E(theta-pi/2) at every theta and this function returns an
// identically zero gradient while the true dE/dtheta need not be zero.
// Neither this function nor VQE can detect that: the template exposes only
// the opaque factory, and a vanishing shift difference is indistinguishable
// from a genuine stationary point, so VQE takes a zero step and reports
// Converged at its starting parameters. Factories must bind the value
// unscaled and leave any textbook rescaling to how callers read the
// parameter, as QAOATemplate does (its bound value is the Rz/Rx angle;
// textbook gamma/beta is half of it).
// TestParameterShiftIsBlindToRescaledFactory pins this failure mode.
//
// Returns the gradient keyed by name plus the number of energy evaluations
// consumed.
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

// VQEOptions configures the VQE loop. Zero values select defaults. Values
// outside a field's domain are rejected with InvalidVQEInputError rather
// than run with: a negative StepSize climbs, a negative or NaN Tolerance
// can never be met, and a non-finite value reaches every parameter
// through the first update.
type VQEOptions struct {
	// InitialParams names starting angles. Missing declared parameters
	// default to 0. Unknown names and non-finite values are rejected.
	InitialParams parameterized.Params
	// StepSize is the initial gradient-descent step (default 0.3). Must
	// be finite and positive; zero selects the default.
	StepSize float64
	// MaxIterations caps the loop (default 200). Must not be negative;
	// zero selects the default.
	MaxIterations int
	// Tolerance is the convergence threshold on |delta E| between
	// consecutive iterations (default 1e-10). Must be finite and
	// positive; zero selects the default.
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

// InvalidVQEInputError indicates a malformed VQE invocation: an option
// outside its domain, a template or Hamiltonian VQE cannot optimize, or
// an evaluation failure that traces back to one of them. Reason is the
// complete message. Err is set only when another package detected the
// problem during an evaluation (parameterized, circuit, quantum); Unwrap
// exposes it so errors.As still reaches that package's type, the same
// contract fmt.Errorf's %w gives callers elsewhere in this module
// (gates/matrix.go, circuit/parallel.go).
type InvalidVQEInputError struct {
	Reason string
	Err    error
}

func (e *InvalidVQEInputError) Error() string {
	return "invalid VQE input: " + e.Reason
}

// Unwrap returns the evaluation error this input error carries, or nil
// for problems VQE detected itself before the first evaluation.
func (e *InvalidVQEInputError) Unwrap() error { return e.Err }

// isFinite reports whether v is neither NaN nor infinite.
func isFinite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

// validateVQEOptions checks the scalar option domains. Zero is the
// "use the default" sentinel for every field, so the checks reject only
// values that are neither zero nor usable: a negative StepSize ascends
// (and the loop's 1e-6 floor would then silently clamp it positive), a
// negative MaxIterations is meaningless, a negative or NaN Tolerance can
// never be met, and a non-finite StepSize or Tolerance turns the first
// update or the convergence test into NaN arithmetic. InitialParams needs
// the template's declared names and is checked in VQE itself.
func validateVQEOptions(opts VQEOptions) error {
	if !isFinite(opts.StepSize) || opts.StepSize < 0 {
		return &InvalidVQEInputError{Reason: fmt.Sprintf("StepSize must be finite and positive (zero selects the default 0.3), got %v", opts.StepSize)}
	}
	if opts.MaxIterations < 0 {
		return &InvalidVQEInputError{Reason: fmt.Sprintf("MaxIterations must not be negative (zero selects the default 200), got %d", opts.MaxIterations)}
	}
	if !isFinite(opts.Tolerance) || opts.Tolerance < 0 {
		return &InvalidVQEInputError{Reason: fmt.Sprintf("Tolerance must be finite and positive (zero selects the default 1e-10), got %v", opts.Tolerance)}
	}
	return nil
}

// validateVQEStructure checks that the template and the Hamiltonian can
// be optimized against each other before any evaluation is paid for.
// The one-gate-per-parameter rule is the template-side precondition of
// the parameter-shift gradient (see parameterShiftGradient). A term whose
// Pauli string is not the register's length would fail inside
// quantum.Expectation on the first evaluation as
// IncompatibleQubitCountError; an identity term (no axes) is exempt
// because Energy adds its coefficient without consulting the state. A
// non-finite coefficient would flow through Energy into a non-finite
// gradient and then a non-finite parameter that parameterized.Bind
// rejects by name, blaming a value the caller supplied finite. All three
// are properties of the inputs, so all are InvalidVQEInputError here.
//
// Axis validity (each axis one of PauliI..PauliZ) is deliberately left to
// quantum.Expectation, which owns that domain and reports
// InvalidPauliAxisError; VQE wraps it at the first evaluation rather than
// duplicating the enum's range.
func validateVQEStructure(h *Hamiltonian, t *parameterized.Template) error {
	stepCounts := t.ParamStepCounts()
	for _, name := range t.ParamNames() {
		if count := stepCounts[name]; count > 1 {
			return &InvalidVQEInputError{Reason: fmt.Sprintf("parameter %q drives %d template steps; the parameter-shift gradient requires exactly one gate per parameter", name, count)}
		}
	}
	for i, term := range h.terms {
		if !isFinite(term.Coeff) {
			return &InvalidVQEInputError{Reason: fmt.Sprintf("Hamiltonian term %d has non-finite coefficient %v", i, term.Coeff)}
		}
		if len(term.Axes) != 0 && len(term.Axes) != t.NumQubits() {
			return &InvalidVQEInputError{Reason: fmt.Sprintf("Hamiltonian term %d has %d Pauli axes but the template has %d qubits", i, len(term.Axes), t.NumQubits())}
		}
	}
	return nil
}

// wrapEvaluationError converts an error from evaluate or
// parameterShiftGradient into InvalidVQEInputError. By the time an
// evaluation runs, VQE has checked every option, every initial parameter,
// the template's parameter structure, and the Hamiltonian's terms, and it
// hands each evaluation a complete, declared, finite parameter set. What
// can still fail is what only running the template reveals: a factory
// returning a gate of the wrong width or with a malformed matrix, or a
// Pauli axis outside the enum. Those are input properties, so the caller
// sees the VQE type, with the detecting package's error kept in Err.
func wrapEvaluationError(where string, err error) error {
	return &InvalidVQEInputError{Reason: where + " failed: " + err.Error(), Err: err}
}

// VQE minimizes <psi(params)|H|psi(params)> with parameter-shift gradients
// and plain descent. Each iteration binds, executes on a fresh dense state,
// evaluates exactly, shifts every parameter, and steps. A step that raises
// the energy reverts the parameters and halves the step size (floor 1e-6).
// The loop stops when |delta E| < Tolerance (Converged) or MaxIterations.
//
// Converged means only that the last accepted step changed the energy by
// less than Tolerance; it does not certify a minimum. Plain descent cannot
// tell a minimum from any other stationary point: started exactly at an
// energy maximum or a saddle, where the gradient vanishes, VQE takes a
// zero step and reports Converged there. Zero is zero up to floating-point
// rounding: a shift difference of two energies that agree to the last bit
// is at most a few ulp, so the reported parameters and energy may differ
// from the start in their last bits, and a one-ulp energy rise can cost a
// rejected iteration (counted in Evaluations, not Iterations) before the
// no-op step is accepted. A template with no parameters is accepted too:
// there is nothing to optimize, so VQE returns the fixed circuit's energy
// after one no-op iteration and reports Converged.
//
// The template must drive exactly one gate per parameter: the parameter
// shift moves every occurrence of a name at once, which breaks the
// two-eigenvalue shift rule when a name feeds several gates. Templates
// violating that precondition are rejected with InvalidVQEInputError rather
// than optimized against a silently wrong (near-zero) gradient.
//
// The template must also bind each parameter as the angle of an
// exp(-i*theta*P/2) rotation, the gates.NewRx/NewRy/NewRz convention that
// the parameterized.Rx/Ry/Rz/Phase factories follow. VQE cannot check
// this: a factory that rescales the value (gates.NewRz(2*value)) yields a
// gradient that is identically zero, so VQE takes no step and reports
// Converged at its starting parameters, indistinguishable from a genuine
// stationary point. See parameterShiftGradient.
//
// Inputs are validated before the first evaluation: nil arguments, option
// values outside their domains (see VQEOptions), initial parameters that
// are undeclared or non-finite, a parameter driving several gates, a
// Hamiltonian term whose Pauli string does not match the template's qubit
// count, and a non-finite Hamiltonian coefficient are all rejected with
// InvalidVQEInputError. Problems only running the template can reveal,
// such as a factory returning a gate wider than its target list, surface
// from the first evaluation and are wrapped in the same type with the
// detecting package's error reachable through errors.As. Coefficients
// that are each finite but whose sum overflows float64 give a non-finite
// gradient, also reported as InvalidVQEInputError; no error names a
// parameter that was finite on entry.
func VQE(h *Hamiltonian, t *parameterized.Template, opts VQEOptions) (*VQEResult, error) {
	if h == nil {
		return nil, &InvalidVQEInputError{Reason: "Hamiltonian must not be nil"}
	}
	if t == nil {
		return nil, &InvalidVQEInputError{Reason: "template must not be nil"}
	}
	if err := validateVQEOptions(opts); err != nil {
		return nil, err
	}
	if err := validateVQEStructure(h, t); err != nil {
		return nil, err
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
		if !isFinite(value) {
			return nil, &InvalidVQEInputError{Reason: fmt.Sprintf("initial parameter %q has non-finite value %v", name, value)}
		}
		params[name] = value
	}

	evals := 0
	energy, err := evaluate(h, t, params)
	if err != nil {
		return nil, wrapEvaluationError("initial energy evaluation", err)
	}
	evals++

	converged := false
	accepted := 0
	for iter := 0; iter < maxIter; iter++ {
		grad, gradEvals, err := parameterShiftGradient(h, t, params, names)
		if err != nil {
			return nil, wrapEvaluationError(fmt.Sprintf("gradient evaluation at iteration %d", iter), err)
		}
		evals += gradEvals
		// Coefficients are finite (validateVQEStructure) and every Pauli
		// expectation is bounded by 1, so a non-finite gradient means the
		// energy sum overflowed float64. Stop here: the step would carry a
		// non-finite value into a parameter, and Bind would then blame that
		// parameter although the caller supplied it finite.
		for _, name := range names {
			if !isFinite(grad[name]) {
				return nil, &InvalidVQEInputError{Reason: fmt.Sprintf("gradient of parameter %q is non-finite (%v) at iteration %d: the Hamiltonian's energy overflows float64", name, grad[name], iter)}
			}
		}

		steps := parameterized.Params{}
		for _, name := range names {
			steps[name] = params[name] - step*grad[name]
		}
		newEnergy, err := evaluate(h, t, steps)
		if err != nil {
			return nil, wrapEvaluationError(fmt.Sprintf("step evaluation at iteration %d", iter), err)
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
