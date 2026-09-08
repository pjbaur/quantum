//go:build redtests

package algorithm

// Red tests for known open defects. Each test below documents a defect
// tracked as a numbered item in docs/enhancement-backlog-2026-08-27.md and
// is expected to fail until that item is closed. The redtests build
// constraint keeps them out of default builds so the regular suite stays
// green; run them with
//
//	go test -tags redtests ./algorithm -run '^TestRed'
//
// When an item is fixed, its test turns green: move it into the regular
// suite and drop it from here.

import (
	"errors"
	"math"
	"testing"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/parameterized"
	"github.com/pjbaur/quantum/quantum"
)

// Backlog item 15: empty parameter name defeats the one-gate-per-parameter
// check.
//
// VQE(template whose parameter is named "") × sentinel collision with the
// fixed-step marker in Template.ParamStepCounts → a "" parameter driving
// several gates passes the up-front check and reaches parameterShiftGradient.
func TestRedVQEEmptyNameParameterEscapesStepCountCheck(t *testing.T) {
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

// Backlog item 16: parameterShiftGradient shifts missing parameters from an
// implicit zero.
//
// parameterShiftGradient(params lacking a declared name that appears in
// names) × implicit zero default in the shifted copies → a gradient is
// returned with nil error, evaluated at value 0, and whether the call errors
// depends on the order of names.
func TestRedParameterShiftMissingParamIsRejected(t *testing.T) {
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

// Backlog item 14: VQE input validation and error taxonomy.
//
// VQE(Hamiltonian or factory structurally incompatible with the
// template register) × late detection → the error surfaces from
// quantum/circuit inside the loop after evaluations were spent, not as
// InvalidVQEInputError up front.
func TestRedVQEStructuralMismatchIsInvalidInput(t *testing.T) {
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

// Backlog item 14: VQE input validation and error taxonomy.
//
// VQE(Hamiltonian with a non-finite coefficient) × non-finite energy
// propagation → the failure is attributed to a template parameter
// (parameterized.InvalidParameterValueError) although every parameter is
// finite; the Hamiltonian doc delegates non-finite detection to the caller.
func TestRedVQENonFiniteHamiltonianIsNotBlamedOnParams(t *testing.T) {
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

// Backlog item 17: parameterShiftGradient undercounts evaluations on
// failure.
//
// parameterShiftGradient(second shifted evaluation fails) × evaluation
// accounting → the returned count omits the first, successful evaluation.
func TestRedParameterShiftCountsEvaluationsBeforeFailure(t *testing.T) {
	h := H2Hamiltonian()
	// Valid single-qubit gate for non-negative values, a two-qubit gate (a
	// dimension mismatch Bind rejects) for negative ones: the +pi/2 shift
	// evaluates, the -pi/2 shift fails.
	factory := func(v float64) quantum.Gate {
		if v < 0 {
			return gates.NewCNOT()
		}
		return gates.NewRy(v)
	}
	tmpl := parameterized.NewTemplate(2)
	if err := tmpl.AddParamGate("a", factory, 0); err != nil {
		t.Fatal(err)
	}

	grad, evals, err := parameterShiftGradient(h, tmpl, parameterized.Params{"a": 0.3}, []string{"a"})
	if err == nil {
		t.Fatalf("grad = %v, evals = %d, err = nil; want the -pi/2 evaluation to fail", grad, evals)
	}
	if evals != 1 {
		t.Fatalf("evals = %d after one successful and one failed evaluation, want 1 (evaluations consumed); err = %v", evals, err)
	}
}
