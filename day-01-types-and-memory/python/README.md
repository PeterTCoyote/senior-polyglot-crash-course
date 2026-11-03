# Python Type System & Memory Optimization for Seniors

## 1. Memory Optimization with `__slots__`
By default, Python instances store attributes in a dynamic dictionary (`__dict__`). For high-throughput services holding millions of objects, that dictionary is extra per-object overhead.
* **The Senior Fix:** `@dataclass(slots=True)` lays out attributes in a fixed structure and drops `__dict__`.
* **Measure the right thing:** `sys.getsizeof(obj)` is only the instance header. Compare a slotted type against a default dataclass **plus** `sys.getsizeof(obj.__dict__)`. Do not treat a single byte count as proof.

## 2. Structural Subtyping with `typing.Protocol`
Python is dynamically duck-typed. Static checkers need an explicit contract.
* **The Senior Fix:** `Protocol` is structural. A class does not inherit from it; if the methods match, mypy/pyright accept it.
* **Runtime trap:** `@runtime_checkable` lets `isinstance(x, Notifier)` work, but it only checks that the **method names exist**, not signatures or return types. Wrong parameter types still pass `isinstance` and fail a type checker.

## 3. Variables Are References
There is no Go-style value-vs-pointer choice. `user` and `pager` are references to heap objects. Mutation is visible to every alias of that object; copy the object (or use an immutable/`frozen` dataclass) when you need isolation.
