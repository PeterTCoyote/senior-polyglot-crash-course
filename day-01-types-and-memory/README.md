# Day 1: Mental Models, Memory, and Type Systems

Welcome to Day 1 of the Senior Polyglot Crash Course. For seasoned developers coming from Java, C#, C++, Scala, or Kotlin, Day 1 strips away beginner syntax and focuses immediately on how Go and Python manage memory, define types, and enforce design contracts.

## 🎯 Senior Mental Model Shift

| Concept | Java / C# / Scala | Go | Python (3.15+) |
| :--- | :--- | :--- | :--- |
| **Type System** | Nominal (Interfaces explicitly declared with `implements`) | Structural (Implicit interfaces; duck typing via behavior) | Gradual typing (`mypy`) + Structural Subtyping (`Protocol`) |
| **Memory Allocation** | Heap-allocated objects, GC, hidden pointer dereference | Value vs. Pointer semantics, stack/heap escape analysis | Heap-allocated reference objects; `__slots__` for memory optimization |
| **Object Orientation** | Class hierarchies, inheritance (`extends`), polymorphism | Composition over inheritance; no classes, only structs | Multiple inheritance, mixins, dataclasses, composition |

---

## 🗺️ What We Cover Today

1. **Go Memory & Structs:** Zero values, embedding, pointer vs value receivers, and the method-set rule (`T` vs `*T`).
2. **Go Structural Interfaces:** Implementing interfaces implicitly; copies in variables vs pointers stored in an interface.
3. **Python Dataclasses & Memory (`__slots__`):** Comparing slotted instances against `__dict__` overhead — not a single `getsizeof` number.
4. **Python Structural Subtyping (`typing.Protocol`):** Static contracts via mypy/pyright; `isinstance` only checks method names.

---

## 🚀 How to Run Today's Code

### Running Go Code
```bash
cd day-01-types-and-memory/go
go run main.go
```

### Running Python Code
```bash
cd day-01-types-and-memory/python
python3 main.py
```

