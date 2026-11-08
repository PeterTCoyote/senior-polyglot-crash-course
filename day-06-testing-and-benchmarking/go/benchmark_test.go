package service_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	service "senior-polyglot-crash-course/day-06-testing-and-benchmarking/go"
)

// ============================================================================
// SENIOR ARCHITECTURAL LESSON: Go testing.B vs Java JMH
// ============================================================================
// In Java, microbenchmarking requires the JMH (Java Microbenchmark Harness) framework.
// Writing naive loops in Java results in JVM HotSpot optimizations (dead code elimination,
// loop unrolling, on-stack replacement) that invalidate benchmark measurements.
//
// In Go, benchmarking is a first-class citizen of the `testing` package:
//   `func BenchmarkXxx(b *testing.B)`
//
// Critical Gotchas for Senior Engineers:
// 1. `b.N` Loop: The Go testing harness automatically scales `b.N` until the benchmark
//    runs for a stable measurement window (typically 1 second).
// 2. `b.ResetTimer()`: Reset the timer AFTER costly setups to isolate the code under test.
// 3. Compiler Dead Code Elimination: If you don't use the return value, the Go compiler's
//    escape analysis / optimizer might eliminate the function call entirely!
//    In JMH you use `Blackhole.consume()`. In Go, assign to a package-level variable.
// 4. Memory Allocations: Always use `-benchmem` (or `b.ReportAllocs()`) to track
//    `allocs/op` (GC pressure) and `B/op` (memory footprint).
// ============================================================================

// Package-level sink variables to prevent the compiler from optimizing away the benchmarked code.
// Equivalent to JMH Blackhole.
var (
	sinkReceipt *service.Receipt
	sinkErr     error
)

// BenchmarkValidateOrder measures the throughput of pure CPU invariant validation.
func BenchmarkValidateOrder(b *testing.B) {
	svc := service.NewOrderService(nil, nil)
	validOrder := service.Order{
		ID:          "ord-bench-1",
		CustomerID:  "cust-bench-1",
		AmountCents: 9999,
		Currency:    "USD",
	}

	b.ReportAllocs() // Report B/op and allocs/op
	b.ResetTimer()   // Exclude initialization overhead

	for i := 0; i < b.N; i++ {
		// Assign to global sink to defeat compiler dead code elimination
		sinkErr = svc.ValidateOrder(validOrder)
	}
}

// BenchmarkProcessPayment_IdempotencyCacheHit measures the throughput of in-memory cache hits.
func BenchmarkProcessPayment_IdempotencyCacheHit(b *testing.B) {
	cachedReceipt := &service.Receipt{
		TransactionID: "txn-cached",
		OrderID:       "ord-cached",
		AmountCharged: 5000,
		Currency:      "EUR",
		ProcessedAt:   time.Now(),
	}

	store := service.NewSafeMemoryStore()
	_ = store.Put("ord-cached", cachedReceipt)

	gateway := &stubGateway{}
	svc := service.NewOrderService(gateway, store)

	order := service.Order{
		ID:          "ord-cached",
		CustomerID:  "cust-1",
		AmountCents: 5000,
		Currency:    "EUR",
	}

	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		sinkReceipt, sinkErr = svc.ProcessPayment(ctx, order)
	}
}

// BenchmarkStore_Comparative demonstrates sub-benchmarks comparing different workloads.
// Equivalent to JMH @Param or parameterized benchmark suites.
func BenchmarkStore_Comparative(b *testing.B) {
	store := service.NewSafeMemoryStore()

	// Sub-benchmark 1: Sequential Reads
	b.Run("ReadContention_SingleKey", func(b *testing.B) {
		_ = store.Put("read-key", &service.Receipt{TransactionID: "tx1"})
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			sinkReceipt, _ = store.Get("read-key")
		}
	})

	// Sub-benchmark 2: Writes with Unique Keys
	b.Run("Sequential_Writes", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("bench-key-%d", i)
			sinkErr = store.Put(key, &service.Receipt{TransactionID: key})
		}
	})
}
