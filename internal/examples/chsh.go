/*
This `internal/examples/chsh.go` file demonstrates the CHSH Bell
inequality:

1. `ChshDemo()` - Prepares a Bell pair, measures the four CHSH correlations
   both exactly and from shots, and assembles the S value that violates the
   classical bound of 2. The exact S sits at the Tsirelson bound
   2*sqrt(2); a finite-shot estimate carries statistical noise and can
   land above that bound (as with seed 7) without new physics.
2. `RunAllChshDemos()` - Runs the demonstration with its banner.

The protocol code lives in `algorithm/chsh.go`; this file is presentation.
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

// ChshDemo demonstrates CHSH inequality violation on a Bell pair.
func ChshDemo() {
	fmt.Println("\n=== CHSH Inequality Demonstration ===")

	s, err := state.New(2)
	if err != nil {
		fmt.Printf("Error creating state: %v\n", err)
		return
	}
	if err := s.ApplyGate(gates.NewHadamard(), 0); err != nil {
		fmt.Printf("Error applying Hadamard: %v\n", err)
		return
	}
	if err := s.ApplyGate(gates.NewCNOT(), 0, 1); err != nil {
		fmt.Printf("Error applying CNOT: %v\n", err)
		return
	}
	fmt.Println("Bell pair |Phi+> = (|00> + |11>)/sqrt(2):")
	printState(s)

	// The four CHSH settings: Alice measures at a0 = 0, a1 = 90 degrees,
	// Bob at b0 = +45, b1 = -45. The 45-degree bases are why the plain
	// {Z, X} settings cannot show a violation: E is cos(thetaA - thetaB),
	// and only rotated bases push S past 2.
	settings := []struct {
		name           string
		thetaA, thetaB float64
	}{
		{"E(a0, b0)", 0, math.Pi / 4},
		{"E(a0, b1)", 0, -math.Pi / 4},
		{"E(a1, b0)", math.Pi / 2, math.Pi / 4},
		{"E(a1, b1)", math.Pi / 2, -math.Pi / 4},
	}
	rng := rand.New(rand.NewSource(7))
	fmt.Println("\nCorrelations (exact vs 400 shots):")
	fmt.Printf("%-12s %-10s %-10s\n", "setting", "exact", "sampled")
	for _, setting := range settings {
		exact, err := algorithm.ChshCorrelation(s, setting.thetaA, setting.thetaB)
		if err != nil {
			fmt.Printf("Error computing %s: %v\n", setting.name, err)
			return
		}
		sampled, err := algorithm.ChshSampledCorrelation(s, setting.thetaA, setting.thetaB, 400, rng)
		if err != nil {
			fmt.Printf("Error sampling %s: %v\n", setting.name, err)
			return
		}
		fmt.Printf("%-12s %-10.4f %-10.4f\n", setting.name, exact, sampled)
	}

	sExact, err := algorithm.ChshSExact(s)
	if err != nil {
		fmt.Printf("Error computing S: %v\n", err)
		return
	}
	sSampled, err := algorithm.ChshSSampled(s, 400, rng)
	if err != nil {
		fmt.Printf("Error sampling S: %v\n", err)
		return
	}
	tsirelson := 2 * math.Sqrt2
	fmt.Printf("\nS = E00 + E01 + E10 - E11\n")
	fmt.Printf("S (exact)   = %.4f\n", sExact)
	fmt.Printf("S (sampled) = %.4f\n", sSampled)
	fmt.Printf("Classical bound: 2     Tsirelson bound: %.4f (exact limit; shot noise can put a sampled S past it)\n", tsirelson)
	if sSampled > 2 {
		fmt.Println("Verdict: classical bound violated - no local hidden variable model explains this.")
	} else {
		fmt.Println("Verdict: shot noise put S under 2 this run; the exact value carries the violation.")
	}
}

// RunAllChshDemos executes the CHSH demonstration.
func RunAllChshDemos() {
	PrintBanner(45, "    CHSH INEQUALITY DEMONSTRATION")
	ChshDemo()
	PrintRule(45)
}
