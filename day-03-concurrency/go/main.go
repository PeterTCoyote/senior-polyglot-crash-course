package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Task represents an incoming unit of work processed by the worker pool.
type Task struct {
	ID      int
	Payload string
}

// Result represents the outcome of a processed task.
type Result struct {
	TaskID int
	Output string
	Err    error
}

func main() {
	// Initialize a root context with a hard timeout of 3 seconds to demonstrate
	// timeout propagation across concurrent workers.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	const numWorkers = 4
	const numTasks = 10

	// Buffered channels for decoupling producers and consumers.
	taskStream := make(chan Task, numTasks)
	resultStream := make(chan Result, numTasks)

	// Use an atomic counter to track active workers for telemetry.
	var activeWorkers int32

	// WaitGroup to coordinate graceful shutdown of workers.
	var workerWG sync.WaitGroup

	// 1. Spawn Worker Pool
	for w := 1; w <= numWorkers; w++ {
		workerWG.Add(1)
		go worker(ctx, w, taskStream, resultStream, &activeWorkers, &workerWG)
	}

	// 2. Producer: Dispatch tasks into the task stream
	go func() {
		defer close(taskStream)
		for i := 1; i <= numTasks; i++ {
			select {
			case <-ctx.Done():
				fmt.Println("[Producer] Context cancelled, stopping task dispatch.")
				return
			case taskStream <- Task{ID: i, Payload: fmt.Sprintf("data-payload-%d", i)}:
				// Successfully dispatched
			}
		}
	}()

	// 3. Consumer / Collector: Aggregate results or handle cancellation/timeout
	var consumerWG sync.WaitGroup
	consumerWG.Add(1)
	go func() {
		defer consumerWG.Done()
		for {
			select {
			case <-ctx.Done():
				fmt.Println("[Consumer] Timeout or cancellation received. Exiting collector.")
				return
			case res, ok := <-resultStream:
				if !ok {
					// resultStream was closed, meaning all workers have completed and shut down.
					fmt.Println("[Consumer] Result stream closed. All work finalized.")
					return
				}
				if res.Err != nil {
					fmt.Printf("[Error] Task %d failed: %v\n", res.TaskID, res.Err)
				} else {
					fmt.Printf("[Success] Task %d processed -> %s\n", res.TaskID, res.Output)
				}
			}
		}
	}()

	// Wait for all workers to finish execution
	workerWG.Wait()
	// Safe to close resultStream once workers terminate and no more results will be sent.
	close(resultStream)

	// Wait for collector to drain remaining items
	consumerWG.Wait()

	fmt.Printf("Execution complete. Final active workers: %d\n", atomic.LoadInt32(&activeWorkers))
}

// worker pulls tasks from taskStream, respects context cancellation, and pushes results.
func worker(
	ctx context.Context,
	workerID int,
	tasks <-chan Task,
	results chan<- Result,
	activeWorkers *int32,
	wg *sync.WaitGroup,
) {
	defer wg.Done()
	atomic.AddInt32(activeWorkers, 1)
	defer atomic.AddInt32(activeWorkers, -1)

	for {
		select {
		case <-ctx.Done():
			// Context expired or was cancelled upstream. Exit immediately to prevent leaks.
			return
		case task, ok := <-tasks:
			if !ok {
				// Channel closed and drained.
				return
			}

			// Simulate processing latency (e.g., I/O bound or heavy computation)
			err := processTask(ctx, task)
			if err != nil {
				results <- Result{TaskID: task.ID, Err: err}
			} else {
				results <- Result{
					TaskID: task.ID,
					Output: fmt.Sprintf("Processed by Worker-%d [Payload: %s]", workerID, task.Payload),
				}
			}
		}
	}
}

// processTask simulates work while respecting non-blocking select patterns.
func processTask(ctx context.Context, task Task) error {
	// Simulate variable workload duration
	workDuration := 400 * time.Millisecond

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(workDuration):
		if task.ID == 7 {
			// Simulate injected business logic error for demonstration
			return errors.New("simulated database constraint violation")
		}
		return nil
	}
}