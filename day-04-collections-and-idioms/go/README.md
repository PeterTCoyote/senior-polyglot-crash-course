# Go: Collections, Idioms, and Data Structures (Day 4)

## Slice Header Semantics

A slice is a three-word structure (on 64-bit):

```
type slice struct {
    ptr unsafe.Pointer  // pointer to the first element of the backing array
    len int             // number of elements in the slice
    cap int             // capacity of the backing array starting from ptr
}
```

- Assigning or passing a slice **copies the header**.
- The backing array is shared until an `append` (or similar growth) allocates a new one.
- `len` and `cap` of the *caller's* header are never mutated by a callee; only the callee's *copy* of the header changes.

### Classic Trap: Shared Backing Array + Append

```go
a := []int{1, 2, 3, 4, 5}
b := a[1:3]          // b shares the same array, len=2, cap=4
b = append(b, 99)    // writes into the shared array → a[3] becomes 99
```

When `len == cap`, `append` allocates a new array, copies the elements, and the original slice is unaffected. Until that point, mutations are visible to every slice that shares the array.

### Sub-slicing and Memory Leaks

Keeping a small sub-slice of a large array keeps the *entire* array alive:

```go
big := make([]byte, 1<<20)          // 1 MiB
small := big[0:10]                  // still references the 1 MiB array
// big goes out of scope → GC cannot free the 1 MiB
```

Idiom: force a copy when you only need a small piece for a long time:

```go
small = append([]byte(nil), big[0:10]...)
// or
small = bytes.Clone(big[0:10]) // Go 1.20+
```

---

## Map Concurrency

Go maps are **not** safe for concurrent read/write or concurrent write/write.

- The runtime detects concurrent map writes and terminates the program with a fatal error (`fatal error: concurrent map writes`).
- Concurrent reads are safe *only if* no writes are happening.
- There is no built-in concurrent map. Common patterns:

  1. `sync.RWMutex` around a plain `map`.
  2. `sync.Map` (optimized for the “write once, read many” or “disjoint keys” cases).
  3. Sharding (multiple maps + mutexes) when contention is high.

Never rely on “it worked in testing.” The race detector (`go test -race`) and the runtime check are your friends.

---

## Generics (Go 1.18+) vs `interface{}` / `any`

Before 1.18 the only way to write generic-looking code was `interface{}` (now aliased as `any`) plus type assertions. That loses static safety and forces boxing.

```go
// Pre-generics style (still valid, but avoid for new libraries)
func FilterAny(s []any, pred func(any) bool) []any { ... }

// Modern
func Filter[T any](s []T, pred func(T) bool) []T { ... }
```

Constraints:

- `any` ≡ `interface{}` — no methods required.
- `comparable` — types that support `==` and `!=` (needed for map keys).
- Custom constraints via interfaces with type sets.

Generics are monomorphized: each concrete instantiation is compiled separately, so there is no runtime boxing cost for the type parameter itself.

---

## Idiomatic Patterns You Should Internalize

1. Prefer `make([]T, 0, estimatedCap)` when you know an upper bound — avoids repeated reallocations.
2. Prefer `append` over manual index management for growth.
3. When returning a slice from a function that owns a larger buffer, either document the sharing or copy.
4. Use generics for reusable algorithms; keep `any` for truly heterogeneous bags (e.g., JSON intermediate values).
5. Always protect maps that can be written from multiple goroutines.

The companion `main.go` demonstrates all of the above with runnable, heavily commented examples.
