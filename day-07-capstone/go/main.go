package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

func main() {
	fmt.Println("================================================================================")
	fmt.Println("🚀 DAY 7 CAPSTONE (Go): High-Throughput Concurrent Worker Pool")
	fmt.Println("   Architecture: CSP Bounded Queues, Atomics, Panic Recovery & Graceful Shutdown")
	fmt.Println("================================================================================")

	// 1. Root context bound to OS signals (Ctrl+C / SIGINT / SIGTERM)
	ctx, stopSignal := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignal()

	// 2. Configure and instantiate pool
	cfg := Config{
		NumWorkers:     4,
		QueueCapacity:  64,
		MaxRetries:     2,
		InitialBackoff: 20 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
	}

	pool := NewWorkerPool[TransactionEvent, EnrichedTransaction](ctx, cfg)
	processor := NewTransactionProcessor(0.015) // 1.5% processing fee

	pool.Start()
	fmt.Printf("[Init] Worker pool initialized with %d workers, queue capacity: %d\n", cfg.NumWorkers, cfg.QueueCapacity)

	// 3. Spawn asynchronous result consumer
	var resultWg sync.WaitGroup
	resultWg.Add(1)

	var (
		totalApproved atomic.Uint64
		totalFlagged  atomic.Uint64
		totalErrors   atomic.Uint64
	)

	go func() {
		defer resultWg.Done()
		for res := range pool.Results() {
			if res.Panicked {
				fmt.Printf("  🚨 [PANIC RECOVERED] Task %s panicked! Runtime preserved. Latency: %v\n", res.TaskID, res.Duration)
				totalErrors.Add(1)
				continue
			}

			if res.Err != nil {
				fmt.Printf("  ⚠️ [FAILURE] Task %s failed after %d attempts: %v\n", res.TaskID, res.Attempts, res.Err)
				totalErrors.Add(1)
				continue
			}

			switch res.Value.Status {
			case "REVIEW_REQUIRED":
				totalFlagged.Add(1)
				fmt.Printf("  🔍 [REVIEW] Task %s: Account %s flagged ($%.2f USD >= $10k)\n",
					res.TaskID, res.Value.AccountID, res.Value.AmountUSD)
			default:
				totalApproved.Add(1)
				fmt.Printf("  ✅ [APPROVED] Task %s: %s %.2f -> $%.2f USD (Fee: $%.2f)\n",
					res.TaskID, res.Value.OriginalCurrency, res.Value.OriginalAmount, res.Value.AmountUSD, res.Value.FeeUSD)
			}
		}
		fmt.Println("[Consumer] Results stream closed. Consumer goroutine finished cleanly.")
	}()

	// 4. Producer: Stream simulated transaction workload
	currencies := []string{"USD", "EUR", "GBP", "JPY"}
	totalItems := 50

	fmt.Printf("[Producer] Streaming %d transactions into pipeline... (Press Ctrl+C to test graceful abort)\n\n", totalItems)

producerLoop:
	for i := 1; i <= totalItems; i++ {
		select {
		case <-ctx.Done():
			fmt.Println("\n[Signal] Interrupt detected! Halting producer submission.")
			break producerLoop
		default:
		}

		txID := fmt.Sprintf("TX-%04d", i)
		accID := fmt.Sprintf("ACC-%03d", rand.IntN(20)+1)
		curr := currencies[rand.IntN(len(currencies))]
		amt := float64(rand.IntN(12000) + 10)

		// Inject intentional chaos to demonstrate resilience features
		var simTransient, simPanic bool
		if i == 7 {
			// Test account suspension failure
			accID = "SUSPENDED-999"
		} else if i == 13 {
			// Test invalid currency validation
			curr = "BITCOIN_INVALID"
		} else if i == 21 {
			// Test retry mechanism with transient network error
			simTransient = true
		} else if i == 33 {
			// Test worker panic isolation
			simPanic = true
		}

		task := Task[TransactionEvent, EnrichedTransaction]{
			ID: txID,
			Payload: TransactionEvent{
				ID:                txID,
				AccountID:         accID,
				Amount:            amt,
				Currency:          curr,
				Timestamp:         time.Now(),
				SimulateTransient: simTransient,
				SimulatePanic:     simPanic,
			},
			Execute: processor.Process,
		}

		// Submit with backpressure handling
		submitCtx, submitCancel := context.WithTimeout(ctx, 2*time.Second)
		err := pool.Submit(submitCtx, task)
		submitCancel()

		if err != nil {
			if ctx.Err() != nil {
				fmt.Println("\n[Producer] Context cancelled during submission.")
				break producerLoop
			}
			fmt.Printf("  [Backpressure] Failed to submit %s: %v\n", txID, err)
		}

		// Throttle producer slightly for realistic visual demonstration
		time.Sleep(30 * time.Millisecond)
	}

	fmt.Println("\n[Lifecycle] Ingress stream completed. Initiating graceful shutdown...")

	// 5. Coordinated Graceful Shutdown
	// Give in-flight tasks and retries up to 3 seconds to complete draining
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer shutdownCancel()

	if err := pool.Shutdown(shutdownCtx); err != nil {
		fmt.Printf("❌ [Shutdown Error] Graceful shutdown deadline exceeded: %v\n", err)
	} else {
		fmt.Println("✅ [Shutdown] All worker goroutines drained and terminated successfully.")
	}

	// 6. Wait for consumer to finish processing all drained results
	resultWg.Wait()

	// 7. Telemetry & Summary
	m := pool.Metrics()
	fmt.Println("\n================================================================================")
	fmt.Println("📊 PIPELINE METRICS & PRODUCTION OBSERVABILITY")
	fmt.Println("================================================================================")
	fmt.Printf("  • Total Tasks Submitted: %d\n", m.Submitted)
	fmt.Printf("  • Successfully Processed: %d\n", m.Processed)
	fmt.Printf("  • Failed / Rejected:     %d\n", m.Failed)
	fmt.Printf("  • Retry Attempts:        %d\n", m.Retried)
	fmt.Printf("  • Recovered Panics:      %d (Zero crashes)\n", m.Panics)
	fmt.Printf("  • Active Workers Remaining: %d\n", m.ActiveWorkers)
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Printf("  • Business Approvals:    %d\n", totalApproved.Load())
	fmt.Printf("  • Compliance Flags:      %d\n", totalFlagged.Load())
	fmt.Printf("  • Total Error Dispatches: %d\n", totalErrors.Load())
	fmt.Println("================================================================================")
}
