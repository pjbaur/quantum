package circuit

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/pjbaur/quantum/quantum"
)

// validateIndependentStates checks that all state pointers in the executions
// are unique. It returns a SharedStateError if any state is shared between
// multiple executions.
func validateIndependentStates(executions []Execution) error {
	seen := make(map[quantum.QuantumState]int)
	for i, exec := range executions {
		if exec.State == nil {
			continue // nil states are caught by executeOne
		}
		if firstIdx, exists := seen[exec.State]; exists {
			return &quantum.SharedStateError{
				DuplicateIndices: []int{firstIdx, i},
			}
		}
		seen[exec.State] = i
	}
	return nil
}

// Execution pairs a circuit with the state it should operate on.
// Circuits and states must be independent to safely execute in parallel.
type Execution struct {
	Circuit *Circuit
	State   quantum.QuantumState
}

// ParallelOptions controls parallel circuit execution.
type ParallelOptions struct {
	// MaxParallelism limits the number of concurrent executions. When <= 0,
	// ExecuteAllParallel uses GOMAXPROCS.
	MaxParallelism int
}

// ExecuteAll applies each circuit to its state serially.
func ExecuteAll(executions []Execution) error {
	for i, exec := range executions {
		if err := executeOne(i, exec); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteAllParallel applies each circuit to its state using worker goroutines.
// It returns the first error encountered; other executions may still run.
//
// SAFETY: All Execution.State values must be unique pointers. Passing the same
// state to multiple executions will return a SharedStateError before any
// concurrent execution begins.
func ExecuteAllParallel(executions []Execution, opts ParallelOptions) error {
	if len(executions) == 0 {
		return nil
	}

	// Validate state independence before parallel execution
	if err := validateIndependentStates(executions); err != nil {
		return err
	}

	workers := opts.MaxParallelism
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
	}
	if workers < 1 {
		workers = 1
	}
	if workers >= len(executions) {
		workers = len(executions)
	}

	if workers == 1 {
		return ExecuteAll(executions)
	}

	jobs := make(chan int)
	var wg sync.WaitGroup
	var errOnce sync.Once
	var firstErr error
	var stop atomic.Bool

	setErr := func(err error) {
		if err == nil {
			return
		}
		errOnce.Do(func() {
			firstErr = err
			stop.Store(true)
		})
	}

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for idx := range jobs {
				if stop.Load() {
					continue
				}
				if err := executeOne(idx, executions[idx]); err != nil {
					setErr(err)
				}
			}
		}()
	}

	for i := range executions {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	return firstErr
}

func executeOne(index int, exec Execution) error {
	if exec.Circuit == nil {
		return fmt.Errorf("execution %d: circuit is nil", index)
	}
	if exec.State == nil {
		return fmt.Errorf("execution %d: state is nil", index)
	}
	if err := exec.Circuit.Execute(exec.State); err != nil {
		return fmt.Errorf("execution %d: %w", index, err)
	}
	return nil
}
