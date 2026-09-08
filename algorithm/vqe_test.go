package algorithm

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/parameterized"
	"github.com/pjbaur/quantum/quantum"
)

// gradientTargetTemplate: Ry(a) q0, CNOT 0->1, Rx(b) q1, evaluated against
// the H2 Hamiltonian.
func gradientTargetTemplate() *parameterized.Template {
	tmpl := parameterized.NewTemplate(2)
	if err := tmpl.AddParamGate("a", parameterized.Ry, 0); err != nil {
		panic(err)
	}
	if err := tmpl.AddGate(gates.NewCNOT(), 0, 1); err != nil {
		panic(err)
	}
	if err := tmpl.AddParamGate("b", parameterized.Rx, 1); err != nil {
		panic(err)
	}
	return tmpl
}

func TestParameterShiftMatchesFiniteDifference(t *testing.T) {
	h := H2Hamiltonian()
	tmpl := gradientTargetTemplate()

	for _, base := range []float64{-1.2, -0.3, 0.0, 0.45, 1.7} {
		params := parameterized.Params{"a": base, "b": base/2 + 0.1}
		grad, evals, err := parameterShiftGradient(h, tmpl, params, tmpl.ParamNames())
		if err != nil {
			t.Fatalf("parameterShiftGradient(base=%v): %v", base, err)
		}
		if evals != 4 { // two parameters, two evaluations each
			t.Fatalf("evals = %d, want 4", evals)
		}
		for _, name := range []string{"a", "b"} {
			const hstep = 1e-6
			plus := parameterized.Params{"a": params["a"], "b": params["b"]}
			minus := parameterized.Params{"a": params["a"], "b": params["b"]}
			plus[name] += hstep
			minus[name] -= hstep
			ePlus, err := evaluate(h, tmpl, plus)
			if err != nil {
				t.Fatalf("evaluate plus: %v", err)
			}
			eMinus, err := evaluate(h, tmpl, minus)
			if err != nil {
				t.Fatalf("evaluate minus: %v", err)
			}
			fd := (ePlus - eMinus) / (2 * hstep)
			if math.Abs(grad[name]-fd) > 1e-6 {
				t.Fatalf("base=%v param=%s: parameter-shift %v != finite-diff %v", base, name, grad[name], fd)
			}
		}
	}
}

func TestEvaluateIsDeterministicAndFinite(t *testing.T) {
	h := H2Hamiltonian()
	tmpl := H2Ansatz()
	e1, err := evaluate(h, tmpl, parameterized.Params{"theta": 0.2})
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	e2, err := evaluate(h, tmpl, parameterized.Params{"theta": 0.2})
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if e1 != e2 {
		t.Fatalf("evaluate not deterministic: %v vs %v", e1, e2)
	}
	if math.IsNaN(e1) || math.IsInf(e1, 0) {
		t.Fatalf("evaluate = %v, want finite", e1)
	}
}

// rescaledGradientTemplate is gradientTargetTemplate with every factory
// binding twice its parameter (Ry(2a), Rx(2b)), the rescaling that
// parameterShiftGradient's doc comment forbids. The stock parameterized
// factories bind unscaled by construction, so the rescale is written as a
// closure over the gates constructors, exactly how a caller would
// introduce the bug.
func rescaledGradientTemplate() *parameterized.Template {
	tmpl := parameterized.NewTemplate(2)
	if err := tmpl.AddParamGate("a", func(v float64) quantum.Gate { return gates.NewRy(2 * v) }, 0); err != nil {
		panic(err)
	}
	if err := tmpl.AddGate(gates.NewCNOT(), 0, 1); err != nil {
		panic(err)
	}
	if err := tmpl.AddParamGate("b", func(v float64) quantum.Gate { return gates.NewRx(2 * v) }, 1); err != nil {
		panic(err)
	}
	return tmpl
}

// TestParameterShiftIsBlindToRescaledFactory pins the failure mode behind
// parameterShiftGradient's exp(-i*theta*P/2) precondition. A factory that
// binds 2*theta makes E periodic in theta with period pi, so the +/- pi/2
// shift difference vanishes: the returned gradient is identically zero
// even where the finite-difference slope clearly is not. Nothing in the
// driver can tell that from a stationary point, so VQE accepts a zero
// step, sees |delta E| = 0 < Tolerance, and reports Converged at its
// starting parameters. If this test ever fails, the doc comment on
// parameterShiftGradient describes behavior that no longer holds.
func TestParameterShiftIsBlindToRescaledFactory(t *testing.T) {
	h := H2Hamiltonian()
	tmpl := rescaledGradientTemplate()

	for _, base := range []float64{-1.2, 0.0, 0.45, 1.7} {
		params := parameterized.Params{"a": base, "b": base/2 + 0.1}
		grad, _, err := parameterShiftGradient(h, tmpl, params, tmpl.ParamNames())
		if err != nil {
			t.Fatalf("parameterShiftGradient(base=%v): %v", base, err)
		}
		for _, name := range []string{"a", "b"} {
			const hstep = 1e-6
			plus := parameterized.Params{"a": params["a"], "b": params["b"]}
			minus := parameterized.Params{"a": params["a"], "b": params["b"]}
			plus[name] += hstep
			minus[name] -= hstep
			ePlus, err := evaluate(h, tmpl, plus)
			if err != nil {
				t.Fatalf("evaluate plus: %v", err)
			}
			eMinus, err := evaluate(h, tmpl, minus)
			if err != nil {
				t.Fatalf("evaluate minus: %v", err)
			}
			fd := (ePlus - eMinus) / (2 * hstep)
			// Every base point above sits on a real slope (|dE/dtheta| is
			// at least 0.15 for both parameters); the threshold keeps the
			// test from silently drifting onto a genuine stationary point,
			// where a zero gradient would prove nothing.
			if math.Abs(fd) < 0.1 {
				t.Fatalf("base=%v param=%s: finite difference %v too small to distinguish from a stationary point", base, name, fd)
			}
			if math.Abs(grad[name]) > 1e-9 {
				t.Errorf("base=%v param=%s: parameter-shift gradient %v, want 0 (a period-pi parameter is invisible to the +/- pi/2 shift)", base, name, grad[name])
			}
		}

		start, err := evaluate(h, tmpl, params)
		if err != nil {
			t.Fatalf("evaluate start: %v", err)
		}
		result, err := VQE(h, tmpl, VQEOptions{InitialParams: params, MaxIterations: 50})
		if err != nil {
			t.Fatalf("VQE(base=%v): %v", base, err)
		}
		if !result.Converged || result.Iterations != 1 {
			t.Errorf("base=%v: VQE Converged=%v Iterations=%d, want a one-iteration no-op reported as Converged", base, result.Converged, result.Iterations)
		}
		if math.Abs(result.Energy-start) > 1e-12 {
			t.Errorf("base=%v: VQE energy %v moved from start %v despite a zero gradient", base, result.Energy, start)
		}
		for name, v := range params {
			if math.Abs(result.Params[name]-v) > 1e-12 {
				t.Errorf("base=%v: VQE moved %s from %v to %v despite a zero gradient", base, name, v, result.Params[name])
			}
		}
	}
}

// TestVQEOptionValidation pins the option domains (backlog item 14).
// Before validation existed, a negative StepSize ascended once and was
// then clamped to +1e-6 by the floor, a negative MaxIterations returned
// unconverged after zero iterations, a negative or NaN Tolerance ran to
// MaxIterations, and a non-finite StepSize or InitialParams value
// surfaced as parameterized.InvalidParameterValueError from the first
// Bind. All are the caller's option literal, so all are
// InvalidVQEInputError.
func TestVQEOptionValidation(t *testing.T) {
	h := H2Hamiltonian()
	tmpl := H2Ansatz()

	cases := []struct {
		name string
		opts VQEOptions
	}{
		{"negative StepSize", VQEOptions{StepSize: -0.3, InitialParams: parameterized.Params{"theta": 0.1}}},
		{"NaN StepSize", VQEOptions{StepSize: math.NaN(), InitialParams: parameterized.Params{"theta": 0.1}}},
		{"Inf StepSize", VQEOptions{StepSize: math.Inf(1), InitialParams: parameterized.Params{"theta": 0.1}}},
		{"negative MaxIterations", VQEOptions{MaxIterations: -1}},
		{"negative Tolerance", VQEOptions{Tolerance: -1, MaxIterations: 5}},
		{"NaN Tolerance", VQEOptions{Tolerance: math.NaN(), MaxIterations: 5}},
		{"NaN InitialParams", VQEOptions{InitialParams: parameterized.Params{"theta": math.NaN()}}},
		{"Inf InitialParams", VQEOptions{InitialParams: parameterized.Params{"theta": math.Inf(-1)}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var e *InvalidVQEInputError
			res, err := VQE(h, tmpl, c.opts)
			if !errors.As(err, &e) {
				t.Errorf("VQE(%+v): err = %v, result = %+v; want InvalidVQEInputError", c.opts, err, res)
			}
		})
	}
}

// TestVQEOptionErrorNamesTheField pins the Reason wording: each message
// names the offending field and its value, so a caller reading the error
// can go straight to the option literal, and says that zero would have
// selected the default.
func TestVQEOptionErrorNamesTheField(t *testing.T) {
	cases := []struct {
		name string
		opts VQEOptions
		want string
	}{
		{"StepSize", VQEOptions{StepSize: -0.3}, "StepSize must be finite and positive (zero selects the default 0.3), got -0.3"},
		{"MaxIterations", VQEOptions{MaxIterations: -1}, "MaxIterations must not be negative (zero selects the default 200), got -1"},
		{"Tolerance", VQEOptions{Tolerance: math.NaN()}, "Tolerance must be finite and positive (zero selects the default 1e-10), got NaN"},
		{"InitialParams", VQEOptions{InitialParams: parameterized.Params{"theta": math.Inf(-1)}}, `initial parameter "theta" has non-finite value -Inf`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var e *InvalidVQEInputError
			_, err := VQE(H2Hamiltonian(), H2Ansatz(), c.opts)
			if !errors.As(err, &e) {
				t.Fatalf("VQE(%+v): err = %v (%T); want InvalidVQEInputError", c.opts, err, err)
			}
			if e.Reason != c.want {
				t.Fatalf("Reason = %q, want %q", e.Reason, c.want)
			}
		})
	}
}

// TestVQEStructuralMismatchIsInvalidInput pins the structural contract
// (backlog item 14): a Hamiltonian whose Pauli strings do not match the
// template's register, or a factory returning a gate wider than its
// target list, is InvalidVQEInputError. Before the check existed the
// first two surfaced as quantum.IncompatibleQubitCountError and the third
// as quantum.InvalidGateApplicationError from inside the first
// evaluation.
func TestVQEStructuralMismatchIsInvalidInput(t *testing.T) {
	twoQubitFactory := func(v float64) quantum.Gate { return gates.NewCNOT() }
	wideFactoryTemplate := parameterized.NewTemplate(2)
	if err := wideFactoryTemplate.AddParamGate("theta", twoQubitFactory, 0); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		h    *Hamiltonian
		tmpl *parameterized.Template
	}{
		{"three-axis term on two-qubit template", NewHamiltonian().AddTerm(1, quantum.PauliZ, quantum.PauliZ, quantum.PauliZ), H2Ansatz()},
		{"one-axis term on two-qubit template", NewHamiltonian().AddTerm(1, quantum.PauliZ), H2Ansatz()},
		{"factory gate wider than its target list", H2Hamiltonian(), wideFactoryTemplate},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var e *InvalidVQEInputError
			res, err := VQE(c.h, c.tmpl, VQEOptions{})
			if !errors.As(err, &e) {
				t.Errorf("VQE: err = %v (%T), result = %+v; want InvalidVQEInputError", err, err, res)
			}
		})
	}
}

// TestVQENonFiniteHamiltonianIsNotBlamedOnParams pins the attribution
// contract (backlog item 14): a NaN or Inf coefficient is the
// Hamiltonian's fault. Before the check existed the non-finite energy
// flowed into a NaN gradient and a NaN step, and parameterized.Bind
// rejected the stepped parameter by name although the caller supplied it
// finite.
func TestVQENonFiniteHamiltonianIsNotBlamedOnParams(t *testing.T) {
	cases := []struct {
		name string
		h    *Hamiltonian
	}{
		{"NaN Pauli term", NewHamiltonian().AddTerm(math.NaN(), quantum.PauliZ, quantum.PauliI)},
		{"Inf identity term", NewHamiltonian().AddTerm(math.Inf(1))},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res, err := VQE(c.h, H2Ansatz(), VQEOptions{InitialParams: parameterized.Params{"theta": 0.1}})
			if err == nil {
				t.Fatalf("VQE returned %+v with no error for a non-finite Hamiltonian", res)
			}
			var pe *parameterized.InvalidParameterValueError
			if errors.As(err, &pe) {
				t.Fatalf("VQE blamed parameter %q (value %v) for a non-finite Hamiltonian coefficient: %v", pe.Name, pe.Value, err)
			}
		})
	}
}

// TestVQEStructuralErrorReasons pins the Reason wording for problems
// found before any evaluation: the message names the term by index and
// states the mismatch or the value, and nothing is wrapped because VQE
// detected the problem itself.
func TestVQEStructuralErrorReasons(t *testing.T) {
	cases := []struct {
		name string
		h    *Hamiltonian
		want string
	}{
		{"axes length", NewHamiltonian().AddTerm(1, quantum.PauliZ), "Hamiltonian term 0 has 1 Pauli axes but the template has 2 qubits"},
		{"NaN coefficient", NewHamiltonian().AddTerm(1, quantum.PauliZ, quantum.PauliI).AddTerm(math.NaN()), "Hamiltonian term 1 has non-finite coefficient NaN"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var e *InvalidVQEInputError
			res, err := VQE(c.h, H2Ansatz(), VQEOptions{})
			if !errors.As(err, &e) {
				t.Fatalf("err = %v (%T), result = %+v; want InvalidVQEInputError", err, err, res)
			}
			if e.Reason != c.want {
				t.Fatalf("Reason = %q, want %q", e.Reason, c.want)
			}
			if cause := errors.Unwrap(err); cause != nil {
				t.Fatalf("errors.Unwrap(err) = %v, want nil for a problem detected before any evaluation", cause)
			}
		})
	}
}

// TestVQEEvaluationErrorKeepsItsCause pins the wrapping contract for
// problems only running the template can reveal: the caller sees
// InvalidVQEInputError, and errors.As still reaches the error type of the
// package that detected the problem. The wide-factory case fails inside
// parameterized.Bind (circuit.AddGate rejects a 2-qubit gate on 1 target);
// the bad-axis case fails inside quantum.Expectation, whose axis domain
// VQE deliberately does not duplicate.
func TestVQEEvaluationErrorKeepsItsCause(t *testing.T) {
	wide := parameterized.NewTemplate(2)
	if err := wide.AddParamGate("theta", func(v float64) quantum.Gate { return gates.NewCNOT() }, 0); err != nil {
		t.Fatal(err)
	}
	var ve *InvalidVQEInputError
	var ge *quantum.InvalidGateApplicationError
	_, err := VQE(H2Hamiltonian(), wide, VQEOptions{})
	if !errors.As(err, &ve) || !errors.As(err, &ge) {
		t.Fatalf("VQE(wide factory): err = %v (%T); want InvalidVQEInputError wrapping InvalidGateApplicationError", err, err)
	}
	if !strings.Contains(ve.Reason, "initial energy evaluation") || !strings.Contains(ve.Reason, ge.Error()) {
		t.Fatalf("Reason = %q, want the evaluation phase and the cause %q", ve.Reason, ge.Error())
	}

	badAxis := NewHamiltonian().AddTerm(1, quantum.PauliAxis(7), quantum.PauliI)
	var ae *quantum.InvalidPauliAxisError
	_, err = VQE(badAxis, H2Ansatz(), VQEOptions{})
	if !errors.As(err, &ve) || !errors.As(err, &ae) {
		t.Fatalf("VQE(bad axis): err = %v (%T); want InvalidVQEInputError wrapping InvalidPauliAxisError", err, err)
	}
}

// TestVQEOverflowingHamiltonianIsNotBlamedOnParams covers the gap the
// up-front coefficient check leaves: coefficients that are each finite
// but whose sum passes float64. The energy is +Inf at the start and at
// one of the two shifted points, so without a guard the parameter-shift
// difference is -Inf and the descent step would carry theta to +Inf,
// where parameterized.Bind would reject it by name. The contract is that
// no error blames a parameter that was finite on entry, so VQE must stop
// at the first non-finite value it sees, here the initial energy, whose
// Reason names the point ("theta") it was evaluated at.
func TestVQEOverflowingHamiltonianIsNotBlamedOnParams(t *testing.T) {
	h := NewHamiltonian().AddTerm(math.MaxFloat64).AddTerm(math.MaxFloat64, quantum.PauliZ, quantum.PauliI)
	var ve *InvalidVQEInputError
	var pe *parameterized.InvalidParameterValueError
	res, err := VQE(h, H2Ansatz(), VQEOptions{InitialParams: parameterized.Params{"theta": 0.1}})
	if errors.As(err, &pe) {
		t.Fatalf("VQE blamed parameter %q (value %v) for an overflowing Hamiltonian: %v", pe.Name, pe.Value, err)
	}
	if !errors.As(err, &ve) {
		t.Fatalf("VQE: err = %v (%T), result = %+v; want InvalidVQEInputError", err, err, res)
	}
	if !strings.Contains(ve.Reason, `"theta"`) || !strings.Contains(ve.Reason, "non-finite") {
		t.Fatalf("Reason = %q, want it to name the non-finite gradient of %q", ve.Reason, "theta")
	}
}

// zOnQubit0 returns coeff * Z0 (identity on qubit 1). On H2Ansatz qubit 0
// is Ry(theta)|0>, untouched by the CNOT's control role, so
// E(theta) = coeff*cos(theta) and dE/dtheta = -coeff*sin(theta), which the
// parameter shift reproduces exactly.
func zOnQubit0(coeff float64) *Hamiltonian {
	return NewHamiltonian().AddTerm(coeff, quantum.PauliZ, quantum.PauliI)
}

// TestVQEHugeStepSizeDoesNotBlameFiniteParameter pins the step guard
// (backlog item 14, round 1). StepSize 1e308 is finite and positive, so
// validateVQEOptions accepts it, but the update params - step*grad
// overflows to +Inf and parameterized.Bind would reject the stepped value
// by name. The contract is that no error blames a parameter the caller
// supplied finite, so VQE must stop at the overflowing step and blame
// StepSize.
func TestVQEHugeStepSizeDoesNotBlameFiniteParameter(t *testing.T) {
	h := zOnQubit0(4) // E = 4 cos(theta); gradient at theta=1 is -4 sin(1)
	tmpl := H2Ansatz()
	opts := VQEOptions{
		InitialParams: parameterized.Params{"theta": 1},
		StepSize:      1e308, // finite and positive: inside the documented domain
	}
	if err := validateVQEOptions(opts); err != nil {
		t.Fatalf("setup: StepSize 1e308 is finite and positive but validateVQEOptions rejected it: %v", err)
	}

	res, err := VQE(h, tmpl, opts)
	var pe *parameterized.InvalidParameterValueError
	if errors.As(err, &pe) {
		t.Fatalf("VQE blamed parameter %q (value %v), which the caller supplied finite: %v", pe.Name, pe.Value, err)
	}
	var ve *InvalidVQEInputError
	if err != nil && !errors.As(err, &ve) {
		t.Fatalf("err = %v (%T), want nil or InvalidVQEInputError", err, err)
	}
	if err == nil {
		for name, v := range res.Params {
			if !isFinite(v) {
				t.Fatalf("result parameter %q = %v, want finite", name, v)
			}
		}
	}
}

// TestVQEStepOverflowReasonBlamesStepSize pins the step guard's Reason:
// it leads with the field at fault and carries the step size and gradient
// whose product overflowed. The gradient is read back from
// parameterShiftGradient so the expected text is exact without pinning a
// float literal.
func TestVQEStepOverflowReasonBlamesStepSize(t *testing.T) {
	h := zOnQubit0(2)
	tmpl := H2Ansatz()
	params := parameterized.Params{"theta": math.Pi / 2}
	grad, _, err := parameterShiftGradient(h, tmpl, params, tmpl.ParamNames())
	if err != nil {
		t.Fatalf("setup gradient: %v", err)
	}
	if !isFinite(grad["theta"]) || grad["theta"] == 0 {
		t.Fatalf("setup: gradient = %v, want finite and non-zero so the step overflows", grad["theta"])
	}

	var ve *InvalidVQEInputError
	_, err = VQE(h, tmpl, VQEOptions{StepSize: 1e308, MaxIterations: 3, InitialParams: params})
	if !errors.As(err, &ve) {
		t.Fatalf("err = %v (%T), want InvalidVQEInputError", err, err)
	}
	want := fmt.Sprintf("StepSize is too large: the step of parameter %q overflows float64 at iteration 0 (step size %v times gradient %v)", "theta", 1e308, grad["theta"])
	if ve.Reason != want {
		t.Fatalf("Reason = %q, want %q", ve.Reason, want)
	}
	if cause := errors.Unwrap(err); cause != nil {
		t.Fatalf("errors.Unwrap(err) = %v, want nil for a problem VQE detected itself", cause)
	}
}

// TestVQENoParameterTemplateReportsOverflowingHamiltonian pins the
// initial-energy guard on the path the gradient guard cannot see (backlog
// item 14, round 1): a template with no parameters has no gradient, so an
// overflowing Hamiltonian used to return Energy +Inf with a nil error.
func TestVQENoParameterTemplateReportsOverflowingHamiltonian(t *testing.T) {
	tmpl := parameterized.NewTemplate(1)
	if err := tmpl.AddGate(gates.NewPauliX(), 0); err != nil {
		t.Fatal(err)
	}
	h := NewHamiltonian().AddTerm(math.MaxFloat64).AddTerm(math.MaxFloat64)
	if err := validateVQEStructure(h, tmpl); err != nil {
		t.Fatalf("setup: both coefficients are finite but validateVQEStructure rejected them: %v", err)
	}

	res, err := VQE(h, tmpl, VQEOptions{})
	var ve *InvalidVQEInputError
	if !errors.As(err, &ve) {
		t.Fatalf("VQE: err = %v, result = %+v; want InvalidVQEInputError for a Hamiltonian whose energy overflows float64", err, res)
	}
}

// TestVQEOverflowingStartEnergyWithFiniteGradientIsReported pins the
// initial-energy guard where the gradient guard passes (backlog item 14,
// round 1): the start energy overflows while both shifted energies stay
// finite, so the gradient is finite and, without the guard, VQE
// optimized from +Inf and returned a nil error.
func TestVQEOverflowingStartEnergyWithFiniteGradientIsReported(t *testing.T) {
	// E(theta) = 0.95M + 0.1M cos(theta). At theta = 0.1 the sum is 1.0495M
	// (+Inf). At theta = 0.1 +/- pi/2 it is 0.95M -/+ 0.00998M, both finite.
	m := math.MaxFloat64
	h := NewHamiltonian().AddTerm(0.95*m).AddTerm(0.1*m, quantum.PauliZ, quantum.PauliI)
	tmpl := H2Ansatz()
	params := parameterized.Params{"theta": 0.1}

	start, err := evaluate(h, tmpl, params)
	if err != nil {
		t.Fatalf("setup evaluate: %v", err)
	}
	if !math.IsInf(start, 1) {
		t.Fatalf("setup: start energy = %v, want +Inf", start)
	}
	grad, _, err := parameterShiftGradient(h, tmpl, params, tmpl.ParamNames())
	if err != nil {
		t.Fatalf("setup gradient: %v", err)
	}
	if !isFinite(grad["theta"]) {
		t.Fatalf("setup: gradient = %v, want finite so the gradient guard does not fire", grad["theta"])
	}

	res, err := VQE(h, tmpl, VQEOptions{InitialParams: params, MaxIterations: 1})
	var pe *parameterized.InvalidParameterValueError
	if errors.As(err, &pe) {
		t.Fatalf("VQE blamed parameter %q (value %v), which the caller supplied finite: %v", pe.Name, pe.Value, err)
	}
	var ve *InvalidVQEInputError
	if !errors.As(err, &ve) {
		t.Fatalf("VQE: err = %v, result = %+v; want InvalidVQEInputError: the starting energy overflowed float64", err, res)
	}
}

// TestVQEEnergyGuardReasons pins the initial-energy Reason: it blames the
// Hamiltonian, names the point when the template has parameters (a
// stepped point is not one the caller chose, and the initial one reads
// the same way), omits it otherwise, and wraps nothing because VQE
// detected the overflow itself.
func TestVQEEnergyGuardReasons(t *testing.T) {
	fixed := parameterized.NewTemplate(1)
	if err := fixed.AddGate(gates.NewPauliX(), 0); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		h    *Hamiltonian
		tmpl *parameterized.Template
		opts VQEOptions
		want string
	}{
		{
			"with parameters",
			NewHamiltonian().AddTerm(math.MaxFloat64).AddTerm(math.MaxFloat64, quantum.PauliZ, quantum.PauliI),
			H2Ansatz(),
			VQEOptions{InitialParams: parameterized.Params{"theta": 0.1}},
			`initial energy is non-finite (+Inf) with "theta"=0.1: the Hamiltonian's energy overflows float64`,
		},
		{
			"no parameters",
			NewHamiltonian().AddTerm(math.MaxFloat64).AddTerm(math.MaxFloat64),
			fixed,
			VQEOptions{},
			"initial energy is non-finite (+Inf): the Hamiltonian's energy overflows float64",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var ve *InvalidVQEInputError
			res, err := VQE(c.h, c.tmpl, c.opts)
			if !errors.As(err, &ve) {
				t.Fatalf("err = %v (%T), result = %+v; want InvalidVQEInputError", err, err, res)
			}
			if ve.Reason != c.want {
				t.Fatalf("Reason = %q, want %q", ve.Reason, c.want)
			}
			if cause := errors.Unwrap(err); cause != nil {
				t.Fatalf("errors.Unwrap(err) = %v, want nil for a problem VQE detected itself", cause)
			}
		})
	}
}

// TestVQENonFiniteStepEnergyIsReported pins the step-energy guard: every
// energy up to the step is finite, the step lands where the Hamiltonian
// overflows, and VQE reports it rather than comparing an infinity. With
// E(theta) = 0.95M + 0.1M cos(theta) and theta = pi - 0.1 the start
// (0.85M) and both shifted energies (0.95M -/+ 0.00998M) are finite and the
// gradient is -0.1M sin(0.1); a StepSize of (pi + 0.1)/|gradient| steps
// to 2 pi, where the energy is 1.05M.
func TestVQENonFiniteStepEnergyIsReported(t *testing.T) {
	m := math.MaxFloat64
	h := NewHamiltonian().AddTerm(0.95*m).AddTerm(0.1*m, quantum.PauliZ, quantum.PauliI)
	tmpl := H2Ansatz()
	theta := math.Pi - 0.1
	params := parameterized.Params{"theta": theta}

	start, err := evaluate(h, tmpl, params)
	if err != nil {
		t.Fatalf("setup evaluate: %v", err)
	}
	if !isFinite(start) {
		t.Fatalf("setup: start energy = %v, want finite", start)
	}
	grad, _, err := parameterShiftGradient(h, tmpl, params, tmpl.ParamNames())
	if err != nil {
		t.Fatalf("setup gradient: %v", err)
	}
	if !isFinite(grad["theta"]) || grad["theta"] >= 0 {
		t.Fatalf("setup: gradient = %v, want finite and negative", grad["theta"])
	}
	step := (math.Pi + 0.1) / -grad["theta"]
	stepped := theta - step*grad["theta"]
	if e, err := evaluate(h, tmpl, parameterized.Params{"theta": stepped}); err != nil || !math.IsInf(e, 1) {
		t.Fatalf("setup: energy at the stepped point %v = %v, %v; want +Inf", stepped, e, err)
	}

	var ve *InvalidVQEInputError
	var pe *parameterized.InvalidParameterValueError
	res, err := VQE(h, tmpl, VQEOptions{InitialParams: params, StepSize: step, MaxIterations: 1})
	if errors.As(err, &pe) {
		t.Fatalf("VQE blamed parameter %q (value %v), which the caller supplied finite: %v", pe.Name, pe.Value, err)
	}
	if !errors.As(err, &ve) {
		t.Fatalf("VQE: err = %v (%T), result = %+v; want InvalidVQEInputError from the step-energy guard", err, err, res)
	}
	want := fmt.Sprintf("step energy at iteration 0 is non-finite (+Inf) with %q=%v: the Hamiltonian's energy overflows float64", "theta", stepped)
	if ve.Reason != want {
		t.Fatalf("Reason = %q, want %q", ve.Reason, want)
	}
}

// TestVQEGradientOverflowMessageMatchesFiniteEnergies pins the gradient
// guard's wording on the case it must not misdescribe (backlog item 14,
// round 1): a single MaxFloat64 term keeps every evaluated energy finite,
// and only the difference of the two shifted energies (-M and +M)
// overflows, so the Reason must not claim the energy overflows.
func TestVQEGradientOverflowMessageMatchesFiniteEnergies(t *testing.T) {
	h := zOnQubit0(math.MaxFloat64) // E = M cos(theta): finite for every theta
	tmpl := H2Ansatz()
	const theta = math.Pi / 2
	for _, shift := range []float64{0, math.Pi / 2, -math.Pi / 2} {
		e, err := evaluate(h, tmpl, parameterized.Params{"theta": theta + shift})
		if err != nil {
			t.Fatalf("setup evaluate(shift %v): %v", shift, err)
		}
		if !isFinite(e) {
			t.Fatalf("setup: energy at shift %v = %v, want finite", shift, e)
		}
	}

	res, err := VQE(h, tmpl, VQEOptions{InitialParams: parameterized.Params{"theta": theta}})
	var ve *InvalidVQEInputError
	if !errors.As(err, &ve) {
		t.Fatalf("VQE: err = %v, result = %+v; want InvalidVQEInputError from the gradient guard", err, res)
	}
	if strings.Contains(ve.Reason, "energy overflows") {
		t.Fatalf("Reason = %q says the energy overflows, but the energy at the start and at both shifted points is finite; only the shift difference overflowed", ve.Reason)
	}
}

// TestVQEGradientGuardReason pins the gradient guard's exact Reason on the
// shift-difference case, so the spec's Decision 3 literal stays tied to
// the code.
func TestVQEGradientGuardReason(t *testing.T) {
	var ve *InvalidVQEInputError
	_, err := VQE(zOnQubit0(math.MaxFloat64), H2Ansatz(), VQEOptions{InitialParams: parameterized.Params{"theta": math.Pi / 2}})
	if !errors.As(err, &ve) {
		t.Fatalf("err = %v (%T), want InvalidVQEInputError", err, err)
	}
	const want = `gradient of parameter "theta" is non-finite (-Inf) at iteration 0: the Hamiltonian's energy at a shifted point or the shift difference overflows float64`
	if ve.Reason != want {
		t.Fatalf("Reason = %q, want %q", ve.Reason, want)
	}
}

// TestVQEEvaluationErrorNamesThePhase pins the two later wrap phrases,
// reachable only through a factory that fails for some values: one that
// returns a two-qubit gate (which parameterized.Bind rejects on a single
// target) once its value passes 10. Starting at 9 the +pi/2 shift crosses
// it during the gradient; starting at 0.1 with a StepSize of 100 the
// descent step does.
func TestVQEEvaluationErrorNamesThePhase(t *testing.T) {
	failsPastTen := func(v float64) quantum.Gate {
		if v > 10 {
			return gates.NewCNOT()
		}
		return gates.NewRy(v)
	}
	newTemplate := func() *parameterized.Template {
		tmpl := parameterized.NewTemplate(2)
		if err := tmpl.AddParamGate("theta", failsPastTen, 0); err != nil {
			t.Fatal(err)
		}
		return tmpl
	}
	cases := []struct {
		name  string
		opts  VQEOptions
		phase string
	}{
		{"gradient", VQEOptions{InitialParams: parameterized.Params{"theta": 9}}, "gradient evaluation at iteration 0"},
		{"step", VQEOptions{InitialParams: parameterized.Params{"theta": 0.1}, StepSize: 100}, "step evaluation at iteration 0"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var ve *InvalidVQEInputError
			var ge *quantum.InvalidGateApplicationError
			_, err := VQE(zOnQubit0(4), newTemplate(), c.opts)
			if !errors.As(err, &ve) || !errors.As(err, &ge) {
				t.Fatalf("err = %v (%T); want InvalidVQEInputError wrapping InvalidGateApplicationError", err, err)
			}
			if want := c.phase + " failed: " + ge.Error(); ve.Reason != want {
				t.Fatalf("Reason = %q, want %q", ve.Reason, want)
			}
		})
	}
}

// TestVQEEmptyNameParameterIsCheckedLikeAnyOther pins the
// one-gate-per-parameter rule for a parameter named "" (backlog item 15).
// The empty string is a declared name like any other: AddParamGate
// accepts it and ParamNames lists it. Before the fix ParamStepCounts
// mistook it for its fixed-step marker and never counted it, so a ""
// parameter driving two gates passed validateVQEStructure and VQE
// optimized it against the higher-harmonic gradient the check exists to
// prevent, reporting Converged.
func TestVQEEmptyNameParameterIsCheckedLikeAnyOther(t *testing.T) {
	h := H2Hamiltonian()
	tmpl := parameterized.NewTemplate(2)
	if err := tmpl.AddParamGate("", parameterized.Ry, 0); err != nil {
		t.Fatal(err)
	}
	if err := tmpl.AddParamGate("", parameterized.Ry, 0); err != nil {
		t.Fatal(err)
	}
	if got := tmpl.ParamNames(); len(got) != 1 || got[0] != "" {
		t.Fatalf("ParamNames = %q, want the single declared name %q", got, "")
	}

	var e *InvalidVQEInputError
	res, err := VQE(h, tmpl, VQEOptions{})
	if !errors.As(err, &e) {
		t.Fatalf("VQE with parameter %q driving 2 template steps: err = %v, result = %+v; want InvalidVQEInputError (the doc says VQE enforces one gate per parameter through ParamStepCounts, whose counts are %v)", "", err, res, tmpl.ParamStepCounts())
	}
}

// h2AnsatzNamed builds H2Ansatz's three-gate structure (X on qubit 1,
// Ry(name) on qubit 0, CNOT 0->1) with a caller-chosen parameter name, so
// a test can compare two names on the identical shape without hand-
// copying the ansatz and risking drift from a future H2Ansatz change.
func h2AnsatzNamed(name string) *parameterized.Template {
	t := parameterized.NewTemplate(2)
	if err := t.AddGate(gates.NewPauliX(), 1); err != nil {
		panic(err)
	}
	if err := t.AddParamGate(name, parameterized.Ry, 0); err != nil {
		panic(err)
	}
	if err := t.AddGate(gates.NewCNOT(), 0, 1); err != nil {
		panic(err)
	}
	return t
}

// TestVQEAcceptsEmptyNameDrivingOneGate pins the other half of "like any
// other": a parameter named "" that drives exactly one gate satisfies
// the rule, so VQE optimizes it and reaches the same energy as the same
// ansatz with the parameter called "theta". The name is an opaque map key
// to every part of the driver; this test passes before and after the
// ParamStepCounts fix and guards against rejecting "" outright.
func TestVQEAcceptsEmptyNameDrivingOneGate(t *testing.T) {
	h := H2Hamiltonian()
	tmpl := h2AnsatzNamed("")

	ref, err := VQE(h, h2AnsatzNamed("theta"), VQEOptions{InitialParams: parameterized.Params{"theta": 0.1}})
	if err != nil {
		t.Fatalf("reference VQE: %v", err)
	}
	res, err := VQE(h, tmpl, VQEOptions{InitialParams: parameterized.Params{"": 0.1}})
	if err != nil {
		t.Fatalf("VQE with parameter %q driving one gate: %v; want it accepted like any other name", "", err)
	}
	if res.Energy != ref.Energy || res.Iterations != ref.Iterations || res.Params[""] != ref.Params["theta"] {
		t.Fatalf("VQE with parameter %q: energy %v after %d iterations at %v, want the same run as with %q: energy %v after %d iterations at %v", "", res.Energy, res.Iterations, res.Params[""], "theta", ref.Energy, ref.Iterations, ref.Params["theta"])
	}
}

// TestParameterShiftRejectsMissingParam pins that a declared parameter
// absent from params is an error whatever names asks to shift (backlog
// item 16). Before the check, the shift wrote a missing name into the
// plus and minus copies at +/- pi/2, so Bind saw a complete binding and
// the helper returned the slope at an implicit 0 with a nil error when
// that name was the only one shifted, while any names that left the gap
// unshifted failed in Bind.
func TestParameterShiftRejectsMissingParam(t *testing.T) {
	h := H2Hamiltonian()
	tmpl := gradientTargetTemplate() // declares a and b
	incomplete := parameterized.Params{"a": 0.3}

	cases := []struct {
		name  string
		names []string
	}{
		{"a only", []string{"a"}},
		{"b only", []string{"b"}},
		{"a then b", []string{"a", "b"}},
		{"b then a", []string{"b", "a"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			grad, evals, err := parameterShiftGradient(h, tmpl, incomplete, c.names)
			if err == nil {
				t.Fatalf("names=%v with params=%v (declared %v): grad = %v, evals = %d, err = nil; want an error for a missing declared parameter", c.names, incomplete, tmpl.ParamNames(), grad, evals)
			}
		})
	}
}

// TestParameterShiftMissingParamErrorMatchesBind pins what the rejection
// looks like: the error is parameterized.MissingParameterError naming the
// first missing parameter in declaration order, the same error with the
// same text that Bind returns for the unshifted params, whatever the order
// of names and even when names is empty; and nothing is evaluated first,
// so the returned gradient is nil and the count is zero.
func TestParameterShiftMissingParamErrorMatchesBind(t *testing.T) {
	h := H2Hamiltonian()
	tmpl := gradientTargetTemplate() // declares a and b

	cases := []struct {
		name   string
		params parameterized.Params
		names  []string
		want   string
	}{
		{"b missing, b shifted", parameterized.Params{"a": 0.3}, []string{"b"}, "b"},
		{"a missing, a shifted", parameterized.Params{"b": 0.1}, []string{"a"}, "a"},
		{"b missing, a shifted", parameterized.Params{"a": 0.3}, []string{"a"}, "b"},
		{"b missing, b then a", parameterized.Params{"a": 0.3}, []string{"b", "a"}, "b"},
		{"both missing, b then a", parameterized.Params{}, []string{"b", "a"}, "a"},
		{"both missing, nothing shifted", parameterized.Params{}, nil, "a"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			grad, evals, err := parameterShiftGradient(h, tmpl, c.params, c.names)
			var me *parameterized.MissingParameterError
			if !errors.As(err, &me) {
				t.Fatalf("names=%v with params=%v: grad = %v, evals = %d, err = %v; want parameterized.MissingParameterError", c.names, c.params, grad, evals, err)
			}
			if me.Name != c.want {
				t.Errorf("missing parameter named %q, want %q (first missing in declaration order, not in names order)", me.Name, c.want)
			}
			if grad != nil || evals != 0 {
				t.Errorf("grad = %v, evals = %d, want nil and 0: an incomplete binding is rejected before any evaluation", grad, evals)
			}
			_, bindErr := tmpl.Bind(c.params)
			if bindErr == nil || err.Error() != bindErr.Error() {
				t.Errorf("err = %q, want Bind's own error for the same params, %q", err, bindErr)
			}
		})
	}
}

// TestParameterShiftUndeclaredNameIsRejectedByBind pins the case the
// completeness check leaves to Bind on purpose: a name in names that the
// template never declared is written into the shifted copies, but no
// shift can hide it, so Bind rejects it as UnknownParameterError at the
// first evaluation with nothing consumed.
func TestParameterShiftUndeclaredNameIsRejectedByBind(t *testing.T) {
	h := H2Hamiltonian()
	tmpl := gradientTargetTemplate() // declares a and b
	complete := parameterized.Params{"a": 0.3, "b": 0.1}

	grad, evals, err := parameterShiftGradient(h, tmpl, complete, []string{"c"})
	var ue *parameterized.UnknownParameterError
	if !errors.As(err, &ue) {
		t.Fatalf("names=[c] on a template declaring %v: grad = %v, evals = %d, err = %v; want parameterized.UnknownParameterError", tmpl.ParamNames(), grad, evals, err)
	}
	if ue.Name != "c" {
		t.Errorf("unknown parameter named %q, want %q", ue.Name, "c")
	}
	if grad != nil || evals != 0 {
		t.Errorf("grad = %v, evals = %d, want nil and 0", grad, evals)
	}
}

// TestParameterShiftDifferentiatesOnlyNamedParams pins the other half of
// the names contract: with a complete binding, names may be any subset in
// any order, the returned map holds exactly those names, each component
// equals the same component of the full gradient, and only the named
// parameters cost evaluations.
func TestParameterShiftDifferentiatesOnlyNamedParams(t *testing.T) {
	h := H2Hamiltonian()
	tmpl := gradientTargetTemplate() // declares a and b
	params := parameterized.Params{"a": 0.45, "b": 0.325}

	full, fullEvals, err := parameterShiftGradient(h, tmpl, params, []string{"a", "b"})
	if err != nil {
		t.Fatalf("full gradient: %v", err)
	}
	if fullEvals != 4 || len(full) != 2 {
		t.Fatalf("full gradient: %d evaluations over %d components, want 4 over 2", fullEvals, len(full))
	}

	reversed, evals, err := parameterShiftGradient(h, tmpl, params, []string{"b", "a"})
	if err != nil {
		t.Fatalf("reversed names: %v", err)
	}
	if evals != 4 || reversed["a"] != full["a"] || reversed["b"] != full["b"] {
		t.Errorf("names=[b a]: grad = %v after %d evaluations, want %v after 4: order must not change the result", reversed, evals, full)
	}

	only, evals, err := parameterShiftGradient(h, tmpl, params, []string{"b"})
	if err != nil {
		t.Fatalf("names=[b]: %v", err)
	}
	if evals != 2 || len(only) != 1 || only["b"] != full["b"] {
		t.Errorf("names=[b]: grad = %v after %d evaluations, want map[b:%v] after 2: only the named parameter is shifted", only, evals, full["b"])
	}
}
