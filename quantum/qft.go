package quantum

import (
	"errors"
	"fmt"
	"math"
	"math/cmplx"
)

// QFT applies the quantum Fourier transform to s in place:
// F|x⟩ = (1/√N) Σ_y e^{2πixy/N} |y⟩ over the N = 2^NumQubits basis
// states — the exact 2ⁿ-dimensional DFT unitary, not a gate decomposition,
// so there is no bit-reversal caveat and the basis index maps directly to
// the phase register the way phase estimation expects.
//
// The transform is computed on the amplitude vector with a radix-2 FFT
// (O(N log N), no matrix construction) and written back in one
// BulkAmplitudeSetter call. Sparse backends are supported but lose their
// sparsity by nature: the Fourier transform of a sparse vector is dense,
// so after QFT every amplitude is generally nonzero.
//
// Returns an error if s is nil or a typed nil, or if s does not implement
// BulkAmplitudeSetter (UnsupportedOperationError). Normalization and
// non-finite amplitude errors come from SetAmplitudes, which rejects the
// write and leaves the state untouched.
func QFT(s QuantumState) error {
	return fourierTransform(s, +1)
}

// InverseQFT applies the inverse quantum Fourier transform, the conjugate
// transpose of F, in place. Same contract as QFT; the two compose to the
// identity up to float error.
func InverseQFT(s QuantumState) error {
	return fourierTransform(s, -1)
}

func fourierTransform(s QuantumState, sign float64) error {
	if isNilQuantumState(s) {
		return errors.New("state must not be nil")
	}
	setter, ok := s.(BulkAmplitudeSetter)
	if !ok {
		return &UnsupportedOperationError{
			Operation:   "QFT",
			Backend:     "*" + typeNameOf(s),
			Alternative: "a backend implementing BulkAmplitudeSetter (dense or sparse state)",
		}
	}

	size := 1 << s.NumQubits()
	amplitudes := make([]complex128, size)
	for i := range amplitudes {
		amplitudes[i] = s.Amplitude(i)
	}

	fftInPlace(amplitudes, sign)

	return setter.SetAmplitudes(amplitudes)
}

// fftInPlace replaces a with its DFT scaled to unit norm: entry k becomes
// (1/√n) Σ_j e^{sign·2πi·jk/n} a[j]. Iterative Cooley-Tukey: bit-reversal
// permutation, then doubling butterflies whose twiddle stride is
// e^{sign·2πi/length} because the exponent sign flips between the forward
// and inverse transforms.
func fftInPlace(a []complex128, sign float64) {
	n := len(a)

	// Bit-reversal permutation.
	for i, j := 1, 0; i < n; i++ {
		bit := n >> 1
		for ; j&bit != 0; bit >>= 1 {
			j ^= bit
		}
		j |= bit
		if i < j {
			a[i], a[j] = a[j], a[i]
		}
	}

	for length := 2; length <= n; length <<= 1 {
		half := length >> 1
		twiddle := cmplx.Exp(complex(0, sign*2*math.Pi/float64(length)))
		for start := 0; start < n; start += length {
			w := complex(1, 0)
			for k := 0; k < half; k++ {
				u := a[start+k]
				v := a[start+k+half] * w
				a[start+k] = u + v
				a[start+k+half] = u - v
				w *= twiddle
			}
		}
	}

	scale := complex(1/math.Sqrt(float64(n)), 0)
	for i := range a {
		a[i] *= scale
	}
}

// typeNameOf reports s's dynamic type name for error messages.
func typeNameOf(s QuantumState) string {
	return fmt.Sprintf("%T", s)
}
