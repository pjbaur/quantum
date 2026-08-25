package backendmath

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/pjbaur/quantum/quantum"
)

func TestValidateTargets(t *testing.T) {
	// want is the rejection expected: "" for none, "range" for a
	// QubitsOutOfRangeError naming wantIndex, "duplicate" for a repeated qubit.
	tests := []struct {
		name      string
		numQubits int
		targets   []int
		want      string
		wantIndex int
	}{
		{name: "single target", numQubits: 3, targets: []int{2}},
		{name: "distinct targets", numQubits: 4, targets: []int{3, 0, 1}},
		{name: "no targets", numQubits: 4, targets: nil},
		{name: "negative index", numQubits: 3, targets: []int{0, -1}, want: "range", wantIndex: -1},
		{name: "index at width", numQubits: 3, targets: []int{3}, want: "range", wantIndex: 3},
		{name: "duplicate", numQubits: 3, targets: []int{1, 2, 1}, want: "duplicate"},
		{name: "range beats duplicate", numQubits: 3, targets: []int{5, 5}, want: "range", wantIndex: 5},
		// Past 64 qubits the bitmask would wrap, so validation falls back to
		// a map. These two pin that the fallback still decides both ways.
		{name: "wide register distinct", numQubits: 100, targets: []int{70, 71}},
		{name: "wide register duplicate", numQubits: 100, targets: []int{70, 70}, want: "duplicate"},
		{name: "exactly 64 qubits", numQubits: 64, targets: []int{63, 0}},
		{name: "exactly 64 qubits duplicate", numQubits: 64, targets: []int{63, 63}, want: "duplicate"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTargets(tt.targets, tt.numQubits)

			switch tt.want {
			case "":
				if err != nil {
					t.Fatalf("ValidateTargets(%v, %d) = %v, want nil", tt.targets, tt.numQubits, err)
				}
			case "duplicate":
				if !errors.Is(err, ErrDuplicateTargets) {
					t.Fatalf("ValidateTargets(%v, %d) = %v, want ErrDuplicateTargets", tt.targets, tt.numQubits, err)
				}
			case "range":
				var rangeErr *quantum.QubitsOutOfRangeError
				if !errors.As(err, &rangeErr) {
					t.Fatalf("ValidateTargets(%v, %d) = %v, want a QubitsOutOfRangeError", tt.targets, tt.numQubits, err)
				}
				if rangeErr.Index != tt.wantIndex {
					t.Errorf("error names index %d, want %d", rangeErr.Index, tt.wantIndex)
				}
				if want := tt.numQubits - 1; rangeErr.MaxIndex != want {
					t.Errorf("error names max index %d, want %d", rangeErr.MaxIndex, want)
				}
			}
		})
	}
}

func TestComboMasks(t *testing.T) {
	tests := []struct {
		name           string
		targets        []int
		wantMasks      []int
		wantTargetMask int
	}{
		{name: "single target", targets: []int{2}, wantMasks: []int{0, 4}, wantTargetMask: 4},
		// targets[0] takes the most significant bit of combo, so combo 1
		// (binary 01) sets targets[1] — qubit 1 — and not qubit 0.
		{name: "two targets in order", targets: []int{0, 1}, wantMasks: []int{0, 2, 1, 3}, wantTargetMask: 3},
		{name: "two targets reversed", targets: []int{1, 0}, wantMasks: []int{0, 1, 2, 3}, wantTargetMask: 3},
		{name: "two spread targets", targets: []int{3, 1}, wantMasks: []int{0, 2, 8, 10}, wantTargetMask: 10},
		{name: "three targets", targets: []int{0, 1, 2}, wantMasks: []int{0, 4, 2, 6, 1, 5, 3, 7}, wantTargetMask: 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			masks := make([]int, len(tt.wantMasks))
			targetMask := ComboMasks(masks, tt.targets)

			for combo, want := range tt.wantMasks {
				if masks[combo] != want {
					t.Errorf("mask for combo %d = %d, want %d (all masks %v)", combo, masks[combo], want, masks)
				}
			}
			if targetMask != tt.wantTargetMask {
				t.Errorf("target mask = %d, want %d", targetMask, tt.wantTargetMask)
			}
		})
	}
}

// TestComboMasksWritesOnlyItsEntries pins the buffer contract the dense
// backend relies on: it hands over one long-lived slice sized for the widest
// gate it has seen, so a narrower gate must write only its own 2^n entries.
func TestComboMasksWritesOnlyItsEntries(t *testing.T) {
	masks := []int{-1, -1, -1, -1, -1, -1, -1, -1}

	if got := ComboMasks(masks, []int{0, 1}); got != 3 {
		t.Errorf("target mask = %d, want 3", got)
	}
	for combo, want := range []int{0, 2, 1, 3} {
		if masks[combo] != want {
			t.Errorf("mask for combo %d = %d, want %d", combo, masks[combo], want)
		}
	}
	for combo := 4; combo < len(masks); combo++ {
		if masks[combo] != -1 {
			t.Errorf("entry %d beyond the gate's width was overwritten with %d", combo, masks[combo])
		}
	}
}

func TestPlanCollapse(t *testing.T) {
	tests := []struct {
		name        string
		draw        float64
		prob0       float64
		prob1       float64
		wantOutcome int
		wantNorm    float64 // the probability whose square root normalizes
	}{
		{name: "draw below prob0", draw: 0.4, prob0: 0.5, prob1: 0.5, wantOutcome: 0, wantNorm: 0.5},
		{name: "draw at prob0", draw: 0.5, prob0: 0.5, prob1: 0.5, wantOutcome: 1, wantNorm: 0.5},
		{name: "draw above prob0", draw: 0.9, prob0: 0.25, prob1: 0.75, wantOutcome: 1, wantNorm: 0.75},
		{name: "certain outcome 0", draw: 0.999, prob0: 1, prob1: 0, wantOutcome: 0, wantNorm: 1},
		// The guard: the draw picks a branch that holds nothing, so the
		// measurement lands on the branch that holds everything, and
		// renormalizes by that branch's probability rather than the empty
		// one's. Both directions of the redirect are covered.
		{name: "redirect to 0", draw: math.Nextafter(1, 0), prob0: 1 - 1e-16, prob1: 0, wantOutcome: 0, wantNorm: 1 - 1e-16},
		{name: "redirect to 1", draw: 0, prob0: 1e-30, prob1: 1, wantOutcome: 1, wantNorm: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collapse, err := PlanCollapse(1, tt.draw, tt.prob0, tt.prob1)
			if err != nil {
				t.Fatalf("PlanCollapse failed: %v", err)
			}
			if collapse.Outcome != tt.wantOutcome {
				t.Fatalf("outcome = %d, want %d", collapse.Outcome, tt.wantOutcome)
			}

			// Renormalize divides by the square root of the surviving
			// branch's probability, so an amplitude equal to that square
			// root comes back as 1.
			amplitude := complex(math.Sqrt(tt.wantNorm), 0)
			if got := collapse.Renormalize(amplitude); math.Abs(real(got)-1) > 1e-12 {
				t.Errorf("Renormalize(%v) = %v, want 1 (branch probability %g)", amplitude, got, tt.wantNorm)
			}
		})
	}
}

// TestPlanCollapseKeeps checks the projection itself: only basis states whose
// measured qubit agrees with the outcome survive, and the qubit index is the
// only bit that matters.
func TestPlanCollapseKeeps(t *testing.T) {
	const qubitIndex = 2

	for _, outcome := range []int{0, 1} {
		draw := 0.0
		if outcome == 1 {
			draw = 0.9
		}
		collapse, err := PlanCollapse(qubitIndex, draw, 0.5, 0.5)
		if err != nil {
			t.Fatalf("PlanCollapse failed: %v", err)
		}
		if collapse.Outcome != outcome {
			t.Fatalf("outcome = %d, want %d", collapse.Outcome, outcome)
		}

		for basisState := 0; basisState < 16; basisState++ {
			bit := (basisState >> qubitIndex) & 1
			want := bit == outcome
			if got := collapse.Keeps(basisState); got != want {
				t.Errorf("outcome %d: Keeps(%d) = %v, want %v", outcome, basisState, got, want)
			}
		}
	}
}

// TestPlanCollapseNoProbability covers the guard's other arm: with both
// branches empty there is nothing to collapse onto, so the caller is told the
// state is not normalized rather than handed a state full of NaN.
func TestPlanCollapseNoProbability(t *testing.T) {
	collapse, err := PlanCollapse(3, 0.5, 0, 0)
	if err == nil {
		t.Fatal("PlanCollapse with two empty branches returned no error, want one")
	}
	if collapse != (Collapse{}) {
		t.Errorf("failed PlanCollapse returned %+v, want the zero Collapse", collapse)
	}
	if want := "measuring qubit 3"; !strings.Contains(err.Error(), want) {
		t.Errorf("error %q does not name the qubit (%q)", err, want)
	}
	if want := "not normalized"; !strings.Contains(err.Error(), want) {
		t.Errorf("error %q does not name the broken invariant (%q)", err, want)
	}
}

// TestPlanCollapseFloorIsPruneEpsilonSquared pins the floor to the sparse
// backend's prune threshold. The two are the same value by design: the
// faintest amplitude that backend keeps is the faintest branch this guard
// treats as populated.
func TestPlanCollapseFloorIsPruneEpsilonSquared(t *testing.T) {
	const pruneEpsilon = 1e-12
	if minBranchProbability != pruneEpsilon*pruneEpsilon {
		t.Errorf("minBranchProbability = %g, want %g", minBranchProbability, pruneEpsilon*pruneEpsilon)
	}

	// Just above the floor the branch is collapsible; just below it the
	// measurement is redirected to the other outcome.
	above := math.Nextafter(minBranchProbability, 1)
	collapse, err := PlanCollapse(0, 0, above, 1)
	if err != nil {
		t.Fatalf("PlanCollapse failed: %v", err)
	}
	if collapse.Outcome != 0 {
		t.Errorf("branch probability %g was treated as empty, want outcome 0", above)
	}

	below := math.Nextafter(minBranchProbability, 0)
	collapse, err = PlanCollapse(0, 0, below, 1)
	if err != nil {
		t.Fatalf("PlanCollapse failed: %v", err)
	}
	if collapse.Outcome != 1 {
		t.Errorf("branch probability %g was collapsed onto, want a redirect to outcome 1", below)
	}
}
