package parameterized_test

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/pjbaur/quantum/circuit"
	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/internal/sparsestate"
	"github.com/pjbaur/quantum/parameterized"
	"github.com/pjbaur/quantum/state"
)

// newTestTemplate returns the canonical mixed template used across tests:
// Ry(theta) on 0, CNOT 0->1, Rx(phi) on 1.
func newTestTemplate() *parameterized.Template {
	t := parameterized.NewTemplate(2)
	if err := t.AddParamGate("theta", parameterized.Ry, 0); err != nil {
		panic(err)
	}
	if err := t.AddGate(gates.NewCNOT(), 0, 1); err != nil {
		panic(err)
	}
	if err := t.AddParamGate("phi", parameterized.Rx, 1); err != nil {
		panic(err)
	}
	return t
}

func TestBindMatchesManuallyBuiltCircuit(t *testing.T) {
	tmpl := newTestTemplate()

	bound, err := tmpl.Bind(parameterized.Params{"theta": 0.4, "phi": 1.1})
	if err != nil {
		t.Fatalf("Bind: %v", err)
	}

	manual, err := circuit.New(2)
	if err != nil {
		t.Fatalf("circuit.New: %v", err)
	}
	if err := manual.AddGate(gates.NewRy(0.4), 0); err != nil {
		t.Fatalf("AddGate Ry: %v", err)
	}
	if err := manual.AddGate(gates.NewCNOT(), 0, 1); err != nil {
		t.Fatalf("AddGate CNOT: %v", err)
	}
	if err := manual.AddGate(gates.NewRx(1.1), 1); err != nil {
		t.Fatalf("AddGate Rx: %v", err)
	}

	// Same output state on dense and sparse backends.
	denseA, _ := state.New(2)
	denseB, _ := state.New(2)
	if err := bound.Execute(denseA); err != nil {
		t.Fatalf("bound.Execute: %v", err)
	}
	if err := manual.Execute(denseB); err != nil {
		t.Fatalf("manual.Execute: %v", err)
	}
	for i := 0; i < 4; i++ {
		a := denseA.Amplitude(i)
		b := denseB.Amplitude(i)
		if a != b {
			t.Fatalf("amplitude %d: bound %v != manual %v", i, a, b)
		}
	}

	sparseA, _ := sparsestate.New(2)
	sparseB, _ := sparsestate.New(2)
	if err := bound.Execute(sparseA); err != nil {
		t.Fatalf("bound.Execute sparse: %v", err)
	}
	if err := manual.Execute(sparseB); err != nil {
		t.Fatalf("manual.Execute sparse: %v", err)
	}
	for i := 0; i < 4; i++ {
		a := sparseA.Amplitude(i)
		b := sparseB.Amplitude(i)
		if a != b {
			t.Fatalf("sparse amplitude %d: bound %v != manual %v", i, a, b)
		}
	}
}

func TestParamNamesFirstUseOrder(t *testing.T) {
	tmpl := parameterized.NewTemplate(3)
	if err := tmpl.AddParamGate("beta", parameterized.Rz, 2); err != nil {
		t.Fatal(err)
	}
	if err := tmpl.AddParamGate("alpha", parameterized.Rx, 0); err != nil {
		t.Fatal(err)
	}
	if err := tmpl.AddParamGate("beta", parameterized.Ry, 1); err != nil {
		t.Fatal(err)
	}
	got := tmpl.ParamNames()
	want := []string{"beta", "alpha"}
	if len(got) != len(want) {
		t.Fatalf("ParamNames() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ParamNames()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestBindErrors(t *testing.T) {
	tmpl := newTestTemplate()

	tests := []struct {
		name    string
		values  parameterized.Params
		wantErr interface{} // pointer to expected error type
	}{
		{"missing phi", parameterized.Params{"theta": 0.1}, &parameterized.MissingParameterError{}},
		{"unknown key", parameterized.Params{"theta": 0.1, "phi": 0.2, "typo": 0.3}, &parameterized.UnknownParameterError{}},
		{"NaN theta", parameterized.Params{"theta": math.NaN(), "phi": 0.2}, &parameterized.InvalidParameterValueError{}},
		{"Inf phi", parameterized.Params{"theta": 0.1, "phi": math.Inf(1)}, &parameterized.InvalidParameterValueError{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tmpl.Bind(tt.values)
			if err == nil {
				t.Fatal("Bind succeeded, want error")
			}
			switch tt.wantErr.(type) {
			case *parameterized.MissingParameterError:
				var e *parameterized.MissingParameterError
				if !errors.As(err, &e) {
					t.Fatalf("err = %T (%v), want MissingParameterError", err, err)
				}
			case *parameterized.UnknownParameterError:
				var e *parameterized.UnknownParameterError
				if !errors.As(err, &e) {
					t.Fatalf("err = %T (%v), want UnknownParameterError", err, err)
				}
			case *parameterized.InvalidParameterValueError:
				var e *parameterized.InvalidParameterValueError
				if !errors.As(err, &e) {
					t.Fatalf("err = %T (%v), want InvalidParameterValueError", err, err)
				}
			}
		})
	}
}

func TestBindErrorMessagesCarryNames(t *testing.T) {
	_, err := newTestTemplate().Bind(parameterized.Params{"theta": 0.1})
	if err == nil || !strings.Contains(err.Error(), "phi") {
		t.Fatalf("err = %v, want message naming %q", err, "phi")
	}
}

func TestAddRejectsOutOfRangeTargets(t *testing.T) {
	tmpl := parameterized.NewTemplate(2)
	if err := tmpl.AddParamGate("theta", parameterized.Ry, 2); err == nil {
		t.Fatal("target 2 on 2-qubit template accepted, want range error")
	}
	if err := tmpl.AddGate(gates.NewCNOT(), 0, 2); err == nil {
		t.Fatal("fixed-gate target 2 accepted, want range error")
	}
}

func TestSameParamDrivesTwoGates(t *testing.T) {
	tmpl := parameterized.NewTemplate(2)
	if err := tmpl.AddParamGate("theta", parameterized.Ry, 0); err != nil {
		t.Fatal(err)
	}
	if err := tmpl.AddParamGate("theta", parameterized.Rz, 1); err != nil {
		t.Fatal(err)
	}
	c, err := tmpl.Bind(parameterized.Params{"theta": 0.7})
	if err != nil {
		t.Fatalf("Bind: %v", err)
	}
	dense, _ := state.New(2)
	if err := c.Execute(dense); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	manual, _ := circuit.New(2)
	_ = manual.AddGate(gates.NewRy(0.7), 0)
	_ = manual.AddGate(gates.NewRz(0.7), 1)
	ref, _ := state.New(2)
	_ = manual.Execute(ref)
	for i := 0; i < 4; i++ {
		a := dense.Amplitude(i)
		b := ref.Amplitude(i)
		if a != b {
			t.Fatalf("amplitude %d: %v != %v", i, a, b)
		}
	}
}

func TestNumQubits(t *testing.T) {
	if got := parameterized.NewTemplate(5).NumQubits(); got != 5 {
		t.Fatalf("NumQubits() = %d, want 5", got)
	}
}

func TestParamStepCounts(t *testing.T) {
	single := parameterized.NewTemplate(2)
	if err := single.AddParamGate("theta", parameterized.Ry, 0); err != nil {
		t.Fatal(err)
	}
	if err := single.AddParamGate("phi", parameterized.Rx, 1); err != nil {
		t.Fatal(err)
	}
	multi := parameterized.NewTemplate(2)
	if err := multi.AddParamGate("theta", parameterized.Ry, 0); err != nil {
		t.Fatal(err)
	}
	if err := multi.AddParamGate("theta", parameterized.Rz, 1); err != nil {
		t.Fatal(err)
	}
	// The empty string is a declared name like any other; a fixed gate in
	// between must not be mistaken for a step it drives, nor it for one.
	empty := parameterized.NewTemplate(2)
	if err := empty.AddParamGate("", parameterized.Ry, 0); err != nil {
		t.Fatal(err)
	}
	if err := empty.AddGate(gates.NewCNOT(), 0, 1); err != nil {
		t.Fatal(err)
	}
	if err := empty.AddParamGate("", parameterized.Rz, 1); err != nil {
		t.Fatal(err)
	}
	fixedOnly := parameterized.NewTemplate(2)
	if err := fixedOnly.AddGate(gates.NewCNOT(), 0, 1); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		tmpl *parameterized.Template
		want map[string]int
	}{
		{"single-use names", single, map[string]int{"theta": 1, "phi": 1}},
		{"name driving two gates", multi, map[string]int{"theta": 2}},
		{"no declared parameters", parameterized.NewTemplate(2), map[string]int{}},
		{"empty name driving two gates around a fixed gate", empty, map[string]int{"": 2}},
		{"fixed gates only", fixedOnly, map[string]int{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.tmpl.ParamStepCounts()
			if len(got) != len(tt.want) {
				t.Fatalf("ParamStepCounts() = %v, want %v", got, tt.want)
			}
			for name, count := range tt.want {
				if got[name] != count {
					t.Fatalf("ParamStepCounts()[%q] = %d, want %d", name, got[name], count)
				}
			}
		})
	}
	// A name the template never declared is absent from the map.
	if _, ok := single.ParamStepCounts()["nope"]; ok {
		t.Fatal(`ParamStepCounts()["nope"] present, want absent`)
	}
}
