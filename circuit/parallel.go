package circuit

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/pjbaur/quantum/quantum"
)

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
func ExecuteAllParallel(executions []Execution, opts ParallelOptions) error {
	if len(executions) == 0 {
		return nil
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
