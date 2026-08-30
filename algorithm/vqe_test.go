package algorithm

import (
	"math"
	"testing"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/parameterized"
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
