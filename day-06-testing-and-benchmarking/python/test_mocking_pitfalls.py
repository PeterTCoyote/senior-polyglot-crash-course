"""
Day 6 – Mocking Gotchas & Pitfalls for Java Developers (Python 3.15+)
Senior Polyglot Crash Course

JAVA TRANSLATION & ARCHITECTURAL COMPARISON:
In Java with Mockito:
    PaymentGateway mock = Mockito.mock(PaymentGateway.class);
    when(mock.charge(...)).thenReturn(receipt);

If you introduce a typo:
    when(mock.chrrge(...)) // COMPILE ERROR! Java compiler refuses to build.

In Python with unittest.mock.MagicMock:
    - By default, MagicMock is completely dynamic (duck typing gone wild).
    - If you write `mock.chrrge(...)` or `mock.nonexistent_method()`, Python does NOT raise an error!
      It dynamically creates and returns a brand-new MagicMock child object.
    - DANGER: If you refactor a method name in production code, your tests will STILL PASS GREEN!
    - SOLUTION: Always use `autospec=True` or `spec=YourInterface` to enforce interface contracts.
"""

from __future__ import annotations

from unittest.mock import MagicMock, create_autospec, patch

try:
    import pytest
except ImportError:
    import _pytest_compat as pytest  # type: ignore[no-redef]
from service import (
    GatewayUnavailableError,
    Order,
    OrderPaymentService,
    PaymentGateway,
    Receipt,
    SafeMemoryStore,
)


def test_pitfall_1_dynamic_mock_silent_typos() -> None:
    """
    Demonstrates the dangerous "Silent Pass" trap of vanilla MagicMock.
    Java developers expect static type enforcement; Python mocks will lie to you by default.
    """
    # 1. Unconstrained MagicMock (ANTI-PATTERN in production code)
    loose_mock = MagicMock()

    # Typo: 'charge_customer' doesn't exist on PaymentGateway!
    # Yet this succeeds silently and returns a new MagicMock instance.
    result = loose_mock.charge_customer("ord-1", 1000, "USD")
    assert isinstance(result, MagicMock)

    # 2. SENIOR DEFENSE: Constrained Mock with `spec=PaymentGateway`
    strict_mock = create_autospec(PaymentGateway)

    # Calling an invalid method now fails immediately at runtime with AttributeError!
    with pytest.raises(AttributeError) as exc_info:
        strict_mock.charge_customer("ord-1", 1000, "USD")
    assert "has no attribute 'charge_customer'" in str(exc_info.value)


def test_mock_with_autospec_verifies_method_signature() -> None:
    """
    create_autospec also verifies parameter count and names match the Protocol definition!
    """
    mock_gateway = create_autospec(PaymentGateway)

    # Missing arguments (charge requires order_id, amount_cents, currency)
    with pytest.raises(TypeError) as exc_info:
        mock_gateway.charge("ord-only")  # missing amount_cents and currency!
    assert "missing a required argument" in str(exc_info.value)


def test_process_payment_with_strict_mock() -> None:
    """
    Idiomatic test using create_autospec with Mockito-style behavior specification.
    """
    mock_gateway = create_autospec(PaymentGateway)
    expected_receipt = Receipt(
        transaction_id="txn-strict-001",
        order_id="ord-001",
        amount_charged=5000,
        currency="USD",
    )
    # Equivalent to Mockito: when(mock_gateway.charge(...)).thenReturn(expected_receipt)
    mock_gateway.charge.return_value = expected_receipt

    store = SafeMemoryStore()
    svc = OrderPaymentService(gateway=mock_gateway, store=store)

    order = Order(id="ord-001", customer_id="cust-01", amount_cents=5000, currency="USD")
    receipt = svc.process_payment(order)

    assert receipt == expected_receipt

    # Equivalent to Mockito: verify(mock_gateway, times(1)).charge("ord-001", 5000, "USD");
    mock_gateway.charge.assert_called_once_with("ord-001", 5000, "USD")


def test_mock_gateway_failure_propagation() -> None:
    """
    Specifying side effects (raising exceptions) on mocks.
    Equivalent to Mockito: when(mock.charge(...)).thenThrow(new GatewayUnavailableError(...))
    """
    mock_gateway = create_autospec(PaymentGateway)
    mock_gateway.charge.side_effect = GatewayUnavailableError("Connection reset by peer")

    store = SafeMemoryStore()
    svc = OrderPaymentService(gateway=mock_gateway, store=store)

    order = Order(id="ord-fail", customer_id="cust-fail", amount_cents=2000, currency="USD")

    with pytest.raises(GatewayUnavailableError) as exc_info:
        svc.process_payment(order)
    assert "Connection reset by peer" in str(exc_info.value)


# Standalone runner for direct CLI invocation
def run_standalone() -> None:
    print("Running mocking pitfalls test suite standalone...")
    test_pitfall_1_dynamic_mock_silent_typos()
    print("  ✓ test_pitfall_1_dynamic_mock_silent_typos passed")
    test_mock_with_autospec_verifies_method_signature()
    print("  ✓ test_mock_with_autospec_verifies_method_signature passed")
    test_process_payment_with_strict_mock()
    print("  ✓ test_process_payment_with_strict_mock passed")
    test_mock_gateway_failure_propagation()
    print("  ✓ test_mock_gateway_failure_propagation passed")
    print("All mocking pitfall tests passed!\n")


if __name__ == "__main__":
    run_standalone()
