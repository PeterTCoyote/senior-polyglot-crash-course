# Day 3: Concurrency and Parallelism Paradigms

## Senior Mental Model: Execution Models at Scale

When designing high-throughput, low-latency distributed or monolithic systems, understanding how runtime schedulers map logical tasks to physical hardware cores is paramount. Below is an architectural comparison of the concurrency models across Java/C#, Go, and Python.

| Dimension | OS Threads (Java / C#) | Go M:N Scheduler (Goroutines) | Python (GIL + Asyncio / Multiprocessing) |
| :--- | :--- | :--- | :--- |
| **Execution Model** | 1:1 mapping to kernel threads. | M:N scheduling (M goroutines multiplexed onto N OS threads). | Single-threaded event loop (asyncio) or separate processes (multiprocessing). |
| **Memory Overhead** | ~1 MB default stack size (can be tuned down, but prone to `StackOverflowError`). | ~2 KB initial growable stack (heaps up to 1 GB dynamically). | ~2-8 KB per coroutine frame; OS processes require full memory replication (Copy-on-Write). |
| **Context Switch Cost** | High (~1-2 µs, requires kernel mode transition, register saving/restoring, TLB flushes). | Low (~100-200 ns, user-space scheduler, cooperative/preemptive yield, minimal register saving). | Near zero for `asyncio` (cooperative yield in user space); High for OS threads/processes due to GIL contention and IPC serialization. |
| **Synchronization** | OS-level mutexes, semaphores, condition variables (futex syscalls under heavy contention). | Communicating Sequential Processes (CSP) via channels + lightweight atomics/mutexes (`sync`, `sync/atomic`). | Cooperative single-threaded execution (no data races within event loop) or multiprocessing IPC queues (`pickle` overhead). |
| **Failure Domain** | Thread crash can corrupt shared heap or terminate process if unhandled. | Unrecovered panic in a goroutine crashes the entire Go runtime process. | Unhandled task exception in asyncio logs warning/error; process isolation in multiprocessing. |