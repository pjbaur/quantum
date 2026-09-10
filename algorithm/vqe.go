package algorithm

import (
	"fmt"
	"math"
	"strings"

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

// maxShiftMagnitude is the largest |theta| the parameter-shift rule
// shifts: 2^26. float64 rounds theta +/- pi/2 to a multiple of theta's
// spacing, so the shift lands within half that spacing of its true
// offset. Below 2^26, theta +/- pi/2 crosses into the next power-of-two
// range for any theta within pi/2 of it, so the shifted value's spacing
// is at most 2^-26 (1.49e-8), the offset is exact to 7.5e-9 rad, and a
// gradient component is off by at most that times the largest slope,
// which the Hamiltonian's coefficient sum bounds. Beyond it the rule
// degrades until it fails: above 2^48 the default 0.3 step times a
// gradient of 0.1 rounds to no change, so VQE would freeze the parameter
// and report Converged; from 2^51 the shift is applied at the wrong
// offset (a multiple of 0.5 or coarser); above 2^54 the spacing exceeds
// pi, theta +/- pi/2 rounds back to theta, both evaluations bind the
// same circuit, and the difference is exactly 0. The
// bound is the even split of the 53-bit significand, 26 bits before the
// binary point and 27 after: the parameter keeps a resolution finer than
// 1e-8 rad, and 2^26 rad is more than 10^7 turns, far beyond any angle a
// descent reaches from an angle a caller chose (200 iterations of a 0.3
// step times a unit gradient walk 60 rad). Larger angles are rejected,
// not reduced mod 2*pi: see parameterShiftGradient.
const maxShiftMagnitude = 1 << 26

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
// Third, float64 must resolve the shift. theta +/- pi/2 is rounded to a
// multiple of theta's spacing, and above 2^54 that spacing exceeds pi, so
// both shifted values round back to theta, the two evaluations bind the
// same circuit, and the half difference is exactly 0 with a nil error,
// again indistinguishable from a stationary point. This one is checked:
// each name in names must be bound to a finite value of magnitude at most
// maxShiftMagnitude (2^26, where the shift lands within 7.5e-9 rad of its
// offset; the constant's comment derives the bound), or the call returns
// InvalidVQEInputError naming the first such name in names order, before
// any evaluation and with the count 0. Only shifted names are checked,
// because an unshifted value enters both evaluations identically, and
// only finite values, because a non-finite one is Bind's to reject as
// before. A larger angle is rejected rather than reduced mod 2*pi. The
// energy is 2*pi-periodic only under the previous paragraph's
// precondition, which this function cannot check, so a reduction would
// silently move the point evaluated for a factory of another period; and
// the reduction itself exceeds float64 (math.Mod(2^60, 2*pi) with the
// float64 constant accumulates about 45 rad of error, returning 5.0824
// where the true reduction, at 256-bit precision, is 4.1219). A caller
// who knows the factory's period reduces before calling, and VQE rejects
// such an initial parameter up front so this check is unreachable from
// it.
//
// params must bind every parameter the template declares, the precondition
// Bind states; names selects which of those to differentiate and may list
// any subset in any order, and repeats: the loop shifts and counts each
// element of names as it is reached, not each unique name, so a name
// listed twice runs its pair of evaluations twice while grad still ends
// up with one entry for it, the last occurrence's slope. VQE never
// repeats a name; it calls with t.ParamNames(), which returns declared
// names without duplicates. Completeness is checked here rather than left
// to Bind because the shift would hide the gap: a name in names that
// params lacks reads as 0 from the map, so the plus and minus copies carry
// it at +/- pi/2, Bind sees a complete binding, and the slope at an
// implicit 0 comes back with a nil error, whereas a missing name that is
// not shifted stays absent and Bind rejects it, so whether the call failed
// depended on which names were shifted. The check reports the first
// missing name in declaration order as parameterized.MissingParameterError,
// the error Bind returns when a declared name is absent, and consumes no
// evaluation. Of Bind's rules it guarantees completeness only: that is
// the one a shift can hide, since the copies keep every key of params, a
// non-finite value stays non-finite when shifted, and a name in names that
// the template never declared is written into the copies where Bind sees
// it. Everything else is left to Bind, which rejects it at the first
// shifted evaluation that reaches it. So a non-finite value ahead of a
// missing name in declaration order is reported here as the missing name,
// where Bind would report the value; a name in names that the template
// never declared is rejected as UnknownParameterError when its own shift
// is evaluated, after the names before it have cost their evaluations,
// unless its bound value is finite with magnitude above maxShiftMagnitude,
// in which case the earlier magnitude check reports it first, at 0
// evaluations, since that check does not distinguish declared names from
// undeclared ones: it runs over every name in names before either loop
// evaluates anything; and with names empty no evaluation runs and nothing
// beyond completeness is checked.
//
// Returns the gradient keyed by name plus the number of energy evaluations
// consumed: two per name in names. On error the gradient is nil and the
// count is still exact: each evaluation is counted as evaluate returns its
// energy, the rule VQE applies to its own evaluations, so the count covers
// every evaluation that completed before the failure and not the failing
// one, which returned no energy (how far into evaluate it got is not
// something a count of energies can express). Like io.Reader's n, the
// count is meaningful alongside a non-nil error; VQE discards it with the
// error, so only a direct caller sees it. The completeness and magnitude
// checks above return 0 because they precede the first evaluation.
func parameterShiftGradient(h *Hamiltonian, t *parameterized.Template, params parameterized.Params, names []string) (map[string]float64, int, error) {
	for _, name := range t.ParamNames() {
		if _, ok := params[name]; !ok {
			return nil, 0, &parameterized.MissingParameterError{Name: name}
		}
	}
	for _, name := range names {
		if v := params[name]; isFinite(v) && math.Abs(v) > maxShiftMagnitude {
			return nil, 0, &InvalidVQEInputError{Reason: fmt.Sprintf("parameter %q cannot be shifted by +/- pi/2: its magnitude must be at most 2^26 (%v), got %v", name, float64(maxShiftMagnitude), v)}
		}
	}
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
		evals++
		eMinus, err := evaluate(h, t, minus)
		if err != nil {
			return nil, evals, err
		}
		evals++
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
	// default to 0. Unknown names, non-finite values, and magnitudes
	// beyond 2^26 (see maxShiftMagnitude) are rejected.
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
// hands each evaluation a complete, declared, finite parameter set (the
// step guard in VQE keeps an overflowing update, or one beyond the
// parameter-shift bound, from reaching Bind or parameterShiftGradient).
// What can still fail is what only running the template reveals: a factory
// returning a gate of the wrong width or with a malformed matrix, or a
// Pauli axis outside the enum. Those are input properties, so the caller
// sees the VQE type, with the detecting package's error kept in Err.
func wrapEvaluationError(where string, err error) error {
	return &InvalidVQEInputError{Reason: where + " failed: " + err.Error(), Err: err}
}

// formatPoint renders the parameter values VQE evaluated at, in the
// template's declaration order so the text is deterministic, with names
// quoted the way every other Reason quotes them.
func formatPoint(names []string, params parameterized.Params) string {
	parts := make([]string, len(names))
	for i, name := range names {
		parts[i] = fmt.Sprintf("%q=%v", name, params[name])
	}
	return strings.Join(parts, ", ")
}

// checkEnergy rejects a non-finite energy that VQE evaluated at a finite
// point. Coefficients are finite (validateVQEStructure) and every Pauli
// expectation is bounded by 1, so a non-finite energy can only mean the
// term sum overflowed float64: a property of the Hamiltonian, which the
// Reason blames. Stopping here matters twice over: the accept/revert
// comparison cannot order infinities, so an infinite baseline accepts any
// finite step and VQE would report a result it never minimized; and a
// template with no parameters has no gradient for the gradient guard to
// catch, so an overflowing Hamiltonian would return Energy +Inf with a nil
// error. The point is named because a stepped point is not one the caller
// chose or can reconstruct; a template with no parameters has no point to
// name.
func checkEnergy(where string, energy float64, names []string, params parameterized.Params) error {
	if isFinite(energy) {
		return nil
	}
	at := ""
	if len(names) > 0 {
		at = " with " + formatPoint(names, params)
	}
	return &InvalidVQEInputError{Reason: fmt.Sprintf("%s is non-finite (%v)%s: the Hamiltonian's energy overflows float64", where, energy, at)}
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
// are undeclared, non-finite, or beyond 2^26 in magnitude (where float64
// no longer resolves the +/- pi/2 shift; see maxShiftMagnitude, which
// says why such an angle is rejected rather than reduced mod 2*pi), a
// parameter driving several gates, a
// Hamiltonian term whose Pauli string does not match the template's qubit
// count, and a non-finite Hamiltonian coefficient are all rejected with
// InvalidVQEInputError. Problems only running the template can reveal,
// such as a factory returning a gate wider than its target list, surface
// from the first evaluation and are wrapped in the same type with the
// detecting package's error reachable through errors.As. Coefficients
// that are each finite but whose sum overflows float64 give a non-finite
// energy or gradient, and a StepSize large enough that a step overflows
// gives a non-finite parameter; VQE checks each energy it evaluates, each
// gradient component, and each stepped value, and reports all three as
// InvalidVQEInputError blaming the Hamiltonian or StepSize. A stepped
// value that leaves the 2^26 bound is rejected the same way, naming the
// step that crossed it, so the gradient never shifts an angle it cannot
// resolve. No error blames a parameter that was finite on entry.
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
		if math.Abs(value) > maxShiftMagnitude {
			return nil, &InvalidVQEInputError{Reason: fmt.Sprintf("initial parameter %q must have magnitude at most 2^26 (%v), beyond which float64 cannot resolve the +/- pi/2 parameter shift, got %v", name, float64(maxShiftMagnitude), value)}
		}
		params[name] = value
	}

	evals := 0
	energy, err := evaluate(h, t, params)
	if err != nil {
		return nil, wrapEvaluationError("initial energy evaluation", err)
	}
	evals++
	if err := checkEnergy("initial energy", energy, names, params); err != nil {
		return nil, err
	}

	converged := false
	accepted := 0
	for iter := 0; iter < maxIter; iter++ {
		grad, gradEvals, err := parameterShiftGradient(h, t, params, names)
		if err != nil {
			return nil, wrapEvaluationError(fmt.Sprintf("gradient evaluation at iteration %d", iter), err)
		}
		evals += gradEvals
		// The shifted energies are evaluated inside parameterShiftGradient
		// and not checked there; any non-finite one makes the gradient
		// non-finite, and so does a difference of two finite energies that
		// overflows (a MaxFloat64 term at theta = pi/2 gives -M and +M).
		// Coefficients are finite (validateVQEStructure) and every Pauli
		// expectation is bounded by 1, so either way the cause is overflow.
		// Stop here: the step would carry a non-finite value into a
		// parameter, and Bind would then blame that parameter although the
		// caller supplied it finite.
		for _, name := range names {
			if !isFinite(grad[name]) {
				return nil, &InvalidVQEInputError{Reason: fmt.Sprintf("gradient of parameter %q is non-finite (%v) at iteration %d: the Hamiltonian's energy at a shifted point or the shift difference overflows float64", name, grad[name], iter)}
			}
		}

		// A finite gradient times an in-domain but huge StepSize (1e308 is
		// finite and positive) still overflows the update, and Bind would
		// again blame the parameter. The current step is at most the
		// StepSize option (halving only shrinks it; the 1e-6 floor cannot
		// overflow anything), so an overflowing step is StepSize's fault.
		steps := parameterized.Params{}
		for _, name := range names {
			steps[name] = params[name] - step*grad[name]
			if !isFinite(steps[name]) {
				return nil, &InvalidVQEInputError{Reason: fmt.Sprintf("StepSize is too large: the step of parameter %q overflows float64 at iteration %d (step size %v times gradient %v)", name, iter, step, grad[name])}
			}
			// The same bound the initial parameters met: past it the next
			// gradient would shift an angle float64 cannot resolve, and
			// parameterShiftGradient would reject it inside the wrap.
			// Stopping here keeps that check unreachable from VQE and
			// names the step that crossed, since the caller chose neither
			// the point nor the iteration.
			if math.Abs(steps[name]) > maxShiftMagnitude {
				return nil, &InvalidVQEInputError{Reason: fmt.Sprintf("the step of parameter %q reaches %v at iteration %d, beyond the 2^26 (%v) within which float64 resolves the +/- pi/2 parameter shift (step size %v times gradient %v)", name, steps[name], iter, float64(maxShiftMagnitude), step, grad[name])}
			}
		}
		newEnergy, err := evaluate(h, t, steps)
		if err != nil {
			return nil, wrapEvaluationError(fmt.Sprintf("step evaluation at iteration %d", iter), err)
		}
		evals++
		if err := checkEnergy(fmt.Sprintf("step energy at iteration %d", iter), newEnergy, names, steps); err != nil {
			return nil, err
		}

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
