package main

import (
	"fmt"
	"math"
	"math/cmplx"
	"math/rand"
)

type Qubit struct {
	Alpha complex128
	Beta  complex128
}

func NewQubit() *Qubit {
	return &Qubit{
		Alpha: 1.0 + 0i,
		Beta:  0.0 + 0i,
	}
}

func (q *Qubit) ApplyHadamard() {
	h00 := complex(1.0/math.Sqrt(2), 0)
	h01 := complex(1.0/math.Sqrt(2), 0)
	h10 := complex(1.0/math.Sqrt(2), 0)
	h11 := complex(-1.0/math.Sqrt(2), 0)

	newAlpha := h00*q.Alpha + h01*q.Beta
	newBeta := h10*q.Alpha + h11*q.Beta

	q.Alpha = newAlpha
	q.Beta = newBeta
}

func (q *Qubit) Measure() int {
	// Fix: Remove unnecessary real() since Abs squared is already float64
	prob0 := cmplx.Abs(q.Alpha) * cmplx.Abs(q.Alpha)
	if rand.Float64() < prob0 {
		q.Alpha = 1.0 + 0i
		q.Beta = 0.0 + 0i
		return 0
	}
	q.Alpha = 0.0 + 0i
	q.Beta = 1.0 + 0i
	return 1
}

func main() {
	trials := 10
	zeros := 0
	ones := 0

	for i := 0; i < trials; i++ {
		qubit := NewQubit()
		fmt.Printf("Initial state: Alpha=%.3f, Beta=%.3f\n", real(qubit.Alpha), real(qubit.Beta))

		qubit.ApplyHadamard()
		fmt.Printf("After Hadamard: Alpha=%.3f, Beta=%.3f\n", real(qubit.Alpha), real(qubit.Beta))

		result := qubit.Measure()
		fmt.Printf("Measured: %d\n", result)
		if result == 0 {
			zeros++
		} else {
			ones++
		}
	}

	fmt.Printf("\nResults: %d zeros, %d ones\n", zeros, ones)
}
