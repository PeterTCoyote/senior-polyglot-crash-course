from dataclasses import dataclass

@dataclass
class DataModel:
    """
    A simple data model.
    """
    id: str
    payload: str

class _InternalHelper:
    """
    This class is intended to be private and is not included in __all__.
    """
    def do_something(self):
        return "Something internal"
