package quantum_test

import (
	"math"
	"math/rand"
	"testing"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/internal/sparsestate"
	"github.com/pjbaur/quantum/quantum"
	"github.com/pjbaur/quantum/state"
)

type sequenceRand struct {
	draws []float64
	next  int
}

func (r *sequenceRand) Float64() float64 {
	v := r.draws[r.next]
	r.next++
	return v
}

// bellState prepares (|00> + |11>)/sqrt(2) on the given backend.
func bellState(t *testing.T, s quantum.QuantumState) quantum.QuantumState {
	t.Helper()
	h := gates.NewHadamard()
	cnot := gates.NewCNOT()
	if err := s.ApplyGate(h, 0); err != nil {
		t.Fatalf("ApplyGate H: %v", err)
	}
	if err := s.ApplyGate(cnot, 0, 1); err != nil {
		t.Fatalf("ApplyGate CNOT: %v", err)
	}
	return s
}

func TestSampleDenseBellStateExactOutcomes(t *testing.T) {
	dense, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	s := bellState(t, dense)

	// 0.25 < 0.5 collapses to |00>; 0.75 to |11>.
	counts, err := quantum.Sample(s, 2, &sequenceRand{draws: []float64{0.25, 0.75}})
	if err != nil {
		t.Fatalf("Sample: %v", err)
	}
	if counts["00"] != 1 || counts["11"] != 1 || len(counts) != 2 {
		t.Errorf("counts = %v, want one 00 and one 11", counts)
	}
}

func TestSampleIsNonDestructive(t *testing.T) {
	dense, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	s := bellState(t, dense)

	before0, before3 := s.Amplitude(0), s.Amplitude(3)
	if _, err := quantum.Sample(s, 50, &sequenceRand{draws: repeatedDraws(0.25, 50)}); err != nil {
		t.Fatalf("Sample: %v", err)
	}

	if got := s.Amplitude(0); got != before0 {
		t.Errorf("amplitude 0 changed: %v -> %v", before0, got)
	}
	if got := s.Amplitude(3); got != before3 {
		t.Errorf("amplitude 3 changed: %v -> %v", before3, got)
	}
	if p := s.Probability(0); math.Abs(p-0.5) > 1e-10 {
		t.Errorf("probability of |00> after sampling = %v, want 0.5", p)
	}

	// A second run with the same draws must produce the same histogram,
	// which only holds if sampling never collapsed the state.
	again, err := quantum.Sample(s, 2, &sequenceRand{draws: []float64{0.25, 0.75}})
	if err != nil {
		t.Fatalf("second Sample: %v", err)
	}
	if again["00"] != 1 || again["11"] != 1 {
		t.Errorf("second histogram = %v, want one 00 and one 11", again)
	}
}

func TestSampleSparseMatchesDense(t *testing.T) {
	dense, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	sparse, err := sparsestate.New(2)
	if err != nil {
		t.Fatalf("sparsestate.New: %v", err)
	}

	draws := []float64{0.1, 0.6, 0.4, 0.9, 0.51}
	denseCounts, err := quantum.Sample(bellState(t, dense), 5, &sequenceRand{draws: draws})
	if err != nil {
		t.Fatalf("dense Sample: %v", err)
	}
	sparseCounts, err := quantum.Sample(bellState(t, sparse), 5, &sequenceRand{draws: draws})
	if err != nil {
		t.Fatalf("sparse Sample: %v", err)
	}

	for key, want := range denseCounts {
		if sparseCounts[key] != want {
			t.Errorf("sparse counts[%q] = %d, want %d (dense %v, sparse %v)",
				key, sparseCounts[key], want, denseCounts, sparseCounts)
		}
	}
	if len(sparseCounts) != len(denseCounts) {
		t.Errorf("sparse histogram %v does not match dense %v", sparseCounts, denseCounts)
	}
}

func TestSampleHistogramSumsToShots(t *testing.T) {
	dense, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	s := bellState(t, dense)

	// Seeded source: deterministic sequence, so the assertions below are
	// stable across runs even though the exact counts are not hand-picked.
	counts, err := quantum.Sample(s, 1000, rand.New(rand.NewSource(1)))
	if err != nil {
		t.Fatalf("Sample: %v", err)
	}

	total := 0
	for key, count := range counts {
		if key != "00" && key != "11" {
			t.Errorf("unexpected outcome %q in histogram %v", key, counts)
		}
		total += count
	}
	if total != 1000 {
		t.Errorf("histogram sums to %d, want 1000", total)
	}
	if len(counts) != 2 {
		t.Errorf("histogram = %v, want both 00 and 11 present", counts)
	}
}

func TestSampleDefaultRandomSource(t *testing.T) {
	dense, err := state.New(1)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}

	// A nil source must fall back to the global default rather than panic.
	// |0> has all its probability on outcome "0", whatever the draws are.
	counts, err := quantum.Sample(dense, 10, nil)
	if err != nil {
		t.Fatalf("Sample: %v", err)
	}
	if len(counts) != 1 || counts["0"] != 10 {
		t.Errorf("counts = %v, want map[0:10]", counts)
	}
}

func repeatedDraws(value float64, n int) []float64 {
	draws := make([]float64, n)
	for i := range draws {
		draws[i] = value
	}
	return draws
}
