#!/usr/bin/env python3
"""
Day 6 – Comprehensive Test & Benchmark Runner (Python 3.15+)
Senior Polyglot Crash Course

Runs the complete Python testing suite:
  1. Parameterized validation tests
  2. Fixture lifecycle and dependency injection tests
  3. Mocking gotchas & contract enforcement tests
  4. Deterministic CPU and memory profiling demonstration
"""

import sys
import subprocess
import test_table_parameterized
import test_fixtures_lifecycle
import test_mocking_pitfalls
import profile_benchmark


def main() -> None:
    print("=" * 70)
    print(" DAY 6: PYTHON TESTING, BENCHMARKING & PROFILING SUITE")
    print("=" * 70)

    # Check if pytest is available
    pytest_available = False
    try:
        import pytest  # type: ignore # noqa: F401
        pytest_available = True
    except ImportError:
        pass

    if pytest_available:
        print("[Runner] pytest detected in environment. Running full pytest suite...\n")
        res = subprocess.run([sys.executable, "-m", "pytest", "-v"], check=False)
        if res.returncode != 0:
            sys.exit(res.returncode)
    else:
        print("[Runner] Running tests via standalone execution harness...\n")
        test_table_parameterized.run_standalone()
        test_fixtures_lifecycle.run_standalone()
        test_mocking_pitfalls.run_standalone()

    print("\n" + "=" * 70)
    print(" EXECUTING PERFORMANCE & MEMORY DIAGNOSTICS")
    print("=" * 70)
    profile_benchmark.demonstrate_cpu_profiling()
    profile_benchmark.demonstrate_memory_tracing()

    print("\n[Runner] Day 6 Python suite completed successfully!")


if __name__ == "__main__":
    main()
