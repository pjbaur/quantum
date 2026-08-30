/*
This `internal/examples/noise.go` file demonstrates open-system simulation with
the density-matrix backend and its noise channels:

 1. `DephasingDemo()` - Shows a |+⟩ state losing phase coherence under a
    dephasing channel: the Bloch vector shrinks along x while populations stay fixed.
 2. `AmplitudeDampingDemo()` - Shows an excited |1⟩ state relaxing toward |0⟩
    (energy loss, e.g. spontaneous emission).
 3. `DepolarizingDemo()` - Shows a pure state shrinking toward the maximally
    mixed state, tracked via purity Tr(ρ²).
 4. `RunAllNoiseDemos()` - A convenience function that runs all demonstrations.

Unlike the state-vector demos, these require density matrices: noise produces
mixed states that no single state vector can represent. The Bloch vector of a
mixed state lies strictly inside the unit sphere, and purity drops below 1.
*/

package examples

import (
	"fmt"
	"math/cmplx"

	"github.com/pjbaur/quantum/circuit"
	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/internal/density"
	"github.com/pjbaur/quantum/state"
	"github.com/pjbaur/quantum/visualization"
)

// printDensityQubit prints the Bloch vector and purity of the reduced state of
// the target qubit.
func printDensityQubit(m *density.Matrix, target int) {
	x, y, z, err := m.ReducedBlochVector(target)
	if err != nil {
		fmt.Printf("Error computing Bloch vector: %v\n", err)
		return
	}
	v := visualization.BlochVector{X: x, Y: y, Z: z}
	fmt.Printf("  Bloch vector: %s\n", visualization.FormatBlochVector(v, 4))
	fmt.Printf("  purity Tr(ρ²): %.4f\n", m.Purity())
}

// DephasingDemo shows phase coherence decay of |+⟩ under a dephasing channel.
func DephasingDemo() {
	fmt.Println("\n=== Dephasing Channel Demonstration ===")
	fmt.Println("Prepare |+⟩ with a Hadamard, then apply dephasing of increasing strength.")
	fmt.Println("The x-component of the Bloch vector decays as 1-2p; populations are untouched.")

	for _, p := range []float64{0, 0.1, 0.25, 0.5} {
		m, err := density.New(1)
		if err != nil {
			fmt.Printf("Error creating density matrix: %v\n", err)
			return
		}
		if err := m.ApplyGate(gates.NewHadamard(), 0); err != nil {
			fmt.Printf("Error applying Hadamard gate: %v\n", err)
			return
		}
		if err := m.ApplyDephasing(0, p); err != nil {
			fmt.Printf("Error applying dephasing channel: %v\n", err)
			return
		}
		fmt.Printf("\np = %.2f:\n", p)
		printDensityQubit(m, 0)
	}
	fmt.Println("\nAt p = 0.5 the state is maximally mixed: no phase information survives.")
}

// AmplitudeDampingDemo shows |1⟩ relaxing toward |0⟩ under amplitude damping.
func AmplitudeDampingDemo() {
	fmt.Println("\n=== Amplitude Damping Channel Demonstration ===")
	fmt.Println("Prepare |1⟩ with a Pauli-X, then apply amplitude damping (energy loss).")
	fmt.Println("The Bloch z-component climbs from -1 (|1⟩) toward +1 (|0⟩).")

	for _, gamma := range []float64{0, 0.2, 0.5, 0.9} {
		m, err := density.New(1)
		if err != nil {
			fmt.Printf("Error creating density matrix: %v\n", err)
			return
		}
		if err := m.ApplyGate(gates.NewPauliX(), 0); err != nil {
			fmt.Printf("Error applying Pauli-X gate: %v\n", err)
			return
		}
		if err := m.ApplyAmplitudeDamping(0, gamma); err != nil {
			fmt.Printf("Error applying amplitude damping channel: %v\n", err)
			return
		}
		fmt.Printf("\nγ = %.2f:\n", gamma)
		printDensityQubit(m, 0)
	}
	fmt.Println("\nAs γ → 1 the qubit decays fully to the ground state |0⟩.")
}

// DepolarizingDemo shows purity loss of |0⟩ under a depolarizing channel.
func DepolarizingDemo() {
	fmt.Println("\n=== Depolarizing Channel Demonstration ===")
	fmt.Println("Apply a depolarizing channel of increasing strength to |0⟩.")
	fmt.Println("The Bloch vector shrinks toward the origin; purity falls toward 1/2.")

	for _, p := range []float64{0, 0.25, 0.5, 0.75} {
		m, err := density.New(1)
		if err != nil {
			fmt.Printf("Error creating density matrix: %v\n", err)
			return
		}
		if err := m.ApplyDepolarizing(0, p); err != nil {
			fmt.Printf("Error applying depolarizing channel: %v\n", err)
			return
		}
		fmt.Printf("\np = %.2f:\n", p)
		printDensityQubit(m, 0)
	}
	fmt.Println("\nAt p = 0.75 the state is maximally mixed: purity 1/2, Bloch vector at the origin.")
}

// NoisyBellDemo executes one Bell circuit on both a state-vector and a
// density-matrix backend — the same circuit.Execute call — then applies
// depolarizing noise to the density matrix. Populations stay correlated
// while off-diagonal coherence and purity decay: the mixed-state
// signature of a state vector cannot represent.
func NoisyBellDemo() {
	fmt.Println("\n=== Noisy Bell Pair Demonstration ===")
	fmt.Println("One circuit (H, CNOT) executed unchanged on a state vector and a")
	fmt.Println("density matrix, then depolarizing noise on both qubits of the latter.")

	bell, err := circuit.New(2)
	if err != nil {
		fmt.Printf("Error creating circuit: %v\n", err)
		return
	}
	if err := bell.AddGate(gates.NewHadamard(), 0); err != nil {
		fmt.Printf("Error adding Hadamard: %v\n", err)
		return
	}
	if err := bell.AddGate(gates.NewCNOT(), 0, 1); err != nil {
		fmt.Printf("Error adding CNOT: %v\n", err)
		return
	}

	ideal, err := state.New(2)
	if err != nil {
		fmt.Printf("Error creating state: %v\n", err)
		return
	}
	if err := bell.Execute(ideal); err != nil {
		fmt.Printf("Error executing circuit on state vector: %v\n", err)
		return
	}
	fmt.Printf("\nIdeal state vector: P(00) = %.4f, P(11) = %.4f\n",
		ideal.Probability(0), ideal.Probability(3))

	for _, p := range []float64{0, 0.05, 0.2} {
		m, err := density.New(2)
		if err != nil {
			fmt.Printf("Error creating density matrix: %v\n", err)
			return
		}
		if err := bell.Execute(m); err != nil {
			fmt.Printf("Error executing circuit on density matrix: %v\n", err)
			return
		}
		for q := 0; q < 2; q++ {
			if err := m.ApplyDepolarizing(q, p); err != nil {
				fmt.Printf("Error applying depolarizing channel: %v\n", err)
				return
			}
		}
		fmt.Printf("\np = %.2f:\n", p)
		fmt.Printf("  P(00) = %.4f, P(11) = %.4f\n", m.Probability(0), m.Probability(3))
		fmt.Printf("  coherence |ρ(00,11)| = %.4f\n", cmplx.Abs(m.Element(0, 3)))
		fmt.Printf("  purity Tr(ρ²) = %.4f\n", m.Purity())
	}
	fmt.Println("\nNoise leaves the populations correlated while coherence and purity")
	fmt.Println("decay — only a density matrix can carry that mixed state.")
}

// RunAllNoiseDemos runs all noise-channel demonstrations in sequence.
func RunAllNoiseDemos() {
	fmt.Println("\n========================================")
	fmt.Println("      NOISE CHANNEL DEMONSTRATIONS")
	fmt.Println("========================================")

	DephasingDemo()
	AmplitudeDampingDemo()
	DepolarizingDemo()
	NoisyBellDemo()

	fmt.Println("\n=== All noise demonstrations completed ===")
}
