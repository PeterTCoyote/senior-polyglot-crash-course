"""
Day 6 – Fixtures and Dependency Injection Lifecycle (Python 3.15+)
Senior Polyglot Crash Course

JAVA TRANSLATION NOTE:
In JUnit 5, test setup and lifecycle management rely on annotations and mutable class fields:
    @BeforeAll static void initSuite() { ... }
    @BeforeEach void setUp() { this.store = new SafeMemoryStore(); }
    @AfterEach void tearDown() { this.store.clear(); }

In Pytest:
    - Fixtures use true Dependency Injection via test function arguments.
    - Setup and Teardown are unified within a single generator function using `yield`.
    - Code before `yield` is setup; code after `yield` is guaranteed teardown.
    - Scope hierarchies (`function`, `module`, `session`) replace JUnit's static methods.
    - Fixtures can depend on other fixtures cleanly without class inheritance hierarchies.
"""

from __future__ import annotations

import time
from typing import Generator

try:
    import pytest
except ImportError:
    import _pytest_compat as pytest  # type: ignore[no-redef]
from service import (
    Order,
    OrderPaymentService,
    PaymentGateway,
    Receipt,
    SafeMemoryStore,
)


# In-memory test double conforming to PaymentGateway Protocol
class StubPaymentGateway:
    def __init__(self, failure: Exception | None = None) -> None:
        self.failure = failure
        self.charge_calls: list[tuple[str, int, str]] = []

    def charge(self, order_id: str, amount_cents: int, currency: str) -> Receipt:
        self.charge_calls.append((order_id, amount_cents, currency))
        if self.failure:
            raise self.failure
        return Receipt(
            transaction_id=f"txn-{order_id}",
            order_id=order_id,
            amount_charged=amount_cents,
            currency=currency,
        )


# ============================================================================
# Pytest Fixture Hierarchy
# ============================================================================

@pytest.fixture(scope="module")
def module_telemetry() -> Generator[dict[str, float], None, None]:
    """
    Module-scoped fixture: Executed once for all tests in this file.
    Equivalent to JUnit 5 @BeforeAll / @AfterAll.
    """
    telemetry = {"start_time": time.time()}
    print("\n[MODULE SETUP] Initializing shared test telemetry...")
    yield telemetry
    duration = time.time() - telemetry["start_time"]
    print(f"\n[MODULE TEARDOWN] Suite execution finished in {duration:.4f}s.")


@pytest.fixture
def memory_store() -> Generator[SafeMemoryStore, None, None]:
    """
    Function-scoped fixture with guaranteed teardown using `yield`.
    Equivalent to @BeforeEach (before yield) and @AfterEach (after yield).
    """
    store = SafeMemoryStore()
    yield store
    # Teardown logic: guaranteed to run even if the test fails
    # Useful for resetting caches, closing file handles, or flushing queues.


@pytest.fixture
def stub_gateway() -> StubPaymentGateway:
    """Provides a fresh stub gateway instance per test."""
    return StubPaymentGateway()


@pytest.fixture
def order_service(stub_gateway: StubPaymentGateway, memory_store: SafeMemoryStore) -> OrderPaymentService:
    """
    Demonstrates Fixture Composition:
    Fixtures can request other fixtures as arguments!
    Pytest resolves the dependency graph and instantiates dependencies in order.
    """
    return OrderPaymentService(gateway=stub_gateway, store=memory_store)


@pytest.fixture
def sample_order() -> Order:
    return Order(
        id="ord-fixture-100",
        customer_id="cust-fixture-01",
        amount_cents=7500,
        currency="USD",
    )


# ============================================================================
# Test Cases Using Injected Fixtures
# ============================================================================

def test_successful_payment_flow(
    order_service: OrderPaymentService,
    stub_gateway: StubPaymentGateway,
    memory_store: SafeMemoryStore,
    sample_order: Order,
) -> None:
    """Pytest injects all requested fixtures by parameter name."""
    receipt = order_service.process_payment(sample_order)

    # In modern Python / pytest, plain `assert` with expressions is idiomatic.
    # Pytest rewrites AST bytecode to print full variable diffs on failure.
    assert receipt.order_id == sample_order.id
    assert receipt.amount_charged == 7500
    assert len(stub_gateway.charge_calls) == 1
    assert len(memory_store) == 1


def test_idempotency_bypasses_gateway(
    order_service: OrderPaymentService,
    stub_gateway: StubPaymentGateway,
    sample_order: Order,
) -> None:
    """Verifies that duplicate order requests return cached receipt without second gateway call."""
    # First payment: hits gateway
    first_receipt = order_service.process_payment(sample_order)

    # Second payment: should hit idempotency store
    second_receipt = order_service.process_payment(sample_order)

    assert first_receipt.transaction_id == second_receipt.transaction_id
    # Assert gateway was only called ONCE
    assert len(stub_gateway.charge_calls) == 1


# Standalone runner for environments without pytest CLI
def run_standalone() -> None:
    print("Running fixture lifecycle test suite standalone...")
    gateway = StubPaymentGateway()
    store = SafeMemoryStore()
    svc = OrderPaymentService(gateway=gateway, store=store)
    order = Order(id="ord-fixture-100", customer_id="cust-01", amount_cents=7500, currency="USD")

    # Test 1
    receipt = svc.process_payment(order)
    assert receipt.order_id == order.id
    assert len(gateway.charge_calls) == 1
    print("  ✓ test_successful_payment_flow passed")

    # Test 2
    second_receipt = svc.process_payment(order)
    assert receipt.transaction_id == second_receipt.transaction_id
    assert len(gateway.charge_calls) == 1
    print("  ✓ test_idempotency_bypasses_gateway passed")
    print("All fixture lifecycle tests passed!\n")


if __name__ == "__main__":
    run_standalone()
