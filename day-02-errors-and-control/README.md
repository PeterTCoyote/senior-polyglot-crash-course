# Day 2: Error Handling, Failures, and Flow Control

## Senior Mental Model: Paradigms at Scale

For senior engineers coming from Java, C#, C++, Rust, or Scala, error handling is not merely a syntax choice—it is a core architectural constraint that dictates how control flow, resource safety, and failure recovery are reasoned about across distributed boundaries.

### 1. Java / C# (`try/catch/finally`)
* **Mechanism:** Implicit non-local control flow driven by stack unwinding and exception tables.
* **Trade-offs:** 
  * *Pros:* Clean separation of happy path and error handling code; stack traces captured automatically.
  * *Cons:* Hidden control flow branches (every line can potentially throw); performance overhead in hot loops when exceptions are thrown; checked exceptions in Java encourage anti-patterns like `catch (Exception e)` or meaningless `throws` declarations; asynchronous and reactive pipelines complicate exception propagation.

### 2. Go (`Errors-as-Values`)
* **Mechanism:** Explicit return values adhering to the built-in `error` interface (`Error() string`).
* **Trade-offs:**
  * *Pros:* Zero hidden control flow; forced inspection of failure modes at the call site; highly predictable performance; simple composition.
  * *Cons:* Boilerplate `if err != nil` checks if not abstracted properly; requires deliberate discipline to wrap errors with contextual metadata (`%w`).

### 3. Python (`Exception Hierarchies` & `Context Managers`)
* **Mechanism:** Dynamic exception propagation with lexical scope unwinding and PEP 343 resource management (`with` statements).
* **Trade-offs:**
  * *Pros:* Rich exception hierarchies; powerful introspection (`sys.exc_info()`, traceback module); deterministic resource cleanup via context managers.
  * *Cons:* Dynamic typing means failure modes are often discovered at runtime; broad `except Exception:` clauses can silently swallow critical signals (like `KeyboardInterrupt` if improperly catching `BaseException`).

---

## Architectural Comparison Matrix

| Dimension | Go (`error` + `defer`) | Python (`try/except` + `with`) | Java / C# (`try/catch` + `try-with-resources`) |
| :--- | :--- | :--- | :--- |
| **Control Flow** | Explicit, linear branching | Implicit stack unwinding | Implicit stack unwinding |
| **Resource Safety** | LIFO `defer` stack | Context Managers (`__exit__`) | `try-with-resources` / `IDisposable` |
| **Performance Overhead** | Near zero (no unwinding cost) | Low-to-moderate (optimized in 3.11+) | Moderate (exception table lookup) |
| **Failure Composability** | Wrapping `%w`, `errors.Is/As` | Exception chaining (`from err`) | Inner exceptions / `AggregateException` |
