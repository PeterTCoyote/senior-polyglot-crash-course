# Senior Dev 7-Day Crash Course: Go & Python

A high-density, production-focused crash course designed specifically for experienced software engineers (Java, C#, C++, Scala, Kotlin, Rust) learning **Go** and **Python**.

This repository skips basic programming concepts (loops, basic conditionals, simple functions) and focuses entirely on **mental model translations**, **type systems**, **concurrency models**, **idiomatic patterns**, and **production gotchas**.

---

## 🎯 Target Audience

- Engineers with **5+ years of experience** in typed, object-oriented, or functional languages.
- Developers looking to quickly adopt Go and Python for microservices, cloud infrastructure, AI/ML integrations, or CLI tooling.
- Teams onboarding senior talent into Go/Python stacks.

---

## 🗺️ 7-Day Curriculum Overview

| Day | Topic | Go Concepts Covered | Python Concepts Covered | Senior Translation Focus |
| :--- | :--- | :--- | :--- | :--- |
| **Day 1** | **Memory & Type Systems** | Pointers, zero values, structs, implicit interfaces, composition over inheritance | Duck typing, structural subtyping (`Protocol`), static typing (`mypy`), `dataclasses`, `__slots__` | Nominal typing vs. Structural/Gradual typing |
| **Day 2** | **Error Handling & Control** | Errors as values (`error` interface), sentinel errors, error wrapping (`%w`), `defer` mechanics | Exception hierarchies, custom context managers (`with`), explicit failure paths | Moving away from standard `try/catch` control flow |
| **Day 3** | **Concurrency & Parallelism** | Goroutines (M:N scheduler), channels (CSP model), `sync.WaitGroup`, `context.Context` | The GIL, `asyncio` event loops (`async`/`await`), `multiprocessing`, `concurrent.futures` | Threading vs. CSP vs. Async Event Loops |
| **Day 4** | **Collections & Data Structures** | Slices (backing arrays, capacity/len bugs), maps (thread safety), generics (`[T any]`) | Comprehensions, generators (`yield`), `itertools`, pattern matching (`match/case`) | Avoiding mutation traps and memory layout gotchas |
| **Day 5** | **Modules & Project Structure** | `go.mod`, package visibility (capitalization), workspaces, internal packages | Virtual environments (`venv`/Poetry), `pyproject.toml`, circular import mitigation | Dependency management and packaging paradigms |
| **Day 6** | **Testing, Benchmarking & Tooling** | Table-driven tests, `go test -race`, built-in benchmarks, `golangci-lint` | `pytest` fixtures, parameterization, mocking, profiling (`cProfile`) | Race conditions and production quality gates |
| **Day 7** | **Capstone Project** | High-throughput concurrent worker pool with graceful shutdown & context propagation | Async ETL pipeline / API client with Pydantic validation & error recovery | Architecture, resilience, and production readiness |

---

## 📂 Repository Structure

```text
.
├── day-01-types-and-memory/
│   ├── go/
│   │   ├── go.mod
│   │   ├── main.go
│   │   └── README.md
│   └── python/
│       ├── main.py
│       └── README.md
├── day-02-errors-and-control/
├── day-03-concurrency/
├── day-04-collections-and-idioms/
├── day-05-modules-and-tooling/
├── day-06-testing-and-benchmarking/
└── day-07-capstone/
```

---

## ⚡ Prerequisites & Setup

### Go Setup
- **Go version:** `1.27+`
- **Verify installation:**
  ```bash
  go version
  ```

### Python Setup
- **Python version:** `3.15+`
- **Setup virtual environment:**
  ```bash
  python3 -m venv .venv
  source .venv/bin/activate  # On Windows: .venv\Scripts\activate
  ```

---

## 🚀 How to Use This Course

1. Navigate to the specific day's directory (e.g., `cd day-01-types-and-memory/go`).
2. Read the day's `README.md` for a high-level mental model comparison.
3. Inspect and run the heavily annotated `main.go` or `main.py` source files.
4. Experiment by modifying the code to observe compiler behaviors, runtime errors, or performance implications.