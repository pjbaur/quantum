package quantum

// Probability returns |c|², the measurement probability contributed by
// amplitude c.
//
// It computes re*re + im*im directly instead of squaring cmplx.Abs, whose
// square root the squaring only undoes again. Unlike cmplx.Abs this has no
// overflow-safe scaling, so it is meant for normalized amplitudes (|c| ≤ 1)
// rather than arbitrary complex values.
func Probability(c complex128) float64 {
	re, im := real(c), imag(c)
	return re*re + im*im
}
