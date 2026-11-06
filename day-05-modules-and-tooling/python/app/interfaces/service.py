from .models import DataModel

class DataService:
    """
    A service that operates on DataModels.
    """
    def __init__(self, engine):
        self.engine = engine

    def process_data(self, model: DataModel):
        print(f"[Service] Processing data for ID: {model.id}")
        self.engine.run()
        print(f"[Service] Data payload processed: {model.payload}")

# Note: This class is NOT exported in __all__ in __init__.py
class _UnexportedService:
    pass
