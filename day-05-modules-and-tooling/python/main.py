import sys
import importlib
from app.core.engine import CoreEngine
from app.interfaces import DataModel, DataService

"""
SENIOR ARCHITECT NOTE:
This main.py demonstrates modularity, interface control, and dynamic inspection.

Layout:
- app/core/engine.py: Implementation details (Internal)
- app/interfaces/: Publicly exported API (The 'Contract')
- main.py: Entry point

We use 'Dependency Injection' by passing the 'engine' instance to 'DataService'.
This avoids circular dependencies because 'DataService' doesn't need to import 'CoreEngine'
directly to know what it's doing; it just needs an object that follows a certain behavior.
"""

def demonstrate_encapsulation():
    print("\n--- 1. Demonstrating Encapsulation & __all__ ---")
    
    # Testing 'from app.interfaces import *' behavior
    # We'll do this by manually triggering the logic that '*' would use
    print("Attempting to access 'DataModel' and 'DataService' from interfaces...")
    try:
        # These are in __all__
        print(f"Successfully accessed: {DataModel}")
        print(f"Successfully accessed: {DataService}")
    except Exception as e:
        print(f"Error accessing public API: {e}")

    print("\nAttempting to access '_InternalHelper' (not in __all__)...")
    # In a real 'from x import *', this wouldn't be imported.
    # We'll check if it's available in the module's namespace.
    interfaces_module = importlib.import_module("app.interfaces")
    if hasattr(interfaces_module, "_InternalHelper"):
        print("Warning: '_InternalHelper' is present in the module namespace (standard Python behavior).")
    else:
        print("_InternalHelper is not visible.")

def demonstrate_dynamic_inspection():
    print("\n--- 2. Dynamic Package Inspection ---")
    
    # Using importlib to inspect the engine at runtime
    engine_module = importlib.import_module("app.core.engine")
    print(f"Inspecting module 'app.core.engine': {engine_module}")
    
    # Check for private attributes
    engine_instance = CoreEngine(config={"env": "production"})
    print(f"Engine Instance: {engine_instance}")
    
    # Accessing private attribute (Allowed in Python, but discouraged!)
    # This is why Python developers use the '_' prefix.
    print(f"Accessing 'private' attribute '_is_running': {engine_instance._is_running}")

def run_system():
    print("\n--- 3. Running the System (Dependency Injection) ---")
    
    # 1. Initialize the low-level component (Core)
    engine = CoreEngine(config={"log_level": "INFO"})
    
    # 2. Inject the dependency into the high-level service
    service = DataService(engine)
    
    # 3. Create data
    data = DataModel(id="REQ-001", payload="{\"status\": \"active\"}")
    
    # 4. Execute
    service.process_data(data)
    
    engine.stop()

if __name__ == "__main__":
    demonstrate_encapsulation()
    demonstrate_dynamic_inspection()
    run_system()
