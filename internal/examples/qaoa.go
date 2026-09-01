/*
This `internal/examples/qaoa.go` file demonstrates QAOA for MaxCut on a
triangle:

1. `QaoaDemo()` - Builds the MaxCut Hamiltonian and the per-edge-parameter
   QAOA template, scans the symmetric (cost angle, mix angle) slice of the
   landscape (bound values are the Rz/Rx rotation angles; the textbook
   gamma/beta is half the angle), hands the best symmetric point to the
   VQE driver, compares against the brute-force optimum, and samples the
   final circuit.
2. `RunAllQaoaDemos()` - Runs the demonstration with its banner.

The builders live in `algorithm/qaoa.go`; this file is presentation.
*/

package examples

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/pjbaur/quantum/algorithm"
	"github.com/pjbaur/quantum/parameterized"
	"github.com/pjbaur/quantum/quantum"
	"github.com/pjbaur/quantum/state"
)

var qaoaTriangleEdges = [][2]int{{0, 1}, {0, 2}, {1, 2}}

// qaoaEnergyAt evaluates the cost at one parameter binding: bind, execute
// on a fresh state, measure the energy.
func qaoaEnergyAt(h *algorithm.Hamiltonian, tmpl *parameterized.Template, params parameterized.Params) (float64, error) {
	c, err := tmpl.Bind(params)
	if err != nil {
		return 0, err
	}
	s, err := state.New(3)
	if err != nil {
		return 0, err
	}
	if err := c.Execute(s); err != nil {
		return 0, err
	}
	return h.Energy(s)
}

// qaoaSymmetricParams expands one (cost angle, mix angle) pair — the Rz/Rx
// rotation angles the template binds directly — to per-parameter values:
// every gamma_e gets costAngle, every beta_q gets mixAngle.
func qaoaSymmetricParams(tmpl *parameterized.Template, costAngle, mixAngle float64) parameterized.Params {
	params := parameterized.Params{}
	for _, name := range tmpl.ParamNames() {
		if len(name) >= 7 && name[:7] == "gamma_e" {
			params[name] = costAngle
		} else {
			params[name] = mixAngle
		}
	}
	return params
}

// QaoaDemo demonstrates QAOA for MaxCut on the triangle graph.
func QaoaDemo() {
	fmt.Println("\n=== QAOA MaxCut Demonstration ===")

	numQubits := 3
	fmt.Println("Graph: triangle on qubits 0, 1, 2 (all edges weight 1)")
	h, err := algorithm.MaxCutHamiltonian(numQubits, qaoaTriangleEdges, nil)
	if err != nil {
		fmt.Printf("Error building Hamiltonian: %v\n", err)
		return
	}
	tmpl, err := algorithm.QAOATemplate(numQubits, qaoaTriangleEdges, 1)
	if err != nil {
		fmt.Printf("Error building template: %v\n", err)
		return
	}
	fmt.Println("Cost: H = Z0Z1 + Z0Z2 + Z1Z2, expected cut = (3 - E)/2")
	fmt.Printf("Template parameters (one gate each, the VQE driver's precondition):\n  %v\n",
		tmpl.ParamNames())

	// Landscape over the symmetric slice of the 6-parameter space. The
	// scanned numbers are Rz/Rx angles, not textbook gamma/beta (which
	// would be half these values).
	fmt.Println("\nLandscape: expected cut on the symmetric slice (cost angle, mix angle),")
	fmt.Println("           all gamma_e = cost angle, all beta_q = mix angle:")
	bestCostAngle, bestMixAngle, bestEnergy := 0.0, 0.0, math.Inf(1)
	fmt.Printf("%-8s", "cost\\mix")
	for j := 0; j <= 8; j++ {
		fmt.Printf(" %6.2f", math.Pi*float64(j)/32)
	}
	fmt.Println()
	for i := 0; i <= 8; i++ {
		costAngle := math.Pi * float64(i) / 16
		fmt.Printf("%-8.4f", costAngle)
		for j := 0; j <= 8; j++ {
			mixAngle := math.Pi * float64(j) / 32
			energy, err := qaoaEnergyAt(h, tmpl, qaoaSymmetricParams(tmpl, costAngle, mixAngle))
			if err != nil {
				fmt.Printf("Error at (cost angle=%g, mix angle=%g): %v\n", costAngle, mixAngle, err)
				return
			}
			if energy < bestEnergy {
				bestCostAngle, bestMixAngle, bestEnergy = costAngle, mixAngle, energy
			}
			fmt.Printf(" %6.2f", algorithm.ExpectedCut(3, energy))
		}
		fmt.Println()
	}
	fmt.Printf("Best symmetric point: cost angle = %.4f, mix angle = %.4f, cut = %.4f\n",
		bestCostAngle, bestMixAngle, algorithm.ExpectedCut(3, bestEnergy))

	result, err := algorithm.VQE(h, tmpl, algorithm.VQEOptions{
		InitialParams: qaoaSymmetricParams(tmpl, bestCostAngle, bestMixAngle),
		MaxIterations: 100,
	})
	if err != nil {
		fmt.Printf("Error running VQE: %v\n", err)
		return
	}
	qaoaCut := algorithm.ExpectedCut(3, result.Energy)
	fmt.Printf("\nVQE from the symmetric start: energy = %.4f, expected cut = %.4f\n",
		result.Energy, qaoaCut)
	fmt.Printf("  iterations accepted: %d, energy evaluations: %d\n",
		result.Iterations, result.Evaluations)

	// Classical brute force.
	best := 0.0
	fmt.Println("\nBrute force over all 8 bitstrings (qubit 0 = LSB):")
	for x := 0; x < 8; x++ {
		bits := []int{x & 1, (x >> 1) & 1, (x >> 2) & 1}
		cut, err := algorithm.CutOfBitstring(qaoaTriangleEdges, nil, bits)
		if err != nil {
			fmt.Printf("Error evaluating cut: %v\n", err)
			return
		}
		if cut > best {
			best = cut
		}
		fmt.Printf("  |%03b>: cut %g\n", x, cut)
	}
	fmt.Printf("Classical optimum: %g, QAOA approximation ratio: %.3f\n",
		best, qaoaCut/best)

	// Sample the optimized circuit.
	c, err := tmpl.Bind(result.Params)
	if err != nil {
		fmt.Printf("Error binding optimal params: %v\n", err)
		return
	}
	s, err := state.New(numQubits)
	if err != nil {
		fmt.Printf("Error creating state: %v\n", err)
		return
	}
	if err := c.Execute(s); err != nil {
		fmt.Printf("Error executing circuit: %v\n", err)
		return
	}
	counts, err := quantum.Sample(s, 200, rand.New(rand.NewSource(5)))
	if err != nil {
		fmt.Printf("Error sampling: %v\n", err)
		return
	}
	fmt.Println("\n200 shots of the optimized circuit (key char 0 = qubit 2):")
	for key, count := range counts {
		b0 := int(key[2] - '0')
		b1 := int(key[1] - '0')
		b2 := int(key[0] - '0')
		cut, err := algorithm.CutOfBitstring(qaoaTriangleEdges, nil, []int{b0, b1, b2})
		if err != nil {
			fmt.Printf("Error evaluating cut: %v\n", err)
			return
		}
		fmt.Printf("  %s: %3d  cut %g\n", key, count, cut)
	}
}

// RunAllQaoaDemos executes the QAOA demonstration.
func RunAllQaoaDemos() {
	PrintBanner(45, "    QAOA MAXCUT DEMONSTRATION")
	QaoaDemo()
	PrintRule(45)
}
