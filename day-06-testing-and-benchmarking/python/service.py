"""
Day 6 – Order & Payment Domain Service (Python 3.15+)
Senior Polyglot Crash Course

Demonstrates:
  1. Structural subtyping via typing.Protocol for clean interface contracts
  2. Thread-safe in-memory idempotency storage
  3. Clean domain exception hierarchies
  4. Decoupled design optimized for unit testing and profiling
"""

from __future__ import annotations

import sys
import threading
from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Protocol, runtime_checkable

# Python 3.10+ introduces `slots=True` in dataclass; dynamically include for 3.10+ / 3.15+
_DATACLASS_KWARGS = {"frozen": True, "slots": True} if sys.version_info >= (3, 10) else {"frozen": True}


# ============================================================================
# Domain Exceptions (Explicit Failure Paths)
# ============================================================================
class OrderDomainError(Exception):
    """Base exception for all order-processing domain errors."""


class InvalidAmountError(OrderDomainError):
    """Raised when an order amount is zero or negative."""


class UnsupportedCurrencyError(OrderDomainError):
    """Raised when an order specifies a currency code not supported by the gateway."""


class EmptyFieldError(OrderDomainError):
    """Raised when mandatory identifiers (order_id, customer_id) are missing."""


class DuplicateOrderError(OrderDomainError):
    """Raised when an order ID conflicts with an existing settled transaction."""


class GatewayUnavailableError(OrderDomainError):
    """Raised when the downstream payment processor is unreachable."""


# ============================================================================
# Domain Models
# ============================================================================
SUPPORTED_CURRENCIES: frozenset[str] = frozenset({"USD", "EUR", "GBP"})


@dataclass(**_DATACLASS_KWARGS)
class Order:
    id: str
    customer_id: str
    amount_cents: int
    currency: str
    metadata: dict[str, str] = field(default_factory=dict)


@dataclass(**_DATACLASS_KWARGS)
class Receipt:
    transaction_id: str
    order_id: str
    amount_charged: int
    currency: str
    processed_at: datetime = field(default_factory=lambda: datetime.now(timezone.utc))


# ============================================================================
# Protocol Contracts (Structural Interfaces)
# ============================================================================
# Java Note: Instead of explicit `implements PaymentGateway`, Python uses
# typing.Protocol (PEP 544). Any class implementing `charge()` satisfies this contract.
@runtime_checkable
class PaymentGateway(Protocol):
    def charge(self, order_id: str, amount_cents: int, currency: str) -> Receipt:
        """Charges downstream payment provider. Raises GatewayUnavailableError on failure."""
        ...


class IdempotencyStore(Protocol):
    def get(self, order_id: str) -> Receipt | None:
        ...

    def put(self, order_id: str, receipt: Receipt) -> None:
        ...


# ============================================================================
# Thread-Safe Idempotency Store
# ============================================================================
class SafeMemoryStore:
    """Thread-safe in-memory cache guarded by a re-entrant or standard lock."""

    def __init__(self) -> None:
        self._lock = threading.Lock()
        self._records: dict[str, Receipt] = {}

    def get(self, order_id: str) -> Receipt | None:
        with self._lock:
            return self._records.get(order_id)

    def put(self, order_id: str, receipt: Receipt) -> None:
        with self._lock:
            if order_id in self._records:
                raise DuplicateOrderError(f"Order '{order_id}' has already been processed.")
            self._records[order_id] = receipt

    def __len__(self) -> int:
        with self._lock:
            return len(self._records)


# ============================================================================
# Application Service
# ============================================================================
class OrderPaymentService:
    """
    Orchestrates order validation, idempotency verification, and payment settlement.
    Dependencies are injected via constructor (no Spring container needed).
    """

    def __init__(self, gateway: PaymentGateway, store: IdempotencyStore) -> None:
        self._gateway = gateway
        self._store = store

    def validate_order(self, order: Order) -> None:
        """Stateless business invariant verification. Perfect for parameterized tests."""
        if not order.id:
            raise EmptyFieldError("Order ID cannot be empty.")
        if not order.customer_id:
            raise EmptyFieldError("Customer ID cannot be empty.")
        if order.amount_cents <= 0:
            raise InvalidAmountError(f"Order amount must be positive, got {order.amount_cents}.")
        if order.currency not in SUPPORTED_CURRENCIES:
            raise UnsupportedCurrencyError(
                f"Currency '{order.currency}' unsupported. Allowed: {sorted(SUPPORTED_CURRENCIES)}."
            )

    def process_payment(self, order: Order) -> Receipt:
        # 1. Validation
        self.validate_order(order)

        # 2. Check Idempotency Store
        cached = self._store.get(order.id)
        if cached is not None:
            return cached

        # 3. Charge Downstream Gateway
        receipt = self._gateway.charge(order.id, order.amount_cents, order.currency)

        # 4. Atomic Recording (Handling concurrent insertion race)
        try:
            self._store.put(order.id, receipt)
        except DuplicateOrderError:
            winning_receipt = self._store.get(order.id)
            if winning_receipt is not None:
                return winning_receipt
            raise

        return receipt
