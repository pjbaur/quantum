package algorithm

import (
	"errors"
	"math"
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
