# Python Error Handling & Context Managers

## Architectural Deep Dive

Python's exception model relies on dynamic type-based dispatch and stack unwinding. For enterprise systems, robust error engineering requires careful exception hierarchies, precise logging, and deterministic resource management via context managers.

### 1. Enterprise Exception Hierarchies
* **Base Application Exception (`AppError`):** All domain exceptions must inherit from a single custom base class (`AppError(Exception)`). Never catch raw `Exception` blindly unless at the absolute application boundary (e.g., top-level worker thread or ASGI/WSGI middleware).
* **Exception Chaining (`raise NewError(...) from err`):** Preserves the original traceback and cause context when translating low-level infrastructure errors (e.g., SQLAlchemy/psycopg2 errors) into high-level domain exceptions.

### 2. Structured Logging Best Practices
* Avoid `print()` or bare `except:` statements.
* Use Python's built-in `logging` module with `logger.exception()` or `exc_info=True` inside `except` blocks to capture full stack traces automatically without losing context.

### 3. Context Managers (`PEP 343`)
* **`__enter__` / `__exit__` Protocols:** Ensure deterministic resource acquisition and release (files, database sessions, thread locks, network sockets) without repetitive `try...finally` boilerplate.
* **`contextlib.contextmanager`:** Allows writing concise generator-based context managers where code before `yield` acts as setup, and code after `yield` (within `finally` or explicit try/except blocks) acts as teardown.