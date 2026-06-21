"""Structured JSON logging for the AIAUDITOR engine service.

Configures structlog to emit JSON matching tech spec \u00a710.1:
  {timestamp, level, service, trace_id, span_id, org_id, engagement_id, event, metadata}

Usage::

    from engine.shared.logging import get_logger

    log = get_logger("engine")
    log.info("reasoning.start", trace_id=trace_id, org_id=org_id, metadata={"control_id": ctrl})
"""
from __future__ import annotations

import logging
import sys
from typing import Any

import structlog

# W3C Trace Context header names
HEADER_TRACEPARENT = "traceparent"
HEADER_TRACESTATE = "tracestate"
HEADER_REQUEST_ID = "X-Request-ID"


def _add_service_field(
    logger: Any, method: str, event_dict: dict[str, Any]
) -> dict[str, Any]:
    """Add service name from logger name into every event."""
    # structlog.PrintLogger does not expose a .name attribute; guard with getattr.
    event_dict.setdefault("service", getattr(logger, "name", None) or "engine")
    return event_dict


def _rename_event_key(
    logger: Any, method: str, event_dict: dict[str, Any]
) -> dict[str, Any]:
    """Rename structlog's 'event' key to match spec \u00a710.1 (event field)."""
    # structlog puts the message in 'event'; ensure we keep it there.
    return event_dict


def _add_log_level(
    logger: Any, method: str, event_dict: dict[str, Any]
) -> dict[str, Any]:
    """Normalise log level to uppercase to match spec \u00a710.1."""
    if "level" in event_dict:
        event_dict["level"] = event_dict["level"].upper()
    return event_dict


def configure_logging(service: str = "engine", level: str = "INFO") -> None:
    """Configure structlog + stdlib logging for structured JSON output.

    Call once at application startup before any logging calls.
    """
    log_level = getattr(logging, level.upper(), logging.INFO)
    logging.basicConfig(
        format="%(message)s",
        stream=sys.stdout,
        level=log_level,
    )

    shared_processors: list[structlog.types.Processor] = [
        structlog.contextvars.merge_contextvars,
        structlog.stdlib.add_log_level,
        _add_log_level,
        _add_service_field,
        structlog.processors.TimeStamper(fmt="iso", utc=True, key="timestamp"),
        structlog.processors.StackInfoRenderer(),
        structlog.processors.format_exc_info,
        structlog.processors.UnicodeDecoder(),
    ]

    structlog.configure(
        processors=[
            *shared_processors,
            structlog.processors.JSONRenderer(),
        ],
        wrapper_class=structlog.make_filtering_bound_logger(log_level),
        context_class=dict,
        logger_factory=structlog.PrintLoggerFactory(),
        cache_logger_on_first_use=True,
    )


def get_logger(service: str = "engine") -> structlog.stdlib.BoundLogger:
    """Return a structlog bound logger tagged with the given service name."""
    configure_logging(service)
    return structlog.get_logger(service)


def parse_traceparent(header: str) -> tuple[str, str] | tuple[None, None]:
    """Parse a W3C traceparent header into (trace_id, span_id).

    Returns (None, None) if the header is invalid.
    """
    if not header:
        return None, None
    parts = header.split("-")
    if len(parts) != 4:
        return None, None
    version, trace_id, parent_id, flags = parts
    if version != "00" or len(trace_id) != 32 or len(parent_id) != 16 or len(flags) != 2:
        return None, None
    return trace_id, parent_id


def format_traceparent(trace_id: str, span_id: str) -> str:
    """Format a W3C traceparent header value."""
    return f"00-{trace_id}-{span_id}-01"

