# Go Testing, Benchmarking & Tooling Deep Dive

This guide is crafted for senior engineers moving from the JVM (JUnit 5, Mockito, JMH, SpotBugs) to Go 1.27.

---

## 0. Project Structure: Why Go Rejects `src/test/java`

In Java, Maven and Gradle require separate `src/main/java` and `src/test/java` source trees because `javac` has no built-in awareness of test files; without separate directories, test classes would end up packaged inside the production JAR.

In Go, **unit tests and benchmarks live in the exact same directory alongside the source files**:

```text
go/
├── service.go          # Production implementation
├── table_test.go       # Unit test (package service_test)
├── mock_test.go        # Unit test (package service_test)
├── race_test.go        # Concurrency test (package service_test)
└── benchmark_test.go   # Microbenchmark (package service_test)
```

### Why This is Idiomatic in Go:
1. **Compiler-Enforced Stripping:** The `go build` toolchain natively ignores any file ending in `*_test.go`. Test code, test dependencies, and test stubs **never** leak into the compiled executable binary.
2. **Black-Box Testing via `package foo_test`:** Notice our test files declare:
   ```go
   package service_test
   ```
   Even though `table_test.go` is in the same directory as `service.go`, the Go compiler treats it as a separate external package. The test is forced to import `service` as a consumer and can only access **exported** (public) symbols.
3. **White-Box Testing via `package foo`:** If you need to test private, unexported helpers or internals, simply declare `package service` in a test file (e.g. `export_test.go`). You get full package-private access without reflection hacks.
4. **When Go Uses a Separate Folder:** A separate root `/test` directory is used in Go **only** for multi-package integration, end-to-end (E2E) suites, or shared fixtures (`testdata/`). Moving unit tests into a separate directory breaks Go package cohesion and is considered an anti-pattern.

---

## 1. Table-Driven Tests: The Idiomatic Go Standard

In Java, parameterized testing requires framework annotations:
```java
@ParameterizedTest(name = "{0}")
@MethodSource("provideTestCases")
void testValidation(String name, Order order, Class<? extends Exception> expected) { ... }
```

In Go, table-driven testing is built into the language using **slices of anonymous structs** and `t.Run()`:

```go
tests := []struct {
    name        string
    order       service.Order
    expectedErr error
}{
    {name: "valid USD", order: validOrder, expectedErr: nil},
    {name: "negative amount", order: negativeOrder, expectedErr: service.ErrInvalidAmount},
}

for _, tc := range tests {
    t.Run(tc.name, func(t *testing.T) {
        t.Parallel() // Subtest executed concurrently across CPU cores
        err := svc.ValidateOrder(tc.order)
        // ...
    })
}
```

### The Go 1.22+ Loop Variable Evolution
> [!IMPORTANT]
> In Go versions prior to 1.22, running `t.Parallel()` inside a `for _, tc := range tests` loop required an explicit shadow copy (`tc := tc`). Without it, all goroutines shared the exact same memory address of `tc`, causing every parallel subtest to evaluate only the *final* element in the slice.
>
> In **Go 1.22+ and Go 1.27**, loop variables have **per-iteration lexical scope**. The shadow copy `tc := tc` is no longer required, though you will frequently encounter it in existing enterprise codebases.

---

## 2. Failure Handling: `t.Errorf` vs `t.Fatalf`

In Java, calling an assertion failure (such as `assertEquals` or `fail()`) throws an unchecked `AssertionError`. This instantly halts the test method, unwinding the stack.

In Go, failure behavior is explicit:
- **`t.Errorf(...)` (Soft Assertion):** Logs the diagnostic failure with filename and line number, marks the test as failed, but **continues executing**. This is equivalent to AssertJ's `SoftAssertions` or JUnit 5's `assertAll()`. It allows you to see all failing fields in a struct simultaneously.
- **`t.Fatalf(...)` (Hard Assertion):** Logs the error and halts **only the current test goroutine** immediately via `runtime.Goexit()`. Use this when a precondition fails (e.g., `if err != nil { t.Fatalf(...) }`), because dereferencing a nil pointer in subsequent lines would trigger an unhandled runtime panic.

---

## 3. Mocking Without Reflection: Fakes & Stubs

Java developers often reach for Mockito immediately:
```java
@Mock PaymentGateway gateway;
when(gateway.charge(any(), any(), any())).thenReturn(receipt);
verify(gateway, times(1)).charge(...);
```

Mockito achieves this through runtime bytecode generation (ByteBuddy) and reflection. Go has neither dynamic class loading nor runtime bytecode interception. 

Instead, Go relies on **Structural Subtyping** and **First-Class Functions**:

```go
// A stub struct whose methods delegate to function fields
type stubGateway struct {
    chargeFn func(ctx context.Context, orderID string, amountCents int64, currency string) (*service.Receipt, error)
    calls    atomic.Int32
}

func (s *stubGateway) Charge(ctx context.Context, orderID string, amountCents int64, currency string) (*service.Receipt, error) {
    s.calls.Add(1)
    return s.chargeFn(ctx, orderID, amountCents, currency)
}
```

### Advantages of Go's Approach
1. **Compile-time safety:** If the interface signature changes, your test fails to compile immediately. No runtime reflection mismatches.
2. **Zero runtime overhead:** No proxy objects, no classloaders, sub-millisecond test execution.
3. **Local reasoning:** The behavior of the fake is visible directly inside the test function.

---

## 4. The Race Detector (`go test -race`)

Testing thread safety in Java is notoriously unreliable. Developers typically create a `CountDownLatch` or thread pool, run 1,000 iterations, and hope a race condition surfaces.

In Go, ThreadSanitizer (TSan v3) is embedded directly into the compiler toolchain:

```bash
go test -race ./...
```

### How It Works Under the Hood
1. When compiled with `-race`, the compiler inserts an instrumentation hook before every 1-byte, 2-byte, 4-byte, and 8-byte memory access.
2. The runtime allocates shadow memory (1 shadow word per memory location) that tracks the goroutine ID, logical timestamp, and operation type (read vs write).
3. If two goroutines access the same memory location simultaneously without a synchronized *happens-before* edge (via channels, mutexes, or atomics), and at least one is a write, the runtime **immediately halts and dumps stack traces for both goroutines**.

---

## 5. Microbenchmarking: `testing.B` vs JMH

In Java, writing valid microbenchmarks requires JMH (Java Microbenchmark Harness) to prevent HotSpot JIT dead code elimination and loop unrolling.

In Go, benchmarking is built into the standard `testing` library:

```go
var sinkReceipt *service.Receipt // Global sink to defeat dead code elimination

func BenchmarkProcessPayment(b *testing.B) {
    // Setup phase (excluded from timer)
    svc := setupService()

    b.ReportAllocs() // Reports B/op and allocs/op
    b.ResetTimer()   // Exclude initialization time

    for i := 0; i < b.N; i++ {
        sinkReceipt, _ = svc.ProcessPayment(ctx, order)
    }
}
```

### The Dead Code Elimination Trap (Go's `Blackhole`)
If you call a function inside `b.N` but discard its return value, the Go compiler's escape analysis and dead code eliminator can optimize away the call entirely.
- In JMH: `bh.consume(result);`
- In Go: Assign the result to a **package-level global sink variable** (`sinkReceipt = ...`). Because the variable is package-exported/package-level, the compiler cannot prove it won't be read elsewhere, forcing real execution.

---

## 6. How to Run the Go Demonstrations

```bash
# 1. Run all tests with verbose output
go test -v ./...

# 2. Run concurrency tests with ThreadSanitizer race detector
go test -race -v -run TestConcurrentIdempotency_Safe ./...

# 3. Observe the Race Detector catch a live race condition (optional)
TRIGGER_RACE=1 go test -race -v -run TestDemonstrateUnsafeRaceCondition ./...

# 4. Run benchmarks with memory allocation metrics
go test -bench=. -benchmem -run=^$ ./...
```
