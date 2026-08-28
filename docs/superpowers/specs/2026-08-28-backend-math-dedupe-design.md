# Backend Math Residual Dedupe — Design

Date: 2026-08-28
Backlog: `docs/enhancement-backlog-2026-08-27.md` item 8
Status: approved

## Problem

`internal/backendmath` already shares the gate-application physics that
`state` (dense) and `internal/sparsestate` agree on — `ValidateTargets`,
`ComboMasks`, `PlanCollapse` — but near-verbatim math remains in both
backends:

1. The `SetAmplitudes` validation prologue (length check, finite scan,
   tolerance check, `NormalizationError` construction) is verbatim.
2. The `SetAmplitude` write-check-rollback scaffold repeats in both, with
   only the container write/rollback differing.
3. `isNormalized` is byte-identical; `probabilitySum` differs only in
   iterating a slice versus a map.
4. The multi-qubit gather/matmul/scatter inner loops are near-verbatim
   (the sparse scatter additionally prunes near-zero amplitudes).
5. The single-qubit 2×2 mixing loops share only their two-line formula;
   their structures differ (dense walks every basis state, sparse groups
   bases).

Adjacent residuals accepted into scope:

- The 1e-10 normalization tolerance is declared independently in five
  packages (`quantum/sample.go`, `state`, `internal/sparsestate`, `qubit`,
  `algorithm`), each cross-referencing the others by prose comment.
- `randFloat64` and the `New` qubit-count validation are verbatim in both
  backends.

The dense-vs-sparse equivalence fuzz (`FuzzDenseSparseGateEquivalence`)
guards the duplication today; the goal is that a correction lands in both
backends at once because it is written once.

## Constraint

The dense backend's `applyMultiQubitGate` is bounds-check-elimination
sensitive, with an in-code "Re-benchmark before changing this" note: a
previous attempt to reshape its buffers cost up to 10% on a 3-qubit gate.
Any shared kernel must accept caller-shaped, caller-owned buffers and
leave the dense loop structure intact, verified by benchmark.

## Approach

Chosen from three candidates:

- **A (chosen): policy in `quantum`, kernel in `backendmath`.**
  Amplitude-vector policy joins `IsFiniteAmplitude`/`Probability` in
  `quantum`, which every package can import. The matmul kernel joins the
  existing physics helpers in `backendmath`.
- **B: all helpers in `backendmath`.** Rejected: `quantum` cannot import
  `backendmath` (it imports `quantum`), so the tolerance const would still
  need a `quantum` home and the policy would have two doorways.
- **C: A plus an accessor-interface transaction layer** so the whole
  `SetAmplitude` body is written once. Rejected: `delete` versus
  "write zero" is awkward for the dense backend, sparse `probabilitySum`
  must stay map-iteration to remain O(nonzero), and after A's helpers each
  `SetAmplitude` is ~10 genuinely container-specific lines — interface
  ceremony buys little on a cold path.

## Design

### New `quantum` helpers — `quantum/normalization.go`

```go
// NormalizationTolerance is the window around 1 within which a state's
// probability sum counts as normalized. One declaration replaces five.
const NormalizationTolerance = 1e-10

// IsNormalizedSum reports whether sum is a normalized probability sum.
func IsNormalizedSum(sum float64) bool
// math.Abs(sum-1) <= NormalizationTolerance

// CheckNormalization returns nil for a normalized sum, or the
// NormalizationError describing the attempt.
func CheckNormalization(attemptedSum, currentSum float64) error
// &NormalizationError{AttemptedSum: attemptedSum, CurrentSum: currentSum}

// ValidateAmplitudeVector checks a candidate amplitude vector for a
// size-wide register and returns its probability sum.
func ValidateAmplitudeVector(values []complex128, size int) (sum float64, err error)
// len mismatch → fmt.Errorf("values slice length %d does not match state size %d", len(values), size)
//   (byte-identical to both backends' current message)
// first non-finite → &NonFiniteAmplitudeError{BasisState: i, Value: v}
// else → Σ|v|²
```

Consumers: both backends' `SetAmplitudes`/`SetAmplitude`,
`quantum/sample.go` (its private tolerance const is deleted),
`qubit.IsNormalized`.

`algorithm.groundStateTolerance` is deliberately untouched: it bounds a
max-amplitude-diff comparison, not a probability sum — the shared value
is coincidental.

### New `backendmath` helpers

```go
// MixCombos computes outputs = matrix · inputs for one base's worth of
// gate-mixed amplitudes. Caller owns and shapes all buffers; matrix must
// be len(outputs) × len(inputs).
func MixCombos(matrix [][]complex128, inputs, outputs []complex128)

// ValidateQubitCount rejects a non-positive register width.
func ValidateQubitCount(numQubits int) error
// &quantum.InvalidQubitCountError{Requested: n, Reason: "must be positive"}

// RandFloat64 draws from src, or the global source when src is nil.
func RandFloat64(src quantum.RandomSource) float64
```

The `backendmath` package doc is amended: loops over containers remain
per-backend, but the matrix-multiply middle those loops share now lives
here (replacing the current "their loops … are theirs alone" claim).

### Backend rewiring

| Site | After |
|---|---|
| `New` ×2 | `backendmath.ValidateQubitCount` guard |
| `randFloat64` method ×2 | deleted; call sites use `backendmath.RandFloat64(s.randSource)` |
| `SetAmplitudes` prologue ×2 | `sum, err := quantum.ValidateAmplitudeVector(values, size)` then `quantum.CheckNormalization(sum, s.probabilitySum())`; container write unchanged |
| `SetAmplitude` ×2 | finite check unchanged; rejected write returns `quantum.CheckNormalization(attemptedSum, s.probabilitySum())`; rollback unchanged |
| `isNormalized` ×2 | deleted; sole call site becomes `quantum.IsNormalizedSum(s.probabilitySum())`; `probabilitySum` stays (slice vs map iteration is a real difference) |
| `applyMultiQubitGate` middle loop ×2 | `backendmath.MixCombos(matrix, inputs, outputs)`; gather/scatter stay per-backend (sparse prunes on scatter); dense buffers and loop structure untouched |
| Single-qubit loops ×2 | unchanged — hottest path in the repo; shared piece is a two-line formula |

## Error handling

Behavior-preserving: no error type, field, or message changes. The length
mismatch message is reproduced byte-identically.

## Testing

- New unit tests: `IsNormalizedSum` boundary table,
  `CheckNormalization` fields, `ValidateAmplitudeVector` paths (length
  message exact-match, first-nonfinite index wins, sum),
  `MixCombos` against hand-computed products (identity, CNOT, H⊗I,
  Toffoli), `RandFloat64` stub order and nil fallback,
  `ValidateQubitCount` table.
- Existing suite, `go test -race`, `go vet`, `gofmt` pass unchanged; no
  test edits.
- Fuzz seed corpora re-run, including `FuzzDenseSparseGateEquivalence`.
- Benchmarks before/after: `BenchmarkApplyMultiQubitGate`,
  `BenchmarkApplyThreeQubitGate`, `BenchmarkApplyGenericTwoQubitGate`,
  sparse gate benchmarks; single-qubit benchmarks as control. Gate: no
  dense regression beyond run-to-run noise.
  **Escape hatch**: if `MixCombos` regresses the dense path, dense keeps
  its inline loop and only sparse uses the kernel; the residual is then
  resolved for sparse and the dense divergence documented in the package
  doc with the benchmark numbers.

## Out of scope

Accessor-interface transactions, single-qubit loop dedupe,
`groundStateTolerance`, any behavior change, backlog items 9–12.

## Documentation

Backlog item 8 gets its `Done (2026-08-28)` blockquote per repo
convention; `backendmath` package doc amended as above.
