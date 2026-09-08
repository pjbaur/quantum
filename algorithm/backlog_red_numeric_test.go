//go:build redtests

package algorithm

// Red tests for known open numeric defects in the algorithm helpers. Each
// test below documents a defect tracked as a numbered item in
// docs/enhancement-backlog-2026-08-27.md and is expected to fail until that
// item is closed. The redtests build constraint keeps them out of default
// builds so the regular suite stays green; run them with
//
//	go test -tags redtests ./algorithm -run '^TestRed'
//
// When an item is fixed, its test turns green: move it into the regular
// suite and drop it from here.

import (
	"math"
	"math/big"
	"testing"

	"github.com/pjbaur/quantum/parameterized"
)

// reduceMod2Pi returns x mod 2*pi computed at 256-bit precision so that a
// huge float64 angle can be compared with its in-range equivalent.
func reduceMod2Pi(x float64) float64 {
	const prec = 256
	pi, _, err := big.ParseFloat("3.14159265358979323846264338327950288419716939937510582097494459230781640628620899", 10, prec, big.ToNearestEven)
	if err != nil {
		panic(err)
	}
	twoPi := new(big.Float).SetPrec(prec).Mul(pi, new(big.Float).SetPrec(prec).SetInt64(2))
	bx := new(big.Float).SetPrec(prec).SetFloat64(x)
	q := new(big.Float).SetPrec(prec).Quo(bx, twoPi)
	qi, _ := q.Int(nil)
	whole := new(big.Float).SetPrec(prec).Mul(twoPi, new(big.Float).SetPrec(prec).SetInt(qi))
	r := new(big.Float).SetPrec(prec).Sub(bx, whole)
	f, _ := r.Float64()
	return f
}

// Backlog item 20: parameterShiftGradient returns an exactly zero gradient
// where the +/- pi/2 shift is not representable.
//
// The rule dE/dtheta = (E(theta+pi/2) - E(theta-pi/2))/2 is computed in
// float64. Every rotation is 2*pi-periodic in its angle up to a global
// phase, so the slope at a huge finite angle equals the slope at its
// reduction mod 2*pi.
func TestRedParameterShiftAtHugeAngleMatchesReducedAngle(t *testing.T) {
	h := H2Hamiltonian()
	tmpl := gradientTargetTemplate()
	huge := math.Ldexp(1, 60)
	reduced := reduceMod2Pi(huge)

	eHuge, err := evaluate(h, tmpl, parameterized.Params{"a": huge, "b": 0.1})
	if err != nil {
		t.Fatalf("evaluate at a=%v: %v", huge, err)
	}
	eReduced, err := evaluate(h, tmpl, parameterized.Params{"a": reduced, "b": 0.1})
	if err != nil {
		t.Fatalf("evaluate at a=%v: %v", reduced, err)
	}
	if math.Abs(eHuge-eReduced) > 1e-9 {
		t.Fatalf("setup: E(a=%v) = %v but E(a=%v) = %v; the energy must be 2*pi-periodic in a for this comparison", huge, eHuge, reduced, eReduced)
	}

	want, _, err := parameterShiftGradient(h, tmpl, parameterized.Params{"a": reduced, "b": 0.1}, []string{"a"})
	if err != nil {
		t.Fatalf("gradient at a=%v: %v", reduced, err)
	}
	if math.Abs(want["a"]) < 1e-3 {
		t.Fatalf("setup: slope at the reduced angle is %v, too close to stationary for the comparison", want["a"])
	}
	got, evals, err := parameterShiftGradient(h, tmpl, parameterized.Params{"a": huge, "b": 0.1}, []string{"a"})
	if err != nil {
		t.Fatalf("gradient at a=%v: %v", huge, err)
	}
	if evals != 2 || math.Abs(got["a"]-want["a"]) > 1e-6 {
		t.Errorf("gradient at a=%v: %v after %d evaluations; want %v (the slope at the same angle reduced mod 2*pi, %v) after 2: E(a) is 2*pi-periodic so the slopes must agree", huge, got["a"], evals, want["a"], reduced)
	}
}
