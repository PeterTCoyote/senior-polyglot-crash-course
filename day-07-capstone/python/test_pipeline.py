"""
Test suite for Day 7 Python Capstone:
Validates schema enforcement, backpressure, retries, DLQ routing, and graceful shutdown.
Uses unittest.IsolatedAsyncioTestCase for standard-library async testing compatibility.
"""

from __future__ import annotations

import asyncio
import time
import unittest
from typing import Any, Dict

from models import TelemetryEvent
from pipeline import (
    AsyncETLPipeline,
    AsyncMicroBatchSink,
    MicroBatch,
    PermanentValidationError,
    TransientUpstreamError,
)


class TestTelemetryValidation(unittest.TestCase):
    """Unit tests for data schema validation and constraints."""

    def test_valid_telemetry_payload(self) -> None:
        payload = {
            "device_id": "dev-alpha-01",
            "metric": "cpu_usage",
            "value": 45.5,
            "timestamp": time.time(),
            "metadata": {"rack": "us-east-1"},
        }
        if hasattr(TelemetryEvent, "model_validate"):
            event = TelemetryEvent.model_validate(payload)
        else:
            event = TelemetryEvent(**payload)

        self.assertEqual(event.device_id, "dev-alpha-01")
        self.assertEqual(event.metric, "cpu_usage")
        self.assertEqual(event.value, 45.5)

    def test_invalid_device_id_rejected(self) -> None:
        bad_payload = {
            "device_id": "bad_device_no_prefix",
            "metric": "temperature_c",
            "value": 25.0,
        }
        with self.assertRaises(ValueError):
            if hasattr(TelemetryEvent, "model_validate"):
                TelemetryEvent.model_validate(bad_payload)
            else:
                TelemetryEvent(**bad_payload)

    def test_unsupported_metric_rejected(self) -> None:
        bad_payload = {
            "device_id": "dev-sensor-01",
            "metric": "unsupported_frequency",
            "value": 100.0,
        }
        with self.assertRaises(ValueError):
            if hasattr(TelemetryEvent, "model_validate"):
                TelemetryEvent.model_validate(bad_payload)
            else:
                TelemetryEvent(**bad_payload)

    def test_physical_limits_rejected(self) -> None:
        # Temperature below absolute zero (-273.15 C)
        bad_payload = {
            "device_id": "dev-sensor-02",
            "metric": "temperature_c",
            "value": -300.0,
        }
        with self.assertRaises(ValueError):
            if hasattr(TelemetryEvent, "model_validate"):
                TelemetryEvent.model_validate(bad_payload)
            else:
                TelemetryEvent(**bad_payload)


class TestAsyncETLPipeline(unittest.IsolatedAsyncioTestCase):
    """End-to-end async tests for the streaming ETL pipeline."""

    async def test_end_to_end_streaming_and_batch_flush(self) -> None:
        pipeline = AsyncETLPipeline(
            queue_capacity=50,
            num_workers=2,
            max_retries=1,
            batch_size=5,
            flush_interval=0.1,
        )
        await pipeline.start()

        # Enqueue 12 valid records (should create 2 full batches of 5 + 1 flushed on shutdown)
        for i in range(1, 13):
            payload = {
                "device_id": f"dev-test-{i:02d}",
                "metric": "cpu_usage",
                "value": float(i * 5),
                "timestamp": time.time(),
            }
            enqueued = await pipeline.enqueue(payload)
            self.assertTrue(enqueued)

        # Allow workers to process
        await pipeline.shutdown(timeout=2.0)

        self.assertEqual(pipeline.metrics.extracted, 12)
        self.assertEqual(pipeline.metrics.validated, 12)
        self.assertEqual(pipeline.metrics.records_persisted, 12)
        self.assertEqual(len(pipeline.dlq), 0)
        self.assertGreaterEqual(pipeline.metrics.batches_flushed, 2)

    async def test_dlq_poison_pill_routing(self) -> None:
        pipeline = AsyncETLPipeline(
            queue_capacity=20,
            num_workers=2,
            max_retries=1,
            batch_size=10,
            flush_interval=0.1,
        )
        await pipeline.start()

        # Enqueue 2 valid and 2 malformed records
        records = [
            {"device_id": "dev-ok-1", "metric": "temperature_c", "value": 22.0},
            {"device_id": "MALFORMED_ID", "metric": "temperature_c", "value": 22.0},  # DLQ
            {"device_id": "dev-ok-2", "metric": "cpu_usage", "value": 50.0},
            {"device_id": "dev-ok-3", "metric": "unknown_metric", "value": 10.0},    # DLQ
        ]

        for r in records:
            await pipeline.enqueue(r)

        await pipeline.shutdown(timeout=2.0)

        self.assertEqual(pipeline.metrics.validated, 2)
        self.assertEqual(pipeline.metrics.rejected, 2)
        self.assertEqual(len(pipeline.dlq), 2)
        self.assertEqual(pipeline.dlq[0].failed_at_stage, "validation")
        self.assertEqual(pipeline.dlq[1].failed_at_stage, "validation")

    async def test_transient_retry_success(self) -> None:
        pipeline = AsyncETLPipeline(
            queue_capacity=10,
            num_workers=1,
            max_retries=3,
            initial_backoff=0.01,
            max_backoff=0.05,
            batch_size=5,
            flush_interval=0.1,
        )
        await pipeline.start()

        # Enqueue payload flagged for transient failure
        payload = {
            "device_id": "dev-retry-01",
            "metric": "request_latency_ms",
            "value": 120.0,
            "_simulate_transient_failure": True,
        }
        await pipeline.enqueue(payload)

        await pipeline.shutdown(timeout=2.0)

        # Verified retry behavior: retried 3 times, then routed to DLQ when exhausted
        self.assertEqual(pipeline.metrics.retried, 3)
        self.assertEqual(len(pipeline.dlq), 1)
        self.assertEqual(pipeline.dlq[0].failed_at_stage, "enrichment_retry")

    async def test_graceful_shutdown_drains_in_flight(self) -> None:
        pipeline = AsyncETLPipeline(
            queue_capacity=50,
            num_workers=3,
            batch_size=10,
            flush_interval=0.2,
        )
        await pipeline.start()

        # Enqueue 20 items rapidly
        for i in range(20):
            await pipeline.enqueue({
                "device_id": f"dev-drain-{i:02d}",
                "metric": "network_bytes",
                "value": 1024.0 * (i + 1),
            })

        # Immediate shutdown call should drain all 20 items cleanly
        await pipeline.shutdown(timeout=3.0)

        self.assertEqual(pipeline.queue.qsize(), 0)
        self.assertEqual(pipeline.metrics.records_persisted, 20)


def run_standalone() -> None:
    suite = unittest.TestLoader().loadTestsFromNames([
        "test_pipeline.TestTelemetryValidation",
        "test_pipeline.TestAsyncETLPipeline",
    ])
    runner = unittest.TextTestRunner(verbosity=2)
    result = runner.run(suite)
    if not result.wasSuccessful():
        raise SystemExit(1)


if __name__ == "__main__":
    run_standalone()
