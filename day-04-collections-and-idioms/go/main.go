// Day 4 – Collections, Idioms, and Data Structures (Go)
// Senior Polyglot Crash Course
//
// Demonstrates:
//   1. Slice header semantics and shared-backing-array traps
//   2. Safe concurrent map access with a thin RWMutex wrapper
//   3. Modern generics: type-safe Filter/Map and a generic Cache

package main

import (
	"fmt"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// 1. Slice header & shared backing array
// ---------------------------------------------------------------------------

func demonstrateSliceSemantics() {
	fmt.Println("=== 1. Slice Header & Shared Backing Array ===")

	// make allocates a backing array of capacity 5, length 3
	a := make([]int, 3, 5)
	a[0], a[1], a[2] = 10, 20, 30
	fmt.Printf("a          = %v  (len=%d, cap=%d)\n", a, len(a), cap(a))

	// b is a new header that points into the same array
	b := a[1:3] // elements 20, 30; remaining capacity = 4 (indices 1..4)
	fmt.Printf("b          = %v  (len=%d, cap=%d)\n", b, len(b), cap(b))

	// append while len < cap reuses the same backing array.
	// The write lands at index 3 of the original array.
	b = append(b, 99)
	fmt.Printf("after append(b, 99):\n")
	fmt.Printf("  b        = %v\n", b)
	// a's length is still 3, so printing a hides the mutation.
	// Extending a's length (without reallocating) reveals the shared write.
	a = a[:4]
	fmt.Printf("  a[:4]    = %v  ← index 3 was overwritten through the shared array!\n", a)

	// Force a reallocation: grow past the original capacity.
	b = append(b, 100, 101) // exceeds remaining capacity → new backing array
	fmt.Printf("after further append that forces reallocation:\n")
	fmt.Printf("  a        = %v  ← original array unchanged by the new b\n", a)
	fmt.Printf("  b        = %v  (new backing array, no longer shares with a)\n", b)

	// Classic memory-leak pattern (commented out so we don't actually leak)
	// big := make([]byte, 1<<20)
	// small := big[:10]               // keeps the whole 1 MiB alive
	// small = append([]byte(nil), big[:10]...)  // safe copy

	fmt.Println()
}

// ---------------------------------------------------------------------------
// 2. Concurrent map – the unsafe way vs a safe wrapper
// ---------------------------------------------------------------------------

// SafeMap is a thin, type-safe wrapper around map[K]V protected by an RWMutex.
// Prefer this (or sync.Map when the access pattern fits) over naked maps that
// are touched by multiple goroutines.
type SafeMap[K comparable, V any] struct {
	mu sync.RWMutex
	m  map[K]V
}

func NewSafeMap[K comparable, V any]() *SafeMap[K, V] {
	return &SafeMap[K, V]{m: make(map[K]V)}
}

func (s *SafeMap[K, V]) Set(key K, value V) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[key] = value
}

func (s *SafeMap[K, V]) Get(key K) (V, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.m[key]
	return v, ok
}

func (s *SafeMap[K, V]) Delete(key K) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, key)
}

func (s *SafeMap[K, V]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.m)
}

func demonstrateConcurrentMap() {
	fmt.Println("=== 2. Concurrent Map Access ===")

	// --- Unsafe illustration (DO NOT DO THIS IN PRODUCTION) ---
	// Uncommenting the following block will usually produce:
	//   fatal error: concurrent map writes
	/*
		unsafe := make(map[int]int)
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				unsafe[n] = n * n   // concurrent writes → fatal
			}(i)
		}
		wg.Wait()
	*/

	// --- Safe wrapper ---
	safe := NewSafeMap[string, int]()
	var wg sync.WaitGroup
	const workers = 50

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("worker-%d", id)
			safe.Set(key, id*id)
			if v, ok := safe.Get(key); ok {
				_ = v // use the value
			}
		}(i)
	}
	wg.Wait()

	fmt.Printf("SafeMap ended with %d entries (expected %d)\n", safe.Len(), workers)
	fmt.Println()
}

// ---------------------------------------------------------------------------
// 3. Generics – Filter, Map, and a simple Cache
// ---------------------------------------------------------------------------

// Filter returns a new slice containing only the elements for which pred is true.
// Zero allocation when the result is empty; otherwise a single allocation of the
// exact required size is possible but we keep it simple here.
func Filter[T any](s []T, pred func(T) bool) []T {
	out := make([]T, 0, len(s)) // optimistic capacity
	for _, v := range s {
		if pred(v) {
			out = append(out, v)
		}
	}
	return out
}

// Map applies f to every element and returns a new slice of the results.
func Map[T, U any](s []T, f func(T) U) []U {
	out := make([]U, len(s))
	for i, v := range s {
		out[i] = f(v)
	}
	return out
}

// Cache is a trivial generic TTL-less cache.
// K must be comparable so it can be used as a map key.
type Cache[K comparable, V any] struct {
	mu    sync.RWMutex
	items map[K]V
}

func NewCache[K comparable, V any]() *Cache[K, V] {
	return &Cache[K, V]{items: make(map[K]V)}
}

func (c *Cache[K, V]) Put(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = value
}

func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.items[key]
	return v, ok
}

func demonstrateGenerics() {
	fmt.Println("=== 3. Generics: Filter, Map, Cache ===")

	nums := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	evens := Filter(nums, func(n int) bool { return n%2 == 0 })
	fmt.Printf("Filter even: %v\n", evens)

	squares := Map(nums, func(n int) int { return n * n })
	fmt.Printf("Map square:  %v\n", squares)

	// Heterogeneous transformation is also fine
	strs := Map(nums, func(n int) string { return fmt.Sprintf("n=%d", n) })
	fmt.Printf("Map to str:  %v\n", strs)

	// Generic cache
	cache := NewCache[string, time.Time]()
	cache.Put("start", time.Now())
	if t, ok := cache.Get("start"); ok {
		fmt.Printf("Cache hit:   %v\n", t.Format(time.RFC3339Nano))
	}

	fmt.Println()
}

func main() {
	demonstrateSliceSemantics()
	demonstrateConcurrentMap()
	demonstrateGenerics()
	fmt.Println("Day 4 Go examples completed.")
}
