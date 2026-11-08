"""
Day 6 – Parameterized Testing with Pytest vs JUnit 5 (Python 3.15+)
Senior Polyglot Crash Course

JAVA TRANSLATION NOTE:
In JUnit 5, you write:
    @ParameterizedTest(name = "{0}")
    @MethodSource("validationData")
    void testValidateOrder(String name, Order order, Class<? extends Exception> expectedEx) {
        if (expectedEx != null) {
            assertThrows(expectedEx, () -> service.validateOrder(order));
        } else {
            assertDoesNotThrow(() -> service.validateOrder(order));
        }
    }

In pytest:
    - We use `@pytest.mark.parametrize` to bind arguments directly to the test function.
    - Each test case is identified with `pytest.param(..., id="case_name")`.
    - Expected exceptions are verified with the `pytest.raises(ExpectedException)` context manager.
"""

from __future__ import annotations

try:
    import pytest
except ImportError:
    import _pytest_compat as pytest  # type: ignore[no-redef]
from service import (
    EmptyFieldError,
    InvalidAmountError,
    Order,
    OrderPaymentService,
    UnsupportedCurrencyError,
)


# Test cases defined cleanly as a list of pytest.param entries.
VALIDATION_CASES = [
    pytest.param(
        Order(id="ord-01", customer_id="cust-01", amount_cents=5000, currency="USD"),
        None,
        id="valid_usd_transaction",
    ),
    pytest.param(
        Order(id="ord-02", customer_id="cust-02", amount_cents=1200, currency="EUR"),
        None,
        id="valid_eur_transaction",
    ),
    pytest.param(
        Order(id="", customer_id="cust-03", amount_cents=2000, currency="USD"),
        EmptyFieldError,
        id="missing_order_id_raises_EmptyFieldError",
    ),
    pytest.param(
        Order(id="ord-04", customer_id="", amount_cents=3500, currency="USD"),
        EmptyFieldError,
        id="missing_customer_id_raises_EmptyFieldError",
    ),
    pytest.param(
        Order(id="ord-05", customer_id="cust-05", amount_cents=0, currency="USD"),
        InvalidAmountError,
        id="zero_amount_raises_InvalidAmountError",
    ),
    pytest.param(
        Order(id="ord-06", customer_id="cust-06", amount_cents=-100, currency="USD"),
        InvalidAmountError,
        id="negative_amount_raises_InvalidAmountError",
    ),
    pytest.param(
        Order(id="ord-07", customer_id="cust-07", amount_cents=1000, currency="JPY"),
        UnsupportedCurrencyError,
        id="unsupported_currency_raises_UnsupportedCurrencyError",
    ),
]


@pytest.mark.parametrize("order,expected_error", VALIDATION_CASES)
def test_validate_order_parameterized(order: Order, expected_error: type[Exception] | None) -> None:
    """
    Demonstrates table-driven parameterized testing in pytest.
    Pytest will generate a distinct, named test node for each pytest.param entry.
    """
    svc = OrderPaymentService(gateway=None, store=None)  # type: ignore[arg-type]

    if expected_error is not None:
        # pytest.raises acts as a context manager, equivalent to assertThrows() in JUnit
        with pytest.raises(expected_error) as exc_info:
            svc.validate_order(order)
        # We can also verify error message contents without regex overhead
        assert len(str(exc_info.value)) > 0
    else:
        # If no error is expected, invoking the method should succeed without raising
        svc.validate_order(order)


# Demonstration runner for execution without pytest CLI
def run_standalone() -> None:
    print("Running parameterized test suite standalone...")
    svc = OrderPaymentService(gateway=None, store=None)  # type: ignore[arg-type]
    passed = 0
    for param in VALIDATION_CASES:
        order, expected_error = param.values
        case_id = param.id
        try:
            if expected_error is not None:
                try:
                    svc.validate_order(order)
                    raise AssertionError(f"[{case_id}] Expected {expected_error.__name__}, but none was raised")
                except expected_error:
                    pass
            else:
                svc.validate_order(order)
            passed += 1
            print(f"  ✓ {case_id}")
        except Exception as e:
            print(f"  ✗ {case_id}: {e}")
            raise
    print(f"All {passed} parameterized validation tests passed!\n")


if __name__ == "__main__":
    run_standalone()
