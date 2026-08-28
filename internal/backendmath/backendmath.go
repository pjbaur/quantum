// Package backendmath holds the state-vector math the dense and sparse
// backends share.
//
// The two backends store amplitudes differently — a dense slice indexed by
// basis state versus a map that prunes near-zero entries — so the loops
// that walk those containers, and the buffers those loops are shaped
// around, are theirs alone. The matrix multiply at the middle of every
// mixing loop is shared here as MixCombos, and so is the rest of the
// physics between the loops: which target qubits a gate may address,
// which basis states a gate mixes, where a measurement leaves the state,
// and the register-width and randomness checks the loops' callers share.
// Keeping one
// implementation of that here means a correction to the physics lands in both
// backends at once, and a divergence between them becomes impossible rather
// than merely unlikely.
package backendmath

import (
	"errors"
	"fmt"
	"math"
	mathrand "math/rand"

	"github.com/pjbaur/quantum/quantum"
)

// minBranchProbability is the probability below which a measurement branch
// is treated as empty rather than collapsed onto. It is far below any
// probability an honest draw can single out: selecting a branch this faint
// needs prob0 to sit within 1e-24 of 1, but float64 resolves values near 1
// only to ~1e-16, so no legitimate outcome is suppressed by the floor. It is
// the square of the sparse backend's prune threshold (1e-12) — the
// probability of the faintest amplitude that backend keeps — so both
// backends treat the same faint branches as carrying no amplitude.
const minBranchProbability = 1e-24

// ErrDuplicateTargets reports a gate applied to the same qubit twice.
var ErrDuplicateTargets = errors.New("targets must be unique")

// ValidateTargets checks that every target qubit of a gate application names
// a qubit of a numQubits-wide register and that no qubit is named twice.
// Targets are reported in the order given, so the first offending target is
// the one described by the error.
func ValidateTargets(targets []int, numQubits int) error {
	// A bitmask tracks the qubits already seen without allocating, but only
	// while every index fits in one word: past 64 qubits the shift would
	// yield zero, so bit is zero and a duplicate is silently accepted, and
	// wide registers pay for a map instead.
	// A dense state that wide is unallocatable and a sparse one that wide is
	// exotic, so the allocating path is effectively unreachable.
	if numQubits <= 64 {
		var seen uint64
		for _, target := range targets {
			if target < 0 || target >= numQubits {
				return outOfRange(target, numQubits)
			}
			bit := uint64(1) << target
			if seen&bit != 0 {
				return ErrDuplicateTargets
			}
			seen |= bit
		}
		return nil
	}

	seen := make(map[int]struct{}, len(targets))
	for _, target := range targets {
		if target < 0 || target >= numQubits {
			return outOfRange(target, numQubits)
		}
		if _, exists := seen[target]; exists {
			return ErrDuplicateTargets
		}
		seen[target] = struct{}{}
	}
	return nil
}

func outOfRange(target, numQubits int) error {
	return &quantum.QubitsOutOfRangeError{
		Index:    target,
		MaxIndex: numQubits - 1,
	}
}

// ComboMasks fills masks with the basis-state offset of every assignment of
// the target qubits and returns the mask of all target bits. It writes
// 2^len(targets) entries, so masks must be at least that long.
//
// Offset combo carries the bits of combo onto the targets, with targets[0]
// taking the most significant bit of combo. That ordering is the convention
// gate matrices are written in, so combo doubles as a row and column index
// into the gate matrix: amplitude base|masks[combo] is the entry that matrix
// row and column combo refer to. A caller walks the basis states with no
// target bit set (those with base&targetMask == 0) and mixes the
// 2^len(targets) amplitudes each one anchors.
//
// The caller owns the slice, which lets a backend applying gates in a loop
// reuse one buffer. That is not only an allocation question: the dense
// backend's inner loops keep their bounds checks eliminated only while the
// slice is shaped in the caller, so returning one from here is not the free
// simplification it looks like. See state.applyMultiQubitGate.
func ComboMasks(masks []int, targets []int) (targetMask int) {
	targetCount := len(targets)
	comboCount := 1 << targetCount

	for _, target := range targets {
		targetMask |= 1 << target
	}

	for combo := 0; combo < comboCount; combo++ {
		mask := 0
		for i, target := range targets {
			shift := targetCount - 1 - i
			if (combo>>shift)&1 == 1 {
				mask |= 1 << target
			}
		}
		masks[combo] = mask
	}

	return targetMask
}

// Collapse is the projection a measurement leaves behind: the outcome that
// was measured, which basis states survive it, and the factor that
// renormalizes them. It is produced by PlanCollapse and then applied by the
// backend to whatever container it keeps its amplitudes in.
type Collapse struct {
	// Outcome is the measured value of the qubit, 0 or 1.
	Outcome int

	mask int     // the measured qubit's bit
	kept int     // the value of that bit in the surviving basis states
	norm float64 // square root of the surviving branch's probability
}

// PlanCollapse chooses the outcome of measuring qubitIndex from the
// probabilities of its two branches and a draw in [0,1), and returns the
// projection that collapses the state onto that outcome.
//
// It returns an error only when neither branch holds any probability, which
// means the state was never normalized and there is nothing to collapse onto.
func PlanCollapse(qubitIndex int, draw, prob0, prob1 float64) (Collapse, error) {
	outcome := 0
	if draw >= prob0 {
		outcome = 1
	}

	// The draw can land on a branch that holds no probability at all.
	// Round-off in prob0 leaves a sliver of the [0,1) draw range pointing
	// at an outcome the state has nothing in, and an amplitude small enough
	// to square to zero still passes the normalization check in the dense
	// backend and is pruned from the map entirely in the sparse one.
	// Collapsing there would divide by a zero normalization factor — filling
	// the dense state with NaN, and leaving every surviving sparse amplitude
	// infinite or the map empty — so measure the other outcome instead: it
	// holds essentially all of the probability, which is what the draw would
	// have selected had prob0 been exact. Both branches empty means the
	// state is not normalized and there is nothing to collapse onto.
	branchProb, otherProb := prob0, prob1
	if outcome == 1 {
		branchProb, otherProb = prob1, prob0
	}
	if branchProb < minBranchProbability {
		if otherProb < minBranchProbability {
			return Collapse{}, fmt.Errorf("measuring qubit %d: neither outcome has any probability (sum %g); state is not normalized",
				qubitIndex, prob0+prob1)
		}
		outcome, branchProb = 1-outcome, otherProb
	}

	return Collapse{
		Outcome: outcome,
		mask:    1 << qubitIndex,
		kept:    outcome << qubitIndex,
		norm:    math.Sqrt(branchProb),
	}, nil
}

// Keeps reports whether basisState survives the collapse. A measurement
// leaves amplitude only where the measured qubit agrees with the outcome;
// everywhere else the amplitude is gone, which the dense backend writes as a
// zero and the sparse backend as an absent map entry.
func (c Collapse) Keeps(basisState int) bool {
	return basisState&c.mask == c.kept
}

// Renormalize scales a surviving amplitude so that the collapsed state is
// normalized again. The surviving branch held only part of the probability,
// and dividing by the square root of that part restores the whole.
//
// The square root is stored as the real number it is and widened here, so the
// division site still sees a constant zero imaginary part — the same code the
// backends generated while this arithmetic lived in them.
func (c Collapse) Renormalize(amplitude complex128) complex128 {
	return amplitude / complex(c.norm, 0)
}

// ValidateQubitCount rejects a non-positive register width with the
// InvalidQubitCountError both backends' constructors return.
func ValidateQubitCount(numQubits int) error {
	if numQubits <= 0 {
		return &quantum.InvalidQubitCountError{
			Requested: numQubits,
			Reason:    "must be positive",
		}
	}
	return nil
}

// RandFloat64 draws from src, or from the global math/rand source when
// src is nil — the fallback both backends' Measure paths share.
func RandFloat64(src quantum.RandomSource) float64 {
	if src != nil {
		return src.Float64()
	}
	return mathrand.Float64()
}

// MixCombos computes one base's worth of gate output, outputs = matrix ·
// inputs, where inputs and outputs hold the 2^k amplitudes a gate mixes
// for one anchor basis state and combo doubles as the matrix row/column
// index (see ComboMasks for the convention).
//
// The caller owns and shapes the buffers — matrix must be len(outputs)
// rows by len(inputs) columns — because the dense backend's inner loops
// keep their bounds checks eliminated only while the slices it iterates
// are shaped in the caller; see state.applyMultiQubitGate.
func MixCombos(matrix [][]complex128, inputs, outputs []complex128) {
	for row := 0; row < len(outputs); row++ {
		sum := complex(0, 0)
		for col := 0; col < len(inputs); col++ {
			sum += matrix[row][col] * inputs[col]
		}
		outputs[row] = sum
	}
}
