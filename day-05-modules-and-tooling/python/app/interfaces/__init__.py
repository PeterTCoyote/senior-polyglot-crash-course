from .models import DataModel
from .service import DataService

# Explicitly defining the public API of this package
__all__ = ["DataModel", "DataService"]

"""
This __init__.py file acts as the 'gatekeeper' for the interfaces package.
By defining __all__, we control what is exposed when a user does:
'from app.interfaces import *'
"""
