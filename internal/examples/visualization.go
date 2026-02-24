package examples

import (
	"fmt"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/qubit"
	"github.com/pjbaur/quantum/state"
	"github.com/pjbaur/quantum/visualization"
)

// StateViewDemo shows a text-based view of a two-qubit Bell state.
func StateViewDemo() {
	fmt.Println("\nState View Demonstration")
	fmt.Println("-------------------------")

	s, err := state.New(2)
	if err != nil {
		fmt.Printf("Error creating state: %v\n", err)
		return
	}
	if err := s.ApplyGate(gates.NewHadamard(), 0); err != nil {
		fmt.Printf("Hadamard error: %v\n", err)
		return
	}
	if err := s.ApplyGate(gates.NewCNOT(), 0, 1); err != nil {
		fmt.Printf("CNOT error: %v\n", err)
		return
	}

	opts := visualization.DefaultStateViewOptions()
	opts.MinProbability = 0.001
	fmt.Println(visualization.FormatStateView(s, opts))
}

// BlochVectorDemo prints a Bloch vector and CSV output for plotting.
func BlochVectorDemo() {
	fmt.Println("\nBloch Vector Demonstration")
	fmt.Println("---------------------------")

	// Use state-vector-first API to apply gates
	s, err := state.New(1)
	if err != nil {
		fmt.Printf("Error creating state: %v\n", err)
		return
	}
	if err := s.ApplyGate(gates.NewHadamard(), 0); err != nil {
		fmt.Printf("Hadamard error: %v\n", err)
		return
	}
	if err := s.ApplyGate(gates.NewS(), 0); err != nil {
		fmt.Printf("S gate error: %v\n", err)
		return
	}

	// Create a qubit from the state amplitudes for visualization
	q, err := qubit.NewWithValues(s.Amplitude(0), s.Amplitude(1))
	if err != nil {
		fmt.Printf("Error creating qubit for visualization: %v\n", err)
		return
	}

	vector := visualization.BlochVectorFromQubit(q)
	fmt.Printf("Vector: %s\n", visualization.FormatBlochVector(vector, 4))
	fmt.Printf("CSV: %s\n", visualization.BlochCSV(vector, 4))
}

// RunAllVisualizationDemos runs all visualization demonstrations.
func RunAllVisualizationDemos() {
	StateViewDemo()
	BlochVectorDemo()
}
