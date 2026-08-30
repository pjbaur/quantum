package algorithm_test

import (
	"errors"
	"math"
	"testing"

	"github.com/pjbaur/quantum/algorithm"
	"github.com/pjbaur/quantum/parameterized"
)

func TestVQEConvergesToH2GroundState(t *testing.T) {
	h := algorithm.H2Hamiltonian()
	tmpl := algorithm.H2Ansatz()

	res, err := algorithm.VQE(h, tmpl, algorithm.VQEOptions{
		InitialParams: parameterized.Params{"theta": 0.1},
	})
	if err != nil {
		t.Fatalf("VQE: %v", err)
	}
	ground := h2GroundEnergy(t)
	if math.Abs(res.Energy-ground) > 1e-6 {
		t.Fatalf("VQE energy %v, want ground %v (Converged=%v, iters=%d)", res.Energy, ground, res.Converged, res.Iterations)
	}
	if !res.Converged {
		t.Fatal("VQE did not converge within default MaxIterations")
	}
	// Evaluation accounting: initial energy + per-iteration (1 accept eval +
	// 2*params gradient evals), plus one eval per reverted step.
	if res.Evaluations <= 0 || res.Evaluations > 3*res.Iterations+2*len(res.Params)+1 {
		t.Fatalf("Evaluations = %d implausible for %d iterations", res.Evaluations, res.Iterations)
	}
}

func TestVQEDefaultsAndExplicitOptions(t *testing.T) {
	h := algorithm.H2Hamiltonian()
	tmpl := algorithm.H2Ansatz()

	// Zero-value options must work: defaults for everything.
	res, err := algorithm.VQE(h, tmpl, algorithm.VQEOptions{})
	if err != nil {
		t.Fatalf("VQE zero opts: %v", err)
	}
	if !res.Converged {
		t.Fatal("zero-value options failed to converge")
	}
	if _, ok := res.Params["theta"]; !ok {
		t.Fatalf("res.Params = %v, want theta present", res.Params)
	}
}

func TestVQEInvalidInputs(t *testing.T) {
	tmpl := algorithm.H2Ansatz()
	if _, err := algorithm.VQE(nil, tmpl, algorithm.VQEOptions{}); err == nil {
		t.Fatal("VQE(nil Hamiltonian) succeeded, want error")
	}
	h := algorithm.H2Hamiltonian()
	if _, err := algorithm.VQE(h, nil, algorithm.VQEOptions{}); err == nil {
		t.Fatal("VQE(nil template) succeeded, want error")
	}
	var e *algorithm.InvalidVQEInputError
	_, err := algorithm.VQE(h, tmpl, algorithm.VQEOptions{InitialParams: parameterized.Params{"nope": 1.0}})
	if !errors.As(err, &e) {
		t.Fatalf("err = %v, want InvalidVQEInputError (unknown initial param)", err)
	}
}

func TestVQERespectsMaxIterations(t *testing.T) {
	h := algorithm.H2Hamiltonian()
	tmpl := algorithm.H2Ansatz()
	res, err := algorithm.VQE(h, tmpl, algorithm.VQEOptions{MaxIterations: 2, Tolerance: 1e-15})
	if err != nil {
		t.Fatalf("VQE: %v", err)
	}
	if res.Converged {
		t.Fatal("Converged=true with MaxIterations=2 and unreachable tolerance")
	}
	if res.Iterations > 2 {
		t.Fatalf("Iterations = %d, want <= 2", res.Iterations)
	}
}
