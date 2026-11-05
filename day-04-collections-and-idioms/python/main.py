#!/usr/bin/env python3
"""
Day 4 – Collections, Idioms, and Data Structures (Python 3.11+)
Senior Polyglot Crash Course

Demonstrates:
  1. Generator expressions vs list comprehensions (memory behaviour)
  2. Idiomatic pipelines with itertools (groupby, chain, islice, …)
  3. Structural pattern matching (match/case) on nested dict/dataclass payloads
"""

from __future__ import annotations

import sys
import itertools
from dataclasses import dataclass
from typing import Iterator, Iterable, Any


# ---------------------------------------------------------------------------
# 1. Generators vs list comprehensions – memory pressure
# ---------------------------------------------------------------------------

def demonstrate_generators_vs_lists() -> None:
    print("=== 1. Generators vs List Comprehensions ===")

    # Simulate a large data source (here we keep it modest so the demo stays fast)
    N = 1_000_000

    # Eager: builds the entire list in memory
    squares_list = [x * x for x in range(N)]
    print(f"list comprehension:  len={len(squares_list):,}  "
          f"type={type(squares_list).__name__}")

    # Lazy: almost zero extra memory; values are produced on demand
    squares_gen = (x * x for x in range(N))
    print(f"generator expression: type={type(squares_gen).__name__}  "
          f"(no len, consumes on iteration)")

    # Consume only the first few values – the rest are never computed
    first_five = list(itertools.islice(squares_gen, 5))
    print(f"first five from generator: {first_five}")

    # Classic pattern: stream → filter → transform without materialising
    def large_stream() -> Iterator[int]:
        for i in range(10_000_000):  # could be a file, network, DB cursor…
            yield i

    # Only the filtered, transformed values that we actually take are computed
    pipeline = (
        x * 2
        for x in large_stream()
        if x % 1000 == 0
    )
    sample = list(itertools.islice(pipeline, 5))
    print(f"streaming pipeline sample: {sample}")
    print()


# ---------------------------------------------------------------------------
# 2. itertools power tools
# ---------------------------------------------------------------------------

def demonstrate_itertools() -> None:
    print("=== 2. itertools Idioms ===")

    # --- chain: flatten one level without building intermediate lists ---
    nested = [[1, 2], [3, 4, 5], [6]]
    flat = list(itertools.chain.from_iterable(nested))
    print(f"chain.from_iterable: {flat}")

    # --- groupby: consecutive groups (input must be sorted by the key) ---
    rows = [
        {"dept": "eng", "name": "alice"},
        {"dept": "eng", "name": "bob"},
        {"dept": "hr",  "name": "carol"},
        {"dept": "hr",  "name": "dave"},
        {"dept": "eng", "name": "erin"},  # note: not adjacent → new group
    ]
    # Sort first so that identical keys are consecutive
    rows.sort(key=lambda r: r["dept"])

    print("groupby by department:")
    for dept, group in itertools.groupby(rows, key=lambda r: r["dept"]):
        names = [r["name"] for r in group]
        print(f"  {dept}: {names}")

    # --- islice: safe windowing of potentially infinite sources ---
    def fib() -> Iterator[int]:
        a, b = 0, 1
        while True:
            yield a
            a, b = b, a + b

    print(f"first 10 Fibonacci (islice): {list(itertools.islice(fib(), 10))}")

    # --- pairwise (3.10+) / classic recipe for sliding windows ---
    data = [10, 20, 30, 40, 50]
    pairs = list(itertools.pairwise(data))  # [(10,20), (20,30), …]
    print(f"pairwise: {pairs}")

    # --- accumulate – running totals / reductions ---
    running = list(itertools.accumulate(data))
    print(f"accumulate (sum): {running}")
    print()


# ---------------------------------------------------------------------------
# 3. Structural pattern matching on nested payloads
# ---------------------------------------------------------------------------

@dataclass
class Money:
    amount: int
    currency: str


@dataclass
class OrderItem:
    sku: str
    qty: int
    price: Money


def handle_payload(payload: dict[str, Any] | object) -> str:
    """
    Illustrates the expressive power of match/case for real-world message
    shapes.  Guards, nested patterns, and capture variables keep the logic
    flat and readable.
    """
    match payload:
        # Mapping pattern with nested sequence + guard
        case {
            "type": "order",
            "customer_id": str(cid),
            "items": [{"sku": str(sku), "qty": int(qty)}, *_]
        } if qty > 0:
            return f"ORDER from {cid}: first item {sku} x{qty}"

        # Cancellation
        case {"type": "cancel", "order_id": str(oid)}:
            return f"CANCEL order {oid}"

        # Dataclass / class pattern (Python 3.10+)
        case OrderItem(sku=sku, qty=qty, price=Money(amount=amt, currency=cur)):
            return f"ITEM {sku} x{qty} @ {amt} {cur}"

        # Sequence of mixed types
        case [str(cmd), *args] if cmd in {"ping", "status"}:
            return f"CMD {cmd} args={args}"

        # OR pattern
        case {"status": "ok" | "success"}:
            return "SUCCESS status"

        # Catch-all
        case _:
            return f"UNHANDLED: {payload!r}"


def demonstrate_pattern_matching() -> None:
    print("=== 3. Structural Pattern Matching ===")

    examples: list[Any] = [
        {
            "type": "order",
            "customer_id": "C-42",
            "items": [
                {"sku": "WIDGET-1", "qty": 3, "price": 9.99},
                {"sku": "GADGET-9", "qty": 1},
            ],
        },
        {"type": "cancel", "order_id": "ORD-1001"},
        OrderItem(sku="FOO", qty=2, price=Money(1500, "USD")),
        ["ping", "server-1", "verbose"],
        {"status": "success", "code": 200},
        {"type": "unknown", "data": 123},
    ]

    for p in examples:
        print(f"  → {handle_payload(p)}")
    print()


# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------

def main() -> None:
    print(f"Python {sys.version}")
    print()
    demonstrate_generators_vs_lists()
    demonstrate_itertools()
    demonstrate_pattern_matching()
    print("Day 4 Python examples completed.")


if __name__ == "__main__":
    main()
