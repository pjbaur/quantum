package algorithm

import (
	"errors"
	"math"

	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/quantum"
)

// chshZZ is the Pauli string both correlation helpers measure after rotating
// each half into the measurement basis.
var chshZZ = []quantum.PauliAxis{quantum.PauliZ, quantum.PauliZ}

// chshRotated returns a clone of s with each half rotated so that measuring
// Z on the clone equals measuring cos(theta)*Z + sin(theta)*X on s. The
// plain Pauli-string API only offers Z/X/Y settings, and those give exactly
// S = 2 on a Bell state — no violation — so the CHSH bases at +-45 degrees
// have to be reached by pre-rotation. The original state is never modified.
func chshRotated(s quantum.QuantumState, thetaA, thetaB float64) (quantum.QuantumState, error) {
	if s == nil {
		return nil, errors.New("state must not be nil")
	}
	rotated := s.Clone()
	if err := rotated.ApplyGate(gates.NewRy(-thetaA), 0); err != nil {
		return nil, err
	}
	if err := rotated.ApplyGate(gates.NewRy(-thetaB), 1); err != nil {
		return nil, err
	}
	return rotated, nil
}

// ChshCorrelation returns the exact correlation E(thetaA, thetaB) =
// <A(thetaA) B(thetaB)> where A/B are Z rotated by thetaA/thetaB in the
// X-Z plane. On the Bell state (|00>+|11>)/sqrt(2) this is
// cos(thetaA - thetaB). thetaA acts on qubit 0, thetaB on qubit 1; s must
// have exactly two qubits.
func ChshCorrelation(s quantum.QuantumState, thetaA, thetaB float64) (float64, error) {
	rotated, err := chshRotated(s, thetaA, thetaB)
	if err != nil {
		return 0, err
	}
	return quantum.Expectation(rotated, chshZZ)
}

// ChshSampledCorrelation estimates the same correlation from shots the way a
// device would, via quantum.SampleExpectation on the rotated clone. rng may
// be nil for the global math/rand source; a seeded source makes the estimate
// reproducible.
func ChshSampledCorrelation(s quantum.QuantumState, thetaA, thetaB float64, shots int, rng quantum.RandomSource) (float64, error) {
	rotated, err := chshRotated(s, thetaA, thetaB)
	if err != nil {
		return 0, err
	}
	return quantum.SampleExpectation(rotated, chshZZ, shots, rng)
}

// chshSettings are the canonical CHSH settings: a0 = 0, a1 = pi/2 for one
// half, b0 = +pi/4, b1 = -pi/4 for the other. On the Bell state they make
// every E = +-sqrt(2)/2 and S = 2*sqrt(2).
var chshSettings = [4]struct{ thetaA, thetaB float64 }{
	{0, math.Pi / 4},
	{0, -math.Pi / 4},
	{math.Pi / 2, math.Pi / 4},
	{math.Pi / 2, -math.Pi / 4},
}

// chshS computes S = E00 + E01 + E10 - E11 from a per-setting correlation
// function, so the exact and sampled variants share one definition.
func chshS(s quantum.QuantumState, correlate func(quantum.QuantumState, float64, float64) (float64, error)) (float64, error) {
	sum := 0.0
	for i, setting := range chshSettings {
		e, err := correlate(s, setting.thetaA, setting.thetaB)
		if err != nil {
			return 0, err
		}
		if i == 3 {
			e = -e
		}
		sum += e
	}
	return sum, nil
}

// ChshSExact returns the exact CHSH S value with the canonical settings.
// 2 <= S <= 2*sqrt(2) on entangled states violates the classical bound 2;
// the Bell state reaches the Tsirelson bound 2*sqrt(2).
func ChshSExact(s quantum.QuantumState) (float64, error) {
	return chshS(s, ChshCorrelation)
}

// ChshSSampled returns the shot-estimated S value with the same settings.
func ChshSSampled(s quantum.QuantumState, shots int, rng quantum.RandomSource) (float64, error) {
	return chshS(s, func(st quantum.QuantumState, a, b float64) (float64, error) {
		return ChshSampledCorrelation(st, a, b, shots, rng)
	})
}
