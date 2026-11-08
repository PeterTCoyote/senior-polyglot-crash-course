"""
Minimal fallback compatibility shim providing core pytest primitives when running
under a standard Python runtime without third-party packages installed.
When `pytest` is installed, this shim is automatically bypassed.
"""

from __future__ import annotations

from typing import Any, Callable, Sequence


class _RaisesContext:
    """Context manager emulating pytest.raises()."""

    def __init__(self, expected_exception: type[BaseException] | tuple[type[BaseException], ...]) -> None:
        self.expected_exception = expected_exception
        self.value: BaseException | None = None

    def __enter__(self) -> _RaisesContext:
        return self

    def __exit__(self, exc_type: Any, exc_val: Any, exc_tb: Any) -> bool:
        if exc_type is None:
            raise AssertionError(f"Expected exception {self.expected_exception} was not raised.")
        if not issubclass(exc_type, self.expected_exception):
            return False  # Let unexpected exception propagate
        self.value = exc_val
        return True  # Suppress expected exception


class _Param:
    """Emulates pytest.param."""

    def __init__(self, *values: Any, id: str | None = None) -> None:
        self.values = values
        self.id = id


def param(*values: Any, id: str | None = None) -> _Param:
    return _Param(*values, id=id)


class _Mark:
    """Emulates pytest.mark."""

    @staticmethod
    def parametrize(argnames: str | Sequence[str], argvalues: Sequence[Any], ids: Sequence[str] | None = None) -> Callable[[Any], Any]:
        def decorator(func: Callable[..., Any]) -> Callable[..., Any]:
            func._pytest_parametrize = (argnames, argvalues)  # type: ignore[attr-defined]
            return func
        return decorator


mark = _Mark()
raises = _RaisesContext


def fixture(*args: Any, **kwargs: Any) -> Callable[[Any], Any]:
    """Emulates @pytest.fixture."""
    def decorator(func: Callable[..., Any]) -> Callable[..., Any]:
        func._pytest_fixture = True  # type: ignore[attr-defined]
        return func

    if args and callable(args[0]):
        return decorator(args[0])
    return decorator
