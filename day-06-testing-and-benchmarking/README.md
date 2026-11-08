# Day 6: Testing, Benchmarking, and Quality Engineering

Welcome to Day 6 of the Senior Polyglot Crash Course. For senior engineers coming from Java (JUnit 5, Mockito, Spring Test, JMH, Checkstyle/SpotBugs, SonarQube), testing in Go and Python represents a fundamental paradigm shift away from heavyweight reflection-based frameworks toward standard-library primitives and lightweight, deterministic tooling.

---

## 🎯 The Senior Java Mental Model Shift

In enterprise Java, testing often relies on:
1. **Reflection & Bytecode Manipulation:** Annotations (`@Test`, `@ParameterizedTest`, `@BeforeEach`, `@ExtendWith`), runtime proxy generation (ByteBuddy, CGLIB in Mockito), and dependency injection containers (`@MockBean`).
2. **Aborting on Failure:** Test assertions throw unchecked `AssertionError` exceptions that instantly terminate the test method (unless using soft assertion libraries like AssertJ).
3. **Separate Benchmarking Harnesses:** JMH is needed to defeat JVM JIT dead-code elimination, warm up the HotSpot compiler, and mitigate CPU branch prediction side effects.
4. **Flaky Concurrency Testing:** Testing thread safety relies on stress loops, `CountDownLatch`, or external harnesses like `jcstress` because JVM runtime doesn't ship with an integrated race detector.

In **Go** and **Python**, these problems are tackled with vastly different trade-offs:

| Dimension | Enterprise Java (JUnit 5 / Mockito / JMH) | Idiomatic Go (1.27) | Modern Python (3.15 / pytest) |
| :--- | :--- | :--- | :--- |
| **Test Runner** | External framework (JUnit Jupiter Engine) driven by reflection. | Built-in `go test` standard tool. Tests are plain Go functions `func TestX(t *testing.T)`. | `pytest` runner with bytecode AST rewriting for plain `assert` expressions. |
| **Parameterized Tests** | `@ParameterizedTest` + `@MethodSource` / `@CsvSource`. Reflection-bound. | **Table-Driven Tests**: Slices of structs executed sequentially or concurrently via `t.Run()`. | `@pytest.mark.parametrize` with explicit test ID generation and cartesian product support. |
| **Failure Mechanics** | Exceptions (`AssertionError`) abort the test method immediately. | **Fail-and-Continue by default**: `t.Errorf()` reports failure and continues (soft assertion); `t.Fatalf()` aborts only current test goroutine. | `assert` raises `AssertionError`, caught and pretty-printed by `pytest` with variable diffing. |
| **Mocking / Fakes** | Dynamic bytecode agents (`@Mock`, `@Spy`, `when(...).thenReturn(...)`). | **Structural subtyping & interface stubs**: Hand-written fakes, function-injection structs. No reflection proxies needed. | `unittest.mock.MagicMock` and monkeypatching. **Trap:** Needs `autospec=True` to avoid accepting nonexistent methods. |
| **Race Detection** | Difficult; requires complex stress tests, ThreadSanitizer flags (rarely used in app dev), or JMM analysis. | **Compiler-level ThreadSanitizer (`go test -race`)**: Deterministic detection of unsynchronized concurrent reads/writes with stack traces. | GIL mitigates low-level memory corruption, but race conditions exist in multi-threading and async task scheduling. |
| **Benchmarking** | JMH with `@Benchmark`, `@Warmup`, and `Blackhole` to defeat JIT optimization. | Built into `testing.B`: `b.N` iteration loops, `b.ResetTimer()`, memory allocation stats (`-benchmem`), compiler sink variables. | `cProfile` (deterministic profiling), `tracemalloc` (heap allocation tracking), and `pytest-benchmark`. |
| **Linter / Quality Gates** | SpotBugs, PMD, Checkstyle, SonarQube (XML/Gradle configuration). | `golangci-lint`: High-performance single-pass engine executing 50+ specialized AST linters. | `ruff` (blazing-fast Rust-based linter/formatter) + `mypy` (static type enforcement). |

---

## 📂 Where Do Tests Live? (The `src/test/java` Question)

A jarring transition for Java developers is the absence of mirrored `src/test/java` directory structures.

### The Maven/Gradle Muscle Memory:
```text
src/
├── main/java/com/example/OrderService.java
└── test/java/com/example/OrderServiceTest.java
```
In Java, Maven requires separate source directories because `javac` has no built-in distinction for test files; without separate directories, test code and test libraries would be packaged into the production `.jar`.

### Why Go and Python Do Not Require This:

| Language | Test Location | How Production Code Excludes Tests | How Black-Box Testing is Enforced |
| :--- | :--- | :--- | :--- |
| **Java** | `src/test/java/` (Strictly mirrored) | Build tool classpath configuration (`<scope>test</scope>`) | Tests share package namespace to access package-private members |
| **Go** | **Same directory** (`service.go` and `table_test.go` sit side-by-side) | **Compiler toolchain rule**: `go build` automatically ignores any file ending in `*_test.go`. Zero test code ever enters the binary. | `package service_test` in the same folder forces tests to import the package as an external consumer. |
| **Python** | Dedicated `tests/` directory (common in libraries) or **co-located** (common in microservices/CLIs) | Packaging config in `pyproject.toml` excludes `tests/` or `test_*.py` when building wheels | Python does not enforce access boundaries; encapsulation is signaled via `_private` conventions. |

> [!NOTE]
> In Go, unit tests and benchmarks **must** live in the same directory as the package they test. A separate root-level `/test` or `/tests` folder in Go is reserved exclusively for integration/E2E suites that coordinate across multiple distinct packages.
> In Python, `pytest` automatically discovers and runs tests whether they reside in a dedicated `tests/` folder or alongside source files.

---

## 🗺️ What We Cover Today

### Go Track (`go/`)
1. **Table-Driven Tests:** Struct-slice test tables, subtests with `t.Run()`, loop variable semantics in Go 1.22/1.27, and `t.Parallel()` scheduling.
2. **Idiomatic Fakes over Mockito:** Structural interface mocking, function-field injection stubs, and interaction tracking without bytecode proxies.
3. **The Race Detector (`go test -race`):** Reproducing an unsynchronized shared-state race condition and resolving it with `sync.RWMutex` and `sync/atomic`.
4. **Built-in Microbenchmarking (`testing.B`):** Benchmarking CPU operations, memory allocation profiling (`-benchmem`), and preventing compiler dead-code elimination using global sinks.
5. **Quality Gates:** Production-grade `.golangci.yml` configuration.

### Python Track (`python/`)
1. **Pytest vs. JUnit:** Parameterized testing via `@pytest.mark.parametrize` vs. Java `@ParameterizedTest`.
2. **Fixtures & Dependency Injection:** `@pytest.fixture` with `yield` generators for unified setup/teardown across `function`, `module`, and `session` scopes.
3. **Mocking Pitfalls for Java Devs:** `unittest.mock.MagicMock`, `@patch` lookup mechanics (where objects are looked up vs. defined), and why `autospec=True` is mandatory to avoid "silent pass" refactoring bugs.
4. **Profiling & Memory Diagnostics:** Deterministic CPU profiling with `cProfile` and memory allocation tracing with `tracemalloc`.

---

## 🚀 Quick Execution Guide

### Go Suite
```bash
cd day-06-testing-and-benchmarking/go

# 1. Run all unit and table-driven tests
go test -v ./...

# 2. Run data race detection (critical quality gate)
go test -race -v -run TestConcurrentIdempotency ./...

# 3. Run microbenchmarks with memory allocation profiling
go test -bench=. -benchmem -run=^$ ./...
```

### Python Suite
```bash
cd day-06-testing-and-benchmarking/python

# 1. Run dual-mode runner (works with stdlib or invokes pytest if installed)
python3 run_all.py

# 2. Run with pytest (if installed in virtual environment)
pytest -v

# 3. Run CPU & memory profiling demonstration
python3 profile_benchmark.py
```
