package algorithm

import (
	"math"
	"testing"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/parameterized"
	"github.com/pjbaur/quantum/quantum"
	"github.com/pjbaur/quantum/state"
)

var triangleEdges = [][2]int{{0, 1}, {0, 2}, {1, 2}}

// triangleBasisState prepares |bits> with bits[k] on qubit k (qubit 0 = LSB).
func triangleBasisState(t *testing.T, bits ...int) quantum.QuantumState {
	t.Helper()
	s, err := state.New(3)
	if err != nil {
		t.Fatalf("state.New(3): %v", err)
	}
	for q, b := range bits {
		if b == 1 {
			if err := s.ApplyGate(gates.NewPauliX(), q); err != nil {
				t.Fatalf("PauliX(%d): %v", q, err)
			}
		}
	}
	return s
}

func TestMaxCutHamiltonianBasisEnergies(t *testing.T) {
	h, err := MaxCutHamiltonian(3, triangleEdges, nil)
	if err != nil {
		t.Fatalf("MaxCutHamiltonian: %v", err)
	}
	cases := []struct {
		bits []int
		want float64
	}{
		{[]int{0, 0, 0}, 3},  // no edges cut: sum of +1 z_i z_j
		{[]int{1, 0, 0}, -1}, // two edges cut
		{[]int{0, 1, 0}, -1},
		{[]int{1, 1, 1}, 3}, // same partition as 000
	}
	for _, c := range cases {
		got, err := h.Energy(triangleBasisState(t, c.bits...))
		if err != nil {
			t.Fatalf("Energy(%v): %v", c.bits, err)
		}
		if math.Abs(got-c.want) > 1e-9 {
			t.Errorf("Energy(%v) = %g, want %g", c.bits, got, c.want)
		}
	}
}

func TestMaxCutHamiltonianWeights(t *testing.T) {
	h, err := MaxCutHamiltonian(3, triangleEdges, []float64{2, 1, 1})
	if err != nil {
		t.Fatalf("MaxCutHamiltonian: %v", err)
	}
	// |100>: edge (0,1) cut with weight 2, edge (0,2) cut with weight 1,
	// edge (1,2) uncut: E = -2 - 1 + 1 = -2.
	got, err := h.Energy(triangleBasisState(t, 1, 0, 0))
	if err != nil {
		t.Fatalf("Energy: %v", err)
	}
	if math.Abs(got-(-2)) > 1e-9 {
		t.Errorf("weighted Energy = %g, want -2", got)
	}
}

func TestQAOATemplateZeroAnglesStartsAtPlusState(t *testing.T) {
	tmpl, err := QAOATemplate(3, triangleEdges, 1)
	if err != nil {
		t.Fatalf("QAOATemplate: %v", err)
	}
	h, err := MaxCutHamiltonian(3, triangleEdges, nil)
	if err != nil {
		t.Fatalf("MaxCutHamiltonian: %v", err)
	}
	params := parameterized.Params{}
	for _, name := range tmpl.ParamNames() {
		params[name] = 0
	}
	c, err := tmpl.Bind(params)
	if err != nil {
		t.Fatalf("Bind: %v", err)
	}
	s, err := state.New(3)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	if err := c.Execute(s); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	// Rz(0) = Rx(0) = I and each edge's CNOT pair cancels, leaving |+++>:
	// <+|Z_i Z_j|+> = 0 per term.
	got, err := h.Energy(s)
	if err != nil {
		t.Fatalf("Energy: %v", err)
	}
	if math.Abs(got) > 1e-9 {
		t.Errorf("zero-angle energy = %g, want 0", got)
	}
}

func TestQAOATemplateParamsDriveSingleGates(t *testing.T) {
	tmpl, err := QAOATemplate(3, triangleEdges, 1)
	if err != nil {
		t.Fatalf("QAOATemplate: %v", err)
	}
	names := tmpl.ParamNames()
	if len(names) != 6 { // 3 gamma_e + 3 beta_q
		t.Fatalf("ParamNames = %v, want 6 names", names)
	}
	for name, count := range tmpl.ParamStepCounts() {
		if count != 1 {
			t.Errorf("parameter %q drives %d steps, VQE requires exactly 1", name, count)
		}
	}
}

func TestQAOATemplateMultiLayerNames(t *testing.T) {
	tmpl, err := QAOATemplate(3, triangleEdges, 2)
	if err != nil {
		t.Fatalf("QAOATemplate: %v", err)
	}
	want := map[string]bool{"gamma_e0_l0": false, "gamma_e0_l1": false, "beta_q2_l1": false}
	for _, name := range tmpl.ParamNames() {
		if _, ok := want[name]; ok {
			want[name] = true
		}
	}
	for name, seen := range want {
		if !seen {
			t.Errorf("ParamNames missing %q for layers=2", name)
		}
	}
}

func TestCutOfBitstringAndExpectedCut(t *testing.T) {
	got, err := CutOfBitstring(triangleEdges, nil, []int{1, 0, 0})
	if err != nil {
		t.Fatalf("CutOfBitstring: %v", err)
	}
	if got != 2 {
		t.Errorf("cut(|100>) = %g, want 2", got)
	}
	weighted, err := CutOfBitstring(triangleEdges, []float64{2, 1, 1}, []int{1, 0, 0})
	if err != nil {
		t.Fatalf("CutOfBitstring: %v", err)
	}
	if weighted != 3 {
		t.Errorf("weighted cut(|100>) = %g, want 3", weighted)
	}
	if c := ExpectedCut(3, -1); c != 2 {
		t.Errorf("ExpectedCut(3, -1) = %g, want 2", c)
	}
}

func TestQAOAInputValidation(t *testing.T) {
	cases := []struct {
		name    string
		num     int
		edges   [][2]int
		weights []float64
	}{
		{"numQubits too small", 1, [][2]int{{0, 1}}, nil},
		{"no edges", 3, nil, nil},
		{"self-loop", 3, [][2]int{{1, 1}}, nil},
		{"vertex out of range", 3, [][2]int{{0, 3}}, nil},
		{"duplicate edge", 3, [][2]int{{0, 1}, {1, 0}}, nil},
		{"weight length mismatch", 3, triangleEdges, []float64{1}},
	}
	for _, c := range cases {
		// Weight validation only applies to MaxCutHamiltonian; QAOATemplate doesn't take weights
		if _, err := MaxCutHamiltonian(c.num, c.edges, c.weights); err == nil {
			t.Errorf("%s: MaxCutHamiltonian must error", c.name)
		}
		// Skip "weight length mismatch" for QAOATemplate since it doesn't accept weights
		if c.name == "weight length mismatch" {
			continue
		}
		if _, err := QAOATemplate(c.num, c.edges, 1); err == nil {
			t.Errorf("%s: QAOATemplate must error", c.name)
		}
	}
	if _, err := QAOATemplate(3, triangleEdges, 0); err == nil {
		t.Error("layers < 1 must error")
	}
}
