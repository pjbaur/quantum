package visualization_test

import (
	"errors"
	"math"
	"testing"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/internal/sparsestate"
	"github.com/pjbaur/quantum/quantum"
	"github.com/pjbaur/quantum/qubit"
	"github.com/pjbaur/quantum/state"
	"github.com/pjbaur/quantum/visualization"
)

func TestBlochVectorFromQubit(t *testing.T) {
	invSqrt2 := 1 / math.Sqrt2

	cases := []struct {
		name     string
		alpha    complex128
		beta     complex128
		expected visualization.BlochVector
	}{
		{
			name:  "zero",
			alpha: 1,
			beta:  0,
			expected: visualization.BlochVector{
				X: 0,
				Y: 0,
				Z: 1,
			},
		},
		{
			name:  "one",
			alpha: 0,
			beta:  1,
			expected: visualization.BlochVector{
				X: 0,
				Y: 0,
				Z: -1,
			},
		},
		{
			name:  "plus",
			alpha: complex(invSqrt2, 0),
			beta:  complex(invSqrt2, 0),
			expected: visualization.BlochVector{
				X: 1,
				Y: 0,
				Z: 0,
			},
		},
		{
			name:  "plus-i",
			alpha: complex(invSqrt2, 0),
			beta:  complex(0, invSqrt2),
			expected: visualization.BlochVector{
				X: 0,
				Y: 1,
				Z: 0,
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			q, err := qubit.NewWithValues(tt.alpha, tt.beta)
			if err != nil {
				t.Fatalf("NewWithValues error: %v", err)
			}

			got := visualization.BlochVectorFromQubit(q)
			if !nearEqual(got.X, tt.expected.X) || !nearEqual(got.Y, tt.expected.Y) || !nearEqual(got.Z, tt.expected.Z) {
				t.Fatalf("Bloch vector mismatch: got=%+v want=%+v", got, tt.expected)
			}
		})
	}
}

func nearEqual(a, b float64) bool {
	const tol = 1e-10
	return math.Abs(a-b) < tol
}

// blochBackends lists the state backends BlochVectorFromState must work
// with; every multi-qubit case below runs against each of them.
var blochBackends = []struct {
	name string
	new  func(numQubits int) (quantum.QuantumState, error)
}{
	{
		name: "dense",
		new: func(numQubits int) (quantum.QuantumState, error) {
			s, err := state.New(numQubits)
			if err != nil {
				return nil, err
			}
			return s, nil
		},
	},
	{
		name: "sparse",
		new: func(numQubits int) (quantum.QuantumState, error) {
			s, err := sparsestate.New(numQubits)
			if err != nil {
				return nil, err
			}
			return s, nil
		},
	},
}

func TestBlochVectorFromState(t *testing.T) {
	cases := []struct {
		name      string
		numQubits int
		prepare   func(t *testing.T, s quantum.QuantumState)
		target    int
		expected  visualization.BlochVector
	}{
		{
			name:      "product_target_0_is_zero_ket",
			numQubits: 2,
			prepare:   prepareProduct,
			target:    0,
			expected:  visualization.BlochVector{X: 0, Y: 0, Z: 1},
		},
		{
			name:      "product_target_1_is_plus_ket",
			numQubits: 2,
			prepare:   prepareProduct,
			target:    1,
			expected:  visualization.BlochVector{X: 1, Y: 0, Z: 0},
		},
		{
			name:      "product_middle_qubit_of_three",
			numQubits: 3,
			prepare:   prepareProduct,
			target:    1,
			expected:  visualization.BlochVector{X: 1, Y: 0, Z: 0},
		},
		{
			name:      "product_top_qubit_of_three",
			numQubits: 3,
			prepare:   prepareProduct,
			target:    2,
			expected:  visualization.BlochVector{X: 0, Y: 0, Z: 1},
		},
		{
			// Either half of a Bell pair is maximally mixed, so its
			// reduced state sits at the centre of the Bloch sphere.
			name:      "bell_pair_target_0_is_maximally_mixed",
			numQubits: 2,
			prepare:   prepareBell,
			target:    0,
			expected:  visualization.BlochVector{X: 0, Y: 0, Z: 0},
		},
		{
			name:      "bell_pair_target_1_is_maximally_mixed",
			numQubits: 2,
			prepare:   prepareBell,
			target:    1,
			expected:  visualization.BlochVector{X: 0, Y: 0, Z: 0},
		},
	}

	for _, tt := range cases {
		for _, backend := range blochBackends {
			t.Run(tt.name+"/"+backend.name, func(t *testing.T) {
				s, err := backend.new(tt.numQubits)
				if err != nil {
					t.Fatalf("%s.New(%d) failed: %v", backend.name, tt.numQubits, err)
				}
				tt.prepare(t, s)

				got, err := visualization.BlochVectorFromState(s, tt.target)
				if err != nil {
					t.Fatalf("BlochVectorFromState(_, %d) failed: %v", tt.target, err)
				}
				if !nearEqual(got.X, tt.expected.X) || !nearEqual(got.Y, tt.expected.Y) || !nearEqual(got.Z, tt.expected.Z) {
					t.Fatalf("Bloch vector mismatch: got=%+v want=%+v", got, tt.expected)
				}
			})
		}
	}
}

func TestBlochVectorFromStateMatchesBlochVectorFromQubit(t *testing.T) {
	invSqrt2 := 1 / math.Sqrt2

	cases := []struct {
		name  string
		alpha complex128
		beta  complex128
	}{
		{name: "zero", alpha: 1, beta: 0},
		{name: "one", alpha: 0, beta: 1},
		{name: "plus", alpha: complex(invSqrt2, 0), beta: complex(invSqrt2, 0)},
		{name: "plus_i", alpha: complex(invSqrt2, 0), beta: complex(0, invSqrt2)},
		{name: "unequal_weights_with_phase", alpha: complex(0.6, 0), beta: complex(0, 0.8)},
	}

	for _, tt := range cases {
		for _, backend := range blochBackends {
			t.Run(tt.name+"/"+backend.name, func(t *testing.T) {
				q, err := qubit.NewWithValues(tt.alpha, tt.beta)
				if err != nil {
					t.Fatalf("NewWithValues error: %v", err)
				}
				want := visualization.BlochVectorFromQubit(q)

				s, err := backend.new(1)
				if err != nil {
					t.Fatalf("%s.New(1) failed: %v", backend.name, err)
				}
				setAmplitudes(t, s, []complex128{tt.alpha, tt.beta})

				got, err := visualization.BlochVectorFromState(s, 0)
				if err != nil {
					t.Fatalf("BlochVectorFromState failed: %v", err)
				}
				if !nearEqual(got.X, want.X) || !nearEqual(got.Y, want.Y) || !nearEqual(got.Z, want.Z) {
					t.Fatalf("Bloch vector mismatch: got=%+v want=%+v", got, want)
				}
			})
		}
	}
}

func TestBlochVectorFromStateErrors(t *testing.T) {
	cases := []struct {
		name       string
		state      func(t *testing.T) quantum.QuantumState
		target     int
		wantsRange bool
	}{
		{
			name:   "nil_state",
			state:  func(*testing.T) quantum.QuantumState { return nil },
			target: 0,
		},
		{
			name:       "negative_target",
			state:      twoQubitState,
			target:     -1,
			wantsRange: true,
		},
		{
			name:       "target_equals_qubit_count",
			state:      twoQubitState,
			target:     2,
			wantsRange: true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := visualization.BlochVectorFromState(tt.state(t), tt.target)
			if err == nil {
				t.Fatalf("BlochVectorFromState(_, %d) error = nil, want an error", tt.target)
			}
			if got != (visualization.BlochVector{}) {
				t.Errorf("BlochVectorFromState(_, %d) = %+v, want the zero vector", tt.target, got)
			}
			if !tt.wantsRange {
				return
			}
			var rangeErr *quantum.QubitsOutOfRangeError
			if !errors.As(err, &rangeErr) {
				t.Fatalf("error = %v, want *quantum.QubitsOutOfRangeError", err)
			}
			if rangeErr.Index != tt.target {
				t.Errorf("error Index = %d, want %d", rangeErr.Index, tt.target)
			}
		})
	}
}

func twoQubitState(t *testing.T) quantum.QuantumState {
	t.Helper()
	s, err := state.New(2)
	if err != nil {
		t.Fatalf("state.New(2) failed: %v", err)
	}
	return s
}

// prepareProduct puts qubit 1 into |+⟩ and leaves every other qubit in |0⟩,
// keeping the register unentangled.
func prepareProduct(t *testing.T, s quantum.QuantumState) {
	t.Helper()
	applyGate(t, s, gates.NewHadamard(), 1)
}

// prepareBell entangles qubits 0 and 1 into (|00⟩+|11⟩)/√2.
func prepareBell(t *testing.T, s quantum.QuantumState) {
	t.Helper()
	applyGate(t, s, gates.NewHadamard(), 0)
	applyGate(t, s, gates.NewCNOT(), 0, 1)
}

func applyGate(t *testing.T, s quantum.QuantumState, gate quantum.Gate, targets ...int) {
	t.Helper()
	if err := s.ApplyGate(gate, targets...); err != nil {
		t.Fatalf("ApplyGate(%s, %v) failed: %v", gate.Name(), targets, err)
	}
}

func setAmplitudes(t *testing.T, s quantum.QuantumState, values []complex128) {
	t.Helper()
	bulk, ok := s.(quantum.BulkAmplitudeSetter)
	if !ok {
		t.Fatalf("backend %T does not implement quantum.BulkAmplitudeSetter", s)
	}
	if err := bulk.SetAmplitudes(values); err != nil {
		t.Fatalf("SetAmplitudes failed: %v", err)
	}
}

func TestFormatBlochVector(t *testing.T) {
	cases := []struct {
		name      string
		vector    visualization.BlochVector
		precision int
		expected  string
	}{
		{
			name: "default_precision",
			vector: visualization.BlochVector{
				X: 0.123456789,
				Y: 0.987654321,
				Z: 0.555555555,
			},
			precision: 4,
			expected:  "x=0.1235 y=0.9877 z=0.5556",
		},
		{
			name: "custom_precision_2",
			vector: visualization.BlochVector{
				X: 1.234,
				Y: 2.345,
				Z: 3.456,
			},
			precision: 2,
			expected:  "x=1.23 y=2.35 z=3.46",
		},
		{
			name: "custom_precision_6",
			vector: visualization.BlochVector{
				X: 0.5,
				Y: 0.5,
				Z: 0.7071,
			},
			precision: 6,
			expected:  "x=0.500000 y=0.500000 z=0.707100",
		},
		{
			name: "zero_precision_defaults_to_4",
			vector: visualization.BlochVector{
				X: 0.111111,
				Y: 0.222222,
				Z: 0.333333,
			},
			precision: 0,
			expected:  "x=0.1111 y=0.2222 z=0.3333",
		},
		{
			name: "negative_precision_defaults_to_4",
			vector: visualization.BlochVector{
				X: 0.111111,
				Y: 0.222222,
				Z: 0.333333,
			},
			precision: -1,
			expected:  "x=0.1111 y=0.2222 z=0.3333",
		},
		{
			name: "negative_values",
			vector: visualization.BlochVector{
				X: -0.5,
				Y: -0.7071,
				Z: -1.0,
			},
			precision: 2,
			expected:  "x=-0.50 y=-0.71 z=-1.00",
		},
		{
			name: "zero_vector",
			vector: visualization.BlochVector{
				X: 0,
				Y: 0,
				Z: 0,
			},
			precision: 2,
			expected:  "x=0.00 y=0.00 z=0.00",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := visualization.FormatBlochVector(tt.vector, tt.precision)
			if got != tt.expected {
				t.Errorf("FormatBlochVector() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestBlochCSV(t *testing.T) {
	cases := []struct {
		name      string
		vector    visualization.BlochVector
		precision int
		expected  string
	}{
		{
			name: "default_precision",
			vector: visualization.BlochVector{
				X: 0.123456789,
				Y: 0.987654321,
				Z: 0.555555555,
			},
			precision: 4,
			expected:  "0.1235,0.9877,0.5556",
		},
		{
			name: "custom_precision_2",
			vector: visualization.BlochVector{
				X: 1.234,
				Y: 2.345,
				Z: 3.456,
			},
			precision: 2,
			expected:  "1.23,2.35,3.46",
		},
		{
			name: "custom_precision_6",
			vector: visualization.BlochVector{
				X: 0.5,
				Y: 0.5,
				Z: 0.7071,
			},
			precision: 6,
			expected:  "0.500000,0.500000,0.707100",
		},
		{
			name: "zero_precision_defaults_to_4",
			vector: visualization.BlochVector{
				X: 0.111111,
				Y: 0.222222,
				Z: 0.333333,
			},
			precision: 0,
			expected:  "0.1111,0.2222,0.3333",
		},
		{
			name: "negative_precision_defaults_to_4",
			vector: visualization.BlochVector{
				X: 0.111111,
				Y: 0.222222,
				Z: 0.333333,
			},
			precision: -1,
			expected:  "0.1111,0.2222,0.3333",
		},
		{
			name: "negative_values",
			vector: visualization.BlochVector{
				X: -0.5,
				Y: -0.7071,
				Z: -1.0,
			},
			precision: 2,
			expected:  "-0.50,-0.71,-1.00",
		},
		{
			name: "zero_vector",
			vector: visualization.BlochVector{
				X: 0,
				Y: 0,
				Z: 0,
			},
			precision: 2,
			expected:  "0.00,0.00,0.00",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := visualization.BlochCSV(tt.vector, tt.precision)
			if got != tt.expected {
				t.Errorf("BlochCSV() = %q, want %q", got, tt.expected)
			}
		})
	}
}
