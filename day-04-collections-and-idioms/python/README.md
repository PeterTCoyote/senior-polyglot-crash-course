# Python: Collections, Idioms, and Data Structures (Day 4)

## Memory Layout & Iteration Cost

### `list`
- Contiguous array of *pointers* to PyObjects.
- Overallocation formula (CPython) grows by ~12.5 % after a threshold → amortized O(1) append.
- Insert/delete in the middle is O(n) because pointers must be shifted.
- A list of a million small integers still costs a million pointers + the integer objects themselves.

### `dict` / `set`
- Since 3.6: compact, insertion-ordered representation.
- Open-addressing with a separate index table. Average O(1), but pathological key distributions can degrade (hash randomization mitigates this).
- Keys must be hashable (`__hash__` + `__eq__`).

### Generators vs Materialized Collections

| Construct | Evaluates | Memory | Use when |
|-----------|-----------|--------|----------|
| List comprehension `[…]` | Eagerly | O(n) | You need random access or multiple passes |
| Generator expression `(…)` | Lazily | O(1) | Streaming, single-pass, large or infinite data |
| Generator function (`yield`) | Lazily | O(1) | Complex control flow, state, or cleanup |

Never turn a multi-gigabyte stream into a list unless you truly need it.

---

## Idiomatic Transformation Pipeline

Prefer composing small, pure functions with generators and `itertools` over large intermediate lists:

```python
from itertools import islice, groupby, chain

# stream → filter → transform → group → materialize only the final result
result = list(
    groupby(
        (transform(x) for x in source if predicate(x)),
        key=keyfn
    )
)
```

Useful `itertools` tools for seniors:

- `chain` / `chain.from_iterable` – flatten without building intermediate lists
- `groupby` – consecutive grouping (input must already be sorted by the key)
- `islice` – safe windowing / pagination of infinite streams
- `tee` – multiple independent iterators over the same source (with memory caveats)
- `accumulate`, `starmap`, `product`, `combinations` – classic combinatorial patterns

---

## Structural Pattern Matching (3.10+)

`match`/`case` is far more than a glorified `switch`. It supports:

- Capture patterns
- Guard clauses (`if` conditions)
- Sequence, mapping, and class patterns
- Nested destructuring
- OR patterns (`|`)
- Wildcard `_`

Ideal for dissecting nested dictionaries, dataclasses, or message payloads without a cascade of `.get()` calls and type checks.

```python
match payload:
    case {"type": "order", "items": [{"sku": sku, "qty": qty}, *_]} if qty > 0:
        ...
    case {"type": "cancel", "order_id": order_id}:
        ...
    case _:
        ...
```

---

## Mental Model for Seniors Coming from Java/C#/Rust

- There is no equivalent of a “slice header” that can accidentally share a mutable buffer; Python objects are reference-counted (or GC’d) and lists hold references.
- Mutation of a list is visible to every reference that points at the same list object.
- “Copy” is almost always shallow unless you explicitly use `copy.deepcopy` or a comprehension that builds new objects.
- Prefer generators and iterators when the data volume is unknown or large; reach for lists only when you need the random-access contract.

The companion `main.py` exercises generators under memory pressure, `itertools` composition, and structural pattern matching on realistic nested payloads.
