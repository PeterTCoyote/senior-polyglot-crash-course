from dataclasses import dataclass
import sys
from typing import Protocol, runtime_checkable

# ============================================================================
# 1. STRUCTURAL SUBTYPING WITH PROTOCOL (Static Duck Typing)
# ============================================================================
@runtime_checkable
class Notifier(Protocol):
    """
    Structural contract: no subclassing required.
    Static checkers (mypy/pyright) verify method names *and* signatures.
    @runtime_checkable makes isinstance() work, but it only checks names.
    """

    def send_notification(self, message: str) -> None: ...

    def get_target(self) -> str: ...


# ============================================================================
# 2. MEMORY LAYOUT: SLOTS VS __dict__
# ============================================================================
@dataclass(slots=True)
class EnterpriseUser:
    """slots=True drops per-instance __dict__; attributes use a fixed layout."""

    user_id: str
    email: str
    role: str
    version: int = 1

    def get_target(self) -> str:
        return self.email

    def send_notification(self, message: str) -> None:
        print(f"[Python EnterpriseUser] Notifying {self.email} ({self.role}): '{message}'")


@dataclass(slots=True)
class PagerDutyService:
    service_key: str
    team_name: str

    def get_target(self) -> str:
        return f"Team: {self.team_name}"

    def send_notification(self, message: str) -> None:
        print(
            f"[Python PagerDuty] Alerting team {self.team_name} "
            f"via key {self.service_key[:4]}...: '{message}'"
        )


@dataclass
class UserWithDict:
    """Default dataclass: each instance carries a __dict__ for attributes."""

    user_id: str
    email: str
    role: str
    version: int = 1


class WrongSignature:
    """Right method names, wrong parameter types. isinstance still returns True."""

    def send_notification(self, message: int) -> None:
        return None

    def get_target(self) -> str:
        return "unused"


def dispatch_alert(notifier: Notifier, alert_msg: str) -> None:
    """
    Parameter type is the Protocol. mypy accepts EnterpriseUser / PagerDutyService
    here; it would reject UserWithDict (no matching methods) and WrongSignature
    (message: int is not str).
    """
    print(f"Target resolved -> {notifier.get_target()}")
    notifier.send_notification(alert_msg)


def instance_overhead(obj: object) -> int:
    """Instance header plus __dict__ if present. Does not include referenced strings."""
    size = sys.getsizeof(obj)
    extra = getattr(obj, "__dict__", None)
    if extra is not None:
        size += sys.getsizeof(extra)
    return size


if __name__ == "__main__":
    print("=== Senior Polyglot Course: Day 1 (Python) ===")

    user = EnterpriseUser(
        user_id="usr-9921",
        email="elena.architect@enterprise.io",
        role="Principal Engineer",
    )
    pager = PagerDutyService(
        service_key="pd-sec-key-88912",
        team_name="Cloud Infrastructure SRE",
    )
    dict_user = UserWithDict(
        user_id=user.user_id,
        email=user.email,
        role=user.role,
        version=user.version,
    )

    print("\n--- Runtime Protocol checks (names only) ---")
    print(f"Is EnterpriseUser a Notifier? {isinstance(user, Notifier)}")
    print(f"Is PagerDutyService a Notifier? {isinstance(pager, Notifier)}")
    print(f"Is UserWithDict a Notifier? {isinstance(dict_user, Notifier)}")
    print(
        f"Is WrongSignature a Notifier? {isinstance(WrongSignature(), Notifier)} "
        "(True: isinstance ignores signatures)"
    )

    print("\n--- Dispatching polymorphic alerts ---")
    dispatch_alert(user, "Kubernetes cluster autoscale event triggered.")
    dispatch_alert(pager, "Database failover replica lag exceeded threshold.")

    print("\n--- Memory: slots vs __dict__ (same fields, same string values) ---")
    print(f"EnterpriseUser (slots): {instance_overhead(user)} bytes")
    print(f"UserWithDict + __dict__: {instance_overhead(dict_user)} bytes")
    print(f"Has __dict__? slots={hasattr(user, '__dict__')} dict={hasattr(dict_user, '__dict__')}")
