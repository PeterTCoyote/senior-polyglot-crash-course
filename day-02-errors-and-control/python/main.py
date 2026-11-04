"""
Day 2: Python Error Handling, Custom Hierarchies, and Context Managers.
Production-grade patterns for senior engineers.
"""

import logging
import sys
from contextlib import contextmanager
from typing import Generator, Any

# Configure structured-style logging
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
    stream=sys.stdout,
)
logger = logging.getLogger("enterprise.core")


# ============================================================================
# 1. Custom Enterprise Exception Hierarchy
# ============================================================================

class AppError(Exception):
    """Base class for all application-level domain errors."""
    def __init__(self, message: str, error_code: int = 5000):
        super().__init__(message)
        self.message = message
        self.error_code = error_code


class DatabaseError(AppError):
    """Raised when persistence or transaction operations fail."""
    def __init__(self, message: str, error_code: int = 5001):
        super().__init__(message, error_code)


class ValidationError(AppError):
    """Raised when payload validation or business invariants fail."""
    def __init__(self, message: str, error_code: int = 4000):
        super().__init__(message, error_code)


# ============================================================================
# 2. Advanced Context Manager (Class-based & Generator-based)
# ============================================================================

class TransactionContext:
    """
    Class-based context manager demonstrating robust resource allocation,
    automatic commit on success, and guaranteed rollback/cleanup on exception.
    """
    def __init__(self, resource_name: str) -> None:
        self.resource_name = resource_name

    def __enter__(self) -> "TransactionContext":
        logger.info(f"[DB_TX] Acquired database connection for resource: {self.resource_name}")
        logger.info("[DB_TX] BEGIN TRANSACTION")
        return self

    def __exit__(self, exc_type: type[BaseException] | None, exc_val: BaseException | None, exc_tb: Any) -> bool:
        if exc_type is not None:
            logger.error(f"[DB_TX] Exception detected ({exc_type.__name__}: {exc_val}). Issuing ROLLBACK.")
            logger.info(f"[DB_TX] Released connection for resource: {self.resource_name}")
            # Returning False propagates the exception up the stack unchanged.
            return False
        
        logger.info("[DB_TX] COMMIT TRANSACTION SUCCESSFUL")
        logger.info(f"[DB_TX] Released connection for resource: {self.resource_name}")
        return True


@contextmanager
def managed_service_resource(service_id: str) -> Generator[str, None, None]:
    """
    Generator-based context manager using contextlib for lightweight resource scoping.
    """
    logger.info(f"[SERVICE] Initializing external service client: {service_id}")
    try:
        yield f"client-handle-{service_id}"
    except Exception as e:
        logger.warning(f"[SERVICE] Service client error intercepted during execution: {e}")
        raise
    finally:
        logger.info(f"[SERVICE] Teardown & cleanup completed for service: {service_id}")


# ============================================================================
# 3. Simulated Business Logic Service
# ============================================================================

def execute_user_workflow(user_id: int) -> None:
    """Executes a workflow inside a transactional context with error translation and chaining."""
    with TransactionContext(f"user-tx-{user_id}") as tx:
        with managed_service_resource("payment-gateway") as client:
            logger.info(f"Processing workflow using {client} for user {user_id}")

            if user_id == 999:
                # Simulate low-level db failure and chain into custom domain exception
                try:
                    raise ConnectionError("Connection refused by PostgreSQL primary cluster")
                except ConnectionError as err:
                    raise DatabaseError(f"Failed to persist record for user {user_id}") from err

            if user_id == 400:
                raise ValidationError(f"Invalid tax ID provided for user {user_id}", error_code=4001)

            logger.info(f"Workflow completed successfully for user {user_id}")


# ============================================================================
# 4. Main Entrypoint & Demonstration Runner
# ============================================================================

if __name__ == "__main__":
    logger.info("=== Starting Python Error Handling & Flow Control Demo ===")

    test_user_ids = [101, 400, 999]

    for uid in test_user_ids:
        logger.info(f"\n--- Running Workflow for User ID: {uid} ---")
        try:
            execute_user_workflow(uid)
        except AppError as app_err:
            logger.error(
                f"Handled Domain Exception -> Code: {app_err.error_code} | Message: {app_err}",
                exc_info=False  # Set to True if stack trace is desired for domain errors
            )
            if app_err.__cause__:
                logger.error(f"  Caused by underlying exception: {app_err.__cause__}")
        except Exception as unhandled:
            logger.exception(f"Unhandled critical system error: {unhandled}")

    logger.info("\n=== Python Demo Complete ===")