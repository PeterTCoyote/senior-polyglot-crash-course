# Day 4: Collections, Idioms, and Data Structures

**Senior Polyglot Crash Course: Go & Python**

> Targeted at seasoned engineers (Java, C#, C++, Scala, Kotlin, Rust) with 10+ years of experience.

This day deliberately skips introductory collection APIs. We focus on the *mental models* that prevent production incidents: memory layouts, ownership of backing storage, concurrency contracts, and the cost of abstraction.

---

## Senior Mental Model Comparison

### Java `ArrayList` / C# `List<T>` vs Go Slices

| Concept | Java `ArrayList` / C# `List<T>` | Go Slice |
|---------|--------------------------------|----------|
| What you hold | Object reference to a growable array wrapper | A **header** (pointer + len + cap) — value type |
| Growth strategy | Usually 1.5× or 2×, amortized O(1) | Doubles capacity when needed; same amortized cost |
| Sub-list / sub-slice | `subList` shares the *same* backing array (Java) or creates a view; mutation can be surprising | Sub-slice *always* shares the backing array until `append` forces a reallocation |
| Capacity control | `ensureCapacity` / `Capacity` property | Explicit `make([]T, len, cap)` and `append` |
| Mutation traps | Concurrent modification → `ConcurrentModificationException` (fail-fast) | Silent data races or unexpected sharing of backing storage |
| Zero value | `null` | `nil` slice (len=0, cap=0, ptr=nil) — perfectly usable |

**Key senior insight**: In Go a slice is a *value*. Passing a slice to a function copies the three-word header. The callee can change `len`/`cap` of *its copy* and can mutate the shared backing array. This is the root of most “why did my original slice change?” bugs.

### Python `list` / `dict` Internal Reality

- **`list`**: Contiguous array of *PyObject pointers*. Resizing follows an overallocation formula (roughly 1.125× after a certain size). Inserts in the middle are O(n). The list object itself is a fixed-size header that points at the array.
- **`dict` (and `set`)**: Open-addressing hash table with a compact key/value array + index table (since Python 3.6 the insertion-ordered compact representation). Average O(1) lookup, but worst-case degradation under pathological hashing is mitigated by randomized hash seeds.
- Memory layout is *not* contiguous for the objects themselves — only the pointers/references are. Large object graphs can therefore have surprising RSS.

### Generics vs Dynamics

| Language | Approach | Compile-time safety | Runtime cost | Idiomatic style |
|----------|----------|---------------------|--------------|-----------------|
| Go 1.18+ | Parametric polymorphism `[T any]` / `[K comparable, V any]` | Yes | Zero (monomorphized) | Prefer generics for libraries; `any` only when truly heterogeneous |
| Pre-1.18 Go | `interface{}` / type assertions | No | Boxing + assertions | Still seen in older codebases |
| Python | Duck typing + gradual typing (`typing.Generic`) | Optional (mypy/pyright) | Runtime type checks only if you add them | Comprehensions, generators, `Protocol`s |

---

## What You Will Build Today

1. **Go**
   - Demonstrate slice header mutation and shared backing arrays.
   - Show why concurrent map access is fatal and how to wrap it safely.
   - Write a small generic, type-safe utility (`Filter` / `Map` style + a concurrent cache).

2. **Python**
   - Contrast generator expressions vs list comprehensions under memory pressure.
   - Compose data pipelines with `itertools`.
   - Use structural pattern matching (`match`/`case`) on nested payloads.

---

## How to Run

```bash
# Go
cd go
go run main.go

# Python (3.11+)
cd python
python main.py
```

---

## Learning Objectives Checklist

- [ ] Explain the three fields of a Go slice header and what happens on `append` when `len == cap`.
- [ ] Predict whether two slices share a backing array after a given sequence of operations.
- [ ] Know that Go maps are not safe for concurrent use and what the runtime does when they are misused.
- [ ] Write a generic function constrained by `comparable` or `any`.
- [ ] Choose between a list comprehension and a generator expression based on memory characteristics.
- [ ] Apply `itertools.groupby`, `chain`, and friends without materializing intermediate lists.
- [ ] Destructure nested dictionaries or dataclasses with `match`/`case`.

---

*Next: Day 5 — Concurrency models (goroutines + channels vs asyncio + threads) and the shared-memory vs message-passing trade-offs.*
