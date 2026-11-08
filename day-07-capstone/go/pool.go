package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

// Sentinel errors representing well-defined operational failure modes.
var (
	ErrPoolClosed   = errors.New("worker pool is closed")
	ErrPoolFull     = errors.New("worker pool queue is full (backpressure active)")
	ErrTaskTimeout  = errors.New("task execution exceeded deadline")
	ErrTaskPanicked = errors.New("worker recovered from task panic")
)

// Config specifies the operational bounds and retry parameters of the WorkerPool.
type Config struct {
	NumWorkers     int           // Fixed number of worker goroutines
	QueueCapacity  int           // Bounded task buffer capacity (prevents OOM)
	MaxRetries     int           // Maximum retry attempts for transient task errors
	InitialBackoff time.Duration // Base backoff duration before exponential escalation
	MaxBackoff     time.Duration // Upper ceiling for exponential backoff
}

// DefaultConfig returns a production-ready baseline configuration.
func DefaultConfig() Config {
	return Config{
		NumWorkers:     8,
		QueueCapacity:  256,
		MaxRetries:     3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     200 * time.Millisecond,
	}
}

// Task encapsulates a unit of work parameterized by input T and output R.
type Task[T any, R any] struct {
	ID      string
	Payload T
	Execute func(ctx context.Context, payload T) (R, error)
}

// Result captures the outcome of a Task execution, including diagnostics and timing.
type Result[R any] struct {
	TaskID    string
	Value     R
	Err       error
	Panicked  bool
	Duration  time.Duration
	Attempts  int
}

// MetricsSnapshot provides a point-in-time immutable view of the pool's atomic counters.
type MetricsSnapshot struct {
	Submitted     uint64
	Processed     uint64
	Failed        uint64
	Retried       uint64
	Panics        uint64
	ActiveWorkers int32
}

// WorkerPool manages a bounded set of worker goroutines processing generic tasks
// over bounded channels with backpressure, panic recovery, and graceful shutdown.
type WorkerPool[T any, R any] struct {
	cfg        Config
	tasksChan  chan Task[T, R]
	resultChan chan Result[R]

	// Lifecycle management
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	drainWg    sync.WaitGroup
	closeOnce  sync.Once
	isClosed   atomic.Bool

	// Lock-free atomic telemetry
	submitted  atomic.Uint64
	processed  atomic.Uint64
	failed     atomic.Uint64
	retried    atomic.Uint64
	panics     atomic.Uint64
	active     atomic.Int32
}

// NewWorkerPool instantiates and initializes a new WorkerPool.
func NewWorkerPool[T any, R any](parentCtx context.Context, cfg Config) *WorkerPool[T, R] {
	if cfg.NumWorkers <= 0 {
		cfg.NumWorkers = 1
	}
	if cfg.QueueCapacity < 0 {
		cfg.QueueCapacity = 0
	}

	ctx, cancel := context.WithCancel(parentCtx)

	return &WorkerPool[T, R]{
		cfg:        cfg,
		tasksChan:  make(chan Task[T, R], cfg.QueueCapacity),
		resultChan: make(chan Result[R], cfg.QueueCapacity),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start spawns the configured worker goroutines.
func (p *WorkerPool[T, R]) Start() {
	p.wg.Add(p.cfg.NumWorkers)
	for i := 1; i <= p.cfg.NumWorkers; i++ {
		go p.workerLoop(i)
	}
}

// Submit queues a task for execution. If the queue is saturated, Submit blocks until
// space is available, the provided ctx is cancelled, or the pool is shut down.
func (p *WorkerPool[T, R]) Submit(ctx context.Context, task Task[T, R]) error {
	if p.isClosed.Load() {
		return ErrPoolClosed
	}

	select {
	case <-p.ctx.Done():
		return ErrPoolClosed
	case <-ctx.Done():
		return ctx.Err()
	case p.tasksChan <- task:
		p.submitted.Add(1)
		return nil
	}
}

// TrySubmit attempts to submit a task without blocking. If the queue is full or the pool
// is closed, it immediately returns an appropriate error (non-blocking backpressure).
func (p *WorkerPool[T, R]) TrySubmit(task Task[T, R]) error {
	if p.isClosed.Load() {
		return ErrPoolClosed
	}

	select {
	case <-p.ctx.Done():
		return ErrPoolClosed
	case p.tasksChan <- task:
		p.submitted.Add(1)
		return nil
	default:
		return ErrPoolFull
	}
}

// Results exposes the read-only result channel. Consumers should read from this
// channel until it is closed during graceful shutdown.
func (p *WorkerPool[T, R]) Results() <-chan Result[R] {
	return p.resultChan
}

// Metrics returns an immutable snapshot of current atomic counters.
func (p *WorkerPool[T, R]) Metrics() MetricsSnapshot {
	return MetricsSnapshot{
		Submitted:     p.submitted.Load(),
		Processed:     p.processed.Load(),
		Failed:        p.failed.Load(),
		Retried:       p.retried.Load(),
		Panics:        p.panics.Load(),
		ActiveWorkers: p.active.Load(),
	}
}

// workerLoop is the core execution loop for an individual worker goroutine.
func (p *WorkerPool[T, R]) workerLoop(workerID int) {
	defer p.wg.Done()
	p.active.Add(1)
	defer p.active.Add(-1)

	for {
		select {
		case task, ok := <-p.tasksChan:
			if !ok {
				// Queue channel closed during shutdown: drain complete
				return
			}
			p.executeWithRecovery(task)

		case <-p.ctx.Done():
			// Root context cancelled: drain any remaining items in queue before exiting
			for {
				select {
				case task, ok := <-p.tasksChan:
					if !ok {
						return
					}
					p.executeWithRecovery(task)
				default:
					return
				}
			}
		}
	}
}

// executeWithRecovery executes the task with panic protection, retry escalation, and timing.
func (p *WorkerPool[T, R]) executeWithRecovery(task Task[T, R]) {
	start := time.Now()
	var (
		resultVal R
		err       error
		panicked  bool
		attempts  int
	)

	// Execute inside isolated panic boundary
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
				p.panics.Add(1)
				p.failed.Add(1)
				err = fmt.Errorf("%w: %v\nstack:\n%s", ErrTaskPanicked, r, debug.Stack())
			}
		}()

		// Retry loop with exponential backoff & jitter
		maxAttempts := 1 + p.cfg.MaxRetries
		for attempt := 1; attempt <= maxAttempts; attempt++ {
			attempts = attempt

			// Check context before executing
			if p.ctx.Err() != nil {
				err = p.ctx.Err()
				break
			}

			resultVal, err = task.Execute(p.ctx, task.Payload)
			if err == nil {
				// Success
				break
			}

			// If error is permanent or retries exhausted, stop retrying
			if attempt == maxAttempts || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				break
			}

			// Transient failure -> increment retried counter and sleep with jitter
			p.retried.Add(1)
			backoff := p.calculateBackoff(attempt)

			select {
			case <-time.After(backoff):
			case <-p.ctx.Done():
				err = p.ctx.Err()
				return
			}
		}
	}()

	duration := time.Since(start)

	if err != nil && !panicked {
		p.failed.Add(1)
	} else if err == nil {
		p.processed.Add(1)
	}

	res := Result[R]{
		TaskID:   task.ID,
		Value:    resultVal,
		Err:      err,
		Panicked: panicked,
		Duration: duration,
		Attempts: attempts,
	}

	// Send result without deadlocking if shutdown occurs
	select {
	case p.resultChan <- res:
	case <-p.ctx.Done():
		// Context cancelled; best-effort deliver or drop to prevent deadlock
		select {
		case p.resultChan <- res:
		default:
		}
	}
}

// calculateBackoff calculates full jitter exponential backoff:
// Backoff = Random(0, Min(MaxBackoff, InitialBackoff * 2^(attempt-1)))
func (p *WorkerPool[T, R]) calculateBackoff(attempt int) time.Duration {
	multiplier := 1 << (attempt - 1)
	base := p.cfg.InitialBackoff * time.Duration(multiplier)
	if base > p.cfg.MaxBackoff || base <= 0 {
		base = p.cfg.MaxBackoff
	}

	// Full jitter prevents thundering herd
	jitterNano := rand.Int64N(int64(base))
	return time.Duration(jitterNano)
}

// Shutdown initiates a graceful shutdown sequence:
// 1. Marks pool closed (rejects new submits)
// 2. Closes tasksChan so workers drain existing queued items
// 3. Awaits worker completion up to the provided timeout context
// 4. Closes resultsChan once all workers terminate
func (p *WorkerPool[T, R]) Shutdown(shutdownCtx context.Context) error {
	p.closeOnce.Do(func() {
		p.isClosed.Store(true)
		close(p.tasksChan) // Causes workers to drain and exit workerLoop
	})

	// Wait for workers in a separate goroutine
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All workers cleanly exited
		close(p.resultChan)
		return nil

	case <-shutdownCtx.Done():
		// Timeout expired before workers finished draining
		p.cancel() // Force cancel remaining in-flight tasks
		<-done     // Wait for workers to observe cancel and finish
		close(p.resultChan)
		return shutdownCtx.Err()
	}
}
