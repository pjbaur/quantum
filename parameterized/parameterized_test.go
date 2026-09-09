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
	"github.com/pjbaur/quantum/quantum"
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

// TestBindWithEmptyNameParameter pins the Bind half of "the same test
// Bind applies" (see ParamStepCounts's doc comment) directly in this
// package, rather than only transitively through algorithm's VQE tests:
// "" is an ordinary map key to Bind, driving two gates around a fixed
// gate, the same shape as TestParamStepCounts's "empty" fixture.
func TestBindWithEmptyNameParameter(t *testing.T) {
	tmpl := parameterized.NewTemplate(2)
	if err := tmpl.AddParamGate("", parameterized.Ry, 0); err != nil {
		t.Fatal(err)
	}
	if err := tmpl.AddGate(gates.NewCNOT(), 0, 1); err != nil {
		t.Fatal(err)
	}
	if err := tmpl.AddParamGate("", parameterized.Rz, 1); err != nil {
		t.Fatal(err)
	}
	bound, err := tmpl.Bind(parameterized.Params{"": 0.7})
	if err != nil {
		t.Fatalf("Bind: %v", err)
	}
	dense, _ := state.New(2)
	if err := bound.Execute(dense); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	manual, _ := circuit.New(2)
	_ = manual.AddGate(gates.NewRy(0.7), 0)
	_ = manual.AddGate(gates.NewCNOT(), 0, 1)
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

// TestZeroValueTemplateAddParamGateDoesNotPanic pins that a Template
// declared without NewTemplate is safe to call (backlog item 18). Before
// the fix AddParamGate wrote to the nil seen map unconditionally and
// panicked with "assignment to entry in nil map"; the map is now allocated
// on the first declaration, so the call errors or succeeds like any other.
func TestZeroValueTemplateAddParamGateDoesNotPanic(t *testing.T) {
	var tmpl parameterized.Template
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("AddParamGate on a zero-value Template panicked: %v; want an error or success, never a panic (NewTemplate is not documented as required)", r)
		}
	}()
	// A zero-value template has no qubits, so any target is out of range,
	// and a call with no targets is rejected too (backlog item 19); either
	// way the call must return an error, never panic.
	err := tmpl.AddParamGate("", parameterized.Ry)
	if err == nil {
		if names := tmpl.ParamNames(); len(names) != 1 || names[0] != "" {
			t.Fatalf("after an accepted AddParamGate(%q): ParamNames() = %q, want [%q]", "", names, "")
		}
		if counts := tmpl.ParamStepCounts(); counts[""] != 1 {
			t.Fatalf("after an accepted AddParamGate(%q): ParamStepCounts() = %v, want map[\"\":1]", "", counts)
		}
	}
}

// TestZeroValueTemplateIsAZeroQubitTemplate pins what the zero value is:
// the template NewTemplate(0) returns. It declares nothing, rejects a
// declaration with an out-of-range target or with none, leaves nothing
// declared after a rejection, and Bind fails the way circuit.New fails
// for a qubit count of zero, after its own parameter checks. This test
// only exercises the out-of-range case (target 0 on a 0-qubit template);
// the no-target case on the zero value is
// TestZeroValueTemplateAddParamGateDoesNotPanic's. Nothing here depends
// on AddParamGate accepting a call with no targets, so the test keeps its
// meaning once such calls are rejected (backlog item 19).
func TestZeroValueTemplateIsAZeroQubitTemplate(t *testing.T) {
	var tmpl parameterized.Template
	if got := tmpl.NumQubits(); got != 0 {
		t.Fatalf("NumQubits() = %d, want 0", got)
	}
	if got := tmpl.ParamNames(); len(got) != 0 {
		t.Fatalf("ParamNames() = %q, want none", got)
	}
	if got := tmpl.ParamStepCounts(); len(got) != 0 {
		t.Fatalf("ParamStepCounts() = %v, want none", got)
	}
	var rangeErr *quantum.QubitsOutOfRangeError
	if err := tmpl.AddParamGate("theta", parameterized.Ry, 0); !errors.As(err, &rangeErr) {
		t.Fatalf("AddParamGate(%q, Ry, 0) on a zero-value Template: err = %v, want QubitsOutOfRangeError", "theta", err)
	}
	if err := tmpl.AddGate(gates.NewHadamard(), 0); !errors.As(err, &rangeErr) {
		t.Fatalf("AddGate(H, 0) on a zero-value Template: err = %v, want QubitsOutOfRangeError", err)
	}
	if got := tmpl.ParamNames(); len(got) != 0 {
		t.Fatalf("ParamNames() after rejected declarations = %q, want none", got)
	}
	var unknownErr *parameterized.UnknownParameterError
	if _, err := tmpl.Bind(parameterized.Params{"theta": 0.1}); !errors.As(err, &unknownErr) {
		t.Fatalf("Bind with an undeclared name on a zero-value Template: err = %v, want UnknownParameterError", err)
	}
	var countErr *quantum.InvalidQubitCountError
	if _, err := tmpl.Bind(parameterized.Params{}); !errors.As(err, &countErr) {
		t.Fatalf("Bind on a zero-value Template: err = %v, want InvalidQubitCountError, the error circuit.New returns for zero qubits", err)
	}
}

// TestAddRejectsNoTargets pins that a gate declared with zero targets is
// rejected at the Add*Gate call (backlog item 19). Before the fix
// checkTargets passed vacuously on an empty list, so the step was declared,
// counted by ParamNames and ParamStepCounts, and only Bind failed, deep
// inside circuit.AddGate.
func TestAddRejectsNoTargets(t *testing.T) {
	tmpl := parameterized.NewTemplate(2)
	err := tmpl.AddParamGate("", parameterized.Ry)
	if err == nil {
		_, bindErr := tmpl.Bind(parameterized.Params{"": 0.1})
		t.Fatalf("AddParamGate(%q, Ry) with no targets accepted: ParamNames() = %q, ParamStepCounts() = %v; Bind then fails with %v; want the step rejected when added", "", tmpl.ParamNames(), tmpl.ParamStepCounts(), bindErr)
	}
	fixed := parameterized.NewTemplate(2)
	if err := fixed.AddGate(gates.NewHadamard()); err == nil {
		_, bindErr := fixed.Bind(parameterized.Params{})
		t.Fatalf("AddGate(H) with no targets accepted; Bind then fails with %v; want the step rejected when added", bindErr)
	}
}

// TestAddNoTargetsErrorNamesTheGateAndDeclaresNothing pins the shape of
// the rejection: the error names the parameter or the gate, the template
// is left as it was, and a nil factory is still reported before the
// missing targets, since it is the earlier argument.
func TestAddNoTargetsErrorNamesTheGateAndDeclaresNothing(t *testing.T) {
	tmpl := parameterized.NewTemplate(2)
	if err := tmpl.AddParamGate("theta", parameterized.Ry); err == nil || !strings.Contains(err.Error(), `"theta"`) {
		t.Fatalf("AddParamGate(%q, Ry) with no targets: err = %v, want an error naming the parameter", "theta", err)
	}
	if names := tmpl.ParamNames(); len(names) != 0 {
		t.Fatalf("ParamNames() after a rejected declaration = %q, want none", names)
	}
	if counts := tmpl.ParamStepCounts(); len(counts) != 0 {
		t.Fatalf("ParamStepCounts() after a rejected declaration = %v, want none", counts)
	}
	if err := tmpl.AddGate(gates.NewHadamard()); err == nil || !strings.Contains(err.Error(), "Hadamard") {
		t.Fatalf("AddGate(H) with no targets: err = %v, want an error naming the gate", err)
	}
	// Neither rejected step was appended, so the template still binds.
	if _, err := tmpl.Bind(parameterized.Params{}); err != nil {
		t.Fatalf("Bind after rejected declarations: %v, want success on an empty template", err)
	}
	if err := tmpl.AddParamGate("theta", nil); err == nil || !strings.Contains(err.Error(), "factory must not be nil") {
		t.Fatalf("AddParamGate(%q, nil) with no targets: err = %v, want the nil-factory error first", "theta", err)
	}
}

// namedGate is a minimal quantum.Gate whose Name() is whatever the test
// chooses, including the empty string, which gates.NewMatrixGate refuses;
// it lets a test exercise a gate name gates.NewHadamard and friends never
// produce.
type namedGate struct{ name string }

func (g namedGate) Name() string { return g.name }
func (g namedGate) Matrix() [][]complex128 {
	return [][]complex128{{1, 0}, {0, 1}}
}

// TestAddNoTargetsErrorRendersEmptyGateNameVisibly pins that the no-target
// error names the gate even when Name() is empty (backlog item 19, round
// 1 fix). The gate name is %q-quoted, as the parameter name already is,
// so an empty name still identifies something rather than vanishing into
// "fixed gate : at least one target is required", which names no gate at
// all.
func TestAddNoTargetsErrorRendersEmptyGateNameVisibly(t *testing.T) {
	tmpl := parameterized.NewTemplate(2)
	err := tmpl.AddGate(namedGate{name: ""})
	if err == nil {
		t.Fatal("AddGate(gate with empty name) with no targets accepted, want an error")
	}
	if strings.Contains(err.Error(), "gate :") || !strings.Contains(err.Error(), `""`) {
		t.Fatalf("AddGate(gate with empty name) with no targets: err = %q; the empty name is not rendered, so the error names no gate", err.Error())
	}
}
