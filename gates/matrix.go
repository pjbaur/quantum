package gates

import "fmt"

// ComposeMatrices multiplies two square matrices (left * right).
func ComposeMatrices(left, right [][]complex128) ([][]complex128, error) {
	leftSize, err := squareSize(left)
	if err != nil {
		return nil, fmt.Errorf("left matrix: %w", err)
	}
	rightSize, err := squareSize(right)
	if err != nil {
		return nil, fmt.Errorf("right matrix: %w", err)
	}
	if leftSize != rightSize {
		return nil, fmt.Errorf("matrix size mismatch: %d vs %d", leftSize, rightSize)
	}

	result := make([][]complex128, leftSize)
	for i := range result {
		result[i] = make([]complex128, leftSize)
		for j := range result[i] {
			sum := complex(0, 0)
			for k := 0; k < leftSize; k++ {
				sum += left[i][k] * right[k][j]
			}
			result[i][j] = sum
		}
	}

	return result, nil
}

// TensorProduct returns the Kronecker product of two square matrices.
func TensorProduct(left, right [][]complex128) ([][]complex128, error) {
	leftSize, err := squareSize(left)
	if err != nil {
		return nil, fmt.Errorf("left matrix: %w", err)
	}
	rightSize, err := squareSize(right)
	if err != nil {
		return nil, fmt.Errorf("right matrix: %w", err)
	}

	size := leftSize * rightSize
	result := make([][]complex128, size)
	for i := range result {
		result[i] = make([]complex128, size)
	}

	for i := 0; i < leftSize; i++ {
		for j := 0; j < leftSize; j++ {
			leftVal := left[i][j]
			for k := 0; k < rightSize; k++ {
				for l := 0; l < rightSize; l++ {
					row := i*rightSize + k
					col := j*rightSize + l
					result[row][col] = leftVal * right[k][l]
				}
			}
		}
	}

	return result, nil
}

func squareSize(matrix [][]complex128) (int, error) {
	if len(matrix) == 0 {
		return 0, fmt.Errorf("matrix must not be empty")
	}
	size := len(matrix)
	for i, row := range matrix {
		if len(row) != size {
			return 0, fmt.Errorf("row %d has length %d, expected %d", i, len(row), size)
		}
	}
	return size, nil
}
