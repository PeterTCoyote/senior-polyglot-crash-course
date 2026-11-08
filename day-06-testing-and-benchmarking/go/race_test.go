package service_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	service "senior-polyglot-crash-course/day-06-testing-and-benchmarking/go"
)

// ============================================================================
// SENIOR ARCHITECTURAL LESSON: The Go Race Detector vs Java Concurrency Testing
// ============================================================================
// In Java, testing concurrent data structures (like a custom cache or map)
// typically involves spinning up an ExecutorService with a CountDownLatch and
// hoping that a race occurs. These "heisenbugs" often pass CI 99.9% of the time
// and only detonate under production load.
//
// In Go, the compiler embeds ThreadSanitizer (TSan v3) via the `-race` flag:
//   go test -race ./...
//
// How it works:
// The compiler instruments every memory read and write with shadow memory tracking.
// If two goroutines access the same memory location concurrently without a synchronized
// happens-before edge (and at least one is a write), the runtime catches it
// DETERMINISTICALLY and prints the exact stack traces of both goroutines.
// ============================================================================

// TestConcurrentIdempotency_Safe verifies thread safety under heavy concurrent access.
// Run with: `go test -race -v -run TestConcurrentIdempotency_Safe ./...`
func TestConcurrentIdempotency_Safe(t *testing.T) {
	const workers = 50
	const ordersPerWorker = 20

	store := service.NewSafeMemoryStore()
	gateway := &stubGateway{
		chargeFn: func(ctx context.Context, orderID string, amountCents int64, currency string) (*service.Receipt, error) {
			return &service.Receipt{
				TransactionID: fmt.Sprintf("txn-%s", orderID),
				OrderID:       orderID,
				AmountCharged: amountCents,
				Currency:      currency,
				ProcessedAt:   time.Now(),
			}, nil
		},
	}

	svc := service.NewOrderService(gateway, store)

	var wg sync.WaitGroup
	wg.Add(workers)

	// Launch 50 concurrent worker goroutines competing to process orders.
	// Several workers will submit identical order IDs to trigger concurrent idempotency checks.
	for w := 0; w < workers; w++ {
		workerID := w
		go func() {
			defer wg.Done()
			for i := 0; i < ordersPerWorker; i++ {
				// Create intentional collisions (only 10 unique order IDs shared across all workers)
				orderID := fmt.Sprintf("order-%d", i%10)
				order := service.Order{
					ID:          orderID,
					CustomerID:  fmt.Sprintf("cust-%d", workerID),
					AmountCents: 1000,
					Currency:    "USD",
				}

				receipt, err := svc.ProcessPayment(context.Background(), order)
				if err != nil {
					t.Errorf("worker %d failed processing %s: %v", workerID, orderID, err)
					return
				}
				if receipt == nil {
					t.Errorf("worker %d got nil receipt for %s", workerID, orderID)
					return
				}
			}
		}()
	}

	wg.Wait()
}

// TestDemonstrateUnsafeRaceCondition demonstrates how `go test -race` catches unsynchronized maps.
//
// By default, this test is skipped so your CI suite stays green.
// To watch the Go Race Detector in action, execute:
//   TRIGGER_RACE=1 go test -race -run TestDemonstrateUnsafeRaceCondition ./...
func TestDemonstrateUnsafeRaceCondition(t *testing.T) {
	if os.Getenv("TRIGGER_RACE") != "1" {
		t.Skip("Skipping intentional race condition demo. Run with: TRIGGER_RACE=1 go test -race -v -run TestDemonstrateUnsafeRaceCondition")
	}

	// This store is NOT protected by sync.RWMutex.
	// Concurrently reading and writing a native Go map causes a fatal runtime crash or data race.
	unsafeStore := service.NewUnsafeMemoryStore()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			_ = unsafeStore.Put("colliding-key", &service.Receipt{TransactionID: "1"})
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			_, _ = unsafeStore.Get("colliding-key")
		}
	}()

	wg.Wait()
}
