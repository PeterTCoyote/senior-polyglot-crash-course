"""
Asynchronous Streaming ETL Pipeline Engine.
Implements bounded queue backpressure, worker coroutine pools,
exponential backoff retries with jitter, Dead Letter Queues (DLQ),
and micro-batch sinks with graceful shutdown.
"""

from __future__ import annotations

import asyncio
import logging
import random
import time
from dataclasses import dataclass, field
from typing import Any, Callable, Dict, List, Optional

from models import DeadLetterRecord, MicroBatch, TelemetryEvent

logger = logging.getLogger("etl_pipeline")


# Custom Exception Hierarchy
class PipelineError(Exception):
    """Base exception for all pipeline errors."""


class TransientUpstreamError(PipelineError):
    """Temporary network or rate-limiting error eligible for retry."""


class PermanentValidationError(PipelineError):
    """Poison pill or fatal schema validation error routed to DLQ."""


@dataclass
class PipelineMetrics:
    """Thread/Task-safe metrics counters for pipeline telemetry."""

    extracted: int = 0
    validated: int = 0
    rejected: int = 0
    retried: int = 0
    batches_flushed: int = 0
    records_persisted: int = 0


class AsyncMicroBatchSink:
    """
    Micro-batching sink that aggregates individual events and flushes them
    to persistence when either batch_size is reached or flush_interval_seconds elapses.
    """

    def __init__(
        self,
        batch_size: int = 20,
        flush_interval_seconds: float = 0.3,
        persist_handler: Optional[Callable[[MicroBatch], Any]] = None,
    ) -> None:
        self.batch_size = batch_size
        self.flush_interval = flush_interval_seconds
        self.persist_handler = persist_handler
        self._buffer: List[TelemetryEvent] = []
        self._lock = asyncio.Lock()
        self._flusher_task: Optional[asyncio.Task[None]] = None
        self._is_running = False
        self._batch_counter = 0

    async def start(self) -> None:
        self._is_running = True
        self._flusher_task = asyncio.create_task(self._periodic_flush())

    async def add(self, event: TelemetryEvent) -> None:
        async with self._lock:
            self._buffer.append(event)
            if len(self._buffer) >= self.batch_size:
                await self._flush_unlocked()

    async def flush(self) -> None:
        async with self._lock:
            await self._flush_unlocked()

    async def _flush_unlocked(self) -> None:
        if not self._buffer:
            return

        self._batch_counter += 1
        batch = MicroBatch(
            batch_id=f"BATCH-{self._batch_counter:04d}",
            events=list(self._buffer),
        )
        self._buffer.clear()

        if self.persist_handler:
            if asyncio.iscoroutinefunction(self.persist_handler):
                await self.persist_handler(batch)
            else:
                self.persist_handler(batch)

    async def _periodic_flush(self) -> None:
        while self._is_running:
            try:
                await asyncio.sleep(self.flush_interval)
                await self.flush()
            except asyncio.CancelledError:
                break

    async def stop(self) -> None:
        self._is_running = False
        if self._flusher_task and not self._flusher_task.done():
            self._flusher_task.cancel()
            try:
                await self._flusher_task
            except asyncio.CancelledError:
                pass
        # Final flush of any pending events
        await self.flush()


class AsyncETLPipeline:
    """
    High-throughput asynchronous ETL pipeline coordinating bounded ingress queues,
    worker task pools, retry escalation with jitter, Dead Letter Queue (DLQ),
    and micro-batched persistence.
    """

    def __init__(
        self,
        queue_capacity: int = 100,
        num_workers: int = 4,
        max_retries: int = 3,
        initial_backoff: float = 0.05,
        max_backoff: float = 0.5,
        batch_size: int = 15,
        flush_interval: float = 0.25,
    ) -> None:
        self.queue_capacity = queue_capacity
        self.num_workers = num_workers
        self.max_retries = max_retries
        self.initial_backoff = initial_backoff
        self.max_backoff = max_backoff

        self.queue: asyncio.Queue[Dict[str, Any]] = asyncio.Queue(maxsize=queue_capacity)
        self.dlq: List[DeadLetterRecord] = []
        self.metrics = PipelineMetrics()
        self.persisted_batches: List[MicroBatch] = []

        self.sink = AsyncMicroBatchSink(
            batch_size=batch_size,
            flush_interval_seconds=flush_interval,
            persist_handler=self._on_batch_persisted,
        )

        self._workers: List[asyncio.Task[None]] = []
        self._is_running = False

    def _on_batch_persisted(self, batch: MicroBatch) -> None:
        self.persisted_batches.append(batch)
        self.metrics.batches_flushed += 1
        self.metrics.records_persisted += len(batch.events)

    async def start(self) -> None:
        """Starts worker tasks and background micro-batch sink."""
        self._is_running = True
        await self.sink.start()
        for worker_id in range(1, self.num_workers + 1):
            task = asyncio.create_task(self._worker_loop(worker_id), name=f"etl-worker-{worker_id}")
            self._workers.append(task)

    async def enqueue(self, raw_payload: Dict[str, Any], timeout: Optional[float] = 2.0) -> bool:
        """
        Enqueues raw event for processing. Applies backpressure if queue is full.
        Returns True if enqueued successfully, False if timed out.
        """
        if not self._is_running:
            raise RuntimeError("Pipeline is not running; cannot accept new records")

        try:
            if timeout is not None:
                await asyncio.wait_for(self.queue.put(raw_payload), timeout=timeout)
            else:
                await self.queue.put(raw_payload)
            self.metrics.extracted += 1
            return True
        except asyncio.TimeoutError:
            return False

    async def _worker_loop(self, worker_id: int) -> None:
        """Individual worker coroutine consuming from the bounded queue."""
        while self._is_running or not self.queue.empty():
            try:
                # Use wait_for to periodically wake up and re-check self._is_running
                raw_payload = await asyncio.wait_for(self.queue.get(), timeout=0.1)
            except asyncio.TimeoutError:
                continue
            except asyncio.CancelledError:
                break

            try:
                await self._process_record(raw_payload)
            except Exception as exc:  # Safety boundary
                logger.error(f"[Worker-{worker_id}] Unhandled error processing record: {exc}")
                self.dlq.append(
                    DeadLetterRecord(
                        raw_payload=raw_payload,
                        error_reason=f"Unhandled exception: {exc}",
                        failed_at_stage="worker_loop",
                    )
                )
                self.metrics.rejected += 1
            finally:
                self.queue.task_done()

    async def _process_record(self, raw: Dict[str, Any]) -> None:
        """Validates, retries transient failures, and sends record to micro-batch sink."""
        # 1. Validation Stage (Pydantic or Fallback)
        try:
            if hasattr(TelemetryEvent, "model_validate"):
                event = TelemetryEvent.model_validate(raw)
            else:
                event = TelemetryEvent(**raw)
        except Exception as val_err:
            self.dlq.append(
                DeadLetterRecord(
                    raw_payload=raw,
                    error_reason=f"Validation failed: {val_err}",
                    failed_at_stage="validation",
                )
            )
            self.metrics.rejected += 1
            return

        # 2. Enrichment & Retry Stage (Simulating downstream network/enrichment)
        should_fail_transient = raw.get("_simulate_transient_failure", False)
        attempts = 0
        success = False

        while attempts <= self.max_retries:
            attempts += 1
            try:
                if should_fail_transient:
                    raise TransientUpstreamError("Simulated 503 Service Unavailable")

                # Successful transformation
                success = True
                break

            except TransientUpstreamError as transient_err:
                if attempts > self.max_retries:
                    # Retries exhausted -> route to DLQ
                    self.dlq.append(
                        DeadLetterRecord(
                            raw_payload=raw,
                            error_reason=f"Transient retries exhausted: {transient_err}",
                            failed_at_stage="enrichment_retry",
                        )
                    )
                    self.metrics.rejected += 1
                    return

                self.metrics.retried += 1
                backoff = self._calculate_backoff(attempts)
                await asyncio.sleep(backoff)

        if success:
            self.metrics.validated += 1
            await self.sink.add(event)

    def _calculate_backoff(self, attempt: int) -> float:
        """Full jitter exponential backoff."""
        multiplier = 2 ** (attempt - 1)
        base = min(self.max_backoff, self.initial_backoff * multiplier)
        return random.uniform(0.0, base)

    async def shutdown(self, timeout: float = 5.0) -> None:
        """
        Coordinated Graceful Shutdown:
        1. Halt new ingress
        2. Drain existing queue items (queue.join())
        3. Cancel and await worker tasks
        4. Flush remaining items in micro-batch sink
        """
        self._is_running = False

        # Wait for queue to be fully drained
        try:
            await asyncio.wait_for(self.queue.join(), timeout=timeout)
        except asyncio.TimeoutError:
            logger.warning(f"Shutdown timeout ({timeout}s) exceeded while draining queue")

        # Cancel all workers
        for worker in self._workers:
            worker.cancel()

        await asyncio.gather(*self._workers, return_exceptions=True)
        self._workers.clear()

        # Stop and flush sink
        await self.sink.stop()
