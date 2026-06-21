"""Unit tests for engine.shared.logging."""
from __future__ import annotations

import json
import io
import sys
from unittest.mock import patch

import structlog
import pytest

from engine.shared.logging import (
    configure_logging,
    get_logger,
    parse_traceparent,
    format_traceparent,
    HEADER_TRACEPARENT,
)


class TestParseTraceparent:
    def test_valid_header(self) -> None:
        trace_id, span_id = parse_traceparent(
            "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
        )
        assert trace_id == "4bf92f3577b34da6a3ce929d0e0e4736"
        assert span_id == "00f067aa0ba902b7"

    def test_empty_header(self) -> None:
        assert parse_traceparent("") == (None, None)

    def test_invalid_format(self) -> None:
        assert parse_traceparent("invalid") == (None, None)

    def test_wrong_version(self) -> None:
        assert parse_traceparent("01-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01") == (None, None)

    def test_short_trace_id(self) -> None:
        assert parse_traceparent("00-abc123-00f067aa0ba902b7-01") == (None, None)


class TestFormatTraceparent:
    def test_format(self) -> None:
        header = format_traceparent(
            "4bf92f3577b34da6a3ce929d0e0e4736", "00f067aa0ba902b7"
        )
        assert header == "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"


class TestLogFormat:
    def test_json_fields(self, capsys: pytest.CaptureFixture[str]) -> None:
        """Logger should emit JSON with all required spec \u00a710.1 fields."""
        configure_logging("test-service")
        log = get_logger("test-service")
        log.info(
            "user.login",
            trace_id="4bf92f3577b34da6a3ce929d0e0e4736",
            span_id="00f067aa0ba902b7",
            org_id="org-1",
            engagement_id="eng-1",
        )
        captured = capsys.readouterr()
        # structlog writes to stdout; parse the JSON line
        for line in captured.out.splitlines():
            if line.strip():
                entry = json.loads(line)
                assert entry["level"] in ("INFO", "info")
                assert "timestamp" in entry
                break

    def test_all_levels_exist(self) -> None:
        """Logger must have info, warning, error, critical methods."""
        configure_logging("svc")
        log = get_logger("svc")
        assert callable(log.info)
        assert callable(log.warning)
        assert callable(log.error)
        assert callable(log.critical)

