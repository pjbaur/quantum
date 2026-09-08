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
	"testing"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/parameterized"
	"github.com/pjbaur/quantum/quantum"
)

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
