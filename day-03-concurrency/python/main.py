import asyncio
from concurrent.futures import ProcessPoolExecutor
import logging
import os
import sys
import time

# Configure structured logging
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] (Process-%(process)d) %(message)s",
    stream=sys.stdout,
)
logger = logging.getLogger(__name__)


# --- 1. CPU-bound Workload (Executed in ProcessPoolExecutor) ---
def compute_cpu_heavy(number: int) -> int:
    """CPU-bound cryptographic hash simulation or heavy mathematical crunching.

    Offloaded to a separate process to avoid blocking the asyncio event loop.
    """
    logger.info("Starting CPU-bound calculation for n=%d", number)
    total = 0
    for i in range(10**7):
        total += i * number
    logger.info("Finished CPU-bound calculation for n=%d", number)
    return total


# --- 2. Async Producer-Consumer & Task Group Pipeline ---
async def worker(worker_id: int, queue: asyncio.Queue, results: list):
    """Consumer worker pulling tasks from an asyncio.Queue."""
    logger.info("Worker %d started.", worker_id)
    while True:
        try:
            # Wait for task with timeout or cooperation point
            task_item = await asyncio.wait_for(queue.get(), timeout=2.0)
        except asyncio.TimeoutError:
            # Queue drained or idle timeout reached
            logger.info("Worker %d timed out waiting for queue items. Exiting.", worker_id)
            break

        try:
            logger.info("Worker %d processing task ID: %d", worker_id, task_item["id"])
            # Simulate network I/O latency
            await asyncio.sleep(0.3)
            
            result = f"Processed-Task-{task_item['id']}-Payload-{task_item['data']}"
            results.append(result)
        except Exception as exc:
            logger.error("Worker %d encountered error on task: %v", worker_id, exc)
        finally:
            queue.task_done()


async def main():
    start_time = time.perf_counter()
    logger.info("Initializing Senior Polyglot Concurrency Runtime (Python 3.11+)...")

    # Initialize Process Pool Executor for CPU-bound operations
    # Max workers bound to physical/logical CPU cores
    max_workers = os.cpu_count() or 2
    process_pool = ProcessPoolExecutor(max_workers=max_workers)
    logger.info("Process pool initialized with %d worker processes.", max_workers)

    # 3. Offload CPU-bound task to ProcessPoolExecutor from Asyncio Event Loop
    loop = asyncio.get_running_loop()
    cpu_future = loop.run_in_executor(process_pool, compute_cpu_heavy, 42)

    # 4. Async Queue Producer-Consumer Setup
    task_queue: asyncio.Queue[dict] = asyncio.Queue()
    results_store: list[str] = []

    # Populate queue with synthetic jobs
    for i in range(1, 11):
        await task_queue.put({"id": i, "data": f"payload-{i}"})

    # 5. Spawn Async Workers using Python 3.11+ TaskGroup (Structured Concurrency)
    num_async_workers = 3
    logger.info("Spawning %d async coroutine workers using asyncio.TaskGroup.", num_async_workers)

    try:
        async with asyncio.TaskGroup() as tg:
            for w_id in range(1, num_async_workers + 1):
                tg.create_task(worker(w_id, task_queue, results_store))
    except Exception as group_exc:
        logger.error("TaskGroup encountered unhandled exception: %s", group_exc)

    # Await the CPU-bound process pool result concurrently
    cpu_result = await cpu_future
    logger.info("CPU-bound process execution result received: %d", cpu_result)

    # Shutdown process pool cleanly
    process_pool.shutdown(wait=True)

    duration = time.perf_counter() - start_time
    logger.info(
        "All concurrency workflows finished successfully in %.4f seconds. Processed results count: %d",
        duration,
        len(results_store),
    )


if __name__ == "__main__":
    # Ensure correct entry point execution
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        logger.warning("Execution interrupted by operator.")