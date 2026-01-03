package circuit

import (
	"testing"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/state"
)

func BenchmarkCircuitExecute(b *testing.B) {
	circuit, err := buildRepresentativeCircuit(10)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		qState := state.New(10)
		if err := circuit.Execute(qState); err != nil {
			b.Fatal(err)
		}
	}
}

func buildRepresentativeCircuit(numQubits int) (*Circuit, error) {
	circuit, err := New(numQubits)
	if err != nil {
		return nil, err
	}

	hadamard := gates.NewHadamard()
	for i := 0; i < numQubits; i++ {
		if err := circuit.AddGate(hadamard, i); err != nil {
			return nil, err
		}
	}

	cnot := gates.NewCNOT()
	for i := 0; i < numQubits-1; i++ {
		if err := circuit.AddGate(cnot, i, i+1); err != nil {
			return nil, err
		}
	}

	return circuit, nil
}
