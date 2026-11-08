# Python Testing, Benchmarking & Tooling Deep Dive

This guide is crafted for senior engineers moving from the JVM (JUnit 5, Mockito, JMH, VisualVM) to Python 3.15 and `pytest`.

---

## 0. Project Structure: `tests/` Directory vs. Co-location

Java developers often wonder whether Python tests should live in a separate `tests/` directory or alongside source files.

In Python, **both layouts are valid and widely used**:

### Pattern A: Dedicated `tests/` Folder (Common in Libraries & Large Apps)
```text
my_package/
├── pyproject.toml
├── src/
│   └── service.py
└── tests/
    ├── conftest.py             # Global suite fixtures
    ├── test_service.py
    └── test_fixtures.py
```
* **Packaging isolation:** Allows build backends (`setuptools`, `hatch`) to exclude test files when packaging `.whl` wheels for PyPI.
* **Global fixtures:** A root `tests/conftest.py` automatically injects fixtures across all test files without explicit imports.

### Pattern B: Co-located Tests (Common in Microservices, Fast-Iterating Apps & Course Modules)
```text
python/
├── service.py
├── test_table_parameterized.py
├── test_fixtures_lifecycle.py
└── test_mocking_pitfalls.py
```
* **Proximity:** Tests sit right next to the code under test, making navigation instant.
* **Pytest Discovery:** `pytest` does not require a specific directory name. It automatically recurses through the working tree discovering any file matching `test_*.py` or `*_test.py`.

---

## 1. Pytest vs. JUnit 5: The Mental Model Shift

In Java, testing frameworks rely heavily on inheritance, annotations, and reflection:
```java
public class OrderServiceTest {
    @Test
    void testProcessPayment() {
        assertEquals("ord-01", receipt.getOrderId());
    }
}
```

In Python, the standard library includes `unittest` (which was modeled after JUnit 3 in 2001). However, the modern Python ecosystem has almost universally adopted **`pytest`**.

### The Magic of Pytest's AST Rewriting
In Pytest, you do not use `assertEquals(expected, actual)` or AssertJ fluent assertions. You write a **plain Python `assert` statement**:

```python
assert receipt.order_id == "ord-01"
```

Pytest intercepts Python's Abstract Syntax Tree (AST) compilation and rewrites `assert` statements at import time. When an assertion fails, Pytest introspects all intermediate expressions, printing detailed, colored diffs of nested data structures without requiring specialized assertion library methods.

---

## 2. Parameterization: Pytest vs. JUnit `@ParameterizedTest`

In JUnit 5:
```java
@ParameterizedTest(name = "{0}")
@MethodSource("provideValidationCases")
void testValidateOrder(String name, Order order, Class<? extends Exception> expectedEx) { ... }
```

In Pytest:
```python
@pytest.mark.parametrize("order,expected_error", [
    pytest.param(valid_order, None, id="valid_usd_transaction"),
    pytest.param(invalid_order, InvalidAmountError, id="negative_amount_raises_error"),
])
def test_validate_order(order: Order, expected_error: type[Exception] | None) -> None:
    if expected_error:
        with pytest.raises(expected_error):
            svc.validate_order(order)
    else:
        svc.validate_order(order)
```

### Key Differences
- **`pytest.param(..., id="...")`:** Allows naming test cases directly inline, creating clean, readable CLI reports (`test_validate_order[valid_usd_transaction] PASSED`).
- **Cartesian Products:** Stacking multiple `@pytest.mark.parametrize` decorators automatically generates a full Cartesian matrix of test cases without writing nested loops.
- **`pytest.raises(...)` Context Manager:** Replaces JUnit's `assertThrows()`. It captures the exception and allows inspection of exception attributes, arguments, and traceback frames.

---

## 3. Fixtures & Dependency Injection Lifecycle

In Java, test setup relies on lifecycle annotations (`@BeforeEach`, `@AfterEach`, `@BeforeAll`) and class-level fields:
```java
class OrderServiceTest {
    private Store store;
    
    @BeforeEach
    void setUp() { store = new Store(); }
    
    @AfterEach
    void tearDown() { store.clear(); }
}
```

In Pytest, lifecycle management is handled via **Fixtures with true Dependency Injection**:

```python
@pytest.fixture
def memory_store() -> Generator[SafeMemoryStore, None, None]:
    # Setup Phase (executed before the test)
    store = SafeMemoryStore()
    yield store
    # Teardown Phase (guaranteed to execute after the test completes)
    store.clear()
```

### Senior Advantages of Pytest Fixtures
1. **Explicit Parameter Injection:** Tests explicitly declare what they depend on by naming the fixture as a function argument (`def test_something(memory_store): ...`).
2. **Fixture Composition:** Fixtures can request other fixtures! A service fixture can take `gateway` and `store` fixtures as arguments. Pytest automatically computes the dependency DAG (Directed Acyclic Graph) and instantiates them in topological order.
3. **Scope Hierarchy:**
   - `scope="function"` (default): Created and destroyed per test function.
   - `scope="module"`: Created once per `.py` test file (like `@BeforeAll`).
   - `scope="session"`: Created once for the entire test run across all files (ideal for spinning up Docker containers via testcontainers, database connection pools, or schema migrations).

---

## 4. The Mockito-to-Python Traps (Dynamic Duck Typing Gotchas)

### ⚠️ Pitfall 1: Silent Pass on Renamed / Nonexistent Methods
In Java Mockito:
```java
PaymentGateway mock = Mockito.mock(PaymentGateway.class);
when(mock.chrrge(...)).thenReturn(receipt); // COMPILE ERROR: method does not exist!
```

In Python with vanilla `unittest.mock.MagicMock`:
```python
mock_gateway = MagicMock()
mock_gateway.chrrge("ord-1", 1000, "USD") # SUCCEEDS SILENTLY!
```
Because Python is dynamically typed, `MagicMock` assumes any accessed attribute or method call is valid, dynamically returning a new `MagicMock` child! If you refactor a method name in production code, **your unit tests will continue to pass green**, leading to catastrophic production outages.

#### The Senior Solution: Always use `autospec=True` or `create_autospec`
```python
from unittest.mock import create_autospec

mock_gateway = create_autospec(PaymentGateway)
mock_gateway.chrrge(...) # Raises AttributeError: Mock has no attribute 'chrrge'!
```
`create_autospec` inspects the target class or Protocol at creation time, enforcing:
- Method name existence
- Parameter names and parameter count
- Correct argument binding

### ⚠️ Pitfall 2: Where to Patch (`mock.patch`)
In Java, Mockito replaces fields or Spring beans in place.
In Python, `@patch("module_name.ClassName")` dynamically rebinds references in namespaces.
**Rule of thumb:** Patch where an object is **looked up**, NOT where it is defined.
If `service.py` does `from gateway import PaymentGateway`, patching `gateway.PaymentGateway` will NOT affect `service.py`! You must patch `service.PaymentGateway`.

---

## 5. Profiling & Memory Diagnostics vs. JVM Tooling

| Capability | JVM | Python |
| :--- | :--- | :--- |
| **Deterministic CPU Profiling** | Async-Profiler, Java Flight Recorder (JFR) | `cProfile` + `pstats` (built-in C-extension) |
| **Memory Allocation Tracking** | JConsole, VisualVM, Eclipse MAT (.hprof) | `tracemalloc` (built-in line-by-line pymalloc tracing) |
| **Microbenchmarks** | JMH (Java Microbenchmark Harness) | `pytest-benchmark` / custom timing loops |

### Interpreting `cProfile` Output
- `ncalls`: Number of times the function was called.
- `tottime`: Total CPU time spent strictly inside that function (excluding subfunction calls). A high `tottime` indicates an algorithmic bottleneck or tight loop.
- `cumtime`: Total CPU time spent in this function and all subfunctions. High `cumtime` with low `tottime` indicates the bottleneck lies in a downstream callee (e.g., waiting on locks or I/O).

---

## 6. How to Run the Python Demonstrations

```bash
cd day-06-testing-and-benchmarking/python

# 1. Run all tests and profiling through the unified runner
python3 run_all.py

# 2. Run with pytest (if installed in virtualenv)
pytest -v

# 3. Run CPU & memory profiling directly
python3 profile_benchmark.py
```
