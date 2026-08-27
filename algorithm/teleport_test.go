package algorithm

import (
	"math"
	"math/rand"
	"strings"
	"testing"

	"github.com/pjbaur/quantum/internal/sparsestate"
	"github.com/pjbaur/quantum/quantum"
	"github.com/pjbaur/quantum/state"
)

// teleportRand replays fixed measurement draws (qubit 0 then qubit 1).
type teleportRand struct{ draws []float64 }

func (r *teleportRand) Float64() float64 {
	v := r.draws[0]
	r.draws = r.draws[1:]
	return v
}

// prepareTeleportInput writes a|000> + b|100>: qubit 0 is the least
// significant bit, so |100> is basis index 1. Values are normalized.
func prepareTeleportInput(t *testing.T, s quantum.QuantumState, a, b complex128) {
	t.Helper()
	norm := math.Sqrt(real(a)*real(a) + imag(a)*imag(a) + real(b)*real(b) + imag(b)*imag(b))
	amps := make([]complex128, 8)
	amps[0] = a / complex(norm, 0)
	amps[1] = b / complex(norm, 0)
	setter, ok := s.(quantum.BulkAmplitudeSetter)
	if !ok {
		t.Fatalf("state %T does not implement BulkAmplitudeSetter", s)
	}
	if err := setter.SetAmplitudes(amps); err != nil {
		t.Fatalf("SetAmplitudes: %v", err)
	}
}

// blochOnQubit2 reads qubit 2's Bloch coordinates via expectations.
func blochOnQubit2(t *testing.T, s quantum.QuantumState) (x, y, z float64) {
	t.Helper()
	axes := []quantum.PauliAxis{quantum.PauliI, quantum.PauliI, quantum.PauliX}
	x, err := quantum.Expectation(s, axes)
	if err != nil {
		t.Fatalf("Expectation IIX: %v", err)
	}
	axes[2] = quantum.PauliY
	y, err = quantum.Expectation(s, axes)
	if err != nil {
		t.Fatalf("Expectation IIY: %v", err)
	}
	axes[2] = quantum.PauliZ
	z, err = quantum.Expectation(s, axes)
	if err != nil {
		t.Fatalf("Expectation IIZ: %v", err)
	}
	return x, y, z
}

// analyticBloch returns the Bloch vector of a|0> + b|1>, normalizing first.
func analyticBloch(a, b complex128) (x, y, z float64) {
	norm := math.Sqrt(real(a)*real(a) + imag(a)*imag(a) + real(b)*real(b) + imag(b)*imag(b))
	a /= complex(norm, 0)
	b /= complex(norm, 0)
	x = 2 * (real(a)*real(b) + imag(a)*imag(b))
	y = 2 * (real(a)*imag(b) - imag(a)*real(b))
	z = real(a)*real(a) + imag(a)*imag(a) - real(b)*real(b) - imag(b)*imag(b)
	return x, y, z
}

func TestTeleportAllCorrectionBranches(t *testing.T) {
	// Each draw pair forces one of the four (m0, m1) outcomes, exercising
	// every correction branch: none, X, Z, ZX.
	branches := []struct {
		name  string
		draws []float64
	}{
		{"m0=0 m1=0", []float64{0.1, 0.1}},
		{"m0=0 m1=1", []float64{0.1, 0.9}},
		{"m0=1 m1=0", []float64{0.9, 0.1}},
		{"m0=1 m1=1", []float64{0.9, 0.9}},
	}

	// Input |+_y> = (|0> + i|1>)/sqrt(2): every Bloch component nonzero.
	wantX, wantY, wantZ := analyticBloch(complex(1, 0), complex(0, 1))

	for _, tc := range branches {
		s, err := state.New(3)
		if err != nil {
			t.Fatalf("state.New: %v", err)
		}
		prepareTeleportInput(t, s, complex(1, 0), complex(0, 1))
		s.SetRandSource(&teleportRand{draws: tc.draws})

		result, err := Teleport(s)
		if err != nil {
			t.Fatalf("%s: Teleport: %v", tc.name, err)
		}
		if result != s {
			t.Fatalf("%s: Teleport returned a different state", tc.name)
		}

		gotX, gotY, gotZ := blochOnQubit2(t, result)
		if math.Abs(gotX-wantX) > 1e-10 || math.Abs(gotY-wantY) > 1e-10 || math.Abs(gotZ-wantZ) > 1e-10 {
			t.Errorf("%s: Bloch (%.6f,%.6f,%.6f), want (%.6f,%.6f,%.6f)",
				tc.name, gotX, gotY, gotZ, wantX, wantY, wantZ)
		}

		// Qubits 0 and 1 must be collapsed to the drawn values: no
		// amplitude survives on indices whose low two bits differ from
		// (m0, m1).
		m0 := int(tc.draws[0] + 0.5)
		m1 := int(tc.draws[1] + 0.5)
		for i := 0; i < 8; i++ {
			if i&1 != m0 || (i>>1)&1 != m1 {
				if p := quantum.Probability(result.Amplitude(i)); p > 1e-12 {
					t.Errorf("%s: amplitude on |%03b> survives (p=%v), want measured collapse",
						tc.name, i, p)
				}
			}
		}
	}
}

func TestTeleportPreservesArbitraryStates(t *testing.T) {
	inputs := []struct{ a, b complex128 }{
		{complex(1, 0), complex(0, 0)},          // |0>
		{complex(0, 0), complex(1, 0)},          // |1>
		{complex(1, 0), complex(1, 0)},          // |+>
		{complex(1, 0), complex(0, 1)},          // |+_y>
		{complex(0.6, -0.2), complex(0.5, 0.6)}, // generic
	}

	// Random inputs from a seeded source: deterministic across runs.
	rng := rand.New(rand.NewSource(42))
	for i := 0; i < 10; i++ {
		inputs = append(inputs, struct{ a, b complex128 }{
			complex(rng.NormFloat64(), rng.NormFloat64()),
			complex(rng.NormFloat64(), rng.NormFloat64()),
		})
	}

	for _, in := range inputs {
		s, err := state.New(3)
		if err != nil {
			t.Fatalf("state.New: %v", err)
		}
		prepareTeleportInput(t, s, in.a, in.b)
		s.SetRandSource(&teleportRand{draws: []float64{0.9, 0.1}}) // X branch

		result, err := Teleport(s)
		if err != nil {
			t.Fatalf("Teleport: %v", err)
		}

		wantX, wantY, wantZ := analyticBloch(in.a, in.b)
		gotX, gotY, gotZ := blochOnQubit2(t, result)
		if math.Abs(gotX-wantX) > 1e-10 || math.Abs(gotY-wantY) > 1e-10 || math.Abs(gotZ-wantZ) > 1e-10 {
			t.Errorf("Teleport(%v|0>+%v|1>): Bloch (%.6f,%.6f,%.6f), want (%.6f,%.6f,%.6f)",
				in.a, in.b, gotX, gotY, gotZ, wantX, wantY, wantZ)
		}
	}
}

func TestTeleportWorksWithSparseBackend(t *testing.T) {
	s, err := sparsestate.New(3)
	if err != nil {
		t.Fatalf("sparsestate.New: %v", err)
	}
	prepareTeleportInput(t, s, complex(1, 0), complex(1, 0))
	s.SetRandSource(&teleportRand{draws: []float64{0.1, 0.1}})

	result, err := Teleport(s)
	if err != nil {
		t.Fatalf("Teleport: %v", err)
	}

	// |+> teleported: x=1, y=z=0 on qubit 2.
	gotX, gotY, gotZ := blochOnQubit2(t, result)
	if math.Abs(gotX-1) > 1e-10 || math.Abs(gotY) > 1e-10 || math.Abs(gotZ) > 1e-10 {
		t.Errorf("sparse Teleport |+>: Bloch (%.6f,%.6f,%.6f), want (1,0,0)", gotX, gotY, gotZ)
	}
}

func TestTeleportRejectsBadStates(t *testing.T) {
	if _, err := Teleport(nil); err == nil {
		t.Error("Teleport(nil) = nil error, want error")
	}

	small, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	_, err = Teleport(small)
	if err == nil {
		t.Fatal("Teleport(2 qubits) = nil error, want error")
	}
	if got := err.Error(); !strings.Contains(got, "at least 3 qubits") {
		t.Errorf("Teleport(2 qubits) error = %q, want mention of at least 3 qubits", got)
	}
}
