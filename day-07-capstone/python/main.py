"""
Production CLI entry point for the Day 7 Python Capstone:
Streaming Async ETL Pipeline with Graceful Signal Handling & DLQ Diagnostics.
"""

from __future__ import annotations

import asyncio
import os
import random
import signal
import sys
import time
from typing import Any, Dict

from pipeline import AsyncETLPipeline


async def main() -> None:
    print("================================================================================")
    print("🚀 DAY 7 CAPSTONE (Python): Asynchronous Streaming ETL Pipeline")
    print("   Architecture: asyncio.Queue Backpressure, Pydantic/Typing, Retries, DLQ, Micro-Batching")
    print("================================================================================")

    pipeline = AsyncETLPipeline(
        queue_capacity=50,
        num_workers=4,
        max_retries=2,
        initial_backoff=0.02,
        max_backoff=0.1,
        batch_size=10,
        flush_interval=0.2,
    )

    await pipeline.start()
    print("[Init] ETL Pipeline started with 4 async workers, bounded queue capacity: 50")

    # Graceful shutdown event
    stop_event = asyncio.Event()

    def handle_signal() -> None:
        print("\n[Signal] OS interrupt detected (SIGINT/SIGTERM)! Triggering graceful shutdown...")
        stop_event.set()

    # Register OS signal handlers
    loop = asyncio.get_running_loop()
    if sys.platform != "win32":
        for sig in (signal.SIGINT, signal.SIGTERM):
            loop.add_signal_handler(sig, handle_signal)

    # Ingress Producer Task: Generates raw simulated IoT telemetry
    metrics_catalog = ["cpu_usage", "memory_mb", "temperature_c", "network_bytes", "request_latency_ms"]
    total_records = 40

    print(f"[Producer] Streaming {total_records} raw telemetry payloads... (Press Ctrl+C to test graceful abort)\n")

    async def produce_telemetry() -> None:
        for i in range(1, total_records + 1):
            if stop_event.is_set():
                break

            metric = random.choice(metrics_catalog)
            val: float
            if metric == "cpu_usage":
                val = round(random.uniform(5.0, 95.0), 2)
            elif metric == "temperature_c":
                val = round(random.uniform(18.0, 85.0), 2)
            elif metric == "memory_mb":
                val = round(random.uniform(512.0, 16384.0), 2)
            else:
                val = round(random.uniform(10.0, 5000.0), 2)

            device_id = f"dev-sensor-{random.randint(1, 10):02d}"
            sim_transient = False

            # Inject intentional failure cases to test resilience
            if i == 5:
                # Malformed device ID
                device_id = "INVALID_DEVICE_WITHOUT_PREFIX"
            elif i == 12:
                # Unsupported metric
                metric = "quantum_flux_unsupported"
            elif i == 20:
                # Value violates physical constraint
                val = -300.0  # Below absolute zero
            elif i == 28:
                # Transient network glitch eligible for retry
                sim_transient = True

            payload: Dict[str, Any] = {
                "device_id": device_id,
                "metric": metric,
                "value": val,
                "timestamp": time.time(),
                "metadata": {"env": "prod", "seq": str(i)},
                "_simulate_transient_failure": sim_transient,
            }

            enqueued = await pipeline.enqueue(payload, timeout=1.0)
            if not enqueued:
                print(f"  [Backpressure] Ingress throttled for record {i}")

            print(f"  📥 [Ingress #{i:02d}] {device_id} | {metric}: {val}")
            await asyncio.sleep(0.04)  # Simulate event cadence

    # Run producer concurrently with stop_event listener
    producer_task = asyncio.create_task(produce_telemetry())
    stop_waiter = asyncio.create_task(stop_event.wait())

    done, pending = await asyncio.wait(
        [producer_task, stop_waiter],
        return_when=asyncio.FIRST_COMPLETED,
    )

    for task in pending:
        task.cancel()

    print("\n[Lifecycle] Ingress stream completed. Initiating graceful shutdown...")
    start_shutdown = time.perf_counter()
    await pipeline.shutdown(timeout=3.0)
    elapsed_shutdown = time.perf_counter() - start_shutdown
    print(f"✅ [Shutdown] Pipeline cleanly drained and flushed in {elapsed_shutdown:.3f}s")

    # Observability & Diagnostics Summary
    m = pipeline.metrics
    print("\n================================================================================")
    print("📊 ETL PIPELINE OBSERVABILITY & TELEMETRY REPORT")
    print("================================================================================")
    print(f"  • Raw Records Extracted:     {m.extracted}")
    print(f"  • Successfully Validated:    {m.validated}")
    print(f"  • Persisted in Micro-Batches:{m.records_persisted} (across {m.batches_flushed} batches)")
    print(f"  • Transient Retries:         {m.retried}")
    print(f"  • Rejected Records in DLQ:   {len(pipeline.dlq)}")
    print("--------------------------------------------------------------------------------")
    print("📋 DEAD-LETTER QUEUE (DLQ) DIAGNOSTICS:")
    if pipeline.dlq:
        for idx, dlq_record in enumerate(pipeline.dlq, 1):
            print(f"  [{idx}] Stage: {dlq_record.failed_at_stage:<15} | Reason: {dlq_record.error_reason}")
    else:
        print("  (Empty - zero failed records)")
    print("================================================================================")


if __name__ == "__main__":
    asyncio.run(main())
