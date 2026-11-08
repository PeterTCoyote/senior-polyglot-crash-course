package main_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	main "senior-polyglot-crash-course/day-07-capstone/go"
)

// 1. Table-Driven Test: Business Rule Validation
func TestWorkerPool_TableDrivenValidation(t *testing.T) {
	processor := main.NewTransactionProcessor(0.02)

	testCases := []struct {
		name           string
		event          main.TransactionEvent
		expectedErr    error
		expectedStatus string
	}{
		{
			name: "Standard approved USD transaction",
			event: main.TransactionEvent{
				ID:        "TX-01",
				AccountID: "ACC-100",
				Amount:    500.00,
				Currency:  "USD",
			},
			expectedErr:    nil,
			expectedStatus: "APPROVED",
		},
		{
			name: "EUR conversion with high-value AML flag",
			event: main.TransactionEvent{
				ID:        "TX-02",
				AccountID: "ACC-200",
				Amount:    10000.00, // 10000 * 1.08 = $10,800 USD -> triggers REVIEW_REQUIRED
				Currency:  "EUR",
			},
			expectedErr:    nil,
			expectedStatus: "REVIEW_REQUIRED",
		},
		{
			name: "Negative amount rejected",
			event: main.TransactionEvent{
				ID:        "TX-03",
				AccountID: "ACC-300",
				Amount:    -50.00,
				Currency:  "USD",
			},
			expectedErr:    main.ErrInvalidAmount,
			expectedStatus: "",
		},
		{
			name: "Unsupported crypto currency rejected",
			event: main.TransactionEvent{
				ID:        "TX-04",
				AccountID: "ACC-400",
				Amount:    1.5,
				Currency:  "DOGE",
			},
			expectedErr:    main.ErrUnsupportedCurr,
			expectedStatus: "",
		},
		{
			name: "Suspended compliance account blocked",
			event: main.TransactionEvent{
				ID:        "TX-05",
				AccountID: "SUSPENDED-001",
				Amount:    100.00,
				Currency:  "USD",
			},
			expectedErr:    main.ErrAccountSuspended,
			expectedStatus: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			res, err := processor.Process(context.Background(), tc.event)
			if tc.expectedErr != nil {
				if !errors.Is(err, tc.expectedErr) {
					t.Fatalf("expected error wrapping %v, got %v", tc.expectedErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if res.Status != tc.expectedStatus {
				t.Errorf("expected status %q, got %q", tc.expectedStatus, res.Status)
			}
		})
	}
}

// 2. Resilience: Worker Panic Recovery
func TestWorkerPool_PanicRecovery(t *testing.T) {
	ctx := context.Background()
	cfg := main.Config{
		NumWorkers:     2,
		QueueCapacity:  10,
		MaxRetries:     0,
		InitialBackoff: 5 * time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
	}

	pool := main.NewWorkerPool[int, int](ctx, cfg)
	pool.Start()

	// Task 1: panics
	panicTask := main.Task[int, int]{
		ID:      "panic-task",
		Payload: 42,
		Execute: func(ctx context.Context, p int) (int, error) {
			panic("intentional fatal panic inside worker task")
		},
	}

	// Task 2: normal task after panic to verify worker pool survived
	normalTask := main.Task[int, int]{
		ID:      "normal-task",
		Payload: 100,
		Execute: func(ctx context.Context, p int) (int, error) {
			return p * 2, nil
		},
	}

	if err := pool.Submit(ctx, panicTask); err != nil {
		t.Fatalf("failed to submit panic task: %v", err)
	}
	if err := pool.Submit(ctx, normalTask); err != nil {
		t.Fatalf("failed to submit normal task: %v", err)
	}

	// Drain results
	results := make(map[string]main.Result[int])
	drainDone := make(chan struct{})
	go func() {
		defer close(drainDone)
		for res := range pool.Results() {
			results[res.TaskID] = res
		}
	}()

	shutdownCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := pool.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("pool shutdown failed: %v", err)
	}
	<-drainDone

	// Verify panic was caught cleanly
	panicRes, exists := results["panic-task"]
	if !exists {
		t.Fatal("panic-task result was not received")
	}
	if !panicRes.Panicked {
		t.Error("expected Panicked to be true")
	}
	if !errors.Is(panicRes.Err, main.ErrTaskPanicked) {
		t.Errorf("expected ErrTaskPanicked, got: %v", panicRes.Err)
	}

	// Verify pool and worker survived to complete normal task
	normalRes, exists := results["normal-task"]
	if !exists {
		t.Fatal("normal-task result was not received; worker died from panic")
	}
	if normalRes.Err != nil {
		t.Errorf("normal task failed unexpectedly: %v", normalRes.Err)
	}
	if normalRes.Value != 200 {
		t.Errorf("expected 200, got %d", normalRes.Value)
	}

	metrics := pool.Metrics()
	if metrics.Panics != 1 {
		t.Errorf("expected 1 panic metric, got %d", metrics.Panics)
	}
}

// 3. Resilience: Retry Logic with Backoff
func TestWorkerPool_RetryBackoff(t *testing.T) {
	ctx := context.Background()
	cfg := main.Config{
		NumWorkers:     1,
		QueueCapacity:  5,
		MaxRetries:     3,
		InitialBackoff: 2 * time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
	}

	pool := main.NewWorkerPool[string, string](ctx, cfg)
	pool.Start()

	var attempts atomic.Int32
	flakyErr := errors.New("temporary downstream outage")

	flakyTask := main.Task[string, string]{
		ID:      "flaky-task",
		Payload: "payload",
		Execute: func(ctx context.Context, p string) (string, error) {
			curr := attempts.Add(1)
			if curr < 3 {
				return "", flakyErr
			}
			return "success-after-retries", nil
		},
	}

	if err := pool.Submit(ctx, flakyTask); err != nil {
		t.Fatalf("failed to submit flaky task: %v", err)
	}

	var finalResult main.Result[string]
	done := make(chan struct{})
	go func() {
		for res := range pool.Results() {
			finalResult = res
		}
		close(done)
	}()

	shutdownCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := pool.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("pool shutdown failed: %v", err)
	}
	<-done

	if finalResult.Err != nil {
		t.Errorf("expected task to eventually succeed, got: %v", finalResult.Err)
	}
	if finalResult.Attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", finalResult.Attempts)
	}
	if attempts.Load() != 3 {
		t.Errorf("expected function executed 3 times, got %d", attempts.Load())
	}
	if pool.Metrics().Retried != 2 {
		t.Errorf("expected 2 retries recorded, got %d", pool.Metrics().Retried)
	}
}

// 4. Lifecycle: Graceful Shutdown
func TestWorkerPool_GracefulShutdown(t *testing.T) {
	ctx := context.Background()
	cfg := main.Config{
		NumWorkers:     3,
		QueueCapacity:  20,
		MaxRetries:     0,
		InitialBackoff: 5 * time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
	}

	pool := main.NewWorkerPool[int, int](ctx, cfg)
	pool.Start()

	numTasks := 15
	for i := 1; i <= numTasks; i++ {
		task := main.Task[int, int]{
			ID:      fmt.Sprintf("task-%d", i),
			Payload: i,
			Execute: func(ctx context.Context, p int) (int, error) {
				time.Sleep(10 * time.Millisecond)
				return p * 10, nil
			},
		}
		if err := pool.Submit(ctx, task); err != nil {
			t.Fatalf("failed to submit task %d: %v", i, err)
		}
	}

	var processedCount atomic.Int32
	resultsDone := make(chan struct{})
	go func() {
		for range pool.Results() {
			processedCount.Add(1)
		}
		close(resultsDone)
	}()

	// Initiate shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := pool.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("graceful shutdown returned error: %v", err)
	}

	<-resultsDone

	// Verify all tasks submitted before shutdown were drained
	if processedCount.Load() != int32(numTasks) {
		t.Errorf("expected %d drained tasks, got %d", numTasks, processedCount.Load())
	}

	// Submitting after shutdown must fail
	rejectedTask := main.Task[int, int]{
		ID:      "after-shutdown",
		Payload: 999,
		Execute: func(ctx context.Context, p int) (int, error) { return p, nil },
	}
	if err := pool.Submit(ctx, rejectedTask); !errors.Is(err, main.ErrPoolClosed) {
		t.Errorf("expected ErrPoolClosed, got: %v", err)
	}
}

// 5. Concurrency & Race Detector: High-Load Concurrent Submissions
func TestWorkerPool_DataRace(t *testing.T) {
	ctx := context.Background()
	cfg := main.Config{
		NumWorkers:     8,
		QueueCapacity:  100,
		MaxRetries:     1,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     5 * time.Millisecond,
	}

	pool := main.NewWorkerPool[int, int](ctx, cfg)
	pool.Start()

	var (
		producerWg       sync.WaitGroup
		numProducers     = 10
		itemsPerProducer = 50
	)

	// Launch concurrent producers
	for p := 0; p < numProducers; p++ {
		producerWg.Add(1)
		go func(prodID int) {
			defer producerWg.Done()
			for i := 0; i < itemsPerProducer; i++ {
				task := main.Task[int, int]{
					ID:      fmt.Sprintf("p%d-item%d", prodID, i),
					Payload: i,
					Execute: func(ctx context.Context, payload int) (int, error) {
						return payload + 1, nil
					},
				}
				_ = pool.Submit(ctx, task)
			}
		}(p)
	}

	var resultsReceived atomic.Int64
	resultsDone := make(chan struct{})
	go func() {
		for range pool.Results() {
			resultsReceived.Add(1)
		}
		close(resultsDone)
	}()

	producerWg.Wait()

	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("shutdown error under load: %v", err)
	}
	<-resultsDone

	expectedTotal := int64(numProducers * itemsPerProducer)
	if resultsReceived.Load() != expectedTotal {
		t.Errorf("expected %d results, got %d", expectedTotal, resultsReceived.Load())
	}
}

// Package-level sink variable to prevent compiler dead-code elimination
var sinkErr error

// 6. Benchmark: Measure Pool Throughput & Allocations
func BenchmarkWorkerPool_Throughput(b *testing.B) {
	ctx := context.Background()
	cfg := main.Config{
		NumWorkers:     8,
		QueueCapacity:  1000,
		MaxRetries:     0,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     time.Millisecond,
	}

	pool := main.NewWorkerPool[int, int](ctx, cfg)
	pool.Start()

	// Ensure pool is always cleaned up after benchmark runs, even on failure
	b.Cleanup(func() {
		shutdownCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		_ = pool.Shutdown(shutdownCtx)
	})

	// Drain results in background
	go func() {
		for range pool.Results() {
		}
	}()

	task := main.Task[int, int]{
		ID:      "bench-task",
		Payload: 42,
		Execute: func(ctx context.Context, payload int) (int, error) {
			return payload * 2, nil
		},
	}

	b.ReportAllocs()

	// Modern Go 1.24+ benchmark loop:
	// b.Loop() automatically handles iteration scaling, resets the timer on entry,
	// and pauses the timer on exit.
	for b.Loop() {
		sinkErr = pool.Submit(ctx, task)
	}
}
