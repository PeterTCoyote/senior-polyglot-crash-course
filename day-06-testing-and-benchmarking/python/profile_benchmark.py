"""
Day 6 – Deterministic Profiling and Memory Diagnostics (Python 3.15+)
Senior Polyglot Crash Course

JAVA TRANSLATION & ARCHITECTURAL COMPARISON:
In Java:
  - CPU profiling is typically done with Java Flight Recorder (JFR), async-profiler, or VisualVM.
  - Memory analysis uses JConsole, Eclipse MAT, or JVM heap dumps (.hprof).
  - Benchmarks use JMH.

In Python:
  - CPU Profiling: `cProfile` is a built-in deterministic C-extension profiler that hooks
    every function call, return, and exception.
  - Memory Tracing: `tracemalloc` tracks every memory block allocated by Python's pymalloc allocator
    with full source file and line-number attribution.
"""

from __future__ import annotations

import cProfile
import io
import pstats
import tracemalloc
from datetime import datetime, timezone
from service import (
    Order,
    OrderPaymentService,
    PaymentGateway,
    Receipt,
    SafeMemoryStore,
)


class BenchmarkStubGateway:
    """Zero-overhead gateway stub for high-throughput benchmark loops."""

    def charge(self, order_id: str, amount_cents: int, currency: str) -> Receipt:
        return Receipt(
            transaction_id=f"txn-{order_id}",
            order_id=order_id,
            amount_charged=amount_cents,
            currency=currency,
        )


def run_benchmark_workload(iterations: int = 20_000) -> None:
    """
    Simulates a realistic mixed workload:
      - 20% Cache misses requiring new charges and storage writes.
      - 80% Cache hits returning pre-settled idempotency receipts.
    """
    gateway = BenchmarkStubGateway()
    store = SafeMemoryStore()
    svc = OrderPaymentService(gateway=gateway, store=store)

    for i in range(iterations):
        # 80% of orders reuse an ID from the first 20%
        key_id = f"ord-{i if i < iterations // 5 else i % (iterations // 5)}"
        order = Order(
            id=key_id,
            customer_id=f"cust-{i % 100}",
            amount_cents=1000 + (i % 500),
            currency="USD",
        )
        svc.process_payment(order)


def demonstrate_cpu_profiling() -> None:
    print("=" * 70)
    print("1. DETERMINISTIC CPU PROFILING (cProfile + pstats)")
    print("=" * 70)

    profiler = cProfile.Profile()
    profiler.enable()

    run_benchmark_workload(iterations=25_000)

    profiler.disable()

    # Format profiling statistics
    stream = io.StringIO()
    stats = pstats.Stats(profiler, stream=stream)
    stats.strip_dirs()
    stats.sort_stats(pstats.SortKey.CUMULATIVE)
    stats.print_stats(12)  # Print top 12 functions by cumulative time

    print(stream.getvalue())
    print("Senior Insight: Notice 'tottime' vs 'cumtime'.")
    print("  - tottime: Time spent strictly inside that function, excluding calls to subfunctions.")
    print("  - cumtime: Total time spent in this function and all called subfunctions.")
    print("  - Look for lock acquisition overhead (threading.Lock) and dictionary hashing.\n")


def demonstrate_memory_tracing() -> None:
    print("=" * 70)
    print("2. MEMORY ALLOCATION PROFILING (tracemalloc)")
    print("=" * 70)

    # Start tracking memory allocations
    tracemalloc.start()
    snapshot_before = tracemalloc.take_snapshot()

    # Execute workload
    run_benchmark_workload(iterations=25_000)

    snapshot_after = tracemalloc.take_snapshot()
    current, peak = tracemalloc.get_traced_memory()
    tracemalloc.stop()

    print(f"Current Memory Usage : {current / 1024:.2f} KiB")
    print(f"Peak Memory Usage    : {peak / 1024:.2f} KiB\n")

    # Compare snapshot difference (equivalent to comparing two JVM heap dumps in Eclipse MAT)
    print("Top 5 Memory Allocation Diffs by File & Line:")
    top_stats = snapshot_after.compare_to(snapshot_before, "lineno")
    for stat in top_stats[:5]:
        print(f"  {stat}")

    print("\nSenior Insight: Python allocates memory via pymalloc for small objects (<= 512 bytes).")
    print("Notice how @dataclass(slots=True) avoids the overhead of instance __dict__ dictionaries.")


if __name__ == "__main__":
    demonstrate_cpu_profiling()
    demonstrate_memory_tracing()
