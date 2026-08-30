package algorithm_test

import (
	"testing"

	"github.com/pjbaur/quantum/algorithm"
	"github.com/pjbaur/quantum/circuit"
	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/parameterized"
	"github.com/pjbaur/quantum/state"
)

// BenchmarkVQEIteration measures one bind + execute + energy evaluation,
// the inner cost unit of the driver.
func BenchmarkVQEIteration(b *testing.B) {
	h := algorithm.H2Hamiltonian()
	tmpl := algorithm.H2Ansatz()
	params := parameterized.Params{"theta": 0.3}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		params["theta"] += 0.01 // vary so binds differ
		c, err := tmpl.Bind(params)
		if err != nil {
			b.Fatal(err)
		}
		s, err := state.New(2)
		if err != nil {
			b.Fatal(err)
		}
		if err := c.Execute(s); err != nil {
			b.Fatal(err)
		}
		if _, err := h.Energy(s); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkManualRebuild measures the pre-binding variational pattern from
// backlog item 4: rebuild gates by hand, Reset, re-Execute on one state.
// The gate sequence (X on 1, Ry on 0, CNOT 0→1) mirrors H2Ansatz so the
// two benchmarks differ only in how the circuit reaches the state.
func BenchmarkManualRebuild(b *testing.B) {
	h := algorithm.H2Hamiltonian()
	s, err := state.New(2)
	if err != nil {
		b.Fatal(err)
	}
	theta := 0.3
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		theta += 0.01
		c, err := circuit.New(2)
		if err != nil {
			b.Fatal(err)
		}
		if err := c.AddGate(gates.NewPauliX(), 1); err != nil {
			b.Fatal(err)
		}
		if err := c.AddGate(gates.NewRy(theta), 0); err != nil {
			b.Fatal(err)
		}
		if err := c.AddGate(gates.NewCNOT(), 0, 1); err != nil {
			b.Fatal(err)
		}
		s.Reset()
		if err := c.Execute(s); err != nil {
			b.Fatal(err)
		}
		if _, err := h.Energy(s); err != nil {
			b.Fatal(err)
		}
	}
}
