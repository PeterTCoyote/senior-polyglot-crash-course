#!/usr/bin/env python3
"""
Day 7 – Capstone Project Test & Execution Runner (Python Track)
Senior Polyglot Crash Course

Runs the complete Python capstone test suite:
  1. Unit validation tests (Pydantic / Dataclass fallback)
  2. Asynchronous end-to-end streaming tests (queue backpressure, batching, DLQ, retries, shutdown)
"""

from __future__ import annotations

import subprocess
import sys
import test_pipeline


def main() -> None:
    print("=" * 80)
    print("🚀 DAY 7 CAPSTONE: PYTHON ASYNC STREAMING ETL TEST SUITE")
    print("=" * 80)

    # Check if pytest is available
    pytest_available = False
    try:
        import pytest  # type: ignore # noqa: F401

        pytest_available = True
    except ImportError:
        pass

    if pytest_available:
        print("[Runner] pytest detected. Executing test suite via pytest...\n")
        res = subprocess.run([sys.executable, "-m", "pytest", "-v", "test_pipeline.py"], check=False)
        if res.returncode != 0:
            sys.exit(res.returncode)
    else:
        print("[Runner] Running tests via standard-library unittest.IsolatedAsyncioTestCase harness...\n")
        test_pipeline.run_standalone()

    print("\n[Runner] ✅ All Day 7 Python tests passed cleanly!")


if __name__ == "__main__":
    main()
