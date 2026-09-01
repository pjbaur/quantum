/*
This `internal/examples/qpe.go` file demonstrates quantum phase estimation:

1. `QpeDemo()` - Estimates the eigenphase of a Phase gate twice: once with
   a 3-bit-exact phase (deterministic readout) and once with 1/3 (a peaked
   distribution showing the finite-register resolution).
2. `RunAllQpeDemos()` - Runs the demonstration with its banner.

The circuit builder lives in `algorithm/qpe.go`; this file is presentation.
*/

package examples

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/pjbaur/quantum/algorithm"
	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/state"
)

// qpeShotTable prints a shot histogram drawn from the exact counting
// distribution. For an exact eigenstate the counting register holds that
// distribution, so drawing from it reproduces device sampling of the
// register.
func qpeShotTable(probs []float64, shots int, rng *rand.Rand) {
	counts := make([]int, len(probs))
	for i := 0; i < shots; i++ {
		r, threshold := rng.Float64(), 0.0
		for m, p := range probs {
			threshold += p
			if r < threshold {
				counts[m]++
				break
			}
		}
	}
	t := 0
	for len(probs) > 1<<t {
		t++
	}
	fmt.Printf("\n%d shots:\n", shots)
	for m, count := range counts {
		if count > 0 {
			fmt.Printf("  |%0*b> = %3d/%d\n", t, m, count, shots)
		}
	}
}

// qpeRun estimates one phase and prints the exact distribution plus the
// shot table.
func qpeRun(label string, phaseTurns float64) {
	t := 3
	u := gates.NewPhase(2 * math.Pi * phaseTurns)
	eigenstate, err := state.New(1)
	if err != nil {
		fmt.Printf("Error creating eigenstate: %v\n", err)
		return
	}
	if err := eigenstate.ApplyGate(gates.NewPauliX(), 0); err != nil {
		fmt.Printf("Error preparing |1>: %v\n", err)
		return
	}
	probs, err := algorithm.PhaseProbabilities(u, eigenstate, t)
	if err != nil {
		fmt.Printf("Error running phase estimation: %v\n", err)
		return
	}
	best, estimate, err := algorithm.EstimatePhase(u, eigenstate, t)
	if err != nil {
		fmt.Printf("Error estimating phase: %v\n", err)
		return
	}

	fmt.Printf("\n%s: U = Phase(2*pi*%.4f), eigenstate |1>, %d counting qubits\n",
		label, phaseTurns, t)
	fmt.Println("Circuit: H on counting register, controlled-U^(2^j) per qubit j,")
	fmt.Println("         inverse QFT on the counting subregister (gate decomposition,")
	fmt.Println("         since quantum.InverseQFT would mix the eigenstate qubit).")
	fmt.Println("\nExact distribution:")
	for m, p := range probs {
		if p > 1e-9 {
			fmt.Printf("  |%0*b>: %.4f\n", t, m, p)
		}
	}
	qpeShotTable(probs, 200, rand.New(rand.NewSource(11)))
	fmt.Printf("Readout: %d/%d = %.4f (true phase %.4f)\n",
		best, 1<<t, estimate, phaseTurns)
}

// QpeDemo demonstrates quantum phase estimation on a Phase gate.
func QpeDemo() {
	fmt.Println("\n=== Quantum Phase Estimation Demonstration ===")

	qpeRun("Exact 3-bit phase", 3.0/8.0)
	fmt.Println()
	qpeRun("Non-representable phase", 1.0/3.0)
	fmt.Println("\nNote: 1/3 is not representable in 3 bits, so the probability")
	fmt.Println("spreads over neighbors of 0.375; more counting qubits narrow it.")
}

// RunAllQpeDemos executes the phase estimation demonstration.
func RunAllQpeDemos() {
	PrintBanner(45, "    QUANTUM PHASE ESTIMATION DEMONSTRATION")
	QpeDemo()
	PrintRule(45)
}
