"""
Data models and schema validation for the streaming ETL pipeline.
Provides Pydantic v2 schemas when available, with a zero-dependency
dataclass validation fallback when third-party packages are not installed.
"""

from __future__ import annotations

import re
import time
from dataclasses import asdict, dataclass, field
from typing import Any, Dict, List, Optional

try:
    from pydantic import BaseModel, Field, ValidationError, field_validator

    HAS_PYDANTIC = True
except ImportError:
    HAS_PYDANTIC = False
    ValidationError = ValueError  # type: ignore[misc,assignment]


ALLOWED_METRICS = {"cpu_usage", "memory_mb", "temperature_c", "network_bytes", "request_latency_ms"}
DEVICE_ID_REGEX = re.compile(r"^dev-[a-zA-Z0-9_-]+$")


if HAS_PYDANTIC:

    class TelemetryEvent(BaseModel):
        """Pydantic v2 Telemetry Event Schema."""

        device_id: str = Field(..., description="Unique hardware device identifier")
        metric: str = Field(..., description="Metric identifier")
        value: float = Field(..., description="Measured floating point value")
        timestamp: float = Field(default_factory=time.time, description="Epoch timestamp in seconds")
        metadata: Dict[str, str] = Field(default_factory=dict, description="Arbitrary key-value metadata")

        @field_validator("device_id")
        @classmethod
        def validate_device_id(cls, v: str) -> str:
            v = v.strip()
            if not DEVICE_ID_REGEX.match(v):
                raise ValueError(f"Invalid device_id '{v}': must match pattern dev-<id>")
            return v

        @field_validator("metric")
        @classmethod
        def validate_metric(cls, v: str) -> str:
            v = v.strip().lower()
            if v not in ALLOWED_METRICS:
                raise ValueError(f"Unsupported metric '{v}'. Allowed: {sorted(ALLOWED_METRICS)}")
            return v

        @field_validator("value")
        @classmethod
        def validate_value(cls, v: float, values: Any) -> float:
            if v < -273.15:
                raise ValueError(f"Value {v} violates absolute zero physical limit")
            return v

else:

    @dataclass
    class TelemetryEvent:  # type: ignore[no-redef]
        """Zero-dependency fallback schema implementing Pydantic validation semantics."""

        device_id: str
        metric: str
        value: float
        timestamp: float = field(default_factory=time.time)
        metadata: Dict[str, str] = field(default_factory=dict)

        def __post_init__(self) -> None:
            # 1. Device ID validation
            self.device_id = str(self.device_id).strip()
            if not DEVICE_ID_REGEX.match(self.device_id):
                raise ValueError(f"Invalid device_id '{self.device_id}': must match pattern dev-<id>")

            # 2. Metric validation
            self.metric = str(self.metric).strip().lower()
            if self.metric not in ALLOWED_METRICS:
                raise ValueError(f"Unsupported metric '{self.metric}'. Allowed: {sorted(ALLOWED_METRICS)}")

            # 3. Value validation
            try:
                self.value = float(self.value)
            except (TypeError, ValueError) as err:
                raise ValueError(f"Metric value must be numeric float, got {self.value}") from err

            if self.value < -273.15:
                raise ValueError(f"Value {self.value} violates absolute zero physical limit")

            if self.metric == "cpu_usage" and not (0.0 <= self.value <= 100.0):
                raise ValueError(f"cpu_usage must be between 0.0 and 100.0, got {self.value}")

        @classmethod
        def model_validate(cls, obj: Any) -> TelemetryEvent:
            if isinstance(obj, cls):
                return obj
            if not isinstance(obj, dict):
                raise ValueError(f"Expected dict or {cls.__name__}, got {type(obj)}")
            return cls(
                device_id=obj.get("device_id", ""),
                metric=obj.get("metric", ""),
                value=obj.get("value", 0.0),
                timestamp=obj.get("timestamp", time.time()),
                metadata=obj.get("metadata", {}),
            )

        def model_dump(self) -> Dict[str, Any]:
            return asdict(self)


@dataclass
class DeadLetterRecord:
    """Encapsulates a rejected or unrecoverable payload for diagnostic inspection."""

    raw_payload: Dict[str, Any]
    error_reason: str
    failed_at_stage: str  # "validation", "enrichment", "sink"
    timestamp: float = field(default_factory=time.time)


@dataclass
class MicroBatch:
    """A batch of validated telemetry events ready for bulk persistence."""

    batch_id: str
    events: List[TelemetryEvent]
    created_at: float = field(default_factory=time.time)
