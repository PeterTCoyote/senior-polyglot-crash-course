# Go Concurrency Architecture Deep Dive

## 1. The GMP Scheduler Model
Go utilizes a work-stealing M:N scheduler consisting of three primary entities:
- **G (Goroutine):** Represents the lightweight execution thread, containing instruction pointer, stack, and status.
- **M (Machine):** Represents an OS thread managed by the kernel.
- **P (Processor):** Represents a logical resource or context required to execute Go code (bounded by `GOMAXPROCS`).

## 2. Communicating Sequential Processes (CSP) & Channels
Go's mantra: *"Do not communicate by sharing memory; instead, share memory by communicating."*
- **Unbuffered Channels:** Synchronous handoff. The sender blocks until a receiver is ready, ensuring strict synchronization and zero-copy transfer semantics.
- **Buffered Channels:** Asynchronous bounded queue. Senders block only when the buffer is full; receivers block when empty. Introduces bounded memory overhead (`cap(ch)`).

## 3. Runtime Panic Triggers
Writing incorrect channel logic results in immediate runtime panics (unrecoverable via standard `recover()` if sent to a closed channel):
1. **Closing a closed channel:** Triggers `panic: close of closed channel`.
2. **Writing to a closed channel:** Triggers `panic: send on closed channel`.
3. **Reading from a closed channel:** Returns the zero-value of the channel type immediately, alongside a `false` second return value (`val, ok := <-ch`).

## 4. Synchronization Primitives & Atomics
- **`sync.WaitGroup`:** Counter-based coordination. Must be passed by pointer (`*sync.WaitGroup`) to avoid shallow copying bugs.
- **Atomics (`sync/atomic`):** Lock-free memory access utilizing CPU cache coherence protocols (MESI). Avoids bus locking overhead when mutating simple counters or pointers.
- **Mutexes (`sync.Mutex` / `sync.RWMutex`):** Guard critical sections. Prefer channels for coordination and ownership transfer; use mutexes for state encapsulation inside structs.

## 5. Cancellation & Timeout Propagation (`context.Context`)
`context.Context` forms a tree-like hierarchy that flows downward through function calls. It provides deadline enforcement, cancellation signals, and request-scoped key-value storage. Always pass `ctx` as the first parameter (`ctx context.Context`) to asynchronous workers and IO-bound boundaries.